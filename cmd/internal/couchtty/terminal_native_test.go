package couchtty

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// This is a real transport join: Console -> PtyRunner -> Zellij -> pair wrap
// -> a raw, agent-named process. It uses no installed Pair session or data.
// Run with PAIR_LIVE_COUCH_NATIVE=1 PAIR_NATIVE_BINARY=/absolute/candidate.
func TestNativeConsoleWrapperZellij(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH_NATIVE") != "1" {
		t.Skip("set PAIR_LIVE_COUCH_NATIVE=1 and PAIR_NATIVE_BINARY to a freshly built pair")
	}
	for _, wrapped := range []bool{false, true} {
		name := "direct-zellij-baseline"
		if wrapped {
			name = "wrapped-zellij"
		}
		t.Run(name, func(t *testing.T) { runNativeConsoleJoin(t, wrapped) })
	}
}

func runNativeConsoleJoin(t *testing.T, wrapped bool) {
	binary := os.Getenv("PAIR_NATIVE_BINARY")
	if !filepath.IsAbs(binary) {
		t.Fatal("PAIR_NATIVE_BINARY must name an absolute, freshly built candidate")
	}
	zellij, err := exec.LookPath("zellij")
	if err != nil {
		t.Fatal(err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	// Keep socket paths below the Unix-domain limit on macOS.
	dir, err := os.MkdirTemp("/tmp", "pcn-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	write := func(name, body string, mode os.FileMode) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if err := os.Mkdir(filepath.Join(dir, "pair-data"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	// Slug generation is outside terminal transport; keep this unrelated helper local.
	write("bin/pair", "#!/bin/sh\nexit 0\n", 0700)
	paths, err := artifactpath.ResolveScoped(filepath.Join(dir, "pair-data"), "native-fixture")
	if err != nil {
		t.Fatal(err)
	}
	receipts := filepath.Join(dir, "receipts")
	fixture := write("codex", "#!"+python+"\n"+nativeAgentFixture, 0700)
	config := write("config.kdl", "host_notification_protocol \"osc9\"\npane_frames false\nshow_startup_tips false\nshow_release_notes false\non_force_close \"quit\"\n", 0600)
	layoutBody := fmt.Sprintf("layout { pane borderless=true command=%q { args \"wrap\" \"--scrollback-log\" %q %q %q; }; }\n", binary, filepath.Join(dir, "scrollback"), fixture, receipts)
	if !wrapped {
		layoutBody = fmt.Sprintf("layout { pane borderless=true command=%q { args %q; }; }\n", fixture, receipts)
	}
	layout := write("layout.kdl", layoutBody, 0600)
	profileEnv, err := runtimebundle.TerminalEnvironment(dir)
	if err != nil {
		t.Fatal(err)
	}
	// ptychild overlays rather than replaces the parent environment. Blank all
	// inherited Pair session metadata before supplying this private fixture's
	// paths, so wrapper event logs and lifecycle watchers cannot touch a host
	// session merely because its environment launched this test process.
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "PAIR_") {
			env = append(env, key+"=")
		}
	}
	env = append(env, "PAIR_PAIR_WRAP_PID_PATH="+paths.PairWrapPID(), "PAIR_SCROLLBACK_EVENTS_PATH="+filepath.Join(dir, "scrollback.events.jsonl"), "PAIR_WRAP_LOG="+filepath.Join(dir, "wrapper.log"), "ZELLIJ=", "ZELLIJ_SESSION_NAME=", "ZELLIJ_PANE_ID=", "XDG_CACHE_HOME="+filepath.Join(dir, "cache"), "XDG_CONFIG_HOME="+filepath.Join(dir, "config"), "ZELLIJ_SOCKET_DIR="+filepath.Join(dir, "socket"), "PAIR_TAG=native-fixture", "PAIR_SCOPE_KEY=", "PAIR_HOME="+dir, "PAIR_DATA_DIR="+filepath.Join(dir, "pair-data"))
	session := filepath.Base(dir)
	host := newVTHost(24, 100)
	inputR, inputW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	con := New(host, inputR)
	var diagnostics bytes.Buffer
	con.SetErrorWriter(&diagnostics)
	runner := &couchcore.PtyRunner{Size: con.ChildSize, Sink: con.Deliver, Environment: func() ([]string, error) {
		return profileEnv, nil
	}}
	handle, err := runner.Start(dir, []string{zellij, "--session", session, "--config", config, "--new-session-with-layout", layout, "--data-dir", filepath.Join(dir, "data")}, env)
	if err != nil {
		t.Fatal(err)
	}
	child := handle.(couchcore.TerminalHandle).Terminal()
	con.Attach(handle.ID(), "native-agent", child)
	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	t.Cleanup(func() {
		if t.Failed() {
			evidence, _ := os.MkdirTemp("/tmp", "pair255-native-failure-")
			rawReceipts, _ := os.ReadFile(receipts)
			wrapLog, _ := os.ReadFile(filepath.Join(dir, "wrapper.log"))
			_ = os.WriteFile(filepath.Join(evidence, "wrapper.log"), wrapLog, 0600)
			_ = os.WriteFile(filepath.Join(evidence, "receipts.txt"), rawReceipts, 0600)
			_ = os.WriteFile(filepath.Join(evidence, "child.raw"), child.Snapshot(), 0600)
			_ = os.WriteFile(filepath.Join(evidence, "parent.raw"), []byte(host.Written()), 0600)
			t.Logf("native failure evidence: %s; receipts=%q", evidence, rawReceipts)
		}
		// Console owns accepted children until presentation release. Join it
		// before killing the private server or disposing any terminal handle.
		con.Stop()
		_ = inputW.Close()
		select {
		case code := <-done:
			if code != 0 {
				t.Errorf("native Console exit%d: %s", code, diagnostics.String())
			}
		case <-time.After(5 * time.Second):
			t.Error("native Console did not join")
		}
		killCtx, cancelKill := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelKill()
		kill := exec.CommandContext(killCtx, zellij, "kill-session", session)
		kill.Env = append(os.Environ(), env...)
		_ = kill.Run()
	})
	readReceipts := func() string { b, _ := os.ReadFile(receipts); return string(b) }
	waitUpTo(t, 15*time.Second, "wrapped native process and CPR", func() bool {
		return strings.Contains(readReceipts(), "CPR=") && strings.Contains(host.childArea(), "native 界 e ready")
	})
	if !strings.Contains(readReceipts(), "CPR=1b5b") || !strings.Contains(readReceipts(), "52\n") {
		t.Fatalf("fixture did not receive a cursor response: %s", readReceipts())
	}
	if !strings.Contains(host.row(24), "native-agent") {
		t.Fatalf("lost chrome: %q", host.row(24))
	}
	// Policy consumes the shortcut while the actual enhanced event is encoded
	// separately by each terminal boundary. Paste carries embedded ESC intact.
	_, _ = inputW.Write([]byte("\x1b[97;1u"))
	waitUpTo(t, 3*time.Second, "enhanced key reaches wrapped agent", func() bool { return strings.Contains(readReceipts(), "61") })
	_, _ = inputW.Write([]byte("\x1b[200~paste-界\x1b[31m\x1b[201~"))
	wantPaste := hex.EncodeToString([]byte("\x1b[200~paste-界\x1b[31m\x1b[201~"))
	waitUpTo(t, 3*time.Second, "paste reaches wrapped agent", func() bool { return strings.Contains(strings.ReplaceAll(readReceipts(), "\n", ""), wantPaste) })
	// Focus and a complete drag cross the native transport, not a fake decoder.
	_, _ = inputW.Write([]byte("\x1b[O\x1b[I\x1b[<0;4;4M\x1b[<32;6;4M\x1b[<0;6;4m"))
	waitUpTo(t, 3*time.Second, "focus and drag receipts", func() bool {
		var raw []byte
		for _, line := range strings.Split(readReceipts(), "\n") {
			b, err := hex.DecodeString(line)
			if err == nil {
				raw = append(raw, b...)
			}
		}
		return bytes.Contains(raw, []byte("\x1b[O")) && bytes.Contains(raw, []byte("\x1b[I")) && bytes.Contains(raw, []byte("\x1b[<0;4;4M")) && bytes.Contains(raw, []byte("\x1b[<32;6;4M")) && bytes.Contains(raw, []byte("\x1b[<0;6;4m"))
	})
	// The fixture emits a split combining mark too. Native Zellij 0.45.1 drops
	// it before our endpoint; the direct subtest establishes that boundary.
	if bytes.Contains(child.Snapshot(), []byte("e\xcc\x81")) {
		t.Log("native Zellij retained combining mark")
	} else {
		t.Log("native Zellij omitted combining mark before Endpoint (compare direct baseline); CJK asserted independently")
	}
	var nextNotification time.Time
	if wrapped {
		_, _ = inputW.Write([]byte("\r"))
		waitUpTo(t, 3*time.Second, "one native notification through Zellij output", func() bool {
			return strings.Count(host.Written(), "\x1b]777;notify;pair;native-fixture-complete\x1b\\") == 1
		})
		nextNotification = time.Now().Add(600 * time.Millisecond)
	}
	// A real second PTY exercises actor switching, hidden output and panel entry.
	second, err := runner.Start(dir, []string{"/bin/sh", "-c", "printf 'shell-counterpart-ready\\r\\n'; cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	secondChild := second.(couchcore.TerminalHandle).Terminal()
	con.Attach(second.ID(), "native-shell", secondChild)
	con.Switch(second.ID())
	waitUpTo(t, 3*time.Second, "real shell actor", func() bool {
		return con.presenter.View().Admitted == secondChild.Endpoint().ID() && strings.Contains(host.childArea(), "shell-counterpart-ready")
	})
	_, _ = inputW.Write([]byte{0})
	waitUpTo(t, 3*time.Second, "panel", func() bool { con.mu.Lock(); defer con.mu.Unlock(); return con.focus.IsPanel() })
	host.SetSize(ptychild.Size{Rows: 28, Cols: 110})
	waitUpTo(t, 3*time.Second, "both real endpoints resized", func() bool {
		a, ea := child.Endpoint().Snapshot(time.Now())
		b, eb := secondChild.Endpoint().Snapshot(time.Now())
		return ea == nil && eb == nil && a.Geometry == (terminal.Geometry{Rows: 27, Cols: 110}) && b.Geometry == (terminal.Geometry{Rows: 27, Cols: 110})
	})
	con.Switch(handle.ID())
	waitUpTo(t, 3*time.Second, "native actor restored", func() bool {
		return con.presenter.View().Admitted == child.Endpoint().ID() && strings.Contains(host.childArea(), "native 界 e ready")
	})
	if err := con.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if wrapped && strings.Count(host.Written(), "native-fixture-complete") != 1 {
		t.Fatalf("notification duplicated on switch: %q", host.Written())
	}
	assertNativeOracle(t, host.Written(), 110, 28, "native 界 e ready", "native-agent")
	// Record a value held only in the agent process. Reattachment must preserve
	// this PID+boot nonce and advance the same counter, not start a replacement.
	var boot string
	for _, line := range strings.Split(readReceipts(), "\n") {
		if strings.HasPrefix(line, "BOOT=") {
			boot = strings.TrimPrefix(line, "BOOT=")
		}
	}
	if boot == "" {
		t.Fatal("fixture did not report process identity")
	}
	_, _ = inputW.Write([]byte("Q"))
	waitUpTo(t, 3*time.Second, "original in-memory state", func() bool {
		return strings.Contains(readReceipts(), "STATE="+boot+":1\n") && strings.Contains(host.childArea(), "persistent-counter=1")
	})
	evidenceDir, err := os.MkdirTemp("/tmp", "pair255-native-reattach-")
	if err != nil {
		t.Fatal(err)
	}
	captureNativeReattachEvidence(t, evidenceDir, "before-detach", child)
	t.Logf("persistent reattach exact child wire/frame evidence (wrapped=%t): %s", wrapped, evidenceDir)
	var brokerBinding []byte
	if wrapped {
		brokerBinding, err = os.ReadFile(paths.PairWrapPID())
		if err != nil || len(bytes.TrimSpace(brokerBinding)) == 0 {
			t.Fatalf("missing live wrapper broker binding: %q %v", brokerBinding, err)
		}
	}
	oldChild, oldID := child, handle.ID()
	con.Switch(second.ID())
	waitUpTo(t, 3*time.Second, "shell holds Console during detach", func() bool { return con.presenter.View().Admitted == secondChild.Endpoint().ID() })
	detachCtx, cancelDetach := context.WithTimeout(context.Background(), 5*time.Second)
	detach := exec.CommandContext(detachCtx, zellij, "--session", session, "action", "detach")
	detach.Env = append(os.Environ(), env...)
	detachedOutput, detachErr := detach.CombinedOutput()
	cancelDetach()
	if detachErr != nil {
		t.Fatalf("private Zellij detach: %v %s", detachErr, detachedOutput)
	}
	select {
	case <-oldChild.Exited():
	case <-time.After(5 * time.Second):
		t.Fatal("detached client did not drain and exit")
	}
	waitUpTo(t, 3*time.Second, "detached client retired and disposed", func() bool { _, err := oldChild.Endpoint().Snapshot(time.Now()); return errors.Is(err, os.ErrClosed) })
	// No --create: the original server and agent must still exist. Notification
	// output follows this fresh client naturally, without an outer-TTY sidecar.
	handle, err = runner.Start(dir, []string{zellij, "--config", config, "--data-dir", filepath.Join(dir, "data"), "attach", session}, env)
	if err != nil {
		t.Fatal(err)
	}
	child = handle.(couchcore.TerminalHandle).Terminal()
	if child.Endpoint().ID() == oldChild.Endpoint().ID() || handle.ID() == oldID {
		t.Fatal("reattachment reused disposed endpoint identity")
	}
	con.Attach(handle.ID(), "native-agent", child)
	con.Switch(handle.ID())
	waitUpTo(t, 5*time.Second, "persistent native frame after reattach", func() bool {
		return con.presenter.View().Admitted == child.Endpoint().ID() && strings.Contains(host.childArea(), "persistent-counter=1") && (!wrapped || strings.Contains(host.childArea(), "native 界 e ready"))
	})
	captureNativeReattachEvidence(t, evidenceDir, "after-reattach", child)
	assertNativeReattachFrames(t, evidenceDir, 110, 27)
	if !wrapped && !bytes.Contains(child.Snapshot(), []byte("native 界")) {
		t.Log("direct Zellij had already omitted earlier CJK row before detach after counter paint; exact pre/post traces preserved, not a detach-induced loss")
	}
	_, _ = inputW.Write([]byte("Q"))
	waitUpTo(t, 3*time.Second, "same agent resumes interaction", func() bool {
		return strings.Contains(readReceipts(), "STATE="+boot+":2\n") && strings.Contains(host.childArea(), "persistent-counter=2")
	})
	if wrapped {
		reattachedBinding, err := os.ReadFile(paths.PairWrapPID())
		if err != nil || !bytes.Equal(brokerBinding, reattachedBinding) {
			t.Fatalf("reattach replaced wrapper broker: before%q after%q err%v", brokerBinding, reattachedBinding, err)
		}
		// Respect wrapper's 500ms native-notification limiter between real turns.
		if remaining := time.Until(nextNotification); remaining > 0 {
			time.Sleep(remaining)
		}
		_, _ = inputW.Write([]byte("\r"))
		waitUpTo(t, 3*time.Second, "surviving wrapper notification follows reattached client", func() bool {
			return strings.Count(host.Written(), "\x1b]777;notify;pair;native-fixture-complete\x1b\\") == 2
		})
		nextNotification = time.Now().Add(600 * time.Millisecond)
	}
	if strings.Count(readReceipts(), "BOOT=") != 1 {
		t.Fatalf("reattachment restarted agent: %s", readReceipts())
	}
	if err := con.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertNativeOracle(t, host.Written(), 110, 28, "persistent-counter=2", "native-agent")
	t.Logf("persistent native reattach retained pid:nonce=%s counter=1→2; endpoints=%s→%s", boot, oldChild.Endpoint().ID(), child.Endpoint().ID())
	// A no-mouse child makes Zellij own native text selection. Inspect its
	// published highlight while the button is still held, then inspect OSC52.
	if err := os.WriteFile(receipts+".select", []byte("select"), 0600); err != nil {
		t.Fatal(err)
	}
	waitUpTo(t, 3*time.Second, "native no-mouse selection surface", func() bool { return strings.Contains(host.childArea(), "selectable-native-text") })
	before, err := child.Endpoint().Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	host.mu.Lock()
	beforeParent := make([]terminal.Cell, 9)
	for i := range beforeParent {
		beforeParent[i] = *host.em.CellAt(i, 0)
	}
	host.mu.Unlock()
	beforeWire := host.Written()
	_, _ = inputW.Write([]byte("\x1b[<0;1;1M\x1b[<32;10;1M"))
	waitUpTo(t, 3*time.Second, "native held-drag highlight before release", func() bool {
		now, err := child.Endpoint().Snapshot(time.Now())
		if err != nil {
			return false
		}
		for i := 0; i < 9; i++ {
			if !reflect.DeepEqual(before.Cells[i].Style, now.Cells[i].Style) {
				return true
			}
		}
		return false
	})
	// Endpoint ingestion precedes publication enqueue. Flush alone cannot
	// acknowledge that generation: first observe highlight on the parent.
	waitUpTo(t, 3*time.Second, "physical held-drag highlight before release", func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		for i := range beforeParent {
			if cell := host.em.CellAt(i, 0); cell != nil && !reflect.DeepEqual(beforeParent[i].Style, cell.Style) {
				return true
			}
		}
		return false
	})
	if err := con.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertNativeHighlight(t, beforeWire, host.Written(), 110, 28)
	_, _ = inputW.Write([]byte("\x1b[<0;10;1m"))
	waitUpTo(t, 3*time.Second, "native selection clipboard", func() bool { return strings.Contains(host.Written(), "\x1b]52;") })
	wire := host.Written()
	start := strings.LastIndex(wire, "\x1b]52;")
	payload := strings.SplitN(wire[start+5:], ";", 2)
	if len(payload) != 2 {
		t.Fatalf("bad clipboard envelope %q", wire[start:])
	}
	encoded := strings.SplitN(payload[1], "\x1b\\", 2)[0]
	copied, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || string(copied) != "selectabl" {
		t.Fatalf("native selection copied %q (%v), want selectabl", copied, err)
	}
	t.Log("native selection after persistent reattach highlighted in endpoint and independent xterm before release; clipboard selectabl emitted on release")
	if wrapped {
		// The hook command exercises the real bound broker and wrapper writer.
		// Each unique envelope must cross Zellij exactly once while both actors
		// keep receiving output, focus transitions and geometry changes.
		var hookEnvelopes []string
		for cycle := 0; cycle < 32; cycle++ {
			message := fmt.Sprintf("native-hook-%02d", cycle)
			if cycle == 0 {
				message = strings.Repeat("m", 4093) + "界"
			}
			hidden := cycle%2 == 0
			target := handle.ID()
			if hidden {
				target = second.ID()
			}
			con.Switch(target)
			waitUpTo(t, 3*time.Second, "stress focus", func() bool { con.mu.Lock(); defer con.mu.Unlock(); return con.focus == FocusActor(target) })
			if cycle%8 == 0 {
				rows, cols := uint16(28), uint16(110)
				if cycle%16 == 0 {
					rows, cols = 30, 112
				}
				host.SetSize(ptychild.Size{Rows: rows, Cols: cols})
				waitUpTo(t, 3*time.Second, "stress geometry", func() bool {
					return child.Size() == (ptychild.Size{Rows: rows - 1, Cols: cols}) && secondChild.Size() == (ptychild.Size{Rows: rows - 1, Cols: cols})
				})
			}
			shellMarker := fmt.Sprintf("shell-stress-%02d", cycle)
			if _, err := secondChild.Write([]byte(shellMarker + "\n")); err != nil {
				t.Fatal(err)
			}
			if hidden {
				waitUpTo(t, 3*time.Second, "selected shell stress output", func() bool { return strings.Contains(host.childArea(), shellMarker) })
			}
			marker := fmt.Sprintf("stress-output-%02d", cycle)
			if err := os.WriteFile(receipts+".output.tmp", []byte(marker), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(receipts+".output.tmp", receipts+".output"); err != nil {
				t.Fatal(err)
			}
			waitUpTo(t, 3*time.Second, "real native output", func() bool { return strings.Contains(readReceipts(), "OUTPUT="+marker+"\n") })
			// Preserve the existing500ms emission cap across native and hook
			// turns; no retry of a lost notification is allowed by this test.
			if remaining := time.Until(nextNotification); remaining > 0 {
				time.Sleep(remaining)
			}
			hookCtx, cancelHook := context.WithTimeout(context.Background(), 3*time.Second)
			hook := exec.CommandContext(hookCtx, binary, "notify", message)
			hook.Env = append(os.Environ(), env...)
			out, hookErr := hook.CombinedOutput()
			cancelHook()
			if hookErr != nil || len(out) != 0 {
				t.Fatalf("hook cycle%d: %v %s", cycle, hookErr, out)
			}
			envelope := "\x1b]777;notify;pair;" + message + "\x1b\\"
			hookEnvelopes = append(hookEnvelopes, envelope)
			waitUpTo(t, 3*time.Second, "exact in-band hook notification and attention", func() bool {
				if strings.Count(host.Written(), envelope) != 1 {
					return false
				}
				con.mu.Lock()
				defer con.mu.Unlock()
				attention := con.attention.Projection(con.panes[handle.ID()].thread)
				return !hidden && len(attention) == 0 || hidden && len(attention) == 1 && attention[0].Text == message
			})
			nextNotification = time.Now().Add(600 * time.Millisecond)
			con.mu.Lock()
			attention := con.attention.Projection(con.panes[handle.ID()].thread)
			con.mu.Unlock()
			if hidden && (len(attention) != 1 || attention[0].Text != message) {
				t.Fatalf("hidden hook cycle%d lost origin: %+v", cycle, attention)
			}
			if !hidden && len(attention) != 0 {
				t.Fatalf("focused hook cycle%d created attention: %+v", cycle, attention)
			}
		}
		con.Switch(handle.ID())
		waitUpTo(t, 3*time.Second, "final stress actor frame", func() bool { return strings.Contains(host.childArea(), "stress-output-31") })
		if strings.Count(readReceipts(), "BOOT=") != 1 {
			t.Fatal("stress restarted original agent")
		}
		if err := con.presenter.Flush(context.Background()); err != nil {
			t.Fatal(err)
		}
		for _, envelope := range hookEnvelopes {
			if n := strings.Count(host.Written(), envelope); n != 1 {
				t.Fatalf("notification replay after stress:count%d", n)
			}
		}
		assertNativeOracle(t, host.Written(), 110, 28, "stress-output-31", "native-agent")
		t.Log("native notification stress:32 unique hook envelopes,4096-byte UTF8 body,16hidden/16focused,4resizes,original reattached agent")
	}

	t.Logf("native Console join (wrapped=%t): CPR, fragmented CJK, synchronized update, enhanced key, paste, focus, drag, shell/panel switches, resize; receipts=%s", wrapped, readReceipts())
}

func assertNativeOracle(t *testing.T, wire string, cols, rows int, body, chrome string) {
	t.Helper()
	request, _ := json.Marshal(map[string]any{"Cols": cols, "Rows": rows, "Chunks": []string{wire}})
	cmd := exec.Command("node", "../../../tests/terminal-oracle/driver.cjs")
	cmd.Stdin = bytes.NewReader(request)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("independent xterm oracle: %v %s", err, out)
	}
	var frames []struct{ Lines []string }
	if err := json.Unmarshal(out, &frames); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 || len(frames[0].Lines) != rows {
		t.Fatalf("invalid oracle frame: %s", out)
	}
	if !strings.Contains(strings.Join(frames[0].Lines[:rows-1], "\n"), body) || !strings.Contains(frames[0].Lines[rows-1], chrome) {
		t.Fatalf("independent native frame body/chrome mismatch: %q", frames[0].Lines)
	}
}

const nativeAgentFixture = `import os,sys,time,tty,select
fd=os.open('/dev/tty',os.O_RDWR)
tty.setraw(fd)
log=open(sys.argv[1],'a',buffering=1)
identity=str(os.getpid())+':'+os.urandom(12).hex()
counter=0
log.write('BOOT='+identity+'\n')
os.write(fd,b'\x1b[6n')
reply=b''
end=time.monotonic()+5
while time.monotonic()<end and not reply.endswith(b'R'):
 if select.select([fd],[],[],.1)[0]: reply+=os.read(fd,4096)
log.write('CPR='+reply.hex()+'\n')
os.write(fd,b'\x1b[?2004h\x1b[?1004h\x1b[?1002h\x1b[?1006h\x1b[?2026h\x1b[2J\x1b[H')
for part in ['native '.encode(),b'\xe7',b'\x95',b'\x8c',b' e',b'\xcc',b'\x81',b' ready']:
 os.write(fd,part)
 time.sleep(.005)
os.write(fd,b'\x1b[?2026l')
while True:
 if os.path.exists(sys.argv[1]+'.output'):
  marker=open(sys.argv[1]+'.output').read()
  os.unlink(sys.argv[1]+'.output')
  os.write(fd,('\x1b[3;1H\x1b[2K'+marker).encode())
  log.write('OUTPUT='+marker+'\n')
 if os.path.exists(sys.argv[1]+'.select'):
  os.unlink(sys.argv[1]+'.select')
  os.write(fd,b'\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[H\x1b[2Kselectable-native-text')
 if not select.select([fd],[],[],.02)[0]: continue
 data=os.read(fd,65536)
 if not data: break
 log.write(data.hex()+'\n')
 if b'Q' in data:
  counter+=data.count(b'Q')
  log.write('STATE='+identity+':'+str(counter)+'\n')
  os.write(fd,('\x1b[2;1H\x1b[2Kpersistent-counter='+str(counter)).encode())
 if b'\r' in data: os.write(fd,b'\x1b]9;native-fixture-complete\x07\x1b]9;native-fixture-complete\x07')
`

func assertNativeHighlight(t *testing.T, before, after string, cols, rows int) {
	t.Helper()
	if !strings.HasPrefix(after, before) {
		t.Fatal("host wire capture lost prefix")
	}
	request, _ := json.Marshal(map[string]any{"Cols": cols, "Rows": rows, "Chunks": []string{before, after[len(before):]}})
	cmd := exec.Command("node", "../../../tests/terminal-oracle/driver.cjs")
	cmd.Stdin = bytes.NewReader(request)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native highlight oracle: %v %s", err, out)
	}
	var frames []struct {
		Cells [][]struct {
			FG, BG  int
			Inverse bool
		}
	}
	if err := json.Unmarshal(out, &frames); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatal("missing selection oracle frames")
	}
	for i := 0; i < 9; i++ {
		if frames[0].Cells[0][i] != frames[1].Cells[0][i] {
			return
		}
	}
	t.Fatal("native held selection did not reach independent xterm styles")
}

// Keep measured baseline limitations reviewable after disposable session cleanup.
func captureNativeReattachEvidence(t *testing.T, dir, phase string, child *ptychild.Child) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, phase+"-child.raw"), child.Snapshot(), 0600); err != nil {
		t.Fatal(err)
	}
	frame, err := child.Endpoint().Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(frame, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, phase+"-endpoint.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
}

// Compare complete native viewports with independently parsed pre/post wires.
// The agent identity/counter assertion separately proves process continuity.
func assertNativeReattachFrames(t *testing.T, dir string, cols, rows int) {
	t.Helper()
	type observed struct {
		Cells json.RawMessage
		X, Y  int
	}
	var frames []observed
	for _, phase := range []string{"before-detach", "after-reattach"} {
		wire, err := os.ReadFile(filepath.Join(dir, phase+"-child.raw"))
		if err != nil {
			t.Fatal(err)
		}
		request, _ := json.Marshal(map[string]any{"Cols": cols, "Rows": rows, "Chunks": []string{string(wire)}})
		cmd := exec.Command("node", "../../../tests/terminal-oracle/driver.cjs")
		cmd.Stdin = bytes.NewReader(request)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("reattach oracle: %v %s", err, out)
		}
		if err := os.WriteFile(filepath.Join(dir, phase+"-xterm.json"), out, 0600); err != nil {
			t.Fatal(err)
		}
		var result []observed
		if err := json.Unmarshal(out, &result); err != nil || len(result) != 1 {
			t.Fatalf("invalid reattach oracle output: %v", err)
		}
		frames = append(frames, result[0])
	}
	if !bytes.Equal(frames[0].Cells, frames[1].Cells) || frames[0].X != frames[1].X || frames[0].Y != frames[1].Y {
		t.Fatalf("native pre/post viewport cells/styles/cursor differ; exact evidence: %s", dir)
	}
}
