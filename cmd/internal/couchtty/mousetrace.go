package couchtty

import (
	"strconv"
	"strings"
	"time"
)

// mouseTracer appends couch's host mouse-mode decisions to the file named by
// COUCH_MOUSE_TRACE, and does nothing at all when that is unset.
//
// It exists because "drag highlight is dead until I restart couch" (#207) has
// a cause no amount of re-reading the code separates: WHICH layer turned the
// host's motion tracking off, and WHEN. couch is the only writer of the real
// terminal's mouse mode, so this probe records the two events that decide it —
// a change couch OBSERVES in the child's stream (`writeChild`), and a mode
// couch ASSERTS itself (`paintNow`'s `couchMayOwnTheMouse` clicks-assert) —
// with the host's mode set at each. A trace showing "child dropped 1002 at T1,
// couch asserted 1000 at T2" localises the trigger to the layer below couch;
// one showing couch asserting 1000 over a host that still holds 1002 localises
// it to couch itself.
//
// Deliberately NOT a visual affordance, like the keystroke probe: the console
// hosts a child terminal, so anything painted would corrupt the child's screen.
type mouseTracer struct{ file *traceFile }

// newMouseTracer returns a nil tracer when the path is empty, and an ERROR when
// a path was given and could not be opened. The path is a PARAMETER, never read
// from env here, for the reason newInputTracer documents at length: a
// constructor that reaches for ambient env opens a real file per Console and
// leaks fds through every test.
func newMouseTracer(path string) (*mouseTracer, error) {
	file, err := openTraceFile("COUCH_MOUSE_TRACE", path)
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
