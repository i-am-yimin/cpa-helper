-- +goose Up
CREATE TABLE antigravity_group_policies (
    auth_name TEXT NOT NULL,
    group_id TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    model_patterns TEXT NOT NULL,
    PRIMARY KEY (auth_name, group_id)
);

-- +goose Down
DROP TABLE antigravity_group_policies;
