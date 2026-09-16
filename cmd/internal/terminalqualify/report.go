// Package terminalqualify measures a candidate against required terminal behavior.
// It is diagnostic tooling, not the production terminal abstraction.
package terminalqualify

import (
	"encoding/json"
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
	Target            string      `json:"target"`
	Comparison        string      `json:"comparison,omitempty"`
	ID                string      `json:"id"`
	Capability        string      `json:"capability"`
	Source            string      `json:"source,omitempty"`
	Status            Status      `json:"status"`
	Detail            string      `json:"detail,omitempty"`
	Expected          Observation `json:"expected,omitempty"`
	Observed          Observation `json:"observed,omitempty"`
	ExpectedTruncated bool        `json:"expected_truncated"`
	ObservedTruncated bool        `json:"observed_truncated"`
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

// boundedEvidence copies presentation evidence, limiting each observation to 64
// entries and 4096 JSON bytes (including escaped keys and values). The flag means
// information was omitted, shortened, or normalized to valid UTF-8. Qualification
// must compare the original observations, never these presentation copies.
func boundedEvidence(original Observation) (Observation, bool) {
	if original == nil {
		return nil, false
	}
	const maxBytes, maxEntries = 4096, 64
	result := Observation{}
	keys := make([]string, 0, len(original))
	for key := range original {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	truncated := false
	for _, rawKey := range keys {
		if len(result) == maxEntries {
			truncated = true
			break
		}
		key := strings.ToValidUTF8(rawKey, "�")
		value := strings.ToValidUTF8(original[rawKey], "�")
		if key != rawKey || value != original[rawKey] {
			truncated = true
		}
		if _, exists := result[key]; exists {
			truncated = true
			continue
		}
		// Reject oversize keys before encoding; a UTF-8 JSON key cannot shrink.
		if len(key) > maxBytes {
			truncated = true
			continue
		}
		result[key] = ""
		encoded, _ := json.Marshal(result)
		if len(encoded) > maxBytes {
			delete(result, key)
			truncated = true
			continue
		}
		// Binary search bounds work even for control characters, whose JSON escapes
		// occupy more bytes than their in-memory representation.
		lo, hi := 0, min(len(value), maxBytes)
		for lo < hi {
			mid := lo + (hi-lo+1)/2
			result[key] = utf8Prefix(value, mid)
			encoded, _ = json.Marshal(result)
			if len(encoded) <= maxBytes {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		result[key] = utf8Prefix(value, lo)
		if result[key] != value {
			truncated = true
		}
	}
	return result, truncated
}

func utf8Prefix(s string, limit int) string {
	for limit > 0 && !utf8.ValidString(s[:limit]) {
		limit--
	}
	return s[:limit]
}
