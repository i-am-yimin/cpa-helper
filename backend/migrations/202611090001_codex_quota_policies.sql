-- +goose Up
CREATE TABLE codex_quota_policies (
    auth_name TEXT PRIMARY KEY,
    policy_json TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- +goose Down
DROP TABLE codex_quota_policies;
