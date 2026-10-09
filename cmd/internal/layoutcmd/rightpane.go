package layoutcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// RightPaneMode is the right terminal's view, derived from zellij's pane report
// alone (#417). Zellij owns the state; nothing persists it, so a native change
// (focus-away leaving fullscreen, a pane closed) can never leave Pair believing
// a stale mode. Alt+Shift+Enter cycles Normal → Focus → Maximize → Normal.
type RightPaneMode uint8

const (
	// ModeNormal: every right terminal tiled, none fullscreen.
	ModeNormal RightPaneMode = iota
	// ModeFocus: one right terminal floats, centered over the dimmed workbench.
	ModeFocus
	// ModeMaximize: one tiled right terminal is zellij-fullscreen.
	ModeMaximize
)

// ObserveRightPaneMode classifies a pane report. A floating right terminal
// decides Focus before fullscreen is consulted: Pair never fullscreens a
// floating pane, and its fullscreen flag is irrelevant to the cycle.
func ObserveRightPaneMode(panes []zellijpane.Pane, registered []string) (RightPaneMode, string, error) {
	var floating, fullscreen string
	for _, p := range panes {
		if p.IsPlugin || !isRightTerminal(p, registered) {
			continue
		}
		if p.IsFloating {
			if floating != "" {
				return ModeNormal, "", fmt.Errorf("ambiguous floating right terminals %s and %s", floating, p.ID)
			}
			floating = p.ID
			continue
		}
		if p.IsFullscreen != nil && *p.IsFullscreen {
			fullscreen = p.ID
		}
	}
	switch {
	case floating != "":
		return ModeFocus, floating, nil
	case fullscreen != "":
		return ModeMaximize, fullscreen, nil
	}
	return ModeNormal, "", nil
}

// FocusModeActive is the one predicate pair-wrap dims the agent pane by, so the
// dim and the cycle cannot disagree about whether focus mode is on.
func FocusModeActive(panes []zellijpane.Pane, registered []string) bool {
	mode, _, err := ObserveRightPaneMode(panes, registered)
	return err == nil && mode == ModeFocus
}

// ExpandRecord is what leaving Normal must remember to come back: the pane to
// return focus to, plus — for focus mode, whose re-embed does not restore the
// tiled slot — the swap layout and split-half order to re-tile onto. It rides
// the existing fullscreen-return record as one line ("<return> swap=<name>
// order=<id>,<id>"), so it adds no artifact family; a bare "<id>" written by an
// older binary decodes as {Return: id}.
type ExpandRecord struct {
	Return string
	Swap   string
	Order  []string
}

func (r ExpandRecord) Encode() string {
	fields := []string{r.Return}
	if r.Swap != "" {
		fields = append(fields, "swap="+r.Swap)
	}
	if len(r.Order) > 0 {
		fields = append(fields, "order="+strings.Join(r.Order, ","))
	}
	return strings.Join(fields, " ")
}

// DecodeExpandRecord is total: unknown tokens are ignored, so a newer writer's
// fields cannot break an older reader's return jump.
func DecodeExpandRecord(line string) ExpandRecord {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ExpandRecord{}
	}
	r := ExpandRecord{Return: fields[0]}
	for _, f := range fields[1:] {
		switch {
		case strings.HasPrefix(f, "swap="):
			r.Swap = strings.TrimPrefix(f, "swap=")
		case strings.HasPrefix(f, "order="):
			r.Order = strings.Split(strings.TrimPrefix(f, "order="), ",")
		}
	}
	return r
}

type StepKind uint8

const (
	StepSave       StepKind = iota // write the plan's ExpandRecord
	StepFloat                      // tiled → floating
	StepPlace                      // center, 75% wide, full height, pinned
	StepShow                       // a CLI float leaves the floating layer hidden
	StepEmbed                      // floating → tiled (lands wherever zellij likes)
	StepRestore                    // re-tile onto the recorded swap layout + half order
	StepFullscreen                 // toggle zellij fullscreen
	StepFocus                      // focus-pane-id
	StepClear                      // remove the record
	StepNudge                      // SIGWINCH pair-wrap so the dim follows the mode
)

type Step struct {
	Kind StepKind
	Pane string
}

// RightPanePlan is the single owner of operation order for all three
// transitions. The executor runs Steps in order and stops at the first failure;
// a failed or unconfirmed effect never authorizes an inverse one.
type RightPanePlan struct {
	From, To RightPaneMode
	Record   ExpandRecord
	Steps    []Step
}

type RightPaneInput struct {
	Panes        []zellijpane.Pane
	Caller, Last string
	Registered   []string
	Record       ExpandRecord
	// Swap is the tab's active swap layout name now; only Normal→Focus records it.
	Swap string
}

func PlanRightPane(in RightPaneInput) (RightPanePlan, error) {
	mode, id, err := ObserveRightPaneMode(in.Panes, in.Registered)
	if err != nil {
		return RightPanePlan{}, err
	}
	if mode == ModeFocus {
		return RightPanePlan{From: ModeFocus, To: ModeMaximize, Record: in.Record, Steps: []Step{
			{StepEmbed, id}, {StepRestore, id}, {StepFullscreen, id}, {StepNudge, ""},
		}}, nil
	}
	fp, err := PlanFullscreen(in.Panes, in.Caller, in.Last, in.Registered, in.Record.Return)
	if err != nil {
		return RightPanePlan{}, err
	}
	switch fp.Operation {
	case FullscreenCollapse:
		steps := []Step{{StepFullscreen, fp.TerminalID}}
		if fp.ReturnID != "" && fp.ReturnID != fp.TerminalID {
			steps = append(steps, Step{StepFocus, fp.ReturnID})
		}
		return RightPanePlan{From: ModeMaximize, To: ModeNormal, Record: in.Record, Steps: append(steps, Step{StepClear, ""})}, nil
	case FullscreenExpand:
		record := ExpandRecord{Return: fp.ReturnID, Swap: in.Swap, Order: splitOrder(in.Panes, in.Registered)}
		return RightPanePlan{From: ModeNormal, To: ModeFocus, Record: record, Steps: []Step{
			{StepSave, ""}, {StepFloat, fp.TerminalID}, {StepPlace, fp.TerminalID}, {StepShow, ""}, {StepNudge, ""},
		}}, nil
	}
	return RightPanePlan{}, nil
}

// splitOrder lists the tiled right terminals top to bottom when the column is
// split — the order a re-embed can scramble (probe: the floated top half came
// back as the bottom one). One terminal has no order to lose.
func splitOrder(panes []zellijpane.Pane, registered []string) []string {
	var halves []zellijpane.Pane
	for _, p := range panes {
		if !p.IsPlugin && !p.IsFloating && isRightTerminal(p, registered) {
			halves = append(halves, p)
		}
	}
	if len(halves) < 2 {
		return nil
	}
	sort.SliceStable(halves, func(i, j int) bool { return halves[i].Y < halves[j].Y })
	ids := make([]string, len(halves))
	for i, p := range halves {
		ids[i] = p.ID
	}
	return ids
}

// restoreTarget maps the recorded swap layout onto the one that fits the pane
// count after the re-embed. Pair's own Alt+Shift+d split leaves the 3-pane
// rung's name active (and dirty) on a 4-pane tab, a name the 4-pane swap cycle
// never reaches; zellij/layouts/main-3.kdl names each rung's split variant
// "<rung>-split", with the base rung's variant called "small-split".
func restoreTarget(name string, halves int) string {
	if name == "" {
		return ""
	}
	split := strings.HasSuffix(name, "-split")
	switch {
	case halves >= 2 && !split:
		if name == "BASE" {
			return "small-split"
		}
		return name + "-split"
	case halves < 2 && split:
		name = strings.TrimSuffix(name, "-split")
		if name == "small" {
			return "BASE"
		}
	}
	return name
}

// ParseTabLayout reads `current-tab-info --json`; a null name reads as "".
func ParseTabLayout(raw []byte) (string, bool, error) {
	var info struct {
		Name  *string `json:"active_swap_layout_name"`
		Dirty bool    `json:"is_swap_layout_dirty"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return "", false, err
	}
	if info.Name == nil {
		return "", info.Dirty, nil
	}
	return *info.Name, info.Dirty, nil
}

// maxSwapCycle bounds the restore: each pane count's swap cycle in main-3.kdl
// has three layouts, so four steps visit every one from any start.
const maxSwapCycle = 4

// restoreTiling re-tiles after a re-embed, which drops the pane wherever zellij
// splits (measured on 0.45.1: under the focused pane). Re-applying the recorded
// swap layout restores the geometry exactly; next-then-previous does not,
// because floating disturbs zellij's swap index.
func restoreTiling(rt FullscreenRuntime, terminal string, record ExpandRecord) error {
	if target := restoreTarget(record.Swap, len(record.Order)); target != "" {
		for i := 0; ; i++ {
			raw, err := rt.CurrentTabJSON()
			if err != nil {
				return fmt.Errorf("tab info: %w", err)
			}
			name, dirty, err := ParseTabLayout(raw)
			if err != nil {
				return fmt.Errorf("tab info: %w", err)
			}
			if name == target && !dirty {
				break
			}
			if i == maxSwapCycle {
				return fmt.Errorf("swap layout %q not reached (at %q dirty=%v)", target, name, dirty)
			}
			if err := rt.RunZellijAction("next-swap-layout"); err != nil {
				return fmt.Errorf("next-swap-layout: %w", err)
			}
		}
	}
	if len(record.Order) != 2 || (terminal != record.Order[0] && terminal != record.Order[1]) {
		return nil
	}
	raw, err := rt.ListPanesJSON()
	if err != nil {
		return fmt.Errorf("list panes: %w", err)
	}
	y := map[string]int{}
	for _, p := range zellijpane.Parse(raw) {
		if !p.IsPlugin && !p.IsFloating {
			y[p.ID] = p.Y
		}
	}
	top, topOK := y[record.Order[0]]
	bottom, bottomOK := y[record.Order[1]]
	if !topOK || !bottomOK || top < bottom {
		return nil
	}
	direction := "down"
	if terminal == record.Order[0] {
		direction = "up"
	}
	return rt.RunZellijAction("move-pane", "--pane-id", terminal, direction)
}

// Focus geometry (#417, operator): 75% of the width, centered, full height.
var focusPlacement = []string{"-x", "12%", "-y", "0", "--width", "75%", "--height", "100%", "--pinned", "true"}

type FullscreenRuntime interface {
	Runtime
	CurrentPaneID() string
	FullscreenStore() workbenchshortcut.FullscreenStore
	// CurrentTabJSON is `zellij action current-tab-info --json`.
	CurrentTabJSON() ([]byte, error)
	// NudgeWrap sends pair-wrap SIGWINCH so it re-reads the mode and redraws the
	// agent dimmed or plain. Needed for a split, where the agent pane keeps its
	// size and so gets no resize of its own.
	NudgeWrap() error
}

// exitCode reads a zellij CLI exit status; -1 when err carries none.
func exitCode(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return -1
}

// RunToggleFocused is the Alt+Shift+Enter executor. Maximize uses native
// fullscreen rather than toggle-no-ui-fullscreen: UI bars cost rows, while this
// action is for gaining columns.
func RunToggleFocused(args []string, rt FullscreenRuntime, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: pair layout toggle-focused")
		return 2
	}
	store := rt.FullscreenStore()
	fail := func(stage string, err error) int {
		diagnostic := fmt.Errorf("pair layout toggle-focused: %s: %w", stage, err)
		store.LogFailure(diagnostic)
		// Shortcut callers discard this CLI channel; diagnostics are retained above.
		fmt.Fprintln(stderr, diagnostic)
		return 1
	}
	unlock, acquired, err := store.TryLock()
	if err != nil {
		return fail("lock", err)
	}
	if !acquired {
		return 0
	}
	defer unlock()
	raw, err := rt.ListPanesJSON()
	if err != nil {
		return fail("list panes", err)
	}
	if !json.Valid(raw) {
		return fail("list panes", fmt.Errorf("invalid JSON"))
	}
	panes := zellijpane.Parse(raw)
	if len(panes) == 0 {
		return fail("list panes", fmt.Errorf("no pane observations"))
	}
	last, _ := rt.LastTerminalPaneID()
	ids, _ := rt.TerminalPaneIDs()
	recorded, err := store.Read()
	if err != nil {
		return fail("read return pane", err)
	}
	in := RightPaneInput{Panes: panes, Caller: rt.CurrentPaneID(), Last: last, Registered: ids, Record: DecodeExpandRecord(recorded)}
	if mode, _, err := ObserveRightPaneMode(panes, ids); err == nil && mode == ModeNormal {
		// Only Normal→Focus records the swap layout. Without it the later
		// re-embed cannot re-tile, which degrades the restore, not the press.
		if tab, err := rt.CurrentTabJSON(); err != nil {
			store.LogFailure(fmt.Errorf("pair layout toggle-focused: tab info: %w", err))
		} else if name, _, err := ParseTabLayout(tab); err != nil {
			store.LogFailure(fmt.Errorf("pair layout toggle-focused: tab info: %w", err))
		} else {
			in.Swap = name
		}
	}
	plan, err := PlanRightPane(in)
	if err != nil {
		return fail("plan", err)
	}
	for _, step := range plan.Steps {
		var stage string
		switch step.Kind {
		case StepSave:
			stage, err = "save return", store.Write(plan.Record.Encode())
		case StepFloat, StepEmbed:
			stage, err = "toggle-pane-embed-or-floating", rt.RunZellijAction("toggle-pane-embed-or-floating", "--pane-id", step.Pane)
		case StepPlace:
			stage, err = "change-floating-pane-coordinates", rt.RunZellijAction(append([]string{"change-floating-pane-coordinates", "--pane-id", step.Pane}, focusPlacement...)...)
		case StepShow:
			// Exit 2 means "already visible": the state we want.
			if stage, err = "show-floating-panes", rt.RunZellijAction("show-floating-panes"); exitCode(err) == 2 {
				err = nil
			}
		case StepRestore:
			// The pane is embedded and usable either way; a degraded re-tile is
			// logged, and the cycle continues to maximize.
			if err := restoreTiling(rt, step.Pane, plan.Record); err != nil {
				store.LogFailure(fmt.Errorf("pair layout toggle-focused: restore tiling target=%s: %w", step.Pane, err))
			}
			continue
		case StepFullscreen:
			stage, err = "toggle-fullscreen", rt.RunZellijAction("toggle-fullscreen", "--pane-id", step.Pane)
		case StepFocus:
			stage, err = "focus-pane-id", rt.RunZellijAction("focus-pane-id", step.Pane)
		case StepClear:
			stage, err = "clear return", store.Clear()
		case StepNudge:
			// Best effort: a missing or stale wrapper only loses the dim.
			if err := rt.NudgeWrap(); err != nil {
				store.LogFailure(fmt.Errorf("pair layout toggle-focused: nudge wrap: %w", err))
			}
			continue
		}
		if err != nil {
			return fail(fmt.Sprintf("%s target=%s return=%s", stage, step.Pane, plan.Record.Return), err)
		}
	}
	return 0
}

func (OSRuntime) CurrentPaneID() string { return os.Getenv("ZELLIJ_PANE_ID") }
func (OSRuntime) FullscreenStore() workbenchshortcut.FullscreenStore {
	return workbenchshortcut.FullscreenReturnStore{DataDir: workbenchshortcut.DataDirFromEnv(), Tag: os.Getenv("PAIR_TAG")}
}

func (OSRuntime) CurrentTabJSON() ([]byte, error) {
	return execZellij("current-tab-info", "--json")
}

func (OSRuntime) NudgeWrap() error {
	tag := os.Getenv("PAIR_TAG")
	if tag == "" {
		tag = "pair"
	}
	paths, err := artifactpath.ResolveScoped(workbenchshortcut.DataDirFromEnv(), tag)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(paths.PairWrapPID())
	if err != nil {
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return fmt.Errorf("invalid wrapper pid %q", raw)
	}
	return syscall.Kill(pid, syscall.SIGWINCH)
}
