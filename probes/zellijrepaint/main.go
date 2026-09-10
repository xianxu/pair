// Conformance probe for pair#209: does zellij repaint a pane from its OWN
// buffer when the host pty is resized (SIGWINCH)?
//
// #209's fix for the dominant failure mode rests entirely on this. When a
// child's last full frame has aged out of the 128 KiB replay ring, no amount of
// byte replay can reconstruct the screen — so the switch asks the child to
// repaint instead. For couch, that child is zellij. If zellij does NOT repaint
// on resize, the fix does not work and the issue needs a different mechanism.
// That is an assumption load-bearing enough to measure rather than assert,
// especially here: #213 established that this zellij version's documented
// behaviour and its actual behaviour diverge.
//
// Method: start a throwaway zellij session in a pty. The pane prints a marker
// ONCE and goes quiet. Drain what the pty has seen, then resize the pty and
// watch. The child emits nothing more, so if the marker reappears on the pty it
// came from zellij re-rendering its own buffer — which is exactly the property
// the fix depends on.
//
//	go run ./probes/zellijrepaint
//
// Result on zellij 0.44.3 / macOS: see the verdict this prints.
//
// Gotchas inherited from probes/zellijscrollregion, and real: zellij refuses to
// nest, so ZELLIJ* must be scrubbed from the child's environment; and a session
// that never appears is a PRECONDITION failure, never a verdict (pair#208).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xianxu/pair/probes/zellijprobe"
	"golang.org/x/sys/unix"
)

const marker = "ZELLIJ_REPAINT_MARKER"

// main is two lines on purpose: os.Exit SKIPS DEFERS, and the likeliest exit
// here is PROBE-INCONCLUSIVE, which is exactly where a leaked session would get
// in the next run's way.
func main() { os.Exit(run()) }

func resize(f *os.File, rows, cols uint16) error {
	return unix.IoctlSetWinsize(int(f.Fd()), unix.TIOCSWINSZ,
		&unix.Winsize{Row: rows, Col: cols})
}

func run() int {
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
		NamePrefix: "zellijrepaint",
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
	time.Sleep(4 * time.Second) // let the first full frame land

	before := session.Seen.String()
	if !strings.Contains(before, marker) {
		fmt.Println("PROBE-INCONCLUSIVE: the marker never reached the pty at all, so a repaint cannot be distinguished from a first paint.")
		fmt.Printf("--- pty tail:\n%s\n", zellijprobe.TailOf(before, 600))
		return 2
	}
	mark := len(before)

	// SIGWINCH: rows only, exactly as the fix nudges (a column change would
	// reflow wrapped lines rather than repaint).
	if err := resize(session.PTY, 23, 80); err != nil {
		fmt.Println("PROBE-ERROR resize:", err)
		return 1
	}
	time.Sleep(1500 * time.Millisecond)
	if err := resize(session.PTY, 24, 80); err != nil {
		fmt.Println("PROBE-ERROR restore:", err)
		return 1
	}
	time.Sleep(2500 * time.Millisecond)

	after := session.Seen.String()
	if len(after) <= mark {
		fmt.Println("VERDICT: NOT REPAINTED — the resize produced no output at all.")
		fmt.Println("  #209's repaint request cannot work against this zellij; the fix needs another mechanism.")
		return 1
	}
	produced := after[mark:]
	fmt.Printf("--- resize produced %d bytes; tail:\n%s\n---\n", len(produced), zellijprobe.TailOf(produced, 400))

	if strings.Contains(produced, marker) {
		fmt.Println("VERDICT: REPAINTED — the marker returned to the pty after a resize, and the child emitted nothing.")
		fmt.Println("  zellij re-rendered the pane from its own buffer, which is what #209's repaint request depends on.")
		return 0
	}
	fmt.Println("VERDICT: PARTIAL — the resize produced output, but not the marker.")
	fmt.Println("  zellij redrew something without restoring content the child is no longer sending;")
	fmt.Println("  #209's fix would then repair the frame but not recover aged-out content.")
	return 1
}
