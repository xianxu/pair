package couchtty

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func (h *mouseTraceHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return h.Write(p)
}

func TestMouseTraceRecordsTypedSelectionAndRelease(t *testing.T) {
	c, _, log := mouseTraceFixture(t)
	c.applyLayout()
	c.switchTo("first", true, arrivalOrdinary)
	c.release()
	requireMouseTrace(t, log(), "\tselect\t", `active="first"`, "policy=couch-any-motion", "outcome=presented", "\trelease\t")
}

func TestMouseTraceRecordsPresentationFailure(t *testing.T) {
	c, h, log := mouseTraceFixture(t)
	c.applyLayout()
	h.failure = errors.New("host rejected frame")
	_, err := c.selectActor("first", true, arrivalOrdinary)
	if err == nil {
		t.Fatal("host failure was hidden")
	}
	requireMouseTrace(t, log(), "outcome=failed", "host rejected frame")
}
