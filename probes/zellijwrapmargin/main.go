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

	// PAIR_PROBE_EDGE=top:<wrap|plain|decom> measures the TOP-edge alternative
	// (region 2..N) instead: whether a wrap at the last row leaves row 1 alone,
	// and whether homing the cursor lands on it with and without origin mode.
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
	// exits 0 either way, so test-smoke records the answer without going red on
	// a bug that lives upstream. The reading that matters is the wrap row
	// against the region's bottom margin.
	switch {
	case strings.Contains(got, "after-wrapping-line row=") && wrapEscaped(got):
		fmt.Println("VERDICT: WRAP ESCAPES THE REGION — autowrap at the bottom margin puts the")
		fmt.Println("  cursor on the reserved row instead of scrolling. Every later line then")
		fmt.Println("  overprints the strip (#223). A NEWLINE at the same spot scrolls correctly.")
	default:
		fmt.Println("VERDICT: WRAP SCROLLS THE REGION — the #223 mechanism is not present in this zellij.")
	}
	return 0
}

// wrapEscaped reports whether the post-wrap cursor row is past the region.
func wrapEscaped(readings string) bool {
	var rows, bottom, wrapRow int
	for _, line := range strings.Split(readings, "\n") {
		if n, _ := fmt.Sscanf(line, "pane rows=%d", &rows); n == 1 {
			bottom = rows - 1
		}
		if strings.HasPrefix(line, "after-wrapping-line ") {
			fmt.Sscanf(strings.TrimPrefix(line, "after-wrapping-line "), "row=%d", &wrapRow)
		}
	}
	return bottom > 0 && wrapRow > bottom
}
