package wrapcmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// pair#438: a delivery to an agent streaming a long turn. The screens are
// re-rendered from the recorded parley.nvim:1 turn (testdata README).

func peerBusyScreen(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/peer/claude/2.1.296/" + name + ".raw")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestPeerComposerReadsRecordedTurnScreens(t *testing.T) {
	for name, want := range map[string]PeerComposerState{"turn-empty": PeerComposerEmpty, "turn-paste": PeerComposerOccupied, "turn-queued": PeerComposerEmpty} {
		m, err := newTerminalModel(94, 39)
		if err != nil {
			t.Fatal(err)
		}
		if err := m.Feed([]byte(peerBusyScreen(t, name))); err != nil {
			t.Fatal(err)
		}
		if got := peerComposerState("claude", m.Snapshot()); got != want {
			t.Errorf("%s: composer %v want %v", name, got, want)
		}
		if !m.FrameComplete() {
			t.Errorf("%s: a closed synchronized update is not a complete frame", name)
		}
		m.Close()
	}
}

func TestTerminalModelFrameComplete(t *testing.T) {
	m, err := newTerminalModel(80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	for _, step := range []struct {
		feed string
		want bool
	}{
		{"plain output", false}, // never framed: cannot say
		{"\x1b[?2026h", false},
		{"partial frame", false},
		{"\x1b[?2026l", true},
		{"between frames", true},
		{"\x1b[?25;2026h", false},
		{"\x1b[?2026;25l", true},
		{"\x1bc", false}, // a reset forgets the framing
	} {
		if err := m.Feed([]byte(step.feed)); err != nil {
			t.Fatal(err)
		}
		if got := m.FrameComplete(); got != step.want {
			t.Fatalf("after %q: FrameComplete %v want %v", step.feed, got, step.want)
		}
	}
}

// peerBusyFixture is a Claude Code turn in progress: the recorded mid-turn
// screen, and one output chunk always in flight, as a streaming agent has.
func peerBusyFixture(t *testing.T, framed bool) (*harnessSessionFake, *peerDelivery, *time.Time, func(string) string) {
	t.Helper()
	screen := func(name string) string {
		s := peerBusyScreen(t, name)
		if !framed {
			s = strings.NewReplacer("\x1b[?2026h", "", "\x1b[?2026l", "").Replace(s)
		}
		return s
	}
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	f.resize(94, 39)
	now := time.Now()
	d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
	f.proxy.peer = d
	peerRender(f, d, screen("turn-empty"))
	d.observeOutput([]byte("the next spinner frame, still being read"))
	peerIntegrationEnqueue(t, d)
	return f, d, &now, screen
}

func TestPeerBusyAgentQueuedMessageSubmits(t *testing.T) {
	f, d, now, screen := peerBusyFixture(t, true)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("no paste into a streaming agent's empty composer: %+v", d.receipt())
	}
	peerRender(f, d, screen("turn-paste"))
	*now = now.Add(PeerSubmitDelay)
	f.proxy.dispatchPeer(&out)
	if !bytes.HasSuffix(out.Bytes(), f.proxy.ttyProfile.keymap.altCR) {
		t.Fatalf("no submit while output is in flight: %+v", d.receipt())
	}
	// Claude Code queues the message behind the running turn: the composer
	// clears and shows the faint queued-message hint; no new turn opens.
	*now = now.Add(time.Second)
	peerRender(f, d, screen("turn-queued"))
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Submitted {
		t.Fatalf("queued message not confirmed: %+v", r)
	}
}

// The chosen policy pastes mid-turn: three minutes of streaming neither
// holds nor expires the delivery, and it is pasted once.
func TestPeerBusyAgentStreamingMinutesLands(t *testing.T) {
	f, d, now, screen := peerBusyFixture(t, true)
	var out bytes.Buffer
	frame := "turn-empty"
	for end := now.Add(3*time.Minute + 30*time.Second); now.Before(end); *now = now.Add(250 * time.Millisecond) {
		peerRender(f, d, screen(frame))
		f.proxy.dispatchPeer(&out)
		switch {
		case d.receipt().Status.Terminal():
		case d.receipt().Status == couchmessage.Delivering && d.state.Phase == couchmessage.PeerDeliveryConfirming:
			frame = "turn-queued"
		case d.receipt().Status == couchmessage.Delivering:
			frame = "turn-paste"
		}
	}
	if r := d.receipt(); r.Status != couchmessage.Submitted || strings.Count(out.String(), "hello") != 1 {
		t.Fatalf("streaming agent: %+v pasted %d times", r, strings.Count(out.String(), "hello"))
	}
}

// A stream that never frames its output cannot be read mid-flight; it keeps
// its receipt queued and names the standing wait.
func TestPeerUnframedStreamWaitsNamingOutput(t *testing.T) {
	f, d, now, screen := peerBusyFixture(t, false)
	var out bytes.Buffer
	deadline := d.receipt().Message.Deadline
	for ; now.Before(deadline.Add(time.Second)); *now = now.Add(250 * time.Millisecond) {
		peerRender(f, d, screen("turn-empty"))
		f.proxy.dispatchPeer(&out)
	}
	if r := d.receipt(); r.Status != couchmessage.Queued || r.Detail != "waiting for rendered output" || out.Len() != 0 {
		t.Fatalf("unframed stream: %+v %q", r, out.String())
	}
}

// Changing output/composer waits cannot consume an accepted message's life.
func TestPeerChangingWaitsDoNotExpire(t *testing.T) {
	f, d, now, _ := peerBusyFixture(t, true)
	var out bytes.Buffer
	deadline := d.receipt().Message.Deadline
	for i := 0; now.Before(deadline); i++ {
		if i%5 == 4 {
			peerRender(f, d, "\x1b[?2026h\x1b[1;1Hhalf a frame")
		} else {
			peerRender(f, d, "\x1b[?2026h\x1b[2J\x1b[1;1Ha dialog, no composer\x1b[?2026l")
		}
		f.proxy.dispatchPeer(&out)
		*now = now.Add(250 * time.Millisecond)
	}
	f.proxy.dispatchPeer(&out)
	r := d.receipt()
	if r.Status != couchmessage.Queued || r.Detail == "" || out.Len() != 0 {
		t.Fatalf("connected wait: %+v", r)
	}
}
