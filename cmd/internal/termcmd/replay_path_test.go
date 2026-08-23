package termcmd

// The replay PATH, as opposed to the replay policy: what termcmd still owns
// after #146 moved the query deny-list to cmd/internal/ptychild. These tests
// pin that the mux still routes a repaint through the strip and still leaves
// the live path unfiltered -- the two properties #127 established.

import (
	"bytes"

	"github.com/xianxu/pair/cmd/internal/ptychild"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// A redraw over a query-bearing buffer emits no query bytes -- the whole point.
//
// Driven through the PRODUCTION path (child output -> replaySnapshotLocked ->
// redrawTab) rather than by handing raw bytes to redrawTab, because the
// stripping now lives in ptychild.Child.Replay. Feeding redrawTab directly
// would assert a composition production no longer performs.
func TestRedrawTabEmitsNoQueries(t *testing.T) {
	var out bytes.Buffer
	m := &terminalMux{stdout: &out}
	tab := &terminalTab{
		id:    1,
		child: ptychild.NewFakeChild([]byte("prompt $ \x1b[c\x1b[?2026$p\x1b[?1006h done")),
	}
	m.tabs = append(m.tabs, tab)

	m.redrawTab(replaySnapshotLocked(tab))

	got := out.String()
	for _, q := range []string{"\x1b[c", "\x1b[?2026$p"} {
		if strings.Contains(got, q) {
			t.Fatalf("redraw replayed query %q: %q", q, got)
		}
	}
	if !strings.Contains(got, "\x1b[?1006h") {
		t.Fatalf("redraw dropped DECSET mouse mode: %q", got)
	}
	if !strings.Contains(got, "prompt $ ") || !strings.Contains(got, " done") {
		t.Fatalf("redraw lost visible text: %q", got)
	}
}

// The invariant that justifies having NO reply-filter on the input path: a live
// query still reaches the host verbatim, and its reply still reaches the app.
// A later refactor moving the filter onto the live path would break capability
// negotiation with every other test still green.
func TestLiveQueryPathIsUnfiltered(t *testing.T) {
	out := &lockedWriter{} // written from copyActiveOutput's goroutine; see below
	m := newTerminalMux("sh", nil, out, io.Discard, &fakeRuntime{})
	m.tabs = append(m.tabs, &terminalTab{id: 1})
	m.active = 0
	go m.copyActiveOutput()
	defer close(m.done)

	query := []byte("\x1b[c\x1b[?2026$p")
	m.output <- ptyChunk{id: 1, data: query}
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(out.String(), string(query)) {
		if time.Now().After(deadline) {
			t.Fatalf("live query did not reach stdout verbatim: %q", out.String())
		}
		time.Sleep(time.Millisecond)
	}
}

// Replies must reach the app: one arriving while its own tab is active is
// solicited, and dropping it would break capability negotiation. This also shows
// the shape of the ACCEPTED RESIDUAL — delivery follows whichever tab is active,
// so a query in flight across a tab switch lands its reply on the new tab.
// That residual is recorded in ## Log and atlas/architecture.md rather than
// pinned by a test: pinning it honestly needs two real PTY-backed tabs, and a
// single-mux assertion would stay green even if the residual were fixed.
func TestPumpStdinForwardsRepliesToChild(t *testing.T) {
	for _, reply := range []string{"\x1b[?62;4;52c", "\x1b[24;1R", "\x1b[?2026;2$y"} {
		t.Run(reply, func(t *testing.T) {
			mux := &fakeMux{}
			pumpStdin(&splitReader{chunks: [][]byte{[]byte(reply)}}, mux, &fakeRuntime{}, io.Discard)
			if got, want := strings.Join(mux.ops, ","), "write:"+reply; got != want {
				t.Fatalf("mux ops = %q, want %q — replies must reach the app", got, want)
			}
		})
	}
}

// The snapshot/append race MOVED with the ring: ptychild.Child guards both
// under its own mutex, and ptychild's TestChildSnapshotDuringPumpIsRaceFree
// drives it against a real child under -race. Asserting it here would now be
// testing termcmd's call into a lock it does not own.

type lockedWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *lockedWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// csiEnd is LENIENT and INTRODUCER-INDEPENDENT on purpose, and rename_input.go
// depends on both properties: malformedEscapeSize routes SS3 (`\x1bO…`) through it
// and feeds the result straight into `input = input[size:]`. A stricter framing
// would consume the whole buffer (swallowing the next keystrokes mid-rename), and a
// buf[1] dispatch would frame "\x1bOX" as a two-byte escape and leak the X into the
// tab name. Pinned BEFORE #128's extraction so a regression is caught, not argued.
func TestCsiEndLenientFramingIsPinned(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"plain CSI", "\x1b[31m", 5},
		{"unterminated", "\x1b[31", -1},
		{"out-of-range param byte still frames", "\x1b[\x00A", 4},
		{"private-mode query", "\x1b[?1006h", 8},
		{"SS3", "\x1bOX", 3},
		{"SS3 with @ final", "\x1bO@", 3},
	}
	for _, c := range cases {
		if got := csiEnd([]byte(c.in)); got != c.want {
			t.Errorf("%s: csiEnd(%q) = %d, want %d", c.name, c.in, got, c.want)
		}
	}
}

// The rename decoder consumes malformedEscapeSize bytes per iteration
// (rename_input.go:117-120), so a 0 would spin forever on malformed input.
func TestMalformedEscapeSizeNeverReturnsZeroOnNonEmptyInput(t *testing.T) {
	for _, in := range []string{"\x1b[", "\x1b[\x00A", "\x1bZ", "\x1b", "\x1b[31m", "\x1bOX"} {
		if got := malformedEscapeSize([]byte(in)); got <= 0 {
			t.Errorf("malformedEscapeSize(%q) = %d — the decoder loop would not advance", in, got)
		}
	}
}

// recordingHost notes when it is closed, so teardown's ordering is assertable
// without a terminal.
type recordingHost struct {
	record  func(string)
	resized chan struct{}
}

func (h *recordingHost) Write(p []byte) (int, error) { return len(p), nil }
func (h *recordingHost) Size() (ptychild.Size, error) {
	return ptychild.Size{Rows: 24, Cols: 80}, nil
}
func (h *recordingHost) MakeRaw() (func() error, error) {
	return func() error { return nil }, nil
}
func (h *recordingHost) Resized() <-chan struct{} { return h.resized }
func (h *recordingHost) Close() error             { h.record("host.Close"); return nil }

// The resize watcher must stop BEFORE any child pty closes, or a SIGWINCH
// during teardown resizes a pty that is being closed (BR-2). This pins the
// ordering; as two defers it depended on LIFO registration and reverting it was
// invisible to every test in the tree.
func TestTeardownStopsTheWatcherBeforeClosingChildren(t *testing.T) {
	var order []string
	host := &recordingHost{
		record:  func(s string) { order = append(order, s) },
		resized: make(chan struct{}),
	}

	teardown(host, func() { order = append(order, "closeChildren") })

	if len(order) != 2 || order[0] != "host.Close" || order[1] != "closeChildren" {
		t.Fatalf("teardown order = %v, want [host.Close closeChildren]", order)
	}
}
