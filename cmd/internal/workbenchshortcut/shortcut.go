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
	ChordAltX
	ChordAltSlash
	ChordAltShiftC
	ChordCtrlAltC
	ChordAltLeft
	ChordAltRight
	ChordAltShiftEnter
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
	ActionNewTab
	ActionCloseTab
	ActionRenameTab
	ActionFocusPane
	ActionFocusLeftAgent
	ActionFocusLeftDraft
	ActionFocusRightTerminal
	ActionOpenScrollback
	ActionConfirmCompact
	ActionConfirmQuit
	ActionToggleFocusedLayout
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
	if p.IsPlugin {
		return PaneRoleOther
	}
	cmd := strings.ToLower(p.TerminalCommand)
	title := strings.ToLower(strings.TrimSpace(p.Title))
	switch {
	case strings.Contains(cmd, "pair wrap"):
		return PaneRoleLeftAgent
	case strings.Contains(cmd, "nvim") && strings.Contains(cmd, "/nvim/init.lua"):
		return PaneRoleLeftDraft
	case strings.Contains(cmd, "pair term"), title == "terminal", strings.HasPrefix(title, "terminal "):
		return PaneRoleRightTerminal
	default:
		return PaneRoleOther
	}
}

func Decide(in ShortcutInput) ShortcutDecision {
	switch in.Role {
	case PaneRoleRightTerminal:
		switch in.Chord {
		case ChordAltT:
			return handle(ActionNewTab)
		case ChordAltW:
			return handle(ActionCloseTab)
		case ChordAltR:
			return handle(ActionRenameTab)
		case ChordAltX:
			return handle(ActionConfirmQuit)
		case ChordAltK:
			target := in.LastLeftPaneID
			if target == "" {
				target = in.DraftPaneID
			}
			return ShortcutDecision{Disposition: DispositionHandle, Action: ActionFocusPane, TargetPaneID: target}
		case ChordAltShiftEnter:
			return handle(ActionToggleFocusedLayout)
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
		case ChordAltX:
			return handle(ActionConfirmQuit)
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

var chordSequences = []struct {
	sequence string
	chord    Chord
}{
	{"\x1bj", ChordAltJ}, {"\x1b[106;3u", ChordAltJ},
	{"\x1bk", ChordAltK}, {"\x1b[107;3u", ChordAltK},
	{"\x1bt", ChordAltT}, {"\x1b[116;3u", ChordAltT},
	{"\x1bw", ChordAltW}, {"\x1b[119;3u", ChordAltW},
	{"\x1br", ChordAltR}, {"\x1b[114;3u", ChordAltR},
	{"\x1bx", ChordAltX}, {"\x1b[120;3u", ChordAltX},
	{"\x1b/", ChordAltSlash}, {"\x1b[47;3u", ChordAltSlash},
	{"\x1bC", ChordAltShiftC}, {"\x1b[67;3u", ChordAltShiftC},
	{"\x1b[99;7u", ChordCtrlAltC},
	{"\x1b[1;3D", ChordAltLeft}, {"\x1b[1;9D", ChordAltLeft}, {"\x1b[3D", ChordAltLeft},
	{"\x1b[1;3C", ChordAltRight}, {"\x1b[1;9C", ChordAltRight}, {"\x1b[3C", ChordAltRight},
	{"\x1b[13;4u", ChordAltShiftEnter},
}

func DecodeChord(data []byte) (Chord, bool) {
	for _, candidate := range chordSequences {
		if string(data) == candidate.sequence {
			return candidate.chord, true
		}
	}
	return ChordUnknown, false
}

func DecodeChordPrefix(data []byte) (Chord, []byte, bool) {
	for _, candidate := range chordSequences {
		if strings.HasPrefix(string(data), candidate.sequence) {
			return candidate.chord, data[len(candidate.sequence):], true
		}
	}
	return ChordUnknown, data, false
}

func FindChord(data []byte) ([]byte, Chord, []byte, []byte, bool) {
	for offset := range data {
		for _, candidate := range chordSequences {
			if strings.HasPrefix(string(data[offset:]), candidate.sequence) {
				end := offset + len(candidate.sequence)
				return data[:offset], candidate.chord, data[offset:end], data[end:], true
			}
		}
	}
	return data, ChordUnknown, nil, nil, false
}

func ChordName(chord Chord) string {
	switch chord {
	case ChordAltJ:
		return "Alt+j"
	case ChordAltK:
		return "Alt+k"
	case ChordAltT:
		return "Alt+t"
	case ChordAltW:
		return "Alt+w"
	case ChordAltR:
		return "Alt+r"
	case ChordAltX:
		return "Alt+x"
	case ChordAltSlash:
		return "Alt+/"
	case ChordAltShiftC:
		return "Alt+Shift+C"
	case ChordCtrlAltC:
		return "Ctrl+Alt+c"
	case ChordAltLeft:
		return "Alt+Left"
	case ChordAltRight:
		return "Alt+Right"
	case ChordAltShiftEnter:
		return "Alt+Shift+Enter"
	default:
		return ""
	}
}

func ChordFromName(name string) (Chord, bool) {
	for _, candidate := range chordSequences {
		if ChordName(candidate.chord) == name {
			return candidate.chord, true
		}
	}
	return ChordUnknown, false
}

func IsChordPrefix(data []byte) bool {
	for _, candidate := range chordSequences {
		if len(data) < len(candidate.sequence) && strings.HasPrefix(candidate.sequence, string(data)) {
			return true
		}
	}
	return false
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
