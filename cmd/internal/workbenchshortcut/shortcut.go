// Package workbenchshortcut contains Pair's pane-local workbench shortcut
// decisions. The package stays pure except for LastLeftPaneStore's small
// sidecar helpers, so pane wrappers can share the same semantics.
package workbenchshortcut

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type PaneRole int

const (
	PaneRoleOther PaneRole = iota
	PaneRoleLeftAgent
	PaneRoleLeftDraft
	PaneRoleRightTerminal
)

type Chord int

const (
	ChordUnknown Chord = iota
	ChordAltJ
	ChordAltK
	ChordAltT
	ChordAltW
	ChordAltR
	ChordAltSlash
	ChordAltShiftC
	ChordCtrlAltC
)

type Disposition int

const (
	DispositionPass Disposition = iota
	DispositionSwallow
	DispositionHandle
)

type ShortcutAction int

const (
	ActionNone ShortcutAction = iota
	ActionFocusPane
	ActionFocusLeftAgent
	ActionFocusLeftDraft
	ActionFocusRightTerminal
	ActionOpenScrollback
	ActionConfirmCompact
)

type ShortcutInput struct {
	Role           PaneRole
	Chord          Chord
	FocusedPaneID  string
	LastLeftPaneID string
	DraftPaneID    string
}

type ShortcutDecision struct {
	Disposition          Disposition
	Action               ShortcutAction
	TargetPaneID         string
	RecordLastLeftPaneID string
}

func RoleForPane(p zellijpane.Pane) PaneRole {
	if p.IsFloating || p.IsPlugin {
		return PaneRoleOther
	}
	cmd := strings.ToLower(p.TerminalCommand)
	title := strings.ToLower(strings.TrimSpace(p.Title))
	switch {
	case strings.Contains(cmd, "pair wrap"):
		return PaneRoleLeftAgent
	case strings.Contains(cmd, "nvim") && strings.Contains(cmd, "/nvim/init.lua"):
		return PaneRoleLeftDraft
	case strings.Contains(cmd, "pair term"), title == "terminal":
		return PaneRoleRightTerminal
	default:
		return PaneRoleOther
	}
}

func Decide(in ShortcutInput) ShortcutDecision {
	switch in.Role {
	case PaneRoleRightTerminal:
		switch in.Chord {
		case ChordAltK:
			target := in.LastLeftPaneID
			if target == "" {
				target = in.DraftPaneID
			}
			return ShortcutDecision{Disposition: DispositionHandle, Action: ActionFocusPane, TargetPaneID: target}
		case ChordAltJ, ChordAltSlash, ChordAltShiftC, ChordCtrlAltC:
			return ShortcutDecision{Disposition: DispositionSwallow}
		default:
			return ShortcutDecision{Disposition: DispositionPass}
		}
	case PaneRoleLeftAgent, PaneRoleLeftDraft:
		switch in.Chord {
		case ChordAltJ:
			if in.Role == PaneRoleLeftAgent {
				return handle(ActionFocusLeftDraft)
			}
			return handle(ActionFocusLeftAgent)
		case ChordAltK:
			return ShortcutDecision{
				Disposition:          DispositionHandle,
				Action:               ActionFocusRightTerminal,
				RecordLastLeftPaneID: in.FocusedPaneID,
			}
		case ChordAltSlash:
			return handle(ActionOpenScrollback)
		case ChordAltShiftC, ChordCtrlAltC:
			return handle(ActionConfirmCompact)
		case ChordAltT, ChordAltW, ChordAltR:
			return ShortcutDecision{Disposition: DispositionSwallow}
		default:
			return ShortcutDecision{Disposition: DispositionPass}
		}
	default:
		return ShortcutDecision{Disposition: DispositionPass}
	}
}

func handle(action ShortcutAction) ShortcutDecision {
	return ShortcutDecision{Disposition: DispositionHandle, Action: action}
}

func DecodeChord(data []byte) (Chord, bool) {
	switch string(data) {
	case "\x1bj", "\x1b[106;3u":
		return ChordAltJ, true
	case "\x1bk", "\x1b[107;3u":
		return ChordAltK, true
	case "\x1bt", "\x1b[116;3u":
		return ChordAltT, true
	case "\x1bw", "\x1b[119;3u":
		return ChordAltW, true
	case "\x1br", "\x1b[114;3u":
		return ChordAltR, true
	case "\x1b/", "\x1b[47;3u":
		return ChordAltSlash, true
	case "\x1bC", "\x1b[67;3u":
		return ChordAltShiftC, true
	case "\x1b[99;7u":
		return ChordCtrlAltC, true
	default:
		return ChordUnknown, false
	}
}

type LastLeftPaneStore struct {
	DataDir string
	Tag     string
}

func (s LastLeftPaneStore) Path() string {
	tag := s.Tag
	if tag == "" {
		tag = "pair"
	}
	return filepath.Join(s.DataDir, "last-left-pane-"+tag)
}

func (s LastLeftPaneStore) Read() (string, error) {
	data, err := os.ReadFile(s.Path())
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (s LastLeftPaneStore) Write(paneID string) error {
	path := s.Path()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(strings.TrimSpace(paneID) + "\n"); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
