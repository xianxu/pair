package launcher

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

type liveQuitLifecycleOps struct {
	runtime OSRuntime
	session string
	trace   []pairlifecycle.CleanupStage
}

func (o *liveQuitLifecycleOps) step(stage pairlifecycle.CleanupStage, effect func() error) error {
	if effect != nil {
		if err := effect(); err != nil {
			return err
		}
	}
	o.trace = append(o.trace, stage)
	return nil
}

func (o *liveQuitLifecycleOps) QuiesceSession(context.Context) error {
	return o.step(pairlifecycle.StageSessionQuiescence, func() error { return o.runtime.DeleteSession(o.session) })
}
func (o *liveQuitLifecycleOps) ReapEditors(context.Context) error {
	return o.step(pairlifecycle.StageEditorReap, nil)
}
func (o *liveQuitLifecycleOps) PreserveScrollback(context.Context, pairlifecycle.CleanupIntent) error {
	return o.step(pairlifecycle.StageScrollbackPreserve, nil)
}
func (o *liveQuitLifecycleOps) CleanupSidecars(context.Context) error {
	return o.step(pairlifecycle.StageSidecarCleanup, nil)
}
func (o *liveQuitLifecycleOps) CleanupPoller(context.Context) error {
	return o.step(pairlifecycle.StagePollerCleanup, nil)
}
func (o *liveQuitLifecycleOps) CleanupCmux(context.Context) error {
	return o.step(pairlifecycle.StageCmuxCleanup, nil)
}
func (o *liveQuitLifecycleOps) Now() time.Time { return time.Unix(100, 0).UTC() }

func TestQuitLifecycleLive(t *testing.T) {
	session := startControlledZellijSession(t)
	ops := &liveQuitLifecycleOps{
		runtime: OSRuntime{sessionQuiescence: newOSSessionQuiescenceOps(), sessionQuiesceWait: 10 * time.Second, sessionQuiescePoll: 50 * time.Millisecond},
		session: session,
	}
	result := pairlifecycle.RunCleanup(t.Context(), pairlifecycle.CleanupCouch, ops)
	if result.Outcome != pairlifecycle.CompletionSuccess {
		t.Fatalf("live cleanup = %+v", result)
	}
	want := []pairlifecycle.CleanupStage{
		pairlifecycle.StageSessionQuiescence, pairlifecycle.StageEditorReap,
		pairlifecycle.StageScrollbackPreserve, pairlifecycle.StageSidecarCleanup,
		pairlifecycle.StagePollerCleanup, pairlifecycle.StageCmuxCleanup,
	}
	if !reflect.DeepEqual(ops.trace, want) {
		t.Fatalf("live trace = %v, want %v", ops.trace, want)
	}
	fake := pairlifecycletest.New(time.Unix(100, 0).UTC())
	request := pairlifecycle.QuitRequest{
		SchemaVersion: pairlifecycle.SchemaVersion,
		Identity:      pairlifecycle.Identity{Nonce: "live-trace", RepoScope: "scope", Tag: "tag", PID: 42, ProcessIdentity: "process"},
		Attempt:       1, Session: "redacted", Mode: pairlifecycle.CleanupPreserveScrollback, CompletionKey: "completion",
	}
	fake.SetSession(request.Session, true)
	fakeTrace, err := pairlifecycletest.RunConformanceScenario(context.Background(), fake, request)
	if err != nil {
		t.Fatal(err)
	}
	var fakeCleanup []string
	for _, event := range fakeTrace {
		if len(event) > len("cleanup:") && event[:len("cleanup:")] == "cleanup:" {
			fakeCleanup = append(fakeCleanup, event)
		}
	}
	var liveCleanup []string
	for _, stage := range ops.trace {
		liveCleanup = append(liveCleanup, "cleanup:"+string(stage))
	}
	if !reflect.DeepEqual(liveCleanup, fakeCleanup) {
		t.Fatalf("fake/live cleanup traces differ: fake %v live %v", fakeCleanup, liveCleanup)
	}
}

var _ pairlifecycle.QuitLifecycleOps = (*liveQuitLifecycleOps)(nil)
