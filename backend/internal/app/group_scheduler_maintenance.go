package app

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"
)

// Quota observations keep refreshing without an open browser. Starting the
// service alone never queries OAuth accounts: a saved policy or an existing
// snapshot opts this integration in.
func (a *App) startGroupSchedulerMaintenance(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	a.groupSchedulerCancel = cancel
	a.groupSchedulerDone = make(chan struct{})
	go func() {
		defer close(a.groupSchedulerDone)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			cycle, stop := context.WithTimeout(ctx, 2*time.Minute)
			if err := a.refreshGroupSchedulerQuotas(cycle); err != nil && ctx.Err() == nil {
				log.Printf("group scheduler maintenance: %v", err)
			}
			stop()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (a *App) refreshGroupSchedulerQuotas(ctx context.Context) error {
	var count int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM antigravity_group_policies`).Scan(&count); err != nil {
		return err
	}
	if count == 0 && strings.TrimSpace(os.Getenv("CPA_HELPER_SCHEDULER_SNAPSHOT")) == "" {
		if _, err := os.Stat(a.groupSchedulerPath()); err != nil {
			return nil
		}
	}
	cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}
	files, err := a.listKeeperRemoteAuthFiles(ctx, cfg)
	if err != nil {
		return err
	}
	var failures []error
	for _, file := range files {
		if keeperCredentialProvider(file) != "antigravity" || keeperBool(file["disabled"]) {
			continue
		}
		if _, err := a.getAntigravityQuota(ctx, keeperString(file["name"]), false); err != nil {
			failures = append(failures, err)
			if ctx.Err() != nil {
				break
			}
		}
	}
	return errors.Join(failures...)
}
