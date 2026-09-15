// Package terminalqualify measures a candidate against required terminal behavior.
// It is diagnostic tooling, not the production terminal abstraction.
package terminalqualify

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type Observation map[string]string
type Status string

const (
	Pass       Status = "pass"
	Fail       Status = "fail"
	NotCovered Status = "not-covered"
)

type Result struct {
	ID         string `json:"id"`
	Capability string `json:"capability"`
	Source     string `json:"source,omitempty"`
	Status     Status `json:"status"`
	Detail     string `json:"detail,omitempty"`
}
type Report struct {
	Candidate string   `json:"candidate"`
	Required  []string `json:"required"`
	Results   []Result `json:"results"`
}

func (r Report) Validate() error {
	if len(r.Required) == 0 {
		return fmt.Errorf("empty required profile")
	}
	required := map[string]bool{}
	for _, id := range r.Required {
		if id == "" || required[id] {
			return fmt.Errorf("empty or duplicate required ID %q", id)
		}
		required[id] = true
	}
	seen := map[string]bool{}
	for _, v := range r.Results {
		if !required[v.ID] || seen[v.ID] {
			return fmt.Errorf("unexpected or duplicate result %q", v.ID)
		}
		if v.Status != Pass && v.Status != Fail && v.Status != NotCovered {
			return fmt.Errorf("invalid status %q", v.Status)
		}
		seen[v.ID] = true
	}
	if len(seen) != len(required) {
		return fmt.Errorf("incomplete profile: %d results for %d requirements", len(seen), len(required))
	}
	return nil
}
func (r Report) Qualified() bool {
	if r.Validate() != nil {
		return false
	}
	for _, v := range r.Results {
		if v.Status != Pass {
			return false
		}
	}
	return true
}

// Compare checks the full expected subset, independent of presentation bounds.
func Compare(want, got Observation) string {
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v, ok := got[k]
		if !ok {
			return fmt.Sprintf("%s: missing; want %q", k, want[k])
		}
		if v != want[k] {
			return fmt.Sprintf("%s: got %q; want %q", k, v, want[k])
		}
	}
	return ""
}
func boundedDetail(s string) string {
	const limit = 4096
	if len(s) <= limit {
		return s
	}
	s = s[:limit-3]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s + "..."
}
func (r Report) Summary() string {
	counts := map[Status]int{}
	for _, v := range r.Results {
		counts[v.Status]++
	}
	return fmt.Sprintf("%d pass, %d fail, %d not-covered; qualified=%t", counts[Pass], counts[Fail], counts[NotCovered], r.Qualified())
}
func mismatchDetails(parts ...string) string { return boundedDetail(strings.Join(parts, "; ")) }
