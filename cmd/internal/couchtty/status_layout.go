package couchtty

import (
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// RenderedStatusBar contains the rows and row-relative hit spans drawn together.
type RenderedStatusBar struct {
	Rows  []RenderedStatusRow
	Chips []ChipSpan
}

// Height is the chrome row count reserved for this exact rendering.
func (b RenderedStatusBar) Height() int { return len(b.Rows) }

// Body joins rows for diagnostic and text rendering consumers.
func (b RenderedStatusBar) Body() string {
	lines := make([]string, len(b.Rows))
	for i, r := range b.Rows {
		lines[i] = r.Body
	}
	return strings.Join(lines, "\n")
}

// statusLayoutWork measures the same planner loops used by production, separately
// accounting for untrusted label bytes and actor visits across packing trials.
type statusLayoutWork struct{ Bytes, Visits int }
type statusChip struct {
	label, short                string
	fullWidth, shortWidth, rank int
}
type statusGroup struct {
	start, end int
	oversized  bool
}
type statusPacking struct {
	rows   [2][]chipCell
	height int
	fits   bool
}

// RenderStatusBar wraps complete groups across at most two rows, retaining
// active and attention chips before ordinary chips when both rows overflow.
func RenderStatusBar(width, maxRows int, m StatusModel) RenderedStatusBar {
	return renderStatusBar(width, maxRows, m, nil)
}
func renderStatusBar(width, maxRows int, m StatusModel, work *statusLayoutWork) RenderedStatusBar {
	if width <= 0 || maxRows <= 0 {
		return RenderedStatusBar{}
	}
	if maxRows > 2 {
		maxRows = 2
	}
	// Run the shared prefix pass without actors or a transient notice. Its budget
	// already reserves the first-row attention badge, without duplicating policy.
	prefixModel := m
	prefixModel.Notice = ""
	_, prefix, budget := renderStatusCells(width, prefixModel, []chipCell{}, true, -1)
	chips, groups := prepareStatusChips(m.Actors, width, work)
	p := packStatusChips(m.Actors, chips, groups, len(chips), width, budget, prefix, maxRows, 0, false, true, work)
	if !p.fits {
		// An active chip wins scarce one-row space, including over fixed controls.
		// This is needed by short panes which cannot afford another chrome row.
		if maxRows == 1 {
			for i, a := range m.Actors {
				if work != nil {
					work.Visits++
				}
				if a.Active {
					activeWidth := chips[i].fullWidth
					if activeWidth > width {
						activeWidth = width
					}
					if budget < activeWidth {
						prefixModel.Attention = 0
					}
					limit := width - activeWidth - 2
					if limit < 0 {
						limit = 0
					}
					_, prefix, budget = renderStatusCells(width, prefixModel, []chipCell{}, true, limit)
					break
				}
			}
		}
		suffix := "+" + strconv.Itoa(len(chips))
		for _, a := range m.Actors {
			if work != nil {
				work.Visits++
			}
			if chipKept(a) && !a.Active {
				suffix += " !"
				break
			}
		}
		reserve := textwidth.Width(suffix) + 2
		// Full labels in overflow candidates give a conservative monotone cost:
		// removing a group leader never makes another retained label grow.
		lo, hi := 0, len(chips)
		for lo < hi {
			mid := (lo + hi + 1) / 2
			trial := packStatusChips(m.Actors, chips, groups, mid, width, budget, prefix, maxRows, reserve, true, false, work)
			if trial.fits {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		p = packStatusChips(m.Actors, chips, groups, lo, width, budget, prefix, maxRows, reserve, true, true, work)
		hidden, hiddenAttention := len(chips)-lo, false
		for i, a := range m.Actors {
			if work != nil {
				work.Visits++
			}
			if chips[i].rank >= lo && chipKept(a) && !a.Active {
				hiddenAttention = true
			}
		}
		// If even a single active chip does not fit with the whole count, show it
		// clipped and omit the count. The drawing pass still owns its actual span.
		if lo == 0 {
			for i, a := range m.Actors {
				if work != nil {
					work.Visits++
				}
				if a.Active {
					row := maxRows - 1
					p = statusPacking{height: maxRows, fits: true}
					p.rows[row] = []chipCell{{actor: i, label: chips[i].label}}
					if maxRows == 1 {
						prefixModel.Attention = 0
						prefix = 0
					}
					hidden--
					break
				}
			}
		}
		if hidden > 0 {
			label := "+" + strconv.Itoa(hidden)
			if hiddenAttention {
				label += " !"
			}
			last := maxRows - 1
			p.height = maxRows
			p.rows[last] = append(p.rows[last], chipCell{actor: -1, label: label})
		}
	}
	if p.height == 0 {
		p.height = 1
	}
	bar := RenderedStatusBar{Rows: make([]RenderedStatusRow, p.height)}
	for i := range bar.Rows {
		model := prefixModel
		limit := -1
		if i > 0 {
			model = StatusModel{Actors: m.Actors, Spinner: m.Spinner, Palette: m.Palette}
		}
		if i == 0 && maxRows == 1 {
			limit = prefix
		}
		if i == len(bar.Rows)-1 {
			model.Notice = m.Notice
		}
		row, _, _ := renderStatusCells(width, model, p.rows[i], true, limit)
		for j := range row.Chips {
			row.Chips[j].Row = i
		}
		bar.Rows[i] = row
		bar.Chips = append(bar.Chips, row.Chips...)
	}
	return bar
}

func prepareStatusChips(actors []StatusActor, width int, work *statusLayoutWork) ([]statusChip, []statusGroup) {
	chips := make([]statusChip, len(actors))
	for i, a := range actors {
		if work != nil {
			work.Bytes += len(a.Label) + len(a.Glyph)
			work.Visits++
		}
		label := rowtext.Sanitize(a.Label)
		short := label
		if a.SlotNumber > 0 {
			short = ":" + strconv.Itoa(a.SlotNumber)
		}
		extra := textwidth.Width(a.Glyph) + textwidth.Width(chipTail(a, 0))
		if a.Active {
			extra++
		}
		chips[i] = statusChip{label: label, short: short, fullWidth: textwidth.Width(label) + extra, shortWidth: textwidth.Width(short) + extra}
	}
	rank := 0
	// Stable priority buckets require no sorting and retain input order in each.
	for priority := 0; priority < 3; priority++ {
		for i, a := range actors {
			if work != nil {
				work.Visits++
			}
			p := 2
			if chipKept(a) {
				p = 1
			}
			if a.Active {
				p = 0
			}
			if p == priority {
				chips[i].rank = rank
				rank++
			}
		}
	}
	if work != nil {
		work.Visits += len(actors)
	}
	spans := chipGroups(actors)
	groups := make([]statusGroup, len(spans))
	for g, span := range spans {
		total := 0
		for i := span[0]; i < span[1]; i++ {
			if work != nil {
				work.Visits++
			}
			w := chips[i].fullWidth
			if i > span[0] {
				w = chips[i].shortWidth
				total += 2
			}
			total += w
		}
		groups[g] = statusGroup{span[0], span[1], total > width}
	}
	return chips, groups
}

// packStatusChips makes allocation-free candidate passes; only the winning
// pass materializes cells. Original oversized groups alone may cross a row.
func packStatusChips(actors []StatusActor, chips []statusChip, groups []statusGroup, keep, width, budget, prefix, maxRows, reserve int, full, materialize bool, work *statusLayoutWork) statusPacking {
	p := statusPacking{height: 1, fits: true}
	row, used := 0, prefix
	capacity := func() int {
		v := width
		if row == 0 {
			v = budget
		}
		if row == maxRows-1 {
			v -= reserve
		}
		return v
	}
	nextRow := func() bool {
		if row+1 >= maxRows {
			return false
		}
		row++
		used = 0
		p.height = row + 1
		return true
	}
	for _, g := range groups {
		groupWidth, first := -1, -1
		for i := g.start; i < g.end; i++ {
			if work != nil {
				work.Visits++
			}
			if chips[i].rank >= keep {
				continue
			}
			if first < 0 {
				first = i
				groupWidth = chips[i].fullWidth
			} else {
				w := chips[i].shortWidth
				if full {
					w = chips[i].fullWidth
				}
				groupWidth += 2 + w
			}
		}
		if first < 0 {
			continue
		}
		gap := 0
		if used > 0 {
			gap = 2
			if actors[first].Hub && first > 0 {
				gap = textwidth.Width(sectionGap)
			}
		}
		if !g.oversized && used+gap+groupWidth > capacity() {
			if !nextRow() {
				p.fits = false
				return p
			}
		}
		previous := false
		for i := g.start; i < g.end; i++ {
			if work != nil {
				work.Visits++
			}
			if chips[i].rank >= keep {
				continue
			}
			label, w := chips[i].label, chips[i].fullWidth
			if previous && !full {
				label, w = chips[i].short, chips[i].shortWidth
			}
			gap = 0
			if used > 0 {
				gap = 2
				if actors[i].Hub && i > 0 {
					gap = textwidth.Width(sectionGap)
				}
			}
			if used+gap+w > capacity() {
				if !g.oversized || !nextRow() {
					p.fits = false
					return p
				}
				label, w = chips[i].label, chips[i].fullWidth
				gap = 0
				previous = false
				if w > capacity() {
					p.fits = false
					return p
				}
			}
			if materialize {
				p.rows[row] = append(p.rows[row], chipCell{actor: i, label: label, section: actors[i].Hub && i > 0 && used > 0})
			}
			used += gap + w
			previous = true
		}
	}
	return p
}
