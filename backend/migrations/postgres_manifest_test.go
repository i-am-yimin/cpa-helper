package migrations

import (
	"strings"
	"testing"
)

func TestPostgresImportBaselineTracksLatestSchema(t *testing.T) {
	if PostgreSQLBaselineVersion != LatestVersion {
		t.Fatal("update the PostgreSQL import baseline and manifest with the schema migration")
	}
	if len(PostgreSQLTables) == 0 {
		t.Fatal("missing PostgreSQL import manifest")
	}
	schema, err := PostgresImportSchema()
	if err != nil || !strings.Contains(string(schema), "CREATE TABLE antigravity_group_policies") || strings.Contains(string(schema), "DROP TABLE antigravity_group_policies") {
		t.Fatalf("invalid import schema: %v", err)
	}
	if !strings.Contains(string(schema), "CREATE TABLE codex_quota_policies") || !strings.Contains(strings.Join(PostgreSQLTables, ","), "codex_quota_policies") || strings.Contains(string(schema), "DROP TABLE codex_quota_policies") {
		t.Fatal("Codex quota latch missing from PostgreSQL import schema or manifest")
	}
}
