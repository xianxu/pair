package couchcore

import (
	"context"
	"errors"
	"testing"
)

func TestLifecycleFailureOwnedLaunchCaptureDefersPublication(t *testing.T) {
	env := newTestEnv(t, "/repo")
	r, store := attemptRecorderFixture()
	env.Couch.LaunchFailures = store
	attempt, err := r.Begin("switch-agent", LifecycleFailureTarget{StartingPath: "/repo"})
	if err != nil {
		t.Fatal(err)
	}
	in := trackedThreadLaunch{Context: WithOperationAttempt(context.Background(), attempt), Nonce: "start-original", Thread: ThreadRecord{Address: ThreadAddress{RepoScope: "scope", Tag: "exact"}, StartingPath: "/repo"}}
	cause := errors.New("registration was not confirmed")
	got := env.Couch.captureLaunchFailure(in, "registration", nil, cause, nil)
	got = env.Couch.finishSwitchFailure(in, got)
	if len(store.Writes) != 0 {
		t.Fatal("nested capture helpers published before terminal completion")
	}
	_, got = attempt.Finish(nil, got)
	if len(store.Writes) != 1 || store.Writes[0].LaunchNonce != "start-original" || store.Writes[0].Nonce != attempt.Nonce() || !errors.Is(got, cause) {
		t.Fatalf("lost single owner or launch correlation: %+v %v", store.Writes, got)
	}
}
