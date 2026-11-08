package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func quotaInt(v int) *int { return &v }

type quotaGuardCPA struct {
	mu                        sync.Mutex
	remote                    map[string]any
	used                      int
	status                    int
	failDisable, failPriority bool
	writes                    []string
}

func quotaGuardFixture(t *testing.T) (*App, AppConfig, *quotaGuardCPA) {
	t.Helper()
	mock := &quotaGuardCPA{used: 90, status: 200, remote: map[string]any{"name": "guard.json", "type": "codex", "auth_index": "guard-index", "priority": 7, "disabled": false, "access_token": "fixture-token", "plan_type": "plus"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v0/management/auth-files":
			_ = json.NewEncoder(w).Encode(map[string]any{"files": []map[string]any{mock.remote}})
		case "/v0/management/auth-files/download":
			_ = json.NewEncoder(w).Encode(mock.remote)
		case "/v0/management/api-call":
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": mock.status, "body": map[string]any{"plan_type": "plus", "rate_limit": map[string]any{"primary_window": map[string]any{"used_percent": mock.used, "limit_window_seconds": 18000, "reset_after_seconds": 3600}}}})
		case "/v0/management/auth-files/status", "/v0/management/auth-files/fields":
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if disabled, ok := payload["disabled"].(bool); ok {
				mock.writes = append(mock.writes, "disabled:"+map[bool]string{true: "true", false: "false"}[disabled])
				if mock.failDisable {
					http.Error(w, "disable unavailable", 503)
					return
				}
				mock.remote["disabled"] = disabled
			}
			if priority, ok := payload["priority"]; ok {
				mock.writes = append(mock.writes, "priority")
				if mock.failPriority {
					http.Error(w, "priority unavailable", 503)
					return
				}
				mock.remote["priority"] = priority
			}
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	a := newQuotaTestApp(t)
	configureKeeperTestCPA(t, a, server.URL, func(cfg *AppConfig) { cfg.CodexKeeper.MaxRetries = 0 })
	cfg, err := a.loadConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := &codexQuotaPolicy{codexQuotaRule: codexQuotaRule{Mode: "custom", FiveHourRemaining: quotaInt(10)}}
	if err := a.saveCodexQuotaPolicy(context.Background(), "guard.json", state); err != nil {
		t.Fatal(err)
	}
	return a, cfg, mock
}

func guardProcess(t *testing.T, a *App, cfg AppConfig) keeperAccountResult {
	t.Helper()
	remote, err := a.codexRemoteItem(context.Background(), cfg, "guard.json")
	if err != nil {
		t.Fatal(err)
	}
	return a.processKeeperAuth(context.Background(), cfg, remote, func(string) {}, true)
}

func TestCodexQuotaLatchNeedsExplicitSafeResumeAndRetainsManualZero(t *testing.T) {
	a, cfg, mock := quotaGuardFixture(t)
	ctx := context.Background()
	// Explicit guards enforce even if general inspection is in dry-run mode.
	if result := guardProcess(t, a, cfg); result.Result != "priority_degraded" {
		t.Fatalf("pause result: %#v", result)
	}
	state, _ := a.getCodexQuotaPolicy(ctx, "guard.json")
	if !state.Paused || state.SyncState != "paused" || state.PreviousPriority == nil || *state.PreviousPriority != 7 {
		t.Fatalf("pause state: %#v", state)
	}
	mock.mu.Lock()
	if !keeperBool(mock.remote["disabled"]) || keeperEffectivePriority(keeperIntPtr(mock.remote["priority"])) != -1 {
		t.Fatal("quota pause did not remove Core routing eligibility")
	}
	if !reflect.DeepEqual(mock.writes, []string{"disabled:true", "priority"}) {
		t.Fatalf("unsafe write order: %v", mock.writes)
	}
	mock.mu.Unlock()
	if err := a.updateKeeperAccountPriority(ctx, "guard.json", 99); err == nil {
		t.Fatal("manual high priority bypassed low quota")
	}
	if err := a.setKeeperAccountDisabled(ctx, "guard.json", false); err == nil {
		t.Fatal("enable bypassed quota pause")
	}
	remote, _ := a.codexRemoteItem(ctx, cfg, "guard.json")
	if err := a.setCodexManualPriorityLocked(ctx, cfg, "guard.json", 99, true, remote); err == nil {
		t.Fatal("resume bypassed active threshold")
	}
	mock.mu.Lock()
	mock.used = 1
	mock.mu.Unlock()
	guardProcess(t, a, cfg)
	state, _ = a.getCodexQuotaPolicy(ctx, "guard.json")
	if !state.Paused {
		t.Fatal("quota recovery automatically resumed")
	}
	// Simulate a process reload against the same persistent database.
	reloaded, err := NewWithOptions(ctx, NewOptions{Migrate: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.Close()
	state, _ = reloaded.getCodexQuotaPolicy(ctx, "guard.json")
	if !state.Paused {
		t.Fatal("restart lost durable pause")
	}
	state.codexQuotaRule = codexQuotaRule{Mode: "off"}
	if err := a.saveCodexQuotaPolicy(ctx, "guard.json", state); err != nil {
		t.Fatal(err)
	}
	guardProcess(t, a, cfg)
	state, _ = a.getCodexQuotaPolicy(ctx, "guard.json")
	if !state.Paused {
		t.Fatal("cancelling thresholds automatically resumed")
	}
	remote, _ = a.codexRemoteItem(ctx, cfg, "guard.json")
	if err := a.setCodexManualPriorityLocked(ctx, cfg, "guard.json", 0, true, remote); err != nil {
		t.Fatal(err)
	}
	guardProcess(t, a, cfg)
	state, _ = a.getCodexQuotaPolicy(ctx, "guard.json")
	if state.Paused || state.ManualPriority == nil || *state.ManualPriority != 0 {
		t.Fatalf("manual zero was not retained: %#v", state)
	}
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if keeperBool(mock.remote["disabled"]) || keeperEffectivePriority(keeperIntPtr(mock.remote["priority"])) != 0 {
		t.Fatal("explicit resume failed or plan default overwrote manual zero")
	}
	n := len(mock.writes)
	if n < 2 || mock.writes[n-2] != "priority" || mock.writes[n-1] != "disabled:false" {
		t.Fatalf("resume did not set priority before enabling: %v", mock.writes)
	}
}

func TestCodexQuotaSyncFailureIsDurableAndRetriesWithoutQuota(t *testing.T) {
	a, cfg, mock := quotaGuardFixture(t)
	mock.failDisable = true
	if got := guardProcess(t, a, cfg); got.Result != "network_error" {
		t.Fatal("unconfirmed pause reported success")
	}
	state, _ := a.getCodexQuotaPolicy(context.Background(), "guard.json")
	if !state.Paused || state.SyncState != "error" || state.SyncError == "" {
		t.Fatalf("lost failed pause intent: %#v", state)
	}
	mock.mu.Lock()
	mock.failDisable = false
	mock.status = 503
	mock.mu.Unlock()
	guardProcess(t, a, cfg)
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !keeperBool(mock.remote["disabled"]) || keeperEffectivePriority(keeperIntPtr(mock.remote["priority"])) != -1 {
		t.Fatal("quota outage prevented retrying durable pause")
	}
}

func TestCodexQuotaResumePreservesOtherDisableReasons(t *testing.T) {
	a, cfg, mock := quotaGuardFixture(t)
	guardProcess(t, a, cfg)
	if err := a.setKeeperAccountDisabled(context.Background(), "guard.json", true); err != nil {
		t.Fatal(err)
	}
	mock.mu.Lock()
	mock.used = 0
	mock.mu.Unlock()
	remote, _ := a.codexRemoteItem(context.Background(), cfg, "guard.json")
	err := a.setCodexManualPriorityLocked(context.Background(), cfg, "guard.json", 4, true, remote)
	if err == nil || !strings.Contains(err.Error(), "人工禁用") {
		t.Fatalf("cleared manual disable via quota resume: %v", err)
	}
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !keeperBool(mock.remote["disabled"]) {
		t.Fatal("manual disable was cleared")
	}
}

func TestCodexQuotaPauseKeepsMinusOneDuringBadCredentialAndRecovery(t *testing.T) {
	a, cfg, mock := quotaGuardFixture(t)
	guardProcess(t, a, cfg)
	mock.mu.Lock()
	mock.status = 401
	mock.mu.Unlock()
	result := guardProcess(t, a, cfg)
	if result.Priority == nil || *result.Priority != -1 {
		t.Fatalf("bad credential cleared quota priority: %#v", result)
	}
	local, err := a.getKeeperState(context.Background(), "guard.json")
	if err != nil || local.Priority == nil || *local.Priority != -1 {
		t.Fatalf("local fallback lost quota priority: %#v %v", local, err)
	}
	state, _ := a.getCodexQuotaPolicy(context.Background(), "guard.json")
	if !state.Paused || !state.CredentialBlocked {
		t.Fatalf("lost independent disable reasons: %#v", state)
	}
	mock.mu.Lock()
	mock.status = 200
	mock.used = 0
	mock.mu.Unlock()
	guardProcess(t, a, cfg)
	state, _ = a.getCodexQuotaPolicy(context.Background(), "guard.json")
	if !state.Paused || state.CredentialBlocked {
		t.Fatalf("health recovery changed quota latch: %#v", state)
	}
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !keeperBool(mock.remote["disabled"]) || keeperEffectivePriority(keeperIntPtr(mock.remote["priority"])) != -1 {
		t.Fatal("health recovery restored quota-paused routing")
	}
}

func TestCodexQuotaWindowIdentityValidityAndInclusiveThresholds(t *testing.T) {
	future, past := time.Now().Add(time.Hour), time.Now().Add(-time.Minute)
	rule := codexQuotaRule{Mode: "custom", FiveHourRemaining: quotaInt(10), WeekRemaining: quotaInt(0)}
	for _, tc := range []struct {
		name  string
		usage keeperUsageInfo
		hit   bool
	}{
		{"5h boundary", keeperUsageInfo{PrimaryUsedValid: true, PrimaryUsedPercent: 90, PrimaryWindowSeconds: quotaInt(18000), PrimaryResetAt: &future}, true},
		{"weekly primary", keeperUsageInfo{PrimaryUsedValid: true, PrimaryUsedPercent: 100, PrimaryWindowSeconds: quotaInt(604800)}, true},
		{"5h secondary", keeperUsageInfo{SecondaryUsedPercent: quotaInt(90), SecondaryWindowSeconds: quotaInt(18000)}, true},
		{"missing primary usage", keeperUsageInfo{PrimaryUsedPercent: 100, PrimaryWindowSeconds: quotaInt(18000)}, false},
		{"unknown duration", keeperUsageInfo{PrimaryUsedValid: true, PrimaryUsedPercent: 100}, false},
		{"expired window", keeperUsageInfo{PrimaryUsedValid: true, PrimaryUsedPercent: 100, PrimaryWindowSeconds: quotaInt(18000), PrimaryResetAt: &past}, false},
		{"invalid usage", keeperUsageInfo{PrimaryUsedValid: true, PrimaryUsedPercent: 101, PrimaryWindowSeconds: quotaInt(18000)}, false},
		{"weekly above zero", keeperUsageInfo{PrimaryUsedValid: true, PrimaryUsedPercent: 99, PrimaryWindowSeconds: quotaInt(604800)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexQuotaHit(rule, codexQuotaObservations(tc.usage)) != ""; got != tc.hit {
				t.Fatalf("hit=%v want %v", got, tc.hit)
			}
		})
	}
}

func TestCodexQuotaMaintenanceBypassesInspectionCacheAndDaemonSwitch(t *testing.T) {
	a, _, mock := quotaGuardFixture(t)
	if err := a.reconcileCodexQuotaPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !keeperBool(mock.remote["disabled"]) {
		t.Fatal("quota guard depended on daemon or refresh cache")
	}
}

func TestCodexQuotaPrioritySyncFailureNeverEnablesAndCanRetry(t *testing.T) {
	a, cfg, mock := quotaGuardFixture(t)
	ctx := context.Background()
	mock.failPriority = true
	if got := guardProcess(t, a, cfg); got.Result != "network_error" {
		t.Fatal("priority sync failure was reported as a completed pause")
	}
	mock.mu.Lock()
	if !keeperBool(mock.remote["disabled"]) {
		t.Fatal("priority failure left routing eligible")
	}
	mock.failPriority = false
	mock.mu.Unlock()
	guardProcess(t, a, cfg)
	state, _ := a.getCodexQuotaPolicy(ctx, "guard.json")
	if state.SyncState != "paused" {
		t.Fatalf("failed to retry pause: %#v", state)
	}
	state.Mode = "off"
	if err := a.saveCodexQuotaPolicy(ctx, "guard.json", state); err != nil {
		t.Fatal(err)
	}
	mock.mu.Lock()
	mock.failPriority = true
	mock.writes = nil
	mock.mu.Unlock()
	remote, _ := a.codexRemoteItem(ctx, cfg, "guard.json")
	if err := a.setCodexManualPriorityLocked(ctx, cfg, "guard.json", 5, true, remote); err == nil {
		t.Fatal("resume ignored priority write failure")
	}
	mock.mu.Lock()
	defer mock.mu.Unlock()
	for _, write := range mock.writes {
		if write == "disabled:false" {
			t.Fatal("resume enabled before priority confirmation")
		}
	}
	if !keeperBool(mock.remote["disabled"]) {
		t.Fatal("failed resume re-enabled routing")
	}
}

func TestCodexQuotaConcurrentPriorityChangesCannotBypassActiveGuard(t *testing.T) {
	a, cfg, mock := quotaGuardFixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := a.updateKeeperAccountPriority(ctx, "guard.json", 99); err == nil {
				t.Error("concurrent priority edit bypassed quota")
			}
		}()
		go func() {
			defer wg.Done()
			remote, err := a.codexRemoteItem(ctx, cfg, "guard.json")
			if err != nil {
				t.Error(err)
				return
			}
			a.processKeeperAuth(ctx, cfg, remote, func(string) {}, true)
		}()
	}
	wg.Wait()
	state, err := a.getCodexQuotaPolicy(ctx, "guard.json")
	if err != nil || !state.Paused || state.SyncState != "paused" || state.Version < 1 {
		t.Fatalf("invalid persisted concurrent state: %#v %v", state, err)
	}
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if !keeperBool(mock.remote["disabled"]) || keeperEffectivePriority(keeperIntPtr(mock.remote["priority"])) != -1 {
		t.Fatal("concurrent writes lost hard pause")
	}
	for _, write := range mock.writes {
		if write == "disabled:false" {
			t.Fatal("concurrent writes temporarily enabled routing")
		}
	}
}

func TestCodexQuotaPerAccountOverrideAndZeroThreshold(t *testing.T) {
	a := newQuotaTestApp(t)
	ctx := context.Background()
	global := &codexQuotaPolicy{codexQuotaRule: codexQuotaRule{Mode: "custom", FiveHourRemaining: quotaInt(10), WeekRemaining: quotaInt(5)}}
	if err := a.saveCodexQuotaPolicy(ctx, "", global); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		rule     codexQuotaRule
		wantMode string
		wantFive *int
	}{
		{codexQuotaRule{Mode: "inherit"}, "custom", quotaInt(10)},
		{codexQuotaRule{Mode: "custom", FiveHourRemaining: quotaInt(0)}, "custom", quotaInt(0)},
		{codexQuotaRule{Mode: "off", FiveHourRemaining: quotaInt(0)}, "off", nil},
	} {
		rule, err := a.effectiveCodexQuotaRule(ctx, &codexQuotaPolicy{codexQuotaRule: tc.rule})
		if err != nil || rule.Mode != tc.wantMode || !reflect.DeepEqual(rule.FiveHourRemaining, tc.wantFive) {
			t.Fatalf("bad rule inheritance: %#v %v", rule, err)
		}
	}
}

func TestCodexQuotaPolicyAPIRequiresAuthAndExplicitResumePriority(t *testing.T) {
	a, cfg, mock := quotaGuardFixture(t)
	handler := a.Routes()
	request := func(method, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
		for _, cookie := range cookies {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	setup := request(http.MethodPost, "/api/auth/setup", `{"username":"admin","password":"test-password","nickname":"Test admin"}`, nil)
	if setup.Code < 200 || setup.Code >= 300 {
		t.Fatalf("setup: %d %s", setup.Code, setup.Body.String())
	}
	cookies := setup.Result().Cookies()
	if got := request(http.MethodPut, "/api/codex-keeper/quota-policy", `{"mode":"off"}`, nil); got.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized quota mutation: %d", got.Code)
	}
	if got := request(http.MethodPut, "/api/codex-keeper/quota-policy", `{"mode":"custom","five_hour_remaining":0,"week_remaining":null}`, cookies); got.Code != 200 {
		t.Fatalf("zero threshold: %d %s", got.Code, got.Body.String())
	}
	for _, path := range []string{"/api/codex-keeper/accounts/guard.json/resume", "/api/codex-keeper/accounts/guard.json/priority"} {
		method := http.MethodPost
		if strings.HasSuffix(path, "/priority") {
			method = http.MethodPatch
		}
		if got := request(method, path, `{}`, cookies); got.Code != http.StatusUnprocessableEntity {
			t.Fatalf("missing explicit priority accepted: %d %s", got.Code, got.Body.String())
		}
	}
	mock.mu.Lock()
	if len(mock.writes) != 0 {
		t.Error("missing priority mutated CPA")
	}
	mock.mu.Unlock()
	guardProcess(t, a, cfg)
	got := request(http.MethodPut, "/api/codex-keeper/accounts/guard.json/quota-policy", `{"mode":"off","paused":false}`, cookies)
	if got.Code != 200 {
		t.Fatalf("disable rule: %d %s", got.Code, got.Body.String())
	}
	state, _ := a.getCodexQuotaPolicy(context.Background(), "guard.json")
	if !state.Paused {
		t.Fatal("policy update allowed clearing durable pause")
	}
}
