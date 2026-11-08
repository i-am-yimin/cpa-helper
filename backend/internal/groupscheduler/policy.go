// Package groupscheduler implements model-group scheduling without changing a
// credential's global priority. Its wire format follows CPA Core LTS v8.
package groupscheduler

import (
	"path"
	"strings"
	"time"
)

type Rule struct {
	AuthName          string    `json:"auth_name"`
	AuthIndex         string    `json:"auth_index"`
	GroupID           string    `json:"group_id"`
	Models            []string  `json:"models"`
	Priority          int       `json:"priority"`
	EffectivePriority int       `json:"effective_priority"`
	ValidUntil        time.Time `json:"valid_until"`
}
type Snapshot struct {
	Version     int       `json:"version"`
	GeneratedAt time.Time `json:"generated_at"`
	Rules       []Rule    `json:"rules"`
}
type Candidate struct {
	ID         string
	Provider   string
	Priority   int
	Status     string
	Attributes map[string]string
	Metadata   map[string]any
}

// MatchScore makes exact model mappings win over broad wildcard groups. A tie
// between different groups is ambiguous and must never pick a group by order.
func MatchScore(patterns []string, model string) int {
	model = strings.ToLower(strings.TrimSpace(model))
	best := -1
	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if pattern == model {
			return 1000000 + len(pattern)
		}
		matched, err := path.Match(pattern, model)
		if err == nil && matched {
			score := len(strings.NewReplacer("*", "", "?", "").Replace(pattern))
			if score > best {
				best = score
			}
		}
	}
	return best
}

func GroupForModel(rules []Rule, model string) (int, bool) {
	selected, best, ambiguous := -1, -1, false
	for i, rule := range rules {
		score := MatchScore(rule.Models, model)
		if score < 0 {
			continue
		}
		if score > best {
			selected, best, ambiguous = i, score, false
		} else if score == best {
			ambiguous = true
		}
	}
	return selected, selected >= 0 && !ambiguous
}

// EffectivePriority only consumes fresh observations. A stale snapshot cannot
// keep a recovered group demoted forever. Manual priorities below -1 are fixed.
func EffectivePriority(priority int, remaining []float64, threshold int, fresh bool) int {
	if priority < -1 || !fresh || len(remaining) == 0 {
		return priority
	}
	for _, value := range remaining {
		if 100-value >= float64(threshold) {
			return -1
		}
	}
	return priority
}

func PriorityFor(candidate Candidate, model string, snapshot Snapshot, now time.Time) (int, bool) {
	if !strings.EqualFold(candidate.Provider, "antigravity") || snapshot.Version != 1 {
		return candidate.Priority, false
	}
	rules := []Rule{}
	for _, rule := range snapshot.Rules {
		// CPA's auth ID is commonly the auth-file path. Match names exactly or by
		// basename, and use the runtime index when the host supplies it.
		id := strings.ReplaceAll(candidate.ID, "\\", "/")
		matches := candidate.ID == rule.AuthName || path.Base(id) == rule.AuthName
		if index := candidate.Attributes["auth_index"]; index != "" && rule.AuthIndex != "" {
			matches = index == rule.AuthIndex
		}
		if matches {
			rules = append(rules, rule)
		}
	}
	selected, ok := GroupForModel(rules, model)
	if !ok {
		return candidate.Priority, false
	}
	rule := rules[selected]
	if !now.Before(rule.ValidUntil) {
		return rule.Priority, true
	}
	return rule.EffectivePriority, true
}
