package main

// Feasibility probe for pair#199: does zellij honor DECSTBM (ESC[1;Nr) set by a
// PANE PROCESS?
//
// couch's reserved row works on the HOST terminal, where couch writes straight
// to the tty. The right pane's writes go through zellij's emulator instead, so
// the whole #199 approach rests on zellij honoring a scroll-region request from
// inside a pane. If it does, `pair term` can reserve a row exactly the way
// couch does. If it does not, the strip needs full compositing -- a different
// and much larger project.
//
// Method: start a throwaway zellij session in a pty; the pane process sets a
// scroll region excluding the bottom row, paints that row, then prints 200
// lines to force many scrolls. The VERDICT reads cursor positions out of the
// pty -- what zellij actually rendered -- and asks whether the marker is still
// painted BELOW the last scrolled line.
//
//	go run ./probes/zellijscrollregion     # or: make test-smoke
//
// Result on zellij 0.44.3 / macOS, 2026-09-07:
//
//	marker last painted at row 23
//	last scroll line at row 21
//	RESULT: DECSTBM HONORED
//
// Three things this probe learned the hard way, all of which produced a
// CONFIDENT WRONG ANSWER before being fixed:
//
//  1. zellij refuses to nest, so ZELLIJ* must be scrubbed from the child's
//     environment -- otherwise it prints a session list, exits, and the probe
//     reads the absence of its marker as "not honored".
//  2. `--session` together with `--layout` means ATTACH, not create. The
//     session never appears and the same false negative follows.
//  3. `dump-screen` is the wrong instrument: it returns pane CONTENT without
//     saying where zellij placed it, so it cannot answer a positional question.
//
// Hence the precondition check: no new session means PROBE-INCONCLUSIVE and a
// non-zero exit, never a verdict. A reading whose precondition failed is not a
// result (pair#208's rule).

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

const marker = "RESERVED_ROW_MARKER"

func writeLayout(dir string) (string, error) {
	src, err := os.ReadFile(filepath.Join(dir, "layout.kdl"))
	if err != nil {
		return "", err
	}
	body := strings.Replace(string(src), "PROBE_SH", filepath.Join(dir, "probe.sh"), 1)
	f, err := os.CreateTemp("", "zellijscrollregion-*.kdl")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		return "", err
	}
	return f.Name(), f.Close()
}

func sessionSet() map[string]bool {
	out, _ := exec.Command("zellij", "list-sessions", "--short").Output()
	set := map[string]bool{}
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			set[l] = true
		}
	}
	return set
}

// syncBuffer is an append-only buffer safe across the reader goroutine and the
// verdict. sync.Mutex rather than a channel because the reader never stops and
// nothing needs to observe individual writes.
type syncBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func (b *syncBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buf)
}

func tailOf(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func main() {
	var session string
	// Assets resolve against THIS file, not the cwd, so the probe runs from
	// anywhere -- `go run ./cmd/probes/zellijscrollregion` from the repo root
	// included.
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("PROBE-ERROR: cannot locate probe assets")
		os.Exit(1)
	}
	dir := filepath.Dir(self)
	// NO --session: with a layout, --session means ATTACH to an existing
	// session ("will be added to the session as a new tab"), so naming it up
	// front made zellij try to attach to a session that did not exist. Without
	// it, the layout starts a NEW session; we find its name by diffing the list.
	before := sessionSet()
	// The repo's own config: it sets show_startup_tips false, without which
	// zellij opens a tips plugin pane that takes focus -- and dump-screen then
	// dumps THAT pane, which the first run mistook for the probe's output.
	// The layout carries a PLACEHOLDER for the probe script, rewritten here to
	// an absolute path: zellij resolves a relative command against its own cwd,
	// which is not ours.
	layout, err := writeLayout(dir)
	if err != nil {
		fmt.Println("PROBE-ERROR layout:", err)
		os.Exit(1)
	}
	defer os.Remove(layout)

	cmd := exec.Command("zellij",
		"--config", filepath.Join(dir, "..", "..", "zellij", "config.kdl"),
		"--layout", layout)
	// ZELLIJ* must be scrubbed: this probe runs INSIDE the pair workbench, and
	// zellij refuses to nest -- it printed a session list and exited, which the
	// first version then mistook for a real (failing) result.
	env := []string{}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "ZELLIJ") {
			continue
		}
		env = append(env, kv)
	}
	cmd.Env = append(env, "TERM=xterm-256color")

	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		fmt.Println("PROBE-ERROR start:", err)
		os.Exit(1)
	}
	defer f.Close()
	// The reader goroutine runs for the probe's whole life and is still running
	// when the verdict reads what it collected -- a pty read only ends when the
	// session dies. An unsynchronised strings.Builder across that boundary is a
	// data race: `go run -race` flags it, and a torn read would corrupt the one
	// artifact the verdict is computed from.
	seen := &syncBuffer{}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				seen.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	defer func() {
		if session != "" {
			exec.Command("zellij", "delete-session", session, "--force").Run()
		}
	}()

	time.Sleep(8 * time.Second)

	after := sessionSet()
	session = ""
	for name := range after {
		if !before[name] {
			session = name
			break
		}
	}
	if session == "" {
		// PRECONDITION, not a result. Reporting NOT HONORED here would be a
		// verdict about zellij drawn from a session that never existed -- the
		// exact defect class #208 spent fourteen rounds on.
		fmt.Println("PROBE-INCONCLUSIVE: no new zellij session appeared; nothing was measured.")
		fmt.Printf("--- pty tail:\n%s\n", tailOf(seen.String(), 600))
		os.Exit(2)
	}
	fmt.Printf("--- pty saw %d bytes; tail:\n%s\n---\n", seen.Len(),
		tailOf(seen.String(), 400))

	// terminal_1 explicitly: never "whatever is focused".
	dump := exec.Command("zellij", "--session", session, "action",
		"dump-screen", "--pane-id", "terminal_1")
	dump.Env = cmd.Env
	out, err := dump.CombinedOutput()
	if err != nil {
		fmt.Println("PROBE-ERROR dump:", err, string(out))
		os.Exit(1)
	}
	_ = out
	// The VERDICT comes from the pty -- what zellij actually rendered to the
	// host terminal -- not from dump-screen, which reports pane content without
	// telling us where zellij placed it. The question is positional: after 200
	// lines scrolled, is the marker still on the pane's bottom row?
	frame := seen.String()
	markerRow, markerOK := lastCursorRowBefore(frame, marker)
	line199Row, line199OK := lastCursorRowBefore(frame, "scroll line 199")
	// tailOf, not frame[len(frame)-3000:]: a bare slice panics whenever the pty
	// produced fewer than 3000 bytes, which is precisely the failed-session
	// case this probe must survive to report. A crash there replaces a
	// diagnosis with a stack trace.
	line000 := strings.Contains(tailOf(frame, 3000), "scroll line 0 ")

	fmt.Printf("marker last painted at row %d (found=%v)\n", markerRow, markerOK)
	fmt.Printf("last scroll line at row %d (found=%v)\n", line199Row, line199OK)
	fmt.Printf("earliest scroll line still on screen: %v\n", line000)

	switch {
	case !markerOK || !line199OK:
		fmt.Println("\nPROBE-INCONCLUSIVE: could not locate both landmarks in the frame.")
		os.Exit(2)
	case markerRow > line199Row:
		fmt.Printf("\nRESULT: DECSTBM HONORED — 200 lines scrolled in rows 1..%d "+
			"while the reserved row %d held its paint.\n", line199Row, markerRow)
	default:
		fmt.Println("\nRESULT: NOT HONORED — the marker did not stay below the scrolling text.")
	}
}

// lastCursorRowBefore finds the row of the most recent CUP (ESC[row;colH) that
// precedes the final occurrence of text.
func lastCursorRowBefore(frame, text string) (int, bool) {
	idx := strings.LastIndex(frame, text)
	if idx < 0 {
		return 0, false
	}
	head := frame[:idx]
	cup := regexp.MustCompile(`\x1b\[(\d+);(\d+)H`)
	all := cup.FindAllStringSubmatch(head, -1)
	if len(all) == 0 {
		return 0, false
	}
	row, err := strconv.Atoi(all[len(all)-1][1])
	if err != nil {
		return 0, false
	}
	return row, true
}
