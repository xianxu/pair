// Timing probe for the repaint nudge: HOW LONG is the screen showing the
// shrink-and-fall before zellij's repaint has fully landed?
//
// zellijrepaint answers WHETHER zellij repaints on SIGWINCH. This answers how
// long that costs, because the answer decides an unrelated design question: the
// operator asked whether couch could run a transition ANIMATION over a switch
// instead of showing the jump. An animation is free only if it fits inside dead
// time the switch already spends; otherwise it is latency added to the
// operator's primary gesture.
//
// Method: same sequence as production (shrink one row, settle, restore) against
// a quiet zellij pane, while a 1 ms sampler records (elapsed, bytes-seen). The
// child prints once and goes silent, so every byte after the mark is zellij
// re-rendering. Reported per run: first byte after the shrink, first byte after
// the restore, and quiescence (no growth for QuietFor).
//
//	go run ./cmd/probes/zellijrepainttiming
//
// Assets are borrowed from cmd/probes/zellijrepaint rather than copied — one
// layout, one probe.sh, so a fix to either reaches both.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/probes/zellijprobe"
	"golang.org/x/sys/unix"
)

const (
	marker   = "ZELLIJ_REPAINT_MARKER"
	quietFor = 150 * time.Millisecond // growth-free span that counts as "done"
	watchFor = 2 * time.Second        // sampling window per run
	runs     = 5
)

func main() { os.Exit(run()) }

func resize(f *os.File, rows, cols uint16) error {
	return unix.IoctlSetWinsize(int(f.Fd()), unix.TIOCSWINSZ,
		&unix.Winsize{Row: rows, Col: cols})
}

type sample struct {
	at  time.Duration
	len int
}

func run() int {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("PROBE-ERROR: cannot locate probe assets")
		return 1
	}
	dir := filepath.Dir(self)
	repo := filepath.Join(dir, "..", "..", "..")
	assets := filepath.Join(dir, "..", "zellijrepaint")

	layout, err := zellijprobe.WriteLayout(assets, map[string]string{
		"PROBE_SH": filepath.Join(assets, "probe.sh"),
	})
	if err != nil {
		fmt.Println("PROBE-ERROR layout:", err)
		return 1
	}
	defer os.Remove(layout)

	env := zellijprobe.Scrub([]string{"ZELLIJ"}, "TERM=xterm-256color")
	session, err := zellijprobe.Start(zellijprobe.Options{
		ConfigFile: filepath.Join(repo, "zellij", "config.kdl"),
		Layout:     layout,
		NamePrefix: "repainttiming",
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
		return 2
	}
	time.Sleep(4 * time.Second)
	if !strings.Contains(session.Seen.String(), marker) {
		fmt.Println("PROBE-INCONCLUSIVE: the marker never reached the pty; a repaint cannot be told from a first paint.")
		return 2
	}

	settle := ptychild.RepaintSettle
	fmt.Printf("--- settle %v, %d runs, quiescence = %v with no growth\n", settle, runs, quietFor)
	fmt.Printf("%-4s %12s %12s %12s %10s\n", "run", "1st@shrink", "1st@restore", "quiet", "bytes")

	for r := 1; r <= runs; r++ {
		mark := session.Seen.Len()
		samples := make([]sample, 0, 2048)
		t0 := time.Now()
		done := make(chan struct{})
		go func() {
			for time.Since(t0) < watchFor {
				samples = append(samples, sample{at: time.Since(t0), len: session.Seen.Len()})
				time.Sleep(time.Millisecond)
			}
			close(done)
		}()

		if err := resize(session.PTY, 23, 80); err != nil {
			fmt.Println("PROBE-ERROR resize:", err)
			return 1
		}
		time.Sleep(settle)
		tRestore := time.Since(t0)
		if err := resize(session.PTY, 24, 80); err != nil {
			fmt.Println("PROBE-ERROR restore:", err)
			return 1
		}
		<-done

		var firstShrink, firstRestore, lastGrowth time.Duration = -1, -1, -1
		prev := mark
		for _, s := range samples {
			if s.len > prev {
				if firstShrink < 0 {
					firstShrink = s.at
				}
				if firstRestore < 0 && s.at > tRestore {
					firstRestore = s.at
				}
				lastGrowth = s.at
				prev = s.len
			}
		}
		quiet := lastGrowth
		fmt.Printf("%-4d %12s %12s %12s %10d\n", r,
			ms(firstShrink), ms(firstRestore), ms(quiet), prev-mark)
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("--- 'quiet' is the last byte of the repaint, measured from the shrink: the window in which the")
	fmt.Println("    screen is mid-transition anyway, and therefore the budget an animation could occupy for free.")
	return 0
}

func ms(d time.Duration) string {
	if d < 0 {
		return "-"
	}
	return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
}
