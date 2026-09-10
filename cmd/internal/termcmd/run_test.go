package termcmd

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestRunTestShortcutRightTerminalActions(t *testing.T) {
	panes := `[
		{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":4,"is_focused":true,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
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
		{name: "alt shift enter fires the three-step expand burst", chord: "Alt+Shift+Enter", wantOps: []string{
			"resize increase left",
			"resize increase left",
			"resize increase left",
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
		{name: "alt up routes grow to draft", chunks: [][]byte{[]byte("\x1b[1;3A")}, wantRTOps: "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairLayoutBigger(),write --pane-id 2 13"},
		{name: "alt down routes shrink to draft", chunks: [][]byte{[]byte("\x1b[1;3B")}, wantRTOps: "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairLayoutSmaller(),write --pane-id 2 13"},
		{name: "alt c routes review toggle to draft", chunks: [][]byte{[]byte("\x1b[99;3u")}, wantRTOps: "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua PairReviewToggle(),write --pane-id 2 13"},
		{name: "layout toggle", chunks: [][]byte{[]byte("\x1b[13;4u")}, wantRTOps: "resize increase left,resize increase left,resize increase left"},
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

func TestPumpStdinTerminalShortcutsDoNotLeakWhenSplit(t *testing.T) {
	for _, seq := range []string{"\x1bt", "\x1b[116;3u"} {
		t.Run(fmt.Sprintf("%q", seq), func(t *testing.T) {
			for split := 1; split < len(seq); split++ {
				rt := &fakeRuntime{}
				mux := &fakeMux{}
				pumpStdin(&splitReader{chunks: [][]byte{
					[]byte(seq[:split]),
					[]byte(seq[split:]),
				}}, mux, rt, io.Discard)

				if got := strings.Join(mux.ops, ","); got != "new-tab" {
					t.Fatalf("split %d ops = %q, want new-tab without residue", split, got)
				}
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
		{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":4,"is_focused":true,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
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
		{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
		{"id":3,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":26,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
		{"id":4,"is_focused":true,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_y":26,"pane_columns":75,"pane_rows":25,"title":"[terminal 1]","terminal_command":null}
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
		{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":4,"is_focused":true,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"},
		{"id":2,"is_focused":true,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"}
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
		{"id":1,"is_focused":true,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"}
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
		{"id":4,"is_focused":true,"is_floating":true,"is_plugin":false,"title":"terminal","terminal_command":"pair term"}
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
func TestARenameCostsExactlyOneZellijSubprocess(t *testing.T) {
	// lockedWriter, not a bare bytes.Buffer: copyActiveOutput writes from its own
	// goroutine while this test polls stdout, which -race (correctly) flags on the
	// double. m.stdout is an *os.File in production, so this is a test-harness
	// fix — do NOT add stdout locking to the mux to silence it.
	stdout := &lockedWriter{}
	rt := &fakeRuntime{}
	mux := &terminalMux{
		pane:   paneWriter{w: stdout},
		rt:     rt,
		output: make(chan ptyChunk, 1),
		done:   make(chan struct{}),
		tabs: []*terminalTab{
			{id: 1, name: "work"},
		},
		active: 0,
	}
	copied := make(chan struct{})
	go func() {
		mux.copyActiveOutput()
		close(copied)
	}()

	tabID, editor, err := mux.beginRename()
	if err != nil {
		t.Fatal(err)
	}
	mux.output <- ptyChunk{id: 1, data: []byte("child redraw\n")}

	deadline := time.After(time.Second)
	for stdout.String() != "child redraw\n" {
		select {
		case <-deadline:
			t.Fatalf("stdout = %q, want child output copied", stdout.String())
		default:
			time.Sleep(time.Millisecond)
		}
	}
	// Three keystrokes into the field, then the child writes again. None of it
	// may reach a subprocess.
	for _, r := range "abc" {
		editor, _ = editor.Apply(RenameEvent{Kind: RenameInsert, Rune: r})
		mux.refreshRename(tabID, editor)
	}
	if got := strings.Join(rt.ops, ","); got != "" {
		t.Fatalf("runtime ops during the rename = %q, want none", got)
	}
	if err := mux.finishRename(tabID, RenameOutcome{Kind: RenameOutcomeCancel, Name: editor.Original()}); err != nil {
		t.Fatal(err)
	}
	close(mux.done)
	<-copied
	// EXACTLY ONE, and it is the degraded title: the active tab's name with the
	// classifier prefix, written once when the rename ends.
	if got := strings.Join(rt.ops, ","); got != "rename-pane terminal work" {
		t.Fatalf("runtime ops for the whole rename = %q, want exactly one on finish", got)
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

func TestPumpStdinRenameEscapeTimeoutThenNextReadForwards(t *testing.T) {
	rt := &fakeRuntime{}
	finished := make(chan RenameOutcome, 1)
	releaseNext := make(chan struct{})
	mux := &fakeMux{activeName: "work", renameFinished: finished}
	reader := &gatedChunksReader{
		chunks:  [][]byte{[]byte("\x1brx\x1b"), []byte("ls\n")},
		release: releaseNext,
	}
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
		if res := short.reservationLocked(); res.ReserveAndPaint("x") != "" {
			t.Fatalf("rows=%d: reserved a row on a pane with no room", rows)
		}
	}
}

func TestTerminalMuxSwitchTabAtColumn(t *testing.T) {
	var stdout bytes.Buffer
	rt := &fakeRuntime{}
	mux := &terminalMux{
		pane: paneWriter{w: stdoutWriter{&stdout}},
		rt:   rt,
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1", child: ptychild.NewFakeChild([]byte("one"))},
			{id: 2, name: "work", child: ptychild.NewFakeChild([]byte("two"))},
		},
		active: 0,
		cols:   40,
	}
	mux.nextTab()
	if mux.active != 1 {
		t.Fatalf("active = %d, want 1", mux.active)
	}
	// A rename still reaches the runtime on every tab switch -- the consumers
	// need a current label -- but it is now the active tab's name alone.
	if !strings.Contains(strings.Join(rt.ops, ","), "rename-pane terminal work") {
		t.Fatalf("ops = %v, want a rename to the active tab's name", rt.ops)
	}
	if !strings.Contains(stdout.String(), "two") {
		t.Fatalf("stdout = %q, want redraw of second tab", stdout.String())
	}
	if strings.Contains(stdout.String(), "\x1b[7m") {
		t.Fatalf("stdout contains obsolete inverse-video tab strip: %q", stdout.String())
	}
}

func TestTerminalMuxNewTabClearsPreviousTabViewport(t *testing.T) {
	var stdout bytes.Buffer
	mux := newTerminalMux("/bin/sh", []string{"-c", "sleep 1"}, &stdout, io.Discard, &fakeRuntime{})
	// The loop is the only writer since #199 M2, so a mux without one writes
	// nothing -- the assertion below is about what reaches the pane.
	go mux.copyActiveOutput()
	if err := mux.newTab(); err != nil {
		t.Fatal(err)
	}
	mux.drainForTest()
	mux.closeAll()

	// The reset is part of the erase since 2026-09-08: \x1b[J paints with the
	// CURRENT background, so clearing while a child's colour is active tints the
	// new tab's screen -- measured with nvim's lualine blue.
	if got := stdout.String(); !strings.HasPrefix(got, hostty.HomeAndClear) {
		t.Fatalf("stdout = %q, want new active tab to clear stale viewport", got)
	}
}

func TestTerminalMuxNewTabPrintsStartupOutputOnce(t *testing.T) {
	var stdout bytes.Buffer
	mux := newTerminalMux("/bin/sh", []string{"-c", "printf unique-startup-marker"}, &stdout, io.Discard, &fakeRuntime{})
	go mux.copyActiveOutput()
	if err := mux.newTab(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mux.mu.Lock()
		done := len(mux.tabs) == 0
		mux.mu.Unlock()
		if done {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	mux.mu.Lock()
	remaining := len(mux.tabs)
	mux.mu.Unlock()
	if remaining != 0 {
		t.Fatal("startup command did not exit")
	}
	if got := strings.Count(stdout.String(), "unique-startup-marker"); got != 1 {
		t.Fatalf("startup marker rendered %d times, want once: %q", got, stdout.String())
	}
}

func TestTerminalMuxBackgroundExitPreservesActiveTab(t *testing.T) {

	mux := &terminalMux{
		pane: paneWriter{w: io.Discard},
		rt:   &fakeRuntime{},
		done: make(chan struct{}),
		tabs: []*terminalTab{
			{id: 1, name: "one"},
			{id: 2, name: "two"},
			{id: 3, name: "three"},
		},
		active: 1,
	}

	mux.removeTab(1)

	if got := mux.activeTabLocked(); got == nil || got.id != 2 {
		t.Fatalf("active tab after background exit = %+v, want id 2", got)
	}
}

func TestTerminalMuxRenameCommitDoesNotRenameReplacementActiveTab(t *testing.T) {

	rt := &fakeRuntime{}
	mux := &terminalMux{
		pane: paneWriter{w: io.Discard},
		rt:   rt,
		done: make(chan struct{}),
		tabs: []*terminalTab{
			{id: 1, name: "one"},
			{id: 2, name: "two"},
		},
		active: 0,
	}
	tabID, editor, err := mux.beginRename()
	if err != nil {
		t.Fatal(err)
	}
	editor, outcome := editor.Apply(RenameEvent{Kind: RenameInsert, Rune: 'x'})
	if outcome.Kind != RenameOutcomeNone {
		t.Fatalf("insert outcome = %#v, want none", outcome)
	}
	mux.refreshRename(tabID, editor)
	_, outcome = editor.Apply(RenameEvent{Kind: RenameCommit})
	rt.ops = nil

	mux.removeTab(1)
	// The title is the DEGRADED one even mid-rename: since #199 M3 the rename
	// field lives on the strip, and the pane title is only ever the active tab's
	// name with the classifier prefix RoleForPane reads.
	if got := strings.Join(rt.ops, ","); got != "rename-pane terminal two" {
		t.Fatalf("runtime ops after target removal = %q, want the degraded title", got)
	}
	if err := mux.finishRename(tabID, outcome); err != nil {
		t.Fatal(err)
	}

	if got := mux.tabs[0].name; got != "two" {
		t.Fatalf("remaining tab name = %q, want original two", got)
	}
}

// A background tab exiting mid-rename REPAINTS THE STRIP but does not take over
// the screen.
//
// The distinction is the whole test. A takeover would clear and replay, throwing
// away the viewport the operator is editing over -- correct to skip. But the tab
// SET just changed, and the takeover used to be the only thing repainting the
// row on this path, so skipping it left the strip listing a tab that no longer
// exists (BR-45, found by the M3 boundary review; reproduced as ZERO bytes
// written after the exit).
//
// rows/cols are set DELIBERATELY: with them at zero stripBytes returns nil and
// the assertion below passes without a strip existing at all -- which is how
// this test would have kept passing through the defect it now pins.
func TestTerminalMuxBackgroundExitDuringRenameRepaintsStripWithoutTakeover(t *testing.T) {

	var stdout bytes.Buffer
	rt := &fakeRuntime{}
	mux := &terminalMux{
		pane: paneWriter{w: stdoutWriter{&stdout}},
		rt:   rt,
		done: make(chan struct{}),
		tabs: []*terminalTab{
			{id: 1, name: "one"},
			{id: 2, name: "two", child: ptychild.NewFakeChild([]byte("active output"))},
		},
		active: 1,
		rows:   24,
		cols:   80,
	}
	tabID, editor, err := mux.beginRename()
	if err != nil {
		t.Fatal(err)
	}
	editor, outcome := editor.Apply(RenameEvent{Kind: RenameInsert, Rune: 'x'})
	if outcome.Kind != RenameOutcomeNone {
		t.Fatalf("insert outcome = %#v, want none", outcome)
	}
	mux.refreshRename(tabID, editor)
	stdout.Reset()
	rt.ops = nil

	mux.removeTab(1)

	if got := strings.Join(rt.ops, ","); got != "rename-pane terminal two" {
		t.Fatalf("runtime ops = %q, want the degraded title for the surviving tab", got)
	}
	got := stdout.String()
	if !strings.Contains(got, "[rename: twox│]") {
		t.Fatalf("stdout = %q, want the strip repainted with the live rename field", got)
	}
	if strings.Contains(got, "one") {
		t.Fatalf("stdout = %q, still lists the tab that exited", got)
	}
	if strings.Contains(got, hostty.HomeAndClear) || strings.Contains(got, "active output") {
		t.Fatalf("stdout = %q, want no wholesale takeover during a rename", got)
	}
}

type stdoutWriter struct {
	*bytes.Buffer
}

type fakeRuntime struct {
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
			{"id":1,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":39,"title":"codex","terminal_command":"pair wrap codex"},
			{"id":2,"is_focused":false,"is_floating":false,"is_plugin":false,"pane_x":0,"pane_columns":75,"pane_rows":12,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua /data/draft-t.md"},
				{"id":4,"is_focused":true,"is_floating":false,"is_plugin":false,"pane_x":75,"pane_columns":75,"pane_rows":51,"title":"terminal","terminal_command":"pair term"}
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
	reported        []string
	ops             []string
	appMouse        bool
	activeName      string
	beginRenameErr  error
	finishRenameErr error
	renameFinished  chan RenameOutcome
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
		mux := &terminalMux{}
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
	{"id":4,"is_focused":true,"is_floating":false,"pane_x":75,"title":"[terminal 1]","terminal_command":"sh -c exec pair term"},
	{"id":7,"is_focused":false,"is_floating":false,"title":"draft","terminal_command":"nvim -u /pair/nvim/init.lua d.md"}
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

// #209's counted invariant, and #204's: a switch issues a REPAINT REQUEST, not
// only a replay write. The replay is the immediate paint; the nudge is what
// makes the result correct rather than probable when the last full frame has
// aged out of the 128 KiB ring.
//
// The fake records resizes, which is the in-process half of the ARCH-MOCK pair;
// the other half is cmd/probes/zellijrepaint, which drives a real zellij and
// confirms it actually repaints from its own buffer on SIGWINCH.
//
// The child is sized first, because that is what production does (children are
// spawned at childSizeLocked) and because the nudge restores the size the CHILD
// remembers rather than one a caller hands it — a child with no geometry has
// nothing to restore and correctly declines (#209 C2).
func TestTabSwitchIssuesARepaintRequestAndRestoresTheSize(t *testing.T) {
	var stdout bytes.Buffer
	incoming := ptychild.NewFakeChild([]byte("two"))
	mux := &terminalMux{
		pane: paneWriter{w: stdoutWriter{&stdout}},
		rt:   &fakeRuntime{},
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1", child: ptychild.NewFakeChild([]byte("one"))},
			{id: 2, name: "work", child: incoming},
		},
		active: 0,
		cols:   40,
		rows:   24,
	}
	mux.mu.Lock()
	want := mux.childSizeLocked()
	mux.mu.Unlock()
	if err := incoming.Resize(want); err != nil {
		t.Fatal(err)
	}

	mux.nextTab()

	resizes := waitForChildResizes(t, incoming, 3)[1:]
	if len(resizes) != 2 {
		t.Fatalf("resizes = %v, want exactly 2 — a nudge is a change AND a restore", resizes)
	}
	if resizes[0].Rows >= resizes[1].Rows {
		t.Errorf("resizes = %v, want the first to shrink rows and the second to restore", resizes)
	}
	if resizes[0].Cols != resizes[1].Cols {
		t.Errorf("resizes = %v, want columns untouched — a column change reflows wrapped lines", resizes)
	}
	if resizes[1] != want {
		t.Errorf("restored to %v, want the child's own size %v", resizes[1], want)
	}
}

// waitForChildResizes polls until the child has recorded n resizes. The nudge
// is asynchronous — it holds the child's geometry lock across a 20 ms settle
// rather than blocking the console's event loop for it (#209 C2).
func waitForChildResizes(t *testing.T, child *ptychild.Child, n int) []ptychild.Size {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if got := child.Resizes(); len(got) >= n {
			return got
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d resizes; got %v", n, child.Resizes())
	return nil
}

// termcmd's leg of the differential (#209 BR-9): what the console WRITES on a
// takeover is exactly what hostty.RepaintFor composes, with no prefix of its
// own and no byte dropped. couchtty's TestSwitchWritesExactlyTheComposedRepaint
// asserts the same thing about the same function, which is what makes the two
// consoles byte-identical for the same child state without a test that can
// drive both; hostty's golden fixes what that function emits.
func TestTakeoverWritesExactlyTheComposedRepaint(t *testing.T) {
	var stdout bytes.Buffer
	incoming := ptychild.NewFakeChild([]byte("\x1b[?1049hretained frame"))
	mux := &terminalMux{
		pane: paneWriter{w: stdoutWriter{&stdout}},
		rt:   &fakeRuntime{},
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1", child: ptychild.NewFakeChild([]byte("one"))},
			{id: 2, name: "work", child: incoming},
		},
		active: 0,
		cols:   40,
		rows:   24,
	}

	mux.mu.Lock()
	replay := replaySnapshotLocked(mux.tabs[1])
	mux.mu.Unlock()
	want := hostty.RepaintFor(incoming, replay)
	if len(want) == 0 {
		t.Fatal("fixture produced nothing to compose; the assertion below would be vacuous")
	}

	mux.nextTab()

	if !strings.Contains(stdout.String(), string(want)) {
		t.Fatalf("takeover wrote %q, want it to contain hostty.RepaintFor's exact composition %q",
			stdout.String(), want)
	}
}

// BR-4's fix reverted SILENTLY — the close review measured it — and a
// disposition of "addressed" that no test defends is a claim, so this is the
// test that should have shipped with it (#209 I1).
//
// The scenario is the operator's: close a tab while the SURVIVING tab's ring
// still holds nothing. The frame standing on the pane belongs to a tab that no
// longer exists, so leaving it there is showing the operator a window into a
// closed thing. C-1 later established that this is true of EVERY takeover, not
// just this one, and made blanking unconditional — so the mutation that reds
// this test is now "make hostty's composition skip HomeAndClear when the replay
// is empty", not a flag at this call site.
func TestClosingATabBlanksTheDeadTabsScreenEvenWithNothingToDraw(t *testing.T) {
	var stdout bytes.Buffer
	survivor := ptychild.NewFakeChild(nil) // ring empty: nothing retained to draw
	mux := &terminalMux{
		pane: paneWriter{w: stdoutWriter{&stdout}},
		rt:   &fakeRuntime{},
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1", child: ptychild.NewFakeChild([]byte("doomed tab content"))},
			{id: 2, name: "terminal 2", child: survivor},
		},
		active: 0,
		cols:   40,
		rows:   24,
	}

	if snap := replaySnapshotLocked(mux.tabs[1]); len(snap) != 0 {
		t.Fatalf("fixture retained %q for the survivor; this test needs an empty ring", snap)
	}
	mux.removeTab(1)

	if !strings.Contains(stdout.String(), hostty.HomeAndClear) {
		t.Fatalf("closing a tab wrote %q, want the screen blanked — the closed tab's "+
			"content must not survive it, and there is nothing retained to overwrite it with",
			stdout.String())
	}
}

// The same enumeration swept for the REPAINT REQUEST rather than the intent
// (#209 I2). removeTab hands the screen to a different child, which is exactly
// the condition the nudge exists for: the survivor's last full frame may have
// aged out of the ring, and only the child still holds it. Binding the request
// to the takeover is what makes this true at every site rather than at the one
// site somebody remembered.
func TestClosingATabAsksTheSurvivingChildToRepaint(t *testing.T) {
	var stdout bytes.Buffer
	survivor := ptychild.NewFakeChild(nil)
	if err := survivor.Resize(ptychild.Size{Rows: 23, Cols: 40}); err != nil {
		t.Fatal(err)
	}
	mux := &terminalMux{
		pane: paneWriter{w: stdoutWriter{&stdout}},
		rt:   &fakeRuntime{},
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1", child: ptychild.NewFakeChild([]byte("doomed"))},
			{id: 2, name: "terminal 2", child: survivor},
		},
		active: 0,
		cols:   40,
		rows:   24,
	}

	before := len(survivor.Resizes())
	mux.removeTab(1)
	got := waitForChildResizes(t, survivor, before+2)
	if got[before].Rows != 22 || got[before+1].Rows != 23 {
		t.Fatalf("resizes = %v, want a shrink-and-restore pair around 23 rows", got[before:])
	}
}
