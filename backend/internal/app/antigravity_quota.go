package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const antigravityQuotaTTL = 5 * time.Minute

// Keep quota queries on the server. CLIProxyAPI resolves $TOKEN$ and refreshes
// OAuth credentials; neither access tokens nor credential documents reach the UI.
var antigravityQuotaURLs = []string{
	"https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary",
	"https://daily-cloudcode-pa.sandbox.googleapis.com/v1internal:retrieveUserQuotaSummary",
	"https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary",
}

type antigravityQuotaBucket struct {
	ID               string                          `json:"id"`
	Label            string                          `json:"label"`
	Window           string                          `json:"window"`
	RemainingPercent float64                         `json:"remaining_percent"`
	ResetAt          *time.Time                      `json:"reset_at"`
	Usage            *keeperQuotaWindowUsageResponse `json:"usage"`
}

type antigravityQuotaGroup struct {
	ID                string                   `json:"id"`
	Label             string                   `json:"label"`
	Buckets           []antigravityQuotaBucket `json:"buckets"`
	Priority          int                      `json:"priority"`
	EffectivePriority int                      `json:"effective_priority"`
	ModelPatterns     []string                 `json:"model_patterns"`
	Degraded          bool                     `json:"degraded"`
}

type antigravityQuotaSnapshot struct {
	Groups      []antigravityQuotaGroup `json:"groups"`
	CheckedAt   *time.Time              `json:"checked_at"`
	AttemptedAt time.Time               `json:"attempted_at"`
	Error       *string                 `json:"error"`
	Stale       bool                    `json:"stale"`
	UsageError  *string                 `json:"usage_error"`
}

type antigravityQuotaEntry struct {
	Snapshot antigravityQuotaSnapshot
	Expires  time.Time
}

type antigravityQuotaCache struct {
	mu       sync.Mutex
	entries  map[string]antigravityQuotaEntry
	inFlight map[string]chan struct{}
}

func (c *antigravityQuotaCache) get(ctx context.Context, key string, force bool, load func() antigravityQuotaSnapshot) (antigravityQuotaSnapshot, error) {
	for {
		c.mu.Lock()
		if c.entries == nil {
			c.entries = make(map[string]antigravityQuotaEntry)
			c.inFlight = make(map[string]chan struct{})
		}
		previous, exists := c.entries[key]
		if exists && !force && time.Now().Before(previous.Expires) {
			c.mu.Unlock()
			return previous.Snapshot, nil
		}
		if pending, ok := c.inFlight[key]; ok {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return antigravityQuotaSnapshot{}, ctx.Err()
			case <-pending:
				force = false
				continue
			}
		}
		pending := make(chan struct{})
		c.inFlight[key] = pending
		c.mu.Unlock()

		snapshot := load()
		c.mu.Lock()
		delete(c.inFlight, key)
		if ctx.Err() != nil {
			close(pending)
			c.mu.Unlock()
			return antigravityQuotaSnapshot{}, ctx.Err()
		}
		ttl := antigravityQuotaTTL
		if snapshot.Error != nil {
			ttl = time.Minute
			if exists && len(previous.Snapshot.Groups) > 0 {
				snapshot.Groups = previous.Snapshot.Groups
				snapshot.CheckedAt = previous.Snapshot.CheckedAt
				snapshot.Stale = true
			}
		}
		expires := time.Now().Add(ttl)
		if snapshot.Error == nil {
			for _, group := range snapshot.Groups {
				for _, bucket := range group.Buckets {
					if bucket.ResetAt != nil && bucket.ResetAt.After(time.Now()) && bucket.ResetAt.Before(expires) {
						expires = *bucket.ResetAt
					}
				}
			}
		}
		// Bound memory even when credentials are repeatedly replaced or deleted.
		if len(c.entries) >= 256 {
			oldestKey := ""
			var oldest time.Time
			for entryKey, entry := range c.entries {
				if oldestKey == "" || entry.Expires.Before(oldest) {
					oldestKey, oldest = entryKey, entry.Expires
				}
			}
			delete(c.entries, oldestKey)
		}
		c.entries[key] = antigravityQuotaEntry{Snapshot: snapshot, Expires: expires}
		close(pending)
		c.mu.Unlock()
		return snapshot, nil
	}
}

func (a *App) getAntigravityQuota(ctx context.Context, name string, force bool) (antigravityQuotaSnapshot, error) {
	cfg, err := a.loadConfig(ctx)
	if err != nil {
		return antigravityQuotaSnapshot{}, err
	}
	files, err := a.listKeeperRemoteAuthFiles(ctx, cfg)
	if err != nil {
		return antigravityQuotaSnapshot{}, validationError("无法读取 CPA 凭证列表")
	}
	file, ok := keeperRemoteAuthFilesByName(files)[name]
	if !ok {
		return antigravityQuotaSnapshot{}, notFoundError("凭证不存在")
	}
	if keeperCredentialProvider(file) != "antigravity" {
		return antigravityQuotaSnapshot{}, validationError("该额度接口仅支持 Antigravity 凭证")
	}
	if keeperBool(file["disabled"]) {
		return antigravityQuotaSnapshot{}, validationError("凭证已禁用，请启用后查询额度")
	}
	index := keeperRemoteAuthIndex(file)
	if index == nil {
		return antigravityQuotaSnapshot{}, validationError("凭证缺少 auth_index，请刷新凭证列表")
	}
	key := cfg.Collector.CLIProxyURL + "\x00" + name + "\x00" + *index + "\x00" + keeperFirstString(file["modtime"], file["updated_at"])
	snapshot, err := a.antigravityQuotaCache.get(ctx, key, force, func() antigravityQuotaSnapshot {
		return a.fetchAntigravityQuota(ctx, cfg, file, *index)
	})
	if err != nil {
		return snapshot, err
	}
	return a.enrichAntigravityQuota(ctx, cfg, name, files, snapshot)
}

func antigravityProjectID(file map[string]any) string {
	if id := keeperFirstString(file["project_id"], file["projectId"]); id != "" {
		return id
	}
	for _, key := range []string{"metadata", "attributes", "installed", "web"} {
		if nested, ok := file[key].(map[string]any); ok {
			if id := keeperFirstString(nested["project_id"], nested["projectId"], nested["gemini_virtual_project"]); id != "" {
				return id
			}
		}
	}
	return ""
}

func (a *App) fetchAntigravityQuota(ctx context.Context, cfg AppConfig, file map[string]any, index string) antigravityQuotaSnapshot {
	result := antigravityQuotaSnapshot{Groups: []antigravityQuotaGroup{}, AttemptedAt: time.Now().UTC()}
	fail := func(message string) antigravityQuotaSnapshot { result.Error = &message; return result }
	project := antigravityProjectID(file)
	if project == "" {
		detail, err := a.getKeeperRemoteAuthFile(ctx, cfg, keeperString(file["name"]))
		if err != nil {
			return fail("无法读取 Antigravity 项目信息")
		}
		project = antigravityProjectID(detail)
	}
	if project == "" {
		return fail("凭证缺少 project_id，请重新授权 Antigravity")
	}
	data, _ := json.Marshal(map[string]string{"project": project})
	// One deadline covers all fallback hosts, including management retries.
	timeout := time.Duration(cfg.CodexKeeper.UsageTimeoutSeconds) * time.Second
	if timeout <= 0 || timeout > 30*time.Second {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	lastError := "Antigravity 未返回可展示的额度"
	for _, target := range antigravityQuotaURLs {
		body := map[string]any{
			"auth_index": index, "method": http.MethodPost, "url": target,
			"header": map[string]string{"Authorization": "Bearer $TOKEN$", "Content-Type": "application/json", "User-Agent": "antigravity/hub/2.23.0 darwin/arm64"},
			"data":   string(data),
		}
		response, payload, err := a.keeperRequest(ctx, cfg, http.MethodPost, "/v0/management/api-call", nil, body, timeout)
		if err != nil || response == nil || response.StatusCode < 200 || response.StatusCode >= 300 {
			lastError = "无法连接 CPA 额度查询接口，请稍后重试"
			if ctx.Err() != nil {
				break
			}
			continue
		}
		var raw map[string]any
		if json.Unmarshal(payload, &raw) != nil {
			lastError = "CPA 额度查询响应格式无效"
			continue
		}
		status := keeperIntPtr(raw["status_code"], raw["statusCode"])
		if status == nil {
			lastError = "CPA 额度查询响应缺少状态码"
			continue
		}
		if *status < 200 || *status >= 300 {
			// Upstream error bodies can contain credentials or account identifiers.
			lastError = fmt.Sprintf("Antigravity 额度查询失败（HTTP %d）", *status)
			if *status == 401 || *status == 403 || *status == 429 {
				break
			}
			continue
		}
		groups := parseAntigravityQuotaGroups(keeperBodyJSON(raw["body"]))
		if len(groups) == 0 {
			lastError = "Antigravity 未返回可展示的额度"
			continue
		}
		now := time.Now().UTC()
		result.Groups, result.CheckedAt = groups, &now
		return result
	}
	return fail(lastError)
}

func parseAntigravityQuotaGroups(payload map[string]any) []antigravityQuotaGroup {
	groups := []antigravityQuotaGroup{}
	rawGroups, _ := payload["groups"].([]any)
	for gi, rawGroup := range rawGroups {
		group, ok := rawGroup.(map[string]any)
		if !ok {
			continue
		}
		label := keeperFirstString(group["displayName"], group["display_name"])
		if label == "" {
			label = fmt.Sprintf("Quota Group %d", gi+1)
		}
		parsed := antigravityQuotaGroup{ID: keeperFirstString(group["groupId"], group["group_id"], group["id"], label), Label: label, Buckets: []antigravityQuotaBucket{}}
		buckets, _ := group["buckets"].([]any)
		for bi, rawBucket := range buckets {
			bucket, ok := rawBucket.(map[string]any)
			if !ok {
				continue
			}
			fraction, exists := bucket["remainingFraction"]
			if !exists {
				fraction = bucket["remaining_fraction"]
			}
			// Missing is unknown, while an explicit zero is an exhausted bucket.
			value, ok := fraction.(float64)
			if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
				continue
			}
			window := keeperString(bucket["window"])
			id := keeperFirstString(bucket["bucketId"], bucket["bucket_id"])
			if id == "" {
				id = fmt.Sprintf("group-%d-bucket-%d", gi+1, bi+1)
			}
			bucketLabel := keeperFirstString(bucket["displayName"], bucket["display_name"], window, id)
			reset := keeperTimeValue(keeperFirstString(bucket["resetTime"], bucket["reset_time"]))
			parsed.Buckets = append(parsed.Buckets, antigravityQuotaBucket{
				ID: id, Label: bucketLabel, Window: window,
				RemainingPercent: math.Round(math.Max(0, math.Min(1, value))*1000) / 10,
				ResetAt:          reset,
			})
		}
		sort.SliceStable(parsed.Buckets, func(i, j int) bool {
			return antigravityWindowOrder(parsed.Buckets[i].Window) < antigravityWindowOrder(parsed.Buckets[j].Window)
		})
		if len(parsed.Buckets) > 0 {
			groups = append(groups, parsed)
		}
	}
	return groups
}

func antigravityWindowOrder(window string) int {
	switch strings.ToLower(window) {
	case "5h", "five_hour":
		return 0
	case "daily", "day":
		return 1
	case "weekly", "week":
		return 2
	default:
		return 3
	}
}
