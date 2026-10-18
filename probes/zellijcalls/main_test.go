package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The shim must act ONLY when invoked as `zellij` (#215 BR-14).
//
// This is not a stylistic guard. `make test-smoke` runs `go run` over every
// directory under probes/ holding a package main, and a bare `zellij` with no
// arguments STARTS A SESSION -- so without this the smoke suite would leak a
// zellij server on every run, which is the exact debris #215 spent a section
// diagnosing. The guard was added with the Go rewrite and nothing covered it.
//
// End-to-end rather than a predicate check: what must be true is that the
// PROCESS does not reach zellij, and a unit test of the name comparison would
// pass just as happily with the guard wired after the exec.
func TestTheShimOnlyActsWhenInvokedAsZellij(t *testing.T) {
	dir := t.TempDir()

	// A fake zellij that records having been run. It must be found via PATH the
	// same way the real lookup works.
	fakeDir := filepath.Join(dir, "path")
	if err := os.MkdirAll(fakeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "zellij-was-run")
	fake := "#!/bin/sh\necho ran >> " + marker + "\n"
	if err := os.WriteFile(filepath.Join(fakeDir, "zellij"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}

	alias := func(name string) string {
		return shimExecutable(t, filepath.Join(dir, name+"-bin", name))
	}

	runIt := func(bin string) {
		t.Helper()
		cmd := exec.Command(bin)
		cmd.Env = append(os.Environ(), "PATH="+fakeDir, "PAIR_ZELLIJ_TRACE=")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s exited non-zero: %v\n%s", filepath.Base(bin), err, out)
		}
	}

	// Built under its own name -- exactly what `go run ./probes/zellijcalls` does.
	runIt(alias("zellijcalls"))
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the shim ran zellij when built under its own name; `make test-smoke` " +
			"would start a session it never tears down, once per run")
	}

	// POSITIVE CONTROL: the same binary named `zellij` must forward. Without
	// this, a shim that never ran anything would pass the assertion above.
	runIt(alias("zellij"))
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("the shim did NOT reach zellij when invoked as zellij, so the check "+
			"above proves nothing: %v", err)
	}
}

// Internal server reexec must retain the PID Zellij spawned. A persistent shim
// parent has the same --server argv as its child and becomes a second owner.
func TestServerReexecReplacesShim(t *testing.T) {
	dir := t.TempDir()
	shim := shimExecutable(t, filepath.Join(dir, "shim", "zellij"))
	fakeDir := filepath.Join(dir, "real")
	if err := os.Mkdir(fakeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Block until the test releases stdin so the reported PID is still live.
	fake := "#!/bin/sh\nprintf '%s\\n' \"$$\" \"$PROBE_SENTINEL\" \"$@\"\nIFS= read -r input\nprintf '%s\\n' \"$input\" >&2\nexit 17\n"
	if err := os.WriteFile(filepath.Join(fakeDir, "zellij"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		args   []string
		server bool
	}{
		{"server", []string{"--server", "/tmp/socket with spaces"}, true},
		{"client", []string{"action", "write-chars", "--server"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			trace := filepath.Join(dir, tc.name+".tsv")
			cmd := exec.CommandContext(ctx, shim, tc.args...)
			cmd.Env = append(os.Environ(), "PATH="+fakeDir, "PAIR_ZELLIJ_TRACE="+trace, "PROBE_SENTINEL=inherited")
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			// Always release/reap the process, including assertion failures.
			defer func() { _ = stdin.Close(); _ = cmd.Wait() }()
			scanner := bufio.NewScanner(stdout)
			if !scanner.Scan() {
				t.Fatalf("missing native PID: %v", scanner.Err())
			}
			pid := scanner.Text()
			if tc.server && pid != fmt.Sprint(cmd.Process.Pid) {
				t.Errorf("native PID %s differs from spawned shim PID %d: wrapper remains a second server owner", pid, cmd.Process.Pid)
			}
			for _, want := range append([]string{"inherited"}, tc.args...) {
				if !scanner.Scan() || scanner.Text() != want {
					t.Errorf("forwarded env/argv: got %q, want %q", scanner.Text(), want)
				}
			}
			if _, err := fmt.Fprintln(stdin, "stdin forwarded"); err != nil {
				t.Fatal(err)
			}
			_ = stdin.Close()
			err = cmd.Wait()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 17 {
				t.Errorf("exit status: got %v, want 17", err)
			}
			if got := stderr.String(); got != "stdin forwarded\n" {
				t.Errorf("stderr/stdin forwarding: got %q", got)
			}
			rows, err := os.ReadFile(trace)
			if tc.server {
				if !os.IsNotExist(err) {
					t.Errorf("server lifetime must not produce a timing row: %q, %v", rows, err)
				}
			} else if err != nil || !strings.Contains(string(rows), "\t17\t\"action\" \"write-chars\" \"--server\"\n") {
				t.Errorf("client timing row missing: %q, %v", rows, err)
			}
		})
	}
}
