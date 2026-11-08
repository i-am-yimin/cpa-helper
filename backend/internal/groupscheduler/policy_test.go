package groupscheduler

import (
	"testing"
	"time"
)

func TestGroupPrioritiesAreIndependentAndRecover(t *testing.T) {
	now := time.Now()
	rules := []Rule{
		{AuthName: "ag.json", GroupID: "gemini", Models: []string{"gemini-*"}, Priority: 8, EffectivePriority: 8, ValidUntil: now.Add(time.Minute)},
		{AuthName: "ag.json", GroupID: "claude-gpt", Models: []string{"claude-*", "gpt-*"}, Priority: 6, EffectivePriority: -1, ValidUntil: now.Add(time.Minute)},
	}
	candidate := Candidate{ID: "/auth/ag.json", Provider: "antigravity", Priority: 0}
	snapshot := Snapshot{Version: 1, Rules: rules}
	for _, tc := range []struct {
		model string
		want  int
	}{{"gemini-3-pro", 8}, {"gpt-5", -1}, {"claude-sonnet", -1}, {"unknown", 0}} {
		got, _ := PriorityFor(candidate, tc.model, snapshot, now)
		if got != tc.want {
			t.Errorf("%s priority=%d, want %d", tc.model, got, tc.want)
		}
	}
	if got, _ := PriorityFor(candidate, "gpt-5", snapshot, now.Add(2*time.Minute)); got != 6 {
		t.Fatalf("stale quota kept the group degraded: %d", got)
	}
	if got := EffectivePriority(6, []float64{100, 100}, 95, true); got != 6 {
		t.Fatal("group did not recover")
	}
	if got := EffectivePriority(6, []float64{100, 0}, 95, true); got != -1 {
		t.Fatal("weekly exhaustion ignored")
	}
	if got := EffectivePriority(-2, []float64{0}, 95, true); got != -2 {
		t.Fatal("manual low priority changed")
	}
	if got := EffectivePriority(21, []float64{0}, 95, true); got != -1 {
		t.Fatal("manual high priority failed to fall back")
	}
	if got := EffectivePriority(6, []float64{0}, 95, false); got != 6 {
		t.Fatal("stale observation affected routing")
	}
}

func TestSpecificGroupsAndAmbiguity(t *testing.T) {
	rules := []Rule{{Models: []string{"gemini-*"}}, {Models: []string{"gemini-4*", "argon*"}}}
	if i, ok := GroupForModel(rules, "gemini-4-pro"); !ok || i != 1 {
		t.Fatal("specific group must win")
	}
	rules = append(rules, Rule{Models: []string{"gemini-4*"}})
	if _, ok := GroupForModel(rules, "gemini-4-pro"); ok {
		t.Fatal("ambiguous mapping must not be attributed")
	}
}
