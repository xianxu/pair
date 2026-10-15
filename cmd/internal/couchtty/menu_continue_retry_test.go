package couchtty

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

type menuRetryRunner struct {
	couchcore.Runner
	failures, attempts int
}

func (r *menuRetryRunner) StartBlocked(ctx context.Context, dir string, argv, env []string, timeout time.Duration) (couchcore.BlockedHandle, error) {
	r.attempts++
	if r.attempts <= r.failures {
		return nil, errors.New("temporary attachment failure")
	}
	return r.Runner.StartBlocked(ctx, dir, argv, env, timeout)
}

func TestFirstRowEnterAndClickShareExplicitContinueRetry(t *testing.T) {
	for _, gesture := range []string{"enter", "click"} {
		for _, failures := range []int{1, 2} {
			t.Run(gesture+string(rune('0'+failures)), func(t *testing.T) {
				ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/repo")
				if err != nil {
					t.Fatal(err)
				}
				runner := &menuRetryRunner{Runner: couchcore.NewFakeRunner(), failures: failures}
				artifacts := couchcore.NewFakeThreadArtifactCollisionChecker()
				owner, err := couchcore.New(ns, runner, couchcore.NewFakePathOps(nil), couchcore.NewFakeGit(nil), couchcore.NewFakeProcOps(), couchcore.NewStore(ns.Dir()), couchcore.FixedClock{T: time.Now()}, couchcore.NewFixedIDGen("attached-client"), nil, artifacts)
				if err != nil {
					t.Fatal(err)
				}
				record, err := owner.Threads.CreateThread(couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"}, StartingPath: "/repo", WorkingPath: "/repo", CreatedAt: time.Now(), Revision: 1})
				if err != nil {
					t.Fatal(err)
				}
				record, err = owner.Threads.GetThread(record.Address)
				if err != nil {
					t.Fatal(err)
				}
				artifacts.SetPairSession(record.Address, "pair-"+string(record.Address.Tag), true)
				row := couchcore.ActionableThreadSummary{Address: record.Address, State: couchcore.ThreadDetached, StartingPath: "/repo", WorkingPath: "/repo"}
				state := NewMenuState([]couchcore.ActionableThreadSummary{row}, couchcore.ThreadAddress{})
				if state.CurrentFrame().SelectedAddress != record.Address {
					t.Fatal("first row was not selected")
				}
				event := MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyEnter}}
				if gesture == "click" {
					event = MenuEvent{Kind: MenuEventMouseSwitch, Address: record.Address}
				}
				next, effects := ReduceMenu(state, event)
				if len(effects) != 1 || effects[0].Operation != "resume" || effects[0].Background {
					t.Fatalf("Continue bypassed explicit resume: %+v", effects)
				}
				result, err := couchcore.DispatchOperation(couchcore.OperationExecutors{LiveOwner: couchcore.CouchLiveOwnerExecutor(owner)}, couchcore.OperationCall{Name: effects[0].Operation, Args: effects[0].Args, Implicit: true, Context: context.Background()})
				if runner.attempts != 2 {
					t.Fatalf("shared attachment helper attempts=%d", runner.attempts)
				}
				completion := MenuEvent{Kind: MenuEventOperationResult, Operation: effects[0].Operation, Attempt: effects[0].Attempt, Address: record.Address, Success: err == nil}
				if err != nil {
					completion.Error = couchcore.OperatorError(err)
				}
				completed, _ := ReduceMenu(next, completion)
				if failures == 1 {
					started, ok := result.(couchcore.StartResult)
					if err != nil || !ok || started.Handle == nil {
						t.Fatalf("retry did not return attachment: %v %v", result, err)
					}
				} else if err == nil || !strings.Contains(completed.Notice.Text, "temporary attachment failure") || !strings.Contains(completed.Notice.Text, "2 attempts") {
					t.Fatalf("exhausted retry was silently skipped: notice=%+v err=%v", completed.Notice, err)
				}
				after, err := owner.Threads.GetThread(record.Address)
				if err != nil || !reflect.DeepEqual(record, after) {
					t.Fatalf("borrowed Continue changed durable state: %+v %v", after, err)
				}
			})
		}
	}
}
