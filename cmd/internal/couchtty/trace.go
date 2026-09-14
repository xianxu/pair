package couchtty

import (
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

// traceFile is the file an operator-requested trace appends to. Both of couch's
// traces write through it (ARCH-DRY):
//   - the keystroke probe (inputtrace.go);
//   - the timing trace below.
//
// The path is a parameter, never read from env here; newInputTracer explains why.
// A write after Close is a no-op, because teardown closes the file while a worker
// may still be finishing a write. The OFF state, a nil tracer, belongs to the two
// tracers, so a traceFile always exists once it has been opened.
type traceFile struct {
	mu sync.Mutex
	f  *diagnosticlog.Writer
}

// openTraceFile returns nil when path is empty, meaning the trace is off. It
// returns an error naming the variable when a path was given but could not be
// opened: the inability to observe must never read as an observation.
func openTraceFile(variable, path string, options ...diagnosticlog.Options) (*traceFile, error) {
	if path == "" {
		return nil, nil
	}
	opts := diagnosticlog.Options{Proof: diagnosticlog.DefaultProof}
	if len(options) > 0 {
		opts = options[0]
	}
	f, err := diagnosticlog.Open(path, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", variable, err)
	}
	return &traceFile{f: f}, nil
}

func (t *traceFile) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.f == nil {
		return nil
	}
	err := t.f.Close()
	t.f = nil
	return err
}

func (t *traceFile) writeLine(line string) {
	if !t.mu.TryLock() {
		return
	}
	defer t.mu.Unlock()
	if t.f == nil {
		return
	}
	_, _ = io.WriteString(t.f, line+"\n")
}

// The COUCH_TRACE timing trace's events (pair#206). Each is one line:
//
//	<unix-ms>\t<event>\t<scope>/<tag>\t<detail>
//
// "-" stands for an absent address or detail. The trace records addresses,
// counts and timings, never content.
const (
	traceStartup       = "startup"        // the process began: stamped with its start, not when the trace opened
	traceFirstFrame    = "first-frame"    // the console's first paint of its own row
	traceInventory     = "inventory"      // an inventory landed; counting them gives the refreshes during the pass
	tracePassSeeded    = "pass-seeded"    // the pass took its queue from an inventory
	traceReattachStart = "reattach-start" // one pass attempt was dispatched
	traceReattachDone  = "reattach-done"  // ... and finished
)

// eventTracer writes the timing trace, and does nothing at all when it is off.
// It exists to measure startup and the reattach pass against a real terminal.
// Like the keystroke probe, it paints nothing: the console hosts a child
// terminal.
type eventTracer struct{ file *traceFile }

func newEventTracer(path string, options ...diagnosticlog.Options) (*eventTracer, error) {
	file, err := openTraceFile("COUCH_TRACE", path, options...)
	if file == nil {
		return nil, err
	}
	return &eventTracer{file: file}, nil
}

// Close releases the trace file. Nil-safe, like record.
func (t *eventTracer) Close() error {
	if t == nil {
		return nil
	}
	return t.file.Close()
}

func (t *eventTracer) record(at time.Time, event string, address couchcore.ThreadAddress, detail string) {
	if t == nil {
		return
	}
	t.file.writeLine(formatTraceEvent(at, event, address, detail))
}

// formatTraceEvent renders one timing-trace line; the format is documented on
// the event constants above.
func formatTraceEvent(at time.Time, event string, address couchcore.ThreadAddress, detail string) string {
	where := "-"
	if address != (couchcore.ThreadAddress{}) {
		where = address.RepoScope + "/" + string(address.Tag)
	}
	if detail == "" {
		detail = "-"
	}
	return strconv.FormatInt(at.UnixMilli(), 10) + "\t" + event + "\t" + where + "\t" + detail
}

// reattachDoneDetail records how a pass attempt ended:
//   - "ok" on success;
//   - the resume's diagnostic code, on a failure that has one;
//   - "error" otherwise.
//
// It records a code, never the error's text, because the trace records no
// content.
func reattachDoneDetail(success bool, diagnostic couchcore.ResumeDiagnosticCode) string {
	switch {
	case success:
		return "ok"
	case diagnostic != "":
		return string(diagnostic)
	default:
		return "error"
	}
}

// SetEventTrace opens the COUCH_TRACE timing trace at path, or leaves it off
// when path is empty. It stamps the startup event with processStart: the
// console did not exist when the process began, so the composition root, which
// read the clock first, supplies it. A failed open is reported the way
// SetInputTrace reports one.
func (c *Console) SetEventTrace(path string, processStart time.Time, options ...diagnosticlog.Options) error {
	tracer, err := newEventTracer(path, options...)
	c.mu.Lock()
	previous := c.events
	c.events = tracer
	c.mu.Unlock()
	_ = previous.Close()
	if err != nil {
		c.publishNotice(Notice{Kind: "trace", Control: true, Body: err.Error()})
		return err
	}
	tracer.record(processStart, traceStartup, couchcore.ThreadAddress{}, "")
	return nil
}

// traceEvent records one timing-trace event, stamped now. The caller must not
// hold c.mu. The lock is taken only to read the tracer; the write happens
// outside it, as all of the console's IO does.
func (c *Console) traceEvent(event string, address couchcore.ThreadAddress, detail string) {
	c.mu.Lock()
	events := c.events
	c.mu.Unlock()
	events.record(time.Now(), event, address, detail)
}
