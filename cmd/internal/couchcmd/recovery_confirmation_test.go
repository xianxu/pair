package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"strings"
	"testing"
)

func TestRecoveryConfirmationStandaloneCLI(t *testing.T) {
	for _, args := range [][]string{{"--recovery-confirmation", "opaque-token"}, {"--recovery-confirmation", "opaque-token", "--json"}} {
		inv, err := ParseCLI(args, couchcore.Operations())
		if err != nil || inv.messageOp != "recovery-confirmation" || inv.recoveryConfirmation != "opaque-token" || inv.ref != "" {
			t.Fatalf("standalone %v: %+v %v", args, inv, err)
		}
	}
	for _, args := range [][]string{{"--recovery-confirmation"}, {"--recovery-confirmation", ""}, {"--recovery-confirmation", "token", "pair:2"}, {"--recovery-confirmation", "token", "--confirm"}} {
		if _, err := ParseCLI(args, couchcore.Operations()); err == nil {
			t.Fatalf("accepted %s", strings.Join(args, " "))
		}
	}
}

func TestRecoveryFreshCLIExecutableWithoutCandidate(t *testing.T) {
	var out, diagnostic bytes.Buffer
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1"}, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &couchcore.RecoveryPreview{Incomplete: true, Fresh: &couchcore.RecoveryChoice{Label: "Start fresh"}}}
		return nil
	})
	if code != 0 || !strings.Contains(out.String(), "couch --reboot pair:1 --confirm") || strings.Contains(out.String(), "--candidate") {
		t.Fatalf("fresh command: %d %s %s", code, &out, &diagnostic)
	}
}
func TestRecoveryConfirmationOutcomeRetainsExactOffer(t *testing.T) {
	offer := couchcore.RecoveryConfirmationOffer{Token: "token", Command: "couch --recovery-confirmation token", Stops: []couchcore.RecoveryStopProcess{{PID: 12, Identity: "generation", Role: "helper"}}}
	got := slotOperationOutcome(nil, &couchcore.RecoveryConfirmationRequired{Offer: offer})
	if got.Code != "confirmation-required" || got.Confirmation == nil || got.Confirmation.Stops[0].Identity != "generation" || !strings.Contains(got.Detail, offer.Command) {
		t.Fatalf("offer lost: %+v", got)
	}
}

func TestRecoveryConfirmationHubTokenReplayUsesReceipt(t *testing.T) {
	r, runner, _ := slotRig(t, func(a messageAuthority) messageAuthority {
		a.hub = "/repo0"
		return a
	})
	caller := r.connect(0)
	calls := 0
	r.slotOps.reference = func(context.Context, string) (string, error) { calls++; return "pair:2", nil }
	r.slotOps.recovery = func(key, actor string, req couchmessage.Request, guard func(context.Context) error, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
		if actor != recoveryActor(caller) || req.Target != "pair:2" || guard == nil {
			t.Fatalf("lost authenticated authority %s %+v", actor, req)
		}
		if err := guard(context.Background()); err != nil {
			return err
		}
		return runner.run(key, req.Op, req.Target, couchcore.LiveRestartOptions{}, started, progress, finished)
	}
	req := slotRequest(caller, "recovery-confirmation", "confirm-id", "")
	req.RecoveryConfirmation = "token"
	first := switchCall(t, r, req)
	if first.Code != "accepted" || runner.count() != 1 {
		t.Fatalf("confirmation refused: %+v", first)
	}
	runner.job(0).finished(nil, nil)
	repeat := switchCall(t, r, req)
	if repeat.Code != "accepted" || repeat.Operation.Status != couchmessage.ReceiptSucceeded || runner.count() != 1 || calls != 1 {
		t.Fatalf("replay %+v calls=%d jobs=%d", repeat, calls, runner.count())
	}
}

func TestRecoveryConfirmationCLIUsesTokenOnlyRequest(t *testing.T) {
	var out, diagnostic bytes.Buffer
	inv, err := ParseCLI([]string{"--recovery-confirmation", "exact-token", "--json"}, couchcore.Operations())
	if err != nil {
		t.Fatal(err)
	}
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, request any, response any) error {
		req := request.(couchmessage.Request)
		if req.Op != "recovery-confirmation" || req.Target != "" || req.RecoveryConfirmation != "exact-token" || req.ID == "" {
			t.Fatalf("invalid standalone request: %+v", req)
		}
		if err := couchmessage.ValidateRequest(req); err != nil {
			t.Fatal(err)
		}
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "accepted", Operation: &couchmessage.OperationReceipt{ID: req.ID, Op: req.Op, Target: "pair:2", Status: couchmessage.ReceiptSucceeded}}
		return nil
	})
	if code != 0 || !strings.Contains(out.String(), "succeeded") {
		t.Fatalf("standalone failed: %d %s %s", code, &out, &diagnostic)
	}
}

func TestRecoveryConfirmationHubOwnershipAndGenerationRechecked(t *testing.T) {
	h := guardHubState(t)
	r, runner, _ := slotRig(t, func(a messageAuthority) messageAuthority {
		a.hubs = h
		return a
	})
	caller := r.connect(0)
	r.waitWorkspace(caller)
	wireGuardTestMemo(r, func(context.Context, string) (couchcore.HubSlot, error) {
		return couchcore.HubSlot{Repository: "/repo", Number: 2}, nil
	})
	r.slotOps.reference = func(context.Context, string) (string, error) { return "pair:2", nil }
	var queuedGuard func(context.Context) error
	r.slotOps.recovery = func(key, actor string, req couchmessage.Request, guard func(context.Context) error, started func(), progress couchcore.OperationProgress, finished func(any, error)) error {
		queuedGuard = guard
		return runner.run(key, req.Op, req.Target, couchcore.LiveRestartOptions{}, started, progress, finished)
	}
	req := slotRequest(caller, "recovery-confirmation", "guarded-confirmation", "")
	req.RecoveryConfirmation = "token"
	if got := switchCall(t, r, req); got.Code != "refused" || runner.count() != 0 {
		t.Fatalf("cross-hub accepted: %+v", got)
	}
	var err error
	h.mu.Lock()
	h.state, err = h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo1", Target: "/repo0", Revision: h.state.Revision()})
	h.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if got := switchCall(t, r, req); got.Code != "accepted" || queuedGuard == nil {
		t.Fatalf("own hub refused: %+v", got)
	}
	if err := queuedGuard(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.world.set(caller, func(s *worldSlot) { s.binding.Start = "replaced-generation" })
	if err := queuedGuard(context.Background()); err == nil {
		t.Fatal("replaced generation retained confirmation authority")
	}
	r.world.set(caller, func(s *worldSlot) { s.binding.Start = caller.Start })
	h.mu.Lock()
	h.state, err = h.state.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/repo", Number: 2}}, Caller: "/repo0", Target: "/repo1", Revision: h.state.Revision()})
	h.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if err := queuedGuard(context.Background()); err == nil {
		t.Fatal("ownership move retained confirmation authority")
	}
}

func TestRecoveryProvenCLIUsesPlainResumeWithoutSavedScreen(t *testing.T) {
	var preview couchcore.RecoveryPreview
	if err := json.Unmarshal([]byte(`{"proven":{"label":"Resume from disk (confirmed)","agent":"claude","native_id":"proven-native"},"incomplete":true,"fresh":{"label":"Start fresh"}}`), &preview); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1"}, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &preview}
		return nil
	})
	if code != 0 || !strings.Contains(out.String(), "proven-native") || !strings.Contains(out.String(), "couch --resume pair:1\n") || strings.Contains(out.String(), "--candidate") {
		t.Fatalf("proven resume lost: %d %s %s", code, &out, &diagnostic)
	}
}

func TestRecoveryProvenCLIBoundsAlternativesWithoutInventingScores(t *testing.T) {
	preview := couchcore.RecoveryPreview{Proven: &couchcore.RecoveryChoice{Label: "Resume from disk (confirmed)", NativeID: "proven-native"}, Candidates: []couchcore.RecoveryChoice{{Key: "one"}, {Key: "two"}, {Key: "three"}, {Key: "four"}, {Key: "five"}}, Fresh: &couchcore.RecoveryChoice{Label: "Start fresh"}}
	var out, diagnostic bytes.Buffer
	code := runMessageCLIWithCall(cliInvocation{kind: cliMessage, messageOp: "resume-candidates", ref: "pair:1"}, messageRuntime(), &out, &diagnostic, func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", RecoveryPreview: &preview}
		return nil
	})
	text := out.String()
	if code != 0 || strings.Contains(text, "--candidate five") || !strings.Contains(text, "--candidate four") || !strings.Contains(text, "established native binding") || strings.Contains(text, "0%") {
		t.Fatalf("alternative bound/proven score: %d %s %s", code, text, &diagnostic)
	}
}
