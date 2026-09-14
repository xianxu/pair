package couchtty

import (
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMouseTracerOffIsNilAndRecordIsNilSafe(t *testing.T) {
	tracer, err := newMouseTracer("")
	if err != nil || tracer != nil {
		t.Fatalf("newMouseTracer(\"\") = %v, %v; want nil, nil (off)", tracer, err)
	}
	tracer.record("child-mode", "none -> 1002") // must not panic on a nil tracer
}

func TestMouseTracerRecordsOneLinePerEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mouse.log")
	tracer, err := newMouseTracer(path)
	if err != nil || tracer == nil {
		t.Fatalf("newMouseTracer(path) = %v, %v", tracer, err)
	}
	tracer.record("child-mode", "1002,1006 -> none")
	tracer.record("assert-clicks", "host-before=none child-mouse=false child-observed=true")
	if err := tracer.Close(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("trace has %d lines, want 2:\n%s", len(lines), body)
	}
	if !strings.Contains(lines[0], "\tchild-mode\t1002,1006 -> none") {
		t.Fatalf("line 0 = %q", lines[0])
	}
	if !strings.Contains(lines[1], "\tassert-clicks\thost-before=none") {
		t.Fatalf("line 1 = %q", lines[1])
	}
}

func TestFormatMouseModes(t *testing.T) {
	for _, tt := range []struct {
		in   []int
		want string
	}{
		{nil, "none"},
		{[]int{1002}, "1002"},
		{[]int{1002, 1006}, "1002,1006"},
	} {
		if got := formatMouseModes(tt.in); got != tt.want {
			t.Fatalf("formatMouseModes(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// mouseTraceHost captures exactly the bytes its controlled write result accepts.
// A barrier allows changing active identity while a production write is pending.
type mouseTraceHost struct {
	*hostty.FakeHost
	limit   int
	failure error
	entered chan struct{}
	resume  chan struct{}
}

func (h *mouseTraceHost) Write(p []byte) (int, error) {
	if h.entered != nil {
		close(h.entered)
		<-h.resume
	}
	n := len(p)
	if h.limit >= 0 && h.limit < n {
		n = h.limit
	}
	_, _ = h.FakeHost.Write(p[:n])
	return n, h.failure
}

func mouseTraceFixture(t *testing.T) (*Console, *mouseTraceHost, func() string) {
	t.Helper()
	h := &mouseTraceHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), limit: -1}
	c := New(h, strings.NewReader(""))
	path := filepath.Join(t.TempDir(), "mouse.log")
	if err := c.SetMouseTrace(path); err != nil {
		t.Fatal(err)
	}
	child := ptychild.NewFakeChild(nil)
	c.Attach("first", "first", child)
	t.Cleanup(func() { c.Stop(); child.Close(); c.workers.Wait(); _ = c.mouseTrace.Close() })
	return c, h, func() string {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
}

func requireMouseTrace(t *testing.T, log string, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if !strings.Contains(log, field) {
			t.Errorf("missing %q in trace:\n%s", field, log)
		}
	}
}

func TestMouseTraceProducerTransitions(t *testing.T) {
	c, h, log := mouseTraceFixture(t)
	c.writeChild([]byte("\x1b[?1002;1006h"))
	requireMouseTrace(t, log(), "\tchild-mode\t", `active="first"`, `thread="legacy/first"`, "scanner-before=none", "scanner-after=1002,1006", "outcome=emitted")
	c.takeOverScreen(nil, nil)
	requireMouseTrace(t, log(), "\ttakeover\t", "scanner-before=1002,1006", "scanner-reset=none", "scanner-after=none", "replay-bytes=0", "target=panel")
	if h.Written() != "\x1b[?1002;1006h"+hostty.EnableKeyboardDisambiguation+string(hostty.RepaintFor(nil, nil))+hostty.EnableKeyboardDisambiguation {
		t.Fatal("tracing altered output")
	}
}

func TestMouseTraceAssertionOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name         string
		limit        int
		err          error
		prefix, want string
	}{
		{"emitted", -1, nil, "", "emitted"},
		{"deferred", -1, nil, "\x1b[", "deferred"},
		{"short", 2, nil, "", "short-write"},
		{"error", 0, errors.New("write\nfailed"), "", "error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, h, log := mouseTraceFixture(t)
			c.activeChild().Feed([]byte("\x1b[?1002l"))
			if tc.prefix != "" {
				c.writeChild([]byte(tc.prefix))
			}
			h.Reset()
			h.limit = tc.limit
			h.failure = tc.err
			c.paintNow()
			requireMouseTrace(t, log(), "\tassert-clicks\t", "source=paint", "scanner-before=none", "outcome="+tc.want, `active="first"`)
			if strings.Contains(log(), "host-before=") {
				t.Fatal("scanner belief still called host state")
			}
			if tc.want == "deferred" && h.Written() != "" {
				t.Fatal("deferred assertion wrote bytes")
			}
			if tc.err != nil {
				requireMouseTrace(t, log(), `error="write\nfailed"`)
			}
		})
	}
}

func TestMouseTraceWriteAttributionBeforeBlockedOutput(t *testing.T) {
	c, h, log := mouseTraceFixture(t)
	h.entered = make(chan struct{})
	h.resume = make(chan struct{})
	done := make(chan struct{})
	go func() { c.writeChild([]byte("\x1b[?1002h")); close(done) }()
	<-h.entered
	c.mu.Lock()
	c.active = "second"
	c.mu.Unlock()
	close(h.resume)
	<-done
	requireMouseTrace(t, log(), `active="first"`, `thread="legacy/first"`, "outcome=emitted")
	if strings.Contains(log(), `active="second"`) {
		t.Fatal("write attributed to later active thread")
	}
}

func TestMouseTraceLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mouse.log")
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
		if err := c.SetMouseTrace(path); err != nil {
			t.Fatal(err)
		}
	})
	waitFor(t, "startup bytes", func() bool { return strings.Contains(f.host.Written(), hostty.EnableMouseClicks) })
	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not stop")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	requireMouseTrace(t, string(body), "source=startup", "\tcleanup\t", "outcome=emitted")
	if !strings.Contains(f.host.Written(), hostty.ResetInteractiveModes) {
		t.Fatal("cleanup bytes missing")
	}
}

func TestMouseTraceReplayAttributionAcrossThreads(t *testing.T) {
	c, _, log := mouseTraceFixture(t)
	c.writeChild([]byte("\x1b[?1002;1006h"))
	other := ptychild.NewFakeChild(nil)
	t.Cleanup(func() { other.Close() })
	other.Feed([]byte("\x1b[?1003;1006h"))
	c.attachThreadActor("second", "second-actor", couchcore.ThreadAddress{RepoScope: "repo", Tag: "second-thread"}, "second", "second", other)
	c.switchTo("second", false, arrivalOrdinary)
	lines := strings.Split(log(), "\n")
	var takeover string
	for _, line := range lines {
		if strings.Contains(line, "\ttakeover\t") {
			takeover = line
		}
	}
	requireMouseTrace(t, takeover, `active="second"`, `actor="second-actor"`, `thread="repo/second-thread"`, `target="second"`, "scanner-before=1002,1006", "scanner-reset=none", "scanner-after=1003,1006", "outcome=emitted")
}

func TestMouseTraceProducerWriteFailures(t *testing.T) {
	for _, producer := range []string{"child", "takeover", "cleanup"} {
		for _, failed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/error=%t", producer, failed), func(t *testing.T) {
				c, h, log := mouseTraceFixture(t)
				h.limit = 2
				want := "short-write"
				if failed {
					h.failure = errors.New("partial failure")
					want = "error"
				}
				switch producer {
				case "child":
					c.writeChild([]byte("\x1b[?1002h"))
				case "takeover":
					c.takeOverScreen(nil, []byte("\x1b[?1002h"))
				case "cleanup":
					c.release()
				}
				requireMouseTrace(t, log(), "outcome="+want, "written=2")
				if len(h.Written()) != 2 {
					t.Fatalf("accepted %d bytes", len(h.Written()))
				}
			})
		}
	}
}

func TestMouseTraceDeferredAssertionEventuallyEmits(t *testing.T) {
	c, h, log := mouseTraceFixture(t)
	c.activeChild().Feed([]byte("\x1b[?1002l"))
	c.writeChild([]byte("\x1b["))
	h.Reset()
	c.paintNow()
	requireMouseTrace(t, log(), "outcome=deferred written=0")
	if h.Written() != "" {
		t.Fatal("deferred write emitted")
	}
	c.writeChild([]byte("m"))
	h.Reset()
	c.paintNow()
	requireMouseTrace(t, log(), "outcome=emitted written="+strconv.Itoa(len(hostty.EnableMouseClicks)))
	if !strings.HasPrefix(h.Written(), hostty.EnableMouseClicks) {
		t.Fatal("later repaint did not emit clicks")
	}
}

func TestMouseWriteResultDetail(t *testing.T) {
	for _, tc := range []struct {
		result mouseWriteResult
		want   string
	}{
		{mouseWriteResult{n: 8}, "emitted"},
		{mouseWriteResult{n: 7}, "short-write"},
		{mouseWriteResult{n: 8, err: errors.New("failure")}, "error"},
		{mouseWriteResult{deferred: true}, "deferred"},
	} {
		requireMouseTrace(t, tc.result.detail(8), "outcome="+tc.want)
	}
}

func TestMouseTraceContextBoundsAndQuoting(t *testing.T) {
	c, _, _ := mouseTraceFixture(t)
	c.mu.Lock()
	p := c.panes[c.active]
	p.actorID = couchcore.ActorID(strings.Repeat("\n\x00", 300))
	p.thread.RepoScope = strings.Repeat("\n\t", 300)
	p.thread.Tag = "tag\nforged"
	c.focus = FocusPanel()
	_, detail := c.mouseTraceContextLocked()
	c.mu.Unlock()
	requireMouseTrace(t, detail, "surface=panel", `\n`)
	if strings.ContainsAny(detail, "\n\t\x00") || len(detail) > 4096 {
		t.Fatalf("unsafe or unbounded detail %q", detail)
	}
}

func TestMouseTraceReleasedAssertion(t *testing.T) {
	c, h, log := mouseTraceFixture(t)
	c.release()
	before := h.Written()
	c.traceMouseClicks("paint")
	requireMouseTrace(t, log(), "outcome=suppressed", "reason=terminal-released")
	if h.Written() != before {
		t.Fatal("mouse assertion wrote after terminal release")
	}
}
