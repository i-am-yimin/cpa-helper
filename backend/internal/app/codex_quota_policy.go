package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// These rules are independent of Keeper's dry-run inspections. An explicitly
// configured quota guard is an actual routing guard, like manual disable.
type codexQuotaRule struct {
	Mode              string `json:"mode"`
	FiveHourRemaining *int   `json:"five_hour_remaining"`
	WeekRemaining     *int   `json:"week_remaining"`
}

type codexQuotaObservation struct {
	Window    string     `json:"window"`
	Remaining int        `json:"remaining"`
	ResetAt   *time.Time `json:"reset_at,omitempty"`
}

type codexQuotaPolicy struct {
	codexQuotaRule
	EffectiveRule     *codexQuotaRule         `json:"effective_rule,omitempty"`
	Paused            bool                    `json:"paused"`
	PausedAt          *time.Time              `json:"paused_at,omitempty"`
	Reason            string                  `json:"reason,omitempty"`
	Trigger           *codexQuotaObservation  `json:"trigger,omitempty"`
	TriggerThreshold  *int                    `json:"trigger_threshold,omitempty"`
	PreviousPriority  *int                    `json:"previous_priority,omitempty"`
	ManualPriority    *int                    `json:"manual_priority,omitempty"`
	ManualDisabled    bool                    `json:"manual_disabled"`
	CredentialBlocked bool                    `json:"credential_blocked"`
	SyncState         string                  `json:"sync_state,omitempty"`
	SyncError         string                  `json:"sync_error,omitempty"`
	SyncedAt          *time.Time              `json:"synced_at,omitempty"`
	CheckedAt         *time.Time              `json:"checked_at,omitempty"`
	Observations      []codexQuotaObservation `json:"observations,omitempty"`
	Version           int64                   `json:"version"`
}

func (a *App) lockCodexAuth(name string) func() {
	// Global edits cannot race a resume checked against the previous global rule.
	a.codexPolicyMu.RLock()
	value, _ := a.codexAuthLocks.LoadOrStore(name, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return func() { mu.Unlock(); a.codexPolicyMu.RUnlock() }
}

func (a *App) getCodexQuotaPolicy(ctx context.Context, name string) (*codexQuotaPolicy, error) {
	state := &codexQuotaPolicy{codexQuotaRule: codexQuotaRule{Mode: "inherit"}}
	if name == "" {
		state.Mode = "off"
	}
	var raw string
	err := a.db.QueryRowContext(ctx, `SELECT policy_json FROM codex_quota_policies WHERE auth_name = ?`, name).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), state); err != nil {
		return nil, fmt.Errorf("读取额度暂停状态失败: %w", err)
	}
	return state, nil
}

func (a *App) saveCodexQuotaPolicy(ctx context.Context, name string, state *codexQuotaPolicy) error {
	state.Version++
	copy := *state
	copy.EffectiveRule = nil
	raw, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	_, err = a.db.ExecContext(ctx, `INSERT INTO codex_quota_policies (auth_name, policy_json, updated_at) VALUES (?, ?, ?) ON CONFLICT(auth_name) DO UPDATE SET policy_json = excluded.policy_json, updated_at = excluded.updated_at`, name, string(raw), dbTime(time.Now()))
	return err
}

func (a *App) effectiveCodexQuotaRule(ctx context.Context, state *codexQuotaPolicy) (codexQuotaRule, error) {
	rule := state.codexQuotaRule
	if rule.Mode == "inherit" {
		global, err := a.getCodexQuotaPolicy(ctx, "")
		if err != nil {
			return rule, err
		}
		rule = global.codexQuotaRule
	}
	if rule.Mode != "custom" {
		return codexQuotaRule{Mode: "off"}, nil
	}
	return rule, nil
}

func codexQuotaRuleEnabled(rule codexQuotaRule) bool {
	return rule.Mode == "custom" && (rule.FiveHourRemaining != nil || rule.WeekRemaining != nil)
}

func validateCodexQuotaRule(rule codexQuotaRule, global bool) error {
	if rule.Mode != "custom" && rule.Mode != "off" && !(rule.Mode == "inherit" && !global) {
		return validationError("额度规则模式无效")
	}
	for _, threshold := range []*int{rule.FiveHourRemaining, rule.WeekRemaining} {
		if threshold != nil && (*threshold < 0 || *threshold > 100) {
			return validationError("剩余额度下限必须在 0% 到 100% 之间；取消勾选表示关闭")
		}
	}
	if rule.Mode == "custom" && !codexQuotaRuleEnabled(rule) {
		return validationError("请设置至少一个窗口下限，或关闭规则")
	}
	return nil
}

// Only explicit, valid window observations can trigger a guard. A missing
// primary window is not an observed 0%, and primary does not imply five hours.
func codexQuotaObservations(usage keeperUsageInfo) []codexQuotaObservation {
	items := []codexQuotaObservation{}
	add := func(valid bool, used int, seconds *int, reset *time.Time) {
		if !valid || used < 0 || used > 100 || seconds == nil || (reset != nil && !reset.After(time.Now())) {
			return
		}
		window := ""
		switch *seconds {
		case 18000:
			window = "5h"
		case 604800:
			window = "week"
		}
		if window != "" {
			items = append(items, codexQuotaObservation{Window: window, Remaining: 100 - used, ResetAt: reset})
		}
	}
	add(usage.PrimaryUsedValid, usage.PrimaryUsedPercent, usage.PrimaryWindowSeconds, usage.PrimaryResetAt)
	if usage.SecondaryUsedPercent != nil {
		add(true, *usage.SecondaryUsedPercent, usage.SecondaryWindowSeconds, usage.SecondaryResetAt)
	}
	return items
}

func codexQuotaHit(rule codexQuotaRule, items []codexQuotaObservation) string {
	if !codexQuotaRuleEnabled(rule) {
		return ""
	}
	for _, item := range items {
		threshold := rule.FiveHourRemaining
		label := "5 小时"
		if item.Window == "week" {
			threshold, label = rule.WeekRemaining, "周"
		}
		if threshold != nil && item.Remaining <= *threshold {
			return fmt.Sprintf("%s剩余额度 %d%% ≤ 下限 %d%%", label, item.Remaining, *threshold)
		}
	}
	return ""
}

func (a *App) codexRemoteItem(ctx context.Context, cfg AppConfig, name string) (map[string]any, error) {
	files, err := a.listKeeperRemoteAuthFiles(ctx, cfg)
	if err != nil {
		return nil, err
	}
	item, ok := keeperRemoteAuthFilesByName(files)[name]
	if !ok {
		return nil, notFoundError("凭证不存在")
	}
	return item, nil
}

func (a *App) failCodexQuotaSync(ctx context.Context, name string, state *codexQuotaPolicy, err error) error {
	state.SyncState, state.SyncError = "error", err.Error()
	return errors.Join(err, a.saveCodexQuotaPolicy(ctx, name, state))
}

// Caller holds the account lock. The durable pause intent precedes remote
// mutations, and disable precedes priority. Never infer hard blocking from -1.
func (a *App) enforceCodexQuotaPause(ctx context.Context, cfg AppConfig, name string, state *codexQuotaPolicy, remote map[string]any) error {
	if !state.Paused {
		return nil
	}
	if keeperBool(remote["disabled"]) && keeperEffectivePriority(keeperIntPtr(remote["priority"])) == -1 && state.SyncState == "paused" {
		return nil
	}
	state.SyncState, state.SyncError = "pending", ""
	if err := a.saveCodexQuotaPolicy(ctx, name, state); err != nil {
		return err
	}
	if err := a.setKeeperRemoteDisabled(ctx, cfg, name, true); err != nil {
		return a.failCodexQuotaSync(ctx, name, state, err)
	}
	low := -1
	if err := a.setKeeperRemotePriority(ctx, cfg, name, &low); err != nil {
		return a.failCodexQuotaSync(ctx, name, state, err)
	}
	confirmed, err := a.codexRemoteItem(ctx, cfg, name)
	if err == nil && (!keeperBool(confirmed["disabled"]) || keeperEffectivePriority(keeperIntPtr(confirmed["priority"])) != -1) {
		err = fmt.Errorf("CPA 尚未确认禁用和优先级 -1")
	}
	if err != nil {
		return a.failCodexQuotaSync(ctx, name, state, err)
	}
	now := time.Now()
	state.SyncState, state.SyncError, state.SyncedAt = "paused", "", &now
	if err := a.saveCodexQuotaPolicy(ctx, name, state); err != nil {
		return err
	}
	remote["disabled"], remote["priority"] = true, -1
	return nil
}

func (a *App) observeCodexQuota(ctx context.Context, cfg AppConfig, name string, state *codexQuotaPolicy, remote map[string]any, usage keeperUsageInfo) error {
	rule, err := a.effectiveCodexQuotaRule(ctx, state)
	if err != nil {
		return err
	}
	now := time.Now()
	wasCredentialBlocked := state.CredentialBlocked
	state.CredentialBlocked = false
	state.CheckedAt, state.Observations = &now, codexQuotaObservations(usage)
	if reason := codexQuotaHit(rule, state.Observations); reason != "" && !state.Paused {
		state.Paused, state.PausedAt, state.Reason = true, &now, reason
		state.PreviousPriority = keeperIntPtr(remote["priority"])
		// Do not release an account that was disabled before this guard owned it.
		state.ManualDisabled = state.ManualDisabled || (keeperBool(remote["disabled"]) && !wasCredentialBlocked)
		for _, observation := range state.Observations {
			threshold := rule.FiveHourRemaining
			if observation.Window == "week" {
				threshold = rule.WeekRemaining
			}
			if threshold != nil && observation.Remaining <= *threshold {
				copy := observation
				state.Trigger, state.TriggerThreshold = &copy, threshold
				break
			}
		}
		state.SyncState = "pending"
	}
	if err := a.saveCodexQuotaPolicy(ctx, name, state); err != nil {
		return err
	}
	return a.enforceCodexQuotaPause(ctx, cfg, name, state, remote)
}

// Recovery needs a fresh, safe observation for every configured window. When
// rules have been disabled, still verify the window that caused the pause.
func codexQuotaRecovered(rule codexQuotaRule, state *codexQuotaPolicy) bool {
	if !codexQuotaRuleEnabled(rule) {
		if state.Trigger == nil || state.TriggerThreshold == nil {
			return false
		}
		rule = codexQuotaRule{Mode: "custom"}
		switch state.Trigger.Window {
		case "5h":
			rule.FiveHourRemaining = state.TriggerThreshold
		case "week":
			rule.WeekRemaining = state.TriggerThreshold
		default:
			return false
		}
	}
	if codexQuotaHit(rule, state.Observations) != "" {
		return false
	}
	for _, window := range []struct {
		name      string
		threshold *int
	}{{"5h", rule.FiveHourRemaining}, {"week", rule.WeekRemaining}} {
		if window.threshold == nil {
			continue
		}
		found := false
		for _, observation := range state.Observations {
			if observation.Window == window.name && observation.Remaining > *window.threshold && observation.Remaining <= 100 && (observation.ResetAt == nil || observation.ResetAt.After(time.Now())) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Called only after a successful inspection, under the per-credential lock.
// Manual priority edits perform their own explicit resume instead.
func (a *App) restoreCodexQuota(ctx context.Context, cfg AppConfig, name string, state *codexQuotaPolicy, remote map[string]any, accountType *string) (bool, error) {
	if !state.Paused || state.ManualDisabled || state.CredentialBlocked {
		return false, nil
	}
	rule, err := a.effectiveCodexQuotaRule(ctx, state)
	if err != nil || !codexQuotaRecovered(rule, state) {
		return false, err
	}
	planPriority := keeperPriorityForType(accountType, cfg.CodexKeeperPriorityRule)
	priority := state.ManualPriority
	if priority == nil {
		priority = state.PreviousPriority
	}
	if priority == nil || *priority == -1 {
		priority = keeperDisplayPriority(planPriority)
	}
	if err := validateCodexManualPriority(*priority); err != nil {
		return false, err
	}
	state.SyncState, state.SyncError = "recovering", ""
	if err := a.saveCodexQuotaPolicy(ctx, name, state); err != nil {
		return false, err
	}
	fail := func(err error) (bool, error) {
		// Keep the durable pause until both the remote state and local commit
		// are confirmed. Retry from a disabled account on the next inspection.
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		low := -1
		err = errors.Join(err, a.setKeeperRemoteDisabled(rollbackCtx, cfg, name, true), a.setKeeperRemotePriority(rollbackCtx, cfg, name, &low))
		return false, a.failCodexQuotaSync(rollbackCtx, name, state, err)
	}
	if err := a.setKeeperRemotePriority(ctx, cfg, name, priority); err != nil {
		return fail(err)
	}
	confirmed, err := a.codexRemoteItem(ctx, cfg, name)
	if err != nil {
		return fail(err)
	}
	if keeperEffectivePriority(keeperIntPtr(confirmed["priority"])) != *priority || !keeperBool(confirmed["disabled"]) || !codexQuotaRecovered(rule, state) {
		return fail(fmt.Errorf("尚未确认恢复优先级或有效额度"))
	}
	if err := a.setKeeperRemoteDisabled(ctx, cfg, name, false); err != nil {
		return fail(err)
	}
	confirmed, err = a.codexRemoteItem(ctx, cfg, name)
	if err != nil {
		return fail(err)
	}
	if keeperBool(confirmed["disabled"]) || keeperEffectivePriority(keeperIntPtr(confirmed["priority"])) != *priority {
		return fail(fmt.Errorf("CPA 尚未确认额度恢复后的优先级或路由状态"))
	}
	recovered := *state
	recovered.Paused, recovered.PausedAt, recovered.Reason = false, nil, ""
	recovered.Trigger, recovered.TriggerThreshold, recovered.PreviousPriority = nil, nil, nil
	// Preserve pre-pause values set outside the panel, including zero, across
	// later inspections rather than replacing them with a plan default.
	if recovered.ManualPriority == nil && (planPriority == nil || *priority != *planPriority) {
		recovered.ManualPriority = priority
	}
	now := time.Now()
	recovered.SyncState, recovered.SyncError, recovered.SyncedAt = "synced", "", &now
	if err := a.saveCodexQuotaPolicy(ctx, name, &recovered); err != nil {
		return fail(err)
	}
	*state = recovered
	remote["disabled"], remote["priority"] = false, *priority
	return true, nil
}

func (a *App) freshCodexQuota(ctx context.Context, cfg AppConfig, name string, state *codexQuotaPolicy, remote map[string]any) error {
	detail, err := a.getKeeperRemoteAuthFile(ctx, cfg, name)
	if err != nil {
		return err
	}
	if detail == nil || keeperString(detail["access_token"]) == "" {
		return conflictError("凭证不可用，请先修复凭证")
	}
	result := a.checkKeeperUsage(ctx, cfg, mergeKeeperObjects(remote, detail))
	if result.StatusCode == nil || *result.StatusCode < 200 || *result.StatusCode >= 300 || isBadKeeperCredential(result) {
		return conflictError("无法确认最新额度和凭证健康，请刷新后重试")
	}
	return a.observeCodexQuota(ctx, cfg, name, state, remote, parseKeeperUsageInfo(result.JSONData))
}

func validateCodexManualPriority(priority int) error {
	if priority < -1000000 || priority > 1000000 || priority == -1 {
		return validationError("优先级须为 -1000000 到 1000000 的整数，-1 保留给额度暂停")
	}
	return nil
}

func (a *App) setCodexPlanPriority(ctx context.Context, name string) error {
	unlock := a.lockCodexAuth(name)
	defer unlock()
	cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}
	remote, err := a.codexRemoteItem(ctx, cfg, name)
	if err != nil {
		return err
	}
	if !keeperIsCodexProvider(keeperCredentialProvider(remote)) {
		return validationError("套餐默认值仅适用于 Codex")
	}
	accountType := accountTypeFromKeeperDetail(remote, nil)
	if accountType == nil || *accountType == "unknown" {
		if local, err := a.getKeeperState(ctx, name); err == nil {
			accountType = local.AccountType
		}
	}
	priority := keeperPriorityForType(accountType, cfg.CodexKeeperPriorityRule)
	if priority == nil {
		return validationError("此凭证没有套餐默认优先级")
	}
	if err := a.setCodexManualPriorityLocked(ctx, cfg, name, *priority, false, remote); err != nil {
		return err
	}
	state, err := a.getCodexQuotaPolicy(ctx, name)
	if err != nil {
		return err
	}
	state.ManualPriority = nil
	return a.saveCodexQuotaPolicy(ctx, name, state)
}

// All resume and priority-edit entry points use this guard. No temporary
// exemption is granted while a threshold is hit, including through enable.
func (a *App) setCodexManualPriorityLocked(ctx context.Context, cfg AppConfig, name string, priority int, resume bool, remote map[string]any) error {
	if err := validateCodexManualPriority(priority); err != nil {
		return err
	}
	state, err := a.getCodexQuotaPolicy(ctx, name)
	if err != nil {
		return err
	}
	if state.Paused {
		if err := a.enforceCodexQuotaPause(ctx, cfg, name, state, remote); err != nil {
			return err
		}
	}
	rule, err := a.effectiveCodexQuotaRule(ctx, state)
	if err != nil {
		return err
	}
	if codexQuotaRuleEnabled(rule) || resume {
		if err := a.freshCodexQuota(ctx, cfg, name, state, remote); err != nil {
			return err
		}
	}
	if reason := codexQuotaHit(rule, state.Observations); reason != "" {
		return conflictError("仍保持暂停：" + reason + "。请取消或调整阈值后手动恢复")
	}
	if codexQuotaRuleEnabled(rule) {
		for _, window := range []struct {
			name      string
			threshold *int
		}{{"5h", rule.FiveHourRemaining}, {"week", rule.WeekRemaining}} {
			if window.threshold == nil {
				continue
			}
			found := false
			for _, observation := range state.Observations {
				if observation.Window == window.name {
					found = true
				}
			}
			if !found {
				return conflictError("缺少已配置窗口的有效额度，暂不能恢复路由")
			}
		}
	}
	if state.Paused && !resume {
		return conflictError("此凭证因额度暂停，请使用“保存优先级并恢复路由”")
	}
	if resume && state.ManualDisabled {
		return conflictError("此凭证另有人工禁用，请先解除人工禁用；额度暂停会继续保留")
	}
	if state.CredentialBlocked {
		return conflictError("此凭证仍不可用，请先修复并刷新")
	}
	// Persist manual priority before the remote write; a subsequent inspection
	// retries this value rather than overwriting it with a plan default.
	state.ManualPriority, state.SyncState, state.SyncError = &priority, "pending", ""
	if err := a.saveCodexQuotaPolicy(ctx, name, state); err != nil {
		return err
	}
	if err := a.setKeeperRemotePriority(ctx, cfg, name, &priority); err != nil {
		return a.failCodexQuotaSync(ctx, name, state, err)
	}
	if resume {
		if err := a.setKeeperRemoteDisabled(ctx, cfg, name, false); err != nil {
			_ = a.setKeeperRemoteDisabled(ctx, cfg, name, true)
			return a.failCodexQuotaSync(ctx, name, state, err)
		}
	}
	confirmed, err := a.codexRemoteItem(ctx, cfg, name)
	if err == nil && (keeperEffectivePriority(keeperIntPtr(confirmed["priority"])) != priority || (resume && keeperBool(confirmed["disabled"]))) {
		err = fmt.Errorf("CPA 尚未确认优先级或路由状态")
	}
	if err != nil {
		if resume {
			// A failed confirmation must not leave an unconfirmed resumed account.
			_ = a.setKeeperRemoteDisabled(ctx, cfg, name, true)
		}
		return a.failCodexQuotaSync(ctx, name, state, err)
	}
	if resume {
		state.Paused, state.PausedAt, state.Reason = false, nil, ""
	}
	now := time.Now()
	state.SyncState, state.SyncError, state.SyncedAt = "synced", "", &now
	if err := a.saveCodexQuotaPolicy(ctx, name, state); err != nil {
		if resume {
			_ = a.setKeeperRemoteDisabled(ctx, cfg, name, true)
		}
		return err
	}
	_, err = a.db.ExecContext(ctx, `UPDATE codex_keeper_auth_states SET priority = ?, disabled = ?, restore_priority = NULL, latest_action = ?, updated_at = ? WHERE auth_name = ?`, priority, keeperBool(confirmed["disabled"]), "手动设置优先级", dbTime(now), name)
	return err
}

func (a *App) codexQuotaPolicyForAccount(account keeperAccount) *codexQuotaPolicy {
	if !keeperIsCodexProvider(account.Provider) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	state, err := a.getCodexQuotaPolicy(ctx, account.Name)
	if err != nil {
		return &codexQuotaPolicy{SyncState: "unknown", SyncError: "无法读取额度暂停状态"}
	}
	rule, err := a.effectiveCodexQuotaRule(ctx, state)
	if err != nil {
		state.SyncState, state.SyncError = "unknown", "无法读取全局额度规则"
		return state
	}
	state.EffectiveRule = &rule
	return state
}

func (a *App) handleCodexQuotaPolicy(w http.ResponseWriter, r *http.Request, name string, global bool) error {
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		return methodNotAllowed()
	}
	if global {
		if r.Method == http.MethodPut {
			a.codexPolicyMu.Lock()
			defer a.codexPolicyMu.Unlock()
		} else {
			a.codexPolicyMu.RLock()
			defer a.codexPolicyMu.RUnlock()
		}
	} else {
		unlock := a.lockCodexAuth(name)
		defer unlock()
		cfg, err := a.loadConfig(r.Context())
		if err != nil {
			return err
		}
		remote, err := a.codexRemoteItem(r.Context(), cfg, name)
		if err != nil {
			return err
		}
		if !keeperIsCodexProvider(keeperCredentialProvider(remote)) {
			return validationError("此规则仅适用于 Codex 凭证")
		}
	}
	state, err := a.getCodexQuotaPolicy(r.Context(), name)
	if err != nil {
		return err
	}
	if r.Method == http.MethodPut {
		var rule codexQuotaRule
		if err := decodeJSON(r, &rule); err != nil {
			return err
		}
		if err := validateCodexQuotaRule(rule, global); err != nil {
			return err
		}
		state.codexQuotaRule = rule
		if err := a.saveCodexQuotaPolicy(r.Context(), name, state); err != nil {
			return err
		}
	}
	rule, err := a.effectiveCodexQuotaRule(r.Context(), state)
	if err != nil {
		return err
	}
	state.EffectiveRule = &rule
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, state)
	return nil
}

func (a *App) startCodexQuotaMaintenance(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	a.codexPolicyCancel, a.codexPolicyDone = cancel, make(chan struct{})
	go func() {
		defer close(a.codexPolicyDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			if err := a.reconcileCodexQuotaPolicies(ctx); err != nil && ctx.Err() == nil {
				log.Printf("codex quota guard: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// This monitor is independent of the daemon switch and ten-minute inspection
// cache. It also keeps disabled-by-quota accounts observable and reconciles
// durable pause intent before attempting any quota network request.
func (a *App) reconcileCodexQuotaPolicies(ctx context.Context) error {
	cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}
	global, err := a.getCodexQuotaPolicy(ctx, "")
	if err != nil {
		return err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT auth_name, policy_json FROM codex_quota_policies WHERE auth_name <> ''`)
	if err != nil {
		return err
	}
	states := map[string]codexQuotaPolicy{}
	for rows.Next() {
		var name, raw string
		if err := rows.Scan(&name, &raw); err != nil {
			rows.Close()
			return err
		}
		var state codexQuotaPolicy
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			rows.Close()
			return err
		}
		states[name] = state
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	needed := codexQuotaRuleEnabled(global.codexQuotaRule)
	for _, state := range states {
		needed = needed || state.Paused || codexQuotaRuleEnabled(state.codexQuotaRule)
	}
	if !needed {
		return nil
	}
	files, err := a.listKeeperRemoteAuthFiles(ctx, cfg)
	if err != nil {
		return err
	}
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, remote := range files {
		if !keeperIsCodexProvider(keeperCredentialProvider(remote)) {
			continue
		}
		name := keeperString(remote["name"])
		state, ok := states[name]
		if !state.Paused && !(state.Mode == "custom" && codexQuotaRuleEnabled(state.codexQuotaRule)) && !((!ok || state.Mode == "inherit") && codexQuotaRuleEnabled(global.codexQuotaRule)) {
			continue
		}
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(item map[string]any) {
			defer wg.Done()
			defer func() { <-sem }()
			cycle, cancel := context.WithTimeout(ctx, 90*time.Second)
			defer cancel()
			a.processKeeperAuth(cycle, cfg, item, func(message string) { log.Print("codex quota guard: " + message) }, false)
		}(remote)
	}
	wg.Wait()
	return nil
}
