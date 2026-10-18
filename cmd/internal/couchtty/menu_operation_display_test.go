package couchtty

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestMenuOperationLongDetailKeepsRowIdentity(t *testing.T) {
	for _, view := range []MenuRootView{MenuViewNormal, MenuViewFocus} {
		for _, width := range []int{40, 60, 120} {
			for _, phase := range []string{"running", "failed", "unconfirmed"} {
				t.Run(fmt.Sprintf("view%d/%d/%s", view, width, phase), func(t *testing.T) {
					row := menuThreads()[0]
					row.PublishedSummary = "working on compiler"
					at := time.Unix(100, 0)
					state := NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
					state.ObservedAt = at
					state.Frames[0].View = view
					state, effects := dispatchThreadOperation(state, "relaunch", row.Address)
					if len(effects) != 1 {
						t.Fatal("relaunch not admitted")
					}
					detail := strings.Repeat("long diagnostic 界 ", 100) + "\nfull final line"
					wantStatus := "relaunch… 7s"
					if phase == "running" {
						state = updateOperationProgress(state, effects[0].Attempt, detail)
					} else {
						state = reduceOperationResult(state, MenuEvent{Operation: "relaunch", Attempt: effects[0].Attempt, Address: row.Address, Error: detail, Unconfirmed: phase == "unconfirmed"})
						wantStatus = phase + ": long"
					}
					rendered := RenderMenuView(state, width, 10, at.Add(7*time.Second), false)
					assertRenderedBounds(t, rendered.Body, width, 10)
					if len(rendered.Extents) != 1 {
						t.Fatalf("row not drawn: %q", rendered.Body)
					}
					lines := strings.Split(string(ansi.Strip([]byte(rendered.Body))), "\r\n")
					line := lines[rendered.Extents[0].Start]
					if !strings.HasPrefix(line, "▸ compiler") || !strings.Contains(line, wantStatus) {
						t.Fatalf("row lost identity or job status: %q; want selected compiler and %q", line, wantStatus)
					}
					job, ok := menuOperationForRow(state, row)
					if !ok || job.Detail != detail {
						t.Fatal("rendering truncated retained detail")
					}
					if phase != "running" {
						copied, ok := selectedMenuOperationDetail(state)
						if !ok || copied != detail {
							t.Fatalf("Ctrl+Y selected detail = %q, %v", copied, ok)
						}
					}
				})
			}
		}
	}
}
