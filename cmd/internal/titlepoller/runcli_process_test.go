package titlepoller

import (
	"context"
	"errors"
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
	"github.com/xianxu/pair/cmd/internal/storagegc"
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
	if mode != "proof" && mode != "command" && mode != "retention" && mode != "publication-pid" && mode != "publication-workspace" {
		os.Exit(2)
	}
	if mode == "retention" {
		coordinator, err := storagegc.NewCoordinator(os.Getenv("PAIR_DATA_DIR"))
		if err != nil {
			panic(err)
		}
		ready := make(chan struct{})
		go func() {
			if err := coordinator.WithLock(context.Background(), func(*storagegc.Locked) error {
				if err := os.WriteFile(os.Getenv("PAIR_617_ENTERED"), []byte("retention lock held"), 0600); err != nil {
					panic(err)
				}
				close(ready)
				select {}
			}); err != nil {
				panic(err)
			}
		}()
		<-ready
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
	if strings.HasPrefix(mode, "publication-") {
		factory = func() Runtime {
			rt := NewOSRuntime()
			rt.publication.AfterTempWrite = func(path string) error {
				raw, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				wanted := strconv.Itoa(os.Getpid()) + "\n"
				if mode == "publication-workspace" {
					wanted = formatCmuxOwner("fixture", "fixture-session")
				}
				if string(raw) != wanted {
					return nil
				}
				if err := os.WriteFile(os.Getenv("PAIR_617_ENTERED"), []byte(path), 0600); err != nil {
					return err
				}
				select {}
			}
			return publicationTitleRuntime{rt}
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
	return startCLITitleGroup(t, mode, true)
}

func startCLITitleGroup(t *testing.T, mode string, ownGroup bool) *cliTitleFixture {
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
	if strings.HasPrefix(mode, "publication-") {
		for _, name := range []string{"title-pid-fixture", "cmux-owner-fixture"} {
			prior := "prior complete generation\n"
			if name == "cmux-owner-fixture" {
				prior = formatCmuxOwner("fixture", "prior-session")
			}
			if err := os.WriteFile(filepath.Join(data, name), []byte(prior), 0600); err != nil {
				t.Fatal(err)
			}
		}
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
	if strings.HasPrefix(mode, "publication-") {
		explicit = append(explicit, "CMUX_WORKSPACE_ID=fixture")
	}
	cmd.Env = testpair.Environment(t, root, explicit...)
	cmd.ExtraFiles = []*os.File{reader}
	if ownGroup {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
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
			if ownGroup {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			} else {
				_ = cmd.Process.Kill()
			}
		}
		select {
		case <-f.done:
		case <-time.After(3 * time.Second):
			t.Error("title fixture not joined")
		}
	})
	return f
}

func TestTitleCLIRequiresOwnedProcessGroup(t *testing.T) {
	f := startCLITitleGroup(t, "proof", false)
	select {
	case err := <-f.done:
		f.done <- err
		if err == nil {
			t.Fatal("title accepted a borrowed process group")
		}
	case <-time.After(time.Second):
		t.Fatal("title admitted polling without its own process group")
	}
	log, _ := os.ReadFile(filepath.Join(f.root, "stderr"))
	if !strings.Contains(string(log), "process group") {
		t.Fatalf("missing owned-group refusal: %s", log)
	}
	if _, err := os.Stat(filepath.Join(f.root, "entered")); !os.IsNotExist(err) {
		t.Fatal("borrowed-group title reached polling")
	}
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

func TestTitleCLILeaseEOFExitsDuringRetentionAcquisition(t *testing.T) {
	f := startCLITitle(t, "retention")
	awaitCLIFile(t, f, "entered")
	// Keep the coordinator lock held throughout this wait and EOF assertion.
	// The child cannot reach its runtime until registration acquires that lock.
	time.Sleep(50 * time.Millisecond)
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

// The real publication/lease boundaries remain active; terminal and transcript
// facts are controlled so both metadata paths reach the same fault deterministically.
type publicationTitleRuntime struct{ OSRuntime }

func (publicationTitleRuntime) SessionAlive(string) bool { return true }
func (publicationTitleRuntime) SessionActivity(string, string) (time.Time, bool) {
	return time.Now(), true
}
func (publicationTitleRuntime) ContextCount(string, string) string      { return "" }
func (publicationTitleRuntime) RenamePane(string, string, string) error { return nil }
func (publicationTitleRuntime) CmuxAvailable() bool                     { return true }
func (publicationTitleRuntime) CmuxRenameWorkspace(string) error        { return nil }

func TestTitleCLILeaseEOFRecoversInterruptedPublications(t *testing.T) {
	for _, tc := range []struct{ mode, name string }{{"publication-pid", "title-pid-fixture"}, {"publication-workspace", "cmux-owner-fixture"}} {
		t.Run(tc.mode, func(t *testing.T) {
			f := startCLITitle(t, tc.mode)
			stage := awaitCLIFile(t, f, "entered")
			target := filepath.Join(f.root, "data", tc.name)
			prior := "prior complete generation\n"
			if tc.mode == "publication-workspace" {
				prior = formatCmuxOwner("fixture", "prior-session")
			}
			checkOld := func() {
				t.Helper()
				if raw, err := os.ReadFile(target); err != nil || string(raw) != prior {
					t.Fatalf("interrupted publication changed target: %q %v", raw, err)
				}
			}
			checkOld()
			c, err := storagegc.NewCoordinator(filepath.Join(f.root, "data"))
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Dir(stage) != c.PendingMetadataDir() {
				t.Fatalf("unmanaged staging path: %s", stage)
			}
			recoveryRan := false
			err = c.TryWithLock(context.Background(), func(*storagegc.Locked) error { recoveryRan = true; return nil })
			if !errors.Is(err, storagegc.ErrCoordinatorBusy) || recoveryRan {
				t.Fatalf("active publisher unprotected: recovery=%v err=%v", recoveryRan, err)
			}
			if _, err := os.Stat(stage); err != nil {
				t.Fatalf("active staging lost: %v", err)
			}
			awaitCLIExit(t, f)
			checkOld()
			if err := c.WithLock(context.Background(), func(held *storagegc.Locked) error {
				n, err := held.RecoverPendingMetadata(context.Background(), c.PendingMetadataDir(), 10)
				if n != 1 {
					t.Errorf("recovered %d entries, want1", n)
				}
				return err
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(stage); !os.IsNotExist(err) {
				t.Fatalf("dead publisher staging survives: %v", err)
			}
			entries, err := os.ReadDir(filepath.Dir(target))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), tc.name+".") {
					t.Fatalf("unclassified sibling residue: %s", entry.Name())
				}
			}
		})
	}
}
