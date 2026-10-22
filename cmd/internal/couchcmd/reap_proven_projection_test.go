package couchcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

// The complete sample knows that a legacy row overlaps protected provenance.
// Its independently budgeted orphan prefix must not erase that protection and
// let the CLI expand a legacy tree into a live/current descendant.
func TestReapProvenProjectionCannotExposeLegacyOverlap(t *testing.T) {
	for _, family := range []string{"left-running", "orphan-nvim"} {
		for _, complete := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/complete=%v", family, complete), func(t *testing.T) {
				target := *effectTarget()
				view := couchcore.ResourceView{}
				for i := 0; i < 32; i++ {
					// Large, incomplete rows consume the orphan detail budget but cannot
					// authorize exact effects. Their data is an explicit refusal, not a reason
					// to permit a different, independently projected cleanup family.
					row := couchcore.ResourceProcessView{Name: strings.Repeat("<", 8192)}
					row.Key = couchcore.ResourceProcessKey{PID: 1000 + i, Identity: fmt.Sprintf("filler-%d", i)}
					row.Orphan, row.OrphanKnown = true, true
					view.Orphans = append(view.Orphans, row)
				}
				row := couchcore.ResourceProcessView{ResourceProvenance: couchcore.ResourceProvenance{ResourceRoot: couchcore.ResourceRoot{Key: target.Key, Slot: target.Slot, Thread: target.Thread, Workspace: target.Workspace}, Root: target.Root, Orphan: true, OrphanKnown: true}}
				if !complete {
					row.Workspace = couchcore.WorkspaceIdentity{}
				}
				view.Orphans = append(view.Orphans, row)
				if family == "left-running" {
					view.LeftRunning = []couchcore.LeftRunningOrphan{{Process: couchcore.ProcessIdentity{PID: target.Key.PID, Identity: target.Key.Identity}}}
				} else {
					view.OrphanNvim = []couchcore.OrphanNvim{orphanRow(target.Key.PID, target.Key.Identity)}
				}
				projected := projectResourceView(view)
				for _, visible := range projected.Orphans {
					if visible.Key.PID == target.Key.PID {
						t.Fatal("fixture target must be beyond the orphan detail prefix")
					}
				}
				if len(projected.Orphans) >= len(view.Orphans) || !strings.Contains(strings.Join(projected.Errors, " "), "orphans omitted") {
					t.Fatalf("fixture did not exceed orphan budget: %d/%d errors%v", len(projected.Orphans), len(view.Orphans), projected.Errors)
				}
				table := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 11, PPID: 1, Identity: "old-child"}, {PID: 12, PPID: 11, Identity: "current-descendant"}}, ids: map[int]string{11: "old-child", 12: "current-descendant"}, diesOn: map[int]syscall.Signal{11: syscall.SIGTERM, 12: syscall.SIGTERM}}
				var requests []couchmessage.Request
				var stdout, stderr bytes.Buffer
				code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true, jsonOutput: true}, testRT{dir: t.TempDir()}, &stdout, &stderr, reapCLIStub(t, projected, &requests), fastOrphanReaper(table))
				var report orphanReapReport
				if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
					t.Fatalf("report: %v stdout%s stderr%s", err, stdout.String(), stderr.String())
				}
				if len(table.signals) != 0 {
					t.Errorf("projection exposed %s legacy tree: signals%v; current descendant must remain protected", family, table.signals)
				}
				for _, outcome := range report.Outcomes {
					if outcome.Row.Key.PID == 11 && outcome.Status == orphanReaped {
						t.Errorf("overlap incorrectly reported reaped: %+v", outcome)
					}
				}
				for _, outcome := range report.LeftRunning {
					if outcome.Entry.Process.PID == 11 && outcome.Status == orphanReaped {
						t.Errorf("overlap incorrectly reported reaped: %+v", outcome)
					}
				}
				if code != 1 {
					t.Errorf("incomplete projected evidence should retain refusal: exit%d report%+v", code, report)
				}
				if len(requests) != 1 {
					t.Errorf("unexpected effects beyond classification: %+v", requests)
				}
			})
		}
	}
}

func TestReapProvenProjectionPreservesUnrelatedLegacyRows(t *testing.T) {
	view := couchcore.ResourceView{
		Orphans: []couchcore.ResourceProcessView{{ResourceProvenance: couchcore.ResourceProvenance{
			ResourceRoot: couchcore.ResourceRoot{Key: couchcore.ResourceProcessKey{PID: 11}},
		}}},
		OrphanNvim: []couchcore.OrphanNvim{orphanRow(11, "old"), orphanRow(11, "replacement"), orphanRow(12, "unrelated")},
		LeftRunning: []couchcore.LeftRunningOrphan{
			{Process: couchcore.ProcessIdentity{PID: 11, Identity: "old"}},
			{Process: couchcore.ProcessIdentity{PID: 11, Identity: "replacement"}},
			{Process: couchcore.ProcessIdentity{PID: 12, Identity: "unrelated"}},
		},
	}
	projected := projectResourceView(view)
	for range 2 {
		if len(projected.OrphanNvim) != 1 || projected.OrphanNvim[0].Key.PID != 12 ||
			len(projected.LeftRunning) != 1 || projected.LeftRunning[0].Process.PID != 12 {
			t.Fatalf("overlap protection changed unrelated legacy rows: %+v", projected)
		}
		projected = projectResourceView(projected)
	}
	if len(view.OrphanNvim) != 3 || len(view.LeftRunning) != 3 ||
		view.OrphanNvim[0].Key.PID != 11 || view.LeftRunning[0].Process.PID != 11 {
		t.Fatal("projection mutated the complete source evidence")
	}
}
