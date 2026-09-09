package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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

	build := func(name string) string {
		out := filepath.Join(dir, name+"-bin", name)
		cmd := exec.Command("go", "build", "-o", out, ".")
		if combined, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", name, err, combined)
		}
		return out
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
	runIt(build("zellijcalls"))
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the shim ran zellij when built under its own name; `make test-smoke` " +
			"would start a session it never tears down, once per run")
	}

	// POSITIVE CONTROL: the same binary named `zellij` must forward. Without
	// this, a shim that never ran anything would pass the assertion above.
	runIt(build("zellij"))
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("the shim did NOT reach zellij when invoked as zellij, so the check "+
			"above proves nothing: %v", err)
	}
}
