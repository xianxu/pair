package couchtty

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func suppressedCount(c *Console, id string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p := c.panes[id]; p != nil {
		return p.suppressedNotifications
	}
	return 0
}

func TestConsoleSuppressesEveryChildProtocolAtEverySplit(t *testing.T) {
	for _, id := range []string{"c1", "c2", "hub"} {
		for _, payload := range []string{"9;secret", "777;notify;pair;secret", "99;i=secret;secret"} {
			for _, end := range []string{"\a", "\x1b\\"} {
				wire := "\x1b]" + payload + end
				t.Run(fmt.Sprintf("%s/%s/%q", id, payload[:2], end), func(t *testing.T) {
					c, host := notificationConsole(t)
					c.Attach("hub", "hub", ptychild.NewFakeChild(nil))
					c.switchTo("c1", true, arrivalOrdinary)
					for cut := 1; cut < len(wire); cut++ {
						host.Reset()
						for _, raw := range []string{wire[:cut], wire[cut:]} {
							ch := chunk{id: id, batch: observedBatch(c.panes[id].child, []byte(raw))}
							c.onChunk(ch)
							c.onChunk(ch)
						}
						c.repaint()
						c.presenter.Flush(context.Background())
						if got := suppressedCount(c, id); got != uint64(cut) {
							t.Fatalf("split %d count=%d", cut, got)
						}
						if strings.Contains(host.Written(), "secret") || strings.Contains(host.Written(), "\x1b]777;") {
							t.Fatalf("child notification leaked: %q", host.Written())
						}
						if got := c.attention.Projection(c.panes[id].thread); len(got) != 0 {
							t.Fatalf("attention=%v", got)
						}
					}
					before := suppressedCount(c, id)
					c.switchTo(id, false, arrivalOrdinary)
					c.repaint()
					if suppressedCount(c, id) != before {
						t.Fatal("focus inflated count")
					}
				})
			}
		}
	}
}

func TestConsoleSuppressionTraceAndSaturation(t *testing.T) {
	c, _ := notificationConsole(t)
	path := filepath.Join(t.TempDir(), "trace")
	c.SetEventTrace(path, time.Now())
	defer c.events.Close()
	c.panes["c2"].suppressedNotifications = math.MaxUint64 - 1
	batch := observedBatch(c.panes["c2"].child, []byte("\x1b]9;secret-title\a\x1b]99;i=secret-id;secret-body\a"))
	c.onChunk(chunk{id: "c2", batch: batch})
	c.onChunk(chunk{id: "c2", batch: batch})
	if suppressedCount(c, "c2") != math.MaxUint64 {
		t.Fatal("counter wrapped")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	trace := string(raw)
	if strings.Count(trace, "notify-suppressed") != 1 || !strings.Contains(trace, "count=18446744073709551615") || !strings.Contains(trace, c.panes["c2"].child.Endpoint().ID()) || !strings.Contains(trace, "legacy/c2") || strings.Contains(trace, "secret") {
		t.Fatalf("trace=%q", trace)
	}
}

func TestConsoleCountsAcknowledgedPrefixBeforeEffectError(t *testing.T) {
	c, _ := notificationConsole(t)
	batch := observedBatch(c.panes["c1"].child, []byte("\x1b]9;secret\a"))
	batch.Terminal.Effects = append(batch.Terminal.Effects, terminal.Effect{Kind: terminal.TitleEffect, EndpointID: c.panes["c1"].child.Endpoint().ID(), Sequence: 2, Text: "bad\x1b"})
	c.onChunk(chunk{id: "c1", batch: batch})
	if suppressedCount(c, "c1") != 1 {
		t.Fatal("lost acknowledged notification prefix")
	}
}

func TestConsoleNotifyDesktopIsSeparateFromChildPolicy(t *testing.T) {
	c, host := notificationConsole(t)
	host.Reset()
	if err := c.NotifyDesktop(context.Background(), "Supervisor", "ready"); err != nil {
		t.Fatal(err)
	}
	if strings.Count(host.Written(), "\x1b]777;notify;Supervisor;ready\x1b\\") != 1 {
		t.Fatalf("desktop output=%q", host.Written())
	}
	if suppressedCount(c, "c1") != 0 {
		t.Fatal("supervisor counted as child")
	}
	c.Stop()
	if err := c.NotifyDesktop(context.Background(), "late", "notice"); err == nil {
		t.Fatal("notification after stop succeeded")
	}
}

func TestConsoleSuppressionPreservesOtherEffects(t *testing.T) {
	c, host := notificationConsole(t)
	c.switchTo("c1", true, arrivalOrdinary)
	host.Reset()
	wire := "\x1b]9;secret\a\x1b]2;kept-title\a\x1b]52;c;aGVsbG8=\a\x1b]8;;https://example.test\x1b\\link\x1b]8;;\x1b\\\a"
	c.onChunk(chunk{id: "c1", batch: observedBatch(c.panes["c1"].child, []byte(wire))})
	if err := c.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := host.Written()
	for _, want := range []string{"\x1b]2;kept-title\x1b\\", "\x1b]52;c;aGVsbG8=\x1b\\", "https://example.test", "link", "\a"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "secret") || suppressedCount(c, "c1") != 1 {
		t.Fatal("notification not suppressed")
	}
	c.onChunk(chunk{id: "c2", batch: observedBatch(c.panes["c2"].child, []byte("\x1b]777;notify;pair;secret\a\a"))})
	attention := c.attention.Projection(c.panes["c2"].thread)
	if len(attention) != 1 || attention[0].Text != "" {
		t.Fatalf("ordinary hidden BEL attention=%v", attention)
	}
}

func TestConsoleSuppressionCounterDiesWithPane(t *testing.T) {
	c, _ := notificationConsole(t)
	c.onChunk(chunk{id: "c2", batch: observedBatch(c.panes["c2"].child, []byte("\x1b]99;;secret\a"))})
	if suppressedCount(c, "c2") != 1 {
		t.Fatal("missing count")
	}
	old := c.panes["c2"].child
	c.expectedExits.mark("c2")
	c.onExit(childExit{id: "c2"})
	_ = old.Close()
	c.Attach("c2", "two", ptychild.NewFakeChild(nil))
	if suppressedCount(c, "c2") != 0 {
		t.Fatal("replacement inherited counter")
	}
	c.onChunk(chunk{id: "c2", batch: observedBatch(c.panes["c2"].child, []byte("\x1b]9;new\a"))})
	if suppressedCount(c, "c2") != 1 {
		t.Fatal("replacement counter did not restart")
	}
}
