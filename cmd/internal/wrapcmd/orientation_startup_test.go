package wrapcmd

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"io"
	"os"
	"testing"
	"time"
)

func TestOrientationLiveStartupFrames(t *testing.T) {
	for _, tc := range []struct {
		agent, path string
		ready       bool
	}{
		{"agy", "testdata/orientation/agy/1.1.25/ready-no-color.raw", true},
		{"codex", "testdata/orientation/codex/0.154.0/loading.raw", false},
		{"codex", "testdata/orientation/codex/0.154.0/trust.raw", false},
		{"codex", "testdata/orientation/codex/0.154.0/ready.raw", true},
		{"codex", "testdata/tty/codex/0.147.0/composer.raw", false},
		{"codex", "testdata/tty/codex/0.152.0/composer.raw", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			raw, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			p := &proxy{agentBasename: tc.agent, stdout: io.Discard, lifecycleEvents: make(chan TurnObservation, 8)}
			p.orientation = newOrientationDelivery(orientation.Request{SchemaVersion: 1, Tag: "t", Agent: tc.agent, Attempt: "n", Body: "read context"})
			states := make(chan orientation.DeliveryState, 1)
			p.orientation.publish = func(s orientation.DeliveryState) { states <- s }
			p.orientation.deadlineAfter = 20 * time.Millisecond
			p.orientation.settleAfter = time.Millisecond
			if err := p.configureHarnessTTY(false, 120, 38); err != nil {
				t.Fatal(err)
			}
			defer p.closeTerminal()
			r, w := io.Pipe()
			defer r.Close()
			out := newDrainBuffer()
			done := make(chan struct{})
			go func() { p.translateStdinFrom(r, out, time.Millisecond); close(done) }()
			var rolling []byte
			p.handleChunk(raw, &rolling)
			var state orientation.DeliveryState
			select {
			case state = <-states:
			case <-time.After(time.Second):
				t.Fatal("no state")
			}
			w.Close()
			<-done
			if (state.Phase == orientation.DeliverySubmitted) != tc.ready {
				t.Fatalf("state %#v, want ready %v", state, tc.ready)
			}
			if !tc.ready && len(out.Bytes()) != 0 {
				t.Fatalf("injected during startup %q", out.Bytes())
			}
			if tc.ready && !bytes.Contains(out.Bytes(), []byte("read context")) {
				t.Fatal("no prompt")
			}
		})
	}
}

func TestOrientationCodexLoadingRemainsPendingThroughHeaderRepaint(t *testing.T) {
	raw, err := os.ReadFile("testdata/orientation/codex/0.154.0/loading.raw")
	if err != nil {
		t.Fatal(err)
	}
	p, _ := orientationProxy(t)
	p.agentBasename = "codex"
	p.orientation.profile, _ = profileForHarness("codex", true)
	p.configureHarnessTTY(false, 120, 38)
	var rolling []byte
	p.handleChunk(raw, &rolling)
	// A partial repaint removes the startup card but leaves the provisional composer.
	p.handleChunk([]byte("\x1b[1;1H\x1b[2K\x1b[2;1H\x1b[2K\x1b[3;1H\x1b[2K\x1b[4;1H\x1b[2K\x1b[5;1H\x1b[2K\x1b[6;1H\x1b[2K\x1b[9;3H"), &rolling)
	p.orientation.mu.Lock()
	ready := p.orientation.ready
	p.orientation.mu.Unlock()
	if ready {
		t.Fatal("a disappearing loading card authorized the provisional composer")
	}
	resolved, err := os.ReadFile("testdata/orientation/codex/0.154.0/ready.raw")
	if err != nil {
		t.Fatal(err)
	}
	p.handleChunk(resolved, &rolling)
	p.orientation.mu.Lock()
	ready = p.orientation.ready
	p.orientation.mu.Unlock()
	if !ready {
		t.Fatal("resolved startup did not release pending readiness")
	}
}

func TestOrientationAgyLongPromptSurvivesPostPasteFooterChange(t *testing.T) {
	p, _ := orientationProxy(t)
	p.agentBasename = "agy"
	p.orientation.profile, _ = profileForHarness("agy", true)
	p.configureHarnessTTY(false, 120, 38)
	body, err := orientation.BuildPrompt(orientation.OrientationContext{Tag: "work", WorkingPath: "/synthetic/project", SourceAgent: "codex", SourceSession: "old-session", TargetAgent: "agy", PairLog: "/synthetic/log-work.md", ScrollbackRaw: "/synthetic/parked.raw", ScrollbackEvents: "/synthetic/parked.events.jsonl", Renderer: "/synthetic/pair", NativeTranscripts: []string{"/synthetic/native.jsonl"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 1000 {
		t.Fatal("expected an actual long orientation prompt")
	}
	p.orientation.request.Body = body
	p.orientation.settleAfter = 100 * time.Millisecond
	states := make(chan orientation.DeliveryState, 1)
	p.orientation.publish = func(s orientation.DeliveryState) { states <- s }
	r, w := io.Pipe()
	defer r.Close()
	out := newDrainBuffer()
	done := make(chan struct{})
	go func() { p.translateStdinFrom(r, out, time.Millisecond); close(done) }()
	var rolling []byte
	ready, err := os.ReadFile("testdata/orientation/agy/1.1.25/ready-no-color.raw")
	if err != nil {
		t.Fatal(err)
	}
	p.handleChunk(ready, &rolling)
	if !waitFor(time.Second, func() bool { return bytes.Contains(out.Bytes(), []byte(body)) }) {
		t.Fatal("no long body paste")
	}
	pasted, err := os.ReadFile("testdata/orientation/agy/1.1.25/pasted-no-color.raw")
	if err != nil {
		t.Fatal(err)
	}
	p.handleChunk(pasted, &rolling)
	select {
	case state := <-states:
		if state.Phase != orientation.DeliverySubmitted {
			t.Fatalf("postpaste state %#v", state)
		}
	case <-time.After(time.Second):
		t.Fatal("no submission")
	}
	w.Close()
	<-done
	if !bytes.Equal(out.Bytes(), []byte("\x1b[200~"+body+"\x1b[201~\r")) {
		t.Fatal("long prompt changed or submission omitted")
	}
}
