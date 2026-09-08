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
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/probes/zellijprobe"
)

const (
	scorcMark = "SCORC_MARK"
	decrcMark = "DECRC_MARK"
)

// main is two lines on purpose: os.Exit SKIPS DEFERS, so any cleanup registered
// in a function that also exits is cleanup that does not run on the paths that
// matter most. This probe's defers delete the zellij session it created and
// remove its temp layout -- and the likeliest exit of all,
// PROBE-INCONCLUSIVE, is exactly where a leaked session hurts, because the next
// run then finds a stale `cursorsaveslots-<pid>` in the way.
//
// So: every path RETURNS a code, and the defers live in run.
// TestNoProbeExitsPastItsOwnCleanup enforces the shape (pair#199 BR-69).
func main() { os.Exit(run()) }

func run() int {
	// Assets resolve against THIS file, not the cwd, so the probe runs from
	// anywhere -- `go run ./probes/cursorsaveslots` from the repo root included.
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("PROBE-ERROR: cannot locate probe assets")
		return 1
	}
	dir := filepath.Dir(self)

	layout, err := zellijprobe.WriteLayout(dir, map[string]string{
		"PROBE_SH": filepath.Join(dir, "probe.sh"),
	})
	if err != nil {
		fmt.Println("PROBE-ERROR layout:", err)
		return 1
	}
	defer os.Remove(layout)

	env := zellijprobe.Scrub([]string{"ZELLIJ"}, "TERM=xterm-256color")
	session, err := zellijprobe.Start(zellijprobe.Options{
		ConfigFile: filepath.Join(dir, "..", "..", "zellij", "config.kdl"),
		Layout:     layout,
		NamePrefix: "cursorsaveslots",
		Rows:       24, Cols: 80,
		Env: env,
	})
	if err != nil {
		fmt.Println("PROBE-ERROR start:", err)
		return 1
	}
	defer session.Close()

	if !session.WaitUntilListed(15 * time.Second) {
		// PRECONDITION, not a result. Reporting NOT HONORED here would be a
		// verdict about zellij drawn from a session that never existed -- the
		// exact defect class #208 spent fourteen rounds on.
		fmt.Println("PROBE-INCONCLUSIVE: the probe's zellij session never appeared; nothing was measured.")
		fmt.Printf("--- pty tail:\n%s\n", zellijprobe.TailOf(session.Seen.String(), 600))
		return 2
	}
	// The pane script paints, saves, restores and marks; give it time to finish
	// before reading the frame.
	time.Sleep(8 * time.Second)

	fmt.Printf("--- pty saw %d bytes; tail:\n%s\n---\n", session.Seen.Len(),
		zellijprobe.TailOf(session.Seen.String(), 400))

	// terminal_1 explicitly: never "whatever is focused".
	out, err := session.Action(env, "dump-screen", "--pane-id", "terminal_1")
	if err != nil {
		fmt.Println("PROBE-ERROR dump:", err, string(out))
		return 1
	}
	frame := session.Seen.String()
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
		return 2
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
		return 0
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
		return 2
	}
	return 0
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
