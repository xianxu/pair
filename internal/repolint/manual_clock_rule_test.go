package repolint

import (
	"strings"
	"testing"
)

func TestManualClockRendezvousRule(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"buffered timer", "func (c *manualClock) after() { _ = make(chan time.Time, 1) }", 1},
		{"buffered initial timer", "func newManualClock() { _ = make(chan time.Time, 2) }", 1},
		{"rendezvous", "func (c *manualClock) after() { _ = make(chan time.Time) }", 0},
		{"explicit zero", "func (c *manualClock) after() { _ = make(chan time.Time, 0) }", 0},
		{"unrelated clock", "func realTimer() { _ = make(chan time.Time, 1) }", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package example\nimport \"time\"\n" + tc.body)
			repository := newFakeRepository()
			repository.index["cmd/internal/example/clock_test.go"] = source
			findings, _, _, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || len(findings) != tc.want {
				t.Fatalf("findings=%v err=%v want=%d", findings, err, tc.want)
			}
			for _, f := range findings {
				if f.Finding.Rule != "manual-clock-rendezvous" || !strings.Contains(f.Finding.Message, "pair#771") {
					t.Fatalf("unexpected finding %+v", f)
				}
			}
		})
	}
}
