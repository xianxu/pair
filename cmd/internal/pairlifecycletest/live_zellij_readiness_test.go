package pairlifecycletest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestControlledZellijWaitsForScreenRenderBeforeProbing(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1" in
 delete-session) exit 0;;
 list-sessions)
  if [ ! -f "$FIXTURE_READY" ]; then : > "$FIXTURE_EARLY_PROBE"; exit 1; fi
  printf 'owned-test [Created 0s ago]\n'; exit 0;;
 --session)
  printf '\033[?2004h'
  sleep 0.1
  : > "$FIXTURE_READY"
  printf '\033[1;1H\033[m'
  exec sleep 30;;
esac
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FIXTURE_READY", filepath.Join(dir, "ready"))
	early := filepath.Join(dir, "early")
	t.Setenv("FIXTURE_EARLY_PROBE", early)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	fixture, err := StartControlledZellij(ctx, "owned-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.Close() })
	if _, err := os.Stat(early); !os.IsNotExist(err) {
		t.Fatalf("session probed before initial client output: %v", err)
	}
}

func TestControlledZellijLiveRowIsExact(t *testing.T) {
	for _, row := range []string{"owned-test-other [Created 0s ago]", "owned-test [Created 0s ago] (EXITED - attach to resurrect)", "warning owned-test", "owned-test failed to connect"} {
		if hasLiveSessionRow(row, "owned-test") {
			t.Fatalf("accepted %q", row)
		}
	}
	if !hasLiveSessionRow("other [Created 0s ago]\nowned-test [Created 0s ago]", "owned-test") {
		t.Fatal("live row rejected")
	}
}

func TestControlledZellijInitialOutputFailureDoesNotProbe(t *testing.T) {
	for _, mode := range []string{"exit", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			early := filepath.Join(dir, "early")
			script := `#!/bin/sh
case "$1" in
 delete-session) exit 0;;
 list-sessions) : > "$FIXTURE_EARLY_PROBE"; exit 1;;
 --session) if [ "$FIXTURE_MODE" = exit ]; then exit 42; fi; exec sleep 30;;
esac
`
			if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("FIXTURE_EARLY_PROBE", early)
			t.Setenv("FIXTURE_MODE", mode)
			ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
			defer cancel()
			if fixture, err := StartControlledZellij(ctx, "owned-test"); err == nil {
				_ = fixture.Close()
				t.Fatal("missing initial output accepted")
			}
			if _, err := os.Stat(early); !os.IsNotExist(err) {
				t.Fatalf("probed uninitialized session: %v", err)
			}
		})
	}
}
