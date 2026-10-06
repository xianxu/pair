package couchcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// TestRenderSlotReportFollowsTheResourceTable renders an observation built
// from the table's own order, so a resource added to the table is printed
// without this test (or the renderer) listing it.
func TestRenderSlotReportFollowsTheResourceTable(t *testing.T) {
	order, err := couchcore.SlotResourceOrder()
	if err != nil {
		t.Fatal(err)
	}
	var obs couchcore.SlotObservation
	for _, id := range order {
		r := couchcore.ResourceObservation{ID: id, State: couchcore.StatePresent}
		if id == couchcore.ResourceDep {
			r.ID = couchcore.DepResource("ariadne")
			r.State = couchcore.StateAbsent
		}
		obs.Resources = append(obs.Resources, r)
	}
	setupAt := len(obs.Resources) - 2
	obs.Resources[setupAt].State, obs.Resources[setupAt].Sub = couchcore.StateUnknown, ""
	obs.Resources[setupAt].Reason = "read setup marker: permission denied"
	var out bytes.Buffer
	renderSlotReport(&out, couchcore.SlotReport{Address: "tools:1", Observation: obs, Plan: couchcore.SlotPlan{
		Stops: []couchcore.PlanStop{{Resource: obs.Resources[setupAt].ID, Class: couchcore.StopUnknown, Reason: "read setup marker: permission denied"}},
	}})
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if lines[0] != "slot tools:1" {
		t.Fatalf("header %q", lines[0])
	}
	if len(lines) != len(obs.Resources)+2 {
		t.Fatalf("got %d lines for %d resources:\n%s", len(lines), len(obs.Resources), out.String())
	}
	for i, r := range obs.Resources {
		if fields := strings.Fields(lines[i+1]); len(fields) < 2 || fields[0] != string(r.ID) {
			t.Errorf("line %d = %q, want resource %s", i+1, lines[i+1], r.ID)
		}
	}
	if !strings.Contains(out.String(), "dep:ariadne    absent") {
		t.Errorf("dependency line missing:\n%s", out.String())
	}
	if want := "plan: stops at " + string(obs.Resources[setupAt].ID) + " (unknown): read setup marker: permission denied"; lines[len(lines)-1] != want {
		t.Errorf("plan line %q, want %q", lines[len(lines)-1], want)
	}
}

func TestRenderSlotReportConverged(t *testing.T) {
	var out bytes.Buffer
	renderSlotReport(&out, couchcore.SlotReport{Address: "pair:2", Observation: couchcore.SlotObservation{
		Resources: []couchcore.ResourceObservation{{ID: couchcore.ResourceEnv, State: couchcore.StatePresent}, {ID: couchcore.ResourceStore, State: couchcore.StatePending, Reason: "waits for env"}},
	}})
	if !strings.Contains(out.String(), "store          pending: waits for env") || !strings.HasSuffix(out.String(), "plan: nothing to do\n") {
		t.Fatalf("output:\n%s", out.String())
	}
}
