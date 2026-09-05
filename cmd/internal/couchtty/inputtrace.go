package couchtty

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// renderInputBytes turns a chunk of operator input into one readable line.
//
// The rendering an operator actually needs is the escape sequence spelled the
// way the chord table spells it -- `\x1b[110;3u`, not `1b 5b 31 31 30 3b 33 75`
// -- because the question being answered is always "did the terminal send the
// bytes couch is watching for". So printable bytes stay printable, ESC becomes
// the same `\x1b` the table uses, and only genuinely unprintable bytes fall
// back to hex.
func renderInputBytes(chunk []byte) string {
	var b strings.Builder
	for _, c := range chunk {
		switch {
		case c == 0x1b:
			b.WriteString(`\x1b`)
		case c == 0x00:
			b.WriteString(`\x00`)
		case c >= 0x20 && c < 0x7f:
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, `\x%02x`, c)
		}
	}
	return b.String()
}

// inputTracer appends the operator's raw keystrokes to the file named by
// COUCH_INPUT_TRACE, and does nothing at all when that is unset.
//
// It exists because "the chord had no effect" has two causes that look
// identical from the operator's chair: couch consumed the bytes and dispatched
// nothing, or the terminal never sent the bytes couch watches for. No amount of
// re-reading the interception table separates them -- only the wire does. macOS
// Option+n is a dead-tilde composer, which is why Pair carries ctrl+alt+n at
// all, so the second cause is a standing possibility for this chord in
// particular.
//
// Deliberately NOT a visual affordance. The console hosts a child terminal;
// a probe that painted anything would corrupt the child's screen, which is a
// worse bug than the one being chased.
type inputTracer struct {
	mu sync.Mutex
	f  *os.File
}

// newInputTracer returns a nil tracer when the path is empty, and an ERROR when
// a path was given and could not be opened.
//
// The PATH IS A PARAMETER. Reading os.Getenv inside the constructor meant every
// Console ever built opened a real file for append -- and couchtty builds many
// per test run, so a developer whose shell exports the variable got one leaked
// fd per Console and a trace file full of fixture bytes. This repo has already
// been bitten by PAIR_SESSION_ID/PAIR_TAG leaking into `make test`; ambient env
// reached from a constructor is the same hazard with a file attached.
//
// Returning nil for both "off" and "could not open" was a second problem: an
// unwritable path produced an empty trace indistinguishable from "no bytes
// arrived", which is precisely the ambiguity the probe exists to resolve. The
// INABILITY to observe must never be presentable as an observation.
func newInputTracer(path string) (*inputTracer, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("COUCH_INPUT_TRACE: %w", err)
	}
	return &inputTracer{f: f}, nil
}

// Close releases the trace file. Nil-safe, like record.
func (t *inputTracer) Close() error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.f == nil {
		return nil
	}
	err := t.f.Close()
	t.f = nil
	return err
}

func (t *inputTracer) record(chunk []byte) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.f == nil {
		return
	}
	fmt.Fprintf(t.f, "%s %s\n", time.Now().Format("15:04:05.000"), renderInputBytes(chunk))
}
