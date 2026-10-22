package couchtty

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// The child reports its own kernel PTY size on request. Content changes cross
// the old wrap threshold without relying on a child redraw or a polling timer.
func TestCollapseNativeChildHeightDoesNotFollowTabContent(t *testing.T) {
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 12})
	reader, input := io.Pipe()
	c := New(host, reader)
	child, err := ptychild.Start(ptychild.Options{
		Argv: []string{"/bin/sh", "-c", "while IFS= read -r token; do printf '%s:' \"$token\"; /bin/stty size; done"},
		Size: ptychild.Size{Rows: 23, Cols: 12},
		Sink: func(ctx context.Context, batch ptychild.OutputBatch) error { return c.Deliver(ctx, "native", batch) },
	})
	if err != nil {
		t.Fatal(err)
	}
	c.attachThreadActor("native", "one", menuAddress("one"), "/w/one", "one", child)
	c.mu.Lock()
	c.active, c.focus = "native", FocusActor("native")
	c.mu.Unlock()
	done := make(chan struct{})
	go func() { c.Run(); close(done) }()
	t.Cleanup(func() {
		c.Stop()
		_ = input.Close()
		_ = reader.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("native geometry Console did not stop")
		}
		if err := child.Close(); err != nil {
			t.Error(err)
		}
	})
	check := func(token string, cols int) {
		t.Helper()
		waitFor(t, "confirmed one-row geometry", func() bool {
			if !c.chromeConfirmed() {
				return false
			}
			c.mu.Lock()
			defer c.mu.Unlock()
			return int(c.size.Cols) == cols && c.statusRows == 1
		})
		if _, err := child.Write([]byte(token + "\n")); err != nil {
			t.Fatal(err)
		}
		want := []byte(fmt.Sprintf("%s:23 %d", token, cols))
		waitFor(t, "native child reports unchanged row budget", func() bool { return bytes.Contains(child.Snapshot(), want) })
	}
	check("before", 12)
	second := ptychild.NewFakeChild(nil)
	t.Cleanup(func() { _ = second.Close() })
	c.attachThreadActor("second", "second-long", menuAddress("second"), "/w/second", "second-long", second)
	if err := c.runTerminalCommand(context.Background(), func() error {
		c.paintNow()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	check("after-content", 12)
	host.SetSize(ptychild.Size{Rows: 24, Cols: 9})
	check("after-width", 9)
}
