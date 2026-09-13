package couchtty

import (
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
