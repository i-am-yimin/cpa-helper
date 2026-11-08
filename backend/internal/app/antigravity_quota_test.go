package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const antigravityQuotaFixture = `{"groups":[{"displayName":"Gemini Models","buckets":[
 {"bucketId":"week","window":"weekly","remainingFraction":1,"resetTime":"2027-01-15T12:00:00Z"},
 {"bucketId":"short","window":"5h","remainingFraction":0,"resetTime":"2027-01-08T17:00:00Z"},
 {"bucketId":"unknown","window":"daily"} ]},
 {"display_name":"Claude and GPT models","buckets":[{"bucket_id":"claude","window":"5h","remaining_fraction":0.4567,"reset_time":"invalid"}]},
 {"displayName":"Unknown","buckets":[{"remainingFraction":null}]},
 {"buckets":[{"remainingFraction":2},{"remainingFraction":-0.2}]}]}`

func TestAntigravityQuotaParsesGroupsWithoutTreatingUnknownAsZero(t *testing.T) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(antigravityQuotaFixture), &payload); err != nil {
		t.Fatal(err)
	}
	groups := parseAntigravityQuotaGroups(payload)
	if len(groups) != 3 || len(groups[0].Buckets) != 2 || groups[0].Buckets[0].ID != "short" {
		t.Fatalf("unexpected groups: %#v", groups)
	}
	if bucket := groups[0].Buckets[0]; bucket.RemainingPercent != 0 || bucket.ResetAt == nil {
		t.Fatalf("exhausted bucket: %#v", bucket)
	}
	if bucket := groups[1].Buckets[0]; bucket.RemainingPercent != 45.7 || bucket.ResetAt != nil {
		t.Fatalf("fraction/invalid reset: %#v", bucket)
	}
	if groups[2].Buckets[0].RemainingPercent != 100 || groups[2].Buckets[1].RemainingPercent != 0 {
		t.Fatal("out-of-range values must be clamped")
	}
}

func TestAntigravityQuotaCacheCoalescesAndKeepsLastGoodResult(t *testing.T) {
	var cache antigravityQuotaCache
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	load := func() antigravityQuotaSnapshot {
		calls.Add(1)
		close(started)
		<-release
		now := time.Now()
		return antigravityQuotaSnapshot{CheckedAt: &now, AttemptedAt: now, Groups: []antigravityQuotaGroup{{Label: "Gemini"}}}
	}
	var wait sync.WaitGroup
	for i := 0; i < 16; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := cache.get(context.Background(), "credential", false, load)
			if err != nil || len(result.Groups) != 1 {
				t.Errorf("get: %#v, %v", result, err)
			}
		}()
	}
	<-started
	close(release)
	wait.Wait()
	if calls.Load() != 1 {
		t.Fatalf("upstream calls = %d, want 1", calls.Load())
	}
	errorText := "upstream HTTP 429"
	failed, err := cache.get(context.Background(), "credential", true, func() antigravityQuotaSnapshot {
		calls.Add(1)
		return antigravityQuotaSnapshot{Error: &errorText, AttemptedAt: time.Now()}
	})
	if err != nil || !failed.Stale || failed.Error == nil || failed.CheckedAt == nil || len(failed.Groups) != 1 || calls.Load() != 2 {
		t.Fatalf("last good result not preserved: %#v, %v", failed, err)
	}
	_, _ = cache.get(context.Background(), "credential", false, load)
	if calls.Load() != 2 {
		t.Fatal("failure backoff was bypassed")
	}
}

func TestAntigravityQuotaCacheExpiresAtResetAndDoesNotCacheCancellation(t *testing.T) {
	var cache antigravityQuotaCache
	reset := time.Now().Add(time.Minute)
	_, _ = cache.get(context.Background(), "reset", false, func() antigravityQuotaSnapshot {
		return antigravityQuotaSnapshot{Groups: []antigravityQuotaGroup{{Buckets: []antigravityQuotaBucket{{ResetAt: &reset}}}}}
	})
	if !cache.entries["reset"].Expires.Equal(reset) {
		t.Fatal("cache must expire at the next reset")
	}
	ctx, cancel := context.WithCancel(context.Background())
	_, err := cache.get(ctx, "cancelled", false, func() antigravityQuotaSnapshot {
		cancel()
		return antigravityQuotaSnapshot{}
	})
	if err == nil || len(cache.inFlight) != 0 {
		t.Fatal("cancelled request must release in-flight waiters")
	}
	if _, exists := cache.entries["cancelled"]; exists {
		t.Fatal("cancelled request poisoned the cache")
	}
	entry := cache.entries["reset"]
	entry.Expires = time.Now().Add(-time.Second)
	cache.entries["reset"] = entry
	loaded := false
	_, _ = cache.get(context.Background(), "reset", false, func() antigravityQuotaSnapshot {
		loaded = true
		return antigravityQuotaSnapshot{}
	})
	if !loaded {
		t.Fatal("expired cache must be refreshed")
	}
}

func TestAntigravityQuotaUsesManagedTokenAndNeverMutatesCredentials(t *testing.T) {
	var calls atomic.Int32
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-management-key" {
			t.Error("management authentication missing")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v0/management/auth-files":
			_, _ = w.Write([]byte(`{"files":[{"name":"ag.json","type":"antigravity","auth_index":"ag-index"},{"name":"codex.json","type":"codex","auth_index":"codex-index"},{"name":"disabled.json","type":"antigravity","disabled":true}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v0/management/auth-files/download":
			_, _ = w.Write([]byte(`{"project_id":"private-project","access_token":"private-access-token","refresh_token":"private-refresh-token"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v0/management/api-call":
			calls.Add(1)
			var request struct {
				Index  string            `json:"auth_index"`
				Method string            `json:"method"`
				URL    string            `json:"url"`
				Header map[string]string `json:"header"`
				Data   string            `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			if request.Index != "ag-index" || request.Method != "POST" || request.Header["Authorization"] != "Bearer $TOKEN$" || request.Data != `{"project":"private-project"}` || !strings.HasSuffix(request.URL, ":retrieveUserQuotaSummary") {
				t.Errorf("unexpected quota request: %#v", request)
			}
			body := antigravityQuotaFixture
			if status != http.StatusOK {
				body = `{"error":{"message":"private-access-token private-project"}}`
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": status, "body": body})
		default:
			t.Errorf("quota lookup attempted unexpected operation: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	a := newQuotaTestApp(t)
	configureKeeperTestCPA(t, a, server.URL, nil)
	for i := 0; i < 2; i++ {
		quota, err := a.getAntigravityQuota(context.Background(), "ag.json", false)
		if err != nil || len(quota.Groups) != 3 || quota.Error != nil {
			t.Fatalf("quota: %#v, %v", quota, err)
		}
		encoded, _ := json.Marshal(quota)
		if strings.Contains(string(encoded), "private-") {
			t.Fatal("credential metadata leaked to quota response")
		}
	}
	if calls.Load() != 1 {
		t.Fatal("cached read repeated upstream query")
	}
	status = http.StatusForbidden
	quota, err := a.getAntigravityQuota(context.Background(), "ag.json", true)
	encoded, _ := json.Marshal(quota)
	if err != nil || !quota.Stale || quota.Error == nil || !strings.Contains(*quota.Error, "403") || strings.Contains(string(encoded), "private-") || calls.Load() != 2 {
		t.Fatalf("unsafe error result: %s, %v", encoded, err)
	}
	for _, name := range []string{"codex.json", "missing.json", "disabled.json"} {
		if _, err := a.getAntigravityQuota(context.Background(), name, true); err == nil {
			t.Fatalf("must reject %s", name)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("unsupported, disabled or missing credentials must not query upstream")
	}
}
