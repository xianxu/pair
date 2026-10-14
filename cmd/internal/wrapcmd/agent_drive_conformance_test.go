package wrapcmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// TestAgentDriveLiveConformance runs the production mailbox, terminal probe,
// translator, lifecycle accounting and single drive writer against real Codex.
// Login is copied read-only; every writable agent path is disposable. No model
// conversation is submitted. Opt in with PAIR_LIVE_AGENT_DRIVE=codex.
func TestAgentDriveLiveConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_AGENT_DRIVE") != "codex" {
		t.Skip("set PAIR_LIVE_AGENT_DRIVE=codex for isolated wrapper drive conformance")
	}
	for _, tc := range []struct {
		name           string
		request        agentdrive.ModelRequest
		outcome        agentdrive.Outcome
		defaultsChange bool
	}{
		{"session", agentdrive.ModelRequest{Model: "gpt-6.1-sol", Effort: "low", Scope: agentdrive.ScopeSession}, agentdrive.OutcomeConfirmed, false},
		{"default", agentdrive.ModelRequest{Model: "gpt-6.1-sol", Effort: "low", Scope: agentdrive.ScopeDefault}, agentdrive.OutcomeConfirmed, true},
		{"unsupported-effort-recovery", agentdrive.ModelRequest{Model: "gpt-6.1-sol", Effort: "not-supported"}, agentdrive.OutcomeFailed, false},
	} {
		t.Run(tc.name, func(t *testing.T) { runIsolatedDriveConformance(t, tc.request, tc.outcome, tc.defaultsChange) })
	}
}

type conformanceDriveWriter struct {
	pty    *os.File
	writes [][]byte
}

func (w *conformanceDriveWriter) Write(b []byte) (int, error) {
	n, e := w.pty.Write(b)
	w.writes = append(w.writes, append([]byte(nil), b[:n]...))
	return n, e
}

func runIsolatedDriveConformance(t *testing.T, request agentdrive.ModelRequest, want agentdrive.Outcome, change bool) {
	t.Helper()
	dir, e := os.MkdirTemp("/tmp", "pair-drive-conformance-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	root, e := filepath.EvalSymlinks(dir)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"home", "codex", "claude", "config", "cache", "data", "state", "repo"} {
		if e := os.MkdirAll(filepath.Join(root, name), 0700); e != nil {
			t.Fatal(e)
		}
	}
	cwd := filepath.Join(root, "repo")
	env := peerLiveEnvironment(os.Environ(), root)
	env = peerLiveLocalAuth(t, "codex", root, cwd, env)
	if e := exec.Command("git", "init", "-q", cwd).Run(); e != nil {
		t.Fatal(e)
	}
	config := filepath.Join(root, "codex", "config.toml")
	// Native configuration is read solely to prove the disposable child did not
	// change it; auth contents never enter test output or retained terminal data.
	nativeHome := os.Getenv("CODEX_HOME")
	if nativeHome == "" {
		home, e := os.UserHomeDir()
		if e != nil {
			t.Fatal(e)
		}
		nativeHome = filepath.Join(home, ".codex")
	}
	nativePath := filepath.Join(nativeHome, "config.toml")
	nativeBefore, nativeErr := os.ReadFile(nativePath)
	executable, e := exec.LookPath("codex")
	if e != nil {
		t.Fatal(e)
	}
	version := exec.Command(executable, "--version")
	version.Env = env
	v, e := version.Output()
	if e != nil {
		t.Fatal(e)
	}
	t.Log(strings.TrimSpace(string(v)))
	cmd := exec.Command(executable, "--no-daemon", "--no-alt-screen", "--sandbox", "read-only", "--ask-for-approval", "never", "-c", "check_for_update_on_startup=false")
	cmd.Env = env
	cmd.Dir = cwd
	master, e := pty.StartWithSize(cmd, &pty.Winsize{Rows: 38, Cols: 120})
	if e != nil {
		t.Fatal(e)
	}
	chunks := make(chan []byte, 16)
	readerCancel := make(chan struct{})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		defer close(chunks)
		b := make([]byte, 32768)
		for {
			n, e := master.Read(b)
			if n > 0 {
				chunk := append([]byte(nil), b[:n]...)
				select {
				case chunks <- chunk:
				case <-readerCancel:
					return
				}
			}
			if e != nil {
				return
			}
		}
	}()
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	classifier := newHarnessTTYLiveClassifier(t, "codex")
	p := classifier.proxy
	p.lifecycleEvents = make(chan TurnObservation, 32)
	var delivery *peerDelivery
	t.Cleanup(func() {
		if delivery != nil {
			delivery.mu.Lock()
			delivery.cancelDriveLocked("test teardown")
			if delivery.settleTimer != nil {
				delivery.settleTimer.Stop()
			}
			delivery.settleProbe = nil
			delivery.mu.Unlock()
		}
		waitBounded := func(d time.Duration) (error, bool) {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case e := <-wait:
				return e, true
			case <-timer.C:
				return nil, false
			}
		}
		join := func(d time.Duration) bool {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-readerDone:
				return true
			case <-timer.C:
				return false
			}
		}
		if e := teardownHarnessTTY(harnessTTYTeardownOps{cancel: func() { close(readerCancel) }, closePTY: master.Close, signal: func() error { return cmd.Process.Signal(os.Interrupt) }, kill: cmd.Process.Kill, waitBounded: waitBounded, joinReader: join, grace: 2 * time.Second}); e != nil {
			t.Error(e)
		}
		if e := classifier.Close(); e != nil {
			t.Error(e)
		}
	})
	writer := &conformanceDriveWriter{pty: master}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(65 * time.Second)
	defer timeout.Stop()
	startupDeadline := time.Now().Add(25 * time.Second)
	var retained, queryTail []byte
	var before []byte
	admitted := false
	var result agentdrive.Result
	for !result.Terminal() {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				t.Fatal("Codex exited before a terminal drive result")
			}
			window := append(queryTail, chunk...)
			for range bytes.Count(window, []byte("\x1b[6n")) {
				if _, e := master.Write([]byte("\x1b[1;1R")); e != nil {
					t.Fatal(e)
				}
			}
			keep := len(window)
			if keep > 3 {
				keep = 3
			}
			queryTail = append(queryTail[:0], window[len(window)-keep:]...)
			if len(retained) < harnessTTYRetentionLimit {
				n := harnessTTYRetentionLimit - len(retained)
				if n > len(chunk) {
					n = len(chunk)
				}
				retained = append(retained, chunk[:n]...)
			}
			state := classifier.Observe(chunk, retained)
			if state == harnessTTYUnauthenticated || state == harnessTTYWorkspaceTrust || state == harnessTTYRecognizerDrift {
				t.Fatalf("isolated startup blocked: %s", state)
			}
		case <-ticker.C:
		case <-timeout.C:
			t.Fatalf("drive did not terminate: %+v; screen=%s", result, strings.ReplaceAll(strings.Join(p.agentDriveScreen().Lines, "\n"), root, "/scratch"))
		}
		for {
			select {
			case observation := <-p.lifecycleEvents:
				if observation.Kind == ObservationUserSubmission || observation.Kind == ObservationBareReturn {
					t.Fatalf("model control opened conversational input: %+v", observation)
				}
				p.processLifecycleObservation(observation)
			default:
				goto lifecycleDrained
			}
		}
	lifecycleDrained:
		if p.turnActive.Load() || p.notificationLifecycle.Generation != 0 {
			t.Fatal("model control opened a conversational turn")
		}
		if delivery == nil {
			screen := p.agentDriveScreen()
			if screen.Idle && screen.Complete && strings.Contains(strings.Join(screen.Lines, "\n"), "GPT-6.1-Sol default") {
				b := couchmessage.Binding{Slot: "isolated:1", Repository: filepath.Join(cwd, ".git"), Scope: "conformance", Tag: "isolated", Session: "isolated-session", Nonce: "isolated-launch", Agent: "codex", Version: strings.TrimSpace(string(v)), PID: cmd.Process.Pid, Start: "isolated-start"}
				delivery = newPeerDelivery(b, time.Now)
				delivery.driveProbe = p.agentDriveScreen
				delivery.settleProbe = p.settledNow
				p.peer = delivery
				delivery.mu.Lock()
				delivery.armSettleLocked()
				delivery.mu.Unlock()
			} else if time.Now().After(startupDeadline) {
				t.Fatal("fresh empty Codex composer not recognized within startup budget")
			}
			continue
		}
		if !admitted {
			o := delivery.agentDriveObservation()
			if !o.Idle {
				continue
			}
			before, e = os.ReadFile(config)
			if e != nil {
				t.Fatal(e)
			}
			result, e = delivery.startAgentDrive("isolated-model", couchmessage.DriveStart{Request: agentdrive.Request{Model: &request}, Instance: o.Instance, Sequence: o.Sequence, Deadline: time.Now().Add(30 * time.Second)})
			if e != nil {
				t.Fatal(e)
			}
			admitted = true
		}
		p.dispatchAgentDrive(writer)
		result, e = delivery.agentDriveResult("isolated-model")
		if e != nil {
			t.Fatal(e)
		}
	}
	if result.Outcome != want {
		t.Fatalf("wrapper outcome=%+v; screen=%s", result, strings.ReplaceAll(strings.Join(result.Tail, "\n"), root, "/scratch"))
	}
	if want == agentdrive.OutcomeConfirmed && (result.Model != request.Model || result.Effort != request.Effort || result.Scope != request.EffectiveScope()) {
		t.Fatalf("wrong confirmed intent %+v", result)
	}
	if want == agentdrive.OutcomeFailed && !strings.Contains(result.Detail, "back at empty prompt") {
		t.Fatalf("recovery unconfirmed %+v", result)
	}
	if !p.agentDriveScreen().Idle {
		t.Fatal("final real composer is not empty")
	}
	after, e := os.ReadFile(config)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Equal(before, after) == change {
		t.Fatalf("disposable defaults changed=%v, want %v", !bytes.Equal(before, after), change)
	}
	if nativeErr == nil {
		nativeAfter, e := os.ReadFile(nativePath)
		if e != nil || !bytes.Equal(nativeBefore, nativeAfter) {
			t.Fatal("native Codex defaults changed")
		}
	}
	if len(writer.writes) < 3 {
		t.Fatal("production writer did not execute the model command")
	}
	var all bytes.Buffer
	for _, b := range writer.writes {
		all.Write(b)
	}
	if !bytes.Contains(all.Bytes(), []byte("/model")) || bytes.Contains(all.Bytes(), []byte("Couch peer")) {
		t.Fatal("driver command missing or peer message chrome inserted")
	}
	t.Logf("outcome=%s scope=%s writes=%d confirmed_steps=%d defaults_changed=%v", result.Outcome, result.Scope, len(writer.writes), result.Step, !bytes.Equal(before, after))
}
