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

// newInputTracer returns nil when tracing is off OR when the file cannot be
// opened. A diagnostic that can take the console down is not worth having, and
// nil is a working tracer here -- record has a nil receiver guard.
func newInputTracer() *inputTracer {
	path := os.Getenv("COUCH_INPUT_TRACE")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil
	}
	return &inputTracer{f: f}
}

func (t *inputTracer) record(chunk []byte) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintf(t.f, "%s %s\n", time.Now().Format("15:04:05.000"), renderInputBytes(chunk))
}
