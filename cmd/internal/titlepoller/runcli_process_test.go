package titlepoller

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

type heldTitleProof struct{ *fakeRuntime }

func (r heldTitleProof) SessionActivity(string, string) (time.Time, bool) {
	if err := os.WriteFile(os.Getenv("PAIR_617_ENTERED"), []byte("proof entered"), 0600); err != nil {
		panic(err)
	}
	select {}
}

// Dispatch the explicit child role before testing.M.Run: RunCLI owns process
// termination and must never run inside a testing.T callback.
func TestMain(m *testing.M) {
	mode := os.Getenv("PAIR_617_CLI_HELPER")
	if mode == "" {
		os.Exit(m.Run())
	}
	if mode != "proof" && mode != "command" {
		os.Exit(2)
	}
	factory := func() Runtime { return NewOSRuntime() }
	if mode == "proof" {
		factory = func() Runtime {
			rt := newFake()
			rt.pid = strconv.Itoa(os.Getpid())
			rt.sessionAliveDflt = true
			pane, _ := artifactpath.ResolveScoped(os.Getenv("PAIR_DATA_DIR"), "fixture")
			rt.mtimes[pane.Pane("claude")] = rt.now
			return heldTitleProof{rt}
		}
	}
	os.Exit(runCLI([]string{"fixture", "claude", "fixture-session"}, os.Getenv, os.Stderr, factory))
}

type cliTitleFixture struct {
	cmd                *exec.Cmd
	writer             *os.File
	done               chan error
	root               string
	identity           string
	descendantPID      string
	descendantIdentity string
}

func startCLITitle(t *testing.T, mode string) *cliTitleFixture {
	t.Helper()
	root := t.TempDir()
	data := filepath.Join(root, "data")
	bin := filepath.Join(root, "bin")
	for _, path := range []string{data, bin} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := artifactpath.ResolveScoped(data, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Pane("claude"), []byte(`{"pane_id":"7"}`), 0600); err != nil {
		t.Fatal(err)
	}
	// The child ignores cooperative signals and remains observable after its
	// immediate shell is killed. Only whole owned-group teardown satisfies EOF.
	script := "#!/bin/sh\ntrap '' HUP INT TERM\nsleep 30 &\necho $! > \"$PAIR_617_DESCENDANT\"\necho $$ > \"$PAIR_617_ENTERED\"\nwait\n"
	if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	owner, err := ParseTitleOwner(strconv.Itoa(os.Getpid()), procutil.Identity(strconv.Itoa(os.Getpid())))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^$")
	explicit := append(owner.Environ(), "PAIR_617_CLI_HELPER="+mode, "PAIR_617_ENTERED="+filepath.Join(root, "entered"), "PAIR_617_DESCENDANT="+filepath.Join(root, "descendant"), "PAIR_DATA_DIR="+data, "PAIR_TAG=fixture", "PAIR_SCOPE_KEY=0123456789abcdef", "PAIR_LAUNCH_ORDINAL=1", "PATH="+bin+":/usr/bin:/bin")
	cmd.Env = testpair.Environment(t, root, explicit...)
	cmd.ExtraFiles = []*os.File{reader}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	log, err := os.Create(filepath.Join(root, "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	f := &cliTitleFixture{cmd: cmd, writer: writer, done: make(chan error, 1), root: root, identity: procutil.Identity(strconv.Itoa(cmd.Process.Pid))}
	go func() { f.done <- cmd.Wait(); _ = log.Close() }()
	t.Cleanup(func() {
		_ = writer.Close()
		// The group is owned only while the exact title or observed descendant
		// identity still exists; never signal a recycled numeric process group.
		if f.identity != "" && procutil.Identity(strconv.Itoa(cmd.Process.Pid)) == f.identity || f.descendantIdentity != "" && procutil.Identity(f.descendantPID) == f.descendantIdentity {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		select {
		case <-f.done:
		case <-time.After(3 * time.Second):
			t.Error("title fixture not joined")
		}
	})
	return f
}
func awaitCLIFile(t *testing.T, f *cliTitleFixture, name string) string {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		if raw, err := os.ReadFile(filepath.Join(f.root, name)); err == nil && len(raw) > 0 {
			return strings.TrimSpace(string(raw))
		}
		time.Sleep(10 * time.Millisecond)
	}
	log, _ := os.ReadFile(filepath.Join(f.root, "stderr"))
	t.Fatalf("title did not enter %s: %s", name, log)
	return ""
}
func awaitCLIExit(t *testing.T, f *cliTitleFixture) {
	t.Helper()
	_ = f.writer.Close()
	select {
	case err := <-f.done:
		f.done <- err
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("lease EOF waited for blocked title work")
	}
}
func TestTitleCLILeaseEOFExitsDuringProof(t *testing.T) {
	f := startCLITitle(t, "proof")
	awaitCLIFile(t, f, "entered")
	awaitCLIExit(t, f)
}
func TestTitleCLILeaseEOFKillsBlockedCommandDescendant(t *testing.T) {
	f := startCLITitle(t, "command")
	awaitCLIFile(t, f, "entered")
	raw := awaitCLIFile(t, f, "descendant")
	pid, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatal(err)
	}
	identity := procutil.Identity(raw)
	if identity == "" {
		t.Fatal("descendant identity missing")
	}
	f.descendantPID, f.descendantIdentity = raw, identity
	awaitCLIExit(t, f)
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		rows, err := procutil.Table()
		if err != nil {
			t.Fatal(err)
		}
		alive := false
		for _, row := range rows {
			if row.PID == pid && row.Identity == identity && !row.Zombie {
				alive = true
			}
		}
		if !alive {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("owned command descendant survived title EOF")
}
