package wrapcmd

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func tailModel(t *testing.T, width, height int, paint string) *terminalModel {
	t.Helper()
	m, err := newTerminalModel(width, height)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	if err := m.Feed([]byte(paint)); err != nil {
		t.Fatal(err)
	}
	return m
}

func promptLine(t *testing.T, lines []string) string {
	t.Helper()
	for _, line := range lines {
		if strings.HasPrefix(line, "❯") {
			return line
		}
	}
	t.Fatalf("no prompt row in tail:\n%s", strings.Join(lines, "\n"))
	return ""
}

// The ghost suggestion and a typed draft read alike once styling is
// stripped (the 10-10 misread); the tail keeps them apart.
func TestTailGhostSuggestionAndTypedDraftDiffer(t *testing.T) {
	ghost, err := os.ReadFile("testdata/tty/claude/2.1.237/composer.raw")
	if err != nil {
		t.Fatal(err)
	}
	m := tailModel(t, 120, 38, string(ghost))
	tail := m.Tail(12)
	if got, want := promptLine(t, tail.Lines), "❯\u00a0‹cursor›‹dim›Try \"refactor Makefile.local\"‹/dim›"; got != want {
		t.Fatalf("ghost prompt row = %q, want %q", got, want)
	}
	if tail.Cursor == nil || tail.Cursor.Col != 3 || tail.Cursor.Hidden {
		t.Fatalf("cursor = %+v", tail.Cursor)
	}

	// The same cells without faint: what an operator's typed draft looks like.
	s := m.Snapshot()
	rows := make([][]uv.Cell, s.Height)
	for y := range rows {
		rows[y] = s.Cells[y*s.Width : (y+1)*s.Width]
		for x := range rows[y] {
			rows[y][x].Style.Attrs &^= uv.AttrFaint
		}
	}
	typed := renderTail(rows, 12, s.Cursor.X, s.Cursor.Y, vt.Cursor{})
	if got, want := promptLine(t, typed.Lines), "❯\u00a0‹cursor›Try \"refactor Makefile.local\""; got != want {
		t.Fatalf("typed prompt row = %q, want %q", got, want)
	}

	draft, err := os.ReadFile("testdata/peer/claude/2.1.286/paste-short.raw")
	if err != nil {
		t.Fatal(err)
	}
	line := promptLine(t, tailModel(t, 120, 38, string(draft)).Tail(12).Lines)
	if strings.Contains(line, "‹dim›") || !strings.Contains(line, "[Couch peer from peer:0") {
		t.Fatalf("captured draft row = %q", line)
	}
}

func TestTailReverseAndHiddenCursor(t *testing.T) {
	tail := tailModel(t, 20, 4, "\x1b[?25lab\x1b[7m \x1b[0m").Tail(10)
	if len(tail.Lines) != 1 || tail.Lines[0] != "ab‹rev› ‹/rev›" {
		t.Fatalf("lines = %q", tail.Lines)
	}
	if got := tail.Cursor.String(); got != "hidden at 1,4" {
		t.Fatalf("cursor = %q", got)
	}
}

func TestTailCursorShape(t *testing.T) {
	tail := tailModel(t, 20, 4, "\x1b[6 qhi").Tail(10)
	if got := tail.Cursor.String(); got != "1,3 bar steady" {
		t.Fatalf("cursor = %q", got)
	}
	if tail.Lines[0] != "hi‹cursor›" {
		t.Fatalf("lines = %q", tail.Lines)
	}
}

func TestTailReadsScrollbackThenScreen(t *testing.T) {
	var paint strings.Builder
	for i := 1; i <= 10; i++ {
		paint.WriteString("l" + string(rune('0'+i%10)) + "\r\n")
	}
	m := tailModel(t, 20, 5, paint.String())
	tail := m.Tail(8)
	want := []string{"l3", "l4", "l5", "l6", "l7", "l8", "l9", "l0", "‹cursor›"}[1:]
	if strings.Join(tail.Lines, "|") != strings.Join(want, "|") {
		t.Fatalf("lines = %q, want %q", tail.Lines, want)
	}
	if tail.Cursor.Row != 8 || tail.Cursor.Col != 1 {
		t.Fatalf("cursor = %+v", tail.Cursor)
	}
	// The alternate screen has no history.
	if err := m.Feed([]byte("\x1b[?1049h\x1b[Halt")); err != nil {
		t.Fatal(err)
	}
	if got := m.Tail(8).Lines; len(got) != 1 || got[0] != "alt‹cursor›" {
		t.Fatalf("alt lines = %q", got)
	}
}

// Screen text that looks like markup cannot pass for it.
func TestTailEscapesLiteralMarkup(t *testing.T) {
	tail := tailModel(t, 40, 4, "a‹dim›b\x1b[2mc\x1b[0m").Tail(10)
	if got, want := tail.Lines[0], "a‹‹dim›b‹dim›c‹/dim›‹cursor›"; got != want {
		t.Fatalf("line = %q, want %q", got, want)
	}
}

func TestTailWideGlyphsAndTrailingBlankRows(t *testing.T) {
	tail := tailModel(t, 20, 6, "你好x\x1b[H").Tail(10)
	if len(tail.Lines) != 1 || tail.Lines[0] != "‹cursor›你好x" {
		t.Fatalf("lines = %q", tail.Lines)
	}
}

// A tail request travels the real endpoint socket to the wrapper's terminal
// model; a wrapper without one says so instead of answering empty.
func TestTailOverEndpointSocket(t *testing.T) {
	// /tmp, not t.TempDir(): a Unix socket path must stay under ~104 bytes,
	// and the package's other socket tests use the same short root.
	namespace, err := os.MkdirTemp("/tmp", "pair-tail-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(namespace) })
	binding := couchmessage.Binding{Slot: "pair:1", Repository: "/fixture/.git", Scope: "fixture-scope", Tag: "t", Session: "s", Nonce: "n", Agent: "claude", Version: "v", PID: os.Getpid(), Start: "start"}
	d := newPeerDelivery(binding, time.Now)
	socket, err := couchmessage.EndpointSocket(namespace, binding)
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(context.Background(), socket, d.handleEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	endpoint := couchmessage.RemoteEndpoint{Namespace: namespace, Binding: binding}

	if _, err := endpoint.Tail(context.Background(), 5); err == nil || !strings.Contains(err.Error(), "no terminal model") {
		t.Fatalf("tail without a model: %v", err)
	}
	m := tailModel(t, 20, 4, "one\r\n\x1b[2mtwo\x1b[0m")
	d.mu.Lock()
	d.tailProbe = m.Tail
	d.mu.Unlock()
	tail, err := endpoint.Tail(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(tail.Lines, "|") != "one|‹dim›two‹/dim›‹cursor›" || tail.Cursor.String() != "2,4 default" {
		t.Fatalf("tail = %q cursor %v", tail.Lines, tail.Cursor)
	}
}
