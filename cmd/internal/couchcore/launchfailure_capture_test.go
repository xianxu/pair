package couchcore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type captureFailureStore struct {
	records []LaunchFailure
	err     error
}

func (s *captureFailureStore) Path() string                   { return "/diagnostics/failures.json" }
func (s *captureFailureStore) Read() ([]LaunchFailure, error) { return s.records, nil }
func (s *captureFailureStore) Save(f LaunchFailure) error {
	s.records = append(s.records, f)
	return s.err
}

func TestLaunchFailureCaptureBeforeCleanup(t *testing.T) {
	env := newTestEnv(t, "/repo")
	store := &captureFailureStore{}
	env.Couch.LaunchFailures = store
	h, err := env.Runner.Start("/repo", []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("registration refused")
	in := trackedThreadLaunch{Thread: ThreadRecord{Address: ThreadAddress{RepoScope: "scope", Tag: "tag"}, StartingPath: "/repo"}, Nonce: "attempt", Args: StartArgs{Stack: "codex"}}
	got := env.Couch.captureLaunchFailure(in, "registration", h, cause, func() error {
		if len(store.records) != 1 || store.records[0].BeforeExit != nil {
			t.Fatalf("pre-cleanup observation: %+v", store.records)
		}
		env.Runner.SetExited(h.ID(), 23)
		return nil
	})
	if !errors.Is(got, cause) {
		t.Fatal(got)
	}
	if len(store.records) != 2 || store.records[1].BeforeExit != nil || store.records[1].AfterExit == nil || *store.records[1].AfterExit != 23 {
		t.Fatalf("observations: %+v", store.records)
	}
	if strings.Contains(got.Error(), "\n") || !strings.Contains(got.Error(), "couch --show") {
		t.Fatal(got)
	}
}

func TestLaunchFailureCaptureExitedAndPersistenceFailure(t *testing.T) {
	env := newTestEnv(t, "/repo")
	diskErr := errors.New("disk full")
	store := &captureFailureStore{err: diskErr}
	env.Couch.LaunchFailures = store
	h, _ := env.Runner.Start("/repo", []string{"pair"}, nil)
	env.Runner.SetExited(h.ID(), 23)
	in := trackedThreadLaunch{Thread: ThreadRecord{Address: ThreadAddress{RepoScope: "scope", Tag: "tag"}, StartingPath: "/repo"}, Nonce: "attempt"}
	cleaned := false
	got := env.Couch.captureLaunchFailure(in, "registration", h, context.Canceled, func() error { cleaned = true; return nil })
	if !cleaned || !errors.Is(got, context.Canceled) || !errors.Is(got, diskErr) {
		t.Fatal(got)
	}
	if store.records[0].BeforeExit == nil || *store.records[0].BeforeExit != 23 {
		t.Fatal(store.records)
	}
	if !strings.Contains(got.Error(), "exit 23") || !strings.Contains(got.Error(), "diagnostics could not be saved") {
		t.Fatal(got)
	}
}

func TestLaunchFailureCaptureSwitchFinalError(t *testing.T) {
	env := newTestEnv(t, "/repo")
	store := &captureFailureStore{}
	env.Couch.LaunchFailures = store
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) {
		return false, errors.New("observer stopped")
	}
	env.Couch.resumeRegistrationTimeout = time.Millisecond
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	args := []string{"--sandbox", "danger-full-access"}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if err == nil || len(store.records) == 0 {
		t.Fatalf("err=%v records=%v", err, store.records)
	}
	last := store.records[len(store.records)-1]
	if !strings.Contains(last.FinalError, "target launch failed") || last.LaunchError == "" || last.Agent != "codex" {
		t.Fatalf("%+v", last)
	}
	if strings.Contains(err.Error(), "\n") || !strings.Contains(err.Error(), "couch --show") {
		t.Fatal(err)
	}
}

func TestLaunchFailureCaptureDurableAcrossSuccessfulSwitch(t *testing.T) {
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
	args := []string{"--sandbox", "danger-full-access"}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	env.Runner.FailNextStart(errors.New("exec codex: permission denied"))
	_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if err == nil {
		t.Fatal("wanted spawn failure")
	}
	reopened := NewLaunchFailureStore(env.Couch.Namespace)
	records, err := reopened.Read()
	if err != nil || len(records) != 1 {
		t.Fatalf("%v %+v", err, records)
	}
	f := records[0]
	if f.HelperStarted || f.Stage != "spawn" || f.BeforeExit != nil || f.AfterExit != nil || !strings.Contains(f.LaunchError, "permission denied") || !strings.Contains(f.FinalError, "target launch failed") {
		t.Fatalf("%+v", f)
	}
	assertVerifiedParkRestored(t, env.Couch.Threads, source)
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	p, err = env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if err != nil {
		t.Fatal(err)
	}
	records, err = reopened.Read()
	if err != nil || len(records) != 1 || records[0].Nonce != f.Nonce {
		t.Fatalf("success erased prior evidence: %v %+v", err, records)
	}
}

func TestLaunchFailureCaptureOutputAndExitBeforeRegistration(t *testing.T) {
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) {
		return false, errors.New("not registered")
	}
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Runner.Child(id).terminal.Feed([]byte("codex: missing executable\n"))
		env.Runner.SetExited(id, 23)
		return nil
	}
	args := []string{}
	p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &args)
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Address: source.Address, Agent: "codex", Argv: args, AcceptedFingerprint: p.Fingerprint})
	if err == nil || !strings.Contains(err.Error(), "exit 23") {
		t.Fatal(err)
	}
	records, err := NewLaunchFailureStore(env.Couch.Namespace).Read()
	if err != nil || len(records) != 1 {
		t.Fatalf("%v %+v", err, records)
	}
	f := records[0]
	if !f.HelperStarted || f.BeforeExit == nil || *f.BeforeExit != 23 || !strings.Contains(string(f.Output), "missing executable") || f.Stream != "pty stdout/stderr" {
		t.Fatalf("%+v", f)
	}
}

func TestLaunchFailureOperatorLineOneAction(t *testing.T) {
	f := LaunchFailure{Stage: "spawn", StartingPath: "/slot", LaunchError: "fork failed\nretry and resume old\x1b[2J"}
	err := &launchFailureError{failure: f, cause: errors.New(f.LaunchError)}
	got := err.Error()
	if strings.ContainsAny(got, "\n\r\x1b") || strings.Count(got, "couch --show") != 1 || strings.Contains(got, "retry") || !strings.Contains(got, "fork failed") {
		t.Fatal(got)
	}
}

func TestLaunchFailureCaptureCorrelatesAttemptAndStore(t *testing.T) {
	for _, writeFails := range []bool{false, true} {
		t.Run(fmt.Sprint(writeFails), func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			store := &captureFailureStore{}
			diskErr := errors.New("disk full")
			if writeFails {
				store.err = diskErr
			}
			env.Couch.LaunchFailures = store
			cause := errors.New("spawn refused")
			in := trackedThreadLaunch{Thread: ThreadRecord{Address: ThreadAddress{RepoScope: "scope", Tag: "tag"}, StartingPath: "/repo"}, Nonce: "exact-attempt-123"}
			got := env.Couch.captureLaunchFailure(in, "spawn", nil, cause, nil)
			if !strings.Contains(got.Error(), in.Nonce) || !strings.Contains(got.Error(), store.Path()) {
				t.Fatalf("missing correlation: %v", got)
			}
			if !errors.Is(got, cause) {
				t.Fatal("lost original cause")
			}
			if writeFails && !errors.Is(got, diskErr) {
				t.Fatal("lost storage cause")
			}
			if strings.Contains(got.Error(), "\n") || strings.Count(got.Error(), "inspect with") != 1 {
				t.Fatal(got)
			}
		})
	}
}

func TestLaunchFailureCapturePartialStartEvidence(t *testing.T) {
	env := newTestEnv(t, "/repo")
	cause := errors.New("ack reader close failed")
	before, after := 23, 23
	setup := &blockedStartError{cause: cause, HelperStarted: true, BeforeExit: &before, AfterExit: &after, Stream: "stderr", Output: []byte("helper failed before registration"), CleanupError: "runner cleanup failed"}
	in := trackedThreadLaunch{Thread: ThreadRecord{Address: ThreadAddress{RepoScope: "scope", Tag: "tag"}, StartingPath: "/repo"}, Nonce: "partial-start"}
	got := env.Couch.captureLaunchFailure(in, "spawn", nil, setup, func() error { return errors.New("rollback failed") })
	if !errors.Is(got, cause) {
		t.Fatal(got)
	}
	records, err := env.Couch.LaunchFailures.Read()
	if err != nil || len(records) != 1 {
		t.Fatalf("%v %+v", err, records)
	}
	f := records[0]
	if !f.HelperStarted || f.BeforeExit == nil || *f.BeforeExit != 23 || f.AfterExit == nil || *f.AfterExit != 23 || string(f.Output) != "helper failed before registration" || !strings.Contains(f.CleanupError, "runner cleanup failed") || !strings.Contains(f.CleanupError, "rollback failed") {
		t.Fatalf("lost partial-start evidence: %+v", f)
	}
}
