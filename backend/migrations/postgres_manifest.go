package migrations

import (
	"bytes"
	"strings"
)

// PostgreSQLBaselineVersion is also the schema understood by the atomic SQLite importer.
// Regenerate the import baseline/manifest when advancing the current schema.
const PostgreSQLBaselineVersion int64 = 202611090001

// PostgreSQLTables is the baseline's dependency order. Keep it in sync with both
// schemas; the importer refuses missing/extra tables or columns.
var PostgreSQLTables = []string{"app_settings", "codex_keeper_auth_states", "codex_keeper_runs", "codex_keeper_run_accounts", "collector_state", "model_prices", "oidc_flows", "pricing_calendar_settings", "pricing_holiday_years", "pricing_holidays", "tutorials", "usage_hourly_rollups", "usage_records", "usage_ingest_dedup", "usage_rollup_state", "users", "oidc_identities", "oidc_sessions", "quota_cards", "quota_resets", "user_api_keys", "user_card_shop_favorites", "user_card_shop_tags", "user_quota_charges", "quota_card_deductions", "antigravity_group_policies", "codex_quota_policies"}

// PostgresImportSchema builds the current import baseline without rewriting an
// already shipped migration or duplicating the historical schema snapshot.
func PostgresImportSchema() ([]byte, error) {
	var schema bytes.Buffer
	for _, name := range []string{"postgres/202609280001_baseline.sql", "postgres/202611080001_antigravity_group_policies.sql", "postgres/202611090001_codex_quota_policies.sql"} {
		data, err := PostgresFS.ReadFile(name)
		if err != nil {
			return nil, err
		}
		up, _, _ := strings.Cut(string(data), "-- +goose Down")
		schema.WriteString(up)
		schema.WriteByte('\n')
	}
	return schema.Bytes(), nil
}
