package couchcmd

import (
	"strings"
	"testing"
)

// The public index is a contiguous Markdown table: prose inside a command cell
// must not split later commands out of the rendered index (#722 BR-7).
func TestSkillCommandIndexIsOneCompleteTable(t *testing.T) {
	const header = "| Need | Command |"
	_, tail, ok := strings.Cut(couchSkill, header)
	if !ok {
		t.Fatal("public command index missing")
	}
	var rows []string
	for _, line := range strings.Split(strings.TrimPrefix(tail, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			break
		}
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			t.Fatalf("command index contains a broken row: %q", line)
		}
		rows = append(rows, line)
	}
	index := strings.Join(rows, "\n")
	for _, command := range []string{"`couch --drain --status`", "`couch --undrain`", "`couch --session-sync read"} {
		if !strings.Contains(index, command) {
			t.Fatalf("command %q escaped the contiguous table", command)
		}
	}
}
