package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"cpa-helper/backend/internal/groupscheduler"
)

func TestGroupSchedulerReadinessUsesRegistryIndependentlyOfHeartbeat(t *testing.T) {
	a := newQuotaTestApp(t)
	var inventory atomic.Value
	inventory.Store(`{"plugins":[{"id":"cpa-group-scheduler","registered":true,"effective_enabled":true,"metadata":{"version":"1.0.0"}}]}`)
	var status atomic.Int32
	status.Store(200)
	cpa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v8/management/plugins" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-management-key" {
			http.Error(w, "missing authorization", 401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(status.Load()))
		_, _ = w.Write([]byte(inventory.Load().(string)))
	}))
	defer cpa.Close()
	configureKeeperTestCPA(t, a, cpa.URL, func(cfg *AppConfig) { cfg.CodexKeeper.MaxRetries = 0 })
	handler := a.Routes()
	setup := httptest.NewRecorder()
	handler.ServeHTTP(setup, httptest.NewRequest(http.MethodPost, "/api/auth/setup", bytes.NewBufferString(`{"username":"admin","password":"test-password","nickname":"Admin"}`)))
	if setup.Code < 200 || setup.Code >= 300 {
		t.Fatalf("setup failed: %d", setup.Code)
	}
	get := func() map[string]any {
		r := httptest.NewRequest(http.MethodGet, "/api/upstreams/group-scheduler", nil)
		for _, cookie := range setup.Result().Cookies() {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var body map[string]any
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
			t.Fatalf("status response: %d %s", w.Code, w.Body.String())
		}
		return body
	}
	if got := get(); got["phase"] != "ready" || got["loaded"] != true || got["active"] != false {
		t.Fatalf("idle loaded plugin misreported: %#v", got)
	}
	path := a.groupSchedulerPath() + ".status"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	heartbeat, _ := json.Marshal(map[string]any{"picked_at": time.Now()})
	if err := os.WriteFile(path, heartbeat, 0600); err != nil {
		t.Fatal(err)
	}
	inventory.Store(`{"plugins":[{"id":"cpa-group-scheduler","registered":true,"effective_enabled":false}]}`)
	if got := get(); got["phase"] != "disabled" || got["loaded"] != false || got["active"] != false {
		t.Fatalf("heartbeat overrode disabled registry: %#v", got)
	}
	status.Store(503)
	if got := get(); got["phase"] != "unknown" || got["loaded"] != false {
		t.Fatalf("registry outage reported missing/ready: %#v", got)
	}
	status.Store(200)
	inventory.Store(`{"plugins":[]}`)
	if got := get(); got["phase"] != "missing" {
		t.Fatalf("missing plugin not identified: %#v", got)
	}
}

func TestAntigravityMaintenanceContinuesPastInvalidCredential(t *testing.T) {
	now := time.Now().UTC()
	reset := now.Add(time.Hour)
	cpa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v0/management/auth-files":
			_, _ = w.Write([]byte(`{"files":[{"name":"bad.json","type":"antigravity"},{"name":"good.json","type":"antigravity","auth_index":"good-index","project_id":"test-project"}]}`))
		case "/v0/management/api-call":
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 200, "body": map[string]any{"groups": []map[string]any{{"groupId": "gpt", "displayName": "Claude and GPT models", "buckets": []map[string]any{{"window": "5h", "remainingFraction": 0, "resetTime": reset.Format(time.RFC3339)}}}}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cpa.Close()
	a := newQuotaTestApp(t)
	configureKeeperTestCPA(t, a, cpa.URL, func(cfg *AppConfig) { cfg.CodexKeeper.DryRun = false; cfg.CodexKeeper.QuotaThreshold = 95 })
	t.Setenv("CPA_HELPER_SCHEDULER_SNAPSHOT", a.groupSchedulerPath())
	if err := a.refreshGroupSchedulerQuotas(context.Background()); err == nil {
		t.Fatal("invalid credential should be reported")
	}
	data, err := os.ReadFile(a.groupSchedulerPath())
	if err != nil {
		t.Fatal(err)
	}
	var snapshot groupscheduler.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Rules) != 1 || snapshot.Rules[0].AuthName != "good.json" || snapshot.Rules[0].EffectivePriority != -1 {
		t.Fatalf("healthy credential was skipped after invalid one: %#v", snapshot)
	}
}

func TestAntigravityWindowUsageIsAttributedToCredentialModelAndResetBoundary(t *testing.T) {
	a := newQuotaTestApp(t)
	ctx := context.Background()
	now := time.Now().UTC()
	reset := now.Add(time.Hour)
	files := []map[string]any{{"name": "ag.json", "type": "antigravity", "auth_index": "ag-index"}, {"name": "other.json", "type": "codex", "auth_index": "other-index"}}
	cfg, err := a.loadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.CodexKeeper.DryRun = false
	cfg.CodexKeeper.QuotaThreshold = 95
	snapshot := antigravityQuotaSnapshot{CheckedAt: &now, Groups: []antigravityQuotaGroup{
		{ID: "gemini", Label: "Gemini Models", Buckets: []antigravityQuotaBucket{{ID: "g5", Window: "5h", RemainingPercent: 100, ResetAt: &reset}}},
		{ID: "gpt", Label: "Claude and GPT models", Buckets: []antigravityQuotaBucket{{ID: "c5", Window: "5h", RemainingPercent: 0, ResetAt: &reset}, {ID: "cw", Window: "weekly", RemainingPercent: 100, ResetAt: &reset}}},
	}}
	// The existing usage fixture's model is gpt-test. One record is in both
	// windows, one only in the weekly window, and another belongs to Codex.
	insertKeeperWindowUsageRecord(t, a, keeperWindowUsageSeed{Dedupe: "ag-short", Timestamp: now.Add(-time.Minute), AuthIndex: "ag-index", InputTokens: 10, OutputTokens: 5})
	insertKeeperWindowUsageRecord(t, a, keeperWindowUsageSeed{Dedupe: "ag-week", Timestamp: now.Add(-24 * time.Hour), AuthIndex: "ag-index", InputTokens: 20, OutputTokens: 5})
	insertKeeperWindowUsageRecord(t, a, keeperWindowUsageSeed{Dedupe: "other", Timestamp: now.Add(-time.Minute), AuthIndex: "other-index", InputTokens: 500})
	got, err := a.enrichAntigravityQuota(ctx, cfg, "ag.json", files, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if got.Groups[0].Buckets[0].Usage.Records != 0 {
		t.Fatal("GPT request attributed to Gemini")
	}
	short, week := got.Groups[1].Buckets[0].Usage, got.Groups[1].Buckets[1].Usage
	if short == nil || week == nil || short.Records != 1 || short.TotalTokens != 15 || week.Records != 2 || week.TotalTokens != 40 {
		t.Fatalf("wrong windows: %#v %#v", short, week)
	}
	if got.Groups[1].EffectivePriority != -1 || got.Groups[0].EffectivePriority != 0 {
		t.Fatal("exhaustion affected the wrong group")
	}
	if snapshot.Groups[1].Buckets[0].Usage != nil {
		t.Fatal("enrichment mutated the cached quota observation")
	}
	data, err := os.ReadFile(a.groupSchedulerPath())
	if err != nil {
		t.Fatal(err)
	}
	var exported groupscheduler.Snapshot
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported.Rules) != 2 || exported.Rules[1].EffectivePriority != -1 {
		t.Fatal("routing snapshot was not published")
	}
	// Policy updates read fresh configuration without replacing cached quotas.
	_, err = a.db.ExecContext(ctx, `INSERT INTO antigravity_group_policies (auth_name,group_id,priority,model_patterns) VALUES (?,?,?,?)`, "ag.json", "gpt", 12, `["gpt-*"]`)
	if err != nil {
		t.Fatal(err)
	}
	got, err = a.enrichAntigravityQuota(ctx, cfg, "ag.json", files, snapshot)
	if err != nil || got.Groups[1].Priority != 12 || got.Groups[1].EffectivePriority != -1 {
		t.Fatalf("policy lost: %#v %v", got, err)
	}
	snapshot.Groups[1].Buckets[0].RemainingPercent = 100
	got, err = a.enrichAntigravityQuota(ctx, cfg, "ag.json", files, snapshot)
	if err != nil || got.Groups[1].EffectivePriority != 12 {
		t.Fatal("group failed to restore configured priority")
	}
}

func TestAntigravityUnknownWindowsRemainUnknownAndDryRunDoesNotChangeRouting(t *testing.T) {
	a := newQuotaTestApp(t)
	ctx := context.Background()
	now := time.Now().UTC()
	cfg, err := a.loadConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cfg.CodexKeeper.DryRun = true
	snapshot := antigravityQuotaSnapshot{CheckedAt: &now, Groups: []antigravityQuotaGroup{{ID: "unknown", Label: "Unknown", Buckets: []antigravityQuotaBucket{{Window: "rolling", RemainingPercent: 0}}}}}
	got, err := a.enrichAntigravityQuota(ctx, cfg, "ag.json", []map[string]any{{"name": "ag.json", "priority": 8}}, snapshot)
	if err != nil || got.Groups[0].Buckets[0].Usage != nil || got.Groups[0].EffectivePriority != 8 {
		t.Fatalf("unknown quota was fabricated: %#v %v", got, err)
	}
}
