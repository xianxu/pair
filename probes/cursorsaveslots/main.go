package main

// Does this emulator keep DECSC (ESC 7 / ESC 8) and SCOSC (CSI s / CSI u) in
// SEPARATE cursor-save slots?
//
//	go run ./probes/cursorsaveslots        # or: make test-smoke
//
// pair#199's tab strip paints with DECSC/DECRC. So does zsh, to draw its
// right-hand prompt -- terminfo sc/rc are ESC 7 / ESC 8. One shared slot means
// our paint clobbers the child's saved cursor and the child's restore lands
// where WE saved: the operator's "l ends with the cursor in the tab bar", a
// right-prompt drawn on the strip's row, and stray attributes (DECSC saves SGR
// and charset too).
//
// If the slots are SEPARATE, moving the paint to CSI s / CSI u closes the hole
// with two constants. If they ALIAS, only tracking the cursor ourselves does,
// and that is a different size of project -- so this probe decides which.
//
// The verdict reads CURSOR POSITIONS out of the pty frame, not dump-screen:
// dump-screen returns pane content without saying where zellij placed it, which
// cannot answer a positional question (learned in probes/zellijscrollregion).

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

const (
	scorcMark = "SCORC_MARK"
	decrcMark = "DECRC_MARK"
)

func writeLayout(dir string) (string, error) {
	src, err := os.ReadFile(filepath.Join(dir, "layout.kdl"))
	if err != nil {
		return "", err
	}
	body := strings.Replace(string(src), "PROBE_SH", filepath.Join(dir, "probe.sh"), 1)
	f, err := os.CreateTemp("", "cursorsaveslots-*.kdl")
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
	// anywhere -- `go run ./probes/cursorsaveslots` from the repo root
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
	frame := seen.String()
	scorcRow, scorcOK := lastCursorRowBefore(frame, scorcMark)
	decrcRow, decrcOK := lastCursorRowBefore(frame, decrcMark)

	fmt.Printf("SCOSC restore landed at row %d (found=%v) — remembered row 5\n", scorcRow, scorcOK)
	fmt.Printf("DECSC restore landed at row %d (found=%v) — remembered row 10\n", decrcRow, decrcOK)

	if os.Getenv("PROBE_DEBUG") != "" {
		for i, line := range strings.Split(string(out), "\n") {
			if t := strings.TrimSpace(line); t != "" {
				fmt.Printf("DUMP row %2d: %s\n", i+1, t)
			}
		}
		for _, m := range []string{scorcMark, decrcMark} {
			if i := strings.LastIndex(frame, m); i >= 0 {
				lo := i - 120
				if lo < 0 {
					lo = 0
				}
				fmt.Printf("--- %s context: %q\n", m, frame[lo:i+len(m)])
			} else {
				fmt.Printf("--- %s: ABSENT from the frame entirely\n", m)
			}
		}
	}
	if !decrcOK {
		// The control. Without a working DECSC the probe has measured nothing.
		fmt.Println("\nPROBE-INCONCLUSIVE: DECSC/DECRC did not work either, so " +
			"the harness is not measuring what it thinks.")
		os.Exit(2)
	}
	if !scorcOK {
		// This IS the result, not a failure to measure: DECSC restored to the
		// row it remembered, and under the identical harness CSI u did not --
		// its marker is nowhere in the frame at all, so the restore went
		// somewhere unrenderable or the sequence was swallowed.
		fmt.Println("\nRESULT: NO USABLE SECOND SLOT — DECSC/DECRC restored correctly " +
			"while CSI s/u did not restore to the row it was given, and its marker " +
			"does not appear on screen at all.")
		fmt.Println("A cursor restore cannot be built on a sequence whose behaviour " +
			"is not even characterisable. The paint must track the cursor instead.")
		return
	}
	// The rows are pane-relative plus zellij's frame offset, so compare the
	// DIFFERENCE rather than absolute numbers: 5 apart means two slots, 0 apart
	// means one.
	switch decrcRow - scorcRow {
	case 5:
		fmt.Println("\nRESULT: SEPARATE SLOTS — CSI s/u does not disturb the child's ESC 7/8.")
	case 0:
		fmt.Println("\nRESULT: ONE SHARED SLOT — CSI s/u is the same storage; the paint " +
			"must track the cursor instead.")
	default:
		fmt.Printf("\nRESULT: UNEXPECTED (%d rows apart) — neither reading is safe to act on.\n",
			decrcRow-scorcRow)
		os.Exit(2)
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
