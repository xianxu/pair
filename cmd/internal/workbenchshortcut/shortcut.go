// Package workbenchshortcut contains Pair's pane-local workbench shortcut
// decisions. The package stays pure except for LastLeftPaneStore's small
// sidecar helpers, so pane wrappers can share the same semantics.
package workbenchshortcut

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
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
	ChordAltD
	ChordAltShiftD
	ChordAltX
	ChordAltN
	ChordCtrlAltN
	ChordAltShiftN
	ChordAltUp
	ChordAltDown
	ChordAltC
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
	ActionConfirmDetach
	ActionConfirmQuit
	ActionRestartPair
	ActionRestartAgent
	ActionSplitTerminalDown
	ActionGrowDraft
	ActionShrinkDraft
	ActionToggleReview
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
	Disposition              Disposition
	Action                   ShortcutAction
	TargetPaneID             string
	RecordLastLeftPaneID     string
	RecordLastTerminalPaneID string
	DraftLuaFunction         string
	FocusDraft               bool
}

type GlobalBinding struct {
	Chord       Chord
	Action      ShortcutAction
	LuaFunction string
	NvimKey     string
	FocusDraft  bool
	// Help is the user-facing description shown by `pair keys` / Alt+h (#132).
	// It is authored HERE because these chords reach nvim through the generated
	// workbench_actions.lua rather than literal vim.keymap.set calls, so no
	// `desc = 'pair: …'` exists for them to derive from. Not rendered into Lua.
	Help string
}

// Keyed literals (not positional): #132 added Help, and a positional list makes
// every future field a silent shift of the one before it.
var globalBindings = []GlobalBinding{
	{Chord: ChordAltD, Action: ActionConfirmDetach, LuaFunction: "PairConfirmDetach", NvimKey: "<M-d>", FocusDraft: true,
		Help: "detach from the session (re-attach with `pair`)"},
	{Chord: ChordAltX, Action: ActionConfirmQuit, LuaFunction: "PairConfirmQuit", NvimKey: "<M-x>", FocusDraft: true,
		Help: "full quit — kill the session and drop it from the resurrect list"},
	{Chord: ChordAltN, Action: ActionRestartPair, LuaFunction: "PairConfirmRestart", NvimKey: "<M-n>", FocusDraft: true,
		Help: "reload pair — kill and re-launch the workbench in place"},
	{Chord: ChordCtrlAltN, Action: ActionRestartPair, LuaFunction: "PairConfirmRestart", NvimKey: "<C-M-n>", FocusDraft: true,
		Help: "reload pair (same as Alt+n)"},
	{Chord: ChordAltShiftN, Action: ActionRestartAgent, LuaFunction: "PairConfirmAgentRestart", NvimKey: "<M-N>", FocusDraft: true,
		Help: "restart only the agent conversation, keeping the workbench"},
	{Chord: ChordAltUp, Action: ActionGrowDraft, LuaFunction: "PairLayoutBigger", NvimKey: "<M-Up>", FocusDraft: false,
		Help: "grow the draft pane along the height ladder"},
	{Chord: ChordAltDown, Action: ActionShrinkDraft, LuaFunction: "PairLayoutSmaller", NvimKey: "<M-Down>", FocusDraft: false,
		Help: "shrink the draft pane along the height ladder"},
	{Chord: ChordAltC, Action: ActionToggleReview, LuaFunction: "PairReviewToggle", NvimKey: "<M-c>", FocusDraft: false,
		Help: "open / show / hide the review pane"},
}

// RoleBinding describes a chord whose behaviour is PANE-LOCAL — it does something
// different, or nothing, outside its role.
//
// These need their own wording home because neither existing source can supply it:
// their behaviour lives in Decide's switch (not enumerable), and their nvim keymaps
// describe the deliberate draft NO-OP ("right-terminal tab helper disabled in
// draft", init.lua:3653-3658). Deriving from nvim would publish "disabled in draft"
// as the description of a working feature (#132).
//
// The table describes the switch; it does not drive it. TestRoleBindingsCoverTerminalSwitch
// keeps the two from diverging.
type RoleBinding struct {
	Chord Chord
	Role  PaneRole
	Help  string
}

var roleBindings = []RoleBinding{
	{Chord: ChordAltT, Role: PaneRoleRightTerminal, Help: "new terminal tab"},
	{Chord: ChordAltW, Role: PaneRoleRightTerminal, Help: "close the current terminal tab"},
	{Chord: ChordAltR, Role: PaneRoleRightTerminal, Help: "rename the current terminal tab"},
	{Chord: ChordAltShiftD, Role: PaneRoleRightTerminal, Help: "split a second terminal below"},
	{Chord: ChordAltK, Role: PaneRoleRightTerminal, Help: "jump back to the left pane you came from"},
	{Chord: ChordAltShiftEnter, Role: PaneRoleRightTerminal, Help: "toggle the focused side's width"},
	// Handled by termcmd.handleTerminalChord (run.go:484-489), NOT by Decide — the
	// terminal chord surface is split across two seams and their sets differ. #132's
	// first cut documented only Decide's, so "Terminal tabs" rendered with no way to
	// change tabs. TestRoleBindingsCoverTerminalSwitch now derives from both.
	{Chord: ChordAltLeft, Role: PaneRoleRightTerminal, Help: "previous terminal tab"},
	{Chord: ChordAltRight, Role: PaneRoleRightTerminal, Help: "next terminal tab"},
}

// RoleBindings returns the pane-local chord descriptions.
func RoleBindings() []RoleBinding {
	return append([]RoleBinding(nil), roleBindings...)
}

func GlobalBindings() []GlobalBinding {
	return append([]GlobalBinding(nil), globalBindings...)
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
	if in.Role == PaneRoleLeftAgent || in.Role == PaneRoleLeftDraft || in.Role == PaneRoleRightTerminal {
		if decision, ok := DecideGlobal(in.Chord); ok {
			return decision
		}
	}
	switch in.Role {
	case PaneRoleRightTerminal:
		switch in.Chord {
		case ChordAltT:
			return handle(ActionNewTab)
		case ChordAltW:
			return handle(ActionCloseTab)
		case ChordAltR:
			return handle(ActionRenameTab)
		case ChordAltShiftD:
			return handle(ActionSplitTerminalDown)
		case ChordAltK:
			target := in.LastLeftPaneID
			if target == "" {
				target = in.DraftPaneID
			}
			// Record which split half the user is leaving: in the tiled tree
			// no right terminal reports is_focused while focus sits in the
			// left stack, so this file is the only memory the return jump
			// (FocusRightTerminal) has of the last-used half.
			return ShortcutDecision{
				Disposition:              DispositionHandle,
				Action:                   ActionFocusPane,
				TargetPaneID:             target,
				RecordLastTerminalPaneID: in.FocusedPaneID,
			}
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
		case ChordAltT, ChordAltW, ChordAltR:
			return ShortcutDecision{Disposition: DispositionSwallow}
		default:
			return ShortcutDecision{Disposition: DispositionPass}
		}
	default:
		return ShortcutDecision{Disposition: DispositionPass}
	}
}

// DecideGlobal resolves a workbench-wide chord without requiring pane
// inventory. Pair-owned input wrappers already establish that the chord came
// from a primary pane; only pane-relative shortcuts need Role/geometry data.
func DecideGlobal(chord Chord) (ShortcutDecision, bool) {
	if binding, ok := globalDraftAction(chord); ok {
		return ShortcutDecision{
			Disposition:      DispositionHandle,
			Action:           binding.Action,
			DraftLuaFunction: binding.LuaFunction,
			FocusDraft:       binding.FocusDraft,
		}, true
	}
	return ShortcutDecision{}, false
}

func globalDraftAction(chord Chord) (GlobalBinding, bool) {
	for _, binding := range globalBindings {
		if binding.Chord == chord {
			return binding, true
		}
	}
	return GlobalBinding{}, false
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
	{"\x1b[100;3u", ChordAltD},
	{"\x1bD", ChordAltShiftD}, {"\x1b[68;4u", ChordAltShiftD},
	{"\x1bx", ChordAltX}, {"\x1b[120;3u", ChordAltX},
	{"\x1b[110;3u", ChordAltN},
	{"\x1b[110;7u", ChordCtrlAltN},
	{"\x1b[78;4u", ChordAltShiftN},
	{"\x1b[1;3A", ChordAltUp},
	{"\x1b[1;3B", ChordAltDown},
	{"\x1b[99;3u", ChordAltC},
	{"\x1b/", ChordAltSlash}, {"\x1b[47;3u", ChordAltSlash},
	{"\x1bC", ChordAltShiftC}, {"\x1b[67;3u", ChordAltShiftC},
	{"\x1b[99;7u", ChordCtrlAltC},
	{"\x1b[1;3D", ChordAltLeft}, {"\x1b[1;9D", ChordAltLeft}, {"\x1b[3D", ChordAltLeft},
	{"\x1b[1;3C", ChordAltRight}, {"\x1b[1;9C", ChordAltRight}, {"\x1b[3C", ChordAltRight},
	{"\x1b[13;4u", ChordAltShiftEnter},
}

func ChordSequences() []string {
	sequences := make([]string, 0, len(chordSequences))
	for _, candidate := range chordSequences {
		sequences = append(sequences, candidate.sequence)
	}
	return sequences
}

// ChordEncodings returns defensive copies of the canonical byte encodings for
// one chord. Consumers that intercept a specific Pair shortcut therefore do
// not duplicate terminal-protocol literals.
func ChordEncodings(chord Chord) [][]byte {
	var sequences [][]byte
	for _, candidate := range chordSequences {
		if candidate.chord == chord {
			sequences = append(sequences, []byte(candidate.sequence))
		}
	}
	return sequences
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
	case ChordAltD:
		return "Alt+d"
	case ChordAltShiftD:
		return "Alt+Shift+d"
	case ChordAltX:
		return "Alt+x"
	case ChordAltN:
		return "Alt+n"
	case ChordCtrlAltN:
		return "Ctrl+Alt+n"
	case ChordAltShiftN:
		return "Alt+Shift+N"
	case ChordAltUp:
		return "Alt+Up"
	case ChordAltDown:
		return "Alt+Down"
	case ChordAltC:
		return "Alt+c"
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

// LastLeftPaneStore and LastTerminalPaneStore remember the pane the user last
// left on each side of the workbench, so the opposite jump (Alt+k) can return
// to it. Same file shape, distinct sidecars per tag.
type LastLeftPaneStore struct {
	DataDir string
	Tag     string
}

func (s LastLeftPaneStore) Path() string {
	paths, ok := panePaths(s.DataDir, s.Tag)
	if !ok {
		return ""
	}
	return paths.LastLeftPane()
}
func (s LastLeftPaneStore) Read() (string, error)     { return readPaneID(s.Path()) }
func (s LastLeftPaneStore) Write(paneID string) error { return writePaneID(s.Path(), paneID) }

type LastTerminalPaneStore struct {
	DataDir string
	Tag     string
}

func (s LastTerminalPaneStore) Path() string {
	paths, ok := panePaths(s.DataDir, s.Tag)
	if !ok {
		return ""
	}
	return paths.LastTerminalPane()
}
func (s LastTerminalPaneStore) Read() (string, error)     { return readPaneID(s.Path()) }
func (s LastTerminalPaneStore) Write(paneID string) error { return writePaneID(s.Path(), paneID) }

// TerminalPaneRegistry records which zellij pane ids host a live `pair term`
// process. It exists because pane identity cannot be derived from zellij's
// pane report alone: zellij 0.44.3 omits `terminal_command` for panes created
// via `action new-pane --direction` (the Alt+Shift+d split), and the #118
// tab-strip pane title ("[terminal 1]", tabs user-renamable) defeats any
// title heuristic. Each `pair term` appends its pane id + pid at startup;
// readers filter by process liveness, so stale entries self-expire.
type TerminalPaneRegistry struct {
	DataDir string
	Tag     string
}

func (r TerminalPaneRegistry) Path() string {
	paths, ok := panePaths(r.DataDir, r.Tag)
	if !ok {
		return ""
	}
	return paths.TerminalPanes()
}

func (r TerminalPaneRegistry) Register(paneID string, pid int) error {
	path := r.Path()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s %d\n", strings.TrimSpace(paneID), pid)
	return err
}

// LiveIDs returns the pane ids whose registering process is still alive —
// first live entry per id wins (file order; dead pids never block newer
// entries for other ids). The sidecar is append-only with no compaction:
// entries are one short line per pair-term start, so growth is negligible
// and liveness filtering makes stale lines inert. Liveness is injected so
// the filtering stays testable without real processes.
func (r TerminalPaneRegistry) LiveIDs(alive func(pid int) bool) ([]string, error) {
	data, err := os.ReadFile(r.Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	seen := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || seen[fields[0]] {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil || !alive(pid) {
			continue
		}
		seen[fields[0]] = true
		ids = append(ids, fields[0])
	}
	return ids, nil
}

// RoleForPaneWith is RoleForPane with the terminal-pane registry overlaid:
// a pane whose id is registered as a live `pair term` is a right terminal
// even when zellij's pane report carries no usable command or title.
func RoleForPaneWith(p zellijpane.Pane, terminalPaneIDs []string) PaneRole {
	role := RoleForPane(p)
	if role != PaneRoleOther || p.IsPlugin || p.ID == "" {
		return role
	}
	for _, id := range terminalPaneIDs {
		if id == p.ID {
			return PaneRoleRightTerminal
		}
	}
	return role
}

// LiveTerminalPaneIDsFromEnv reads the terminal-pane registry for the current
// PAIR_DATA_DIR/PAIR_TAG and returns the pane ids whose registering process is
// still alive. Every OSRuntime that needs "which panes are pair terminals"
// delegates here — layoutcmd (focus + width toggle), termcmd (chord routing),
// and clipcmd (the #125 auto-paste gate) — so the registry construction has one
// source rather than a copy per consumer. Liveness is injected so callers can
// keep using their own procutil wrapper without this package importing it.
func LiveTerminalPaneIDsFromEnv(alive func(pid int) bool) ([]string, error) {
	reg := TerminalPaneRegistry{DataDir: DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
	return reg.LiveIDs(alive)
}

// DataDirFromEnv resolves pair's data dir by delegating to the canonical
// resolver (adapt.DataDir) — one source for the PAIR_DATA_DIR → XDG →
// ~/.local/share/pair chain.
func DataDirFromEnv() string {
	return adapt.DataDir()
}

func panePaths(dataDir, tag string) (artifactpath.Paths, bool) {
	if tag == "" {
		tag = "pair"
	}
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	return paths, err == nil
}

func readPaneID(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writePaneID(path, paneID string) error {
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
