package wrapcmd

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func peerTestMessage(now time.Time) couchmessage.Message {
	return couchmessage.Message{ID: "test-353", From: couchmessage.Binding{Slot: "brain:0"}, To: couchmessage.Binding{Slot: "pair:1"}, Body: "hello", Deadline: now.Add(time.Second)}
}
func TestPeerDeliveryPreservesHumanInputAndDeadline(t *testing.T) {
	for _, interrupt := range []string{"operator", "deadline", "image"} {
		t.Run(interrupt, func(t *testing.T) {
			now := time.Now()
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
			f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
			f.proxy.peer = d
			if err := d.reserve("test-353", 0); err != nil {
				t.Fatal(err)
			}
			if err := d.enqueue(peerTestMessage(now)); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			f.proxy.dispatchPeer(&out)
			if !strings.Contains(out.String(), "hello") {
				t.Fatalf("no paste: %q", out.String())
			}
			before := out.String()
			switch interrupt {
			case "operator":
				d.admitInput([]byte("human"))
			case "image":
				d.admitImage()
			case "deadline":
				// The window runs from the paste (pair#427).
				now = now.Add(couchmessage.DeliveryTimeout)
			}
			f.proxy.dispatchPeer(&out)
			// A later matching render must not revive automatic submission.
			f.output(strings.Replace(claudeLiveComposerPaint(), "alpha", peerEnvelope(peerTestMessage(now)), 1))
			f.proxy.dispatchPeer(&out)
			if out.String() != before {
				t.Fatalf("automatic bytes after %s: %q", interrupt, out.String())
			}
			if r := d.receipt(); !r.Status.Terminal() {
				t.Fatalf("not terminal: %+v", r)
			}
		})
	}
}

func TestPeerDeliveryOccupiedComposerWritesNothing(t *testing.T) {
	now := time.Now()
	f := newHarnessSessionFake(t, "claude", true)
	defer f.close()
	f.output(claudeLiveComposerPaint())
	d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
	f.proxy.peer = d
	if err := d.reserve("test-353", 0); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(peerTestMessage(now)); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 {
		t.Fatalf("overwrote composer: %q", out.String())
	}
}

// peerEmptyComposer and peerFilledComposer repaint the fake Claude's whole
// screen, as an agent does on each frame.
func peerEmptyComposer() string {
	return "\x1b[2J\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H"
}
func peerFilledComposer(lines ...string) string {
	return "\x1b[2J" + claudeBox(5, "❯", "136;136;136", lines...) + "\x1b[?25h\x1b[7;3H"
}

// peerRender delivers one frame the way the wrapper sees agent output.
func peerRender(f *harnessSessionFake, d *peerDelivery, paint string) {
	d.observeOutput([]byte(paint))
	f.output(paint)
}

// pair#427 Spec 1-2: paste, wait the fixed delay, submit, then confirm after
// the fact. What the agent drew is never compared with the envelope: the
// boot-time case rendered the text unlike any projection, and the multi-line
// case (#418) collapses it.
func TestPeerDeliverySubmitsAfterDelayThenConfirms(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rendered []string // the composer after the paste; nil leaves it unrecognized
		confirm  func(*harnessSessionFake, *peerDelivery)
		want     couchmessage.Status
		evidence string
	}{
		{"composer clears", []string{"unrelated boot-time text"}, func(f *harnessSessionFake, d *peerDelivery) { peerRender(f, d, peerEmptyComposer()) }, couchmessage.Submitted, ""},
		{"collapsed multi-line", []string{"[Pasted text #1 +5 lines]"}, func(f *harnessSessionFake, d *peerDelivery) { peerRender(f, d, peerEmptyComposer()) }, couchmessage.Submitted, ""},
		{"turn opens", nil, func(f *harnessSessionFake, d *peerDelivery) { f.proxy.turnActive.Store(true) }, couchmessage.Submitted, ""},
		{"never consumed", []string{"unrelated boot-time text"}, nil, couchmessage.Indeterminate, "composer still holds text; no turn started"},
		{"empty, paste never shown", nil, func(f *harnessSessionFake, d *peerDelivery) { peerRender(f, d, peerEmptyComposer()) }, couchmessage.Indeterminate, "never showed the paste"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			f := newHarnessSessionFake(t, "claude", true)
			defer f.close()
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
			paste := out.String()
			if !strings.Contains(paste, "hello") {
				t.Fatalf("no paste: %q", paste)
			}
			if tc.rendered != nil {
				peerRender(f, d, peerFilledComposer(tc.rendered...))
			} else {
				peerRender(f, d, "\x1b[2J")
			}
			f.proxy.dispatchPeer(&out)
			if out.String() != paste {
				t.Fatal("submitted before the delay")
			}
			// Past the original paste-by deadline: the window runs from the paste.
			now = now.Add(PeerSubmitDelay + time.Second)
			f.proxy.dispatchPeer(&out)
			if out.String() != paste+"\r" {
				t.Fatalf("missing single submit: %q", out.String())
			}
			if r := d.receipt(); r.Status != couchmessage.Delivering {
				t.Fatalf("a submit write is not a landed submission: %+v", r)
			}
			if tc.confirm != nil {
				tc.confirm(f, d)
			}
			f.proxy.dispatchPeer(&out)
			if tc.want == couchmessage.Indeterminate {
				if d.receipt().Status.Terminal() {
					t.Fatalf("decided before the window: %+v", d.receipt())
				}
				now = now.Add(couchmessage.DeliveryTimeout)
				f.proxy.dispatchPeer(&out)
			}
			r := d.receipt()
			if r.Status != tc.want || !strings.Contains(r.Detail, tc.evidence) || (tc.want == couchmessage.Indeterminate && !strings.HasPrefix(r.Detail, "uncertain: ")) {
				t.Fatalf("receipt %+v, want %s with %q", r, tc.want, tc.evidence)
			}
			if out.String() != paste+"\r" {
				t.Fatalf("extra automatic bytes: %q", out.String())
			}
		})
	}
}

// pair#427 Done-when: a send right after a restart, while the agent is still
// booting. The composer appears only after the old 30s admission deadline;
// the paste-by budget covers it and the window starts at the paste.
func TestPeerDeliveryDuringSimulatedBoot(t *testing.T) {
	now := time.Now()
	f := newHarnessSessionFake(t, "claude", true)
	defer f.close()
	m := peerTestMessage(now)
	m.Deadline = now.Add(couchmessage.PasteTimeout)
	d := newPeerDelivery(m.To, func() time.Time { return now })
	f.proxy.peer = d
	if err := d.reserve(m.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	peerRender(f, d, "\x1b[2J  Claude Code starting…")
	f.proxy.dispatchPeer(&out)
	now = now.Add(40 * time.Second)
	f.proxy.dispatchPeer(&out)
	if out.Len() != 0 || d.receipt().Status != couchmessage.Queued {
		t.Fatalf("pasted into a booting agent: %q %+v", out.String(), d.receipt())
	}
	peerRender(f, d, peerEmptyComposer())
	f.proxy.dispatchPeer(&out)
	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("no paste once the composer came up: %q %+v", out.String(), d.receipt())
	}
	// The booting agent echoes the paste as plain text.
	peerRender(f, d, peerFilledComposer("[Couch peer from brain:0; delivery test-353]", "hello"))
	now = now.Add(PeerSubmitDelay)
	f.proxy.dispatchPeer(&out)
	if !strings.HasSuffix(out.String(), "\r") {
		t.Fatalf("no submit: %q", out.String())
	}
	f.proxy.turnActive.Store(true)
	f.proxy.dispatchPeer(&out)
	if r := d.receipt(); r.Status != couchmessage.Submitted {
		t.Fatalf("receipt %+v", r)
	}
}

func TestPeerDeliveryUsesStdinOwnerAndDoesNotSubmitAfterTyping(t *testing.T) {
	now := time.Now()
	f := newHarnessSessionFake(t, "claude", true)
	defer f.close()
	f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
	m := peerTestMessage(now)
	m.Deadline = now.Add(time.Minute)
	d := newPeerDelivery(m.To, time.Now)
	f.proxy.peer = d
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	out := newDrainBuffer()
	done := make(chan struct{})
	go func() { f.proxy.translateStdinFrom(reader, out, time.Millisecond); close(done) }()
	defer func() {
		writer.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("stdin owner did not exit")
		}
	}()
	if err := d.reserve(m.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.enqueue(m); err != nil {
		t.Fatal(err)
	}
	if !waitFor(time.Second, func() bool { return bytes.Contains(out.Bytes(), []byte("hello")) }) {
		t.Fatal("pending message never reached input writer")
	}
	if _, err := writer.Write([]byte("operator")); err != nil {
		t.Fatal(err)
	}
	if !waitFor(time.Second, func() bool { return bytes.HasSuffix(out.Bytes(), []byte("operator")) }) {
		t.Fatal("operator input not forwarded")
	}
	if !waitFor(time.Second, func() bool { return d.receipt().Status.Terminal() }) {
		t.Fatal("interruption did not terminate delivery")
	}
	// Typing after the paste leaves the text's fate unknown (pair#427).
	if bytes.Contains(out.Bytes(), []byte{'\r'}) || d.receipt().Status != couchmessage.Indeterminate {
		t.Fatalf("auto-submit after typing: %q %+v", out.Bytes(), d.receipt())
	}
}

func TestPeerReservationExpiresAndOutputInvalidatesObservation(t *testing.T) {
	now := time.Now()
	d := newPeerDelivery(peerTestMessage(now).To, func() time.Time { return now })
	d.observeOutput([]byte("spinner"))
	if err := d.reserve("stale", 0); err == nil {
		t.Fatal("stale output sequence reserved")
	}
	d.outputForwarded()
	if err := d.reserve("first", 1); err != nil {
		t.Fatal(err)
	}
	now = now.Add(3 * time.Second)
	if err := d.enqueue(peerTestMessage(now)); err == nil {
		t.Fatal("late commit accepted")
	}
	if err := d.reserve("new", 1); err != nil {
		t.Fatalf("expired reservation stranded: %v", err)
	}
}
