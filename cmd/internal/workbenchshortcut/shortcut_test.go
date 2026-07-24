package workbenchshortcut

import (
	"os"
	"path/filepath"
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
			name: "right terminal title fallback",
			pane: zellijpane.Pane{ID: "3", IsFocused: true, Title: "terminal"},
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
			name:  "right terminal alt j is no-op",
			role:  PaneRoleRightTerminal,
			chord: ChordAltJ,
			want:  ShortcutDecision{Disposition: DispositionSwallow},
		},
		{
			name:     "right terminal alt k returns to last left pane",
			role:     PaneRoleRightTerminal,
			chord:    ChordAltK,
			lastLeft: "1",
			draft:    "2",
			want:     ShortcutDecision{Disposition: DispositionHandle, Action: ActionFocusPane, TargetPaneID: "1"},
		},
		{
			name:  "right terminal alt k falls back to draft",
			role:  PaneRoleRightTerminal,
			chord: ChordAltK,
			draft: "2",
			want:  ShortcutDecision{Disposition: DispositionHandle, Action: ActionFocusPane, TargetPaneID: "2"},
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
		{name: "legacy alt slash", in: []byte("\x1b/"), want: ChordAltSlash, ok: true},
		{name: "legacy alt shift c", in: []byte("\x1bC"), want: ChordAltShiftC, ok: true},
		{name: "kkp alt t", in: []byte("\x1b[116;3u"), want: ChordAltT, ok: true},
		{name: "kkp ctrl alt c", in: []byte("\x1b[99;7u"), want: ChordCtrlAltC, ok: true},
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
