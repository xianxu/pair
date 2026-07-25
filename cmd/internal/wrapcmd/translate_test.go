package wrapcmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTranslateChunk(t *testing.T) {
	p := &proxy{sendKM: sendKeymap{
		plainCR: []byte{'\\', '\r'},
		altCR:   []byte{'\r'},
		altBS:   []byte{0x15},
	}}

	tests := []struct {
		name      string
		in        []byte
		startPase bool
		wantOut   []byte
		wantHold  []byte
		wantPaste bool
	}{
		{
			name:    "plain text passes through",
			in:      []byte("hello world"),
			wantOut: []byte("hello world"),
		},
		{
			name:    "Enter becomes backslash-Enter",
			in:      []byte("hi\r"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "Alt+Enter becomes plain Enter",
			in:      []byte("hi\x1b\r"),
			wantOut: []byte("hi\r"),
		},
		{
			name:    "mixed: Enter and Alt+Enter in same chunk",
			in:      []byte("a\rb\x1b\rc\r"),
			wantOut: []byte("a\\\rb\rc\\\r"),
		},
		{
			name:      "bracketed paste preserves embedded \\r",
			in:        []byte("\x1b[200~line1\rline2\r\x1b[201~"),
			wantOut:   []byte("\x1b[200~line1\rline2\r\x1b[201~"),
			wantPaste: false, // ends out of paste mode
		},
		{
			name:      "Enter after paste end gets rewritten",
			in:        []byte("\x1b[200~x\r\x1b[201~\r"),
			wantOut:   []byte("\x1b[200~x\r\x1b[201~\\\r"),
			wantPaste: false,
		},
		{
			name:      "paste start, mid-paste chunk",
			in:        []byte("\x1b[200~pasted text\r"),
			wantOut:   []byte("\x1b[200~pasted text\r"),
			wantPaste: true,
		},
		{
			name:      "paste continues into chunk, ends",
			startPase: true,
			in:        []byte("more\rstuff\x1b[201~Enter\r"),
			wantOut:   []byte("more\rstuff\x1b[201~Enter\\\r"),
			wantPaste: false,
		},
		{
			name:     "trailing ESC alone is held back",
			in:       []byte("hi\x1b"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b"),
		},
		{
			name:     "trailing partial bpStart held back",
			in:       []byte("hi\x1b[20"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[20"),
		},
		{
			name:      "trailing partial bpEnd inside paste held back",
			startPase: true,
			in:        []byte("data\x1b[20"),
			wantOut:   []byte("data"),
			wantHold:  []byte("\x1b[20"),
			wantPaste: true,
		},
		{
			name:    "ESC followed by non-CR non-[200 is passed through ESC",
			in:      []byte("hi\x1b[A"), // arrow up
			wantOut: []byte("hi\x1b[A"),
		},
		{
			name:    "KKP plain Enter becomes backslash-Enter",
			in:      []byte("hi\x1b[13u"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "KKP plain Enter (explicit no-modifier) becomes backslash-Enter",
			in:      []byte("hi\x1b[13;1u"),
			wantOut: []byte("hi\\\r"),
		},
		{
			name:    "KKP Alt+Enter becomes plain Enter",
			in:      []byte("hi\x1b[13;3u"),
			wantOut: []byte("hi\r"),
		},
		{
			name:    "mixed KKP and legacy in one chunk",
			in:      []byte("a\rb\x1b[13;3uc\x1b[13u"),
			wantOut: []byte("a\\\rb\rc\\\r"),
		},
		{
			name:    "KKP arrow key still passes through (\\x1b[A)",
			in:      []byte("a\x1b[Ab"),
			wantOut: []byte("a\x1b[Ab"),
		},
		{
			name:     "partial KKP Alt+Enter held back at chunk end",
			in:       []byte("hi\x1b[13;3"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[13;3"),
		},
		{
			name:     "partial KKP plain Enter held back at chunk end",
			in:       []byte("hi\x1b[13"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[13"),
		},
		{
			name:    "legacy Alt+Backspace becomes Ctrl+U",
			in:      []byte("hi\x1b\x7f"),
			wantOut: []byte("hi\x15"),
		},
		{
			name:    "KKP Alt+Backspace becomes Ctrl+U",
			in:      []byte("hi\x1b[127;3u"),
			wantOut: []byte("hi\x15"),
		},
		{
			name:    "plain Backspace (lone DEL) passes through",
			in:      []byte("hi\x7f"),
			wantOut: []byte("hi\x7f"),
		},
		{
			name:    "mixed: Alt+Backspace and Alt+Enter in one chunk",
			in:      []byte("a\x1b\x7fb\x1b\rc"),
			wantOut: []byte("a\x15b\rc"),
		},
		{
			name:     "partial KKP Alt+Backspace held back at chunk end",
			in:       []byte("hi\x1b[127;3"),
			wantOut:  []byte("hi"),
			wantHold: []byte("\x1b[127;3"),
		},
	}

	t.Run("codex keymap", func(t *testing.T) {
		px := &proxy{sendKM: sendKeymapByAgent["codex"]}
		cases := []struct{ in, want []byte }{
			{[]byte("hi\r"), []byte("hi\n")},         // Enter → newline
			{[]byte("hi\x1b\r"), []byte("hi\r")},     // legacy Alt+Enter → CR submit
			{[]byte("hi\x1b[13;3u"), []byte("hi\r")}, // KKP Alt+Enter → CR submit
			{[]byte("a\rb\x1b\r"), []byte("a\nb\r")},
			{[]byte("\x1b[200~text\rmore\x1b[201~"), []byte("\x1b[200~text\rmore\x1b[201~")}, // paste untouched
		}
		for _, c := range cases {
			got, _, _ := px.translateChunk(c.in, false)
			if !bytes.Equal(got, c.want) {
				t.Errorf("in=%q: got %q, want %q", c.in, got, c.want)
			}
		}
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOut, gotHold, gotPaste := p.translateChunk(tc.in, tc.startPase)
			if !bytes.Equal(gotOut, tc.wantOut) {
				t.Errorf("out: got %q, want %q", gotOut, tc.wantOut)
			}
			if !bytes.Equal(gotHold, tc.wantHold) {
				t.Errorf("hold: got %q, want %q", gotHold, tc.wantHold)
			}
			if gotPaste != tc.wantPaste {
				t.Errorf("paste: got %v, want %v", gotPaste, tc.wantPaste)
			}
		})
	}
}

func TestTranslateStdinHandlesWorkbenchShortcutWithoutReturnRemap(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		wantHandled string
		wantOut     string
	}{
		{name: "alt k", in: "\x1bkhello\r", wantHandled: "Alt+k", wantOut: "hello\r"},
		{name: "alt x", in: "\x1b[120;3u", wantHandled: "Alt+x"},
		{name: "agent alt shift enter passes through", in: "\x1b[13;4u", wantOut: "\x1b[13;4u"},
		{name: "payload before alt k", in: "hello\r\x1bk", wantHandled: "Alt+k", wantOut: "hello\r"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &proxy{}
			var handled []string
			p.workbenchShortcutHandler = func(chord string) bool {
				handled = append(handled, chord)
				return true
			}
			var out bytes.Buffer

			p.translateStdinFrom(strings.NewReader(tt.in), &out, time.Millisecond)

			if got := strings.Join(handled, ","); got != tt.wantHandled {
				t.Fatalf("handled = %q, want %q", got, tt.wantHandled)
			}
			if got := out.String(); got != tt.wantOut {
				t.Fatalf("out = %q, want %q", got, tt.wantOut)
			}
		})
	}
}

func TestTranslateStdinHandlesSplitWorkbenchShortcut(t *testing.T) {
	p := &proxy{}
	var handled []string
	p.workbenchShortcutHandler = func(chord string) bool {
		handled = append(handled, chord)
		return true
	}
	reader, writer := io.Pipe()
	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		p.translateStdinFrom(reader, &out, 50*time.Millisecond)
		close(done)
	}()

	_, _ = writer.Write([]byte("\x1b"))
	time.Sleep(5 * time.Millisecond)
	_, _ = writer.Write([]byte("j"))
	_ = writer.Close()
	<-done

	if got := strings.Join(handled, ","); got != "Alt+j" {
		t.Fatalf("handled = %q, want Alt+j", got)
	}
	if got := out.String(); got != "" {
		t.Fatalf("out = %q, want empty", got)
	}
}

type fakeDraftRouteRuntime struct {
	panes     []byte
	cached    string
	ops       []string
	failFocus bool
}

func (f *fakeDraftRouteRuntime) CachedDraftPaneID() (string, bool) {
	return f.cached, f.cached != ""
}

func (f *fakeDraftRouteRuntime) ListPanesJSON() ([]byte, error) {
	return f.panes, nil
}

func (f *fakeDraftRouteRuntime) RunZellijAction(args ...string) error {
	f.ops = append(f.ops, strings.Join(args, " "))
	if f.failFocus && len(args) > 0 && args[0] == "focus-pane-id" {
		return errors.New("focus failed")
	}
	return nil
}

func TestTranslateStdinRoutesGlobalHotkeysThroughRuntime(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		lua   string
		focus bool
	}{
		{name: "alt d", in: "\x1b[100;3u", lua: "PairConfirmDetach", focus: true},
		{name: "alt x", in: "\x1b[120;3u", lua: "PairConfirmQuit", focus: true},
		{name: "alt n", in: "\x1b[110;3u", lua: "PairConfirmRestart", focus: true},
		{name: "ctrl alt n", in: "\x1b[110;7u", lua: "PairConfirmRestart", focus: true},
		{name: "shift alt n", in: "\x1b[78;4u", lua: "PairConfirmAgentRestart", focus: true},
		{name: "alt up", in: "\x1b[1;3A", lua: "PairLayoutBigger"},
		{name: "alt down", in: "\x1b[1;3B", lua: "PairLayoutSmaller"},
		{name: "alt c", in: "\x1b[99;3u", lua: "PairReviewToggle"},
	}
	panes := []byte(`[
		{"id":1,"is_focused":true,"is_plugin":false,"terminal_command":"pair wrap codex"},
		{"id":2,"is_focused":false,"is_plugin":false,"terminal_command":"nvim -u /pair/nvim/init.lua /data/draft.md"}
	]`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := &fakeDraftRouteRuntime{panes: panes}
			var reported []string
			p := &proxy{
				draftRouteRuntime: rt,
				shortcutErrorReporter: func(err error) {
					reported = append(reported, err.Error())
				},
			}
			var out bytes.Buffer

			p.translateStdinFrom(strings.NewReader(tt.in), &out, time.Millisecond)

			if out.Len() != 0 {
				t.Fatalf("child bytes = %q, want none", out.String())
			}
			want := "write --pane-id 2 28,write --pane-id 2 14,write-chars --pane-id 2 :lua " +
				tt.lua + "(),write --pane-id 2 13"
			if tt.focus {
				want = "focus-pane-id 2," + want
			}
			if got := strings.Join(rt.ops, ","); got != want {
				t.Fatalf("ops = %q, want %q", got, want)
			}
			if len(reported) != 0 {
				t.Fatalf("reported = %v, want none", reported)
			}
		})
	}
}

func TestTranslateStdinReportsFocusFailureWithoutWriting(t *testing.T) {
	rt := &fakeDraftRouteRuntime{cached: "2", failFocus: true}
	var reported []string
	p := &proxy{
		draftRouteRuntime: rt,
		shortcutErrorReporter: func(err error) {
			reported = append(reported, err.Error())
		},
	}
	var out bytes.Buffer

	p.translateStdinFrom(strings.NewReader("\x1b[110;3u"), &out, time.Millisecond)

	if out.Len() != 0 {
		t.Fatalf("child bytes = %q, want none", out.String())
	}
	if got := strings.Join(rt.ops, ","); got != "focus-pane-id 2" {
		t.Fatalf("ops = %q, want focus only", got)
	}
	if len(reported) != 1 || !strings.Contains(reported[0], "focus") {
		t.Fatalf("reported = %v, want focus failure", reported)
	}
}

func TestTranslateStdinConsumesGlobalHotkeyWhenDraftMissing(t *testing.T) {
	rt := &fakeDraftRouteRuntime{panes: []byte(`[{"id":1,"is_focused":true,"terminal_command":"pair wrap codex"}]`)}
	var reported []string
	p := &proxy{
		draftRouteRuntime: rt,
		shortcutErrorReporter: func(err error) {
			reported = append(reported, err.Error())
		},
	}
	var out bytes.Buffer

	p.translateStdinFrom(strings.NewReader("\x1b[110;3u"), &out, time.Millisecond)

	if out.Len() != 0 {
		t.Fatalf("child bytes = %q, want none", out.String())
	}
	if len(reported) != 1 || !strings.Contains(reported[0], "draft pane") {
		t.Fatalf("reported = %v, want missing draft pane", reported)
	}
}

func TestHandleWorkbenchShortcutRunsAgentProductionPath(t *testing.T) {
	dir := t.TempDir()
	fakebin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(fakebin, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "zellij.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + logPath + "\n"
	if err := os.WriteFile(filepath.Join(fakebin, "zellij"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakebin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_TAG", "work")
	t.Setenv("ZELLIJ_PANE_ID", "17")

	p := &proxy{}
	if !p.handleWorkbenchShortcut("Alt+k") {
		t.Fatal("Alt+k was not handled")
	}
	if !p.handleWorkbenchShortcut("Alt+j") {
		t.Fatal("Alt+j was not handled")
	}

	sidecar, err := os.ReadFile(filepath.Join(dir, "last-left-pane-work"))
	if err != nil {
		t.Fatal(err)
	}
	if string(sidecar) != "17\n" {
		t.Fatalf("last-left pane = %q, want 17", sidecar)
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(logged) != "action move-focus right\naction move-focus down\n" {
		t.Fatalf("zellij actions = %q", logged)
	}
}
