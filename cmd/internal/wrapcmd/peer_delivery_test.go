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
				now = now.Add(2 * time.Second)
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

func TestPeerDeliveryRequiresFreshMatchingRenderThenSubmitsOnce(t *testing.T) {
	now := time.Now()
	f := newHarnessSessionFake(t, "claude", true)
	defer f.close()
	f.output("\x1b[?2004h" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
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
	f.proxy.dispatchPeer(&out)
	if out.String() != paste {
		t.Fatal("submit without fresh render")
	}
	paint := "\x1b[2J" + claudeBox(5, "❯", "136;136;136", strings.Split(peerEnvelope(m), "\n")...) + "\x1b[?25h\x1b[7;3H"
	d.observeOutput([]byte(paint))
	f.output(paint)
	f.proxy.dispatchPeer(&out)
	if out.String() != paste+"\r" {
		t.Fatalf("missing single submit: %q", out.String())
	}
	f.proxy.dispatchPeer(&out)
	if out.String() != paste+"\r" || d.receipt().Status != couchmessage.Submitted {
		t.Fatal("submission repeated or status missing")
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
	if bytes.Contains(out.Bytes(), []byte{'\r'}) || d.receipt().Status != couchmessage.Cancelled {
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
