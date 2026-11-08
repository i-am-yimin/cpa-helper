package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	backendApp "cpa-helper/backend/internal/app"
)

func TestAntigravityQuotaRequiresAccountStatusAccessAndAdminForForcedRefresh(t *testing.T) {
	cpa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v0/management/auth-files":
			_, _ = w.Write([]byte(`{"files":[{"name":"ag.json","type":"antigravity","auth_index":"ag-index","project_id":"project"}]}`))
		case "/v0/management/api-call":
			_, _ = w.Write([]byte(`{"status_code":200,"body":{"groups":[{"displayName":"Gemini","buckets":[{"window":"5h","remainingFraction":0.5}]}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cpa.Close()
	t.Setenv("CPA_HELPER_DATA_DIR", t.TempDir())
	a, err := backendApp.New()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	handler := a.Routes()
	admin := requestJSON(t, handler, http.MethodPost, "/api/auth/setup", map[string]any{
		"username": "admin", "password": "test-password", "nickname": "Admin",
	}, nil, nil)
	requestJSON(t, handler, http.MethodPut, "/api/settings", map[string]any{
		"cliaproxy_url": cpa.URL, "management_key": "test-management-key", "collector_enabled": false,
	}, admin, nil)
	requestJSON(t, handler, http.MethodPost, "/api/users", map[string]any{
		"username": "member", "password": "member-password", "nickname": "Member", "is_admin": false,
	}, admin, nil)
	member := requestJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]any{
		"username": "member", "password": "member-password",
	}, nil, nil)
	path := "/api/codex-keeper/accounts/ag.json/quota"
	requestJSONExpectStatus(t, handler, http.MethodGet, path, nil, nil, http.StatusUnauthorized)
	requestJSONExpectStatus(t, handler, http.MethodGet, path, nil, member, http.StatusForbidden)
	requestJSON(t, handler, http.MethodPut, "/api/settings", map[string]any{"allow_user_account_status": true}, admin, nil)
	requestJSON(t, handler, http.MethodGet, path, nil, member, nil)
	requestJSONExpectStatus(t, handler, http.MethodPost, path, nil, member, http.StatusForbidden)
	requestJSON(t, handler, http.MethodPost, path, nil, admin, nil)
	requestJSONExpectStatus(t, handler, http.MethodDelete, path, nil, admin, http.StatusMethodNotAllowed)
}
