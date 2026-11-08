// One-time, root-only identity binding for an explicitly verified IdP account.
// Build from backend/: go build -o bin/bind-oidc ../deploy/bind-oidc.go
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type binding struct {
	Issuer        string `json:"issuer"`
	Subject       string `json:"subject"`
	LocalUsername string `json:"local_username"`
}

func bindIdentity(db *sql.DB, cfg binding) (bool, error) {
	u, err := url.Parse(cfg.Issuer)
	if err != nil || u.Scheme != "https" || u.Host == "" || cfg.Subject == "" || strings.TrimSpace(cfg.LocalUsername) == "" {
		return false, errors.New("explicit HTTPS issuer, subject and local username are required")
	}
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var id int
	var admin bool
	var disabled sql.NullString
	if err := tx.QueryRow("SELECT id, is_admin, disabled_at FROM users WHERE username = ?", cfg.LocalUsername).Scan(&id, &admin, &disabled); err != nil {
		return false, err
	}
	if !admin || disabled.Valid {
		return false, errors.New("target must be an enabled existing administrator")
	}
	var boundID int
	err = tx.QueryRow("SELECT user_id FROM oidc_identities WHERE issuer = ? AND subject = ?", cfg.Issuer, cfg.Subject).Scan(&boundID)
	if err == nil {
		if boundID != id {
			return false, errors.New("identity already belongs to another user")
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM oidc_identities WHERE user_id = ?", id).Scan(&count); err != nil {
		return false, err
	}
	if count != 0 {
		return false, errors.New("administrator already has a different identity; refusing replacement")
	}
	if _, err := tx.Exec("INSERT INTO oidc_identities (issuer, subject, user_id) VALUES (?, ?, ?)", cfg.Issuer, cfg.Subject, id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func main() {
	config := flag.String("config", "/opt/ai-gateway/secrets/oidc.json", "root-owned verified identity document")
	database := flag.String("database", "/opt/ai-gateway/helper/db/cpa_helper.sqlite3", "existing panel database")
	flag.Parse()
	if os.Geteuid() != 0 {
		log.Fatal("run as root")
	}
	info, err := os.Stat(*config)
	if err != nil {
		log.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		log.Fatal("identity document must be private (0600)")
	}
	raw, err := os.ReadFile(*config)
	if err != nil {
		log.Fatal(err)
	}
	var cfg binding
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(*database); err != nil {
		log.Fatal(err)
	}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(*database)}
	q := u.Query()
	q.Set("mode", "rw")
	q.Set("_pragma", "busy_timeout(10000)")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	backupDir := filepath.Join("/opt/ai-gateway/backups", "sso-binding-"+time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		log.Fatal(err)
	}
	backup := filepath.Join(backupDir, "helper.sqlite3")
	if _, err := db.Exec("VACUUM INTO ?", backup); err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(backup, 0600); err != nil {
		log.Fatal(err)
	}
	changed, err := bindIdentity(db, cfg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Verified administrator identity binding ready (created=%t); private database snapshot saved.\n", changed)
}
