package couchcmd

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"strings"
	"testing"
)

func TestDrainCensusRetainsUnregisteredLiveRows(t *testing.T) {
	r, _, _ := slotRig(t)
	b := r.connect(1)
	known := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
	unknown := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: "unregistered"}
	hosted := couchcore.ThreadAddress{RepoScope: b.Scope, Tag: "hosted-only"}
	inv := couchtty.ResourceInventorySnapshot{InventoryKnown: true,
		Rows:   []couchcore.ActionableThreadSummary{{Address: known, State: couchcore.ThreadLive}, {Address: unknown, State: couchcore.ThreadLive}},
		Hosted: []couchcore.LiveTTYObservation{{Address: known}, {Address: hosted}}}
	rows := drainCensusRows(inv, r.s)
	if len(rows) != 2 {
		t.Fatalf("unsupported live wrappers disappeared or admitted wrapper duplicated: %+v", rows)
	}
	for _, row := range rows {
		if row.State != couchcore.DrainStuck || row.Reason == "" || row.Workspace == "" {
			t.Fatalf("missing actionable unknown: %+v", row)
		}
	}
	inv.InventoryKnown = false
	rows = drainCensusRows(inv, r.s)
	if len(rows) != 1 || rows[0].Workspace != "fleet" || rows[0].State != couchcore.DrainStuck {
		t.Fatalf("incomplete census granted absence: %+v", rows)
	}
}

func TestDrainCLILostMutationReplyNeverRepeats(t *testing.T) {
	inv, err := ParseCLI([]string{"--drain", "--reason", "restart"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	calls := 0
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(_ context.Context, _ string, req, resp any) error {
		calls++
		if req.(couchmessage.Request).ID == "" {
			t.Fatal("start has no recovery identity")
		}
		return context.DeadlineExceeded
	})
	if code == 0 || calls != 1 || !strings.Contains(stderr.String(), "--drain --status") {
		t.Fatalf("lost reply retried or lacks recovery: code%d calls%d stdout%q stderr%q", code, calls, out.String(), stderr.String())
	}
}

func TestDrainCLILostAckReplyNamesCheckpointGeneration(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	inv, err := ParseCLI([]string{"--drain", "--ack", id}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(context.Context, string, any, any) error { return context.DeadlineExceeded })
	if code == 0 || !strings.Contains(stderr.String(), id) || !strings.Contains(stderr.String(), "--drain --status") {
		t.Fatalf("lost ack lacks generation/recovery: %d %q", code, stderr.String())
	}
}

func TestDrainStewardAuthority(t *testing.T) {
	r := newHubMoveRig(t)
	for i, b := range r.callers {
		if err := r.s.authorizeDrainSteward(context.Background(), b); (err == nil) != (i == 0) {
			t.Fatalf("hub%d: %v", i, err)
		}
	}
	if r.s.authorizeDrainSteward(context.Background(), r.worker) == nil {
		t.Fatal("worker became steward")
	}
	for _, caller := range []couchmessage.Binding{r.callers[0], r.callers[1], r.worker} {
		req := callerRequest(caller, "drain")
		req.Drain = &couchmessage.DrainRequest{Action: "status"}
		got := r.s.handle(context.Background(), req)
		if got.Code != "ok" {
			t.Fatalf("read-only status: %+v", got)
		}
	}
}

func TestDrainProcessAdmissionUsesHubLease(t *testing.T) {
	h := guardHubState(t)
	h.drainGuard.Store(ptrDrainGuard(func(call couchcore.OperationCall) error { return couchmessage.ErrDraining }))
	called := false
	_, err := dispatchHubOperation(h, couchcore.OperationCall{Name: "resume", Args: map[string]string{"path": "/worker"}, Context: context.Background()}, func(couchcore.OperationCall) (any, error) { called = true; return nil, nil })
	if err == nil || called {
		t.Fatal("held process operation executed")
	}
	if !drainControlPlaneCall(h.snapshot(), couchcore.OperationCall{Name: "resume", Args: map[string]string{"path": "/repo0"}}) {
		t.Fatal("steward recovery blocked")
	}
	if drainControlPlaneCall(h.snapshot(), couchcore.OperationCall{Name: "resume", Args: map[string]string{"path": "/repo1"}}) {
		t.Fatal("other hub bypassed")
	}
}

func TestDrainCLIStatusPrintsBoard(t *testing.T) {
	inv, err := ParseCLI([]string{"--drain", "--status"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	calls := 0
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(_ context.Context, _ string, req, resp any) error {
		calls++
		if req.(couchmessage.Request).Drain.Action != "status" {
			t.Fatal("status mutated intent")
		}
		*resp.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Drain: &couchcore.DrainBoard{State: couchcore.DrainStuck, Detail: "queue unknown\x1b]evil\a", Rows: []couchcore.DrainRow{{Workspace: "pair:5", State: couchcore.DrainStuck, Reason: "no ack"}}}}
		return nil
	})
	if code != 0 || calls != 1 || !strings.Contains(out.String(), "queue unknown") || !strings.Contains(out.String(), "no ack") || strings.ContainsAny(out.String(), "\x1b\a") {
		t.Fatalf("code%d calls%d stdout%q stderr%q", code, calls, out.String(), stderr.String())
	}
}
