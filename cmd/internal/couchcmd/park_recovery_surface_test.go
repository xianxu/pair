package couchcmd

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"strings"
	"testing"
)

func TestRecoverParkCLIRequiresExactConfirmedSlot(t *testing.T) {
	for _, args := range [][]string{{"--recover-park", "pair:2", "--confirm"}, {"--recover-park", "pair:0", "--json", "--confirm"}} {
		got, err := ParseCLI(args, couchcore.Operations())
		if err != nil || got.messageOp != "recover-park" || !got.confirmed {
			t.Errorf("%v: %+v %v", args, got, err)
		}
	}
	for _, args := range [][]string{{"--recover-park", "pair:2"}, {"--recover-park", "pair", "--confirm"}, {"--recover-park", "pair:2", "--confirm", "--force-unknown"}, {"--park", "pair:2", "--confirm"}, {"--retry-park", "pair:2", "--confirm"}, {"--abandon-park", "pair:2", "--confirm"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestRecoverParkSocketRequiresAuthenticatedCallerAndRetainsReceipt(t *testing.T) {
	r, runner, _ := slotRig(t)
	unregistered := r.world.add(3)
	req := slotRequest(unregistered, "recover-park", "park-1", "pair:2")
	req.Confirmed = true
	if got := switchCall(t, r, req); got.Code == "accepted" || runner.count() != 0 {
		t.Fatalf("unauthenticated caller: %+v", got)
	}
	r.slotOps.guardedRun = func(key, op, target string, opts couchcore.LiveRestartOptions, guard func(context.Context) error, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
		if err := guard(context.Background()); err != nil {
			return err
		}
		return runner.run(key, op, target, opts, started, progress, finished)
	}
	caller := r.connect(0)
	req = slotRequest(caller, "recover-park", "park-1", "pair:2")
	req.Confirmed = true
	unconfirmed := req
	unconfirmed.Confirmed = false
	if got := switchCall(t, r, unconfirmed); got.Code != "confirmation-required" || runner.count() != 0 {
		t.Fatalf("unconfirmed %+v", got)
	}
	got := switchCall(t, r, req)
	if got.Code != "accepted" || runner.count() != 1 {
		t.Fatalf("authenticated caller: %+v jobs=%d", got, runner.count())
	}
	duplicate := switchCall(t, r, req)
	if duplicate.Code != "accepted" || runner.count() != 1 {
		t.Fatalf("duplicate %+v jobs=%d", duplicate, runner.count())
	}
	runner.job(0).started()
	runner.job(0).finished(nil, nil)
	status := switchCall(t, r, statusRequest(caller, "park-1"))
	if status.Operation == nil || status.Operation.Status != "succeeded" {
		t.Fatalf("receipt %+v", status)
	}
}

func TestRecoverParkDuplicateReceiptDoesNotBindAgain(t *testing.T) {
	r, runner, _ := slotRig(t)
	caller := r.connect(0)
	bindings := 0
	r.slotOps.preparePark = func(ctx context.Context, target string, _ couchcore.RepositoryName, guard func(context.Context) error) (slotOperationRunner, error) {
		bindings++
		return runner.run, nil
	}
	req := slotRequest(caller, "recover-park", "bound-once", "pair:2")
	req.Confirmed = true
	for i := 0; i < 2; i++ {
		if got := r.slotOps.handle(context.Background(), caller, req); got.Code != "accepted" {
			t.Fatalf("admit %+v", got)
		}
	}
	if bindings != 1 || runner.count() != 1 {
		t.Fatalf("bindings=%d jobs=%d", bindings, runner.count())
	}
}

func TestRecoverParkReceiptRetainsOutcomeAndNextActionWithoutStartedChild(t *testing.T) {
	result := couchcore.ParkRecoveryResult{Outcome: "interrupted-cleanup-retired", OperatorOutcome: "resume-from-disk", NextCommand: "couch --resume pair:2", Diagnostic: "scrollback incomplete", Thread: couchcore.ThreadRecord{Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: "thread"}}}
	if _, ok := any(result).(couchcore.StartedChild); ok {
		t.Fatal("recovery result must never start a child")
	}
	for _, value := range []any{result, notedResult{value: result, note: "inspected"}} {
		got := slotOperationOutcome(value, nil)
		if got.Code != result.Outcome || got.Tag != "thread" || !strings.Contains(got.Warning, result.NextCommand) || !strings.Contains(got.Warning, result.Diagnostic) {
			t.Fatalf("lost recovery outcome: %+v", got)
		}
	}
}

func TestRecoverParkCurrentTargetGuardRejectsReplacement(t *testing.T) {
	bound := couchcore.OperationCall{Args: map[string]string{"repo-scope": "scope", "tag": "original", "park-nonce": "nonce"}}
	for _, tc := range []struct {
		scope, tag, nonce string
		wantErr           bool
	}{{"scope", "original", "nonce", false}, {"scope", "original", "", false}, {"scope", "original", "new-park", true}, {"scope", "replacement", "nonce", true}, {"other", "original", "nonce", true}} {
		current := couchcore.OperationCall{Args: map[string]string{"repo-scope": tc.scope, "tag": tc.tag, "park-nonce": tc.nonce}}
		if err := validateParkRecoveryTarget(bound, current); (err != nil) != tc.wantErr {
			t.Errorf("current %+v: %v", current.Args, err)
		}
	}
}

func TestRecoverParkUnconfirmedDoesNotBind(t *testing.T) {
	r, runner, _ := slotRig(t)
	caller := r.connect(0)
	bindings := 0
	r.slotOps.preparePark = func(context.Context, string, couchcore.RepositoryName, func(context.Context) error) (slotOperationRunner, error) {
		bindings++
		return runner.run, nil
	}
	request := slotRequest(caller, "recover-park", "no-confirm", "pair:2")
	got := r.slotOps.handle(context.Background(), caller, request)
	if got.Code != "confirmation-required" || bindings != 0 || runner.count() != 0 {
		t.Fatalf("unconfirmed binding: response=%+v bindings=%d jobs=%d", got, bindings, runner.count())
	}
}

func TestRecoverParkUnknownCacheRefusesWithRefreshAdvice(t *testing.T) {
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), strings.NewReader(""))
	t.Cleanup(console.Stop)
	ops := consoleSlotOperations(console, &couchcore.Couch{}, nil)
	_, err := ops.preparePark(context.Background(), "pair:2", couchcore.RepositoryName{Key: "/src/pair"}, nil)
	if err == nil || !strings.Contains(err.Error(), "refresh") || !strings.Contains(err.Error(), "retry") {
		t.Fatalf("unknown cached selection needs refresh/retry advice: %v", err)
	}
}

func TestRecoverParkRetainedReceiptPrecedesRepositoryLookup(t *testing.T) {
	r, runner, _ := slotRig(t)
	caller := r.connect(0)
	bindings := 0
	r.slotOps.preparePark = func(context.Context, string, couchcore.RepositoryName, func(context.Context) error) (slotOperationRunner, error) {
		bindings++
		return runner.run, nil
	}
	request := slotRequest(caller, "recover-park", "retained-before-lookup", "pa:2")
	request.Confirmed = true
	first := r.slotOps.handle(context.Background(), caller, request)
	if first.Code != "accepted" {
		t.Fatalf("initial: %+v", first)
	}
	lookups := 0
	r.slotOps.names = func(context.Context) ([]couchcore.RepositoryName, error) {
		lookups++
		return nil, errors.New("repository catalog unavailable")
	}
	duplicate := r.slotOps.handle(context.Background(), caller, request)
	if duplicate.Code != "accepted" || duplicate.Operation.ID != first.Operation.ID || lookups != 0 || bindings != 1 || runner.count() != 1 {
		t.Fatalf("retained duplicate: %+v lookups=%d bindings=%d jobs=%d", duplicate, lookups, bindings, runner.count())
	}
	request.Target = "removed-alias:2"
	conflict := r.slotOps.handle(context.Background(), caller, request)
	if conflict.Code != "id-conflict" || lookups != 0 || bindings != 1 || runner.count() != 1 {
		t.Fatalf("retained conflict: %+v lookups=%d bindings=%d jobs=%d", conflict, lookups, bindings, runner.count())
	}
}
