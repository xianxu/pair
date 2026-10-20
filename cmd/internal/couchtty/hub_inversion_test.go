package couchtty

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func hubCells(t *testing.T, body string, width int) []terminal.Cell {
	t.Helper()
	rows, err := terminal.StyledRows(body, width, 1)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// Compare against the normal style at the terminal-cell boundary: hub changes
// only reverse video, even through glyph resets, clipping and placeholders.
func TestHubTabInversionComposesEveryState(t *testing.T) {
	for bits := 0; bits < 16; bits++ {
		for _, idle := range []IdleLevel{IdleFresh, IdleStale} {
			for width := 1; width <= 24; width++ {
				t.Run(fmt.Sprintf("state%d/idle%d/width%d", bits, idle, width), func(t *testing.T) {
					a := idleChip("hub", idle)
					a.Blocked, a.Bell, a.Active, a.Placeholder = bits&1 != 0, bits&2 != 0, bits&4 != 0, bits&8 != 0
					a.Loading = a.Placeholder
					model := StatusModel{Palette: darkTrueColor, Actors: []StatusActor{a, {Label: "peer", Thread: couchcore.ThreadAddress{Tag: "peer"}}}}
					normal := RenderStatusRow(width, model)
					model.Actors[0].Hub = true
					hub := RenderStatusRow(width, model)
					if !reflect.DeepEqual(normal.Chips, hub.Chips) {
						t.Fatal("inversion changed click spans")
					}
					before, after := hubCells(t, normal.Body, width), hubCells(t, hub.Body, width)
					length := len("hub*")
					if a.Active {
						length += 2
					}
					if a.Loading {
						length += 2
					}
					for col := range before {
						want := before[col]
						if col < length {
							want.Style.Attrs |= uv.AttrReverse
						}
						if !reflect.DeepEqual(after[col], want) {
							t.Fatalf("column %d = %+v, want %+v; %q", col, after[col], want, hub.Body)
						}
					}
				})
			}
		}
	}
}

func TestHubTabInversionLeavesFoldCountsNormal(t *testing.T) {
	model := StatusModel{Actors: []StatusActor{{Label: "hub", Hub: true, Thread: couchcore.ThreadAddress{Tag: "hub"}}}}
	for i := 1; i <= 8; i++ {
		model.Actors = append(model.Actors, StatusActor{Label: fmt.Sprintf("peer:%d", i), GroupKey: "peer", SlotNumber: i})
	}
	for _, width := range []int{16, 20, 30} {
		got := RenderStatusRow(width, model)
		cells := hubCells(t, got.Body, width)
		if !strings.Contains(got.Body, "+") {
			t.Fatalf("fixture did not fold: %q", got.Body)
		}
		for col, c := range cells {
			if (c.Style.Attrs&uv.AttrReverse != 0) != (col < 3) {
				t.Fatalf("width %d column %d: inversion leaked or missing", width, col)
			}
		}
	}
}

func TestHubSwitcherUnderlineComposesWithSelection(t *testing.T) {
	for _, hub := range []bool{false, true} {
		for _, selected := range []bool{false, true} {
			for _, blocked := range []bool{false, true} {
				for _, color := range []bool{false, true} {
					state, one := idleMenuState(t)
					if hub {
						state.Hub = one.StartingPath
					}
					state.Blocked = map[couchcore.ThreadAddress]bool{one.Address: blocked}
					state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{one.Address: {{Text: "hello"}}}
					if selected {
						state.Frames[0].SelectedAddress = one.Address
						state.Frames[0].SelectedKey = one.RowKey
					}
					lines, _ := renderRootMenuFrame(state, state.CurrentFrame(), 100, 16, idleNow, color)
					menu := strings.Join(lines, "\n")
					line := menuLineContaining(t, menu, "pair:1±*")
					cells := hubCells(t, line+"X", 120)
					for _, c := range cells {
						if c.Content == "X" {
							if c.Style.Attrs != 0 || c.Style.Underline != uv.UnderlineNone {
								t.Fatal("hub style leaked beyond row")
							}
							break
						}
						if c.Content == "" {
							continue
						}
						if got := c.Style.Attrs&uv.AttrReverse != 0; got != selected {
							t.Fatalf("reverse = %v, want selected=%v: %q", got, selected, line)
						}
						if got := c.Style.Underline != uv.UnderlineNone; got != hub {
							t.Fatalf("underline = %v, want hub=%v: %q", got, hub, line)
						}
						if selected && hub && c.Style.Attrs&uv.AttrBold == 0 {
							t.Fatalf("selected hub bold missing: %q", line)
						}
					}
					if selected && !strings.Contains(line, "▸") {
						t.Fatal("selected hub lost arrow")
					}
					if hub && blocked && color && !strings.Contains(line, "\x1b[31m") {
						t.Fatal("hub lost blocked red")
					}
					detail := menuLineContaining(t, menu, "hello")
					if !selected && strings.Contains(detail, "\x1b[7m") {
						t.Fatal("hub inversion leaked into attention")
					}
				}
			}
		}
	}
}
