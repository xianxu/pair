package termcmd

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// A lone ESC is the first byte of every legacy Alt chord, so the pump holds
// it. Without a deadline it was held until the NEXT keystroke, which then
// decided what the ESC became: ESC,ESC reached nvim as two bytes at once
// (insert mode left on the second press), and ESC,j became ChordAltJ (#234).
func TestBareEscapeIsForwardedAfterTheDeadline(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{}
	timer := newFiringEscapeTimer()

	pumpStdinWithTimer(&splitReader{chunks: [][]byte{{0x1b}}}, mux, rt, io.Discard, timer)

	if got := strings.Join(mux.ops, ","); got != "write:\x1b" {
		t.Fatalf("ops = %q, want the ESC forwarded exactly once", got)
	}
	// The EOF path also forwards a held ESC, so the ops alone cannot tell the
	// deadline flush from the EOF flush: the arm is the evidence.
	if timer.resets == 0 {
		t.Fatal("held ESC never armed the deadline")
	}
}

func TestEscapeThenJAfterTheDeadlineIsTwoKeysNotAltJ(t *testing.T) {
	rt := &fakeRuntime{}
	wrote := make(chan string, 2)
	mux := &fakeMux{wrote: wrote}
	release := make(chan struct{})
	reader := &gatedChunksReader{chunks: [][]byte{{0x1b}, {'j'}}, release: release}
	timer := newFiringEscapeTimer()
	done := make(chan struct{})

	go func() {
		pumpStdinWithTimer(reader, mux, rt, io.Discard, timer)
		close(done)
	}()

	select {
	case got := <-wrote:
		if got != "\x1b" {
			t.Fatalf("first write = %q, want the bare ESC", got)
		}
	case <-time.After(time.Second):
		t.Fatal("deadline did not forward the held ESC")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pump did not finish")
	}
	if got := strings.Join(mux.ops, ","); got != "write:\x1b,write:j" {
		t.Fatalf("ops = %q, want ESC then j as two keys, not ChordAltJ", got)
	}
}

// Every chord, every split point, both sides of the deadline. Generated from
// the chord table so a chord added later is covered without editing this.
//
// Draft-only chords reach the shell byte-for-byte; every other recognised
// chord is consumed. Errors reported through reportError (the fake runtime
// has no panes) are expected and not asserted; ChordAltR begins a rename,
// which is also not a write.
func TestEveryChordSplitAtEveryByteResolvesAgainstTheDeadline(t *testing.T) {
	for _, seq := range workbenchshortcut.ChordSequences() {
		chord, ok := workbenchshortcut.DecodeChord([]byte(seq))
		if !ok {
			t.Fatalf("%q did not decode to a chord", seq)
		}
		for cut := 1; cut < len(seq); cut++ {
			head, tail := []byte(seq[:cut]), []byte(seq[cut:])
			t.Run(fmt.Sprintf("%q/split%d", seq, cut), func(t *testing.T) {
				mux := &fakeMux{activeName: "work"}
				timer := beforeDeadline()
				pumpStdinWithTimer(&splitReader{chunks: [][]byte{head, tail}}, mux, &fakeRuntime{}, io.Discard, timer)
				if workbenchshortcut.IsDraftChord(chord) {
					if got := strings.Join(mux.ops, ","); got != "write:"+seq {
						t.Fatalf("draft-only chord %q: ops = %q, want the raw bytes forwarded", seq, got)
					}
				} else {
					for _, op := range mux.ops {
						if strings.HasPrefix(op, "write:") {
							t.Fatalf("complete chord reached child:%v", mux.ops)
						}
					}
				}
				if len(head) == 1 && head[0] == 27 && timer.resets == 0 {
					t.Fatal("bare Escape had no ambiguity deadline")
				}
			})
		}
	}
	if got := strings.Join(forwardedOnTheDeadline(t, []byte{27}), ","); got != "write:\x1b" {
		t.Fatalf("Escape deadline:%q", got)
	}
}

// forwardedOnTheDeadline feeds head as one read, lets the fake deadline fire,
// waits for the pump to write something, and only THEN releases EOF. The ops
// it returns therefore contain whatever the expiry branch forwarded before
// EOF could — the EOF flush cannot be what produced the first write.
func forwardedOnTheDeadline(t *testing.T, head []byte) []string {
	t.Helper()
	wrote := make(chan string, 4)
	mux := &fakeMux{wrote: wrote}
	release := make(chan struct{})
	reader := &gatedEOFReader{data: head, release: release}
	timer := newFiringEscapeTimer()
	done := make(chan struct{})
	go func() {
		pumpStdinWithTimer(reader, mux, &fakeRuntime{}, io.Discard, timer)
		close(done)
	}()
	select {
	case <-wrote:
	case <-time.After(time.Second):
		t.Fatalf("held %q was never forwarded on the deadline", head)
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pump did not finish after EOF")
	}
	if timer.resets == 0 {
		t.Fatal("held prefix never armed the deadline")
	}
	return mux.ops
}

// A torn SGR mouse report is held by the same buffer as a chord prefix, so it
// meets the same deadline. Chosen, not incidental (#234 BR-5): couch arms only
// for a lone ESC, but pair term's stdin is a local zellij pty where a report
// torn across reads is re-joined within microseconds, and a prefix left
// pending with no deadline is the stuck keyboard this issue fixed.
func TestTornMouseReportDoesNotBecomeRawChildInput(t *testing.T) {
	head, tail := []byte("\x1b[<0;8"), []byte(";1M")
	mux := &fakeMux{}
	pumpStdinWithTimer(&splitReader{chunks: [][]byte{head}}, mux, &fakeRuntime{}, io.Discard, beforeDeadline())
	if len(mux.ops) != 0 {
		t.Fatalf("incomplete report escaped:%v", mux.ops)
	}
	pumpStdinWithTimer(&splitReader{chunks: [][]byte{head, tail}}, mux, &fakeRuntime{}, io.Discard, beforeDeadline())
	if got := strings.Join(mux.ops, ","); got != "write:"+string(head)+string(tail) {
		t.Fatalf("completed report:%q", got)
	}
}

// The fakes above prove the arm rule; this proves the REAL timer behind
// pumpStdin honours it — newRealEscapeTimer starts stopped and drained, and a
// Reset from that state must actually tick. Bounded by a one-second wait, far
// above the 35 ms deadline, so it cannot flake on a loaded machine.
func TestTheRealDeadlineForwardsABareEscape(t *testing.T) {
	rt := &fakeRuntime{}
	wrote := make(chan string, 2)
	mux := &fakeMux{wrote: wrote}
	release := make(chan struct{})
	reader := &gatedChunksReader{chunks: [][]byte{{0x1b}, {'j'}}, release: release}
	done := make(chan struct{})

	started := time.Now()
	go func() {
		pumpStdin(reader, mux, rt, io.Discard)
		close(done)
	}()

	select {
	case got := <-wrote:
		if got != "\x1b" {
			t.Fatalf("first write = %q, want the bare ESC", got)
		}
		if elapsed := time.Since(started); elapsed < workbenchshortcut.EscapeAmbiguity {
			t.Fatalf("ESC forwarded after %v, before the %v deadline", elapsed, workbenchshortcut.EscapeAmbiguity)
		}
	case <-time.After(time.Second):
		t.Fatal("the real deadline never forwarded the held ESC")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pump did not finish")
	}
	if got := strings.Join(mux.ops, ","); got != "write:\x1b,write:j" {
		t.Fatalf("ops = %q, want ESC then j as two keys", got)
	}
}
