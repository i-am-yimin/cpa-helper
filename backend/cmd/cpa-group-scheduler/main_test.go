package main

import (
	"cpa-helper/backend/internal/groupscheduler"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestModelRequestUsesGroupPriorityAcrossGlobalPriorityTiers(t *testing.T) {
	now := time.Now()
	candidates := []groupscheduler.Candidate{{ID: "ag.json", Provider: "antigravity", Priority: 0}, {ID: "codex.json", Provider: "codex", Priority: 4}}
	snapshot := groupscheduler.Snapshot{Version: 1, Rules: []groupscheduler.Rule{
		{AuthName: "ag.json", Models: []string{"gpt-*"}, Priority: 8, EffectivePriority: -1, ValidUntil: now.Add(time.Minute)},
		{AuthName: "ag.json", Models: []string{"gemini-*"}, Priority: 8, EffectivePriority: 8, ValidUntil: now.Add(time.Minute)},
	}}
	ids, ok := highestCandidates(candidates, "gpt-5", snapshot, now)
	if !ok || len(ids) != 1 || ids[0] != "codex.json" {
		t.Fatalf("exhausted group beat Codex: %v", ids)
	}
	snapshot.Rules[0].EffectivePriority = 8
	ids, ok = highestCandidates(candidates, "gpt-5", snapshot, now)
	if !ok || ids[0] != "ag.json" {
		t.Fatal("recovered GPT group failed to beat Codex")
	}
}

func TestPluginProtocolRegistersAcrossTiersAndPicksFromSharedSnapshot(t *testing.T) {
	file := filepath.Join(t.TempDir(), "snapshot.json")
	t.Setenv("CPA_GROUP_SCHEDULER_SNAPSHOT", file)
	snapshot := groupscheduler.Snapshot{Version: 1, Rules: []groupscheduler.Rule{{AuthName: "ag.json", Models: []string{"gpt-*"}, Priority: 8, EffectivePriority: 8, ValidUntil: time.Now().Add(time.Minute)}}}
	data, _ := json.Marshal(snapshot)
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	register, err := handleMethod("plugin.register", nil)
	if err != nil || !strings.Contains(string(register), `"scheduler_across_priorities":true`) {
		t.Fatalf("bad registration: %s %v", register, err)
	}
	scheduler.checked = time.Time{}
	response, err := handleMethod("scheduler.pick", []byte(`{"Model":"gpt-5","Candidates":[{"ID":"ag.json","Provider":"antigravity","Priority":0},{"ID":"codex.json","Provider":"codex","Priority":4}]}`))
	if err != nil || !strings.Contains(string(response), `"AuthID":"ag.json"`) {
		t.Fatalf("bad pick: %s %v", response, err)
	}
	if _, err := os.Stat(file + ".status"); err != nil {
		t.Fatal("missing proof of routing heartbeat")
	}
}
