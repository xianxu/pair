package termcmd

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestRunTestShortcutRightTerminalActions(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":4,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
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
		{name: "alt shift d splits terminal down", chord: "Alt+Shift+d", wantOps: []string{
			`quiet new-pane --direction down --name terminal -- sh -c zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`,
		}},
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
		{name: "alt shift enter uses native fullscreen", chord: "Alt+Shift+Enter", wantOps: []string{
			"toggle-fullscreen --pane-id 4",
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
		{"id":2,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":4,"is_focused":true,"is_fullscreen":false,"is_floating":true,"is_plugin":false,"title":"review","terminal_command":"nvim -u /pair/nvim/review.lua /tmp/review.md"}
	]`
	for _, chord := range []string{"Alt+r", "Alt+Shift+d"} {
		t.Run(chord, func(t *testing.T) {
			rt := &fakeRuntime{panesJSON: panes}
			var stderr bytes.Buffer
			code := RunWithRuntime([]string{"--test-shortcut", chord}, strings.NewReader(""), &bytes.Buffer{}, &stderr, rt)
			if code != 0 {
				t.Fatalf("code = %d stderr=%q", code, stderr.String())
			}
			if len(rt.ops) != 0 {
				t.Fatalf("ops = %v, want none", rt.ops)
			}
		})
	}
}

func TestRunTestShortcutRecordsLeftPane(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua"},
		{"id":3,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
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

func TestRunTestShortcutGlobalLayoutToggle(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
	]`
	rt := &fakeRuntime{panesJSON: panes}
	var stderr bytes.Buffer
	code := RunWithRuntime([]string{"--test-shortcut", "Alt+Shift+Enter"}, strings.NewReader(""), &bytes.Buffer{}, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if len(rt.ops) != 1 || rt.ops[0] != "toggle-fullscreen --pane-id 3" || rt.fullscreenRecord != "1" {
		t.Fatalf("ops = %v, return=%s; want native toggle and agent return", rt.ops, rt.fullscreenRecord)
	}
}

func TestPumpStdinDecodesSplitAltChord(t *testing.T) {
	rt := &fakeRuntime{}
	stdin := splitReader{chunks: [][]byte{{0x1b}, {'t'}}}
	mux := &fakeMux{}
	timer := beforeDeadline()

	pumpStdinWithTimer(&stdin, mux, rt, &bytes.Buffer{}, timer)

	if strings.Join(mux.ops, ",") != "new-tab" {
		t.Fatalf("mux ops = %v, want new-tab", mux.ops)
	}
	// The held ESC armed the deadline once; the tail completing the chord
	// stopped it (#234).
	if timer.resets != 1 || timer.stops == 0 {
		t.Fatalf("timer resets=%d stops=%d, want armed once on the held ESC and stopped on the chord", timer.resets, timer.stops)
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
		{name: "new tab kkp", chunks: [][]byte{[]byte("\x1b[116;3u")}, wantMux: "new-tab"},
		{name: "close tab", chunks: [][]byte{{0x1b, 'w'}}, wantMux: "close-tab"},
		{name: "rename tab", chunks: [][]byte{{0x1b, 'r'}, []byte("work\r")}, wantMux: "rename-begin:,rename-preview:w:1,rename-preview:wo:2,rename-preview:wor:3,rename-preview:work:4,rename-finish:1:work"},
		{name: "previous tab", chunks: [][]byte{[]byte("\x1b[1;3D")}, wantMux: "prev-tab"},
		{name: "next tab", chunks: [][]byte{[]byte("\x1b[1;3C")}, wantMux: "next-tab"},
		{name: "split terminal down as a native tiled split", chunks: [][]byte{[]byte("\x1b[68;4u")}, wantRTOps: `quiet new-pane --direction down --name terminal -- sh -c zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`},
		{name: "alt d routes detach to draft", chunks: [][]byte{[]byte("\x1b[100;3u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmDetach(),write --pane-id 2 13"},
		{name: "alt x routes quit to draft", chunks: [][]byte{[]byte("\x1b[120;3u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmQuit(),write --pane-id 2 13"},
		{name: "alt n routes restart to draft", chunks: [][]byte{[]byte("\x1b[110;3u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmRestart(),write --pane-id 2 13"},
		{name: "ctrl alt n routes restart to draft", chunks: [][]byte{[]byte("\x1b[110;7u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmRestart(),write --pane-id 2 13"},
		{name: "shift alt n routes agent restart to draft", chunks: [][]byte{[]byte("\x1b[78;4u")}, wantRTOps: "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmAgentRestart(),write --pane-id 2 13"},
		{name: "alt up passes through outside draft", chunks: [][]byte{[]byte("\x1b[1;3A")}, wantMux: "write:\x1b[1;3A"},
		{name: "alt down passes through outside draft", chunks: [][]byte{[]byte("\x1b[1;3B")}, wantMux: "write:\x1b[1;3B"},
		{name: "alt c routes review toggle to draft", chunks: [][]byte{[]byte("\x1b[99;3u")}, wantRTOps: "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairReviewToggle(),write --pane-id 2 13"},
		{name: "layout toggle", chunks: [][]byte{[]byte("\x1b[13;4u")}, wantRTOps: "toggle-fullscreen --pane-id 4"},
		{name: "mouse top row passes to child", chunks: [][]byte{[]byte("\x1b[<0;8;1M")}, wantMux: "write:\x1b[<0;8;1M"},
		{name: "mouse shell row passes through", chunks: [][]byte{[]byte("\x1b[<0;8;2M")}, wantMux: "write:\x1b[<0;8;2M"},
		{name: "mouse wheel up scrolls zellij viewport", chunks: [][]byte{[]byte("\x1b[<64;8;5M")}, wantRTOps: "scroll-up"},
		{name: "mouse wheel down scrolls zellij viewport", chunks: [][]byte{[]byte("\x1b[<65;8;5M")}, wantRTOps: "scroll-down"},
		{name: "mouse wheel passes through when app enabled mouse", chunks: [][]byte{[]byte("\x1b[<64;8;5M")}, appMouse: true, wantMux: "write:\x1b[<64;8;5M"},
		// #216: the from-anywhere tab chord. In THIS pane it needs no delivery —
		// it is already here — so it must reach the same mux calls Alt+Left/Right
		// do, and must NOT emit a zellij action (wantRTOps stays empty).
		{name: "alt shift left switches tab in place", chunks: [][]byte{[]byte("\x1b[1;4D")}, wantMux: "prev-tab"},
		{name: "alt shift right switches tab in place", chunks: [][]byte{[]byte("\x1b[1;4C")}, wantMux: "next-tab"},
		// Split arrival: the chord straddles two reads, which is what the `held`
		// buffer exists for. A sequence lost here reads as a dead key.
		{name: "alt shift left split across reads", chunks: [][]byte{[]byte("\x1b[1;4"), []byte("D")}, wantMux: "prev-tab"},
		{name: "alt shift right split across reads", chunks: [][]byte{[]byte("\x1b[1;"), []byte("4C")}, wantMux: "next-tab"},
		// Modifier bits ride IN the button field, so these used to fall to the
		// default arm and write SGR bytes into a child that never asked (#213).
		{name: "ctrl wheel up still scrolls", chunks: [][]byte{[]byte("\x1b[<80;8;5M")}, wantRTOps: "scroll-up"},
		{name: "ctrl wheel down still scrolls", chunks: [][]byte{[]byte("\x1b[<81;8;5M")}, wantRTOps: "scroll-down"},
		{name: "shift wheel up still scrolls", chunks: [][]byte{[]byte("\x1b[<68;8;5M")}, wantRTOps: "scroll-up"},
		{name: "alt wheel down still scrolls", chunks: [][]byte{[]byte("\x1b[<73;8;5M")}, wantRTOps: "scroll-down"},
		// An app that asked for mouse still receives the modifier verbatim:
		// translation is for the child that did NOT ask.
		{name: "ctrl wheel passes through when app enabled mouse", chunks: [][]byte{[]byte("\x1b[<80;8;5M")}, appMouse: true, wantMux: "write:\x1b[<80;8;5M"},
		{name: "plain bytes", chunks: [][]byte{[]byte("ls\n")}, wantMux: "write:ls\n"},
		{name: "shortcut then payload in one read", chunks: [][]byte{[]byte("\x1btls\n")}, wantMux: "new-tab,write:ls\n"},
		{name: "payload then shortcut in one read", chunks: [][]byte{[]byte("ls\n\x1bt")}, wantMux: "write:ls\n,new-tab"},
		{name: "mouse wheel then payload in one read", chunks: [][]byte{[]byte("\x1b[<64;8;5Mls\n")}, wantMux: "write:ls\n", wantRTOps: "scroll-up"},
		{name: "payload then mouse wheel in one read", chunks: [][]byte{[]byte("ls\n\x1b[<64;8;5M")}, wantMux: "write:ls\n", wantRTOps: "scroll-up"},
		// A RELEASE ("m" terminator) is a complete event. Holding it back as an
		// unfinished press parks it — and everything typed after it — in `held`,
		// which reads as a dead keyboard and leaves the child app stuck in a
		// mouse drag (nvim: stuck in visual selection).
		{name: "mouse release passes to child", chunks: [][]byte{[]byte("\x1b[<0;8;2m")}, wantMux: "write:\x1b[<0;8;2m"},
		{name: "keystroke after mouse release is not swallowed", chunks: [][]byte{[]byte("\x1b[<0;8;2m"), []byte("a")}, wantMux: "write:\x1b[<0;8;2m,write:a"},
		{name: "drag then release then payload in one read", chunks: [][]byte{[]byte("\x1b[<0;8;2M\x1b[<0;9;2mls\n")}, wantMux: "write:\x1b[<0;8;2M,write:\x1b[<0;9;2m,write:ls\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &fakeRuntime{}
			mux := &fakeMux{appMouse: tt.appMouse}
			var stdout bytes.Buffer
			pumpStdinWithTimer(&splitReader{chunks: tt.chunks}, mux, rt, &stdout, beforeDeadline())
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
	if len(mux.reported) != 1 || !strings.Contains(mux.reported[0], "focus") {
		t.Fatalf("reported = %v, want focus error", mux.reported)
	}
}

func TestSplitTerminalDownIsNativeTiledSplit(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":4,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
	]`
	rt := &fakeRuntime{panesJSON: panes, currentPaneID: "4"}

	if err := splitTerminalDown(rt); err != nil {
		t.Fatal(err)
	}

	// One native op: no geometry math, no floating flags — zellij splits the
	// client-focused pane (the invoking terminal) downward in the tiled tree.
	want := []string{
		`quiet new-pane --direction down --name terminal -- sh -c zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`,
	}
	if strings.Join(rt.ops, ",") != strings.Join(want, ",") {
		t.Fatalf("ops = %v, want %v", rt.ops, want)
	}
}

func TestTerminalAltKRecordsLeavingSplitHalf(t *testing.T) {
	rt := &fakeRuntime{lastLeft: "1"}
	var stderr bytes.Buffer
	code := RunWithRuntime([]string{"--test-shortcut", "Alt+k"}, strings.NewReader(""), &bytes.Buffer{}, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if strings.Join(rt.recordedTerminal, ",") != "4" {
		t.Fatalf("recorded terminal = %v, want [4] (the focused terminal pane)", rt.recordedTerminal)
	}
	if strings.Join(rt.ops, ",") != "focus-pane-id 1" {
		t.Fatalf("ops = %v, want focus-pane-id 1", rt.ops)
	}
}

func TestSplitHalfChordsWorkViaRegistry(t *testing.T) {
	// The live shape of an Alt+Shift+d split half in zellij 0.44.3: the pane
	// report carries NO terminal_command (--direction-created) and the #118
	// tab-strip title ("[terminal 1]") defeats the title fallback. Only the
	// terminal-pane registry identifies it.
	panes := `[
		{"id":1,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":26,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
		{"id":4,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_y":26,"pane_columns":75,"pane_rows":25,"title":"[terminal 1]","terminal_command":null}
	]`
	rt := &fakeRuntime{panesJSON: panes, currentPaneID: "4", terminalPaneIDs: []string{"3", "4"}, lastLeft: "2"}
	var stderr bytes.Buffer
	code := RunWithRuntime([]string{"--test-shortcut", "Alt+k"}, strings.NewReader(""), &bytes.Buffer{}, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	if strings.Join(rt.recordedTerminal, ",") != "4" {
		t.Fatalf("recorded terminal = %v, want [4]", rt.recordedTerminal)
	}
	if strings.Join(rt.ops, ",") != "focus-pane-id 2" {
		t.Fatalf("ops = %v, want focus-pane-id 2", rt.ops)
	}
}

func TestChordRoleResolvesOwnPaneUnderAmbiguousFocus(t *testing.T) {
	// zellij can report several panes focused at once (per-client focus; seen
	// live in the tiled smoke: draft AND terminal both is_focused). Bytes on
	// pair term's stdin can only mean its OWN pane is the input target, so
	// role resolution must prefer ZELLIJ_PANE_ID over the is_focused scan —
	// otherwise the draft wins by list order and the chord silently passes.
	panes := `[
		{"id":1,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":4,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"},
		{"id":2,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"}
	]`
	rt := &fakeRuntime{panesJSON: panes, currentPaneID: "4"}
	var stderr bytes.Buffer
	code := RunWithRuntime([]string{"--test-shortcut", "Alt+Shift+d"}, strings.NewReader(""), &bytes.Buffer{}, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d stderr=%q", code, stderr.String())
	}
	want := `quiet new-pane --direction down --name terminal -- sh -c zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`
	if strings.Join(rt.ops, ",") != want {
		t.Fatalf("ops = %v, want the split (role must resolve to own terminal pane)", rt.ops)
	}
}

func TestSplitTerminalDownRefusesWithoutRightTerminal(t *testing.T) {
	rt := &fakeRuntime{panesJSON: `[
		{"id":1,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"}
	]`}

	if err := splitTerminalDown(rt); err == nil {
		t.Fatal("want error when no right terminal pane exists")
	}
	if len(rt.ops) != 0 {
		t.Fatalf("ops = %v, want none", rt.ops)
	}
}

func TestPumpStdinConsumesGlobalChordWhenDraftMissing(t *testing.T) {
	rt := &fakeRuntime{panesJSON: `[
		{"id":4,"is_focused":true,"is_fullscreen":false,"is_floating":true,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
	]`}
	mux := &fakeMux{}

	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[110;3u")}}, mux, rt, io.Discard)

	if len(mux.ops) != 0 {
		t.Fatalf("mux ops = %v, want recognized chord consumed", mux.ops)
	}
	if len(mux.reported) != 1 || !strings.Contains(mux.reported[0], "draft pane") {
		t.Fatalf("reported = %v, want missing draft pane error", mux.reported)
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

func TestPumpStdinRenameCmdDeleteDeletesToStart(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{activeName: "work"}

	pumpStdin(&splitReader{chunks: [][]byte{
		[]byte("\x1br\x1b[D"),
		[]byte("\x1b[127;9u\r"),
	}}, mux, rt, io.Discard)

	want := "rename-begin:work,rename-preview:work:3,rename-preview:k:0,rename-finish:1:k"
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
	if len(mux.reported) != 1 {
		t.Fatalf("reported = %v, want one rename error", mux.reported)
	}
}

// Only FINISH can fail now: a keystroke changes the model and repaints the row,
// and since #199 M3 nothing on that path talks to a subprocess -- the pane title
// is written once, on commit. The outcome must survive that one failure.
func TestPumpStdinRenameFinishFailurePreservesOutcome(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{
		activeName:      "work",
		finishRenameErr: exec.ErrNotFound,
	}

	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1brx\r")}}, mux, rt, io.Discard)

	if mux.activeName != "workx" {
		t.Fatalf("active name = %q, want committed workx", mux.activeName)
	}
	if len(mux.reported) != 1 {
		t.Fatalf("reported = %v, want the finish error only", mux.reported)
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

// ARCH-CONSTRAINTS enforced rather than asserted (BR-50). The plan declares
// "the degraded title keeps one spawn on tab switch only, NOT on every render",
// and until #199 M3 that was false: the rename field was packed into the pane
// TITLE, so opening a rename and typing into it forked `zellij action
// rename-pane` once PER KEYSTROKE -- on the interaction path ARCH-CONSTRAINTS
// names as the one that matters, and the cost this issue's Problem statement
// opens with. The strip carries the field now, so the whole rename costs one
// spawn, on commit. A budget with no test is a sentence in a plan.

func TestPumpStdinRenameBareEscapeCancelsOnTimer(t *testing.T) {
	rt := &fakeRuntime{}
	finished := make(chan RenameOutcome, 1)
	mux := &fakeMux{activeName: "work", renameFinished: finished}
	reader := &gatedEOFReader{data: []byte("\x1br\x1b"), release: make(chan struct{})}
	timer := newFiringEscapeTimer()
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

func TestPumpStdinRenameEscapeTimeoutThenNextReadForwards(t *testing.T) {
	rt := &fakeRuntime{}
	finished := make(chan RenameOutcome, 1)
	releaseNext := make(chan struct{})
	mux := &fakeMux{activeName: "work", renameFinished: finished}
	reader := &gatedChunksReader{
		chunks:  [][]byte{[]byte("\x1brx\x1b"), []byte("ls\n")},
		release: releaseNext,
	}
	timer := newFiringEscapeTimer()
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
	close(releaseNext)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stdin pump did not finish after second chunk")
	}
	want := "rename-begin:work,rename-preview:workx:5,rename-finish:2:work,write:ls\n"
	if got := strings.Join(mux.ops, ","); got != want {
		t.Fatalf("ops = %q, want %q", got, want)
	}
}

func TestPumpStdinRenameEscapeContinuationBeatsTimer(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{activeName: "work"}
	timer := newFiringEscapeTimer()
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

// The caret is composed ONCE, and this is where. It used to be placed
// independently by the strip and by the pane title; #199 M3 gave both
// RenameEditor.Field and then retired the title's copy of the field entirely,
// so this is the only composer left in the tree.
func TestTheRenameFieldPlacesTheCaretAtTheCursor(t *testing.T) {
	editor := NewRenameEditor("work")
	editor, _ = editor.Apply(RenameEvent{Kind: RenameMoveLeft})
	if got := editor.Field(); got != "wor│k" {
		t.Fatalf("rename field = %q, want the caret before the last rune", got)
	}
}

func TestTerminalMuxSetPaneTitleTargetsOwnPane(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &terminalMux{rt: rt, paneID: "7"}
	if err := mux.setPaneTitle("terminal work"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(rt.ops, ","); got != "rename-pane --pane-id 7 terminal work" {
		t.Fatalf("runtime ops = %q, want own-pane rename", got)
	}
}

func TestRightTerminalPaneShellMatchesLayout3(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "zellij", "layouts", "main-3.kdl"))
	if err != nil {
		t.Fatal(err)
	}
	want := `args "-c" "` + strings.ReplaceAll(rightTerminalPaneShell, `"`, `\"`) + `"`
	if !strings.Contains(string(data), want) {
		t.Fatalf("layout3 terminal shell drifted from Go split action\nwant KDL line containing: %s", want)
	}
}

func TestParseSGRMousePress(t *testing.T) {
	event, ok := mouseinput.Parse([]byte("\x1b[<64;12;1M"))
	if !ok || event.Button != 64 || event.X != 12 || event.Y != 1 {
		t.Fatalf("mouse = (%+v,%v), want ({button:64 x:12 y:1},true)", event, ok)
	}
	if event.Release {
		t.Fatal("M terminator is a press, not a release")
	}
	// 'm' is the RELEASE terminator — a complete event, not a partial press.
	release, ok := mouseinput.Parse([]byte("\x1b[<0;12;1m"))
	if !ok || !release.Release || release.Button != 0 || release.X != 12 || release.Y != 1 {
		t.Fatalf("release = (%+v,%v), want ({button:0 x:12 y:1 release:true},true)", release, ok)
	}
}

// TestUpdateMouseMode moved to ptychild's TestScreenMouseMode with every case
// intact, plus the split-read case the old chunk-at-a-time scanner could not
// pass. Mouse state is the child's now, read through tab.child.Mouse().

func TestTerminalMuxPaneTitleShowsTabs(t *testing.T) {
	mux := &terminalMux{
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1"},
			{id: 2, name: "work"},
			{id: 3, name: "terminal 3"},
		},
		active: 1,
	}
	// DEGRADED since #199 M3: the title is the active tab's name, not the whole
	// tab set packed into one rename argument. The strip carries tab state now;
	// the title is back to being a label for the two consumers that read it when
	// the pane is not focused (layoutflow.go:62, shortcut.go:189), both of which
	// also match on the pane's command.
	if got := mux.paneTitleLocked(); got != "terminal work" {
		t.Fatalf("pane title = %q, want the active tab's name behind the "+
			"classifier prefix", got)
	}
	// And it must not be the packed form any more -- the thing #199 replaced.
	if strings.Contains(mux.paneTitleLocked(), "[") {
		t.Fatalf("pane title still packs the tab set: %q", mux.paneTitleLocked())
	}
}

// Was TestTerminalMuxChildUsesFullPaneHeight, which asserted the pre-#199
// contract: with no strip, the child got the whole pane. M3 reserves the bottom
// row, so the child gets one less -- the off-by-one the whole reserved-row
// design IS. Renamed rather than edited in place, because the old name now
// describes the opposite of the intended behaviour.
func TestTerminalMuxChildStopsOneRowShortOfThePane(t *testing.T) {
	mux := &terminalMux{rows: 51, cols: 80}
	if got := mux.childSizeLocked(); got.Rows != 50 || got.Cols != 80 {
		t.Fatalf("child size = %+v, want 50x80 -- the pane minus the strip's row", got)
	}

	// A pane too short to reserve from gives the child everything and draws no
	// strip. A zero-row pty is not a thing, and a pane that short has no room
	// for chrome anyway.
	for _, rows := range []uint16{0, 1} {
		short := &terminalMux{rows: rows, cols: 80}
		if got := short.childSizeLocked(); got.Rows != rows {
			t.Fatalf("rows=%d: child got %d; a pane with no room to reserve keeps it all",
				rows, got.Rows)
		}
		if cells, err := short.chromeLocked(-1); err != nil || len(cells) != 0 {
			t.Fatalf("rows=%d chrome=%v err=%v", rows, cells, err)
		}
	}
}

func (f *fakeRuntime) LastTerminalPaneID() (string, error) {
	return f.lastTerminal, nil
}

func (f *fakeRuntime) TerminalPaneIDs() ([]string, error) {
	return f.terminalPaneIDs, nil
}

func (f *fakeRuntime) RegisterTerminalPane() error {
	f.registeredTerminalPane = true
	return nil
}

func (f *fakeRuntime) RecordLastTerminalPaneID(id string) error {
	f.recordedTerminal = append(f.recordedTerminal, id)
	f.lastTerminal = id
	return nil
}

func (f *fakeRuntime) CachedDraftPaneID() (string, bool) {
	return f.cachedDraft, f.cachedDraft != ""
}

func (f *fakeRuntime) CurrentPaneID() string {
	return f.currentPaneID
}

func (f *fakeRuntime) ListPanesJSON() ([]byte, error) {
	f.listCalls++
	if f.failList {
		return nil, errors.New("pane inventory must not run")
	}
	if f.panesJSON == "" {
		return []byte(`[
			{"id":1,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
			{"id":2,"is_focused":false,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
				{"id":4,"is_focused":true,"is_fullscreen":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
		]`), nil
	}
	return []byte(f.panesJSON), nil
}

func TestPumpStdinRoutesCachedGlobalWithoutPaneInventory(t *testing.T) {
	rt := &fakeRuntime{cachedDraft: "2", failList: true}
	mux := &fakeMux{}

	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[110;3u")}}, mux, rt, io.Discard)

	if rt.listCalls != 0 {
		t.Fatalf("list calls = %d, want 0 for global chord", rt.listCalls)
	}
	if len(mux.reported) != 0 {
		t.Fatalf("reported = %v, want successful cached route", mux.reported)
	}
	want := "focus-pane-id 2,write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairConfirmRestart(),write --pane-id 2 13"
	if got := strings.Join(rt.ops, ","); got != want {
		t.Fatalf("runtime ops = %q, want %q", got, want)
	}
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

func (f *fakeRuntime) RunZellijActionQuiet(args ...string) error {
	f.ops = append(f.ops, "quiet "+strings.Join(args, " "))
	return nil
}

func (f *fakeRuntime) reportedUnused(err error) {
	f.reported = append(f.reported, err.Error())
}

func (f *fakeRuntime) ShellCommand() (string, []string) {
	return "/bin/sh", []string{"-i"}
}

type fakeMux struct {
	reported []string
	ops      []string
	// wrote, when non-nil, receives every writeActive payload as it happens,
	// so a test can observe a deadline flush before releasing the next read.
	wrote           chan string
	appMouse        bool
	ownsScreen      bool
	activeName      string
	beginRenameErr  error
	finishRenameErr error
	renameFinished  chan RenameOutcome
}

func (f *fakeMux) writeEvents(events []terminal.InputEvent) {
	var data []byte
	for _, e := range events {
		data = append(data, e.Raw...)
	}
	f.writeActive(data)
}
func (f *fakeMux) writeActive(data []byte) {
	f.ops = append(f.ops, "write:"+string(data))
	if f.wrote != nil {
		f.wrote <- string(data)
	}
}

func (f *fakeMux) activeChildOwnsScreen() bool { return f.ownsScreen }

func (f *fakeMux) newTab() error {
	f.ops = append(f.ops, "new-tab")
	return nil
}

func (f *fakeMux) closeActive() {
	f.ops = append(f.ops, "close-tab")
}

func (f *fakeMux) beginRename() (int, RenameEditor, error) {
	f.ops = append(f.ops, "rename-begin:"+f.activeName)
	return 1, NewRenameEditor(f.activeName), f.beginRenameErr
}

func (f *fakeMux) refreshRename(_ int, editor RenameEditor) {
	f.ops = append(f.ops, fmt.Sprintf("rename-preview:%s:%d", editor.Text(), editor.Cursor()))
}

func (f *fakeMux) finishRename(_ int, outcome RenameOutcome) error {
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

// Recorded rather than printed: these used to reach os.Stderr -- the pane's own
// terminal -- from the input goroutine, outside the writer loop (#199 M2).
func (f *fakeMux) reportError(err error) {
	if err != nil {
		f.reported = append(f.reported, err.Error())
	}
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

type gatedChunksReader struct {
	chunks  [][]byte
	release <-chan struct{}
}

func (r *gatedChunksReader) Read(p []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	if len(r.chunks) == 1 {
		<-r.release
	}
	chunk := r.chunks[0]
	r.chunks = r.chunks[1:]
	return copy(p, chunk), nil
}

type firingEscapeTimer struct {
	ch       chan time.Time
	autoFire bool
	resets   int
	stops    int
}

func newFiringEscapeTimer() *firingEscapeTimer {
	return &firingEscapeTimer{ch: make(chan time.Time, 1), autoFire: true}
}

// beforeDeadline is the interleaving where every read lands before the
// escape-ambiguity deadline fires: the timer arms and stops but never ticks.
// Tests that split a chord across reads use it instead of the real timer so
// they cannot race a 35 ms wall clock (#234).
func beforeDeadline() *firingEscapeTimer {
	timer := newFiringEscapeTimer()
	timer.autoFire = false
	return timer
}

func (t *firingEscapeTimer) C() <-chan time.Time {
	return t.ch
}

func (t *firingEscapeTimer) Reset(time.Duration) {
	t.resets++
	if !t.autoFire {
		return
	}
	select {
	case t.ch <- time.Now():
	default:
	}
}

func (t *firingEscapeTimer) StopAndDrain() {
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

// The terminal chord surface is split across TWO seams: workbenchshortcut.Decide's
// terminal branch and handleTerminalChord here — and their sets differ (this one
// has AltLeft/AltRight; AltR is special-cased at run.go:408). #132's help was built
// from Decide alone, so Alt+←/Alt+→ shipped undocumented under a section titled
// "Terminal tabs".
//
// This is the mirror of TestRoleBindingsCoverTerminalSwitch: every chord THIS seam
// claims must also be described in RoleBindings, so neither seam can grow a chord
// the help does not know about.
func TestEveryHandledTerminalChordIsDocumented(t *testing.T) {
	documented := map[workbenchshortcut.Chord]bool{}
	for _, rb := range workbenchshortcut.RoleBindings() {
		documented[rb.Chord] = true
	}
	// Bound from the sentinel, not the last-named chord: the previous
	// `chord <= ChordAltShiftEnter` silently dropped every chord appended after
	// it (#216 PQ-3).
	for chord := workbenchshortcut.ChordUnknown + 1; chord < workbenchshortcut.ChordMax(); chord++ {
		rt := &fakeRuntime{}
		mux := &fakeMux{}
		if !handleTerminalChord(chord, mux, rt) {
			continue
		}
		// Same exemption TestRoleBindingsCoverTerminalSwitch already applies:
		// a global carries its Help on GlobalBinding, so requiring it in
		// RoleBindings too would duplicate the wording (ARCH-DRY).
		if _, isGlobal := workbenchshortcut.DecideGlobal(chord); isGlobal {
			continue
		}
		if !documented[chord] {
			t.Errorf("handleTerminalChord handles chord %v but workbenchshortcut.RoleBindings() does not describe it — `pair keys` would omit it", chord)
		}
	}
}

// The synthesised pane on #220's fast path must classify the way the real
// report would. If RoleForPane's predicate changes and this constant does not,
// every chord from the terminal routes as PaneRoleOther — silently.
func TestRightTerminalClassifierClassifiesAsARightTerminal(t *testing.T) {
	pane := zellijpane.Pane{ID: "4", TerminalCommand: rightTerminalClassifier}
	if got := workbenchshortcut.RoleForPaneWith(pane, nil); got != workbenchshortcut.PaneRoleRightTerminal {
		t.Fatalf("RoleForPaneWith(synthesised) = %v, want PaneRoleRightTerminal", got)
	}
}

// #220's fast paths avoid a 590ms `list-panes --json`. Both executed in ZERO
// tests when they were written (BR-3), so a coverage profile showed count 0 on
// every line of them. These assert the saving actually happens AND that the
// registry gate holds — the env var says which pane we are, not what kind
// (BR-5), so an unregistered pane must fall back rather than synthesise a
// right-terminal role.
// The rule (#220 BR-13): a fast path substituting for an existing slow path is
// not tested by asserting listCalls==0 and that it declines when gated — that
// pins the SAVING and the GATE while leaving the ANSWER unpinned. It is tested
// when its answer is differentially compared against the slow path on a SHARED
// fixture.
//
// The first version of this test also could not discriminate: it set
// cachedDraft "2" against a fixture whose draft was also id 2, so "reads the
// cache" and "agrees with the report" were the same observation. The draft here
// is id 7 — distinct from every other id in the fixture — so a fast path
// reading the wrong sidecar produces a different answer.
const workbenchFixturePanes = `[
	{"id":4,"is_focused":true,"is_fullscreen":false,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
	{"id":7,"is_focused":false,"is_fullscreen":false,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"}
]`

func TestFocusedWorkbenchPanesFastPathAnswersWhatTheSlowPathWould(t *testing.T) {
	fast := &fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}, cachedDraft: "7", panesJSON: workbenchFixturePanes}
	// Same world; the gate fails, so this one walks the pane report.
	slow := &fakeRuntime{currentPaneID: "4", cachedDraft: "7", panesJSON: workbenchFixturePanes}

	fastPanes, err := focusedWorkbenchPanes(fast)
	if err != nil {
		t.Fatal(err)
	}
	slowPanes, err := focusedWorkbenchPanes(slow)
	if err != nil {
		t.Fatal(err)
	}

	if fast.listCalls != 0 {
		t.Errorf("fast path called ListPanesJSON %d times, want 0 — the 590ms is the point", fast.listCalls)
	}
	if slow.listCalls != 1 {
		t.Fatalf("slow path called ListPanesJSON %d times, want 1 — the fixture is not exercising it", slow.listCalls)
	}
	if fastPanes.focused.ID != slowPanes.focused.ID || fastPanes.draft.ID != slowPanes.draft.ID {
		t.Fatalf("fast = focused %q draft %q; slow = focused %q draft %q — the two paths disagree",
			fastPanes.focused.ID, fastPanes.draft.ID, slowPanes.focused.ID, slowPanes.draft.ID)
	}
	if fastPanes.draft.ID != "7" {
		t.Errorf("draft = %q, want 7 — the id that only the cache and the report agree on", fastPanes.draft.ID)
	}
	if a, b := workbenchshortcut.RoleForPaneWith(fastPanes.focused, fast.terminalPaneIDs),
		workbenchshortcut.RoleForPaneWith(slowPanes.focused, slow.terminalPaneIDs); a != b {
		t.Errorf("role fast=%v slow=%v — the synthesised pane classifies differently", a, b)
	}
}

func TestFocusedWorkbenchPanesFallsBackWhenTheGateFails(t *testing.T) {
	for _, test := range []struct {
		name string
		rt   *fakeRuntime
	}{
		{"pane is not registered", &fakeRuntime{currentPaneID: "4", terminalPaneIDs: nil, cachedDraft: "7", panesJSON: workbenchFixturePanes}},
		{"no current pane id", &fakeRuntime{currentPaneID: "", terminalPaneIDs: []string{"4"}, cachedDraft: "7", panesJSON: workbenchFixturePanes}},
		{"no cached draft", &fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}, panesJSON: workbenchFixturePanes}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := focusedWorkbenchPanes(test.rt); err != nil {
				t.Fatal(err)
			}
			if test.rt.listCalls != 1 {
				t.Errorf("ListPanesJSON called %d times, want 1 — the fast path must decline here", test.rt.listCalls)
			}
		})
	}
}

func TestCurrentRightTerminalPaneFastPathAnswersWhatTheSlowPathWould(t *testing.T) {
	fast := &fakeRuntime{currentPaneID: "4", terminalPaneIDs: []string{"4"}, panesJSON: workbenchFixturePanes}
	slow := &fakeRuntime{currentPaneID: "4", panesJSON: workbenchFixturePanes}

	fastPane, fastOK, err := currentRightTerminalPane(fast)
	if err != nil {
		t.Fatal(err)
	}
	slowPane, slowOK, err := currentRightTerminalPane(slow)
	if err != nil {
		t.Fatal(err)
	}
	if fast.listCalls != 0 || slow.listCalls != 1 {
		t.Fatalf("list calls fast=%d slow=%d, want 0 and 1", fast.listCalls, slow.listCalls)
	}
	if fastOK != slowOK || fastPane.ID != slowPane.ID {
		t.Fatalf("fast = %q/%v, slow = %q/%v — the two paths disagree", fastPane.ID, fastOK, slowPane.ID, slowOK)
	}
	if fastPane.ID != "4" {
		t.Errorf("id = %q, want 4", fastPane.ID)
	}
}

func TestTerminalHelpAndChangelogRouteWithoutFocusChange(t *testing.T) {
	for _, tc := range []struct{ raw, fn string }{{"\x1b[104;3u", "PairOpenHelp"}, {"\x1b[108;3u", "PairOpenChangelog"}} {
		rt := &fakeRuntime{cachedDraft: "2", failList: true}
		mux := &fakeMux{}
		pumpStdin(&splitReader{chunks: [][]byte{[]byte(tc.raw)}}, mux, rt, io.Discard)
		want := "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua " + tc.fn + "(),write --pane-id 2 13"
		if got := strings.Join(rt.ops, ","); got != want {
			t.Fatalf("ops=%q want=%q", got, want)
		}
		if rt.listCalls != 0 || len(mux.reported) != 0 {
			t.Fatalf("unexpected query/error: %d %v", rt.listCalls, mux.reported)
		}
	}
}

type fakeRuntime struct {
	fullscreenRecord       string
	fullscreenErrors       []string
	panesJSON              string
	cachedDraft            string
	currentPaneID          string
	lastLeft               string
	lastTerminal           string
	recordedTerminal       []string
	terminalPaneIDs        []string
	registeredTerminalPane bool
	listCalls              int
	failList               bool
	ops                    []string
	reported               []string
	failFocus              bool
}

type stdoutWriter struct {
	*bytes.Buffer
}
