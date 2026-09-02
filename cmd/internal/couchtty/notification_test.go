package couchtty

import (
	"bytes"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func notificationConsole(t *testing.T) (*Console, *hostty.FakeHost) {
	t.Helper()
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	con := New(host, bytes.NewReader(nil))
	con.Attach("c1", "one", ptychild.NewFakeChild(nil))
	con.Attach("c2", "two", ptychild.NewFakeChild(nil))
	t.Cleanup(con.Stop)
	return con, host
}

func observedBatch(screen *ptychild.Screen, raw []byte) ptychild.OutputBatch {
	screen.Feed(raw)
	return ptychild.OutputBatch{
		Raw: raw, Parts: screen.TakeOutputParts(), RingEnd: screen.StreamEnd(),
		ReplaySafeEnd: screen.ReplaySafeEnd(),
	}
}

func TestOutputBatchFocusOrder(t *testing.T) {
	con, host := notificationConsole(t)
	con.switchTo("c2", false, arrivalOrdinary)
	host.Reset()

	// Delivery happened while c1 was inactive, but processing happens after
	// c1 becomes active. Processing-time focus owns ordinary output.
	queued := chunk{id: "c1", batch: ptychild.OutputBatch{
		Raw: []byte("queued"), Parts: []ptychild.OutputPart{{Bytes: []byte("queued")}},
		RingEnd: 6, ReplaySafeEnd: 6,
	}}
	con.switchTo("c1", false, arrivalOrdinary)
	host.Reset()
	con.onChunk(queued)
	if got := host.Written(); !stringsContains(got, "queued") {
		t.Fatalf("newly focused actor output was lost: %q", got)
	}

	host.Reset()
	con.switchTo("c2", false, arrivalOrdinary)
	host.Reset()
	con.onChunk(chunk{id: "c1", batch: queued.batch})
	if got := host.Written(); stringsContains(got, "queued") {
		t.Fatalf("inactive actor painted queued output: %q", got)
	}
}

func TestSplitNotificationAcrossTakeover(t *testing.T) {
	con, host := notificationConsole(t)
	var screen ptychild.Screen
	envelope := notifyosc.Encode("review ready")
	cut := len(notifyosc.Prefix) + 2
	host.Reset()
	con.onChunk(chunk{id: "c1", batch: observedBatch(&screen, envelope[:cut])})
	if got := host.Written(); bytes.Contains([]byte(got), envelope[:cut]) {
		t.Fatalf("partial canonical envelope reached host: %q", got)
	}

	con.switchTo("c2", false, arrivalOrdinary)
	host.Reset()
	con.onChunk(chunk{id: "c1", batch: observedBatch(&screen, envelope[cut:])})
	if got := []byte(host.Written()); !bytes.Contains(got, envelope) {
		t.Fatalf("completed envelope was not forwarded atomically: %q", got)
	}
}

func TestCrossActorNotificationDeferral(t *testing.T) {
	con, host := notificationConsole(t)
	var active, inactive ptychild.Screen
	envelope := notifyosc.Encode("tests need approval")
	host.Reset()

	con.onChunk(chunk{id: "c1", batch: observedBatch(&active, []byte("\x1b[31"))})
	before := host.Written()
	con.onChunk(chunk{id: "c2", batch: observedBatch(&inactive, envelope)})
	if got := host.Written(); got != before {
		t.Fatalf("inactive notification spliced into partial active sequence: before %q after %q", before, got)
	}

	con.onChunk(chunk{id: "c1", batch: observedBatch(&active, []byte("m"))})
	if got := []byte(host.Written()); !bytes.Contains(got, append([]byte("\x1b[31m"), envelope...)) {
		t.Fatalf("deferred notification did not flush after safe boundary: %q", got)
	}
}

func TestConsoleInactiveNotificationCreatesAttentionAndFocusedDoesNot(t *testing.T) {
	con, _ := notificationConsole(t)
	var screen ptychild.Screen
	inactive := observedBatch(&screen, notifyosc.Encode("review ready"))
	con.onChunk(chunk{id: "c2", batch: inactive, focusedAtDelivery: false})
	con.mu.Lock()
	c2 := con.panes["c2"].thread
	got := attentionTexts(con.attention.Projection(c2))
	con.mu.Unlock()
	if len(got) != 1 || got[0] != "review ready" {
		t.Fatalf("inactive attention = %v", got)
	}

	con.mu.Lock()
	con.attention.Acknowledge(con.attention.Capture(c2))
	con.syncAttentionLocked()
	con.mu.Unlock()
	con.switchTo("c2", false, arrivalOrdinary)
	var focusedScreen ptychild.Screen
	con.onChunk(chunk{id: "c2", batch: observedBatch(&focusedScreen, notifyosc.Encode("already seen")), focusedAtDelivery: true})
	con.mu.Lock()
	got = attentionTexts(con.attention.Projection(c2))
	con.mu.Unlock()
	if len(got) != 0 {
		t.Fatalf("focused notification became unread: %v", got)
	}
}

// Acknowledgement has ONE authority: switchTo, which is the only place that
// knows a landing actually happened. finishOperation keeps just the failure
// half, because a switch that failed never landed and its capture still has to
// be released.
func TestSwitchAttentionAcknowledgesOnLandingAndReleasesOnFailure(t *testing.T) {
	t.Run("a failed switch keeps the notifications", func(t *testing.T) {
		con, _ := notificationConsole(t)
		con.mu.Lock()
		address := con.panes["c2"].thread
		con.attention.Mark(address, "captured")
		capture := con.attention.Capture(address)
		con.attention.Mark(address, "later")
		con.mu.Unlock()

		con.finishOperation(operationCompletion{
			origin: MenuOperationOrigin{Operation: "switch", Attempt: 1, Address: address, AttentionCapture: capture},
			err:    errOperationQueueOverloaded,
		})

		con.mu.Lock()
		got := attentionTexts(con.attention.Projection(address))
		con.mu.Unlock()
		if len(got) != 2 {
			t.Fatalf("failed switch cleared attention: %v", got)
		}
	})

	t.Run("landing clears them", func(t *testing.T) {
		con, _ := notificationConsole(t)
		con.mu.Lock()
		address := con.panes["c2"].thread
		con.attention.Mark(address, "captured")
		con.attention.Mark(address, "later")
		con.mu.Unlock()

		con.switchTo("c2", true, arrivalNotification)

		con.mu.Lock()
		got := attentionTexts(con.attention.Projection(address))
		con.mu.Unlock()
		if len(got) != 0 {
			t.Fatalf("landing retained attention: %v", got)
		}
	})
}

func TestExpectedParkExitDropsOnlyExitedActorAttention(t *testing.T) {
	con, _ := notificationConsole(t)
	con.mu.Lock()
	one := con.panes["c1"].thread
	two := con.panes["c2"].thread
	con.attention.Mark(one, "gone")
	con.attention.Mark(two, "kept")
	con.expectedExits["c1"] = true
	con.mu.Unlock()
	con.onExit(childExit{id: "c1", code: 0})
	if got := con.attention.Projection(one); len(got) != 0 {
		t.Fatalf("exited actor attention remains: %+v", got)
	}
	if got := attentionTexts(con.attention.Projection(two)); len(got) != 1 || got[0] != "kept" {
		t.Fatalf("other actor attention changed: %v", got)
	}
}

func TestAttentionHandlingStartsNoAuxiliaryWork(t *testing.T) {
	con, _ := notificationConsole(t)
	var screen ptychild.Screen
	con.onChunk(chunk{id: "c2", batch: observedBatch(&screen, notifyosc.Encode("ready"))})
	if len(con.refreshRequests) != 0 || len(con.operationQueue.requests) != 0 {
		t.Fatalf("attention started auxiliary work: refresh=%d operations=%d", len(con.refreshRequests), len(con.operationQueue.requests))
	}
}

func TestConsoleMenuAttentionSelectsNewestUnreadWithoutReordering(t *testing.T) {
	con, _ := notificationConsole(t)
	con.mu.Lock()
	one, two := con.panes["c1"].thread, con.panes["c2"].thread
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{
		{Address: one, Name: "one", State: couchcore.ThreadLive},
		{Address: two, Name: "two", State: couchcore.ThreadLive},
	}, one)
	con.menuReady = true
	con.attention.Mark(two, "newest")
	con.syncAttentionLocked()
	con.focus = FocusActor("c1")
	con.mu.Unlock()

	con.onHotkey()
	con.mu.Lock()
	selected := con.menu.CurrentFrame().SelectedAddress
	first, second := con.menu.Inventory[0].Address, con.menu.Inventory[1].Address
	con.mu.Unlock()
	if selected != two {
		t.Fatalf("selected = %+v, want newest unread %+v", selected, two)
	}
	if first != one || second != two {
		t.Fatalf("attention reordered actors: %+v, %+v", first, second)
	}
}

func stringsContains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}

// Rule 2 of every landing, asserted independently of rule 1: arriving on an
// actor clears its bell whatever brought the operator there. A test that only
// checks `previous` passes while the bell stays lit -- and a lit bell on the
// actor the operator is SITTING IN makes NewestActor() name it, so the next
// ctrl-space opens the switcher on the wrong row.
func TestEveryArrivalAcknowledgesTheLandedActor(t *testing.T) {
	for _, how := range []arrival{arrivalOrdinary, arrivalNotification, arrivalPrevious} {
		con, _ := notificationConsole(t)
		con.mu.Lock()
		two := con.panes["c2"].thread
		con.attention.Mark(two, "paged")
		con.mu.Unlock()

		con.switchTo("c2", true, how)

		con.mu.Lock()
		got := attentionTexts(con.attention.Projection(two))
		newest := con.attention.NewestActor()
		con.mu.Unlock()
		if len(got) != 0 {
			t.Fatalf("arrival %d left the landed actor notifying: %v", how, got)
		}
		if newest == two {
			t.Fatalf("arrival %d left NewestActor naming the actor the operator is attached to", how)
		}
	}
}

// The counterpart: a landing that never happens must not clear the bell.
func TestSwitchToAnUnknownActorDoesNotAcknowledge(t *testing.T) {
	con, _ := notificationConsole(t)
	con.mu.Lock()
	two := con.panes["c2"].thread
	con.attention.Mark(two, "paged")
	con.mu.Unlock()

	con.switchTo("no-such-actor", true, arrivalOrdinary)

	con.mu.Lock()
	got := attentionTexts(con.attention.Projection(two))
	con.mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("a switch that did not land cleared attention: %v", got)
	}
}

// The Spec's headline sequence, driven through the console rather than the pure
// model: N1 -> N2 -> manual detour still leaves ctrl+backspace pointing at the
// actor the operator was actually working in.
func TestConsolePreviousSurvivesNotificationHops(t *testing.T) {
	con, _ := notificationConsole(t)
	con.Attach("c3", "three", ptychild.NewFakeChild(nil))
	con.mu.Lock()
	one, two, three := con.panes["c1"].thread, con.panes["c2"].thread, con.panes["c3"].thread
	con.mu.Unlock()

	con.switchTo("c1", true, arrivalOrdinary)     // working in one
	con.switchTo("c2", true, arrivalNotification) // paged to two
	con.switchTo("c3", true, arrivalNotification) // paged to three
	con.switchTo("c2", true, arrivalOrdinary)     // manual detour

	con.mu.Lock()
	previous, ok := con.tracker.Previous()
	con.mu.Unlock()
	if !ok || previous != one {
		t.Fatalf("previous = (%+v, %v), want %+v -- notification hops must not spend the slot",
			previous, ok, one)
	}
	_ = two
	_ = three
}

// ctrl+backspace is routed by the production input path in both encodings, not
// by calling the handler: reducer support is not user reachability.
func TestConsolePreviousKeyRoutesInBothEncodings(t *testing.T) {
	for _, chord := range []string{"\x08", "\x1b[127;5u"} {
		var it Interceptor
		_, hit, _ := it.FeedHit([]byte(chord))
		if hit != HitPrevious {
			t.Fatalf("chord %q decoded as %v, want HitPrevious", chord, hit)
		}
	}
}

// With no previous, and with a previous that is no longer attached, the key
// says so on the status row rather than blanking the screen or doing nothing.
func TestConsolePreviousWithNowhereToGoSaysSo(t *testing.T) {
	t.Run("no previous recorded", func(t *testing.T) {
		con, _ := notificationConsole(t)
		con.onPreviousHotkey()
		if got := con.feed.Latest(); !stringsContains(got, "previous") {
			t.Fatalf("status = %q, want a previous-related notice", got)
		}
	})

	t.Run("previous is no longer attached", func(t *testing.T) {
		con, _ := notificationConsole(t)
		con.mu.Lock()
		one := con.panes["c1"].thread
		con.mu.Unlock()
		con.switchTo("c1", true, arrivalOrdinary)
		con.switchTo("c2", true, arrivalOrdinary)
		con.mu.Lock()
		// The thread stays durable, but its pane goes away -- park, detach, exit.
		delete(con.panes, "c1")
		con.mu.Unlock()

		con.onPreviousHotkey()

		if got := con.feed.Latest(); !stringsContains(got, "no longer attached") {
			t.Fatalf("status = %q, want the not-attached notice", got)
		}
		con.mu.Lock()
		active := con.active
		con.mu.Unlock()
		if active != "c2" {
			t.Fatalf("active = %q, want the operator left where they were", active)
		}
		_ = one
	})
}

// A lifecycle operation's child exit is expected whichever way the race goes.
// The exit channel and the operation-result channel are separate select cases,
// so roughly half the time the completion lands first and clears InFlight --
// leaving consumeExpectedParkExitLocked's InFlight arm nothing to match. Both
// halves must know about both operations, or every other alt+d prints an exit
// notice for a child the operator deliberately stopped.
func TestDeliberateChildExitsAreExpectedInEitherEventOrder(t *testing.T) {
	for _, operation := range []string{"park", "detach"} {
		for _, completionFirst := range []bool{false, true} {
			name := operation
			if completionFirst {
				name += "/completion-first"
			} else {
				name += "/exit-first"
			}
			t.Run(name, func(t *testing.T) {
				con, _ := notificationConsole(t)
				con.mu.Lock()
				address := con.panes["c2"].thread
				origin := MenuOperationOrigin{Operation: operation, Attempt: 1, Address: address}
				con.menu.InFlight = origin
				con.mu.Unlock()

				if completionFirst {
					con.finishOperation(operationCompletion{origin: origin})
					con.mu.Lock()
					con.menu.InFlight = MenuOperationOrigin{} // ReduceMenu clears it
					expected := con.consumeExpectedParkExitLocked("c2", address)
					con.mu.Unlock()
					if !expected {
						t.Fatalf("%s exit after its completion was reported as unexpected", operation)
					}
					return
				}
				con.mu.Lock()
				expected := con.consumeExpectedParkExitLocked("c2", address)
				con.mu.Unlock()
				if !expected {
					t.Fatalf("%s exit before its completion was reported as unexpected", operation)
				}
			})
		}
	}
}
