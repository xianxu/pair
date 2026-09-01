package couchtty

import (
	"fmt"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
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
	thread, ok := findMenuThread(state.Inventory, frame.Thread)
	if !ok {
		return "threads"
	}
	parts := []string{"threads", thread.Label()}
	switch frame.Kind {
	case MenuFrameActions:
		parts = append(parts, "actions")
	case MenuFrameConfirmation:
		leaf := frame.Action
		if leaf == "leave" {
			leaf = "leave couch"
		}
		parts = append(parts, leaf)
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
		thread, _ := findMenuThread(state.Inventory, frame.Thread)
		title := "park " + thread.Label() + "?"
		if frame.Action == "leave" {
			title = "leave couch?"
		}
		return renderItemMenuFrame(title, filterMenuItems(confirmationMenuItems(frame.Action, thread), frame.Filter), confirmationDisplaySelection(frame), frame.Filter, width, height)
	case MenuFrameText:
		return []string{clipMenuLine(menuItemLabel(frame.Action), width), "", clipMenuLine("> "+frame.Input, width)}
	case MenuFrameStart:
		pathMarker, agentMarker := "▸ ", "  "
		if frame.FormField == MenuFieldAgent {
			pathMarker, agentMarker = "  ", "▸ "
		}
		lines := []string{
			clipMenuLine("start thread", width), "",
			selectedMenuLine(pathMarker+"path  "+frame.Path, frame.FormField == MenuFieldPath, width),
			selectedMenuLine(agentMarker+"agent "+frame.Agent+menuSourceSuffix(string(frame.PreviewAgentSource)), frame.FormField == MenuFieldAgent, width),
		}
		if frame.PreviewArgvSource != "" {
			lines = append(lines, clipMenuLine("  args  "+string(frame.PreviewArgvSource), width))
		}
		return lines
	default:
		return []string{"menu unavailable"}
	}
}

func renderRootMenuFrame(state MenuState, frame MenuFrame, width, height int, now time.Time, color256 bool) []string {
	visible := visibleRootThreads(state.Inventory, frame)
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
	selected := 0
	for i, thread := range visible {
		if thread.Address == frame.SelectedAddress {
			selected = i
			break
		}
	}
	start := selected - rowBudget + 1
	if start < 0 {
		start = 0
	}
	end := start + rowBudget
	if end > len(visible) {
		end = len(visible)
	}
	if len(visible) == 0 {
		lines = append(lines, "  (no match)")
	}
	for _, thread := range visible[start:end] {
		selectedRow := thread.Address == frame.SelectedAddress
		marker := "  "
		if selectedRow {
			marker = "▸ "
		}
		stateText := "live"
		if !thread.Live() {
			stateText = "parked · " + relativeMenuAge(now, thread.LastActiveAt)
		}
		suffix := "  " + stateText
		if len(state.Attention[thread.Address]) > 0 {
			suffix += " *"
		}
		prefixWidth := width - textwidth.Width(suffix)
		if prefixWidth < 0 {
			prefixWidth = 0
		}
		plain := clipMenuLine(fmt.Sprintf("%s%s  %s", marker, thread.Label(), thread.WorkingPath), prefixWidth) + suffix
		if selectedRow {
			plain = selectedMenuLine(plain, true, width)
		} else if !thread.Live() && color256 {
			plain = ageColor(AgeBandFor(now, thread.LastActiveAt)) + plain + "\x1b[0m"
		}
		lines = append(lines, plain)
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
