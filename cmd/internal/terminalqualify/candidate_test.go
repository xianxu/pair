package terminalqualify

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	vt "github.com/charmbracelet/x/vt"
)

func TestCandidateASCIIAndReplyIsolation(t *testing.T) {
	a, err := NewCandidate(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewCandidate(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	for _, test := range []struct {
		c                      *Candidate
		input, cursor, replies string
	}{{a, "AB\x1b[6n", "2,0", "\x1b[1;3R"}, {b, "Z\x1b[6n", "1,0", "\x1b[1;2R"}, {a, "\x1b[6n", "2,0", "\x1b[1;3R"}} {
		got, err := test.c.Execute(context.Background(), []string{test.input}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got["cursor"] != test.cursor || got["replies"] != test.replies {
			t.Fatalf("cursor %q replies %q", got["cursor"], got["replies"])
		}
		if got["cell:9,2"] != " " {
			t.Fatalf("blank %q", got["cell:9,2"])
		}
	}
}

func TestCandidateDimensions(t *testing.T) {
	for _, d := range [][2]int{{0, 1}, {1, 0}, {-1, 5}, {262145, 1}, {513, 512}} {
		if c, err := NewCandidate(d[0], d[1]); err == nil {
			c.Close()
			t.Fatalf("accepted %v", d)
		}
	}
}

func TestCandidateCancellationAndClose(t *testing.T) {
	c, err := NewCandidate(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = c.Execute(ctx, []string{"x"}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Execute(context.Background(), nil, nil); err == nil {
		t.Fatal("execute after close")
	}
}

func TestCandidateReplyOverflow(t *testing.T) {
	c, err := NewCandidate(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.Execute(context.Background(), nil, func(e *vt.Emulator) { e.SendText(strings.Repeat("x", 1024*1024+1)) })
	if err == nil || !strings.Contains(err.Error(), "reply") {
		t.Fatalf("error %v", err)
	}
}

func TestCandidateCloseJoinsActiveExecution(t *testing.T) {
	c, err := NewCandidate(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, err := c.Execute(context.Background(), nil, func(e *vt.Emulator) { close(entered); <-release; e.SendText("reply") })
		done <- err
	}()
	<-entered
	closed := make(chan struct{})
	go func() { c.Close(); close(closed) }()
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not join")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Execute did not join")
	}
}

// stalledReplyTransport never drains a byte; closing it releases its reader and
// the emulator's blocked reply writer, independently exercising cancellation.
type stalledReplyTransport struct {
	entered chan struct{}
	closed  chan struct{}
	pipe    io.Closer
	once    sync.Once
}

func (s *stalledReplyTransport) Read([]byte) (int, error) {
	close(s.entered)
	<-s.closed
	return 0, io.EOF
}
func (s *stalledReplyTransport) Close() error {
	s.once.Do(func() { close(s.closed); _ = s.pipe.Close() })
	return nil
}

func TestCandidateCancellationUnblocksReplyPipe(t *testing.T) {
	e := vt.NewEmulator(10, 3)
	transport := &stalledReplyTransport{entered: make(chan struct{}), closed: make(chan struct{}), pipe: e.InputPipe().(io.Closer)}
	c, err := newCandidate(e, transport, transport)
	if err != nil {
		t.Fatal(err)
	}
	<-transport.entered
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := c.Execute(ctx, nil, func(e *vt.Emulator) { close(started); e.SendText("blocked") })
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("blocked reply writer leaked")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-c.drainDone:
	default:
		t.Fatal("reader not joined")
	}
}

type failedReplyReader struct{}

func (failedReplyReader) Read([]byte) (int, error) { return 0, errors.New("injected read failure") }
func TestCandidateReadFailureUnblocksWriter(t *testing.T) {
	e := vt.NewEmulator(10, 3)
	c, err := newCandidate(e, failedReplyReader{}, e.InputPipe().(io.Closer))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	done := make(chan error, 1)
	go func() { _, err := c.Execute(context.Background(), []string{"\x1b[6n"}, nil); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("missing transport error")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("failed reader left writer blocked")
	}
}

func TestCandidateCopiesScreenAndCallbackObservations(t *testing.T) {
	c, err := NewCandidate(10, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	got, err := c.Execute(context.Background(), []string{"\x1b]2;Title\x07\x1b]7;file:///tmp\x07\x07\x1b[?25l\x1b[6 q\x1b[38;2;18;52;86m\x1b[48;5;123mX"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"cell:0,0": "X", "cell-width:0,0": "1", "fg:0,0": "#123456", "bg:0,0": "ansi:123", "title": "Title", "cwd": "file:///tmp", "bells": "1", "cursor-visible": "false", "history-lines": "0"}
	for key, want := range expected {
		if got[key] != want {
			t.Errorf("%s=%q want %q", key, got[key], want)
		}
	}
	if _, err = c.Execute(context.Background(), []string{"\x1b[HZZ\x1b]2;Changed\x07"}, nil); err != nil {
		t.Fatal(err)
	}
	if got["cell:0,0"] != "X" || got["title"] != "Title" {
		t.Fatal("snapshot changed after next Execute")
	}
}
