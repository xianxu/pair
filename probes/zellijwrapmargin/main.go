// Conformance probe for pair#223: where does zellij put the cursor when a line
// WRAPS at the bottom margin of a DECSTBM scroll region?
//
// `pair term` reserves its tab strip with ESC[1;N-1r and relies on zellij
// scrolling rows 1..N-1 while row N stays put. probes/zellijscrollregion
// established that zellij honors the region — but only ever printed SHORT
// lines. The operator's live symptom (#223) dies on exactly the one line of an
// `ls -la` that wraps, after which every later line overprints the strip row.
// So the untested case is autowrap at the bottom margin, and this measures it.
//
// Method: a throwaway zellij session whose pane process sets the region, sits on
// its bottom margin, and prints first a short line and then a line longer than
// the pane is wide — asking zellij for the cursor position (DSR, ESC[6n) after
// each. The answers go to a file, so the reading is zellij's own cursor rather
// than an inference from where it rendered text.
//
//	go run ./probes/zellijwrapmargin
//
// Result on zellij 0.44.3 / macOS, 2026-09-10 (pane 22 rows, region 1..21):
//
//	at-bottom-margin      row=21 col=1
//	after-short-line      row=21 col=1    newline scrolls the region: correct
//	after-wrapping-line   row=22 col=11   autowrap lands ON the reserved row
//	after-its-newline     row=22 col=1    and stays there
//	VERDICT: WRAP ESCAPES THE REGION
//
// The TOP-edge alternative (PAIR_PROBE_EDGE=top:<mode>, region 2..N) measured
// the same day, because "put the strip at the top instead" was the obvious
// workaround and deserved a reading rather than an argument:
//
//	top:scroll   ordinary scrolling            strip SURVIVES
//	top:wrap     one wrap at the last row      strip SCROLLED AWAY — the wrap
//	                                           scrolls the whole screen
//	top:plain    ESC[H, no origin mode         draws ON the strip
//	top:decom    origin mode, trusted to DECRC draws ON the strip — zellij's
//	                                           DECRC does not restore DECOM
//	top:decom2   origin mode re-asserted       row 2; strip SURVIVES
//	top:childregion  child sets ITS OWN region  strip OVERWRITTEN — DECSTBM
//	                                           params are absolute even under
//	                                           DECOM, so the child's rows 1..10
//	                                           include the strip
//	top:nvim     real nvim, scrolling          strip OVERWRITTEN by line "94"
//
// Which is why the strip did NOT move to the top: it would trade a shell-output
// bug for corruption under every full-screen app, unless pair rewrote the
// child's DECSTBM parameters in flight. The bottom edge keeps the child's rows
// and the pane's rows in one coordinate system; that asymmetry is the one
// hostty/reserve.go recorded when it refused EdgeTop.
//
// So the wrap ignores the region at BOTH edges; only the failure differs. And
// zellij reports CPR relative to the region's top, so row=0 is absolute row 1.
//
// FIXED UPSTREAM in zellij v0.45.0 — zellij-org/zellij#5357, "fix(grid): scroll
// the region when a line wraps at its bottom margin". v0.44.3's line_wrap()
// never consulted the scroll region: on the last screen row it scrolled the
// whole screen (the top-edge reading above), anywhere else it moved the cursor
// down a row (the bottom-edge one). Re-measured against the official v0.45.1
// release, 2026-09-10:
//
//	after-wrapping-line   row=21 col=11   (region 1..21): stays on the margin
//	VERDICT: WRAP SCROLLS THE REGION      and top:wrap — TOP STRIP SURVIVED
//
// (Region 1..21 with `pane_frame_style "full"` in pair's config, which frames
// the pane as 0.44.3 did; the first 0.45.1 reading was 1..23, taken before
// that key existed, under 0.45's frameless "titles" default.)
//
// Windows Terminal shipped the same class of bug (microsoft/terminal#19016).
//
// A session that never appears is a PRECONDITION failure, never a verdict
// (pair#208).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xianxu/pair/probes/zellijprobe"
)

func main() { os.Exit(run()) }

func run() int {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("PROBE-ERROR: cannot locate probe assets")
		return 1
	}
	dir := filepath.Dir(self)

	outFile, err := os.CreateTemp("", "zellijwrapmargin-*.txt")
	if err != nil {
		fmt.Println("PROBE-ERROR temp:", err)
		return 1
	}
	outPath := outFile.Name()
	_ = outFile.Close()
	defer os.Remove(outPath)

	// PAIR_PROBE_EDGE=top:<mode> measures the TOP-edge alternative (region
	// 2..N) instead. Modes: scroll (ordinary scrolling), wrap (one wrap at the
	// last row), plain (home, no origin mode), decom (origin mode trusted to
	// DECRC), decom2 (origin mode re-asserted), childregion (a child's own
	// DECSTBM) and nvim (real nvim scrolling). Each reads whether row 1 is still
	// the strip.
	script, mode := "probe.sh", ""
	if edge := os.Getenv("PAIR_PROBE_EDGE"); strings.HasPrefix(edge, "top:") {
		script, mode = "probe_top.sh", strings.TrimPrefix(edge, "top:")
	}
	layout, err := zellijprobe.WriteLayout(dir, map[string]string{
		"PROBE_SH":   filepath.Join(dir, script),
		"PROBE_OUT":  outPath,
		"PROBE_MODE": mode,
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
		NamePrefix: "zellijwrapmargin",
		Rows:       24, Cols: 80,
		Env: env,
	})
	if err != nil {
		fmt.Println("PROBE-ERROR start:", err)
		return 1
	}
	defer session.Close()

	if !session.WaitUntilListed(15 * time.Second) {
		fmt.Println("PROBE-INCONCLUSIVE: the probe's zellij session never appeared; nothing was measured.")
		fmt.Printf("--- pty tail:\n%s\n", zellijprobe.TailOf(session.Seen.String(), 600))
		return 2
	}

	deadline := time.Now().Add(15 * time.Second)
	var got string
	for time.Now().Before(deadline) {
		b, _ := os.ReadFile(outPath)
		got = string(b)
		if strings.Contains(got, "DONE") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Print(got)
	if !strings.Contains(got, "DONE") {
		fmt.Println("PROBE-INCONCLUSIVE: the pane process never finished its readings.")
		return 2
	}
	if mode == "nvim" {
		// Drive nvim the way a person reads a file: half-pages down, a jump,
		// half-pages back up. Each is a scroll nvim may implement with its own
		// DECSTBM region.
		time.Sleep(2 * time.Second)
		for _, keys := range [][]string{
			{"write", "4"}, {"write", "4"}, {"write-chars", "120G"},
			{"write", "21"}, {"write", "21"}, {"write-chars", "jjjjjjjjjj"},
		} {
			_, _ = session.Action(env, keys...)
			time.Sleep(300 * time.Millisecond)
		}
		time.Sleep(time.Second)
	}
	if mode != "" {
		// Content, not position, is the question for the top edge: is row 1
		// still the strip? dump-screen's first line IS the viewport's row 1.
		b, err := session.Action(env, "dump-screen")
		if err != nil || len(b) == 0 {
			fmt.Println("PROBE-INCONCLUSIVE: dump-screen returned nothing:", err)
			return 2
		}
		lines := strings.Split(string(b), "\n")
		fmt.Printf("row 1: |%s|\n", strings.TrimRight(lines[0], " "))
		if len(lines) > 1 {
			fmt.Printf("row 2: |%s|\n", strings.TrimRight(lines[1], " "))
		}
		if strings.TrimSpace(lines[0]) == "TOP_STRIP" {
			fmt.Println("VERDICT: TOP STRIP SURVIVED")
		} else {
			fmt.Println("VERDICT: TOP STRIP OVERWRITTEN")
		}
		return 0
	}
	// A measurement, not an assertion: it reports what this zellij does and
	// exits 0 for either VERDICT, so test-smoke records the answer without going
	// red on a bug that lives upstream. But a reading it could not PARSE is not
	// an answer, and exits 2 — the first version fell through to the "fixed"
	// verdict on an empty CPR reply or a failed tput, which is the one lie a
	// regression check must never tell (pair#208; #223 BR-1).
	switch verdict(got) {
	case wrapEscapes:
		fmt.Println("VERDICT: WRAP ESCAPES THE REGION — autowrap at the bottom margin puts the")
		fmt.Println("  cursor on the reserved row instead of scrolling. Every later line then")
		fmt.Println("  overprints the strip (#223). A NEWLINE at the same spot scrolls correctly.")
	case wrapScrolls:
		fmt.Println("VERDICT: WRAP SCROLLS THE REGION — the #223 mechanism is not present in this zellij.")
	default:
		fmt.Println("PROBE-INCONCLUSIVE: the readings did not parse into a pane height and a")
		fmt.Println("  post-wrap row ON or BELOW the region's bottom margin; nothing was measured.")
		return 2
	}
	return 0
}

type wrapVerdict int

const (
	inconclusive wrapVerdict = iota
	wrapScrolls
	wrapEscapes
)

// verdict reads the pane height and the post-wrap row out of the pane
// process's report. It returns a verdict ONLY when both parsed and the row
// is one the setup can produce: the cursor started on the region's bottom
// margin, so after a wrap it is either still there (scrolled) or below it
// (escaped). Anything else — a missing line, a zero, a row above the margin
// — means the setup failed, and that is not a finding about zellij.
func verdict(readings string) wrapVerdict {
	var rows, wrapRow int
	var haveRows, haveWrap bool
	for _, line := range strings.Split(readings, "\n") {
		if n, err := fmt.Sscanf(line, "pane rows=%d", &rows); n == 1 && err == nil {
			haveRows = true
		}
		if rest, ok := strings.CutPrefix(line, "after-wrapping-line "); ok {
			if n, err := fmt.Sscanf(rest, "row=%d", &wrapRow); n == 1 && err == nil {
				haveWrap = true
			}
		}
	}
	bottom := rows - 1
	switch {
	case !haveRows || !haveWrap || bottom < 1:
		return inconclusive
	case wrapRow == bottom:
		return wrapScrolls
	case wrapRow == rows:
		return wrapEscapes
	default:
		return inconclusive
	}
}
