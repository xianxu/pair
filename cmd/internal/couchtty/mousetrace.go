package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"strconv"
	"strings"
	"time"
)

// mouseTracer appends couch's host mouse-mode decisions to the file named by
// COUCH_MOUSE_TRACE, and does nothing at all when that is unset.
//
// It records scanner beliefs and attempted host writes at live, takeover,
// startup, paint and cleanup boundaries. A successful Write is byte acceptance,
// not a terminal query. Concurrent records do not establish terminal wire order.
// No child output, prompt or replay content is recorded.
type mouseTracer struct{ file *traceFile }

// newMouseTracer returns a nil tracer when the path is empty, and an ERROR when
// a path was given and could not be opened. The path is a PARAMETER, never read
// from env here, for the reason newInputTracer documents at length: a
// constructor that reaches for ambient env opens a real file per Console and
// leaks fds through every test.
func newMouseTracer(path string, options ...diagnosticlog.Options) (*mouseTracer, error) {
	file, err := openTraceFile("COUCH_MOUSE_TRACE", path, options...)
	if file == nil {
		return nil, err
	}
	return &mouseTracer{file: file}, nil
}

// Close releases the trace file. Nil-safe, like record.
func (t *mouseTracer) Close() error {
	if t == nil {
		return nil
	}
	return t.file.Close()
}

// record writes one line: <unix-ms>\t<event>\t<detail>. Nil-safe: the OFF state
// is a nil tracer, so every call site reads the field and calls unconditionally.
func (t *mouseTracer) record(event, detail string) {
	if t == nil {
		return
	}
	if detail == "" {
		detail = "-"
	}
	t.file.writeLine(strconv.FormatInt(time.Now().UnixMilli(), 10) + "\t" + event + "\t" + detail)
}

// formatMouseModes renders a Screen.MouseModes() value for the trace, e.g.
// "1002,1006" or "none".
func formatMouseModes(modes []int) string {
	if len(modes) == 0 {
		return "none"
	}
	parts := make([]string, len(modes))
	for i, m := range modes {
		parts[i] = strconv.Itoa(m)
	}
	return strings.Join(parts, ",")
}

// mouseWriteResult observes an existing write; it never retries or changes policy.
type mouseWriteResult struct {
	n        int
	err      error
	deferred bool
	released bool
}

func (r mouseWriteResult) detail(want int) string {
	outcome := "emitted"
	switch {
	case r.released:
		outcome = "suppressed"
	case r.deferred:
		outcome = "deferred"
	case r.err != nil:
		outcome = "error"
	case r.n != want:
		outcome = "short-write"
	}
	detail := "outcome=" + outcome + " written=" + strconv.Itoa(r.n) + " requested=" + strconv.Itoa(want)
	if r.released {
		detail += " reason=terminal-released"
	}
	if r.deferred {
		detail += " reason=unsafe-paint"
	}
	if r.err != nil {
		detail += " error=" + mouseTraceQuote(r.err.Error())
	}
	return detail
}

// Bound before quoting so control bytes cannot forge records or inflate a field
// beyond 512 escaped bytes. This is a diagnostic label, not an identity lookup.
func mouseTraceQuote(s string) string {
	if len(s) > 128 {
		s = s[:128] + "…"
	}
	return strconv.Quote(s)
}

// Called with c.mu held. Snapshot identity before IO, including the retained
// actor under a panel, whose mouse belief the current policy still consults.
func (c *Console) mouseTraceContextLocked() (*mouseTracer, string) {
	if c.mouseTrace == nil {
		return nil, ""
	}
	surface := "actor"
	if c.focus.IsPanel() {
		surface = "panel"
	}
	detail := "active=" + mouseTraceQuote(c.active) + " surface=" + surface
	if p := c.panes[c.active]; p != nil {
		detail += " actor=" + mouseTraceQuote(string(p.actorID)) + " thread=" + mouseTraceQuote(p.thread.RepoScope+"/"+string(p.thread.Tag)) +
			" child-mouse=" + strconv.FormatBool(p.child.Mouse()) + " child-observed=" + strconv.FormatBool(p.child.MouseObserved())
	}
	return c.mouseTrace, detail
}

func (c *Console) traceMouseClicks(source string) {
	c.mu.Lock()
	tracer, context := c.mouseTraceContextLocked()
	before := c.hostScan.MouseModes()
	c.mu.Unlock()
	result := c.writeOwn(hostty.EnableMouseClicks)
	if tracer == nil {
		return
	}
	tracer.record("assert-clicks", context+" source="+source+" scanner-before="+formatMouseModes(before)+" "+result.detail(len(hostty.EnableMouseClicks)))
}
