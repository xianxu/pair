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
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/probes/zellijprobe"
)

const marker = "RESERVED_ROW_MARKER"

// main is two lines on purpose: os.Exit SKIPS DEFERS, so cleanup registered in a
// function that also exits does not run on the paths that matter most -- and the
// likeliest exit here, PROBE-INCONCLUSIVE, is exactly where a leaked
// `zellijscrollregion-<pid>` session gets in the next run's way.
//
// Every path RETURNS a code; the defers live in run.
// TestNoProbeExitsPastItsOwnCleanup enforces the shape (pair#199 BR-69).
func main() { os.Exit(run()) }

func run() int {
	// Assets resolve against THIS file, not the cwd, so the probe runs from
	// anywhere -- `go run ./probes/zellijscrollregion` from the repo root
	// included.
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
		NamePrefix: "zellijscrollregion",
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
	// 200 lines have to actually scroll before the frame answers anything.
	time.Sleep(8 * time.Second)

	fmt.Printf("--- pty saw %d bytes; tail:\n%s\n---\n", session.Seen.Len(),
		zellijprobe.TailOf(session.Seen.String(), 400))

	// terminal_1 explicitly: never "whatever is focused".
	if out, err := session.Action(env, "dump-screen", "--pane-id", "terminal_1"); err != nil {
		fmt.Println("PROBE-ERROR dump:", err, string(out))
		return 1
	}
	// The VERDICT comes from the pty -- what zellij actually rendered to the
	// host terminal -- not from dump-screen, which reports pane content without
	// telling us where zellij placed it. The question is positional: after 200
	// lines scrolled, is the marker still on the pane's bottom row?
	frame := session.Seen.String()
	markerRow, markerOK := lastCursorRowBefore(frame, marker)
	line199Row, line199OK := lastCursorRowBefore(frame, "scroll line 199")
	// tailOf, not frame[len(frame)-3000:]: a bare slice panics whenever the pty
	// produced fewer than 3000 bytes, which is precisely the failed-session
	// case this probe must survive to report. A crash there replaces a
	// diagnosis with a stack trace.
	// A CHECKED precondition, not a printed aside. If the earliest line is
	// still on screen the region never scrolled at all, and "the marker is
	// below the last line" would be true for a reason that proves nothing --
	// the probe would report HONORED without having tested anything.
	scrolled := !strings.Contains(zellijprobe.TailOf(frame, 3000), "scroll line 0\r")

	fmt.Printf("marker last painted at row %d (found=%v)\n", markerRow, markerOK)
	fmt.Printf("last scroll line at row %d (found=%v)\n", line199Row, line199OK)
	fmt.Printf("region actually scrolled (line 0 gone): %v\n", scrolled)

	switch {
	case !scrolled:
		fmt.Println("\nPROBE-INCONCLUSIVE: the region never scrolled, so a marker " +
			"below the last line proves nothing.")
		return 2
	case !markerOK || !line199OK:
		fmt.Println("\nPROBE-INCONCLUSIVE: could not locate both landmarks in the frame.")
		return 2
	case markerRow > line199Row:
		fmt.Printf("\nRESULT: DECSTBM HONORED — 200 lines scrolled in rows 1..%d "+
			"while the reserved row %d held its paint.\n", line199Row, markerRow)
	default:
		fmt.Println("\nRESULT: NOT HONORED — the marker did not stay below the scrolling text.")
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
