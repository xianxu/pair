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
// "The chord fired" is observed as the child never seeing the bytes. That
// guarantee is structural, not a property of Decide's dispositions: the pump
// consumes a recognised chord via chordRest before any writeActive, and the
// two executors it hands the chord to cannot write — handleChord is never
// given the mux, and handleTerminalChord holds it but calls only the tab
// verbs (newTab/closeActive/previousTab/nextTab/reportError). So a write of
// chord bytes is exactly the misroute. Errors reported through reportError
// (the fake runtime has no panes) are expected and not asserted; ChordAltR
// begins a rename, which is also not a write.
func TestEveryChordSplitAtEveryByteResolvesAgainstTheDeadline(t *testing.T) {
	const nonChord = 'q'
	for _, seq := range workbenchshortcut.ChordSequences() {
		for cut := 1; cut < len(seq); cut++ {
			head, tail := []byte(seq[:cut]), []byte(seq[cut:])
			name := fmt.Sprintf("%q|%q", head, tail)

			// Guard the axis: q must extend no prefix and complete no chord,
			// or case (c) would silently change meaning when a chord lands
			// on q.
			extended := append(append([]byte(nil), head...), nonChord)
			if workbenchshortcut.IsChordPrefix(extended) {
				t.Fatalf("%q+q is still a chord prefix; pick another non-chord byte", head)
			}
			if _, ok := workbenchshortcut.DecodeChord(extended); ok {
				t.Fatalf("%q+q is a chord; pick another non-chord byte", head)
			}

			t.Run("tail before deadline fires the chord/"+name, func(t *testing.T) {
				mux := &fakeMux{activeName: "work"}
				timer := beforeDeadline()
				pumpStdinWithTimer(&splitReader{chunks: [][]byte{head, tail}}, mux, &fakeRuntime{}, io.Discard, timer)
				for _, op := range mux.ops {
					if strings.HasPrefix(op, "write:") {
						t.Fatalf("ops = %v: chord bytes reached the child", mux.ops)
					}
				}
				if timer.resets == 0 {
					t.Fatal("held prefix never armed the deadline")
				}
			})

			t.Run("deadline first forwards the prefix as typed/"+name, func(t *testing.T) {
				mux := &fakeMux{}
				timer := newFiringEscapeTimer()
				pumpStdinWithTimer(&splitReader{chunks: [][]byte{head}}, mux, &fakeRuntime{}, io.Discard, timer)
				if got := strings.Join(mux.ops, ","); got != "write:"+string(head) {
					t.Fatalf("ops = %q, want the prefix forwarded once", got)
				}
				// Same as the bare-ESC regression: the EOF flush writes the
				// same bytes, so only the arm proves the deadline was in play.
				if timer.resets == 0 {
					t.Fatal("held prefix never armed the deadline")
				}
			})

			t.Run("a non-chord byte resolves the prefix as typed/"+name, func(t *testing.T) {
				mux := &fakeMux{}
				timer := beforeDeadline()
				pumpStdinWithTimer(&splitReader{chunks: [][]byte{head, {nonChord}}}, mux, &fakeRuntime{}, io.Discard, timer)
				if got := strings.Join(mux.ops, ","); got != "write:"+string(head)+string(nonChord) {
					t.Fatalf("ops = %q, want prefix+q forwarded, no chord", got)
				}
			})
		}
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
