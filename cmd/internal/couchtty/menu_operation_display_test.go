package couchtty

import (
	"fmt"
	"reflect"
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
						state = updateOperationProgress(state, effects[0].Attempt, detail, state.ObservedAt)
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

// Progress age belongs to the current stage, not the operation's preparation.
func TestMenuOperationStageAgeProjectsRowAndNotice(t *testing.T) {
	at := time.Unix(100, 0)
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.ObservedAt = at
	state, effects := dispatchThreadOperation(state, "resume", menuAddress("couch-one"))
	attempt := effects[0].Attempt
	state.ObservedAt = at.Add(20 * time.Second)
	state = updateOperationProgress(state, attempt, "waiting for agent readiness", state.ObservedAt)
	state.ObservedAt = at.Add(23 * time.Second)
	state = updateOperationProgress(state, attempt, "waiting for agent readiness", state.ObservedAt)
	now := at.Add(25 * time.Second)
	_, job, _ := menuOperationForAttempt(state, attempt)
	if row := menuOperationRowText(job, now); !strings.Contains(row, "5s") || strings.Contains(row, "25s") {
		t.Fatalf("stage row age = %q, want 5s readiness", row)
	}
	view := string(ansi.Strip([]byte(RenderMenu(state, 160, 24, now, false))))
	if !strings.Contains(view, "waiting for agent readiness, 5s") {
		t.Fatalf("progress notice has no current-stage age: %q", view)
	}
	state.ObservedAt = at.Add(30 * time.Second)
	state = updateOperationProgress(state, attempt, "waiting for registration", state.ObservedAt)
	_, job, _ = menuOperationForAttempt(state, attempt)
	if row := menuOperationRowText(job, at.Add(32*time.Second)); row != "resume… 2s · waiting for registration" {
		t.Fatalf("new stage did not reset age: %q", row)
	}
	before := cloneMenuState(state)
	state = updateOperationProgress(state, attempt+1, "stale", state.ObservedAt)
	if !reflect.DeepEqual(state, before) {
		t.Fatal("stale attempt changed progress")
	}
	state = reduceOperationResult(state, MenuEvent{Operation: "resume", Attempt: attempt, Address: menuAddress("couch-one"), Success: true})
	before = cloneMenuState(state)
	state = updateOperationProgress(state, attempt, "late", state.ObservedAt)
	if !reflect.DeepEqual(state, before) {
		t.Fatal("terminal attempt changed progress")
	}
}

func TestMenuOperationWithoutStageKeepsOperationAge(t *testing.T) {
	at := time.Unix(100, 0)
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.ObservedAt = at
	state, effects := dispatchThreadOperation(state, "resume", menuAddress("couch-one"))
	_, job, _ := menuOperationForAttempt(state, effects[0].Attempt)
	if row := menuOperationRowText(job, at.Add(25*time.Second)); !strings.Contains(row, "25s") {
		t.Fatalf("ordinary row = %q", row)
	}
	if notice := renderedMenuNotice(state, at.Add(25*time.Second)); !strings.Contains(notice, "resuming compiler") || strings.Contains(notice, "25s") {
		t.Fatalf("ordinary notice = %q", notice)
	}
}
