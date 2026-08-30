// Package pairlifecycletest provides a stateful lifecycle fake shared by Pair
// entry-path and coordinator tests. State changes only through modeled durable
// publication, delivery, cleanup, and restart operations.
package pairlifecycletest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

type PublicationState string

const (
	Prepared  PublicationState = "prepared"
	Committed PublicationState = "committed"
)

type requestState struct {
	request pairlifecycle.QuitRequest
	state   PublicationState
}

type completionState struct {
	result pairlifecycle.CleanupResult
	state  PublicationState
}

// Fake models durable request/result state separately from process-local
// ownership. Restart deliberately clears only the latter.
type Fake struct {
	mu sync.Mutex

	now         time.Time
	requests    map[string]requestState
	completions map[string]completionState
	sessions    map[string]bool
	triggers    map[string]int
	inProgress  map[string]bool
	done        map[string]map[pairlifecycle.CleanupStage]bool
	failures    map[pairlifecycle.CleanupStage]error
	active      string
	effective   []pairlifecycle.CleanupStage
	attempts    []pairlifecycle.CleanupStage
}

func New(now time.Time) *Fake {
	return &Fake{
		now: now, requests: map[string]requestState{}, completions: map[string]completionState{},
		sessions: map[string]bool{}, triggers: map[string]int{}, inProgress: map[string]bool{},
		done: map[string]map[pairlifecycle.CleanupStage]bool{}, failures: map[pairlifecycle.CleanupStage]error{},
	}
}

func (f *Fake) PrepareRequest(request pairlifecycle.QuitRequest) error {
	if err := pairlifecycle.ValidateQuitRequest(request); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	key := attemptKey(request)
	if current, ok := f.requests[key]; ok && !reflect.DeepEqual(current.request, request) {
		return errors.New("immutable prepared request conflicts")
	}
	f.requests[key] = requestState{request: request, state: Prepared}
	return nil
}

func (f *Fake) CommitRequest(request pairlifecycle.QuitRequest) error {
	if err := pairlifecycle.ValidateQuitRequest(request); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	key := attemptKey(request)
	if current, ok := f.requests[key]; ok && !reflect.DeepEqual(current.request, request) {
		return errors.New("immutable committed request conflicts")
	}
	f.requests[key] = requestState{request: request, state: Committed}
	return nil
}

func (f *Fake) SetSession(session string, present bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions[session] = present
}

func (f *Fake) SessionPresent(session string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sessions[session]
}

func (f *Fake) DeliverTrigger(request pairlifecycle.QuitRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := attemptKey(request)
	stored, ok := f.requests[key]
	if !ok || stored.state != Committed || !reflect.DeepEqual(stored.request, request) {
		return errors.New("trigger requires the exact committed request")
	}
	if !f.sessions[request.Session] {
		return fmt.Errorf("exact session %q is absent", request.Session)
	}
	f.triggers[key]++
	if _, completed := f.completions[key]; !completed {
		f.inProgress[key] = true
		f.active = key
		if f.done[key] == nil {
			f.done[key] = map[pairlifecycle.CleanupStage]bool{}
		}
	}
	return nil
}

func (f *Fake) DeliveredTriggers(request pairlifecycle.QuitRequest) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.triggers[attemptKey(request)]
}

func (f *Fake) InProgress(request pairlifecycle.QuitRequest) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inProgress[attemptKey(request)]
}

func (f *Fake) PrepareCompletion(request pairlifecycle.QuitRequest, result pairlifecycle.CleanupResult) error {
	return f.publishCompletion(request, result, Prepared)
}

func (f *Fake) CommitCompletion(request pairlifecycle.QuitRequest, result pairlifecycle.CleanupResult) error {
	return f.publishCompletion(request, result, Committed)
}

func (f *Fake) publishCompletion(request pairlifecycle.QuitRequest, result pairlifecycle.CleanupResult, state PublicationState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := attemptKey(request)
	if existing, ok := f.completions[key]; ok && !reflect.DeepEqual(existing.result, result) {
		return errors.New("immutable completion conflicts")
	}
	f.completions[key] = completionState{result: result, state: state}
	if state == Committed {
		delete(f.inProgress, key)
	}
	return nil
}

func (f *Fake) CompletionState(request pairlifecycle.QuitRequest) PublicationState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.completions[attemptKey(request)].state
}

func (f *Fake) Restart() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.active = ""
	f.inProgress = map[string]bool{}
}

func (f *Fake) FailStage(stage pairlifecycle.CleanupStage, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		delete(f.failures, stage)
		return
	}
	f.failures[stage] = err
}

func (f *Fake) EffectiveTrace() []pairlifecycle.CleanupStage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]pairlifecycle.CleanupStage(nil), f.effective...)
}

func (f *Fake) AttemptTrace() []pairlifecycle.CleanupStage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]pairlifecycle.CleanupStage(nil), f.attempts...)
}

func (f *Fake) run(ctx context.Context, stage pairlifecycle.CleanupStage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if f.active == "" || !f.inProgress[f.active] {
		return errors.New("no active lifecycle attempt")
	}
	f.attempts = append(f.attempts, stage)
	if f.done[f.active][stage] {
		return nil
	}
	if err := f.failures[stage]; err != nil {
		return err
	}
	if stage == pairlifecycle.StageSessionQuiescence {
		request := f.requests[f.active].request
		f.sessions[request.Session] = false
	}
	f.done[f.active][stage] = true
	f.effective = append(f.effective, stage)
	return nil
}

func (f *Fake) QuiesceSession(ctx context.Context) error {
	return f.run(ctx, pairlifecycle.StageSessionQuiescence)
}
func (f *Fake) ReapEditors(ctx context.Context) error {
	return f.run(ctx, pairlifecycle.StageEditorReap)
}
func (f *Fake) PreserveScrollback(ctx context.Context, _ pairlifecycle.CleanupIntent) error {
	return f.run(ctx, pairlifecycle.StageScrollbackPreserve)
}
func (f *Fake) CleanupSidecars(ctx context.Context) error {
	return f.run(ctx, pairlifecycle.StageSidecarCleanup)
}
func (f *Fake) CleanupPoller(ctx context.Context) error {
	return f.run(ctx, pairlifecycle.StagePollerCleanup)
}
func (f *Fake) CleanupCmux(ctx context.Context) error {
	return f.run(ctx, pairlifecycle.StageCmuxCleanup)
}
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func attemptKey(request pairlifecycle.QuitRequest) string {
	return fmt.Sprintf("%s/%d", request.Identity.Nonce, request.Attempt)
}

var _ pairlifecycle.QuitLifecycleOps = (*Fake)(nil)

type EffectTrace []string

// RunConformanceScenario drives the durable lifecycle boundaries shared by
// fake and live adapters. Values are semantic labels only: no PID, path, or
// timestamp enters the trace.
func RunConformanceScenario(ctx context.Context, fake *Fake, request pairlifecycle.QuitRequest) (EffectTrace, error) {
	trace := EffectTrace{}
	if err := fake.PrepareRequest(request); err != nil {
		return trace, err
	}
	trace = append(trace, "request:prepared")
	fake.Restart()
	trace = append(trace, "restart:prepared-request")
	if err := fake.CommitRequest(request); err != nil {
		return trace, err
	}
	trace = append(trace, "request:committed")
	fake.Restart()
	trace = append(trace, "restart:committed-request")
	for range 2 {
		if err := fake.DeliverTrigger(request); err != nil {
			return trace, err
		}
		trace = append(trace, "trigger:delivered")
	}
	result := pairlifecycle.RunCleanup(ctx, pairlifecycle.CleanupCouch, fake)
	if result.Outcome != pairlifecycle.CompletionSuccess {
		return trace, fmt.Errorf("cleanup outcome %q", result.Outcome)
	}
	for _, stage := range fake.EffectiveTrace() {
		trace = append(trace, "cleanup:"+string(stage))
	}
	if err := fake.PrepareCompletion(request, result); err != nil {
		return trace, err
	}
	trace = append(trace, "completion:prepared")
	fake.Restart()
	trace = append(trace, "restart:prepared-completion")
	if err := fake.CommitCompletion(request, result); err != nil {
		return trace, err
	}
	trace = append(trace, "completion:committed")
	return trace, nil
}
