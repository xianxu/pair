// probes/couchnestedrows answers the one question pair#199 M3 could not close
// by unit test or by a standalone smoke run: do TWO reserved rows compose?
//
// Under couch there are two, and they belong to different terminals:
//
//	couch          reserves the HOST terminal's bottom row  (actor strip)
//	  zellij       gets a pty one row shorter
//	    pair term  reserves its PANE's bottom row            (tab strip)
//	      shell    gets a pane one row shorter
//
// Each Reservation is computed from its own Host.Size(), so they SHOULD
// compose. "Should" is the word that has already cost this milestone four
// defects, and the standalone smoke test cannot see this at all -- it runs
// `pair term` with no outer reservation above it.
//
// Method. The probe is its own outer host: `couchnestedrows outer` reserves the
// bottom row of the pty it was handed and runs zellij in the rest, exactly the
// way couch does (the same hostty.Reservation, since #199 M1 made it shared).
// The parent feeds that pty into a REAL terminal emulator (charmbracelet/x/vt)
// and then reads the screen. That is the instrument that matters: raw bytes
// cannot answer a positional question, and this whole milestone's defects were
// positional.
//
//	go run ./probes/couchnestedrows            # against ./bin/pair
//	go run ./probes/couchnestedrows /path/pair
//
// A reading whose precondition failed is not a result (pair#208's rule): no
// zellij, no built binary, or a session that never came up is
// PROBE-INCONCLUSIVE and a non-zero exit, never a verdict.
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
	"golang.org/x/term"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/probes/zellijprobe"
)

const (
	// The outer row's text. Distinctive because the verdict looks for it on an
	// exact row of a real screen, and a substring of ordinary shell output
	// would make a false positive indistinguishable from a pass.
	outerMarker = "OUTER-COUCH-ROW"

	hostCols = 100
	hostRows = 40

	// What the arithmetic must produce if the two reservations compose:
	// 40 host rows - 1 (couch's) = 39 for zellij; the borderless pane is all 39;
	// 39 - 1 (pair term's) = 38 for the shell.
	wantShellRows = hostRows - 2

	altT     = "\x1bt"
	altR     = "\x1br"
	altLeft  = "\x1b[1;3D"
	altRight = "\x1b[1;3C"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "outer" {
		runOuter(os.Args[2:])
		return
	}
	runProbe()
}

// ---------------------------------------------------------------- outer host

// runOuter is couch's half: reserve the bottom row of the terminal we were
// handed, run zellij in what is left, and keep the row painted.
//
// It repaints on a TICKER rather than on row-dirty, which couch uses. That is
// deliberately harsher than production: a timer paints while the pane's child
// is mid-anything, so if the outer row's paint can disturb the inner strip,
// this maximises the chance of catching it.
func runOuter(args []string) {
	if len(args) < 4 {
		fatalOuter("usage: couchnestedrows outer <config-file> <layout> <session> <data-dir>")
	}
	configFile, layout, session, dataDir := args[0], args[1], args[2], args[3]

	ws, err := pty.GetsizeFull(os.Stdout)
	if err != nil {
		fatalOuter("measure the host terminal: %v", err)
	}
	res, err := hostty.NewReservation(ws.Rows, hostty.EdgeBottom)
	if err != nil {
		fatalOuter("reserve: %v", err)
	}

	// Raw, or the pty's line discipline swallows the parent's keystrokes into
	// a line buffer and echoes them onto the screen the verdict reads.
	if restore, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		defer term.Restore(int(os.Stdin.Fd()), restore)
	}

	// --config, the FILE, not --config-dir: with the directory form zellij
	// still opened its startup-tips plugin, which takes focus and swallows the
	// probe's keystrokes (`show_startup_tips false` lives in that file, and
	// probes/zellijscrollregion pays the same toll for the same reason).
	cmd := exec.Command("zellij",
		"--config", configFile,
		"--new-session-with-layout", layout,
		"--session", session)
	cmd.Env = childEnv(dataDir)
	inner, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: res.ChildRows(), Cols: ws.Cols})
	if err != nil {
		fatalOuter("start zellij: %v", err)
	}

	// ONE writer to the host, the rule #199 M2 established for termcmd and the
	// same rule here: zellij's output and our row paint both go through this
	// channel, so a paint can never land inside an escape sequence of the
	// child's that is still being written.
	writes := make(chan []byte, 256)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for b := range writes {
			_, _ = os.Stdout.Write(b)
		}
	}()

	// Blocking, deliberately. A dropped chunk would corrupt the screen the
	// verdict is computed from, and a corrupt screen reads as a FAILURE -- the
	// one outcome a probe must never manufacture. The parent drains the host
	// pty continuously, so the only way this blocks is a parent that has
	// already stopped caring, and it kills this process on the way out.
	post := func(b []byte) { writes <- b }

	post([]byte(res.ReserveAndPaint(outerMarker)))

	go func() { _, _ = io.Copy(inner, os.Stdin) }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 8192)
		for {
			n, err := inner.Read(buf)
			if n > 0 {
				post(append([]byte(nil), buf[:n]...))
			}
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			post([]byte(res.Release()))
			close(writes)
			wg.Wait()
			_ = cmd.Wait()
			return
		case <-ticker.C:
			post([]byte(res.ReserveAndPaint(outerMarker)))
		}
	}
}

func fatalOuter(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "OUTER-ERROR: "+format+"\n", a...)
	os.Exit(1)
}

// childEnv scrubs the two prefixes that make a nested run lie.
//
// ZELLIJ*: zellij refuses to nest -- with them set it prints a session list and
// exits, and the verdict reads the absence of a strip as a failure
// (probes/zellijscrollregion learned this the hard way).
//
// PAIR_*: this probe runs INSIDE a pair workbench, and `pair term` writes pane
// registry files under PAIR_DATA_DIR keyed by PAIR_TAG. Inheriting them would
// have the probe's throwaway tabs overwrite the operator's live session state.
func childEnv(dataDir string) []string {
	return zellijprobe.Scrub([]string{"ZELLIJ", "PAIR_"},
		"TERM=xterm-256color",
		"SHELL=/bin/sh",
		"PAIR_DATA_DIR="+dataDir,
		"PAIR_TAG=couchnestedrows",
	)
}

// -------------------------------------------------------------------- verdict

// screen is the parent's model of what the operator would see. A real emulator,
// because every question here is "what is on row N", which bytes cannot answer.
type screen struct {
	mu    sync.Mutex
	em    *vt.Emulator
	raw   strings.Builder
	strip oscStripper
}

func newScreen() *screen {
	em := vt.NewEmulator(hostCols, hostRows)
	// Drain the emulator's reply pipe or it wedges mid-write: zellij and the
	// shell both send capability queries, and the emulator answers into its own
	// output.
	go func() { _, _ = io.Copy(io.Discard, em) }()
	return &screen{em: em}
}

func (s *screen) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raw.Write(p)
	// OSC never reaches the model. MEASURED 2026-09-08, and it is the
	// instrument that is wrong, not the terminal: this emulator leaks NON-ASCII
	// bytes out of an OSC payload onto the screen --
	//
	//	vt.NewEmulator(40, 5) <- "AB\x1b]0;terminal 日本語\aCD"  renders "AB語CD"
	//
	// zellij sets the pane title (OSC 0) on every rename keystroke, so with the
	// payload unfiltered the tab strip's row reads as the tail of the TITLE and
	// the probe reports a rendering defect that no real terminal has. An OSC
	// sets the window title and draws nothing, so dropping it is the faithful
	// model of a terminal that parses it correctly.
	if _, err := s.em.Write(s.strip.filter(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// oscStripper drops OSC sequences across chunk boundaries -- a pty read splits
// them wherever it likes, so this cannot be a per-chunk regexp.
type oscStripper struct{ state int }

const (
	oscText = iota
	oscAfterESC
	oscInside
	oscInsideESC
)

func (o *oscStripper) filter(p []byte) []byte {
	out := make([]byte, 0, len(p))
	for _, b := range p {
		switch o.state {
		case oscText:
			if b == 0x1b {
				o.state = oscAfterESC
				continue
			}
			out = append(out, b)
		case oscAfterESC:
			if b == ']' {
				o.state = oscInside
				continue
			}
			o.state = oscText
			out = append(out, 0x1b, b)
		case oscInside:
			switch b {
			case 0x07: // BEL terminator
				o.state = oscText
			case 0x1b:
				o.state = oscInsideESC
			}
		case oscInsideESC:
			// ESC \ is ST; anything else was a malformed OSC, and dropping the
			// rest of it is closer to a real terminal than emitting it as text.
			o.state = oscInside
			if b == 0x5c {
				o.state = oscText
			}
		}
	}
	return out
}

// row reads a 1-based row, right-trimmed.
func (s *screen) row(n int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var b strings.Builder
	for x := 0; x < hostCols; x++ {
		if c := s.em.CellAt(x, n-1); c != nil {
			b.WriteString(c.Content)
		}
	}
	return strings.TrimRight(b.String(), " ")
}

// lastColumn is the 1-based rightmost non-blank CELL on a row -- how far the
// row actually reaches on screen. Cells, not runes: the emulator has already
// resolved wide characters into the two columns they occupy, which is exactly
// the distinction a rune count gets wrong.
func (s *screen) lastColumn(n int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	last := 0
	for x := 0; x < hostCols; x++ {
		c := s.em.CellAt(x, n-1)
		if c != nil && strings.TrimSpace(c.Content) != "" {
			last = x + 1
		}
	}
	return last
}

// rawTail is the last n bytes the host pty produced, for a failure that the
// screen alone does not explain -- an error the child printed and then scrolled
// away, most often.
func (s *screen) rawTail(n int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw := s.raw.String()
	if len(raw) > n {
		raw = raw[len(raw)-n:]
	}
	return fmt.Sprintf("%q", raw)
}

// lastTitle is the most recent OSC 0 title the pane set. The rename editor
// draws its field there -- which is the zellij pane FRAME, and therefore the
// one piece of this feature that M4's borderless=true makes invisible.
func (s *screen) lastTitle() string {
	s.mu.Lock()
	raw := s.raw.String()
	s.mu.Unlock()
	i := strings.LastIndex(raw, "\x1b]0;")
	if i < 0 {
		return ""
	}
	rest := raw[i+4:]
	if j := strings.IndexAny(rest, "\a\x1b"); j >= 0 {
		return rest[:j]
	}
	return rest
}

func (s *screen) all() string {
	var b strings.Builder
	for n := 1; n <= hostRows; n++ {
		fmt.Fprintf(&b, "%2d |%s\n", n, s.row(n))
	}
	return b.String()
}

func runProbe() {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		inconclusive("cannot locate probe assets")
	}
	dir := filepath.Dir(self)
	repo := filepath.Join(dir, "..", "..", "..") // cmd/probes/couchnestedrows -> repo root

	pairBin := filepath.Join(repo, "bin", "pair")
	if len(os.Args) > 1 {
		pairBin = os.Args[1]
	}
	pairBin, err := filepath.Abs(pairBin)
	if err != nil || !executable(pairBin) {
		inconclusive("no built pair at %s (make bin/pair)", pairBin)
	}
	if _, err := exec.LookPath("zellij"); err != nil {
		inconclusive("zellij is not on PATH")
	}
	exe, err := os.Executable()
	if err != nil {
		inconclusive("cannot re-exec self as the outer host: %v", err)
	}

	dataDir, err := os.MkdirTemp("", "couchnestedrows-data-*")
	if err != nil {
		inconclusive("temp data dir: %v", err)
	}
	defer os.RemoveAll(dataDir)

	layout, err := zellijprobe.WriteLayout(dir, map[string]string{"PAIR_BIN": pairBin})
	if err != nil {
		inconclusive("layout: %v", err)
	}
	defer os.Remove(layout)

	session := fmt.Sprintf("couchnestedrows-%d", os.Getpid())
	defer exec.Command("zellij", "delete-session", session, "--force").Run()

	cmd := exec.Command(exe, "outer",
		filepath.Join(repo, "zellij", "config.kdl"), layout, session, dataDir)
	cmd.Env = childEnv(dataDir)
	host, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: hostRows, Cols: hostCols})
	if err != nil {
		inconclusive("start the outer host: %v", err)
	}
	stop := func() { _ = host.Close(); _ = cmd.Process.Kill() }
	defer stop()

	sc := newScreen()
	go func() {
		buf := make([]byte, 8192)
		for {
			n, rerr := host.Read(buf)
			if n > 0 {
				_, _ = sc.Write(buf[:n])
			}
			if rerr != nil {
				return
			}
		}
	}()

	send := func(s string) { _, _ = io.WriteString(host, s) }

	// Readiness, not a fixed sleep: zellij + pair term + a shell is several
	// seconds on a cold cache and a hung one must be reported as inconclusive
	// rather than measured as a failure.
	if !waitFor(20*time.Second, func() bool {
		return strings.Contains(sc.row(hostRows-1), "[terminal 1]")
	}) {
		fmt.Print(sc.all())
		stop()
		inconclusive("the tab strip never appeared; the session did not come up")
	}

	fails := 0
	check := func(name string, pass bool, detail string) {
		if pass {
			fmt.Printf("PASS  %-52s %s\n", name, detail)
			return
		}
		fails++
		fmt.Printf("FAIL  %-52s %s\n", name, detail)
	}

	// 1. The arithmetic, end to end. This is the composition question stated as
	//    a number the shell itself reports: 40 - 1 (couch's row) - 1 (the tab
	//    strip's) = 38. An off-by-one anywhere in the chain lands here.
	send("stty size\r")
	got := waitForRowMatching(sc, 10*time.Second, func(r string) bool {
		return strings.Contains(r, fmt.Sprintf("%d %d", wantShellRows, hostCols))
	})
	check("both reservations come out of the shell's size", got != "",
		fmt.Sprintf("want %q on screen; %s", fmt.Sprintf("%d %d", wantShellRows, hostCols), describe(got)))

	// 2. Neither row eats the other, and the child's scroll reaches neither.
	send("seq 1 400\r")
	time.Sleep(2 * time.Second)
	strip, outer := sc.row(hostRows-1), sc.row(hostRows)
	check("outer row survives a flood in the pane", strings.Contains(outer, outerMarker),
		fmt.Sprintf("row %d = %q", hostRows, outer))
	check("tab strip survives a flood in the pane", strings.Contains(strip, "[terminal 1]"),
		fmt.Sprintf("row %d = %q", hostRows-1, strip))
	check("the flood stayed above both rows",
		!strings.Contains(strip, "400") && !strings.Contains(outer, "400"),
		fmt.Sprintf("rows %d/%d = %q / %q", hostRows-1, hostRows, strip, outer))

	// 3. M3.7(c): a WIDE name and a LONG one, read off a real screen rather
	//    than trusted to the unit test's arithmetic. A wide rune is two columns
	//    and one rune; a renderer that counts runes draws a row that overflows
	//    into the row below -- which here is couch's.
	trace := func(label string) {
		fmt.Printf("      %-22s strip=%-40q title=%q\n", label, sc.row(hostRows-1), sc.lastTitle())
	}
	send(altT)
	time.Sleep(1500 * time.Millisecond)
	trace("after alt+t")

	// M3.7(b), THE STEP THAT WAS TICKED WITHOUT BEING RUN (BR-47). The defer-
	// and-owe gate can only be reached by requesting a paint while the child's
	// stream is MID-SEQUENCE, which means the load generator has to emit
	// ESCAPES. `yes` and `seq` emit none, so both earlier attempts at this step
	// exercised nothing -- the plan's own 2026-09-07 revision records the first
	// of those as a defect, and the flood above repeats it. This one emits an
	// SGR pair per line, and tab switches request a paint against it.
	send("for i in $(seq 1 6000); do printf '\033[31mred\033[0m\n'; done\r")
	time.Sleep(1500 * time.Millisecond)
	for i := 0; i < 8; i++ {
		send(altLeft)
		time.Sleep(250 * time.Millisecond)
		send(altRight)
		time.Sleep(250 * time.Millisecond)
	}
	time.Sleep(2500 * time.Millisecond)
	strip, outer = sc.row(hostRows-1), sc.row(hostRows)
	// Back on tab 2: the loop switches away and back, so the active tab is the
	// one the flood is running in -- which is also the harshest case, since a
	// paint is requested against a stream that is actively mid-sequence.
	check("the strip survives tab switches under an ESCAPE-carrying flood",
		strip == "terminal 1 [terminal 2]",
		fmt.Sprintf("row %d = %q", hostRows-1, strip))
	check("no paint landed on the outer row under that flood",
		outer == outerMarker,
		fmt.Sprintf("row %d = %q", hostRows, outer))
	// A paint that landed INSIDE one of the child's sequences shows up as strip
	// text in the child's area rather than on its own row.
	check("no strip fragment landed in the child's area",
		!strings.Contains(childArea(sc), "terminal 1]") && !strings.Contains(childArea(sc), "OUTER-COUCH"),
		fmt.Sprintf("child area tail = %q", tail(childArea(sc), 120)))

	// The rename FIELD has to be on the strip, because M4 takes away the only
	// other surface it had: the field is drawn into the zellij pane title, and
	// the pane title is the pane FRAME.
	send(altR)
	time.Sleep(1200 * time.Millisecond)
	check("the rename field is visible without a pane frame",
		strings.Contains(sc.row(hostRows-1), "[rename:"),
		fmt.Sprintf("row %d = %q", hostRows-1, sc.row(hostRows-1)))
	send("\x1b") // cancel; the rename proper is driven below
	time.Sleep(800 * time.Millisecond)

	committed := rename(send, func() string { return sc.row(hostRows - 1) }, "日本語", trace)
	check("a wide rename commits", committed,
		fmt.Sprintf("row %d = %q", hostRows-1, sc.row(hostRows-1)))
	trace("after commit")
	strip = sc.row(hostRows - 1)
	check("a wide tab name renders intact", strings.Contains(strip, "[日本語]"),
		fmt.Sprintf("row %d = %q", hostRows-1, strip))
	check("both tabs are on the strip", strings.Contains(strip, "terminal 1"),
		fmt.Sprintf("row %d = %q", hostRows-1, strip))
	check("a wide name did not push the row over the outer one",
		strings.Contains(sc.row(hostRows), outerMarker),
		fmt.Sprintf("row %d = %q", hostRows, sc.row(hostRows)))

	long := strings.Repeat("long-tab-name-", 12) // 168 columns into a 100-column pane
	committed = rename(send, func() string { return sc.row(hostRows - 1) }, long, trace)
	check("a long rename commits", committed,
		fmt.Sprintf("row %d = %q", hostRows-1, sc.row(hostRows-1)))
	trace("after long commit")
	strip = sc.row(hostRows - 1)
	check("a long name is truncated, not dropped",
		strings.Contains(strip, "long-tab-name") && !strings.Contains(strip, long) &&
			sc.lastColumn(hostRows-1) <= hostCols,
		fmt.Sprintf("row %d ends at column %d: %q", hostRows-1, sc.lastColumn(hostRows-1), strip))
	check("truncation left the outer row alone", strings.Contains(sc.row(hostRows), outerMarker),
		fmt.Sprintf("row %d = %q", hostRows, sc.row(hostRows)))

	fmt.Println()
	fmt.Print(sc.all())
	fmt.Println()
	if fails > 0 {
		fmt.Printf("raw tail:\n%s\n\n", sc.rawTail(3000))
		fmt.Printf("RESULT: %d check(s) failed -- the two rows do NOT compose as designed\n", fails)
		stop()
		os.Exit(1)
	}
	fmt.Println("RESULT: TWO RESERVED ROWS COMPOSE")
}

// rename drives the real Alt+R rename: the editor opens PREFILLED with the
// current name, so the field is cleared with backspaces before the new name is
// typed. Enter commits.
//
// It WAITS on the strip at each step rather than sleeping a guessed interval.
// A fixed sleep sent Enter while a long name was still arriving through zellij,
// and the run then reported a rename that "did not commit" -- an instrument
// artefact that reads exactly like a defect.
func rename(send func(string), strip func() string, name string, trace func(string)) bool {
	settled := func(want string) bool {
		return waitFor(10*time.Second, func() bool { return strings.Contains(strip(), want) })
	}
	send(altR)
	if !settled("[rename:") {
		trace("rename never opened")
		return false
	}
	trace("rename opened")
	send(strings.Repeat("\x7f", 200))
	if !settled("[rename: │]") {
		trace("field never cleared")
		return false
	}
	trace("field cleared")
	send(name)
	// The strip TRUNCATES, so the tail of a long name never appears on it; wait
	// on a prefix that fits instead, and on the field having stopped growing.
	head := name
	if len([]rune(head)) > 8 {
		head = string([]rune(head)[:8])
	}
	if !settled("[rename: " + head) {
		trace("name never landed")
		return false
	}
	stable := ""
	waitFor(10*time.Second, func() bool {
		now := strip()
		if now == stable {
			return true
		}
		stable = now
		return false
	})
	trace("name typed")
	send("\r")
	if !waitFor(10*time.Second, func() bool { return !strings.Contains(strip(), "[rename:") }) {
		trace("rename never committed")
		return false
	}
	return true
}

func waitFor(limit time.Duration, ok func() bool) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if ok() {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// waitForRowMatching returns the first row anywhere on the screen that matches,
// or "" if none did before the deadline.
func waitForRowMatching(sc *screen, limit time.Duration, ok func(string) bool) string {
	var found string
	waitFor(limit, func() bool {
		for n := 1; n <= hostRows; n++ {
			if r := sc.row(n); ok(r) {
				found = r
				return true
			}
		}
		return false
	})
	return found
}

// childArea is everything the child can draw on: every row above the strip.
func childArea(sc *screen) string {
	var b strings.Builder
	for n := 1; n < hostRows-1; n++ {
		b.WriteString(sc.row(n))
		b.WriteString("\n")
	}
	return b.String()
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func describe(row string) string {
	if row == "" {
		return "not found"
	}
	return fmt.Sprintf("found %q", row)
}

func executable(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0
}

func inconclusive(format string, a ...any) {
	fmt.Printf("PROBE-INCONCLUSIVE: "+format+"\n", a...)
	os.Exit(1)
}
