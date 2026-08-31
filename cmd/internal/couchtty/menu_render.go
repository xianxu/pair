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
	MenuLayoutWide
	MenuLayoutNarrow
)

type MenuRect struct {
	X, Y          int
	Width, Height int
}

type MenuLayout struct {
	Kind   MenuLayoutKind
	Frames []MenuRect
}

// ChooseMenuLayout is pure geometry. Every emitted rectangle is contained by
// the caller's terminal dimensions; undersized terminals get no partial UI.
func ChooseMenuLayout(state MenuState, width, height int) MenuLayout {
	if width < menuMinWidth || height < menuMinHeight {
		return MenuLayout{Kind: MenuLayoutResize}
	}
	count := len(state.Frames)
	if count < 1 {
		count = 1
	}
	if count == 1 {
		return MenuLayout{Kind: MenuLayoutSingle, Frames: []MenuRect{{Width: width, Height: height}}}
	}
	if width >= menuMinWidth*count {
		layout := MenuLayout{Kind: MenuLayoutWide, Frames: make([]MenuRect, count)}
		contentWidth := width - (count - 1)
		x := 0
		for i := range layout.Frames {
			remaining := contentWidth
			for j := 0; j < i; j++ {
				remaining -= layout.Frames[j].Width
			}
			frameWidth := remaining / (count - i)
			layout.Frames[i] = MenuRect{X: x, Width: frameWidth, Height: height}
			x += frameWidth + 1
		}
		return layout
	}
	layout := MenuLayout{Kind: MenuLayoutNarrow, Frames: make([]MenuRect, count)}
	y := 0
	for i := range layout.Frames {
		remaining := height - y
		frameHeight := remaining / (count - i)
		layout.Frames[i] = MenuRect{Y: y, Width: width, Height: frameHeight}
		y += frameHeight
	}
	return layout
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
	layout := ChooseMenuLayout(state, width, height)
	if layout.Kind == MenuLayoutResize {
		return "resize terminal to at least 40x10"
	}
	frames := state.Frames
	if len(frames) == 0 {
		frames = []MenuFrame{{Kind: MenuFrameRoot}}
	}
	blocks := make([][]string, len(layout.Frames))
	for i, rect := range layout.Frames {
		frame := frames[i]
		blocks[i] = renderMenuFrame(state, frame, rect.Width, rect.Height, now, color256)
	}
	var lines []string
	switch layout.Kind {
	case MenuLayoutWide:
		lines = combineMenuColumns(blocks, layout.Frames)
	case MenuLayoutNarrow:
		for i, block := range blocks {
			lines = append(lines, fitMenuBlock(block, layout.Frames[i].Width, layout.Frames[i].Height)...)
		}
	default:
		lines = fitMenuBlock(blocks[0], width, height)
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\r\n")
}

func renderMenuFrame(state MenuState, frame MenuFrame, width, height int, now time.Time, color256 bool) []string {
	switch frame.Kind {
	case MenuFrameRoot:
		return renderRootMenuFrame(state, frame, width, height, now, color256)
	case MenuFrameActions:
		thread, _ := findMenuThread(state.Inventory, frame.Thread)
		return renderItemMenuFrame("actions · "+thread.Label(), filterMenuItems(menuActionItems(thread), frame.Filter), frame.SelectedItem, width, height)
	case MenuFrameConfirmation:
		thread, _ := findMenuThread(state.Inventory, frame.Thread)
		return renderItemMenuFrame("park "+thread.Label()+"?", []string{"cancel", "park " + thread.Label()}, confirmationDisplaySelection(frame, thread), width, height)
	case MenuFrameText:
		return []string{clipMenuLine(frame.Action, width), "", clipMenuLine("> "+frame.Input, width)}
	case MenuFrameStart:
		pathMarker, agentMarker := "▸ ", "  "
		if frame.FormField == MenuFieldAgent {
			pathMarker, agentMarker = "  ", "▸ "
		}
		return []string{
			clipMenuLine("start thread", width), "",
			selectedMenuLine(pathMarker+"path  "+frame.Path, frame.FormField == MenuFieldPath, width),
			selectedMenuLine(agentMarker+"agent "+frame.Agent, frame.FormField == MenuFieldAgent, width),
		}
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
	if state.Notice != "" {
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
		plain := clipMenuLine(fmt.Sprintf("%s%s  %s  %s", marker, thread.Label(), thread.WorkingPath, stateText), width)
		if selectedRow {
			plain = selectedMenuLine(plain, true, width)
		} else if !thread.Live() && color256 {
			plain = ageColor(AgeBandFor(now, thread.LastActiveAt)) + plain + "\x1b[0m"
		}
		if state.Bells[thread.Address] {
			plain = clipStyledMenuLine(plain+" *", width)
		}
		lines = append(lines, plain)
	}
	if frame.Filter != "" {
		lines = append(lines, clipMenuLine("filter: "+frame.Filter, width))
	}
	if state.Notice != "" {
		lines = append(lines, clipMenuLine(state.Notice, width))
	}
	return lines
}

func renderItemMenuFrame(title string, items []string, selected string, width, height int) []string {
	lines := []string{clipMenuLine(title, width), ""}
	budget := height - len(lines)
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
		isSelected := item == selected || strings.HasPrefix(item, selected+" ")
		marker := "  "
		if isSelected {
			marker = "▸ "
		}
		lines = append(lines, selectedMenuLine(marker+item, isSelected, width))
	}
	return lines
}

func confirmationDisplaySelection(frame MenuFrame, thread couchcore.ActionableThreadSummary) string {
	if frame.SelectedItem == "park" {
		return "park"
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

func combineMenuColumns(blocks [][]string, rects []MenuRect) []string {
	height := 0
	for i, block := range blocks {
		if n := len(fitMenuBlock(block, rects[i].Width, rects[i].Height)); n > height {
			height = n
		}
	}
	lines := make([]string, height)
	for row := 0; row < height; row++ {
		var b strings.Builder
		for col, block := range blocks {
			if col > 0 {
				b.WriteString("│")
			}
			line := ""
			if row < len(block) && row < rects[col].Height {
				line = clipStyledMenuLine(block[row], rects[col].Width)
			}
			b.WriteString(line)
			if col < len(blocks)-1 {
				padding := rects[col].Width - styledMenuWidth(line)
				if padding > 0 {
					b.WriteString(strings.Repeat(" ", padding))
				}
			}
		}
		lines[row] = b.String()
	}
	return lines
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
