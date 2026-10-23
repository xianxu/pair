package cdp

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestWaitJoinsReaderAfterClose(t *testing.T) {
	reader, writer := io.Pipe()
	commands, commandSink := io.Pipe()
	defer commands.Close()
	c := New(reader, commandSink)
	defer reader.Close()
	defer writer.Close()
	defer c.Close()
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Close joined a still-blocked reader: %v", err)
	}
	writer.Close()
	if err := c.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
}
