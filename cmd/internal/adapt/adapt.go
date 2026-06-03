// Package adapt writes the per-session adaptation flight recorder: one
// JSON line per harness-adaptation event into
// $PAIR_DATA_DIR/adapt-<tag>.jsonl.
//
// pair adapts each agent harness across a handful of integration aspects
// (return-key remap, overlay suspension, slug generation, …; see
// atlas/how-to-bring-up-a-new-harness-cli.md). Harnesses drift — a renamed
// picker string or transcript shape silently breaks an adaptation — and the
// breakage manifests as *silence*, not an error. The flight recorder makes
// that drift observable: every adaptation logs when it fires AND when it
// near-misses (the harness did something we half-recognized but no matcher
// caught), so `pair-doctor` can read the trace and point at the broken aspect.
//
// Multiple components append concurrently (pair-wrap, pair-slug, plus shell
// and Lua emitters writing the same line format from other processes). All
// appends are O_APPEND of a single sub-PIPE_BUF line, which the kernel keeps
// atomic across processes; bin/pair truncates the file once at session launch
// so no writer ever races on truncation.
package adapt

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// maxDetail caps the free-text detail field. detail can carry a snippet of
// agent output (e.g. an unrecognized prompt string), so it is bounded both to
// keep lines small and to limit how much transcript content lands on disk.
const maxDetail = 200

// Outcome enumerates what happened when an adaptation point was reached.
type Outcome string

const (
	// Fired: the adaptation matched and acted as designed.
	Fired Outcome = "fired"
	// Bypass: the adaptation deliberately stepped aside (e.g. plain Enter
	// passed through as a bare CR because an overlay was active).
	Bypass Outcome = "bypass"
	// NearMiss: the harness emitted something we half-recognized but no
	// specific matcher caught it — the fingerprint of drift.
	NearMiss Outcome = "near-miss"
	// Fail: the adaptation was expected to work but couldn't (e.g. a session
	// id never resolved).
	Fail Outcome = "fail"
)

// event is one line of the flight recorder. Flat by design: detail is a
// single capped string, never a nested object, so the shell and Lua emitters
// can produce the identical shape with a one-line printf.
type event struct {
	TS      string `json:"ts"`
	Comp    string `json:"comp"`
	Agent   string `json:"agent"`
	Aspect  int    `json:"aspect"`
	Signal  string `json:"signal"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail,omitempty"`
}

// Logger appends adaptation events for one component. A nil *Logger is a safe
// no-op, so callers never have to nil-check — telemetry must never block or
// crash the thing it observes.
type Logger struct {
	mu     sync.Mutex
	w      io.Writer
	closer io.Closer
	comp   string
	agent  string
	now    func() time.Time
}

// DataDir returns $PAIR_DATA_DIR or the XDG default. This is the canonical
// home for all per-session pair files; callers should use it rather than
// re-deriving the path.
func DataDir() string {
	if d := os.Getenv("PAIR_DATA_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "pair")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "pair")
}

// New builds a Logger writing to w. Used directly by tests; production code
// uses Open. comp names the writing component (e.g. "pair-wrap"); agent is
// the active harness basename.
func New(w io.Writer, comp, agent string) *Logger {
	return &Logger{w: w, comp: comp, agent: agent, now: time.Now}
}

// Open opens the session flight recorder for appending and returns a Logger.
// Returns nil (a no-op Logger) when $PAIR_TAG is unset or the file can't be
// opened — telemetry failures are never fatal. The caller owns Close.
func Open(comp, agent string) *Logger {
	tag := os.Getenv("PAIR_TAG")
	if tag == "" {
		return nil
	}
	path := filepath.Join(DataDir(), "adapt-"+tag+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	l := New(f, comp, agent)
	l.closer = f
	return l
}

// Log appends one event. Safe on a nil receiver. Errors are swallowed.
func (l *Logger) Log(aspect int, signal string, outcome Outcome, detail string) {
	if l == nil || l.w == nil {
		return
	}
	line := marshalEvent(l.now().UTC(), l.comp, l.agent, aspect, signal, outcome, detail)
	l.mu.Lock()
	_, _ = l.w.Write(line)
	l.mu.Unlock()
}

// Close releases the underlying file, if any. Safe on a nil receiver.
func (l *Logger) Close() error {
	if l == nil || l.closer == nil {
		return nil
	}
	return l.closer.Close()
}

// marshalEvent renders one newline-terminated JSON line. Pure (time passed in)
// so the line format is unit-testable without touching the clock or env.
func marshalEvent(ts time.Time, comp, agent string, aspect int, signal string, outcome Outcome, detail string) []byte {
	line, _ := json.Marshal(event{
		TS:      ts.Format(time.RFC3339),
		Comp:    comp,
		Agent:   agent,
		Aspect:  aspect,
		Signal:  signal,
		Outcome: string(outcome),
		Detail:  truncate(detail, maxDetail),
	})
	return append(line, '\n')
}

// truncate caps s to at most n bytes without splitting a multi-byte rune, so
// the result is always valid UTF-8 (and thus valid JSON).
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Back up off any continuation bytes (0b10xxxxxx) at the cut point.
	for n > 0 && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}
