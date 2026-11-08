// cpa-group-scheduler is a native CPA Core v8 scheduler plugin. Build on Linux
// using -buildmode=c-shared; the native ABI adapter is in native_linux.go.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"cpa-helper/backend/internal/groupscheduler"
)

var scheduler struct {
	sync.Mutex
	snapshot  groupscheduler.Snapshot
	checked   time.Time
	heartbeat time.Time
	cursors   map[string]uint64
}

func main() {}

func snapshotPath() string { return strings.TrimSpace(os.Getenv("CPA_GROUP_SCHEDULER_SNAPSHOT")) }

func handleMethod(method string, raw []byte) ([]byte, error) {
	switch method {
	case "plugin.register", "plugin.reconfigure":
		if snapshotPath() == "" {
			return nil, fmt.Errorf("CPA_GROUP_SCHEDULER_SNAPSHOT must name the shared Helper snapshot file")
		}
		return json.Marshal(map[string]any{"ok": true, "result": map[string]any{
			"schema_version": 7,
			"metadata":       map[string]any{"Name": "cpa-group-scheduler", "Version": "1.0.0", "Author": "CPA-Helper", "GitHubRepository": "https://github.com/i-am-yimin/cpa-helper"},
			"capabilities":   map[string]any{"scheduler": true, "scheduler_across_priorities": true},
		}})
	case "scheduler.pick":
		var request struct {
			Model      string
			Candidates []groupscheduler.Candidate
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			return nil, err
		}
		scheduler.Lock()
		defer scheduler.Unlock()
		now := time.Now().UTC()
		if scheduler.checked.IsZero() || now.Sub(scheduler.checked) >= time.Second {
			// Drop, rather than indefinitely retaining, an unreadable or corrupt file.
			scheduler.snapshot = groupscheduler.Snapshot{}
			if data, err := os.ReadFile(snapshotPath()); err == nil {
				_ = json.Unmarshal(data, &scheduler.snapshot)
			}
			scheduler.checked = now
		}
		ids, matched := highestCandidates(request.Candidates, request.Model, scheduler.snapshot, now)
		if !matched || len(ids) == 0 {
			return json.Marshal(map[string]any{"ok": true, "result": map[string]any{"Handled": false}})
		}
		sort.Strings(ids)
		selected := ids[0]
		strategy := os.Getenv("CPA_GROUP_SCHEDULER_STRATEGY")
		if strategy != "fill-first" {
			if scheduler.cursors == nil || len(scheduler.cursors) > 256 {
				scheduler.cursors = map[string]uint64{}
			}
			key := request.Model + "\x00" + strings.Join(ids, "\x00")
			selected = ids[scheduler.cursors[key]%uint64(len(ids))]
			scheduler.cursors[key]++
		}
		if now.Sub(scheduler.heartbeat) > 30*time.Second {
			heartbeat, _ := json.Marshal(map[string]any{"picked_at": now})
			// A heartbeat proves an actual model-group decision, not merely a loaded file.
			if err := os.WriteFile(snapshotPath()+".status", heartbeat, 0600); err == nil {
				scheduler.heartbeat = now
			}
		}
		return json.Marshal(map[string]any{"ok": true, "result": map[string]any{"AuthID": selected, "Handled": true}})
	default:
		return nil, fmt.Errorf("unsupported plugin method: %s", method)
	}
}

func highestCandidates(candidates []groupscheduler.Candidate, model string, snapshot groupscheduler.Snapshot, now time.Time) ([]string, bool) {
	best := 0
	ids := []string{}
	matched := false
	for _, candidate := range candidates {
		if candidate.ID == "" || candidate.Status == "disabled" {
			continue
		}
		priority, applied := groupscheduler.PriorityFor(candidate, model, snapshot, now)
		matched = matched || applied
		if len(ids) == 0 || priority > best {
			best = priority
			ids = []string{candidate.ID}
		} else if priority == best {
			ids = append(ids, candidate.ID)
		}
	}
	return ids, matched
}
