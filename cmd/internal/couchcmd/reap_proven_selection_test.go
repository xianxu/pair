package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

const provenanceRefusal = "provenance incomplete; use the exact proven reap or restart Couch"

func selectionHealthyRows() []procutil.Process {
	return []procutil.Process{{PID: 11, PPID: 1, Identity: "old-child", Name: "codex"}, {PID: 12, PPID: 11, Identity: "current-descendant", Name: "codex"}, {PID: 20, PPID: 1, Identity: "current-own", Name: "zellij"}, {PID: 40, PPID: 1, Identity: "current-foreign", Name: "zellij"}}
}
func selectionSetWorld(table *composedReapTable, rows []procutil.Process) {
	table.rows = nil
	table.ids = map[int]string{}
	table.diesOn = map[int]syscall.Signal{11: syscall.SIGTERM, 12: syscall.SIGTERM, 51: syscall.SIGTERM, 52: syscall.SIGTERM}
	for _, row := range rows {
		table.rows = append(table.rows, launcher.ProcessRow{PID: row.PID, PPID: row.PPID, Identity: row.Identity, Name: row.Name})
		table.ids[row.PID] = row.Identity
	}
	table.reader.setRows(rows)
}
func selectionNvimRow() procutil.Process {
	identity := "1.0"
	if runtime.GOOS == "linux" {
		identity = "1"
	}
	return procutil.Process{PID: 51, PPID: 1, Identity: identity, Name: "nvim"}
}

// Called only after a joined worker sweep, before its next request. The shared
// fixture cadence is an hour; no concurrent collection can read these settings.
func selectionEnableLegacyNvim(s *resourceSampler, table *composedReapTable) {
	s.now = time.Now
	s.policy.OrphanNvimAge = 0
	table.reader.mu.Lock()
	table.reader.cwdPath = "/repo"
	table.reader.mu.Unlock()
}
func selectionRunCLI(t *testing.T, rig *serviceRig, rt testRT, table *composedReapTable) (orphanReapReport, int, string) {
	t.Helper()
	call := func(ctx context.Context, _ string, raw, out any) error {
		*out.(*couchmessage.Response) = rig.s.handle(ctx, raw.(couchmessage.Request))
		return nil
	}
	var stdout, stderr bytes.Buffer
	code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true, jsonOutput: true}, rt, &stdout, &stderr, call, fastOrphanReaper(table))
	var report orphanReapReport
	if stdout.Len() > 0 {
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("report: %v stdout%s stderr%s", err, stdout.String(), stderr.String())
		}
	} else if code != 1 || stderr.Len() == 0 {
		t.Fatalf("missing outcome or explicit refusal: exit%d stderr%s", code, stderr.String())
	}
	return report, code, stdout.String() + stderr.String()
}

func TestReapProvenSelectionCannotForgetLegacyProtection(t *testing.T) {
	for _, scenario := range []string{"overflow", "overflow-then-healthy", "repeated-overflow-then-healthy", "conflict-then-healthy", "incomplete-census-then-healthy", "incomplete-inventory-then-healthy", "unknown-detached-then-healthy"} {
		t.Run(scenario, func(t *testing.T) {
			rig, rt, table := composedReapWorld(t)
			rt.env["COUCH_THREAD_TAG"] = "t1" // This authenticated hub does not own old11.
			healthy := selectionHealthyRows()
			observed := append([]procutil.Process(nil), healthy...)
			sampler := rig.s.resources.Load()
			inventory := sampler.slots()
			switch {
			case strings.Contains(scenario, "overflow"):
				for i := 0; i < couchcore.ResourceMaxProcesses; i++ {
					observed = append(observed, procutil.Process{PID: 1000 + i, PPID: 20, Identity: fmt.Sprintf("current-%d", i), Name: "codex"})
				}
			case strings.HasPrefix(scenario, "conflict"):
				observed[0].PPID = 40
				// Child12 is introduced after the conflict, so it never independently
				// acquires a legitimate exact cleanup permit for caller hub1's slot.
				observed = []procutil.Process{observed[0], observed[2], observed[3]}
			case strings.HasPrefix(scenario, "incomplete-census"):
				table.reader.mu.Lock()
				table.reader.tableErr = errors.New("incomplete census")
				table.reader.mu.Unlock()
			case strings.HasPrefix(scenario, "incomplete-inventory"):
				incomplete := inventory
				incomplete.Complete = false
				sampler.slots = func() couchcore.ResourceSlotInventory { return incomplete }
			case strings.HasPrefix(scenario, "unknown-detached"):
				observed = append(observed, procutil.Process{PID: 61, PPID: 1, Identity: "unknown-detached", Name: "sdlc"})
			}
			selectionSetWorld(table, observed)
			first, err := sampler.sweep(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			diagnostics := strings.Join(first.view.Errors, " ")
			if strings.Contains(scenario, "overflow") && !strings.Contains(diagnostics, "selected process limit exceeded") {
				t.Fatalf("fixture did not reach selection cap: %s", diagnostics)
			}
			if strings.HasPrefix(scenario, "conflict") && !strings.Contains(diagnostics, "conflicting process ownership") {
				t.Fatalf("fixture did not reach conflict: %s", diagnostics)
			}
			if strings.HasPrefix(scenario, "repeated-overflow") {
				if _, err = sampler.sweep(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			if strings.HasSuffix(scenario, "then-healthy") {
				table.reader.mu.Lock()
				table.reader.tableErr = nil
				table.reader.mu.Unlock()
				sampler.slots = func() couchcore.ResourceSlotInventory { return inventory }
				selectionEnableLegacyNvim(sampler, table)
				selectionSetWorld(table, append(healthy, selectionNvimRow()))
				recovered, err := sampler.sweep(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(strings.Join(recovered.view.Errors, " "), "selected process limit exceeded") {
					t.Fatal("fixture never returned to healthy capacity")
				}
				foundNvim := false
				for _, row := range recovered.view.OrphanNvim {
					if row.Key.PID == 51 {
						foundNvim = true
					}
				}
				if !foundNvim {
					t.Fatal("healthy fixture did not classify the independent legacy nvim")
				}
			}
			report, code, output := selectionRunCLI(t, rig, rt, table)
			if len(table.signals) != 0 {
				t.Errorf("%s exposed a legacy tree: signals%v", scenario, table.signals)
			}
			for _, outcome := range report.LeftRunning {
				if outcome.Status == orphanReaped {
					t.Errorf("legacy overlap incorrectly reported reaped: %+v", outcome)
				}
			}
			for _, outcome := range report.Outcomes {
				if outcome.Status == orphanReaped {
					t.Errorf("legacy nvim incorrectly reported reaped: %+v", outcome)
				}
			}
			if table.Identity(11) != "old-child" || table.Identity(12) != "current-descendant" {
				t.Error("foreign process or current descendant ended")
			}
			if code != 1 || !strings.Contains(output, provenanceRefusal) {
				t.Errorf("missing sticky refusal: exit%d output%s", code, output)
			}
		})
	}
}

func TestReapProvenExactRemainsUsableAfterLegacyRefusal(t *testing.T) {
	rig, rt, table := composedReapWorld(t)
	sampler := rig.s.resources.Load()
	table.reader.mu.Lock()
	table.reader.tableErr = errors.New("incomplete census")
	table.reader.mu.Unlock()
	if _, err := sampler.sweep(t.Context()); err != nil {
		t.Fatal(err)
	}
	table.reader.mu.Lock()
	table.reader.tableErr = nil
	table.reader.mu.Unlock()
	selectionEnableLegacyNvim(sampler, table)
	// Both old children retain their exact provenance through the unknown pass.
	rows := append(selectionHealthyRows(), procutil.Process{PID: 31, PPID: 1, Identity: "foreign-child", Name: "codex"}, selectionNvimRow())
	selectionSetWorld(table, rows)
	report, code, output := selectionRunCLI(t, rig, rt, table)
	found := false
	for _, row := range report.Proven {
		if row.Target.Key.PID == 11 && row.Status == orphanReaped {
			found = true
		}
	}
	if !found || len(table.signals) != 1 || len(table.signals[11]) != 1 || table.signals[11][0] != syscall.SIGTERM {
		t.Fatalf("guarded exact target unavailable: signals%v report%+v output%s", table.signals, report, output)
	}
	if table.Identity(12) != "current-descendant" || table.Identity(31) != "foreign-child" || table.Identity(51) != selectionNvimRow().Identity {
		t.Fatal("exact cleanup touched unadmitted or legacy processes")
	}
	if code != 1 || !strings.Contains(output, provenanceRefusal) {
		t.Fatalf("lost legacy refusal alongside exact progress: exit%d output%s", code, output)
	}
}

func TestReapProvenFreshServiceRetainsCompleteLegacyCleanup(t *testing.T) {
	rig, rt, table := composedReapWorld(t)
	sampler := rig.s.resources.Load()
	if _, err := sampler.sweep(t.Context()); err != nil {
		t.Fatal(err)
	}
	selectionEnableLegacyNvim(sampler, table)
	rows := []procutil.Process{{PID: 20, PPID: 1, Identity: "current-own", Name: "zellij"}, {PID: 40, PPID: 1, Identity: "current-foreign", Name: "zellij"}, {PID: 52, PPID: 1, Identity: "clean-runner", Name: "codex"}, selectionNvimRow()}
	selectionSetWorld(table, rows)
	if err := sampler.leftRunning.Add(t.Context(), []couchcore.LeftRunningOrphan{{Origin: "slot:/repo", Process: couchcore.ProcessIdentity{PID: 52, Identity: "clean-runner"}, Command: "codex", LeftAt: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	report, code, output := selectionRunCLI(t, rig, rt, table)
	if code != 0 || strings.Contains(output, provenanceRefusal) {
		t.Fatalf("fresh complete service inherited refusal: exit%d output%s", code, output)
	}
	if len(table.signals) != 2 || len(table.signals[51]) != 1 || len(table.signals[52]) != 1 {
		t.Fatalf("normal legacy families not reaped: signals%v report%+v", table.signals, report)
	}
	if len(report.Outcomes) != 1 || report.Outcomes[0].Status != orphanReaped || len(report.LeftRunning) != 1 || report.LeftRunning[0].Status != orphanReaped {
		t.Fatalf("legacy results: %+v", report)
	}
}
