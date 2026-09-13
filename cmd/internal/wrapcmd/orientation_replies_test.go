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

// steppedReplyReader acknowledges its second Read only after the first reply
// has been sent to the production input owner's buffered channel.
type steppedReplyReader struct {
	reply  chan []byte
	queued chan struct{}
	closed chan struct{}
	reads  int
}

func (r *steppedReplyReader) Read(buf []byte) (int, error) {
	r.reads++
	if r.reads == 1 {
		select {
		case data := <-r.reply:
			return copy(buf, data), nil
		case <-r.closed:
			return 0, io.EOF
		}
	}
	if r.reads == 2 {
		close(r.queued)
	}
	<-r.closed
	return 0, io.EOF
}
func TestOrientationDueSubmitSurvivesPrioritizedInput(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        orientation.DeliveryPhase
	}{
		{"solicited reply", "\x1b[1;1R", orientation.DeliverySubmitted},
		{"operator text", "operator", orientation.DeliveryCancelled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, states := orientationProxy(t)
			p.orientation.settleAfter = time.Nanosecond
			// No deadline can rescue a lost settle event in this regression.
			p.orientation.deadlineAfter = time.Hour
			reader := &steppedReplyReader{reply: make(chan []byte), queued: make(chan struct{}), closed: make(chan struct{})}
			consumed := make(chan struct{})
			release := make(chan struct{})
			p.orientation.settleReadyHook = func() { close(consumed); <-release }
			out := newDrainBuffer()
			done := make(chan struct{})
			go func() { p.translateStdinFrom(reader, out, time.Millisecond); close(done) }()
			t.Cleanup(func() {
				select {
				case <-release:
				default:
					close(release)
				}
				close(reader.closed)
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("input scheduler did not stop")
				}
			})
			var rolling []byte
			p.handleChunk([]byte("\x1b[6n"+claudeLiveComposerPaint()), &rolling)
			select {
			case <-consumed:
			case <-time.After(time.Second):
				t.Fatal("settle timer not consumed")
			}
			reader.reply <- []byte(tc.input)
			select {
			case <-reader.queued:
			case <-time.After(time.Second):
				t.Fatal("reply not queued")
			}
			close(release)
			select {
			case state := <-states:
				if state.Phase != tc.want {
					t.Fatalf("state %#v want %s", state, tc.want)
				}
			case <-time.After(time.Second):
				t.Fatal("consumed settle event was lost after prioritized input")
			}
			if !waitFor(time.Second, func() bool { return bytes.Contains(out.Bytes(), []byte(tc.input)) }) {
				t.Fatalf("input lost %q", out.Bytes())
			}
			expected := []byte("\x1b[200~read context\x1b[201~" + tc.input)
			if tc.want == orientation.DeliverySubmitted {
				expected = append(expected, '\r')
			}
			if !bytes.Equal(out.Bytes(), expected) {
				t.Fatalf("writes %q want %q", out.Bytes(), expected)
			}
		})
	}
}
