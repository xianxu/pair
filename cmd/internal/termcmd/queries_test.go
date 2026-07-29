package termcmd

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// Every row of the deny-list is stripped out of a replay.
func TestStripTerminalQueriesRemovesEachRow(t *testing.T) {
	rows := []struct {
		name  string
		query string
	}{
		{"DA1", "\x1b[c"},
		{"DA1 explicit zero", "\x1b[0c"},
		{"DA2", "\x1b[>c"},
		{"XTVERSION", "\x1b[>q"},
		{"kitty flags query", "\x1b[?u"},
		{"DSR cursor position", "\x1b[6n"},
		{"OSC 10 foreground", "\x1b]10;?\x07"},
		{"OSC 11 background", "\x1b]11;?\x07"},
		{"OSC 11 ST terminated", "\x1b]11;?\x1b\\"},
		{"DECRQM 2026", "\x1b[?2026$p"},
		{"DECRQM 2031", "\x1b[?2031$p"},
		{"OSC 4 colour", "\x1b]4;12;?\x07"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			in := "before" + row.query + "after"
			if got := string(stripTerminalQueries([]byte(in))); got != "beforeafter" {
				t.Fatalf("strip(%q) = %q, want %q", in, got, "beforeafter")
			}
		})
	}
}

// The backstop for the deny-list: for every final byte the table matches, a
// legitimate sequence sharing that final must SURVIVE. A greedy rule here is
// silent — it breaks mouse mode, key encoding, or the cursor shape with nothing
// failing.
func TestStripTerminalQueriesPreservesLegitimateSequences(t *testing.T) {
	keep := []struct {
		name string
		seq  string
		why  string
	}{
		{"DECSET SGR mouse", "\x1b[?1006h", "shares \\x1b[? with DECRQM; updateMouseMode parses it"},
		{"DECSET button mouse", "\x1b[?1002h", "same"},
		{"DECSET bracketed paste", "\x1b[?2004h", "same"},
		{"DECRST mouse off", "\x1b[?1006l", "same, reset form"},
		{"kitty flags push", "\x1b[>1u", "shares the u final with the \\x1b[?u query"},
		{"kitty flags pop", "\x1b[<u", "same; stripping it drops the app to legacy keys"},
		{"kitty key chord", "\x1b[110;3u", "Alt+n as a live chord, u final"},
		{"DECSCUSR cursor shape", "\x1b[5 q", "nvim emits per mode change; q final like XTVERSION"},
		{"SGR reset", "\x1b[0m", "ordinary styling"},
		{"OSC 0 title", "\x1b]0;title\x07", "non-query OSC"},
		{"DA1 reply", "\x1b[?62;4;52c", "shell echo puts replies in the buffer; c final"},
		{"DECRPM report", "\x1b[?2026;2$y", "reply terminates $y, not $p"},
		{"kitty flags reply", "\x1b[?0u", "reply, not the \\x1b[?u query literal"},
		{"DSR cursor report", "\x1b[24;1R", "the reply to 6n"},
	}
	for _, k := range keep {
		t.Run(k.name, func(t *testing.T) {
			in := "before" + k.seq + "after"
			if got := string(stripTerminalQueries([]byte(in))); got != in {
				t.Fatalf("strip(%q) = %q, want unchanged (%s)", in, got, k.why)
			}
		})
	}
}

// The stored buffer is re-sliced to the last 128 KiB, so it can begin or end
// mid-sequence. Never drop-to-end: that would swallow the visible screen.
func TestStripTerminalQueriesHandlesTruncatedSequences(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"unterminated tail", "visible text\x1b[?100"},
		{"bare escape at end", "visible text\x1b"},
		{"unterminated OSC tail", "visible text\x1b]11;?"},
		{"buffer starts mid-sequence", "6;4;52c\x1b[?1006hvisible"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(stripTerminalQueries([]byte(tt.in))); got != tt.in {
				t.Fatalf("strip(%q) = %q, want unchanged", tt.in, got)
			}
		})
	}
}

// A redraw over a query-bearing buffer emits no query bytes — the whole point.
func TestRedrawTabEmitsNoQueries(t *testing.T) {
	var out bytes.Buffer
	m := &terminalMux{stdout: &out}
	m.redrawTab([]byte("prompt $ \x1b[c\x1b[?2026$p\x1b[?1006h done"))
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

// The accepted residual, pinned so it reads as a known boundary rather than a
// latent bug: a query in flight from tab A when the user switches to B still
// delivers its reply to B. Closing it needs outstanding-query state this issue
// deliberately does not build.
func TestReplyGoesToActiveTabNotTheQueryingTab(t *testing.T) {
	mux := &fakeMux{}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[?62;4;52c")}}, mux, &fakeRuntime{}, io.Discard)
	if got := strings.Join(mux.ops, ","); got != "write:\x1b[?62;4;52c" {
		t.Fatalf("mux ops = %q — the reply is delivered to whichever tab is active", got)
	}
}

// The scan reads a snapshot taken under m.mu, so a redraw concurrent with
// appends is race-free. The test writer needs its own mutex: m.stdout is a bare
// *bytes.Buffer here and is written from both goroutines — in production it is
// an *os.File, where that is interleaving, not a data race. Do NOT "fix" this by
// locking stdout in production.
func TestRedrawSnapshotIsRaceFree(t *testing.T) {
	m := &terminalMux{stdout: &lockedWriter{}}
	tab := &terminalTab{id: 1}
	m.tabs = append(m.tabs, tab)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			m.appendBuffer(1, []byte("output\x1b[c"))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			m.mu.Lock()
			snapshot := bufferSnapshotLocked(tab)
			m.mu.Unlock()
			m.redrawTab(snapshot)
		}
	}()
	wg.Wait()
}

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
