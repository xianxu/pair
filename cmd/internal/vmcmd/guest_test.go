package vmcmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGuestInvocationLiteralAndClean(t *testing.T) {
	t.Setenv("SECRET_HOST_VALUE", "must-not-forward")
	argv := []string{"printf", "$(touch injected)", "a b", "'quoted'", ""}
	p := GuestPaths{Input: "/Volumes/input", Results: "/Volumes/results", WorkRoot: "/private/tmp/pair-run"}
	got, err := BuildGuestInvocation(RunRequest{Args: argv}, p)
	if err != nil {
		t.Fatal(err)
	}
	wantTail := append([]string{"/bin/bash", "--noprofile", "--norc", p.Input + "/guest-run.sh", p.Input + "/checkout", p.WorkRoot, p.Results, "--"}, argv...)
	if !reflect.DeepEqual(got.Args[len(got.Args)-len(wantTail):], wantTail) {
		t.Fatalf("argv: %q", got.Args)
	}
	if !reflect.DeepEqual(got.Args[:2], []string{"/usr/bin/env", "-i"}) {
		t.Fatal(got.Args)
	}
	if !reflect.DeepEqual(got.Args[2:2+len(got.Env)], got.Env) {
		t.Fatal("env assignments missing")
	}
	for _, e := range got.Env {
		if strings.Contains(e, "SECRET_HOST_VALUE") {
			t.Fatal(e)
		}
	}
	argv[0] = "mutated"
	if got.Args[len(got.Args)-5] != "printf" {
		t.Fatal("argv aliased")
	}
	for _, p := range []GuestPaths{{Input: "relative", Results: "/r", WorkRoot: "/w"}, {Input: "/i", Results: "/i/r", WorkRoot: "/w"}, {Input: "/i", Results: "/r", WorkRoot: "/"}} {
		if _, err := BuildGuestInvocation(RunRequest{Args: []string{"true"}}, p); err == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
}

func guestFixture(t *testing.T) (string, string, string, string) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "input", "checkout")
	results := filepath.Join(root, "results")
	work := filepath.Join(root, "work")
	for _, d := range []string{filepath.Join(source, ".git"), results} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(source, "tracked"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "guest-run.sh")
	if err := os.WriteFile(script, GuestScript, 0600); err != nil {
		t.Fatal(err)
	}
	return source, work, results, script
}
func executeGuest(source, work, results, script string, args ...string) ([]byte, error) {
	cmd := exec.Command("/usr/bin/env", append([]string{"-i", "PATH=/usr/bin:/bin", "/bin/bash", "--noprofile", "--norc", script, source, work, results, "--"}, args...)...)
	return cmd.CombinedOutput()
}
func TestGuestScriptLiteralExitAndPrivateRoots(t *testing.T) {
	source, work, results, script := guestFixture(t)
	program := `[ "$PWD" = "$1/checkout" ] || exit 91
[ "$HOME" = "$1/home" ] && [ "$TMPDIR" = "$1/tmp" ] || exit 92
[ -d .git ] || exit 93
printf '%s\n' "$2" "$3" > "$4/argv"
printf changed > tracked
printf diagnostic > "$TMPDIR/zellij.log"
exit 37`
	out, err := executeGuest(source, work, results, script, "/bin/bash", "--noprofile", "--norc", "-c", program, "test", work, "$(touch injected)", "a b", results)
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 37 {
		t.Fatalf("exit: %v %s", err, out)
	}
	data, _ := os.ReadFile(filepath.Join(results, "argv"))
	if string(data) != "$(touch injected)\na b\n" {
		t.Fatalf("argv %q", data)
	}
	data, _ = os.ReadFile(filepath.Join(source, "tracked"))
	if string(data) != "original" {
		t.Fatal("source mutated")
	}
	data, _ = os.ReadFile(filepath.Join(results, "status"))
	if string(data) != "exit_code=37\ncollection_error=0\n" {
		t.Fatalf("status %q", data)
	}
	data, _ = os.ReadFile(filepath.Join(results, "diagnostics", "zellij-0.log"))
	if string(data) != "diagnostic" {
		t.Fatalf("logs %q", data)
	}
}
func TestGuestScriptRefusesUnsafeRoots(t *testing.T) {
	for _, kind := range []string{"existing", "overlap", "symlink", "gitfile"} {
		t.Run(kind, func(t *testing.T) {
			source, work, results, script := guestFixture(t)
			switch kind {
			case "existing":
				os.Mkdir(work, 0700)
			case "overlap":
				work = filepath.Join(source, "work")
			case "symlink":
				os.Symlink(results, work)
			case "gitfile":
				os.Remove(filepath.Join(source, ".git"))
				os.WriteFile(filepath.Join(source, ".git"), []byte("gitdir: /host/private"), 0600)
			}
			if out, err := executeGuest(source, work, results, script, "/usr/bin/true"); err == nil {
				t.Fatalf("accepted %s %s", kind, out)
			}
			if _, err := os.Stat(filepath.Join(results, "status")); !os.IsNotExist(err) {
				t.Fatal("mutated results before refusal")
			}
		})
	}
}
func TestGuestScriptCollectionFailureCannotSucceed(t *testing.T) {
	source, work, results, script := guestFixture(t)
	os.Mkdir(filepath.Join(results, "diagnostics"), 0700)
	if out, err := executeGuest(source, work, results, script, "/usr/bin/true"); err == nil {
		t.Fatalf("collection failure hidden: %s", out)
	}
}

func TestGuestScriptDiagnosticBounds(t *testing.T) {
	source, work, results, script := guestFixture(t)
	program := `for n in 0 1 2 3 4 5 6 7 8 9; do /usr/bin/head -c 70000 /dev/zero > "$TMPDIR/zellij-$n.log"; done`
	if out, err := executeGuest(source, work, results, script, "/bin/bash", "-c", program); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	files, err := os.ReadDir(filepath.Join(results, "diagnostics"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 8 {
		t.Fatalf("collected %d logs", len(files))
	}
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > 65536 {
			t.Fatalf("oversized log %d", info.Size())
		}
	}
}
func TestGuestScriptScanFailureCannotSucceed(t *testing.T) {
	source, work, results, script := guestFixture(t)
	program := `/bin/rmdir "$TMPDIR"`
	if out, err := executeGuest(source, work, results, script, "/bin/bash", "-c", program); err == nil {
		t.Fatalf("diagnostic scan failure hidden: %s", out)
	}
}

func TestGuestScriptDoesNotFollowDiagnosticManifestSymlink(t *testing.T) {
	source, work, results, script := guestFixture(t)
	program := `/bin/ln -s "$1/tracked" "$2/diagnostic-files"`
	_, err := executeGuest(source, work, results, script, "/bin/bash", "-c", program, "test", source, work)
	data, _ := os.ReadFile(filepath.Join(source, "tracked"))
	if string(data) != "original" {
		t.Fatalf("collector followed symlink: %q", data)
	}
	if err == nil {
		t.Fatal("manifest collision must fail collection")
	}
}

func TestGuestScriptDiagnosticDiscoveryIsBounded(t *testing.T) {
	source, work, results, script := guestFixture(t)
	program := `for ((n=0; n<1000; n++)); do : > "$TMPDIR/zellij-diagnostic-entry-with-a-long-name-$n.log"; done`
	out, err := executeGuest(source, work, results, script, "/bin/bash", "-c", program)
	if err == nil {
		t.Fatalf("discovery overflow silently succeeded: %s", out)
	}
	info, statErr := os.Stat(filepath.Join(work, "diagnostic-files"))
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Size() > 65537 {
		t.Fatalf("unbounded discovery manifest: %d bytes", info.Size())
	}
	status, readErr := os.ReadFile(filepath.Join(results, "status"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(status) != "exit_code=0\ncollection_error=1\n" {
		t.Fatalf("overflow status %q", status)
	}
}
