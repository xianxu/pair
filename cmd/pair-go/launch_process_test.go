package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunLaunchWithFakeZellij(t *testing.T) {
	rt := t.TempDir()
	bin := filepath.Join(rt, "bin")
	data := filepath.Join(rt, "data")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(data, "pair"), 0o755); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(rt, "zellij.log")
	zellij := filepath.Join(bin, "zellij")
	script := `#!/usr/bin/env bash
printf '%s\n' "$*" >> "` + log + `"
case "$*" in
  "list-sessions --short") printf 'pair-live\npair-detached\npair-exited\n' ;;
  "list-sessions --no-formatting") printf 'pair-live [Created]\npair-detached [Created]\npair-exited [Created] (EXITED - attach to resurrect)\n' ;;
  "--session pair-live action list-clients") printf 'CLIENTS\n1\n' ;;
  "--session pair-detached action list-clients") printf 'CLIENTS\n' ;;
  *attach*|*new-session*|*--new-session-with-layout*|*delete-session*) printf 'MUTATING %s\n' "$*" >> "` + log + `"; exit 99 ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	historical := filepath.Join(data, "pair", "draft-pair-old.md")
	if err := os.WriteFile(historical, []byte("draft"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(historical, now, now); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HOME", filepath.Join(rt, "home"))
	t.Setenv("XDG_DATA_HOME", data)

	var stdout, stderr bytes.Buffer
	code := run([]string{"launch", "claude"}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("code = %d, want 3; stderr:\n%s", code, stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{"prototype decision", "action=pick"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
	logBytes, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logBytes), "MUTATING") {
		t.Fatalf("fake zellij recorded mutating invocation:\n%s", string(logBytes))
	}
}
