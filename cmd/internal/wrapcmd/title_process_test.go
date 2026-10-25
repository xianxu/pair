package wrapcmd

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/titlepoller"
)

// Dispatch only this test's self-exec title command. Ordinary test binaries must
// never treat arbitrary argv as a command or use an inherited live owner.
func titleProcessTestCommand() {
	if os.Getenv("PAIR_617_TITLE_HELPER") == "1" && len(os.Args) > 1 && os.Args[1] == "title" {
		os.Exit(titlepoller.RunCLI(os.Args[2:], os.Getenv, os.Stderr))
	}
}

func TestTitleWrapperHelper(t *testing.T) {
	if os.Getenv("PAIR_617_TITLE_HELPER") != "1" {
		return
	}
	fmt.Fprintln(os.Stderr, "title fixture: helper entered")
	go func() {
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		<-timer.C
		if _, err := os.Stat(os.Getenv("PAIR_617_AGENT_PID")); err != nil {
			stack := make([]byte, 512*1024)
			n := runtime.Stack(stack, true)
			_, _ = os.Stderr.Write(stack[:n])
		}
	}()
	os.Exit(Run([]string{"--scrollback-log", os.Getenv("PAIR_617_SCROLLBACK"), os.Getenv("PAIR_617_AGENT")}, os.Stdin, os.Stdout, os.Stderr))
}

type titleWrapperFixture struct {
	cmd                                                    *exec.Cmd
	done                                                   chan error
	input                                                  *os.File
	titlePID                                               string
	data, tag, root                                        string
	observedPID, observedIdentity, agentPID, agentIdentity string
}

func startTitleWrapperFixture(t *testing.T, root, tag string) *titleWrapperFixture {
	t.Helper()
	data := filepath.Join(root, "data")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	paths, err := artifactpath.ResolveScoped(data, tag)
	if err != nil {
		t.Fatal(err)
	}
	pane, err := paths.PaneChecked("codex")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pane, []byte(`{"pane_id":"0"}`), 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	agent := filepath.Join(bin, "codex")
	// Native child writes its PID before blocking; cleanup touches only this
	// disposable child, even when the wrapper is intentionally SIGKILLed.
	script := "#!/bin/sh\ntrap '' HUP\necho $$ > \"$PAIR_617_AGENT_PID\"\nexec sleep 30\n"
	if err := os.WriteFile(agent, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte("#!/bin/sh\nprintf '%s\\n' 'fixture-title-session'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestTitleWrapperHelper$")
	env := testpair.Environment(t, root)
	for k, v := range map[string]string{
		"PAIR_SCROLLBACK_EVENTS_PATH": paths.ScrollbackEvents("codex"), "PAIR_617_SCROLLBACK": paths.ScrollbackRaw("codex"), "PAIR_617_TITLE_HELPER": "1", "PAIR_617_AGENT": agent, "PAIR_617_AGENT_PID": filepath.Join(root, "agent-"+tag),
		"HOME": filepath.Join(root, "home"), "PAIR_DATA_DIR": data, "PAIR_TAG": tag, "PAIR_SCOPE_KEY": "0123456789abcdef",
		"PAIR_AGENT": "codex", "PAIR_SESSION_NAME": "fixture-title-session", "PAIR_LAUNCH_ORDINAL": "1", "PAIR_LAUNCH_NONCE": "fixture-title-" + tag,
		"PAIR_SESSION_ID": "", "PAIR_RETENTION_START_ID": "", "COUCH_TREE": "", "COUCH_THREAD_SCOPE": "", "COUCH_THREAD_TAG": "",
		"PATH": bin + string(os.PathListSeparator) + os.Getenv("PATH"),
	} {
		env = setEnv(env, k, v)
	}
	cmd.Env = env
	cmd.Stdin = reader
	log, err := os.OpenFile(filepath.Join(root, "wrapper-"+tag+".log"), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = reader.Close()
	f := &titleWrapperFixture{cmd: cmd, done: make(chan error, 1), input: writer, titlePID: paths.TitlePID(), data: data, tag: tag, root: root}
	go func() { f.done <- cmd.Wait(); _ = log.Close() }()
	t.Cleanup(func() {
		_ = writer.Close()
		_ = cmd.Process.Kill()
		select {
		case <-f.done:
		case <-time.After(5 * time.Second):
			t.Error("wrapper cleanup did not join")
		}
		for _, owned := range [][2]string{{f.agentPID, f.agentIdentity}, {f.observedPID, f.observedIdentity}} {
			if owned[0] != "" && owned[1] != "" && procutil.Identity(owned[0]) == owned[1] {
				pid, _ := strconv.Atoi(owned[0])
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})
	return f
}

func (f *titleWrapperFixture) title(t *testing.T) (string, string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var titleReadErr error
	var titlePID string
	for time.Now().Before(deadline) {
		f.observeAgentIdentity(procutil.Identity)

		raw, err := os.ReadFile(f.titlePID)
		titleReadErr = err
		if err == nil {
			pid := strings.TrimSpace(string(raw))
			titlePID = pid
			if identity := liveTitleIdentity(t, pid); identity != "" {
				f.observedPID, f.observedIdentity = pid, identity
				if f.agentIdentity != "" {
					return pid, identity
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	log, _ := os.ReadFile(filepath.Join(f.root, "wrapper-"+f.tag+".log"))
	t.Fatalf("leased title readiness incomplete: agent PID=%q identity=%q; title PID=%q identity=%q read=%v; wrapper identity=%q; log: %s", f.agentPID, f.agentIdentity, titlePID, f.observedIdentity, titleReadErr, procutil.Identity(strconv.Itoa(f.cmd.Process.Pid)), log)
	return "", ""
}

// observeAgentIdentity samples only this disposable native child's publication.
func (f *titleWrapperFixture) observeAgentIdentity(identity func(string) string) {
	if f.agentPID == "" {
		if raw, err := os.ReadFile(filepath.Join(f.root, "agent-"+f.tag)); err == nil {
			f.agentPID = strings.TrimSpace(string(raw))
		}
	}
	if f.agentPID != "" && f.agentIdentity == "" {
		f.agentIdentity = identity(f.agentPID)
	}
}

func TestTitleFixtureReobservesUnavailableAgentIdentity(t *testing.T) {
	root := t.TempDir()
	f := &titleWrapperFixture{root: root, tag: "unknown-identity"}
	if err := os.WriteFile(filepath.Join(root, "agent-"+f.tag), []byte("4321\n"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	identity := func(pid string) string {
		if pid != "4321" {
			t.Fatalf("identity PID=%q", pid)
		}
		calls++
		if calls == 1 {
			return ""
		}
		if calls == 2 {
			return "100.123"
		}
		return "different-generation"
	}
	f.observeAgentIdentity(identity)
	if f.agentPID != "4321" || f.agentIdentity != "" {
		t.Fatalf("unknown sample=%+v", f)
	}
	f.observeAgentIdentity(identity)
	if f.agentIdentity != "100.123" {
		t.Fatal("unavailable identity became permanently cached")
	}
	f.observeAgentIdentity(identity)
	if calls != 2 || f.agentIdentity != "100.123" {
		t.Fatal("established generation was replaced")
	}
}

func liveTitleIdentity(t *testing.T, pid string) string {
	t.Helper()
	rows, err := procutil.Table()
	if err != nil {
		t.Fatalf("process table unavailable: %v", err)
	}
	for _, row := range rows {
		if strconv.Itoa(row.PID) == pid && !row.Zombie {
			return row.Identity
		}
	}
	return ""
}

func awaitTitleGone(t *testing.T, pid, identity string) {
	t.Helper()
	start := time.Now()
	for time.Since(start) < 3*time.Second {
		if liveTitleIdentity(t, pid) != identity {
			t.Logf("title owner death -> exit in %v", time.Since(start))
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("title %s generation %s survived owner death", pid, identity)
}

func TestWrapperTitleLeaseSIGKILL(t *testing.T) {
	f := startTitleWrapperFixture(t, t.TempDir(), "title-owner")
	pid, identity := f.title(t)
	if err := f.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	awaitTitleGone(t, pid, identity)
}

func TestWrapperTitleLeaseNotInherited(t *testing.T) {
	f := startTitleWrapperFixture(t, t.TempDir(), "title-inherit")
	pid, identity := f.title(t)
	// The native child was exec'd by the actual wrapper and is still alive.
	raw, err := os.ReadFile(filepath.Join(f.root, "agent-"+f.tag))
	if err != nil {
		t.Fatal(err)
	}
	child := strings.TrimSpace(string(raw))
	if procutil.StrictIdentity(child) == "" {
		t.Fatal("native child control is not live")
	}
	sentinel := exec.Command("sleep", "30")
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sentinel.Process.Kill(); _ = sentinel.Wait() }()
	if err := f.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	awaitTitleGone(t, pid, identity)
	if liveTitleIdentity(t, child) == "" {
		t.Fatal("native child did not survive to prove writer non-inheritance")
	}
	if liveTitleIdentity(t, strconv.Itoa(sentinel.Process.Pid)) == "" {
		t.Fatal("unrelated sentinel died")
	}
}

func TestWrapperTitleLeaseReplacement(t *testing.T) {
	root := t.TempDir()
	first := startTitleWrapperFixture(t, root, "title-replace")
	old, identity := first.title(t)
	if err := first.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	awaitTitleGone(t, old, identity)
	next := startTitleWrapperFixture(t, root, "title-replace")
	pid, newIdentity := next.title(t)
	if pid == old && identity == newIdentity {
		t.Fatal("replacement retained old title generation")
	}
	if err := next.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	awaitTitleGone(t, pid, newIdentity)
}

func TestGenericWrapperDoesNotBorrowInheritedTitle(t *testing.T) {
	t.Setenv("PAIR_617_TITLE_HELPER", "1")
	for key, value := range map[string]string{"PAIR_TAG": "borrowed", "PAIR_SCOPE_KEY": "0123456789abcdef", "PAIR_DATA_DIR": t.TempDir(), "PAIR_SESSION_NAME": "borrowed-session", "PAIR_AGENT": "codex"} {
		t.Setenv(key, value)
	}
	title, err := startWrapperTitleProcess("codex", "", io.Discard)
	if title != nil {
		defer title.Close()
		t.Fatal("generic command borrowed a Pair title owner from inherited environment")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrapperTitleSkipsCouchHosted(t *testing.T) {
	for _, tc := range []struct {
		name, scope, tag string
	}{
		{"scope", "host-scope", ""},
		{"tag", "", "host-tag"},
		{"both", "host-scope", "host-tag"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := t.TempDir()
			paths, err := artifactpath.ResolveScoped(data, "hosted-title")
			if err != nil {
				t.Fatal(err)
			}
			for key, value := range map[string]string{
				"PAIR_617_TITLE_HELPER": "1", "PAIR_TAG": "hosted-title",
				"PAIR_SCOPE_KEY": "0123456789abcdef", "PAIR_DATA_DIR": data,
				"PAIR_SESSION_NAME": "fixture-title-session", "PAIR_AGENT": "codex",
				"COUCH_THREAD_SCOPE": tc.scope, "COUCH_THREAD_TAG": tc.tag,
			} {
				t.Setenv(key, value)
			}
			// All standalone admission fields, including the scoped capture, are
			// valid. Either hosted marker must prevent even starting a title child.
			title, err := startWrapperTitleProcess("codex", paths.ScrollbackRaw("codex"), io.Discard)
			if title != nil {
				t.Cleanup(func() { _ = title.Close() })
				t.Errorf("hosted wrapper started title child PID %d", title.cmd.Process.Pid)
			}
			if err != nil {
				t.Fatalf("hosted wrapper title admission: %v", err)
			}
			if _, err := os.Stat(paths.TitlePID()); !os.IsNotExist(err) {
				t.Errorf("hosted wrapper created title PID metadata: %v", err)
			}
		})
	}
}

func TestWrapperTitleFixtureDoesNotTouchInheritedCapture(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, "foreign-events")
	const sentinel = "capture belonging to another live wrapper\n"
	if err := os.WriteFile(foreign, []byte(sentinel), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_SCROLLBACK_EVENTS_PATH", foreign)
	t.Setenv("PAIR_LAUNCH_NONCE", "foreign-nonce")
	t.Setenv("PAIR_AGENT_READY_PATH", filepath.Join(root, "foreign-ready"))
	f := startTitleWrapperFixture(t, filepath.Join(root, "fixture"), "isolation")
	pid, identity := f.title(t)
	if err := f.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	awaitTitleGone(t, pid, identity)
	got, err := os.ReadFile(foreign)
	if err != nil || string(got) != sentinel {
		t.Fatalf("fixture overwrote inherited capture: %q %v", got, err)
	}
	paths, err := artifactpath.ResolveScoped(f.data, f.tag)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(paths.ScrollbackEvents("codex")); err != nil || info.Size() == 0 {
		t.Fatal("disposable capture write control missing")
	}
}

// A stopped title cannot execute its EOF watcher. The wrapper's bounded
// fallback must therefore kill its owned group, including commands already
// started by that title, while leaving another group alive.
func TestWrapperTitleForcedShutdownKillsOwnedGroup(t *testing.T) {
	root := t.TempDir()
	pidfile := filepath.Join(root, "command-pid")
	cmd := exec.Command("/bin/sh", "-c", `trap '' HUP INT TERM; sleep 30 & echo $! > "$PAIR_TITLE_COMMAND_PID"; kill -STOP $$`)
	cmd.Env = testpair.Environment(t, root, "PAIR_TITLE_COMMAND_PID="+pidfile)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	cmd.ExtraFiles = []*os.File{reader}
	if err := cmd.Start(); err != nil {
		writer.Close()
		t.Fatal(err)
	}
	titlePID := strconv.Itoa(cmd.Process.Pid)
	p := &wrapperTitleProcess{writer: writer, cmd: cmd, identity: procutil.Identity(titlePID), done: make(chan error, 1)}
	waited := make(chan struct{})
	go func() { p.done <- cmd.Wait(); close(waited) }()
	var childPID, childIdentity string
	t.Cleanup(func() {
		_ = writer.Close()
		if p.identity != "" && procutil.Identity(titlePID) == p.identity || childIdentity != "" && procutil.Identity(childPID) == childIdentity {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		select {
		case <-waited:
		case <-time.After(time.Second):
			t.Error("forced-title fixture was not joined")
		}
	})
	for end := time.Now().Add(time.Second); time.Now().Before(end); {
		if raw, e := os.ReadFile(pidfile); e == nil {
			childPID = strings.TrimSpace(string(raw))
			childIdentity = liveTitleIdentity(t, childPID)
			if childIdentity != "" {
				break
			}
		}
		time.Sleep(time.Millisecond)
	}
	if p.identity == "" || childIdentity == "" {
		t.Fatal("title/owned command positive controls unavailable")
	}
	sentinel := exec.Command("sleep", "30")
	sentinel.Env = testpair.Environment(t, root)
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	sentinelPID := strconv.Itoa(sentinel.Process.Pid)
	sentinelIdentity := liveTitleIdentity(t, sentinelPID)
	t.Cleanup(func() { _ = sentinel.Process.Kill(); _ = sentinel.Wait() })
	if sentinelIdentity == "" {
		t.Fatal("unrelated-group sentinel unavailable")
	}
	t.Logf("forced title close: %v", p.Close())
	if liveTitleIdentity(t, sentinelPID) != sentinelIdentity {
		t.Fatal("forced title cleanup killed another process group")
	}
	for end := time.Now().Add(time.Second); time.Now().Before(end); {
		if liveTitleIdentity(t, childPID) != childIdentity {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("wrapper forced fallback left its title's command alive")
}
