package termcmd

import (
	"bytes"
	"io"
	"os"
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
		{name: "new tab", chord: "Alt+t", wantOps: []string{"new-tab --name terminal --layout-string " + terminalTabLayout}},
		{name: "close tab", chord: "Alt+w", wantOps: []string{"close-tab"}},
		{name: "rename tab", chord: "Alt+r", wantOps: []string{"run --floating --close-on-exit --name rename tab -- pair term rename-tab-prompt"}},
		{name: "alt j swallowed", chord: "Alt+j"},
		{name: "alt k last left", chord: "Alt+k", last: "1", wantOps: []string{"focus-pane-id 1"}},
		{name: "alt k draft fallback", chord: "Alt+k", wantOps: []string{"focus-pane-id 2"}},
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

func TestPumpStdinDecodesSplitAltChord(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_focused":true,"is_floating":false,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
	]`
	rt := &fakeRuntime{panesJSON: panes}
	stdin := splitReader{chunks: [][]byte{{0x1b}, {'t'}}}
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer readEnd.Close()
	defer writeEnd.Close()

	pumpStdin(&stdin, writeEnd, rt, &bytes.Buffer{})

	if strings.Join(rt.ops, ",") != "new-tab --name terminal --layout-string "+terminalTabLayout {
		t.Fatalf("ops = %v, want split Alt+t to open terminal tab", rt.ops)
	}
}

func TestRenameTabPromptRenamesFromFloatingPane(t *testing.T) {
	rt := &fakeRuntime{}
	var stdout, stderr bytes.Buffer
	code := RunWithRuntime([]string{"rename-tab-prompt"}, strings.NewReader("work\n"), &stdout, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if strings.Join(rt.ops, ",") != "rename-tab work" {
		t.Fatalf("ops = %v, want rename-tab work", rt.ops)
	}
	if stdout.String() != "tab name: " {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

type fakeRuntime struct {
	panesJSON string
	lastLeft  string
	ops       []string
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
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

type splitReader struct {
	chunks [][]byte
}

func (r *splitReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	chunk := r.chunks[0]
	r.chunks = r.chunks[1:]
	copy(p, chunk)
	return len(chunk), nil
}
