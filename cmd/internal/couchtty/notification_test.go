package couchtty

import (
	"bytes"
	"testing"

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

func stringsContains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
