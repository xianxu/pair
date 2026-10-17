package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"image/color"
)

func TestPeerReceiverInventoryHasExplicitPolicy(t *testing.T) {
	for _, agent := range launcher.AgentInventory() {
		if _, ok := peerReceiverAgents[agent]; !ok {
			t.Errorf("inventory agent %q has no explicit receiver policy", agent)
		}
	}
	for agent := range peerReceiverAgents {
		if !launcher.IsSupportedAgent(agent) {
			t.Errorf("receiver policy for unknown agent %q", agent)
		}
	}
	if !peerReceiverAgents["qoder"] {
		t.Error("Qoder receiver must register with Couch")
	}
}

func qoderRecorded(t *testing.T, version, screen string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/peer/qoder/" + version + "/" + screen + ".raw")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestPeerQoderRecordedScreens(t *testing.T) {
	for _, tc := range []struct {
		version, screen string
		want            PeerComposerState
	}{
		{"1.1.71", "idle", PeerComposerEmpty}, {"1.1.66", "trust", PeerComposerUnknown},
	} {
		t.Run(tc.screen, func(t *testing.T) {
			f := newHarnessSessionFake(t, "qoder", true)
			t.Cleanup(f.close)
			f.resize(94, 38)
			f.output(qoderRecorded(t, tc.version, tc.screen))
			s := f.proxy.terminal.Snapshot()
			t.Logf("cursor=%+v visible=%v active=%v picker=%v", s.Cursor, s.CursorVisible, qoderComposerActive(s), f.proxy.pickerActive.Load())
			if got := peerComposerState("qoder", s); got != tc.want {
				t.Errorf("composer=%v want %v", got, tc.want)
			}
			if got := f.proxy.settledNow(); got != (tc.want == PeerComposerEmpty) {
				t.Errorf("settled=%v", got)
			}
			if tc.screen == "trust" && !f.proxy.pickerActive.Load() {
				t.Error("trust dialog must hold peer delivery as a picker")
			}
		})
	}
}

// Native Qoder input starts at column 3; unlike its placeholder, text is
// bright and starts under the painted cursor, including multiline continuations.
func peerQoderPaint(glyph string, lines ...string) string {
	out := "\x1b[2J" + qoderBox(6, glyph, "")
	for i, line := range lines {
		out += fmt.Sprintf("\x1b[%d;1H\x1b[K\x1b[%d;4H\x1b[38;2;238;238;235m%s\x1b[39m", 7+i, 7+i, line)
	}
	out += fmt.Sprintf("\x1b[7;2H%s\x1b[%d;1H\x1b[38;2;137;137;135m────────────────────────────────────────\x1b[39m\x1b[?25l\x1b[7;4H", glyph, 7+len(lines))
	return out
}

func TestPeerQoderConservativeText(t *testing.T) {
	for _, tc := range []struct {
		name, paint, text string
		want              PeerComposerState
	}{
		{"empty", peerQoderPaint("*", ""), "", PeerComposerEmpty},
		{"default glyph", peerQoderPaint(">", ""), "", PeerComposerEmpty},
		{"draft", peerQoderPaint("*", "hello"), "hello", PeerComposerOccupied},
		{"below cursor", peerQoderPaint("*", "", "below"), "\nbelow", PeerComposerOccupied},
		{"blank newline", peerQoderPaint("*", "", "") + "\x1b[8;4H", "\n", PeerComposerOccupied},
		{"placeholder words typed", peerQoderPaint("*", "Type your message or @path/to/file"), "Type your message or @path/to/file", PeerComposerOccupied},
		{"slash", peerQoderPaint("*", "/help"), "", PeerComposerUnknown},
		{"shell", peerQoderPaint("*", "!ls"), "", PeerComposerUnknown},
		{"picker glyph", peerQoderPaint("❯", "1. Allow"), "", PeerComposerUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := peerSnapshot(t, tc.paint)
			if got := peerComposerState("qoder", s); got != tc.want {
				t.Errorf("state %v want %v", got, tc.want)
			}
			if text, ok := peerComposerText("qoder", s); text != tc.text || ok != (tc.want != PeerComposerUnknown) {
				t.Errorf("text=%q known=%v want=%q", text, ok, tc.text)
			}
		})
	}
}

// The production startup gate runs with private Unix sockets and a controlled
// read-only workspace probe. No real Couch or harness process is contacted.
func TestPeerQoderRuntimeRegistersEndpoint(t *testing.T) {
	root := t.TempDir()
	namespace := filepath.Join(root, "messages")
	t.Setenv("COUCH_STORE_DIR", namespace)
	t.Setenv("COUCH_THREAD_SCOPE", "fixture-scope")
	t.Setenv("COUCH_THREAD_TAG", "fixture-qoder")
	t.Setenv("PAIR_SESSION_NAME", "fixture-session")
	t.Setenv("PAIR_LAUNCH_NONCE", "fixture-launch")
	primary := filepath.Join(root, "pair")
	identity := map[string]any{"schema_version": 2, "repo": "pair", "repo_identity": filepath.Join(primary, ".git"), "primary_root": primary, "fleet_root": root, "environment_root": root, "worktree_root": primary, "kind": "primary", "address": "pair:0", "slot": 0, "branch": nil, "head": nil, "resting_branch": "main"}
	raw, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "sdlc"), []byte("#!/bin/sh\ncat <<'IDENTITY'\n"+string(raw)+"\nIDENTITY\n"), 0700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, "qoder")
	if err = os.WriteFile(executable, []byte("#!/bin/sh\nprintf '1.1.71\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	socket, err := couchmessage.SocketPath(namespace, "registry")
	if err != nil {
		t.Fatal(err)
	}
	opened := make(chan couchmessage.SessionHello, 1)
	server, err := couchmessage.StartSessionServer(context.Background(), socket, couchmessage.SessionHandler{
		Open:  func(_ couchmessage.SessionToken, h couchmessage.SessionHello) error { opened <- h; return nil },
		Frame: func(couchmessage.SessionToken, couchmessage.SessionFrame) {}, Closed: func(couchmessage.SessionToken) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	f := newHarnessSessionFake(t, "qoder", true)
	t.Cleanup(f.close)
	f.resize(94, 38)
	f.output(qoderRecorded(t, "1.1.71", "idle"))
	runtime, err := f.proxy.startPeerRuntime(executable)
	if err != nil {
		t.Fatal(err)
	}
	if runtime == nil {
		t.Fatal("Qoder startup skipped its message endpoint")
	}
	t.Cleanup(runtime.Close)
	select {
	case hello := <-opened:
		if hello.Binding.Agent != "qoder" || hello.Binding.Version != "1.1.71" {
			t.Fatalf("hello=%+v", hello)
		}
		endpoint := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: hello.Binding}
		tail, err := endpoint.Tail(context.Background(), 8)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.Join(tail.Lines, "\n"), "Type your message") {
			t.Fatalf("tail=%+v", tail)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Qoder registry never received hello")
	}
}

func TestPeerQoderTrustThenPasteSubmitOnce(t *testing.T) {
	f := newHarnessSessionFake(t, "qoder", true)
	t.Cleanup(f.close)
	f.resize(94, 38)
	d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
	sink := &recordingPeerSink{}
	clock := &fakeTimers{}
	d.session = sink
	d.afterFunc = clock.after
	d.settleProbe = f.proxy.settleState
	f.proxy.peer = d
	now := time.Now()
	d.now = func() time.Time { return now }
	render := func(paint string) {
		// Startup terminal queries belong to the terminal round-trip test, not
		// this screen-only fake; account for output through the same mailbox.
		d.observeOutput(nil)
		f.output(paint)
	}
	render(qoderRecorded(t, "1.1.66", "trust"))
	clock.fireAll()
	peerIntegrationEnqueue(t, d)
	var input bytes.Buffer
	f.proxy.dispatchPeer(&input)
	if input.Len() != 0 {
		t.Fatal("peer pasted into folder trust dialog")
	}
	if f.proxy.settledNow() {
		t.Fatal("trust dialog settled")
	}
	if got := f.enter(); string(got) != "\r" {
		t.Fatalf("trust confirmation=%q", got)
	}
	render("\x1b[2J\x1b[H" + qoderRecorded(t, "1.1.71", "idle"))
	clock.fireAll()
	if !d.settled {
		t.Fatal("idle Qoder failed to publish Settled")
	}
	f.proxy.dispatchPeer(&input)
	paste := input.String()
	if !strings.HasPrefix(paste, "\x1b[200~") || !strings.HasSuffix(paste, "\x1b[201~") {
		t.Fatalf("paste=%q detail=%s", paste, d.receipt().Detail)
	}
	f.proxy.dispatchPeer(&input)
	if input.String() != paste {
		t.Fatal("duplicate or early submit")
	}
	render(peerQoderPaint("*", "[Couch peer from brain:0]", "hello"))
	now = now.Add(PeerSubmitDelay)
	f.proxy.dispatchPeer(&input)
	f.proxy.dispatchPeer(&input)
	if input.String() != paste+"\r" {
		t.Fatalf("submit=%q", input.String())
	}
	render("\x1b[2J\x1b[H" + qoderRecorded(t, "1.1.71", "idle"))
	f.proxy.dispatchPeer(&input)
	if d.receipt().Status != couchmessage.Submitted {
		t.Fatalf("receipt=%+v", d.receipt())
	}
	if input.String() != paste+"\r" {
		t.Fatal("duplicate input after confirmation")
	}
}

func TestPeerQoderPlaceholderRequiresPaintedCursorAndStyle(t *testing.T) {
	for _, change := range []string{"style", "cursor background", "moved cursor", "extra row"} {
		t.Run(change, func(t *testing.T) {
			s := peerSnapshot(t, qoderRecorded(t, "1.1.71", "idle"))
			switch change {
			case "style":
				s.CellAt(5, s.Cursor.Y).Style.Fg = color.RGBA{238, 238, 235, 255}
			case "cursor background":
				s.CellAt(3, s.Cursor.Y).Style.Bg = nil
			case "moved cursor":
				s.Cursor.X++
			case "extra row":
				// Keep a valid multiline composer while adding authored text under the
				// placeholder. It cannot be interpreted as an empty single-line hint.
				s = peerSnapshot(t, qoderBox(6, "*", "Type your message or @path/to/file", "human")+"\x1b[?25l\x1b[7;4H")
			}
			if peerComposerState("qoder", s) != PeerComposerOccupied {
				t.Fatal("placeholder mutation authorized overwriting input")
			}
		})
	}
}

func TestPeerQoderDraftAndPickerInterruptAutomaticPaste(t *testing.T) {
	for _, scenario := range []string{"draft", "unrendered input", "picker"} {
		t.Run(scenario, func(t *testing.T) {
			f := newHarnessSessionFake(t, "qoder", true)
			t.Cleanup(f.close)
			f.output(peerQoderPaint("*", ""))
			d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
			d.session = &recordingPeerSink{}
			f.proxy.peer = d
			peerIntegrationEnqueue(t, d)
			switch scenario {
			case "draft":
				peerRender(f, d, peerQoderPaint("*", "human"))
			case "unrendered input":
				d.admitInput([]byte("human"))
				d.inputForwarded(true, false)
			case "picker":
				f.resize(94, 38)
				peerRender(f, d, qoderRecorded(t, "1.1.66", "trust"))
			}
			var input bytes.Buffer
			f.proxy.dispatchPeer(&input)
			if input.Len() != 0 {
				t.Fatalf("%s allowed automatic paste: %q", scenario, input.String())
			}
		})
	}
}

func TestPeerQoderRecordedPickerScreensDecline(t *testing.T) {
	for _, screen := range []string{"overlay", "selection"} {
		t.Run(screen, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/tty/qoder/1.1.60/" + screen + ".raw")
			if err != nil {
				t.Fatal(err)
			}
			f := newHarnessSessionFake(t, "qoder", true)
			t.Cleanup(f.close)
			f.resize(120, 38)
			f.output(string(raw))
			if peerComposerState("qoder", f.proxy.terminal.Snapshot()) != PeerComposerUnknown {
				t.Fatal("picker read as composer")
			}
			if f.proxy.settledNow() {
				t.Fatal("picker settled")
			}
		})
	}
}

func TestPeerEnabledReceiversHaveCapturedComposer(t *testing.T) {
	captures := map[string]string{
		"claude": "testdata/tty/claude/2.1.237/composer.raw",
		"codex":  "testdata/peer/codex/0.159.2/startup.raw",
		"qoder":  "testdata/peer/qoder/1.1.71/idle.raw",
	}
	for agent, enabled := range peerReceiverAgents {
		if !enabled {
			continue
		}
		path, ok := captures[agent]
		if !ok {
			t.Errorf("enabled receiver %s has no captured composer case", agent)
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if peerComposerState(agent, peerSnapshot(t, string(raw))) != PeerComposerEmpty {
			t.Errorf("enabled receiver %s lacks an empty composer parser", agent)
		}
	}
}
