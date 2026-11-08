package main

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestBindingPreservesRolesAndRejectsIdentityReplacement(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		setup     string
		wantError bool
	}{
		{"enabled administrator", "", false},
		{"non-admin", "UPDATE users SET is_admin=0 WHERE id=1", true},
		{"disabled admin", "UPDATE users SET disabled_at='disabled' WHERE id=1", true},
		{"identity owned by someone else", "INSERT INTO oidc_identities VALUES ('https://cas.example/application/o/panel/', 'verified-subject', 2)", true},
		{"different existing binding", "INSERT INTO oidc_identities VALUES ('https://other.example/', 'other-subject', 1)", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			_, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, is_admin INTEGER, disabled_at TEXT);
CREATE TABLE oidc_identities (issuer TEXT, subject TEXT, user_id INTEGER UNIQUE, PRIMARY KEY(issuer,subject));
INSERT INTO users VALUES(1,'existing-admin',1,NULL),(2,'other-user',0,NULL);`)
			if err != nil {
				t.Fatal(err)
			}
			if scenario.setup != "" {
				if _, err := db.Exec(scenario.setup); err != nil {
					t.Fatal(err)
				}
			}
			cfg := binding{Issuer: "https://cas.example/application/o/panel/", Subject: "verified-subject", LocalUsername: "existing-admin"}
			changed, err := bindIdentity(db, cfg)
			if (err != nil) != scenario.wantError {
				t.Fatalf("error=%v, wantError=%v", err, scenario.wantError)
			}
			if scenario.wantError {
				return
			}
			if !changed {
				t.Fatal("expected initial binding")
			}
			changed, err = bindIdentity(db, cfg)
			if err != nil || changed {
				t.Fatalf("idempotent binding: changed=%v, error=%v", changed, err)
			}
			var admin int
			if err := db.QueryRow("SELECT is_admin FROM users WHERE id=1").Scan(&admin); err != nil || admin != 1 {
				t.Fatal("administrator role changed")
			}
		})
	}
}
