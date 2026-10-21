package repolint

import (
	"strings"
	"testing"
)

func TestZellijReadAttemptBypassRule(t *testing.T) {
	for _, tc := range []struct {
		name, pkg, source string
		want              int
	}{
		{"bounded entry", "launcher", "func (s ZellijSource) runContext() { s.runQueryAttempt(ctx, deadline, args...) }", 0},
		{"pointer entry", "launcher", "func (s *ZellijSource) runContext() { s.runQueryAttempt(ctx, deadline, args...) }", 0},
		{"other method", "launcher", "func (s ZellijSource) Snapshot() { s.runQueryAttempt(ctx, deadline, args...) }", 1},
		{"other receiver", "launcher", "func (s Other) runContext() { s.runQueryAttempt(ctx, deadline, args...) }", 1},
		{"bare function owner", "launcher", "func runContext() { s.runQueryAttempt(ctx, deadline, args...) }", 1},
		{"bare call", "launcher", "func query() { runQueryAttempt(ctx, deadline, args...) }", 1},
		{"parenthesized call", "launcher", "func query() { (s.runQueryAttempt)(ctx, deadline, args...) }", 1},
		{"method expression", "launcher", "func query() { ZellijSource.runQueryAttempt(s, ctx, deadline, args...) }", 1},
		{"nested bypass", "launcher", "func query() { defer func() { s.runQueryAttempt(ctx, deadline, args...) }() }", 1},
		{"package initializer", "launcher", "var result = s.runQueryAttempt(ctx, deadline, args...)", 1},
		{"method declaration", "launcher", "func (s ZellijSource) runQueryAttempt() {}", 0},
		{"comment and literal", "launcher", "// s.runQueryAttempt(ctx, deadline)\nvar text = \"runQueryAttempt()\"", 0},
		{"ordinary entry call", "launcher", "func query() { s.runContext(ctx, args...) }", 0},
		{"other package", "other", "func query() { s.runQueryAttempt(ctx, deadline, args...) }", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package " + tc.pkg + "\n" + tc.source + "\n")
			repository := newFakeRepository()
			repository.index["cmd/internal/"+tc.pkg+"/query.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("count=%d err=%v findings=%+v want=%d", count, err, findings, tc.want)
			}
			for _, found := range findings {
				if found.Finding.Rule != "zellij-read-attempt-bypass" || !strings.Contains(found.Finding.Message, "pair#748") {
					t.Fatalf("unexpected finding: %+v", found)
				}
			}
		})
	}
}
