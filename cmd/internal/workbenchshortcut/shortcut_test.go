package workbenchshortcut

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestPaneRole(t *testing.T) {
	tests := []struct {
		name string
		pane zellijpane.Pane
		want PaneRole
	}{
		{
			name: "pair wrap agent",
			pane: zellijpane.Pane{ID: "1", IsFocused: true,
				TerminalCommand: "sh -c exec pair wrap --scrollback-log /tmp/raw codex --no-alt-screen"},
			want: PaneRoleLeftAgent,
		},
		{
			name: "draft nvim init",
			pane: zellijpane.Pane{ID: "2", IsFocused: true,
				TerminalCommand: `nvim -u "/pair/nvim/init.lua" "/data/draft-pair.md"`},
			want: PaneRoleLeftDraft,
		},
		{
			name: "right terminal command",
			pane: zellijpane.Pane{ID: "3", IsFocused: true,
				TerminalCommand: "pair term"},
			want: PaneRoleRightTerminal,
		},
		{
			name: "floating right terminal command",
			pane: zellijpane.Pane{ID: "4", IsFocused: true, IsFloating: true,
				TerminalCommand: "pair term"},
			want: PaneRoleRightTerminal,
		},
		{
			name: "right terminal title fallback",
			pane: zellijpane.Pane{ID: "3", IsFocused: true, Title: "terminal"},
			want: PaneRoleRightTerminal,
		},
		{
			name: "right terminal tab strip title fallback",
			pane: zellijpane.Pane{ID: "3", IsFocused: true, Title: "terminal 1 [work] terminal 3"},
			want: PaneRoleRightTerminal,
		},
		{
			name: "floating review is other",
			pane: zellijpane.Pane{ID: "4", IsFocused: true, IsFloating: true,
				TerminalCommand: "nvim -u /pair/nvim/review.lua /tmp/review.md"},
			want: PaneRoleOther,
		},
		{
			name: "plugin is other",
			pane: zellijpane.Pane{ID: "5", IsFocused: true, IsPlugin: true, Title: "status-bar"},
			want: PaneRoleOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoleForPane(tt.pane); got != tt.want {
				t.Fatalf("RoleForPane() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShortcutDecision(t *testing.T) {
	tests := []struct {
		name     string
		role     PaneRole
		chord    Chord
		focused  string
		lastLeft string
		draft    string
		want     ShortcutDecision
	}{
		{
			name:  "right terminal new tab",
			role:  PaneRoleRightTerminal,
			chord: ChordAltT,
			want:  ShortcutDecision{Disposition: DispositionHandle, Action: ActionNewTab},
		},
		{
			name:  "right terminal close tab",
			role:  PaneRoleRightTerminal,
			chord: ChordAltW,
			want:  ShortcutDecision{Disposition: DispositionHandle, Action: ActionCloseTab},
		},
		{
			name:  "right terminal rename tab",
			role:  PaneRoleRightTerminal,
			chord: ChordAltR,
			want:  ShortcutDecision{Disposition: DispositionHandle, Action: ActionRenameTab},
		},
		{
			name:  "right terminal split down",
			role:  PaneRoleRightTerminal,
			chord: ChordAltShiftD,
			want:  ShortcutDecision{Disposition: DispositionHandle, Action: ActionSplitTerminalDown},
		},
		{
			name:  "right terminal alt x handles quit outside shell",
			role:  PaneRoleRightTerminal,
			chord: ChordAltX,
			want: ShortcutDecision{
				Disposition:      DispositionHandle,
				Action:           ActionConfirmQuit,
				DraftLuaFunction: "PairConfirmQuit",
				FocusDraft:       true,
			},
		},
		{
			name:  "right terminal alt j is no-op",
			role:  PaneRoleRightTerminal,
			chord: ChordAltJ,
			want:  ShortcutDecision{Disposition: DispositionSwallow},
		},
		{
			name:  "right terminal alt shift enter toggles focused layout",
			role:  PaneRoleRightTerminal,
			chord: ChordAltShiftEnter,
			want:  ShortcutDecision{Disposition: DispositionHandle, Action: ActionToggleFocusedLayout},
		},
		{
			name:     "right terminal alt k returns to last left pane and records the leaving half",
			role:     PaneRoleRightTerminal,
			chord:    ChordAltK,
			lastLeft: "1",
			draft:    "2",
			focused:  "4",
			want: ShortcutDecision{
				Disposition:              DispositionHandle,
				Action:                   ActionFocusPane,
				TargetPaneID:             "1",
				RecordLastTerminalPaneID: "4",
			},
		},
		{
			name:    "right terminal alt k falls back to draft",
			role:    PaneRoleRightTerminal,
			chord:   ChordAltK,
			draft:   "2",
			focused: "4",
			want: ShortcutDecision{
				Disposition:              DispositionHandle,
				Action:                   ActionFocusPane,
				TargetPaneID:             "2",
				RecordLastTerminalPaneID: "4",
			},
		},
		{
			name:    "left agent alt k records focused pane then focuses terminal",
			role:    PaneRoleLeftAgent,
			chord:   ChordAltK,
			focused: "1",
			want: ShortcutDecision{
				Disposition:          DispositionHandle,
				Action:               ActionFocusRightTerminal,
				RecordLastLeftPaneID: "1",
			},
		},
		{
			name:  "left draft alt j focuses agent",
			role:  PaneRoleLeftDraft,
			chord: ChordAltJ,
			want:  ShortcutDecision{Disposition: DispositionHandle, Action: ActionFocusLeftAgent},
		},
		{
			name:  "left agent alt j focuses draft",
			role:  PaneRoleLeftAgent,
			chord: ChordAltJ,
			want:  ShortcutDecision{Disposition: DispositionHandle, Action: ActionFocusLeftDraft},
		},
		{
			name:  "left pane tab helper is swallowed",
			role:  PaneRoleLeftDraft,
			chord: ChordAltR,
			want:  ShortcutDecision{Disposition: DispositionSwallow},
		},
		{
			name:  "left draft alt shift enter passes through",
			role:  PaneRoleLeftDraft,
			chord: ChordAltShiftEnter,
			want:  ShortcutDecision{Disposition: DispositionPass},
		},
		{
			name:  "other panes pass through",
			role:  PaneRoleOther,
			chord: ChordAltR,
			want:  ShortcutDecision{Disposition: DispositionPass},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(ShortcutInput{
				Role:           tt.role,
				Chord:          tt.chord,
				FocusedPaneID:  tt.focused,
				LastLeftPaneID: tt.lastLeft,
				DraftPaneID:    tt.draft,
			})
			if got != tt.want {
				t.Fatalf("Decide() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDecodeChord(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want Chord
		ok   bool
	}{
		{name: "legacy alt j", in: []byte("\x1bj"), want: ChordAltJ, ok: true},
		{name: "legacy alt k", in: []byte("\x1bk"), want: ChordAltK, ok: true},
		{name: "legacy alt x", in: []byte("\x1bx"), want: ChordAltX, ok: true},
		{name: "legacy alt slash", in: []byte("\x1b/"), want: ChordAltSlash, ok: true},
		{name: "legacy alt shift c", in: []byte("\x1bC"), want: ChordAltShiftC, ok: true},
		{name: "kkp alt t", in: []byte("\x1b[116;3u"), want: ChordAltT, ok: true},
		{name: "kkp alt x", in: []byte("\x1b[120;3u"), want: ChordAltX, ok: true},
		{name: "kkp alt shift d", in: []byte("\x1b[68;4u"), want: ChordAltShiftD, ok: true},
		{name: "kkp ctrl alt c", in: []byte("\x1b[99;7u"), want: ChordCtrlAltC, ok: true},
		{name: "kkp alt shift enter", in: []byte("\x1b[13;4u"), want: ChordAltShiftEnter, ok: true},
		{name: "ordinary text", in: []byte("t"), ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DecodeChord(tt.in)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("DecodeChord(%q) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestDecodeGlobalChord(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Chord
	}{
		{name: "alt d", in: "\x1b[100;3u", want: ChordAltD},
		{name: "alt x", in: "\x1b[120;3u", want: ChordAltX},
		{name: "alt n", in: "\x1b[110;3u", want: ChordAltN},
		{name: "ctrl alt n", in: "\x1b[110;7u", want: ChordCtrlAltN},
		{name: "shift alt n", in: "\x1b[78;4u", want: ChordAltShiftN},
		{name: "alt up", in: "\x1b[1;3A", want: ChordAltUp},
		{name: "alt down", in: "\x1b[1;3B", want: ChordAltDown},
		{name: "alt c", in: "\x1b[99;3u", want: ChordAltC},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DecodeChord([]byte(tt.in))
			if !ok || got != tt.want {
				t.Fatalf("DecodeChord(%q) = %v,%v; want %v,true", tt.in, got, ok, tt.want)
			}
			name := ChordName(got)
			roundTrip, ok := ChordFromName(name)
			if !ok || roundTrip != got {
				t.Fatalf("ChordFromName(ChordName(%v)) = %v,%v", got, roundTrip, ok)
			}
		})
	}
}

func TestGlobalDecisionMatrix(t *testing.T) {
	globals := []struct {
		chord  Chord
		action ShortcutAction
		lua    string
		focus  bool
	}{
		{ChordAltD, ActionConfirmDetach, "PairConfirmDetach", true},
		{ChordAltX, ActionConfirmQuit, "PairConfirmQuit", true},
		{ChordAltN, ActionRestartPair, "PairConfirmRestart", true},
		{ChordCtrlAltN, ActionRestartPair, "PairConfirmRestart", true},
		{ChordAltShiftN, ActionRestartAgent, "PairConfirmAgentRestart", true},
		{ChordAltUp, ActionGrowDraft, "PairLayoutBigger", false},
		{ChordAltDown, ActionShrinkDraft, "PairLayoutSmaller", false},
		{ChordAltC, ActionToggleReview, "PairReviewToggle", false},
	}
	roles := []PaneRole{PaneRoleLeftAgent, PaneRoleLeftDraft, PaneRoleRightTerminal}
	for _, global := range globals {
		for _, role := range roles {
			got := Decide(ShortcutInput{Role: role, Chord: global.chord})
			want := ShortcutDecision{
				Disposition:      DispositionHandle,
				Action:           global.action,
				DraftLuaFunction: global.lua,
				FocusDraft:       global.focus,
			}
			if got != want {
				t.Fatalf("Decide(role=%v, chord=%v) = %#v, want %#v", role, global.chord, got, want)
			}
		}
	}
}

func TestRenderedLuaGlobalMapsMatchCommittedFile(t *testing.T) {
	want := RenderLuaGlobalMaps()
	got, err := os.ReadFile(filepath.Join("..", "..", "..", "nvim", "workbench_actions.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("nvim/workbench_actions.lua is stale\n\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestDecodeAltArrowChords(t *testing.T) {
	tests := []struct {
		seq  string
		want Chord
	}{
		{seq: "\x1b[1;3D", want: ChordAltLeft},
		{seq: "\x1b[1;3C", want: ChordAltRight},
		{seq: "\x1b[1;9D", want: ChordAltLeft},
		{seq: "\x1b[1;9C", want: ChordAltRight},
	}
	for _, tt := range tests {
		got, ok := DecodeChord([]byte(tt.seq))
		if !ok || got != tt.want {
			t.Fatalf("DecodeChord(%q) = %v,%v; want %v,true", tt.seq, got, ok, tt.want)
		}
	}
}

func TestLastLeftPaneStore(t *testing.T) {
	dir := t.TempDir()
	store := LastLeftPaneStore{DataDir: dir, Tag: "pair"}
	if got := store.Path(); got != filepath.Join(dir, "last-left-pane-pair") {
		t.Fatalf("Path() = %q", got)
	}
	if got, err := store.Read(); err != nil || got != "" {
		t.Fatalf("Read missing = (%q, %v), want empty nil", got, err)
	}
	if err := store.Write("42"); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got, err := store.Read(); err != nil || got != "42" {
		t.Fatalf("Read() = (%q, %v), want 42 nil", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".last-left-pane-pair.*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}

	emptyTag := LastLeftPaneStore{DataDir: dir}
	if got := emptyTag.Path(); got != filepath.Join(dir, "last-left-pane-pair") {
		t.Fatalf("empty tag Path() = %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "last-left-pane-pair")); err != nil {
		t.Fatalf("expected written sidecar: %v", err)
	}
}

func TestLastTerminalPaneStoreIsDistinctFromLastLeft(t *testing.T) {
	dir := t.TempDir()
	store := LastTerminalPaneStore{DataDir: dir, Tag: "pair"}
	if got := store.Path(); got != filepath.Join(dir, "last-terminal-pane-pair") {
		t.Fatalf("Path() = %q", got)
	}
	if err := store.Write("7"); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Read(); err != nil || got != "7" {
		t.Fatalf("Read() = (%q, %v), want 7 nil", got, err)
	}
	left := LastLeftPaneStore{DataDir: dir, Tag: "pair"}
	if got, err := left.Read(); err != nil || got != "" {
		t.Fatalf("left store polluted: (%q, %v)", got, err)
	}
}

func TestTerminalPaneRegistry(t *testing.T) {
	dir := t.TempDir()
	reg := TerminalPaneRegistry{DataDir: dir, Tag: "pair"}
	if err := reg.Register("4", 100); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register("7", 200); err != nil {
		t.Fatal(err)
	}
	alive := func(pid int) bool { return pid == 100 }
	ids, err := reg.LiveIDs(alive)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "4" {
		t.Fatalf("LiveIDs = %v, want [4] (dead pids filtered)", ids)
	}
	empty := TerminalPaneRegistry{DataDir: dir, Tag: "other"}
	if ids, err := empty.LiveIDs(alive); err != nil || len(ids) != 0 {
		t.Fatalf("missing registry = (%v, %v), want empty nil", ids, err)
	}
}

func TestRoleForPaneWithRegisteredTerminals(t *testing.T) {
	// zellij 0.44.3 reports terminal_command=null for panes created via
	// `action new-pane --direction`, and the #118 tab-strip title
	// ("[terminal 1]") defeats the title fallback — the registry is the
	// authoritative signal for pair-owned terminal panes.
	split := zellijpane.Pane{ID: "4", IsFocused: true, Title: "[terminal 1]"}
	if got := RoleForPane(split); got != PaneRoleOther {
		t.Fatalf("RoleForPane = %v, want Other without registry", got)
	}
	if got := RoleForPaneWith(split, []string{"4"}); got != PaneRoleRightTerminal {
		t.Fatalf("RoleForPaneWith = %v, want RightTerminal via registry", got)
	}
	review := zellijpane.Pane{ID: "9", IsFocused: true, Title: "review",
		TerminalCommand: "nvim -u /pair/nvim/review.lua /tmp/review.md"}
	if got := RoleForPaneWith(review, []string{"4"}); got != PaneRoleOther {
		t.Fatalf("RoleForPaneWith(review) = %v, want Other", got)
	}
}

// Every global chord must carry help text. This is the "cannot ship undocumented"
// property for the 8 chords whose wording lives nowhere else — they reach nvim via
// the GENERATED workbench_actions.lua, not literal keymap.set calls, so parsing
// init.lua would silently miss them (#132).
func TestEveryGlobalBindingHasHelp(t *testing.T) {
	for _, b := range GlobalBindings() {
		if strings.TrimSpace(b.Help) == "" {
			t.Errorf("chord %v (%s) has no Help text", b.Chord, b.NvimKey)
		}
	}
}

// The role table must cover every chord Decide actually handles for the right
// terminal, or the help silently omits a working key. Their nvim descs describe the
// deliberate draft NO-OP ("right-terminal tab helper disabled in draft"), so there
// is no honest alternative wording source (#132 PQ-2).
func TestRoleBindingsCoverTerminalSwitch(t *testing.T) {
	handled := []Chord{ChordAltT, ChordAltW, ChordAltR, ChordAltShiftD, ChordAltShiftEnter, ChordAltK}
	for _, chord := range handled {
		found := false
		for _, rb := range RoleBindings() {
			if rb.Chord != chord {
				continue
			}
			found = true
			if strings.TrimSpace(rb.Help) == "" {
				t.Errorf("chord %v is handled for the right terminal but has no Help", chord)
			}
		}
		if !found {
			t.Errorf("chord %v is handled for the right terminal but is absent from roleBindings", chord)
		}
	}
}
