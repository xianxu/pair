package couchcmd

import (
	"bytes"
	"context"
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

// reconcileRT injects a slot catalog and a provisioner whose reconcile blocks.
type reconcileRT struct {
	testRT
	catalog    *couchcore.SlotCatalogFake
	workspaces provisionFunc
}

type provisionFunc func(context.Context, couchcore.ProvisionRequest) (couchcore.ProvisionResult, error)

func (f provisionFunc) Ensure(ctx context.Context, r couchcore.ProvisionRequest) (couchcore.ProvisionResult, error) {
	return f(ctx, r)
}

func (r reconcileRT) NewCouchWith(runner couchcore.Runner, namespace couchcore.CouchNamespace) (*couchcore.Couch, error) {
	c, err := r.testRT.NewCouchWith(runner, namespace)
	if err == nil {
		c.Slots = r.catalog
		c.Workspaces = r.workspaces
	}
	return c, err
}

// TestReconcileCLIShowsTheReportOnABlockingFailure: a failed couch
// --reconcile prints the resources and plan it acted on beside the advice.
func TestReconcileCLIShowsTheReportOnABlockingFailure(t *testing.T) {
	primary := "/f/pair"
	identity := couchcore.WorkspaceIdentity{RepoIdentity: "/f/pair/.git", PrimaryRoot: primary}
	observation := couchcore.SlotObservation{Primary: primary, Number: 1, Resources: []couchcore.ResourceObservation{
		{ID: couchcore.ResourceEnv, State: couchcore.StatePresent},
		{ID: couchcore.ResourceSetup, State: couchcore.StateAbsent, Reason: "no marker"},
	}}
	rt := reconcileRT{
		testRT:  newRT(t),
		catalog: &couchcore.SlotCatalogFake{Repositories: map[string]couchcore.SlotRepository{primary: {Identity: identity}}},
		workspaces: func(context.Context, couchcore.ProvisionRequest) (couchcore.ProvisionResult, error) {
			return couchcore.ProvisionResult{}, &couchcore.SlotReconcileError{Address: "pair:1", Repo: "pair",
				Failure: couchcore.ReconcileFailure{Resource: couchcore.ResourceSetup, Class: couchcore.FailureHandoff, Cause: "Error: missing substrate"},
				Result:  couchcore.ReconcileResult{Observation: observation}}
		},
	}
	out, errw, code := runTypedRTWith(rt, couchcore.OperationCall{Name: "reconcile", Args: map[string]string{"ref": "/f/worktree/pair-slot1/pair"}})
	if code != 1 || !strings.Contains(out, "slot pair:1") || !strings.Contains(out, "setup") || !strings.Contains(errw, "ask the pair:0 agent") {
		t.Fatalf("code %d\nstdout:\n%s\nstderr:\n%s", code, out, errw)
	}
}

// runTypedRTWith runs one typed operation through the CLI's real dispatch with
// any runtime (runTypedRT's shape for wrapped runtimes).
func runTypedRTWith(rt Runtime, call couchcore.OperationCall) (string, string, int) {
	var out, errw bytes.Buffer
	op, ok := Resolve(call.Name)
	if !ok {
		return "", "unknown operation", 2
	}
	code := runTypedOperation(op, call.Args, nil, false, "", nil, nil, strings.NewReader(""), &out, &errw, rt)
	return out.String(), errw.String(), code
}
