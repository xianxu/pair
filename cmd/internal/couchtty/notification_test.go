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
	con.switchTo("c2", false)
	host.Reset()

	// Delivery happened while c1 was inactive, but processing happens after
	// c1 becomes active. Processing-time focus owns ordinary output.
	queued := chunk{id: "c1", batch: ptychild.OutputBatch{
		Raw: []byte("queued"), Parts: []ptychild.OutputPart{{Bytes: []byte("queued")}},
		RingEnd: 6, ReplaySafeEnd: 6,
	}}
	con.switchTo("c1", false)
	host.Reset()
	con.onChunk(queued)
	if got := host.Written(); !stringsContains(got, "queued") {
		t.Fatalf("newly focused actor output was lost: %q", got)
	}

	host.Reset()
	con.switchTo("c2", false)
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

	con.switchTo("c2", false)
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
	con.switchTo("c2", false)
	var focusedScreen ptychild.Screen
	con.onChunk(chunk{id: "c2", batch: observedBatch(&focusedScreen, notifyosc.Encode("already seen")), focusedAtDelivery: true})
	con.mu.Lock()
	got = attentionTexts(con.attention.Projection(c2))
	con.mu.Unlock()
	if len(got) != 0 {
		t.Fatalf("focused notification became unread: %v", got)
	}
}

func TestCanonicalWrapperNotificationProjectsToStatusAndSwitcher(t *testing.T) {
	con, host := notificationConsole(t)
	var screen ptychild.Screen
	host.Reset()
	con.onChunk(chunk{
		id: "c2", batch: observedBatch(&screen, notifyosc.Encode("agent stopped working")),
		focusedAtDelivery: false,
	})

	con.mu.Lock()
	c2 := con.panes["c2"].thread
	switcherMessages := attentionTexts(con.menu.Attention[c2])
	con.mu.Unlock()
	if len(switcherMessages) != 1 || switcherMessages[0] != "agent stopped working" {
		t.Fatalf("switcher projection = %v", switcherMessages)
	}

	wantStatus := RenderStatusRow(80, StatusModel{Actors: []StatusActor{
		{Label: "one", Active: true},
		{Label: "two", Bell: true},
	}})
	if got := host.Written(); !stringsContains(got, wantStatus) {
		t.Fatalf("status projection lacks pending actor chip: got %q want fragment %q", got, wantStatus)
	}
}

func TestSwitchAttentionAcknowledgesCapturedMessagesOnlyOnSuccess(t *testing.T) {
	con, _ := notificationConsole(t)
	con.mu.Lock()
	address := con.panes["c2"].thread
	con.attention.Mark(address, "captured")
	capture := con.attention.Capture(address)
	con.attention.Mark(address, "later")
	con.mu.Unlock()

	origin := MenuOperationOrigin{Operation: "switch", Attempt: 1, Address: address, AttentionCapture: capture}
	con.finishOperation(operationCompletion{origin: origin, err: errOperationQueueOverloaded})
	con.mu.Lock()
	got := attentionTexts(con.attention.Projection(address))
	retry := con.attention.Capture(address)
	con.mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("failed switch cleared attention: %v", got)
	}

	origin.AttentionCapture = retry
	con.finishOperation(operationCompletion{origin: origin})
	con.mu.Lock()
	got = attentionTexts(con.attention.Projection(address))
	con.mu.Unlock()
	if len(got) != 0 {
		t.Fatalf("successful switch retained captured attention: %v", got)
	}
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
