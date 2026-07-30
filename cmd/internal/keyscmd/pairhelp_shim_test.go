package keyscmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// bin/pair-help is what Alt+h actually runs, and nothing mechanically checked that
// it invokes `pair keys`. tests/pair-embedded-runtime-test.sh only asserts the file
// is executable, so repointing it back at `pair -h` — the #132 bug — would pass
// every test in the tree.
//
// Same pattern as contextcmd/panejson_kdl_test.go: run the real shell script with
// the external binaries stubbed on PATH, so production flow and test flow share the
// seam (ARCH-MOCK). Here `pair` records its argv and `less` becomes cat.
func TestPairHelpShimInvokesPairKeys(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	shim := filepath.Join(repo, "bin", "pair-help")
	if _, err := os.Stat(shim); err != nil {
		t.Skipf("shim not present: %v", err)
	}

	binDir := t.TempDir()
	argvLog := filepath.Join(t.TempDir(), "argv")

	// `pair` records how it was called and emits a marker as the "help" body.
	pairStub := "#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"" + argvLog + "\"\necho MARKER_BODY\n"
	if err := os.WriteFile(filepath.Join(binDir, "pair"), []byte(pairStub), 0o755); err != nil {
		t.Fatal(err)
	}
	// `less` must not be a pager in a test — pass content through.
	if err := os.WriteFile(filepath.Join(binDir, "less"), []byte("#!/bin/sh\nexec cat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// tput may be absent in a bare environment; the shim already falls back to 80.
	if err := os.WriteFile(filepath.Join(binDir, "tput"), []byte("#!/bin/sh\necho 100\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// NOTE: needs the agent command sandbox disabled, like the termcmd/wrapcmd pty
	// tests. The shim's `mktemp -t` (for its lesskey file) writes to the platform temp
	// dir, and a spawned subprocess does not inherit the harness's write grant there.
	// Redirecting TMPDIR does not help — macOS mktemp resolves its own directory.
	cmd := exec.Command("bash", shim)
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pair-help failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "MARKER_BODY") {
		t.Errorf("shim output did not come from the stubbed pair: %s", out)
	}

	argv, err := os.ReadFile(argvLog)
	if err != nil {
		t.Fatalf("shim never invoked `pair`: %v", err)
	}
	got := strings.Fields(string(argv))
	if len(got) == 0 || got[0] != "keys" {
		t.Fatalf("shim invoked `pair %v`, want `pair keys …` — Alt+h must page the keybindings, not the CLI synopsis (#132)", got)
	}
	// It must also ask for centring, since the shim deleted its own awk math.
	if !strings.Contains(string(argv), "--center") {
		t.Errorf("shim argv %q lacks --center; centring moved into Go", got)
	}
}
