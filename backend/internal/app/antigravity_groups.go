package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cpa-helper/backend/internal/groupscheduler"
)

type antigravityGroupPolicy struct {
	AuthName string   `json:"auth_name"`
	GroupID  string   `json:"group_id"`
	Priority int      `json:"priority"`
	Models   []string `json:"model_patterns"`
}

func antigravityWindowSeconds(window string) int {
	switch strings.ToLower(window) {
	case "5h", "five_hour":
		return 5 * 60 * 60
	case "daily", "day":
		return 24 * 60 * 60
	case "weekly", "week":
		return 7 * 24 * 60 * 60
	default:
		return 0
	}
}

func defaultAntigravityModels(label string) []string {
	label = strings.ToLower(label)
	switch {
	case strings.Contains(label, "claude") && strings.Contains(label, "gpt"):
		return []string{"claude-*", "gpt-*"}
	case strings.Contains(label, "argon"):
		return []string{"gemini-4*", "argon*"}
	case strings.Contains(label, "gemini"):
		return []string{"gemini-*"}
	}
	return []string{}
}

func (a *App) antigravityPolicies(ctx context.Context, name string) (map[string]antigravityGroupPolicy, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT group_id, priority, model_patterns FROM antigravity_group_policies WHERE auth_name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]antigravityGroupPolicy{}
	for rows.Next() {
		policy := antigravityGroupPolicy{AuthName: name}
		var models string
		if err := rows.Scan(&policy.GroupID, &policy.Priority, &models); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(models), &policy.Models); err != nil {
			return nil, err
		}
		result[policy.GroupID] = policy
	}
	return result, rows.Err()
}

func (a *App) enrichAntigravityQuota(ctx context.Context, cfg AppConfig, name string, files []map[string]any, snapshot antigravityQuotaSnapshot) (antigravityQuotaSnapshot, error) {
	policies, err := a.antigravityPolicies(ctx, name)
	if err != nil {
		return snapshot, err
	}
	// Cached quota observations are immutable; usage and policies are read fresh.
	groups := make([]antigravityQuotaGroup, len(snapshot.Groups))
	copy(groups, snapshot.Groups)
	snapshot.Groups = groups
	now := time.Now().UTC()
	rules := make([]groupscheduler.Rule, len(groups))
	windows := make([][]*keeperQuotaWindowUsage, len(groups))
	var start, end time.Time
	authIndex := ""
	basePriority := 0
	sourceAccounts := map[string]string{}
	aliases := map[string][]string{}
	for _, file := range files {
		fileName := keeperString(file["name"])
		addKeeperAuthObjectAliases(aliases, file)
		addKeeperSourceAccountAlias(sourceAccounts, fileName, fileName)
		if email := keeperString(file["email"]); email != "" {
			addKeeperSourceAccountAlias(sourceAccounts, email, fileName)
		}
		if fileName == name {
			if priority := keeperIntPtr(file["priority"]); priority != nil {
				basePriority = *priority
			}
			if index := keeperRemoteAuthIndex(file); index != nil {
				authIndex = *index
			}
		}
	}
	for gi := range groups {
		group := &groups[gi]
		group.Buckets = append([]antigravityQuotaBucket(nil), group.Buckets...)
		group.ModelPatterns = defaultAntigravityModels(group.Label)
		group.Priority = basePriority
		if policy, ok := policies[group.ID]; ok {
			group.Priority, group.ModelPatterns = policy.Priority, policy.Models
		}
		remaining := []float64{}
		fresh := !snapshot.Stale && snapshot.Error == nil && len(group.Buckets) > 0
		validUntil := now.Add(antigravityQuotaTTL)
		if snapshot.CheckedAt == nil {
			fresh = false
		} else if expiry := snapshot.CheckedAt.Add(antigravityQuotaTTL); expiry.Before(validUntil) {
			validUntil = expiry
		}
		windows[gi] = make([]*keeperQuotaWindowUsage, len(group.Buckets))
		for bi, bucket := range group.Buckets {
			remaining = append(remaining, bucket.RemainingPercent)
			seconds := antigravityWindowSeconds(bucket.Window)
			if bucket.ResetAt == nil || !bucket.ResetAt.After(now) {
				fresh = false
			}
			if bucket.ResetAt != nil && bucket.ResetAt.Before(validUntil) {
				validUntil = *bucket.ResetAt
			}
			if seconds == 0 || bucket.ResetAt == nil || len(group.ModelPatterns) == 0 {
				continue
			}
			window := &keeperQuotaWindowUsage{WindowEnd: *bucket.ResetAt, ResetAt: *bucket.ResetAt, WindowSeconds: seconds, WindowStart: bucket.ResetAt.Add(-time.Duration(seconds) * time.Second), Stale: snapshot.Stale || !bucket.ResetAt.After(now), WindowSource: "reported"}
			windows[gi][bi] = window
			start, end = keeperQuotaWindowBounds(start, end, window)
		}
		group.EffectivePriority = groupscheduler.EffectivePriority(group.Priority, remaining, cfg.CodexKeeper.QuotaThreshold, fresh && !cfg.CodexKeeper.DryRun)
		group.Degraded = group.EffectivePriority != group.Priority
		rules[gi] = groupscheduler.Rule{AuthName: name, AuthIndex: authIndex, GroupID: group.ID, Models: group.ModelPatterns, Priority: group.Priority, EffectivePriority: group.EffectivePriority, ValidUntil: validUntil}
	}
	if !start.IsZero() && !end.IsZero() {
		records, loadErr := a.keeperUsageRecordsInRange(ctx, start, end)
		if loadErr != nil {
			message := "窗口用量读取失败，请重试"
			snapshot.UsageError = &message
		} else {
			for _, record := range records {
				matched, ok := keeperAccountNameForUsageRecord(record, sourceAccounts, aliases)
				if !ok || matched != name {
					continue
				}
				gi, ok := groupscheduler.GroupForModel(rules, valueOr(record.Model, ""))
				if !ok {
					continue
				}
				for _, window := range windows[gi] {
					if keeperRecordInQuotaWindow(record, window) {
						addRecordToKeeperQuotaWindowUsage(window, record, nil)
					}
				}
			}
			for gi := range groups {
				for bi := range groups[gi].Buckets {
					groups[gi].Buckets[bi].Usage = keeperQuotaWindowUsageResponseFrom(windows[gi][bi])
				}
			}
		}
	}
	if err := a.writeGroupSchedulerSnapshot(name, rules); err != nil {
		message := "调度快照写入失败，请检查数据目录权限"
		snapshot.UsageError = &message
	}
	return snapshot, nil
}

func (a *App) groupSchedulerPath() string {
	if configured := strings.TrimSpace(os.Getenv("CPA_HELPER_SCHEDULER_SNAPSHOT")); configured != "" {
		return configured
	}
	return filepath.Join(a.dataDir, "group-scheduler.json")
}

func (a *App) writeGroupSchedulerSnapshot(name string, rules []groupscheduler.Rule) error {
	a.groupSchedulerMu.Lock()
	defer a.groupSchedulerMu.Unlock()
	target := a.groupSchedulerPath()
	snapshot := groupscheduler.Snapshot{Version: 1, GeneratedAt: time.Now().UTC(), Rules: []groupscheduler.Rule{}}
	if data, err := os.ReadFile(target); err == nil {
		var previous groupscheduler.Snapshot
		if err := json.Unmarshal(data, &previous); err != nil {
			return err
		}
		for _, rule := range previous.Rules {
			if rule.AuthName != name {
				snapshot.Rules = append(snapshot.Rules, rule)
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	snapshot.Rules = append(snapshot.Rules, rules...)
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".group-scheduler-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), target)
}

func (a *App) handleGroupScheduler(w http.ResponseWriter, r *http.Request) error {
	user, err := a.readyUser(r.Context(), r)
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		return forbiddenError("需要管理员权限")
	}
	switch r.Method {
	case http.MethodGet:
		var heartbeat struct {
			PickedAt time.Time `json:"picked_at"`
		}
		if data, err := os.ReadFile(a.groupSchedulerPath() + ".status"); err == nil {
			_ = json.Unmarshal(data, &heartbeat)
		}
		active := !heartbeat.PickedAt.IsZero() && time.Since(heartbeat.PickedAt) >= 0 && time.Since(heartbeat.PickedAt) < 2*time.Minute
		writeJSON(w, http.StatusOK, map[string]any{"active": active, "last_pick_at": heartbeat.PickedAt, "plugin": "cpa-group-scheduler"})
		return nil
	case http.MethodPut:
		var policy antigravityGroupPolicy
		if err := decodeJSON(r, &policy); err != nil {
			return err
		}
		if policy.Priority < -1000000 || policy.Priority > 1000000 || policy.Priority == -1 {
			return validationError("优先级须为 -1000000 到 1000000 的整数，-1 保留给自动降级")
		}
		if len(policy.Models) == 0 || len(policy.Models) > 100 {
			return validationError("请配置 1 到 100 个模型匹配规则")
		}
		for _, model := range policy.Models {
			if len(model) > 200 || strings.TrimSpace(model) == "" || strings.ContainsAny(model, "[]\\/") {
				return validationError("模型规则仅支持模型名称、* 和 ? 通配符")
			}
		}
		snapshot, err := a.getAntigravityQuota(r.Context(), policy.AuthName, false)
		if err != nil {
			return err
		}
		exists := false
		for _, group := range snapshot.Groups {
			if group.ID == policy.GroupID {
				exists = true
			}
		}
		if !exists {
			return validationError("额度分组不存在，请刷新后重试")
		}
		models, _ := json.Marshal(policy.Models)
		_, err = a.db.ExecContext(r.Context(), `INSERT INTO antigravity_group_policies (auth_name, group_id, priority, model_patterns) VALUES (?, ?, ?, ?) ON CONFLICT(auth_name, group_id) DO UPDATE SET priority = excluded.priority, model_patterns = excluded.model_patterns`, policy.AuthName, policy.GroupID, policy.Priority, string(models))
		if err != nil {
			return err
		}
		snapshot, err = a.getAntigravityQuota(r.Context(), policy.AuthName, false)
		if err != nil {
			return err
		}
		if snapshot.UsageError != nil {
			return fmt.Errorf("%s", *snapshot.UsageError)
		}
		writeJSON(w, http.StatusOK, snapshot)
		return nil
	default:
		return requireMethod(r, http.MethodGet)
	}
}
