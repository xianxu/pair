package couchtty

import (
	"fmt"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

const (
	menuMinWidth  = 40
	menuMinHeight = 10
)

type MenuLayoutKind uint8

const (
	MenuLayoutUnknown MenuLayoutKind = iota
	MenuLayoutResize
	MenuLayoutSingle
)

type MenuRect struct {
	X, Y          int
	Width, Height int
}

type MenuLayout struct {
	Kind   MenuLayoutKind
	Frames []MenuRect
}

type MenuCursorIntent struct {
	Row int
	Col int
}

type RenderedMenu struct {
	Body   string
	Cursor *MenuCursorIntent
}

// ChooseMenuLayout is pure geometry. Every emitted rectangle is contained by
// the caller's terminal dimensions; undersized terminals get no partial UI.
func ChooseMenuLayout(width, height int) MenuLayout {
	if width < menuMinWidth || height < menuMinHeight {
		return MenuLayout{Kind: MenuLayoutResize}
	}
	return MenuLayout{Kind: MenuLayoutSingle, Frames: []MenuRect{{Width: width, Height: height}}}
}

type AgeBand uint8

const (
	AgeUnknown AgeBand = iota
	AgeRecent
	AgeDays
	AgeOld
)

func AgeBandFor(now, lastActive time.Time) AgeBand {
	age := now.Sub(lastActive)
	if age < 24*time.Hour {
		return AgeRecent
	}
	if age < 7*24*time.Hour {
		return AgeDays
	}
	return AgeOld
}

// RenderMenu turns pure state into bounded ANSI text. It never reads the
// clock, terminal, store, or process state; all environment input is explicit.
func RenderMenu(state MenuState, width, height int, now time.Time, color256 bool) string {
	return RenderMenuView(state, width, height, now, color256).Body
}

// RenderMenuView is the complete pure presentation decision. Cursor coordinates
// are 1-based terminal cells in Body after clipping and banner insertion.
func RenderMenuView(state MenuState, width, height int, now time.Time, color256 bool) RenderedMenu {
	layout := ChooseMenuLayout(width, height)
	if layout.Kind == MenuLayoutResize {
		return RenderedMenu{Body: "resize terminal to at least 40x10"}
	}
	frames := state.Frames
	if len(frames) == 0 {
		frames = []MenuFrame{{Kind: MenuFrameRoot}}
	}
	frame := frames[len(frames)-1]
	lines := renderMenuFrame(state, frame, width, height, now, color256)
	if len(lines) > 0 {
		breadcrumb := clipMenuLine(menuBreadcrumb(state, frame), width)
		if color256 {
			breadcrumb = "\x1b[38;5;245m" + breadcrumb + "\x1b[0m"
		}
		lines[0] = breadcrumb
	}
	if notice := renderedMenuNotice(state); notice != "" {
		lines = append(lines, "")
		copy(lines[2:], lines[1:])
		lines[1] = clipMenuLine(notice, width)
	}
	lines = fitMenuBlock(lines, width, height)
	if len(lines) > height {
		lines = lines[:height]
	}
	return RenderedMenu{Body: strings.Join(lines, "\r\n"), Cursor: menuCursorIntent(frame, lines, width)}
}

func renderedMenuNotice(state MenuState) string {
	notice := state.Notice
	if notice.Text == "" {
		if state.ProjectionPending {
			return "refresh pending"
		}
		return ""
	}
	if notice.Level == MenuNoticeError {
		if state.ProjectionPending {
			return "error: " + notice.Text + "; refresh pending"
		}
		return "error: " + notice.Text
	}
	if notice.Level == MenuNoticeProgress {
		frames := [...]string{"◐", "◓", "◑", "◒"}
		return frames[int(state.SpinnerPhase)%len(frames)] + " " + notice.Text + "…"
	}
	return notice.Text
}

func menuCursorIntent(frame MenuFrame, lines []string, width int) *MenuCursorIntent {
	prefix := ""
	switch frame.Kind {
	case MenuFrameRoot, MenuFrameActions, MenuFrameConfirmation:
		if frame.Filter == "" {
			return nil
		}
		prefix = "filter: "
	case MenuFrameText:
		prefix = "> "
	case MenuFrameStart:
		if frame.FormField != MenuFieldPath {
			return nil
		}
		prefix = "▸ path  "
	default:
		return nil
	}
	for index, styled := range lines {
		plain := string(ansi.Strip([]byte(styled)))
		if !strings.HasPrefix(plain, prefix) {
			continue
		}
		col := textwidth.Width(plain) + 1
		if col > width {
			col = width
		}
		return &MenuCursorIntent{Row: index + 1, Col: col}
	}
	return nil
}

func menuBreadcrumb(state MenuState, frame MenuFrame) string {
	if frame.Kind == MenuFrameRoot {
		return "threads"
	}
	if frame.Kind == MenuFrameStart {
		return "start thread"
	}
	if frame.Kind == MenuFrameConfirmation && frame.Action == "leave" {
		// A global frame: it names couch, not a thread.
		return "threads › leave couch"
	}
	thread, ok := findMenuThread(state.Inventory, frame.Thread)
	if !ok {
		return "threads"
	}
	parts := []string{"threads", thread.Label()}
	switch frame.Kind {
	case MenuFrameActions:
		parts = append(parts, "actions")
	case MenuFrameConfirmation:
		// `leave` never reaches here -- it returns early above as a global frame.
		parts = append(parts, frame.Action)
	case MenuFrameText:
		parts = append(parts, menuItemLabel(frame.Action))
	}
	return strings.Join(parts, " › ")
}

func renderMenuFrame(state MenuState, frame MenuFrame, width, height int, now time.Time, color256 bool) []string {
	switch frame.Kind {
	case MenuFrameRoot:
		return renderRootMenuFrame(state, frame, width, height, now, color256)
	case MenuFrameActions:
		thread, _ := findMenuThread(state.Inventory, frame.Thread)
		return renderItemMenuFrame("actions · "+thread.Label(), filterMenuItems(menuActionItems(thread), frame.Filter), frame.SelectedItem, frame.Filter, width, height)
	case MenuFrameConfirmation:
		// The title argument is vestigial at every call site: RenderMenuView
		// overwrites line 0 with the breadcrumb. What the operator reads is the
		// ITEM, which is why the item names the action's cost.
		thread, _ := findMenuThread(state.Inventory, frame.Thread)
		title := "park " + thread.Label() + "?"
		if frame.Action == "archive" {
			title = "archive " + thread.Label() + "?"
		}
		return renderItemMenuFrame(title, filterMenuItems(confirmationMenuItems(state, frame), frame.Filter), confirmationDisplaySelection(frame), frame.Filter, width, height)
	case MenuFrameText:
		return []string{clipMenuLine(menuItemLabel(frame.Action), width), "", clipMenuLine("> "+frame.Input, width)}
	case MenuFrameStart:
		return renderStartMenuFrame(state, frame, width, height)
	default:
		return []string{"menu unavailable"}
	}
}

func renderStartMenuFrame(state MenuState, frame MenuFrame, width, height int) []string {
	pathMarker, agentMarker := "▸ ", "  "
	if frame.FormField == MenuFieldAgent {
		pathMarker, agentMarker = "  ", "▸ "
	}
	lines := []string{
		clipMenuLine("start thread", width), "",
		selectedMenuLine(pathMarker+"path  "+frame.Path, frame.FormField == MenuFieldPath, width),
	}
	fixedRows := 4
	if frame.PreviewResolution.ArgvSource != "" {
		fixedRows++
	}
	if frame.CompletionTruncated {
		fixedRows++
	}
	if renderedMenuNotice(state) != "" {
		fixedRows++
	}
	budget := max(height-fixedRows, 0)
	if budget > len(frame.CompletionCandidates) {
		budget = len(frame.CompletionCandidates)
	}
	selected := frame.CompletionSelected
	if selected < 0 || selected >= len(frame.CompletionCandidates) {
		selected = 0
	}
	start := selected - budget + 1
	if start < 0 {
		start = 0
	}
	if end := min(start+budget, len(frame.CompletionCandidates)); start < end {
		for index := start; index < end; index++ {
			isSelected := index == selected
			marker := "  "
			if isSelected {
				marker = "▸ "
			}
			lines = append(lines, selectedMenuLine(marker+frame.CompletionCandidates[index], isSelected, width))
		}
	}
	if frame.CompletionTruncated {
		lines = append(lines, clipMenuLine("  … more matching directories", width))
	}
	lines = append(lines, selectedMenuLine(agentMarker+"agent "+frame.Agent+menuSourceSuffix(string(frame.PreviewResolution.AgentSource)), frame.FormField == MenuFieldAgent, width))
	if frame.PreviewResolution.ArgvSource != "" {
		lines = append(lines, clipMenuLine("  args  "+string(frame.PreviewResolution.ArgvSource), width))
	}
	return lines
}

// rootStateText is the right-hand column of one switcher row.
//
// It used to be two cases -- live, or "parked" for everything else -- which
// made the operator's one detached thread claim to be parked, and gave the nine
// unusable ones no way to appear at all. Every state and every reason has a
// label, and the guard that keeps it that way iterates the vocabulary rather
// than listing cases here (Go has no exhaustive-switch check).
func rootStateText(thread couchcore.ActionableThreadSummary, now time.Time) string {
	switch thread.State {
	case couchcore.ThreadLive:
		return "live"
	case couchcore.ThreadDetached:
		return "detached · " + relativeMenuAge(now, thread.LastActiveAt)
	case couchcore.ThreadParked:
		return "parked · " + relativeMenuAge(now, thread.LastActiveAt)
	case couchcore.ThreadBusy:
		return "parking…"
	case couchcore.ThreadArchived:
		return "archived"
	}
	return thread.Reason.Label()
}

func renderRootMenuFrame(state MenuState, frame MenuFrame, width, height int, now time.Time, color256 bool) []string {
	visible := visibleRootThreads(state.Inventory, frame)
	// Labels are disambiguated against the WHOLE inventory, not the filtered
	// view: a name that is unique only because the filter hid its twin would
	// change as the operator types.
	labelRows := make([]couchcore.LabelRow, 0, len(state.Inventory))
	for _, thread := range state.Inventory {
		labelRows = append(labelRows, couchcore.LabelRow{Address: thread.Address, Label: thread.Label()})
	}
	labels := couchcore.DisambiguateLabels(labelRows)
	lines := []string{"threads", ""}
	rowBudget := height - len(lines)
	if frame.Filter != "" {
		rowBudget--
	}
	if state.Notice.Text != "" || state.ProjectionPending {
		rowBudget--
	}
	if rowBudget < 1 {
		rowBudget = 1
	}
	if len(visible) == 0 {
		lines = append(lines, "  (no match)")
	}
	type rootLine struct {
		text       string
		selected   bool
		actorStart bool
	}
	var rows []rootLine
	selectedStart, selectedEnd := 0, 0
	for _, thread := range visible {
		selectedRow := thread.Address == frame.SelectedAddress
		marker := "  "
		if selectedRow {
			marker = "▸ "
		}
		suffix := "  " + rootStateText(thread, now)
		prefixWidth := width - textwidth.Width(suffix)
		if prefixWidth < 0 {
			prefixWidth = 0
		}
		plain := clipMenuLine(fmt.Sprintf("%s%s  %s", marker, labels[thread.Address], thread.WorkingPath), prefixWidth) + suffix
		if selectedRow {
			plain = selectedMenuLine(plain, true, width)
		} else if !thread.Live() && color256 {
			plain = ageColor(AgeBandFor(now, thread.LastActiveAt)) + plain + "\x1b[0m"
		}
		if selectedRow {
			selectedStart = len(rows)
		}
		rows = append(rows, rootLine{text: plain, selected: selectedRow, actorStart: true})
		for _, message := range state.Attention[thread.Address] {
			if message.Text != "" {
				rows = append(rows, rootLine{text: clipMenuLine("    "+message.Text, width)})
			}
		}
		if selectedRow {
			selectedEnd = len(rows)
		}
	}
	start := selectedEnd - rowBudget
	if start < 0 {
		start = 0
	}
	for start < selectedStart && start < len(rows) && !rows[start].actorStart {
		start++
	}
	end := min(start+rowBudget, len(rows))
	for _, row := range rows[start:end] {
		if row.selected {
			row.text = selectedMenuLine(string(ansi.Strip([]byte(row.text))), true, width)
		}
		lines = append(lines, row.text)
	}
	if frame.Filter != "" {
		lines = append(lines, clipMenuLine("filter: "+frame.Filter, width))
	}
	return lines
}

func renderItemMenuFrame(title string, items []string, selected, filter string, width, height int) []string {
	lines := []string{clipMenuLine(title, width), ""}
	budget := height - len(lines)
	if filter != "" {
		budget--
	}
	if budget < 0 {
		budget = 0
	}
	selectedIndex := 0
	for i, item := range items {
		if item == selected || strings.HasPrefix(item, selected+" ") {
			selectedIndex = i
			break
		}
	}
	start := selectedIndex - budget + 1
	if start < 0 {
		start = 0
	}
	end := start + budget
	if end > len(items) {
		end = len(items)
	}
	for _, item := range items[start:end] {
		isSelected := menuItemID(item) == selected
		marker := "  "
		if isSelected {
			marker = "▸ "
		}
		lines = append(lines, selectedMenuLine(marker+menuItemLabel(item), isSelected, width))
	}
	if filter != "" {
		lines = append(lines, clipMenuLine("filter: "+filter, width))
	}
	return lines
}

func menuSourceSuffix(source string) string {
	if source == "" {
		return ""
	}
	return " (" + source + ")"
}

func confirmationDisplaySelection(frame MenuFrame) string {
	if frame.SelectedItem == frame.Action {
		return frame.Action
	}
	return "cancel"
}

func selectedMenuLine(line string, selected bool, width int) string {
	plain := clipMenuLine(line, width)
	if selected {
		return "\x1b[7m" + plain + "\x1b[0m"
	}
	return plain
}

func relativeMenuAge(now, lastActive time.Time) string {
	age := now.Sub(lastActive)
	if age < 0 {
		age = 0
	}
	if age < time.Hour {
		return "<1h ago"
	}
	if age < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(age/time.Hour))
	}
	return fmt.Sprintf("%dd ago", int(age/(24*time.Hour)))
}

func ageColor(band AgeBand) string {
	switch band {
	case AgeRecent:
		return "\x1b[38;5;250m"
	case AgeDays:
		return "\x1b[38;5;245m"
	default:
		return "\x1b[38;5;240m"
	}
}

func fitMenuBlock(block []string, width, height int) []string {
	if len(block) > height {
		block = block[:height]
	}
	out := make([]string, len(block))
	for i, line := range block {
		out[i] = clipStyledMenuLine(line, width)
	}
	return out
}

func clipMenuLine(line string, width int) string {
	return truncate(sanitize(line), width)
}

func clipStyledMenuLine(line string, width int) string {
	if styledMenuWidth(line) <= width {
		return line
	}
	return clipMenuLine(string(ansi.Strip([]byte(line))), width)
}

func styledMenuWidth(line string) int {
	return textwidth.Width(string(ansi.Strip([]byte(line))))
}
