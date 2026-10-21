// Package couchschedule defines the pure scheduled-task declaration and time rules.
package couchschedule

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const maxTaskSpecBytes = 32 * 1024

// TaskSpec is a validated task declaration. Timeout includes admission/build/run.
type TaskSpec struct {
	Version              int
	Cron                 string
	Timeout              time.Duration
	Scope, Owner, Missed string
	ScriptRun, Heavy     bool
}

// ParseTaskSpec accepts exactly the version-one declaration vocabulary.
func ParseTaskSpec(raw []byte) (TaskSpec, error) {
	var zero TaskSpec
	if len(raw) > maxTaskSpecBytes || !utf8.Valid(raw) {
		return zero, fmt.Errorf("schedule must be UTF-8 and at most %d bytes", maxTaskSpecBytes)
	}
	known := map[string]bool{"version": true, "cron": true, "timeout": true, "scope": true, "owner": true, "missed": true, "script_run": true, "heavy": true}
	owned, unknown, err := strictjson.SplitObject(raw, known, nil)
	if err != nil {
		return zero, err
	}
	if len(unknown) > 0 {
		return zero, fmt.Errorf("schedule contains unknown members")
	}
	for _, field := range []string{"version", "cron", "timeout"} {
		if _, ok := owned[field]; !ok {
			return zero, fmt.Errorf("schedule requires %s", field)
		}
	}
	wire := struct {
		Version   int    `json:"version"`
		Cron      string `json:"cron"`
		Timeout   string `json:"timeout"`
		Scope     string `json:"scope"`
		Owner     string `json:"owner"`
		Missed    string `json:"missed"`
		ScriptRun bool   `json:"script_run"`
		Heavy     bool   `json:"heavy"`
	}{Scope: "machine", Missed: "skip"}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return zero, fmt.Errorf("schedule fields: %w", err)
	}
	if wire.Version != 1 {
		return zero, fmt.Errorf("schedule version must be 1")
	}
	if _, err := NewCalendar(wire.Cron, time.UTC); err != nil {
		return zero, fmt.Errorf("schedule cron: %w", err)
	}
	timeout, err := time.ParseDuration(wire.Timeout)
	if err != nil || timeout <= 0 || timeout > 24*time.Hour {
		return zero, fmt.Errorf("schedule timeout must be positive and at most 24h")
	}
	if wire.Scope != "machine" && wire.Scope != "team" {
		return zero, fmt.Errorf("schedule scope must be machine or team")
	}
	if wire.Missed != "skip" && wire.Missed != "once" {
		return zero, fmt.Errorf("schedule missed must be skip or once")
	}
	_, hasOwner := owned["owner"]
	if (hasOwner || wire.Scope == "team") && !validOperatorSlug(wire.Owner) {
		return zero, fmt.Errorf("schedule owner must be a normalized operator slug")
	}
	return TaskSpec{Version: wire.Version, Cron: wire.Cron, Timeout: timeout, Scope: wire.Scope, Owner: wire.Owner, Missed: wire.Missed, ScriptRun: wire.ScriptRun, Heavy: wire.Heavy}, nil
}

// ValidTaskName permits a bounded ASCII directory component, never a path.
func ValidTaskName(name string) bool {
	if len(name) == 0 || len(name) > 128 || !asciiAlnum(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !asciiAlnum(name[i]) && name[i] != '-' && name[i] != '_' {
			return false
		}
	}
	return true
}

func asciiAlnum(b byte) bool { return b >= 'a' && b <= 'z' || b >= '0' && b <= '9' }

// OperatorSlug normalizes an effective Git user.name to its ASCII operator slug.
func OperatorSlug(name string) string {
	var out strings.Builder
	separator := false
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if separator && out.Len() > 0 {
				out.WriteByte('-')
			}
			out.WriteRune(r)
			separator = false
		} else {
			separator = true
		}
	}
	return out.String()
}

func validOperatorSlug(slug string) bool {
	return slug != "" && len(slug) <= 128 && OperatorSlug(slug) == slug
}

var operatorEntry = regexp.MustCompile(`^- ([a-z0-9]+(?:-[a-z0-9]+)*) \(([^()]+)\): (.+)$`)

// ParseOperators returns the slugs in the single level-two Operators section.
// Absence is permitted; OwnerAdmission refuses team tasks without a listed owner.
func ParseOperators(raw []byte) ([]string, error) {
	if len(raw) > 1024*1024 || !utf8.Valid(raw) {
		return nil, fmt.Errorf("team document must be bounded UTF-8")
	}
	var result []string
	seen := map[string]bool{}
	inside, found := false, false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "## Operators" {
			if found {
				return nil, fmt.Errorf("duplicate Operators section")
			}
			inside, found = true, true
			continue
		}
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			inside = false
			continue
		}
		if !inside || line == "" {
			continue
		}
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "+") {
			parts := operatorEntry.FindStringSubmatch(line)
			if parts == nil || !validOperatorSlug(parts[1]) || strings.TrimSpace(parts[2]) == "" || strings.TrimSpace(parts[3]) == "" {
				return nil, fmt.Errorf("malformed Operators entry %q", line)
			}
			if seen[parts[1]] {
				return nil, fmt.Errorf("duplicate operator %s", parts[1])
			}
			seen[parts[1]] = true
			result = append(result, parts[1])
		}
	}
	return result, nil
}

// OwnerAdmission evaluates routing ownership; it does not consult process identity.
func OwnerAdmission(spec TaskSpec, operators []string, currentOperator string) (bool, string) {
	if spec.Scope == "machine" {
		return true, ""
	}
	if spec.Scope != "team" {
		return false, "invalid-scope"
	}
	if !validOperatorSlug(currentOperator) {
		return false, "missing-operator"
	}
	if !validOperatorSlug(spec.Owner) {
		return false, "invalid-owner"
	}
	for _, owner := range operators {
		if owner == spec.Owner {
			if owner != currentOperator {
				return false, "not-owner"
			}
			return true, ""
		}
	}
	return false, "owner-not-listed"
}
