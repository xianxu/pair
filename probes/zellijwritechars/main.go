// Conformance probe for pair#211: does `zellij action write-chars` deliver a
// payload byte-exact to the pane process?
//
// The draft editor sends every prompt as ONE `write-chars <body>` call
// (nvim/draft_send.lua). On 2026-09-07 a 2,447-byte send lost one contiguous
// 1,025-byte run out of its middle, head and tail intact, while a ~180 KB send
// minutes earlier arrived whole. 1,025 is suspiciously close to macOS's tty
// input queue (TTYHOG / MAX_INPUT = 1024): a pty master write blocks — returns
// EAGAIN on a non-blocking fd — once the slave's queue is full, and a writer that
// drops the unwritten remainder of a short write loses about one queue-full.
//
// Method: a throwaway zellij session whose pane process is this binary
// re-executed as a raw-mode recorder, appending every byte it reads to a file.
// The probe sends numbered-line payloads across the 1 KB boundary and compares
// what arrived. The recorder can read slowly (PAIR_PROBE_READER=slow), which
// holds the slave queue full the way a busy agent does; the fast reader is the
// control. PAIR_PROBE_WRAP=<pair binary> puts `pair wrap` between zellij and the
// recorder, under the claude TTY profile, so the stdin translator is in the path.
//
//	go run ./probes/zellijwritechars
//	PAIR_PROBE_READER=slow go run ./probes/zellijwritechars
//	PAIR_PROBE_WRAP=$PWD/bin/pair go run ./probes/zellijwritechars
//
// A session that never appears is a PRECONDITION failure, never a verdict
// (pair#208).
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/xianxu/pair/probes/zellijprobe"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "record" {
		os.Exit(record(os.Getenv("PAIR_PROBE_OUT"), os.Getenv("PAIR_PROBE_READER") == "slow"))
	}
	os.Exit(run())
}

// record is the pane process: raw mode, so the tty neither edits nor echoes nor
// caps a line, then every byte read goes to outPath. The file is created before
// the first read so the probe can use its existence as readiness.
func record(outPath string, slow bool) int {
	out, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return 1
	}
	defer out.Close()
	if state, err := term.MakeRaw(0); err == nil {
		defer term.Restore(0, state)
	}
	size := 64 << 10
	if slow {
		size = 64
	}
	buf := make([]byte, size)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return 1
			}
		}
		if err != nil {
			return 0
		}
		if slow {
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// payload is numbered lines, so a hole is readable directly: the first missing
// line number is where the cut starts. It is exactly size bytes.
func payload(trial, size int) []byte {
	var b bytes.Buffer
	for line := 1; b.Len() < size; line++ {
		fmt.Fprintf(&b, "t%02d-%05d abcdefghijklmnopqrstuvwxyz0123\n", trial, line)
	}
	return b.Bytes()[:size]
}

type result struct {
	complete bool   // the end marker arrived
	exact    bool   // the body arrived byte-exact
	detail   string // where it diverged, when it did
}

// compare finds the trial's body between its markers in everything recorded so
// far. A missing END marker means the send has not (or never) finished arriving.
func compare(recorded []byte, trial int, want []byte) result {
	begin := []byte(fmt.Sprintf("<<B%02d>>", trial))
	end := []byte(fmt.Sprintf("<<E%02d>>", trial))
	i := bytes.Index(recorded, begin)
	if i < 0 {
		return result{detail: "begin marker absent"}
	}
	rest := recorded[i+len(begin):]
	j := bytes.Index(rest, end)
	if j < 0 {
		return result{detail: fmt.Sprintf("end marker absent after %d bytes", len(rest))}
	}
	got := rest[:j]
	if bytes.Equal(got, want) {
		return result{complete: true, exact: true}
	}
	head := 0
	for head < len(got) && head < len(want) && got[head] == want[head] {
		head++
	}
	tail := 0
	for tail < len(got)-head && tail < len(want)-head && got[len(got)-1-tail] == want[len(want)-1-tail] {
		tail++
	}
	return result{complete: true, detail: fmt.Sprintf(
		"sent %d got %d: cut at byte %d, resumed at byte %d (missing %d, extra %d)",
		len(want), len(got), head, len(want)-tail, len(want)-tail-head, len(got)-tail-head)}
}

func run() int {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Println("PROBE-ERROR: cannot locate probe assets")
		return 1
	}
	dir := filepath.Dir(self)
	bin, err := os.Executable()
	if err != nil {
		fmt.Println("PROBE-ERROR executable:", err)
		return 1
	}
	tmp, err := os.MkdirTemp("", "zellijwritechars-*")
	if err != nil {
		fmt.Println("PROBE-ERROR temp:", err)
		return 1
	}
	defer os.RemoveAll(tmp)
	outPath := filepath.Join(tmp, "recorded.bin")
	if err := os.MkdirAll(filepath.Join(tmp, "data"), 0o700); err != nil {
		fmt.Println("PROBE-ERROR data dir:", err)
		return 1
	}

	// Direct: the recorder is the pane process (extra args ignored). Wrapped:
	// `pair wrap <tmp>/claude record` -- the symlink's basename selects the
	// claude TTY profile, which is what turns the translator on.
	argv := map[string]string{"PROBE_BIN": bin, "PROBE_A1": "record", "PROBE_A2": "-", "PROBE_A3": "-"}
	wrap := os.Getenv("PAIR_PROBE_WRAP")
	if wrap != "" {
		link := filepath.Join(tmp, "claude")
		if err := os.Symlink(bin, link); err != nil {
			fmt.Println("PROBE-ERROR symlink:", err)
			return 1
		}
		argv = map[string]string{"PROBE_BIN": wrap, "PROBE_A1": "wrap", "PROBE_A2": link, "PROBE_A3": "record"}
	}
	layout, err := zellijprobe.WriteLayout(dir, argv)
	if err != nil {
		fmt.Println("PROBE-ERROR layout:", err)
		return 1
	}
	defer os.Remove(layout)

	// PAIR_* is scrubbed too: a probe run from inside a pair session must not
	// write into that session's data dir. The wrapped run's own trace lands in
	// tmp instead.
	env := zellijprobe.Scrub([]string{"ZELLIJ", "PAIR_"}, "TERM=xterm-256color",
		"PAIR_PROBE_OUT="+outPath, "PAIR_PROBE_READER="+os.Getenv("PAIR_PROBE_READER"),
		"PAIR_DATA_DIR="+filepath.Join(tmp, "data"), "PAIR_TAG=probe")
	session, err := zellijprobe.Start(zellijprobe.Options{
		ConfigFile: filepath.Join(dir, "..", "..", "zellij", "config.kdl"),
		Layout:     layout,
		NamePrefix: "zellijwritechars",
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
	ready := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(outPath); err == nil {
			break
		}
		if time.Now().After(ready) {
			fmt.Println("PROBE-INCONCLUSIVE: the recorder pane never started; nothing was measured.")
			if screen, err := session.Action(env, "dump-screen"); err == nil {
				fmt.Printf("--- pane:\n%s\n", strings.TrimRight(string(screen), " \n"))
			}
			return 2
		}
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond) // raw mode is set after the file exists

	reader := os.Getenv("PAIR_PROBE_READER")
	if reader == "" {
		reader = "fast"
	}
	sizes := []int{512, 1000, 1024, 1025, 1100, 2447, 4096, 16384, 180000}
	path := "direct"
	if wrap != "" {
		path = "via pair wrap"
	}
	fmt.Printf("zellij write-chars delivery, reader=%s, %s\n", reader, path)
	lossy, inconclusive := 0, 0
	for trial, size := range sizes {
		want := payload(trial, size)
		msg := fmt.Sprintf("<<B%02d>>%s<<E%02d>>", trial, want, trial)
		start := time.Now()
		if out, err := session.Action(env, "write-chars", msg); err != nil {
			fmt.Printf("  %7d  PROBE-ERROR write-chars: %v %s\n", size, err, out)
			inconclusive++
			continue
		}
		// Arrival time scales with the reader: a slow reader drains 64 bytes
		// per 20 ms, ~3.2 KB/s.
		deadline := time.Now().Add(10*time.Second + time.Duration(size/3)*time.Millisecond)
		var r result
		for {
			recorded, _ := os.ReadFile(outPath)
			r = compare(recorded, trial, want)
			if r.complete || time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		switch {
		case !r.complete:
			fmt.Printf("  %7d  INCOMPLETE  %s\n", size, r.detail)
			lossy++
		case r.exact:
			fmt.Printf("  %7d  exact       %v\n", size, time.Since(start).Round(time.Millisecond))
		default:
			fmt.Printf("  %7d  LOSSY       %s\n", size, r.detail)
			lossy++
		}
	}
	// The wrapped reading is only about the translator if the translator ran:
	// pair-wrap traces every stdin chunk it translates, so count those rather
	// than trusting that the `claude` basename selected a profile.
	if wrap != "" {
		translated := 0
		traces, _ := filepath.Glob(filepath.Join(tmp, "data", "**", "wrap-events-*.jsonl"))
		more, _ := filepath.Glob(filepath.Join(tmp, "data", "wrap-events-*.jsonl"))
		for _, f := range append(traces, more...) {
			b, _ := os.ReadFile(f)
			translated += bytes.Count(b, []byte(`"mode":"translate"`))
		}
		fmt.Printf("pair-wrap translated %d stdin chunk(s)\n", translated)
		if translated == 0 {
			fmt.Println("PROBE-INCONCLUSIVE: pair-wrap's trace shows no translated input; the translator was not in the path.")
			return 2
		}
	}
	switch {
	case inconclusive > 0:
		fmt.Println("PROBE-INCONCLUSIVE: some sends could not be issued; see above.")
		return 2
	case lossy > 0:
		fmt.Printf("VERDICT: WRITE-CHARS LOSES BYTES (%d of %d sends)\n", lossy, len(sizes))
	default:
		fmt.Println("VERDICT: WRITE-CHARS BYTE-EXACT")
	}
	return 0
}

