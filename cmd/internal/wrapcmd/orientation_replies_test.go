package wrapcmd

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"io"
	"testing"
	"time"
)

func TestOrientationTerminalRepliesDoNotCancel(t *testing.T) {
	p, states := orientationProxy(t)
	r, w := io.Pipe()
	defer r.Close()
	out := newDrainBuffer()
	done := make(chan struct{})
	go func() { p.translateStdinFrom(r, out, 10*time.Millisecond); close(done) }()
	var rolling []byte
	p.handleChunk([]byte("\x1b[6n\x1b[c"), &rolling)
	for _, chunk := range []string{"\x1b[1;", "1R", "\x1b[?1;2c"} {
		w.Write([]byte(chunk))
	}
	p.handleChunk([]byte(claudeLiveComposerPaint()), &rolling)
	select {
	case state := <-states:
		if state.Phase != orientation.DeliverySubmitted {
			t.Fatal(state)
		}
	case <-time.After(time.Second):
		t.Fatal("no delivery")
	}
	w.Close()
	<-done
	if !bytes.HasPrefix(out.Bytes(), []byte("\x1b[1;1R\x1b[?1;2c")) {
		t.Fatalf("reply bytes changed %q", out.Bytes())
	}
}
func TestOrientationUnsolicitedReplyAndReplyWithInputCancel(t *testing.T) {
	for _, input := range []string{"\x1b[1;1R", "\x1b[?1;2coperator", "\x1b"} {
		t.Run(input, func(t *testing.T) {
			p, states := orientationProxy(t)
			r, w := io.Pipe()
			defer r.Close()
			out := newDrainBuffer()
			done := make(chan struct{})
			go func() { p.translateStdinFrom(r, out, 5*time.Millisecond); close(done) }()
			var rolling []byte
			p.handleChunk([]byte("\x1b[c"), &rolling)
			w.Write([]byte(input))
			p.handleChunk([]byte(claudeLiveComposerPaint()), &rolling)
			select {
			case state := <-states:
				if state.Phase != orientation.DeliveryCancelled {
					t.Fatal(state)
				}
			case <-time.After(time.Second):
				t.Fatal("no cancellation")
			}
			w.Close()
			<-done
			if bytes.Contains(out.Bytes(), []byte("read context")) {
				t.Fatal("pasted after operator input")
			}
		})
	}
}
