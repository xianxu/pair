// Live probe for pair#211: does a draft send reach a REAL Claude Code intact,
// and does sending it as a bracketed paste change that?
//
// probes/zellijwritechars established that zellij write-chars and pair wrap's
// stdin translator deliver byte-exact. The loss is downstream: auditing pair's
// send log against Claude Code's transcripts (1,902 matched sends, 2026-09)
// found that a send arriving in three or more ~1 KiB tty reads loses whole
// middle reads — 8 of 11 three-read sends, 0 of 31 two-read sends — on every
// Claude Code version from 2.1.258 to 2.1.285. The operator's draft send was
// unbracketed input, so Claude had to infer the paste from read timing.
//
// Method: a throwaway zellij session running `pair wrap claude --model haiku`
// in a temp cwd — the agent pane's process shape. Each trial sends numbered
// lines the way nvim/draft_send.lua does (write-chars, settle, Alt+Enter) and
// reads what Claude received back out of its own transcript, so a hole is a
// list of line numbers rather than a reconstruction. Plain and bracketed sends
// alternate so both see the same session state.
//
// It SPENDS TOKENS (a dozen short haiku turns), so under test-smoke — which runs
// every probe — it reports PROBE-SKIPPED unless explicitly asked for:
//
//	PAIR_PROBE_CLAUDE=1 go run ./probes/claudedraftsend [path to pair binary]
//
// It reads Claude's transcripts directly. That is native-authority knowledge
// production code must take from sessioninventory; a diagnostic probe measuring
// what the agent actually received is the one place the raw record is the
// point.
//
// A session that never reaches Claude's composer is a PRECONDITION failure,
// never a verdict (pair#208).
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/xianxu/pair/probes/zellijprobe"
)

func main() { os.Exit(run()) }

const trialsPerMode = 3

// payload is size bytes of numbered lines between per-trial markers, behind a
// one-line instruction that keeps the reply (and the cost) to a word.
func payload(trial, size int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Reply with only the word OK.\n<<B%02d>>\n", trial)
	for line := 1; b.Len() < size; line++ {
		fmt.Fprintf(&b, "t%02d-%05d abcdefghijklmnopqrstuvwxyz0123\n", trial, line)
	}
	fmt.Fprintf(&b, "<<E%02d>>", trial)
	return b.String()
}

// received returns the text of the transcript's user message carrying the
// trial's begin marker, or "" while it has not arrived.
func received(projectGlob string, trial int) string {
	begin := fmt.Sprintf("<<B%02d>>", trial)
	files, _ := filepath.Glob(projectGlob)
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 1<<20), 16<<20)
		for sc.Scan() {
			line := sc.Bytes()
			if !bytes.Contains(line, []byte(begin)) || !bytes.Contains(line, []byte(`"type":"user"`)) {
				continue
			}
			var rec struct {
				Message struct {
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(line, &rec) != nil {
				continue
			}
			var s string
			if json.Unmarshal(rec.Message.Content, &s) == nil {
				fh.Close()
				return s
			}
			var parts []struct {
				Type, Text string
			}
			if json.Unmarshal(rec.Message.Content, &parts) == nil {
				var b strings.Builder
				for _, p := range parts {
					if p.Type == "text" {
						b.WriteString(p.Text)
					}
				}
				fh.Close()
				return b.String()
			}
		}
		fh.Close()
	}
	return ""
}

var pasteTag = regexp.MustCompile(`</?pasted_content[^>]*>`)

// compact drops what Claude adds that is not a loss: the <pasted_content> tags
// it wraps pasted runs in, and the blank lines it inserts at their boundaries —
// which can land inside a numbered line and must not read as a lost line.
func compact(s string) string {
	s = pasteTag.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' {
			return -1
		}
		return r
	}, s)
}

// lost is how many bytes of want (whitespace-free) are missing from got, and
// where the hole starts. Zero means the send arrived whole.
func lost(got, want string) (int, int) {
	g, w := compact(got), compact(want)
	if i := strings.Index(g, compact(want[:40])); i >= 0 {
		g = g[i:]
	}
	if strings.HasPrefix(g, w) {
		return 0, 0
	}
	head := 0
	for head < len(g) && head < len(w) && g[head] == w[head] {
		head++
	}
	tail := 0
	for tail < len(g)-head && tail < len(w)-head && g[len(g)-1-tail] == w[len(w)-1-tail] {
		tail++
	}
	return len(w) - tail - head, head
}

func screen(s *zellijprobe.Session, env []string) string {
	b, _ := s.Action(env, "dump-screen")
	return string(b)
}

// idle is Claude's composer with nothing running: the shortcuts hint shows only
// then.
func idle(sc string) bool { return strings.Contains(sc, "? for shortcuts") }

func run() int {
	if os.Getenv("PAIR_PROBE_CLAUDE") != "1" {
		fmt.Println("PROBE-SKIPPED: drives a real Claude and spends tokens; set PAIR_PROBE_CLAUDE=1 to run it.")
		return 0
	}
	_, self, _, _ := runtime.Caller(0)
	dir := filepath.Dir(self)
	pair := filepath.Join(dir, "..", "..", "bin", "pair")
	if len(os.Args) > 1 {
		pair = os.Args[1]
	}
	claude, err := exec.LookPath("claude")
	if err != nil {
		fmt.Println("PROBE-ERROR: claude not on PATH")
		return 1
	}
	version, _ := exec.Command(claude, "--version").Output()

	tmp, err := os.MkdirTemp("", "claudedraftsend-*")
	if err != nil {
		fmt.Println("PROBE-ERROR temp:", err)
		return 1
	}
	defer os.RemoveAll(tmp)
	work := filepath.Join(tmp, "work")
	data := filepath.Join(tmp, "data")
	for _, d := range []string{work, data} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			fmt.Println("PROBE-ERROR mkdir:", err)
			return 1
		}
	}
	// Claude names a project dir after the cwd with every non-alphanumeric
	// byte dashed; the temp dir's unique basename is enough to find ours, and
	// to delete only ours.
	home, _ := os.UserHomeDir()
	projects := filepath.Join(home, ".claude", "projects")
	projectDirs := filepath.Join(projects, "*"+filepath.Base(tmp)+"*")
	defer func() {
		dirs, _ := filepath.Glob(projectDirs)
		for _, d := range dirs {
			_ = os.RemoveAll(d)
		}
	}()

	layout, err := zellijprobe.WriteLayout(dir, map[string]string{
		"PROBE_PAIR": pair, "PROBE_CWD": work, "PROBE_CLAUDE": claude,
	})
	if err != nil {
		fmt.Println("PROBE-ERROR layout:", err)
		return 1
	}
	defer os.Remove(layout)

	// CLAUDE* is scrubbed so a run from inside Claude Code does not look
	// nested; PAIR_* so it cannot write into the calling pair session's data.
	env := zellijprobe.Scrub([]string{"ZELLIJ", "PAIR_", "CLAUDE"}, "TERM=xterm-256color",
		"PAIR_DATA_DIR="+data, "PAIR_TAG=probe")
	session, err := zellijprobe.Start(zellijprobe.Options{
		ConfigFile: filepath.Join(dir, "..", "..", "zellij", "config.kdl"),
		Layout:     layout,
		NamePrefix: "claudedraftsend",
		Rows:       40, Cols: 120,
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

	// Reach an idle composer, accepting the folder-trust dialog a fresh cwd
	// raises.
	deadline := time.Now().Add(60 * time.Second)
	confirms := 0
	for {
		sc := screen(session, env)
		if idle(sc) {
			break
		}
		// The dialog's default is "No, exit": move to "Yes" first and confirm
		// only once it is the highlighted row. Confirm with Enter, then Alt+Enter
		// on the next pass — pair wrap may remap a bare Return.
		if strings.Contains(sc, "trust this folder") {
			switch {
			case strings.Contains(sc, "❯ No"):
				_, _ = session.Action(env, "send-keys", "Down")
			case strings.Contains(sc, "❯ Yes"):
				confirms++
				key := "Enter"
				if confirms%2 == 0 {
					key = "Alt Enter"
				}
				_, _ = session.Action(env, "send-keys", key)
			}
		}
		if time.Now().After(deadline) {
			fmt.Println("PROBE-INCONCLUSIVE: Claude never reached an idle composer; nothing was measured.")
			fmt.Printf("--- pane:\n%s\n", strings.TrimRight(sc, " \n"))
			return 2
		}
		time.Sleep(time.Second)
	}

	fmt.Printf("draft send into real Claude Code %s", version)
	sizes := []int{2447, 3500, 4600}
	type tally struct{ sends, lossy int }
	results := map[bool]*tally{false: {}, true: {}}
	trial := 0
	for round := 0; round < trialsPerMode; round++ {
		for _, bracketed := range []bool{false, true} {
			size := sizes[round%len(sizes)]
			msg := payload(trial, size)
			wire := msg
			if bracketed {
				wire = "\x1b[200~" + msg + "\x1b[201~"
			}
			// The production shape: body, settle, Alt+Enter (draft_send.lua).
			if out, err := session.Action(env, "write-chars", wire); err != nil {
				fmt.Printf("PROBE-ERROR write-chars: %v %s\n", err, out)
				return 2
			}
			time.Sleep(100 * time.Millisecond)
			if out, err := session.Action(env, "send-keys", "Alt Enter"); err != nil {
				fmt.Printf("PROBE-ERROR submit: %v %s\n", err, out)
				return 2
			}
			var got string
			arrive := time.Now().Add(60 * time.Second)
			for got == "" && time.Now().Before(arrive) {
				time.Sleep(500 * time.Millisecond)
				got = received(filepath.Join(projectDirs, "*.jsonl"), trial)
			}
			if got == "" {
				fmt.Printf("PROBE-INCONCLUSIVE: trial %d never reached the transcript.\n", trial)
				fmt.Printf("--- pane:\n%s\n", strings.TrimRight(screen(session, env), " \n"))
				return 2
			}
			n, at := lost(got, msg)
			mode := "plain    "
			if bracketed {
				mode = "bracketed"
			}
			t := results[bracketed]
			t.sends++
			if n > 0 {
				t.lossy++
				fmt.Printf("  %s %5dB  LOST %d bytes from compact offset %d\n", mode, len(msg), n, at)
			} else {
				fmt.Printf("  %s %5dB  exact\n", mode, len(msg))
			}
			// Wait out the reply so every trial starts at an idle composer.
			settle := time.Now().Add(90 * time.Second)
			for !idle(screen(session, env)) && time.Now().Before(settle) {
				time.Sleep(time.Second)
			}
			time.Sleep(time.Second)
			trial++
		}
	}
	fmt.Printf("plain: %d of %d sends lost lines; bracketed: %d of %d\n",
		results[false].lossy, results[false].sends, results[true].lossy, results[true].sends)
	switch {
	case results[true].lossy > 0:
		fmt.Println("VERDICT: BRACKETED PASTE DOES NOT PROTECT THE SEND")
	case results[false].lossy > 0:
		fmt.Println("VERDICT: PLAIN SENDS LOSE LINES; BRACKETED SENDS ARRIVE WHOLE")
	default:
		fmt.Println("VERDICT: NO LOSS REPRODUCED — plain sends arrived whole this run")
	}
	return 0
}
