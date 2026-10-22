package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

func awaitProvenNative(ctx context.Context, check func() (bool, error)) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		ok, err := check()
		if err != nil || ok {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// TestLiveProvenOrphanReap uses only the shared disposable fleet and public
// Couch commands. The native sampler must remember actual ancestry before the
// fixture parent exits; no provenance or caller credentials are manufactured.
func TestLiveProvenOrphanReap(t *testing.T) {
	if os.Getenv("PAIR_LIVE_PROVEN_REAP") != "1" {
		t.Skip("set PAIR_LIVE_PROVEN_REAP=1 for disposable public-command conformance")
	}
	binaries := testcouch.Build(t)
	for _, kind := range []string{"claude", "codex"} {
		if !t.Run(kind, func(t *testing.T) {
			started := time.Now()
			fleet := testcouch.New(t, binaries, testcouch.Options{ExplicitHub: true, Workers: 1, AgentKind: kind, RuntimeBudget: 120 * time.Second})
			fleet.Environment = append(fleet.Environment, "COUCH_RESOURCE_CADENCE=1s")
			setup, stopSetup := context.WithTimeout(t.Context(), 180*time.Second)
			defer stopSetup()
			if err := fleet.Prepare(setup); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
			defer cancel()
			if err := fleet.Start(ctx); err != nil {
				t.Fatal(err)
			}
			if err := fleet.StartWorker(ctx, 1); err != nil {
				t.Fatal(err)
			}
			old, err := fleet.Agent(1)
			if err != nil {
				t.Fatal(err)
			}
			cohort, err := fleet.RetainDescendants(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			resources := func() (agentfixture.ResourceObservation, error) { return fleet.Resources(ctx) }
			keys := map[int]string{}
			for _, p := range cohort.Processes {
				keys[p.PID] = p.Generation
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				view, e := resources()
				if e != nil {
					return false, e
				}
				seen := 0
				for _, row := range view.Processes {
					if keys[row.Key.PID] == row.Key.Identity && !row.Orphan {
						seen++
					}
				}
				return seen == len(keys), nil
			}); err != nil {
				t.Fatalf("initial provenance: %v", err)
			}
			if err = cohort.ReleaseParent(ctx); err != nil {
				t.Fatal(err)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				rows, _, e := (procutil.NativeResourceReader{}).Table(ctx)
				if e != nil {
					return false, e
				}
				seen := 0
				for _, row := range rows {
					if keys[row.PID] == row.Identity && row.PPID == 1 {
						seen++
					}
				}
				return seen == len(keys), nil
			}); err != nil {
				t.Fatalf("native reparent barrier: %v", err)
			}
			// Freeze only the monitor-proven old worker cohort before the
			// fixture crash. Retained children and the steward cannot enter it.
			beforeCrash, err := resources()
			if err != nil {
				t.Fatal(err)
			}
			var oldCohort []testcouch.Resource
			for _, row := range beforeCrash.Processes {
				if row.Slot == "fixture:1" && keys[row.Key.PID] != row.Key.Identity {
					oldCohort = append(oldCohort, testcouch.Resource{ID: strconv.Itoa(row.Key.PID), Generation: row.Key.Identity, Kind: row.Name})
				}
			}
			if err = cohort.CrashActor(ctx, 1, oldCohort); err != nil {
				t.Fatal(err)
			}
			// The old terminal is genuinely dead, so normal public recovery
			// does not need to approve stopping its already-absent live panes.
			out, err := fleet.ActorCLI(ctx, 0, "--reboot", "fixture:1", "--confirm", "--json")
			if err != nil || out.ExitCode != 0 {
				t.Fatalf("public reboot after exact fixture crash: %+v %v", out, err)
			}
			var reboot couchmessage.OperationReceipt
			if err = json.Unmarshal([]byte(out.Stdout), &reboot); err != nil {
				t.Fatal(err)
			}
			if reboot.Status != couchmessage.ReceiptSucceeded {
				t.Fatalf("reboot not succeeded: %+v", reboot)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				now, e := fleet.Agent(1)
				return e == nil && now.Generation != old.Generation, nil
			}); err != nil {
				t.Fatal(err)
			}
			current, err := fleet.Agent(1)
			if err != nil {
				t.Fatal(err)
			}
			if reboot.Tag == "" {
				t.Fatalf("successful reboot omitted new binding tag: %+v", reboot)
			}
			if err = fleet.WaitActors(ctx, 2); err != nil {
				t.Fatalf("new actor registration: %v", err)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				actors, e := fleet.Actors(ctx)
				if e != nil {
					return false, e
				}
				for _, actor := range actors.Actors {
					if actor.Binding.Slot == "fixture:1" && actor.Binding.Tag == reboot.Tag {
						return true, nil
					}
				}
				return false, nil
			}); err != nil {
				t.Fatalf("new exact binding registration: %v", err)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				view, e := resources()
				if e != nil {
					return false, e
				}
				seen := 0
				for _, row := range view.Orphans {
					if keys[row.Key.PID] == row.Key.Identity && row.Orphan && row.OrphanKnown {
						seen++
					}
				}
				return seen == len(keys), nil
			}); err != nil {
				t.Fatalf("proven old identities: %v", err)
			}
			preview, err := fleet.ActorCLI(ctx, 0, "--reap-orphans", "--json")
			if !provenPreviewRefusal(preview, err) {
				t.Fatalf("preview did not produce a joined intentional refusal: %+v %v", preview, err)
			}
			var previewReport orphanReapReport
			if err = json.Unmarshal([]byte(preview.Stdout), &previewReport); err != nil {
				t.Fatal(err)
			}
			if previewReport.Confirmed || len(previewReport.Proven) != len(keys) {
				t.Fatalf("preview multiplicity: %+v", previewReport)
			}
			for _, row := range previewReport.Proven {
				if keys[row.Target.Key.PID] != row.Target.Key.Identity || row.Status != orphanRefused {
					t.Fatalf("unexpected preview row: %+v", row)
				}
			}
			textPreview, err := fleet.ActorCLI(ctx, 0, "--reap-orphans")
			if !provenPreviewRefusal(textPreview, err) {
				t.Fatalf("text preview did not produce a joined intentional refusal: %+v %v", textPreview, err)
			}
			for pid, identity := range keys {
				if procutil.Identity(strconv.Itoa(pid)) != identity {
					t.Fatalf("preview changed pid%d", pid)
				}
				if !strings.Contains(textPreview.Stdout, fmt.Sprintf("pid=%d identity=%s", pid, identity)) {
					t.Fatalf("text preview omitted exact pid%d/%s: %s", pid, identity, textPreview.Stdout)
				}
			}
			confirmedStarted := time.Now()
			confirmed, err := fleet.ActorCLI(ctx, 0, "--reap-orphans", "--confirm", "--json")
			confirmedElapsed := time.Since(confirmedStarted)
			if err != nil || confirmed.ExitCode != 0 {
				t.Fatalf("confirmed: %+v %v", confirmed, err)
			}
			// Decode through the actual CLI report so field identity and one result per
			// frozen member are checked independently from native absence.
			var report orphanReapReport
			if err = json.Unmarshal([]byte(confirmed.Stdout), &report); err != nil {
				t.Fatal(err)
			}
			if !report.Confirmed || len(report.Proven) != len(keys) {
				t.Fatalf("confirmed multiplicity: %+v", report)
			}
			seen := map[int]bool{}
			for _, row := range report.Proven {
				if keys[row.Target.Key.PID] != row.Target.Key.Identity || row.Status != orphanReaped || seen[row.Target.Key.PID] {
					t.Fatalf("unexpected exact outcome: %+v", row)
				}
				seen[row.Target.Key.PID] = true
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				for pid, identity := range keys {
					if procutil.Identity(strconv.Itoa(pid)) == identity {
						return false, nil
					}
				}
				return true, nil
			}); err != nil {
				t.Fatalf("exact old set not ended: %v", err)
			}
			if procutil.Identity(strconv.Itoa(current.PID)) != current.Generation {
				t.Fatal("current agent ended")
			}
			body := "proven-reap-current-still-live-" + kind
			sent, err := fleet.ActorCLI(ctx, 0, "--send-to", "fixture:1", "--message", body)
			if err != nil || sent.ExitCode != 0 {
				t.Fatalf("post-reap delivery: %+v %v", sent, err)
			}
			if err = awaitProvenNative(ctx, func() (bool, error) {
				accepted, e := fleet.Accepted(1)
				if errors.Is(e, os.ErrNotExist) {
					return false, nil
				}
				for _, input := range accepted {
					if input.PID == current.PID && strings.Contains(input.Body, body) {
						return true, nil
					}
				}
				return false, e
			}); err != nil {
				t.Fatal(err)
			}
			t.Logf("native proven reap: personality=%s old_exact=%v exact_outcome_count=%d current=%d/%s confirmed_elapsed=%s total_elapsed=%s result=%s", kind, keys, len(report.Proven), current.PID, current.Generation, confirmedElapsed, time.Since(started), confirmed.Stdout)
			cleanup, stopCleanup := context.WithTimeout(context.Background(), 15*time.Second)
			defer stopCleanup()
			if err = fleet.Close(cleanup); err != nil {
				t.Fatal(err)
			}
			if err = fleet.AssertCleaned(); err != nil {
				t.Fatal(err)
			}
		}) {
			break
		}
	}
}

// A nonzero CLI is expected only with the complete joined result from the
// fixture, never on unknown transport, truncation or cleanup failure.
func provenPreviewRefusal(result agentfixture.Result, err error) bool {
	return err != nil && strings.HasPrefix(result.ID, "cli-") && result.ExitCode == 1 && result.Error == "exit status 1" && !result.Truncated && !result.CleanupUncertain
}

func TestProvenPreviewRefusalRequiresJoinedTypedResult(t *testing.T) {
	good := agentfixture.Result{ID: "cli-1", ExitCode: 1, Error: "exit status 1"}
	if !provenPreviewRefusal(good, errors.New("fixture CLI: exit status 1")) {
		t.Fatal("rejected intentional refused preview")
	}
	for _, bad := range []agentfixture.Result{{ID: "cli-1", ExitCode: 1, Error: "read failed"}, {ID: "cli-1", ExitCode: 1, Error: "exit status 1", Truncated: true}, {ID: "cli-1", ExitCode: 1, Error: "exit status 1", CleanupUncertain: true}, {ExitCode: 1, Error: "exit status 1"}} {
		if provenPreviewRefusal(bad, errors.New("failure")) {
			t.Fatalf("accepted uncertain result: %+v", bad)
		}
	}
}
