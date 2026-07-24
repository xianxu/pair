package termcmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestRunTestShortcutRightTerminalActions(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_focused":true,"is_floating":false,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
	]`
	tests := []struct {
		name    string
		chord   string
		last    string
		wantOps []string
	}{
		{name: "new tab stays local", chord: "Alt+t"},
		{name: "close tab stays local", chord: "Alt+w"},
		{name: "rename tab stays local", chord: "Alt+r"},
		{name: "alt j swallowed", chord: "Alt+j"},
		{name: "alt k last left", chord: "Alt+k", last: "1", wantOps: []string{"focus-pane-id 1"}},
		{name: "alt k draft fallback", chord: "Alt+k", wantOps: []string{"focus-pane-id 2"}},
		{name: "alt shift enter floats terminal", chord: "Alt+Shift+Enter", wantOps: []string{
			"toggle-pane-embed-or-floating --pane-id 3",
			"change-floating-pane-coordinates --pane-id 3 --x 33% --y 0% --width 67% --height 100% --borderless true --pinned true",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &fakeRuntime{panesJSON: panes, lastLeft: tt.last}
			var stderr bytes.Buffer
			stdin := ""
			if tt.chord == "Alt+r" {
				stdin = "work\n"
			}
			code := RunWithRuntime([]string{"--test-shortcut", tt.chord}, strings.NewReader(stdin), &bytes.Buffer{}, &stderr, rt)
			if code != 0 {
				t.Fatalf("code = %d stderr=%q", code, stderr.String())
			}
			if strings.Join(rt.ops, ",") != strings.Join(tt.wantOps, ",") {
				t.Fatalf("ops = %v, want %v", rt.ops, tt.wantOps)
			}
		})
	}
}

func TestRunTestShortcutIgnoresNonTerminalPane(t *testing.T) {
	panes := `[
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":4,"is_focused":true,"is_floating":true,"is_plugin":false,"title":"review","terminal_command":"nvim -u /pair/nvim/review.lua /tmp/review.md"}
	]`
	rt := &fakeRuntime{panesJSON: panes}
	var stderr bytes.Buffer
	code := RunWithRuntime([]string{"--test-shortcut", "Alt+r"}, strings.NewReader(""), &bytes.Buffer{}, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want none", rt.ops)
	}
}

func TestRunTestShortcutRecordsLeftPane(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":true,"is_floating":false,"is_plugin":false,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":3,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
	]`
	rt := &fakeRuntime{panesJSON: panes}
	var stderr bytes.Buffer
	code := RunWithRuntime([]string{"--test-shortcut", "Alt+k"}, strings.NewReader(""), &bytes.Buffer{}, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if rt.lastLeft != "1" {
		t.Fatalf("lastLeft = %q, want 1", rt.lastLeft)
	}
	if strings.Join(rt.ops, ",") != "focus-pane-id 3" {
		t.Fatalf("ops = %v, want focus terminal", rt.ops)
	}
}

func TestRunTestShortcutIgnoresLeftLayoutToggle(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":true,"is_floating":false,"is_plugin":false,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
	]`
	rt := &fakeRuntime{panesJSON: panes}
	var stderr bytes.Buffer
	code := RunWithRuntime([]string{"--test-shortcut", "Alt+Shift+Enter"}, strings.NewReader(""), &bytes.Buffer{}, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want none", rt.ops)
	}
}

func TestPumpStdinDecodesSplitAltChord(t *testing.T) {
	rt := &fakeRuntime{}
	stdin := splitReader{chunks: [][]byte{{0x1b}, {'t'}}}
	mux := &fakeMux{}

	pumpStdin(&stdin, mux, rt, &bytes.Buffer{})

	if strings.Join(mux.ops, ",") != "new-tab" {
		t.Fatalf("mux ops = %v, want new-tab", mux.ops)
	}
}

func TestPumpStdinHandlesTerminalTabActions(t *testing.T) {
	tests := []struct {
		name      string
		chunks    [][]byte
		appMouse  bool
		wantMux   string
		wantRTOps string
	}{
		{name: "new tab", chunks: [][]byte{{0x1b, 't'}}, wantMux: "new-tab"},
		{name: "close tab", chunks: [][]byte{{0x1b, 'w'}}, wantMux: "close-tab"},
		{name: "rename tab", chunks: [][]byte{{0x1b, 'r'}, []byte("work\r")}, wantMux: "rename:work"},
		{name: "previous tab", chunks: [][]byte{[]byte("\x1b[1;3D")}, wantMux: "prev-tab"},
		{name: "next tab", chunks: [][]byte{[]byte("\x1b[1;3C")}, wantMux: "next-tab"},
		{name: "layout toggle", chunks: [][]byte{[]byte("\x1b[13;4u")}, wantRTOps: "toggle-pane-embed-or-floating --pane-id 3,change-floating-pane-coordinates --pane-id 3 --x 33% --y 0% --width 67% --height 100% --borderless true --pinned true"},
		{name: "mouse top row", chunks: [][]byte{[]byte("\x1b[<0;8;1M")}, wantMux: "switch-at:8"},
		{name: "mouse shell row passes through", chunks: [][]byte{[]byte("\x1b[<0;8;2M")}, wantMux: "write:\x1b[<0;8;2M"},
		{name: "mouse wheel up scrolls zellij viewport", chunks: [][]byte{[]byte("\x1b[<64;8;5M")}, wantRTOps: "scroll-up"},
		{name: "mouse wheel down scrolls zellij viewport", chunks: [][]byte{[]byte("\x1b[<65;8;5M")}, wantRTOps: "scroll-down"},
		{name: "mouse wheel passes through when app enabled mouse", chunks: [][]byte{[]byte("\x1b[<64;8;5M")}, appMouse: true, wantMux: "write:\x1b[<64;8;5M"},
		{name: "plain bytes", chunks: [][]byte{[]byte("ls\n")}, wantMux: "write:ls\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &fakeRuntime{}
			mux := &fakeMux{appMouse: tt.appMouse}
			var stdout bytes.Buffer
			pumpStdin(&splitReader{chunks: tt.chunks}, mux, rt, &stdout)
			if strings.Join(mux.ops, ",") != tt.wantMux {
				t.Fatalf("mux ops = %q, want %q", strings.Join(mux.ops, ","), tt.wantMux)
			}
			if strings.Join(rt.ops, ",") != tt.wantRTOps {
				t.Fatalf("runtime ops = %q, want %q", strings.Join(rt.ops, ","), tt.wantRTOps)
			}
		})
	}
}

func TestParseSGRMousePress(t *testing.T) {
	event, ok := parseSGRMousePress([]byte("\x1b[<64;12;1M"))
	if !ok || event.button != 64 || event.x != 12 || event.y != 1 {
		t.Fatalf("mouse = (%+v,%v), want ({button:64 x:12 y:1},true)", event, ok)
	}
	if _, ok := parseSGRMousePress([]byte("\x1b[<0;12;1m")); ok {
		t.Fatal("release event should not parse as press")
	}
}

func TestUpdateMouseMode(t *testing.T) {
	tests := []struct {
		name  string
		start bool
		data  []byte
		want  bool
	}{
		{name: "enable basic mouse", data: []byte("\x1b[?1000h"), want: true},
		{name: "enable sgr mouse", data: []byte("\x1b[?1006h"), want: true},
		{name: "enable multiple modes", data: []byte("\x1b[?1000;1006h"), want: true},
		{name: "disable mouse", start: true, data: []byte("\x1b[?1000l"), want: false},
		{name: "unrelated private mode preserves state", start: true, data: []byte("\x1b[?25l"), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := updateMouseMode(tt.start, tt.data); got != tt.want {
				t.Fatalf("updateMouseMode(%v, %q) = %v, want %v", tt.start, tt.data, got, tt.want)
			}
		})
	}
}

func TestTerminalMuxPaneTitleShowsTabs(t *testing.T) {
	mux := &terminalMux{
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1"},
			{id: 2, name: "work"},
			{id: 3, name: "terminal 3"},
		},
		active: 1,
	}
	if got := mux.paneTitleLocked(); got != "terminal 1 [work] terminal 3" {
		t.Fatalf("pane title = %q", got)
	}
}

func TestTerminalMuxTabStripRanges(t *testing.T) {
	mux := &terminalMux{
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1"},
			{id: 2, name: "work"},
		},
		active: 0,
	}
	line, ranges := mux.tabStripLocked(40)
	if !strings.Contains(line, "[1:terminal 1]") || !strings.Contains(line, " 2:work ") {
		t.Fatalf("line = %q", line)
	}
	if len(ranges) != 2 {
		t.Fatalf("ranges = %+v", ranges)
	}
	if ranges[0].start != 1 || ranges[0].index != 0 || ranges[1].index != 1 {
		t.Fatalf("ranges = %+v", ranges)
	}
}

func TestTerminalMuxSwitchTabAtColumn(t *testing.T) {
	var stdout bytes.Buffer
	rt := &fakeRuntime{}
	mux := &terminalMux{
		stdout: stdoutWriter{&stdout},
		rt:     rt,
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1", buffer: []byte("one")},
			{id: 2, name: "work", buffer: []byte("two")},
		},
		active: 0,
		cols:   40,
		ranges: []tabRange{{start: 1, end: 14, index: 0}, {start: 16, end: 23, index: 1}},
	}
	mux.switchTabAtColumn(18)
	if mux.active != 1 {
		t.Fatalf("active = %d, want 1", mux.active)
	}
	if !strings.Contains(strings.Join(rt.ops, ","), "rename-pane terminal 1 [work]") {
		t.Fatalf("ops = %v", rt.ops)
	}
	if !strings.Contains(stdout.String(), "two") {
		t.Fatalf("stdout = %q, want redraw of second tab", stdout.String())
	}
}

type stdoutWriter struct {
	*bytes.Buffer
}

type fakeRuntime struct {
	panesJSON string
	lastLeft  string
	ops       []string
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	if f.panesJSON == "" {
		return []byte(`[
			{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"codex","terminal_command":"pair wrap codex"},
			{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
			{"id":3,"is_focused":true,"is_floating":false,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
		]`), nil
	}
	return []byte(f.panesJSON), nil
}

func (f *fakeRuntime) LastLeftPaneID() (string, error) {
	return f.lastLeft, nil
}

func (f *fakeRuntime) RecordLastLeftPaneID(id string) error {
	f.lastLeft = id
	return nil
}

func (f *fakeRuntime) RunZellijAction(args ...string) error {
	f.ops = append(f.ops, strings.Join(args, " "))
	return nil
}

func (f *fakeRuntime) ShellCommand() (string, []string) {
	return "/bin/sh", []string{"-i"}
}

type fakeMux struct {
	ops      []string
	appMouse bool
}

func (f *fakeMux) writeActive(data []byte) {
	f.ops = append(f.ops, "write:"+string(data))
}

func (f *fakeMux) newTab() error {
	f.ops = append(f.ops, "new-tab")
	return nil
}

func (f *fakeMux) closeActive() {
	f.ops = append(f.ops, "close-tab")
}

func (f *fakeMux) renameActive(name string) {
	f.ops = append(f.ops, "rename:"+name)
}

func (f *fakeMux) previousTab() {
	f.ops = append(f.ops, "prev-tab")
}

func (f *fakeMux) nextTab() {
	f.ops = append(f.ops, "next-tab")
}

func (f *fakeMux) switchTabAtColumn(x int) {
	f.ops = append(f.ops, fmt.Sprintf("switch-at:%d", x))
}

func (f *fakeMux) appMouseMode() bool {
	return f.appMouse
}

type splitReader struct {
	chunks [][]byte
}

func (r *splitReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	chunk := r.chunks[0]
	n := copy(p, chunk)
	if n == len(chunk) {
		r.chunks = r.chunks[1:]
	} else {
		r.chunks[0] = chunk[n:]
	}
	return n, nil
}
