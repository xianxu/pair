package couchtty

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// Queue through Deliver so the real focus-at-delivery capture is exercised;
// process explicitly to make a focus change between the two phases deterministic.
func queuePolicyDelivery(t *testing.T, c *Console, id string, b ptychild.OutputBatch) (chunk, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- c.Deliver(ctx, id, b) }()
	select {
	case ch := <-c.chunks:
		return ch, done
	case <-ctx.Done():
		t.Fatal("delivery did not queue")
		return chunk{}, nil
	}
}
func finishPolicyDelivery(t *testing.T, c *Console, ch chunk, done <-chan error) {
	t.Helper()
	c.onChunk(ch)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := c.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestConsoleClipboardPolicyThroughDelivery(t *testing.T) {
	c, host := notificationConsole(t)
	c.switchTo("c1", false, arrivalOrdinary)
	host.Reset()
	selected := []byte("\x1b]52;c;c2VsZWN0ZWQ=\x1b\\")
	hidden := []byte("\x1b]52;c;aGlkZGVu\x1b\\")
	for _, tc := range []struct {
		id   string
		wire []byte
	}{{"c1", selected}, {"c2", hidden}} {
		ch, done := queuePolicyDelivery(t, c, tc.id, observedBatch(c.panes[tc.id].child, tc.wire))
		finishPolicyDelivery(t, c, ch, done)
	}
	if n := bytes.Count([]byte(host.Written()), selected); n != 1 {
		t.Errorf("selected clipboard writes=%d want1", n)
	}
	if n := bytes.Count([]byte(host.Written()), hidden); n != 0 {
		t.Errorf("hidden clipboard writes=%d want0", n)
	}
	beforeSelection := bytes.Count([]byte(host.Written()), hidden)
	c.switchTo("c2", false, arrivalOrdinary)
	c.repaint()
	c.presenter.Flush(context.Background())
	if n := bytes.Count([]byte(host.Written()), hidden); n != beforeSelection {
		t.Errorf("hidden clipboard replayed after selection: before%d after%d", beforeSelection, n)
	}
	if n := bytes.Count([]byte(host.Written()), selected); n != 1 {
		t.Errorf("selected clipboard replayed after selection:%d", n)
	}
	for _, id := range []string{"c1", "c2"} {
		ch, done := queuePolicyDelivery(t, c, id, observedBatch(c.panes[id].child, []byte("\x1b]52;c;?\a")))
		finishPolicyDelivery(t, c, ch, done)
		if got := string(bytes.Join(c.panes[id].child.Writes(), nil)); got != "\x1b]52;c;\x1b\\" {
			t.Errorf("%s local clipboard response=%q", id, got)
		}
	}
	if bytes.Contains([]byte(host.Written()), []byte("\x1b]52;c;?")) {
		t.Fatal("clipboard read leaked to physical terminal")
	}
}

func TestConsoleNotificationUsesCapturedDeliveryFocusOnce(t *testing.T) {
	for _, focused := range []bool{false, true} {
		t.Run(map[bool]string{false: "hidden-at-delivery", true: "focused-at-delivery"}[focused], func(t *testing.T) {
			c, host := notificationConsole(t)
			origin := "c1"
			initial := "c2"
			next := "c1"
			if focused {
				initial, next = "c1", "c2"
			}
			c.switchTo(initial, false, arrivalOrdinary)
			host.Reset()
			wire := notifyosc.Encode("policy-attention")
			ch, done := queuePolicyDelivery(t, c, origin, observedBatch(c.panes[origin].child, wire))
			if ch.focusedAtDelivery != focused {
				t.Fatal("Deliver captured wrong focus")
			}
			c.switchTo(next, false, arrivalOrdinary)
			finishPolicyDelivery(t, c, ch, done)
			envelope := []byte("\x1b]777;notify;pair;policy-attention\x1b\\")
			if n := bytes.Count([]byte(host.Written()), envelope); n != 1 {
				t.Fatalf("outer notification count=%d", n)
			}
			c.mu.Lock()
			attention := attentionTexts(c.attention.Projection(c.panes[origin].thread))
			c.mu.Unlock()
			if focused && len(attention) != 0 || !focused && (len(attention) != 1 || attention[0] != "policy-attention") {
				t.Fatalf("focusedAtDelivery=%v attention=%v", focused, attention)
			}
			// Redelivery of the same typed batch and later paints cannot repeat effects.
			ch.ack = nil
			c.onChunk(ch)
			c.repaint()
			c.presenter.Flush(context.Background())
			if n := bytes.Count([]byte(host.Written()), envelope); n != 1 {
				t.Fatalf("notification replayed=%d", n)
			}
		})
	}
}
