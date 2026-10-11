package wrapcmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// pair#428 Spec 1: only real keystrokes count as operator input for a
// delivery. Mouse, focus and terminal reports do not, solicited or not.
var peerInputClasses = []struct {
	name      string
	chunks    []string
	keystroke bool
}{
	{"text", []string{"human"}, true},
	{"ctrl-c", []string{"\x03"}, true},
	{"backspace", []string{"\x7f"}, true},
	{"legacy alt+enter", []string{"\x1b\r"}, true},
	{"kitty alt+enter (kink 47)", []string{"\x1b[13;3u"}, true},
	{"kitty key", []string{"\x1b[97u"}, true},
	{"arrow", []string{"\x1b[A"}, true},
	{"modified F3 is not a CPR", []string{"\x1b[1;5R"}, true},
	{"operator paste", []string{"\x1b[200~text\x1b[201~"}, true},
	{"focus in", []string{"\x1b[I"}, false},
	{"focus out", []string{"\x1b[O"}, false},
	{"sgr mouse motion", []string{"\x1b[<35;10;5M"}, false},
	{"sgr mouse click", []string{"\x1b[<0;10;5M\x1b[<0;10;5m"}, false},
	{"sgr wheel", []string{"\x1b[<64;40;12M\x1b[<65;40;12M"}, false},
	{"split sgr mouse", []string{"\x1b[<35;1", "0;5M"}, false},
	{"urxvt mouse", []string{"\x1b[32;10;5M"}, false},
	{"x10 mouse", []string{"\x1b[M #!"}, false},
	{"split x10 mouse", []string{"\x1b[M", " #!"}, false},
	{"color-scheme notice", []string{"\x1b[?997;1n"}, false},
	{"unsolicited DSR", []string{"\x1b[0n"}, false},
	{"unsolicited DA1", []string{"\x1b[?62;22c"}, false},
	{"unsolicited DA2", []string{"\x1b[>1;10;0c"}, false},
	{"unsolicited DECRPM", []string{"\x1b[?2026;2$y"}, false},
	{"kitty flags report", []string{"\x1b[?1u"}, false},
	{"window report", []string{"\x1b[8;40;120t"}, false},
	{"osc color reply", []string{"\x1b]11;rgb:0000/0000/0000\x1b\\"}, false},
	{"dcs version reply", []string{"\x1bP>|zellij\x1b\\"}, false},
	{"mouse then text", []string{"\x1b[<35;10;5Mx"}, true},
	{"text then mouse", []string{"x\x1b[<35;10;5M"}, true},
}

func TestPeerInputOnlyKeystrokesCount(t *testing.T) {
	for _, tc := range peerInputClasses {
		t.Run(tc.name, func(t *testing.T) {
			d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
			got := false
			for _, chunk := range tc.chunks {
				human, _ := d.admitInput([]byte(chunk))
				got = got || human
				d.inputForwarded(human, false)
			}
			d.mu.Lock()
			pending := len(d.replies.input)
			d.mu.Unlock()
			if got != tc.keystroke {
				t.Fatalf("keystroke=%t want %t", got, tc.keystroke)
			}
			if pending != 0 && !tc.keystroke {
				t.Fatalf("complete input retained %d framing bytes", pending)
			}
		})
	}
}

// peerPastedDelivery is a delivery in its paste→submit half.
func peerPastedDelivery(t *testing.T) (*harnessSessionFake, *peerDelivery, *bytes.Buffer, *time.Time) {
	t.Helper()
	now := time.Now()
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	f.output(peerEmptyComposer())
	m := peerTestMessage(now)
	d := newPeerDelivery(m.To, func() time.Time { return now })
	f.proxy.peer = d
	if err := d.reserve(m.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(string(out.Bytes()), "hello") {
		t.Fatalf("no paste: %q", string(out.Bytes()))
	}
	peerRender(f, d, peerFilledComposer("pasted"))
	return f, d, &out, &now
}

// Each class through admitInput and dispatchPeer: a non-keystroke leaves the
// delivery to submit and confirm; a keystroke interrupts it, as before.
func TestPeerDeliverySurvivesNonKeystrokes(t *testing.T) {
	for _, tc := range peerInputClasses {
		if strings.Contains(tc.name, "paste") {
			continue // an operator paste is held, not interpreted (TestPeerDraftPasteWaitsForDelivery)
		}
		t.Run(tc.name, func(t *testing.T) {
			f, d, out, now := peerPastedDelivery(t)
			for _, chunk := range tc.chunks {
				human, held := d.admitInput([]byte(chunk))
				if held {
					t.Fatalf("held %q", chunk)
				}
				d.inputForwarded(human, false)
			}
			*now = now.Add(PeerSubmitDelay)
			f.proxy.dispatchPeer(out)
			peerRender(f, d, peerEmptyComposer())
			f.proxy.dispatchPeer(out)
			r := d.receipt()
			if tc.keystroke {
				if r.Status == couchmessage.Submitted || !strings.Contains(r.Detail, "operator input") {
					t.Fatalf("keystroke did not interrupt: %+v", r)
				}
				return
			}
			if r.Status != couchmessage.Submitted || !strings.HasSuffix(string(out.Bytes()), "\r") {
				t.Fatalf("receipt %+v, out %q", r, string(out.Bytes()))
			}
		})
	}
}

// Kink 47 (ops:0, 10-10 13:05): focus in, the operator's kitty Alt+Enter,
// focus out. Only the Alt+Enter is operator input.
func TestPeerKink47FocusThenAltEnter(t *testing.T) {
	d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
	for _, step := range []struct {
		chunk string
		human bool
	}{{"\x1b[I", false}, {"\x1b[13;3u", true}, {"\x1b[O", false}} {
		if human, _ := d.admitInput([]byte(step.chunk)); human != step.human {
			t.Fatalf("%q: human=%t", step.chunk, human)
		}
		d.inputForwarded(step.human, false)
	}
}

// A lone ESC is a keystroke once the flush interval passes with no
// continuation; it must not leave a delivery waiting for "incomplete
// terminal input" until the next byte.
func TestPeerLoneEscapeResolvesAtFlush(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	now := time.Now()
	d.now = func() time.Time { return now }
	human, _ := d.admitInput([]byte("\x1b"))
	if human || !d.inputForwarded(human, false) {
		t.Fatal("a lone ESC is not yet known to be a key")
	}
	d.flushPartialInput()
	peerIntegrationEnqueue(t, d)
	now = now.Add(2 * time.Second)
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(string(out.Bytes()), "hello") {
		t.Fatalf("delivery still blocked: %+v", d.receipt())
	}
}

// peerLoopFixture runs the real input writer over a pipe, with the delivery
// already pasted.
func peerLoopFixture(t *testing.T) (*harnessSessionFake, *peerDelivery, io.Writer, *drainBuffer) {
	t.Helper()
	f := newHarnessSessionFake(t, "claude", true)
	t.Cleanup(f.close)
	f.output(peerEmptyComposer())
	m := peerTestMessage(time.Now())
	m.Deadline = time.Now().Add(time.Minute)
	d := newPeerDelivery(m.To, time.Now)
	f.proxy.peer = d
	reader, writer := io.Pipe()
	out := newDrainBuffer()
	done := make(chan struct{})
	go func() { f.proxy.translateStdinFrom(reader, out, time.Millisecond); close(done) }()
	t.Cleanup(func() {
		writer.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("stdin owner did not exit")
		}
	})
	if err := d.reserve(m.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
	if !waitFor(time.Second, func() bool { return bytes.Contains(out.Bytes(), []byte("hello")) }) {
		t.Fatal("no paste")
	}
	peerRender(f, d, peerFilledComposer("pasted"))
	return f, d, writer, out
}

const peerDraftPaste = "\x1b[200~draft\x1b[201~"

// writeDraftSend is the draft pane's transaction as the agent pane's stdin
// sees it (nvim/draft_send.lua): focus, the paste, a settle, Alt+Enter, focus.
func writeDraftSend(t *testing.T, w io.Writer, settle time.Duration) {
	t.Helper()
	for _, chunk := range []string{"\x1b[I", peerDraftPaste, "", "\x1b\r", "\x1b[O"} {
		if chunk == "" {
			time.Sleep(settle)
			continue
		}
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
}

// pair#428 Done-when: a draft send during a delivery waits for it instead of
// interleaving; both submit, in order, unmixed.
func TestPeerDraftPasteWaitsForDelivery(t *testing.T) {
	f, d, w, out := peerLoopFixture(t)
	writeDraftSend(t, w, 20*time.Millisecond)
	if bytes.Contains(out.Bytes(), []byte("draft")) {
		t.Fatalf("draft interleaved with the delivery: %q", out.Bytes())
	}
	if !waitFor(time.Second, func() bool { return bytes.HasSuffix(out.Bytes(), []byte("\r")) }) {
		t.Fatalf("delivery never submitted: %q", out.Bytes())
	}
	peerRender(f, d, peerEmptyComposer())
	if !waitFor(time.Second, func() bool { return bytes.Contains(out.Bytes(), []byte("draft")) }) {
		t.Fatalf("draft never released: %q %+v", out.Bytes(), d.receipt())
	}
	if r := d.receipt(); r.Status != couchmessage.Submitted {
		t.Fatalf("delivery: %+v", r)
	}
	got := string(out.Bytes())
	robot, draft := strings.Index(got, "hello"), strings.Index(got, "draft")
	if submit := strings.Index(got[robot:], "\r") + robot; !(robot < submit && submit < draft) {
		t.Fatalf("out of order: %q", got)
	}
	if !waitFor(time.Second, func() bool { return strings.Count(string(out.Bytes()), "\r") == 2 }) {
		t.Fatalf("draft submit lost: %q", string(out.Bytes()))
	}
	d.mu.Lock()
	holding := d.holding
	d.mu.Unlock()
	if holding {
		t.Fatal("hold outlived its queue")
	}
}

// Past PeerDraftHold the operator's paste goes anyway, and the delivery that
// never confirmed reads as interrupted, honestly.
func TestPeerDraftHoldIsBounded(t *testing.T) {
	_, d, w, out := peerLoopFixture(t)
	start := time.Now()
	writeDraftSend(t, w, 20*time.Millisecond)
	if !waitFor(2*PeerDraftHold, func() bool { return bytes.Contains(out.Bytes(), []byte("draft")) }) {
		t.Fatal("hold never released")
	}
	if waited := time.Since(start); waited < PeerDraftHold-100*time.Millisecond {
		t.Fatalf("released early, after %s", waited)
	}
	if !waitFor(time.Second, func() bool { return d.receipt().Status.Terminal() }) {
		t.Fatalf("delivery not ended: %+v", d.receipt())
	}
	if r := d.receipt(); r.Status != couchmessage.Indeterminate || !strings.Contains(r.Detail, "operator input") {
		t.Fatalf("receipt %+v", r)
	}
}

// Held chunks replay with their arrival spacing.
func TestHeldInputKeepsGaps(t *testing.T) {
	d := newPeerDelivery(peerTestMessage(time.Now()).To, time.Now)
	var h heldInput
	defer h.stop()
	h.add([]byte("a"))
	time.Sleep(60 * time.Millisecond)
	h.add([]byte("b"))
	start := time.Now()
	if data, ok := h.next(d); !ok || string(data) != "a" {
		t.Fatal("first chunk not due at release")
	}
	if _, ok := h.next(d); ok {
		t.Fatal("second chunk ignored its gap")
	}
	<-h.timerC()
	if data, ok := h.next(d); !ok || string(data) != "b" || time.Since(start) < 50*time.Millisecond {
		t.Fatalf("second chunk %q after %s", data, time.Since(start))
	}
}

func TestViewportShift(t *testing.T) {
	var rows []string
	for i := 0; i < 30; i++ {
		rows = append(rows, fmt.Sprintf("line %02d", i))
	}
	view := func(shift int) []string { return append([]string(nil), rows[30-10-shift:30-shift]...) }
	spaced := view(0)
	spaced[4] = "  line   24  " // padding and wide-cell spacing differ
	repainted := view(0)
	repainted[9], repainted[8] = "spinner ⠋", "spinner ⠙"
	for _, tc := range []struct {
		name  string
		view  []string
		shift int
		ok    bool
	}{
		{"at the bottom", view(0), 0, true},
		{"scrolled three", view(3), 3, true},
		{"scrolled to the top", view(20), 20, true},
		{"whitespace ignored", spaced, 0, true},
		{"two repainted rows", repainted, 0, true},
		{"short dump padded", view(0)[:10], 0, true},
		{"unrelated", []string{"x", "y", "z", "w"}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shift, ok := viewportShift(rows, 10, tc.view)
			if shift != tc.shift || ok != tc.ok {
				t.Fatalf("shift=%d ok=%t", shift, ok)
			}
		})
	}
}

// pair#428 Done-when: scrolled up (composer out of view) holds the paste;
// scrolling back releases it.
func TestPeerDeliveryWaitsForComposerInView(t *testing.T) {
	f, d := peerIntegrationFixture(t)
	now := time.Now()
	d.now = func() time.Time { return now }
	var inView atomic.Bool
	probes := make(chan struct{}, 8)
	d.viewProbe = func() bool { probes <- struct{}{}; return inView.Load() }
	peerIntegrationEnqueue(t, d)
	var out bytes.Buffer
	settle := func() {
		t.Helper()
		select {
		case <-probes:
		case <-time.After(time.Second):
			t.Fatal("no viewport probe")
		}
		waitFor(time.Second, func() bool { d.mu.Lock(); defer d.mu.Unlock(); return !d.view.checking })
	}
	f.proxy.dispatchPeer(&out)
	settle()
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 || !strings.Contains(d.receipt().Detail, "in view") {
		t.Fatalf("pasted while scrolled: %q %+v", out.String(), d.receipt())
	}
	inView.Store(true)
	now = now.Add(peerViewFresh)
	f.proxy.dispatchPeer(&out)
	settle()
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("scrolling back did not release: %+v", d.receipt())
	}
}

func TestCursorRowInView(t *testing.T) {
	var rows []string
	for i := 0; i < 30; i++ {
		rows = append(rows, fmt.Sprintf("line %02d", i))
	}
	dump := func(shift int) string { return strings.Join(rows[20-shift:30-shift], "\n") + "\n" }
	for _, tc := range []struct {
		name   string
		dump   string
		inView bool
	}{
		{"bottom", dump(0), true},
		{"small scroll keeps the cursor row", dump(4), true},
		{"cursor row scrolled off", dump(5), false},
		{"far up", dump(20), false},
		{"unjudgeable reads as in view", "something else\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The cursor sits on screen row 5 of 10.
			if got := cursorRowInView(rows, 10, 5, tc.dump); got != tc.inView {
				t.Fatalf("inView=%t", got)
			}
		})
	}
}
