package launcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZellijSourceClassifiesSessions(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "zellij.log")
	zellij := filepath.Join(dir, "zellij")
	script := `#!/usr/bin/env bash
printf '%s\n' "$*" >> "` + log + `"
case "$*" in
  "list-sessions --short") printf 'pair-live\npair-detached\npair-gone\nother\n' ;;
  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-gone [Created] (EXITED - attach to resurrect)\n' ;;
  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ZellijSource{Path: zellij}.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	want := []Session{
		{Name: "pair-detached", State: SessionDetached},
		{Name: "pair-gone", State: SessionExited},
		{Name: "pair-live", State: SessionAttached},
	}
	if len(got) != len(want) {
		t.Fatalf("Snapshot returned %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Snapshot[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}
