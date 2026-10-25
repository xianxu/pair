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
		{"bounded entry", "zellijquery", "func (s Source) ReadContext() { s.readAttempt(ctx, deadline, args...) }", 0},
		{"pointer entry", "zellijquery", "func (s *Source) ReadContext() { s.readAttempt(ctx, deadline, args...) }", 0},
		{"other method", "zellijquery", "func (s Source) Snapshot() { s.readAttempt(ctx, deadline, args...) }", 1},
		{"other receiver", "zellijquery", "func (s Other) ReadContext() { s.readAttempt(ctx, deadline, args...) }", 1},
		{"bare function owner", "zellijquery", "func ReadContext() { s.readAttempt(ctx, deadline, args...) }", 1},
		{"bare call", "zellijquery", "func query() { readAttempt(ctx, deadline, args...) }", 1},
		{"parenthesized call", "zellijquery", "func query() { (s.readAttempt)(ctx, deadline, args...) }", 1},
		{"method expression", "zellijquery", "func query() { Source.readAttempt(s, ctx, deadline, args...) }", 1},
		{"nested selector bypass", "zellijquery", "var result = decorate(s.readAttempt).Field", 1},
		{"nested bypass", "zellijquery", "func query() { defer func() { s.readAttempt(ctx, deadline, args...) }() }", 1},
		{"package initializer", "zellijquery", "var result = s.readAttempt(ctx, deadline, args...)", 1},
		{"method declaration", "zellijquery", "func (s Source) readAttempt() {}", 0},
		{"comment and literal", "zellijquery", "// s.readAttempt(ctx, deadline)\nvar text = \"readAttempt()\"", 0},
		{"method value", "zellijquery", "var bypass = s.readAttempt", 1},
		{"ordinary entry call", "zellijquery", "func query() { s.ReadContext(ctx, args...) }", 0},
		{"other package", "other", "func query() { s.readAttempt(ctx, deadline, args...) }", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package " + tc.pkg + "\n" + tc.source + "\n")
			repository := newFakeRepository()
			repository.index["cmd/internal/"+tc.pkg+"/query.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			var attempts []locatedFinding
			for _, finding := range findings {
				if finding.Finding.Rule == "zellij-read-attempt-bypass" {
					attempts = append(attempts, finding)
				}
			}
			findings = attempts // isolate this rule from the independent admission contract
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
