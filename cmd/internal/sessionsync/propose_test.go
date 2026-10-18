package sessionsync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type proposeFixture struct {
	Name, Base, After, Markers, Refuse string
}

// The propose fixtures are shared with nvim/review/resolve_fixtures_test.lua,
// which parses the markers with the pane's grammar and checks that accepting
// or rejecting them gives the fixture's texts.
func TestProposeMarkersFixtures(t *testing.T) {
	paths, _ := filepath.Glob(filepath.Join(editFixturesDir, "propose", "*.json"))
	if len(paths) == 0 {
		t.Fatal("no propose fixtures")
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fx proposeFixture
		if err := json.Unmarshal(raw, &fx); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		t.Run(fx.Name, func(t *testing.T) {
			got, err := ProposeMarkers(fx.Base, fx.After)
			if fx.Refuse != "" {
				var r *Refusal
				if !errors.As(err, &r) || r.Code != fx.Refuse || r.Exit != ExitRefused {
					t.Fatalf("err=%v, want refusal %s", err, fx.Refuse)
				}
				return
			}
			if err != nil || got != fx.Markers {
				t.Fatalf("got %q err=%v, want %q", got, err, fx.Markers)
			}
		})
	}
}

// Every propose refusal names the one fallback, the side-doc proposal.
func TestProposeRefusalsNameTheSideDocRemedy(t *testing.T) {
	for _, c := range [][2]string{{"a\nb\nc\nd\n", "A\nB\nC\nD\n"}, {"x ~y~\n", "x z\n"}} {
		_, err := ProposeMarkers(c[0], c[1])
		var r *Refusal
		if !errors.As(err, &r) || !strings.Contains(r.Reason, "side-doc proposal") {
			t.Fatalf("refusal %v does not name the side-doc remedy", err)
		}
	}
}
