package termcmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestRunTestShortcutRightTerminalActions(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal-filler","terminal_command":"tail -f /dev/null"},
		{"id":4,"is_focused":true,"is_floating":true,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
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
		{name: "alt x routes quit to draft", chord: "Alt+x", wantOps: []string{
			"focus-pane-id 2",
			"write --pane-id 2 28",
			"write --pane-id 2 14",
			"write-chars --pane-id 2 :lua PairConfirmQuit()",
			"write --pane-id 2 13",
		}},
		{name: "alt j swallowed", chord: "Alt+j"},
		{name: "alt k last left", chord: "Alt+k", last: "1", wantOps: []string{"focus-pane-id 1"}},
		{name: "alt k draft fallback", chord: "Alt+k", wantOps: []string{"focus-pane-id 2"}},
		{name: "alt shift enter changes floating geometry once", chord: "Alt+Shift+Enter", wantOps: []string{
			"change-floating-pane-coordinates --pane-id 4 --x 37 --y 0 --width 113 --height 51 --borderless false --pinned true",
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
		{name: "rename tab", chunks: [][]byte{{0x1b, 'r'}, []byte("work\r")}, wantMux: "rename-begin:,rename-preview:w:1,rename-preview:wo:2,rename-preview:wor:3,rename-preview:work:4,rename-finish:1:work"},
		{name: "previous tab", chunks: [][]byte{[]byte("\x1b[1;3D")}, wantMux: "prev-tab"},
		{name: "next tab", chunks: [][]byte{[]byte("\x1b[1;3C")}, wantMux: "next-tab"},
		{name: "alt d routes detach to draft", chunks: [][]byte{[]byte("\x1b[100;3u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmDetach(),write --pane-id 2 13"},
		{name: "alt x routes quit to draft", chunks: [][]byte{[]byte("\x1b[120;3u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmQuit(),write --pane-id 2 13"},
		{name: "alt n routes restart to draft", chunks: [][]byte{[]byte("\x1b[110;3u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmRestart(),write --pane-id 2 13"},
		{name: "ctrl alt n routes restart to draft", chunks: [][]byte{[]byte("\x1b[110;7u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmRestart(),write --pane-id 2 13"},
		{name: "shift alt n routes agent restart to draft", chunks: [][]byte{[]byte("\x1b[78;4u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmAgentRestart(),write --pane-id 2 13"},
		{name: "alt up routes grow to draft", chunks: [][]byte{[]byte("\x1b[1;3A")}, wantRTOps: "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairLayoutBigger(),write --pane-id 2 13"},
		{name: "alt down routes shrink to draft", chunks: [][]byte{[]byte("\x1b[1;3B")}, wantRTOps: "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairLayoutSmaller(),write --pane-id 2 13"},
		{name: "alt c routes review toggle to draft", chunks: [][]byte{[]byte("\x1b[99;3u")}, wantRTOps: "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairReviewToggle(),write --pane-id 2 13"},
		{name: "layout toggle", chunks: [][]byte{[]byte("\x1b[13;4u")}, wantRTOps: "change-floating-pane-coordinates --pane-id 4 --x 37 --y 0 --width 113 --height 51 --borderless false --pinned true"},
		{name: "mouse top row passes to child", chunks: [][]byte{[]byte("\x1b[<0;8;1M")}, wantMux: "write:\x1b[<0;8;1M"},
		{name: "mouse shell row passes through", chunks: [][]byte{[]byte("\x1b[<0;8;2M")}, wantMux: "write:\x1b[<0;8;2M"},
		{name: "mouse wheel up scrolls zellij viewport", chunks: [][]byte{[]byte("\x1b[<64;8;5M")}, wantRTOps: "scroll-up"},
		{name: "mouse wheel down scrolls zellij viewport", chunks: [][]byte{[]byte("\x1b[<65;8;5M")}, wantRTOps: "scroll-down"},
		{name: "mouse wheel passes through when app enabled mouse", chunks: [][]byte{[]byte("\x1b[<64;8;5M")}, appMouse: true, wantMux: "write:\x1b[<64;8;5M"},
		{name: "plain bytes", chunks: [][]byte{[]byte("ls\n")}, wantMux: "write:ls\n"},
		{name: "shortcut then payload in one read", chunks: [][]byte{[]byte("\x1btls\n")}, wantMux: "new-tab,write:ls\n"},
		{name: "payload then shortcut in one read", chunks: [][]byte{[]byte("ls\n\x1bt")}, wantMux: "write:ls\n,new-tab"},
		{name: "mouse wheel then payload in one read", chunks: [][]byte{[]byte("\x1b[<64;8;5Mls\n")}, wantMux: "write:ls\n", wantRTOps: "scroll-up"},
		{name: "payload then mouse wheel in one read", chunks: [][]byte{[]byte("ls\n\x1b[<64;8;5M")}, wantMux: "write:ls\n", wantRTOps: "scroll-up"},
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

func TestPumpStdinReportsFocusFailureWithoutWriting(t *testing.T) {
	rt := &fakeRuntime{cachedDraft: "2", failFocus: true}
	mux := &fakeMux{}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[110;3u")}}, mux, rt, io.Discard)
	if got := strings.Join(rt.ops, ","); got != "focus-pane-id 2" {
		t.Fatalf("runtime ops = %q, want focus only", got)
	}
	if len(rt.reported) != 1 || !strings.Contains(rt.reported[0], "focus") {
		t.Fatalf("reported = %v, want focus error", rt.reported)
	}
}

func TestPumpStdinConsumesGlobalChordWhenDraftMissing(t *testing.T) {
	rt := &fakeRuntime{panesJSON: `[
		{"id":4,"is_focused":true,"is_floating":true,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
	]`}
	mux := &fakeMux{}

	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[110;3u")}}, mux, rt, io.Discard)

	if len(mux.ops) != 0 {
		t.Fatalf("mux ops = %v, want recognized chord consumed", mux.ops)
	}
	if len(rt.reported) != 1 || !strings.Contains(rt.reported[0], "draft pane") {
		t.Fatalf("reported = %v, want missing draft pane error", rt.reported)
	}
}

func TestPumpStdinRenameCommitsInFrameWithoutChildPrompt(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{activeName: "work"}
	var stdout bytes.Buffer

	pumpStdin(&splitReader{chunks: [][]byte{
		[]byte("\x1br"),
		[]byte("界\r"),
		[]byte("ls\n"),
	}}, mux, rt, &stdout)

	want := "rename-begin:work,rename-preview:work界:5,rename-finish:1:work界,write:ls\n"
	if got := strings.Join(mux.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no content-area prompt", stdout.String())
	}
}

func TestPumpStdinRenameConsumesSameReadSuffix(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{activeName: "work"}

	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1brx\rls\n")}}, mux, rt, io.Discard)

	want := "rename-begin:work,rename-preview:workx:5,rename-finish:1:workx"
	if got := strings.Join(mux.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
}

func TestPumpStdinRenameCancelsOnEOF(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{activeName: "work"}

	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1br"), []byte("x")}}, mux, rt, io.Discard)

	want := "rename-begin:work,rename-preview:workx:5,rename-finish:2:work"
	if got := strings.Join(mux.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
}

func TestPumpStdinRenameEntryFailureConsumesInput(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{activeName: "work", beginRenameErr: exec.ErrNotFound}

	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1brx\r")}}, mux, rt, io.Discard)

	if got := strings.Join(mux.ops, ","); got != "rename-begin:work" {
		t.Fatalf("ops = %q, want failed begin only", got)
	}
	if len(rt.reported) != 1 {
		t.Fatalf("reported = %v, want one rename error", rt.reported)
	}
}

func TestPumpStdinRenameRefreshAndFinishFailuresPreserveOutcome(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{
		activeName:       "work",
		refreshRenameErr: exec.ErrNotFound,
		finishRenameErr:  exec.ErrNotFound,
	}

	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1brx\r")}}, mux, rt, io.Discard)

	if mux.activeName != "workx" {
		t.Fatalf("active name = %q, want committed workx", mux.activeName)
	}
	if len(rt.reported) != 2 {
		t.Fatalf("reported = %v, want refresh and finish errors", rt.reported)
	}
}

func TestPumpStdinRenameConsumesShortcutMouseAndPaste(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{activeName: "work"}
	input := "\x1br\x1b[110;3u\x1b[<0;3;2M\x1b[200~hidden\x1b[201~\r"

	pumpStdin(&splitReader{chunks: [][]byte{[]byte(input)}}, mux, rt, io.Discard)

	want := "rename-begin:work,rename-finish:1:work"
	if got := strings.Join(mux.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
}

func TestPumpStdinRenameBareEscapeCancelsOnTimer(t *testing.T) {
	rt := &fakeRuntime{}
	finished := make(chan RenameOutcome, 1)
	mux := &fakeMux{activeName: "work", renameFinished: finished}
	reader := &gatedEOFReader{data: []byte("\x1br\x1b"), release: make(chan struct{})}
	timer := newFiringRenameTimer()
	done := make(chan struct{})

	go func() {
		pumpStdinWithTimer(reader, mux, rt, io.Discard, timer)
		close(done)
	}()

	select {
	case outcome := <-finished:
		if outcome.Kind != RenameOutcomeCancel || outcome.Name != "work" {
			t.Fatalf("outcome = %#v, want cancel work", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("rename timer did not cancel")
	}
	close(reader.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stdin pump did not finish after EOF")
	}
}

func TestPumpStdinRenameEscapeContinuationBeatsTimer(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{activeName: "work"}
	timer := newFiringRenameTimer()
	timer.autoFire = false

	pumpStdinWithTimer(&splitReader{chunks: [][]byte{
		[]byte("\x1br\x1b"),
		[]byte("[D"),
		[]byte("\r"),
	}}, mux, rt, io.Discard, timer)

	want := "rename-begin:work,rename-preview:work:3,rename-finish:1:work"
	if got := strings.Join(mux.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
	if timer.resets == 0 || timer.stops == 0 {
		t.Fatalf("timer resets=%d stops=%d, want both exercised", timer.resets, timer.stops)
	}
}

func TestRenamePaneTitlePlacesCursorInActiveFrameField(t *testing.T) {
	mux := &terminalMux{
		tabs: []*terminalTab{
			{name: "terminal 1"},
			{name: "work"},
			{name: "terminal 3"},
		},
		active: 1,
	}
	editor := NewRenameEditor("work")
	editor, _ = editor.Apply(RenameEvent{Kind: RenameMoveLeft})
	if got := mux.renamePaneTitleLocked(editor); got != "terminal 1 [rename: wor│k] terminal 3" {
		t.Fatalf("rename title = %q", got)
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

func TestTerminalMuxChildUsesFullPaneHeight(t *testing.T) {
	mux := &terminalMux{rows: 51, cols: 80}
	got := mux.childSizeLocked()
	if got.Rows != 51 || got.Cols != 80 {
		t.Fatalf("child size = %+v, want full 51x80 pane", got)
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
	}
	mux.nextTab()
	if mux.active != 1 {
		t.Fatalf("active = %d, want 1", mux.active)
	}
	if !strings.Contains(strings.Join(rt.ops, ","), "rename-pane terminal 1 [work]") {
		t.Fatalf("ops = %v", rt.ops)
	}
	if !strings.Contains(stdout.String(), "two") {
		t.Fatalf("stdout = %q, want redraw of second tab", stdout.String())
	}
	if strings.Contains(stdout.String(), "\x1b[7m") {
		t.Fatalf("stdout contains obsolete inverse-video tab strip: %q", stdout.String())
	}
}

func TestTerminalMuxBackgroundExitPreservesActiveTab(t *testing.T) {
	pty1, peer1, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer peer1.Close()
	pty2, peer2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer peer2.Close()
	defer pty2.Close()
	pty3, peer3, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer peer3.Close()
	defer pty3.Close()

	mux := &terminalMux{
		stdout: io.Discard,
		rt:     &fakeRuntime{},
		done:   make(chan struct{}),
		tabs: []*terminalTab{
			{id: 1, name: "one", cmd: exec.Command("true"), pty: pty1},
			{id: 2, name: "two", cmd: exec.Command("true"), pty: pty2},
			{id: 3, name: "three", cmd: exec.Command("true"), pty: pty3},
		},
		active: 1,
	}

	mux.removeTab(1)

	if got := mux.activeTabLocked(); got == nil || got.id != 2 {
		t.Fatalf("active tab after background exit = %+v, want id 2", got)
	}
}

type stdoutWriter struct {
	*bytes.Buffer
}

type fakeRuntime struct {
	panesJSON   string
	cachedDraft string
	lastLeft    string
	ops         []string
	reported    []string
	failFocus   bool
}

func (f *fakeRuntime) CachedDraftPaneID() (string, bool) {
	return f.cachedDraft, f.cachedDraft != ""
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	if f.panesJSON == "" {
		return []byte(`[
			{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
			{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
			{"id":3,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal-filler","terminal_command":"tail -f /dev/null"},
			{"id":4,"is_focused":true,"is_floating":true,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
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
	if f.failFocus && len(args) > 0 && args[0] == "focus-pane-id" {
		return exec.ErrNotFound
	}
	return nil
}

func (f *fakeRuntime) ReportShortcutError(err error) {
	f.reported = append(f.reported, err.Error())
}

func (f *fakeRuntime) ShellCommand() (string, []string) {
	return "/bin/sh", []string{"-i"}
}

type fakeMux struct {
	ops              []string
	appMouse         bool
	activeName       string
	beginRenameErr   error
	refreshRenameErr error
	finishRenameErr  error
	renameFinished   chan RenameOutcome
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

func (f *fakeMux) beginRename() (RenameEditor, error) {
	f.ops = append(f.ops, "rename-begin:"+f.activeName)
	return NewRenameEditor(f.activeName), f.beginRenameErr
}

func (f *fakeMux) refreshRename(editor RenameEditor) error {
	f.ops = append(f.ops, fmt.Sprintf("rename-preview:%s:%d", editor.Text(), editor.Cursor()))
	return f.refreshRenameErr
}

func (f *fakeMux) finishRename(outcome RenameOutcome) error {
	f.ops = append(f.ops, fmt.Sprintf("rename-finish:%d:%s", outcome.Kind, outcome.Name))
	if outcome.Kind == RenameOutcomeCommit {
		f.activeName = outcome.Name
	}
	if f.renameFinished != nil {
		f.renameFinished <- outcome
	}
	return f.finishRenameErr
}

func (f *fakeMux) previousTab() {
	f.ops = append(f.ops, "prev-tab")
}

func (f *fakeMux) nextTab() {
	f.ops = append(f.ops, "next-tab")
}

func (f *fakeMux) appMouseMode() bool {
	return f.appMouse
}

type splitReader struct {
	chunks [][]byte
}

type gatedEOFReader struct {
	data    []byte
	sent    bool
	release chan struct{}
}

func (r *gatedEOFReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, r.data), nil
	}
	<-r.release
	return 0, io.EOF
}

type firingRenameTimer struct {
	ch       chan time.Time
	autoFire bool
	resets   int
	stops    int
}

func newFiringRenameTimer() *firingRenameTimer {
	return &firingRenameTimer{ch: make(chan time.Time, 1), autoFire: true}
}

func (t *firingRenameTimer) C() <-chan time.Time {
	return t.ch
}

func (t *firingRenameTimer) Reset(time.Duration) {
	t.resets++
	if !t.autoFire {
		return
	}
	select {
	case t.ch <- time.Now():
	default:
	}
}

func (t *firingRenameTimer) StopAndDrain() {
	t.stops++
	select {
	case <-t.ch:
	default:
	}
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
