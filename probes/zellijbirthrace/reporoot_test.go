package main

import (
	"path/filepath"
	"testing"
)

func TestPokeRejectsMissingCheckoutBeforeStartingTrials(t *testing.T) {
	t.Setenv("PAIR_PROBE_REPO", filepath.Join(t.TempDir(), "missing"))
	// Zero trials still checks the prerequisite; no session needs starting.
	if got := runPoke([]string{"-n", "0"}); got != 1 {
		t.Fatalf("missing checkout exit=%d, want 1", got)
	}
}
