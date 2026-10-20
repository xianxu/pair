package couchcore

import (
	"context"
	"errors"
	"testing"
)

func TestCompleteFailureAddress(t *testing.T) {
	exact := ThreadAddress{RepoScope: "scope", Tag: "exact"}
	other := ThreadAddress{RepoScope: "other", Tag: "different"}
	for _, tc := range []struct {
		name                      string
		preferred, fallback, want ThreadAddress
	}{
		{"preferred exact", exact, ThreadAddress{}, exact},
		{"different complete operation target wins", other, exact, other},
		{"scope cannot erase", ThreadAddress{RepoScope: "other"}, exact, exact},
		{"tag cannot erase", ThreadAddress{Tag: "different"}, exact, exact},
		{"no hybrid", ThreadAddress{RepoScope: "scope"}, ThreadAddress{Tag: "exact"}, ThreadAddress{}},
		{"no reverse hybrid", ThreadAddress{Tag: "exact"}, ThreadAddress{RepoScope: "scope"}, ThreadAddress{}},
		{"no evidence", ThreadAddress{}, ThreadAddress{}, ThreadAddress{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := completeFailureAddress(tc.preferred, tc.fallback); got != tc.want {
				t.Fatalf("got=%+v want=%+v", got, tc.want)
			}
		})
	}
}

func FuzzCompleteFailureAddressNeverCombinesIdentity(f *testing.F) {
	f.Add("scope", "", "", "exact")
	f.Add("", "different", "scope", "exact")
	f.Add("other", "different", "scope", "exact")
	f.Fuzz(func(t *testing.T, scope, tag, fallbackScope, fallbackTag string) {
		preferred := ThreadAddress{RepoScope: scope, Tag: ThreadTag(tag)}
		fallback := ThreadAddress{RepoScope: fallbackScope, Tag: ThreadTag(fallbackTag)}
		got := completeFailureAddress(preferred, fallback)
		if got != (ThreadAddress{}) && (got.RepoScope == "" || got.Tag == "" || got != preferred && got != fallback) {
			t.Fatalf("manufactured partial or hybrid identity: got=%+v preferred=%+v fallback=%+v", got, preferred, fallback)
		}
		if scope != "" && tag != "" && got != preferred {
			t.Fatalf("lost complete preferred identity: got=%+v preferred=%+v", got, preferred)
		}
		if fallbackScope != "" && fallbackTag != "" && got == (ThreadAddress{}) {
			t.Fatalf("lost complete fallback identity: fallback=%+v", fallback)
		}
	})
}

func TestLifecycleFailureExplicitPartialTargetPersistsRefusal(t *testing.T) {
	for _, target := range []LifecycleFailureTarget{
		{Address: ThreadAddress{RepoScope: "scope"}, Target: "pair:5"},
		{Address: ThreadAddress{Tag: "tag"}, StartingPath: "/repo"},
		{Address: ThreadAddress{Tag: "tag"}},
	} {
		r, store := attemptRecorderFixture()
		call := OperationCall{Name: "resume", FailureTarget: &target}
		attempt, _ := r.Begin(call.Name, lifecycleFailureTargetForCall(call))
		attempt.Finish(nil, errors.New("resolution refused"))
		history, err := store.Read()
		if err != nil || len(history) != 1 || history[0].Address != (ThreadAddress{}) || attempt.DiagnosticWarning() != "" {
			t.Fatalf("partial explicit target was not persisted: target=%+v history=%+v err=%v warning=%q", target, history, err, attempt.DiagnosticWarning())
		}
		if target.Target != "" && history[0].Target != target.Target || history[0].StartingPath != target.StartingPath {
			t.Fatalf("lost reference/path evidence: target=%+v history=%+v", target, history)
		}
	}
}

func TestLifecycleFailureScopeOnlyOuterCallRetainsLaunchEvidence(t *testing.T) {
	env := newTestEnv(t, "/repo")
	r, store := attemptRecorderFixture()
	call := OperationCall{Name: "resume", Args: map[string]string{"repo-scope": "scope", "ref": "pair:5"}}
	attempt, _ := r.Begin(call.Name, lifecycleFailureTargetForCall(call))
	child, err := env.Runner.Start("/repo", []string{"pair"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	h := &erasedLaunchOutput{Handle: child, raw: []byte("resume refused before registering")}
	env.Runner.SetExited(h.ID(), 23)
	address := ThreadAddress{RepoScope: "scope", Tag: "exact"}
	in := trackedThreadLaunch{Context: WithOperationAttempt(context.Background(), attempt), Nonce: "start-original", Thread: ThreadRecord{Address: address, StartingPath: "/repo"}}
	cause := errors.New("registration was not confirmed")
	captured := env.Couch.captureLaunchFailure(in, "registration", h, cause, nil)
	_, result := attempt.Finish(nil, captured)
	history, err := store.Read()
	if err != nil || len(history) != 1 || attempt.DiagnosticWarning() != "" {
		t.Fatalf("nested launch evidence was not persisted: history=%+v read=%v result=%v warning=%q", history, err, result, attempt.DiagnosticWarning())
	}
	f := history[0]
	if f.Address != address || f.LaunchNonce != in.Nonce || string(f.Output) != string(h.raw) || f.BeforeExit == nil || *f.BeforeExit != 23 || f.AfterExit == nil || *f.AfterExit != 23 || !errors.Is(result, cause) || len(store.Writes) != 1 {
		t.Fatalf("lost exact launch evidence: %+v result=%v", f, result)
	}
}

func TestLifecycleFailureObservePartialCallRetainsCompleteAddress(t *testing.T) {
	for _, args := range []map[string]string{{"repo-scope": "other"}, {"tag": "other"}} {
		r, store := attemptRecorderFixture()
		address := ThreadAddress{RepoScope: "scope", Tag: "exact"}
		attempt, _ := r.Begin("resume", LifecycleFailureTarget{Address: address})
		attempt.ObserveCall(OperationCall{Name: "resume", Args: args})
		attempt.Finish(nil, errors.New("preparation refused"))
		history, err := store.Read()
		if err != nil || len(history) != 1 || history[0].Address != address || attempt.DiagnosticWarning() != "" {
			t.Fatalf("partial call erased frozen address: args=%v history=%+v err=%v warning=%q", args, history, err, attempt.DiagnosticWarning())
		}
	}
}

func TestLifecycleFailureUnresolvedPartialAddressPersistsRefusal(t *testing.T) {
	for _, args := range []map[string]string{
		{"repo-scope": "scope", "ref": "pair:5"},
		{"tag": "tag", "ref": "pair:5"},
		{"repo-scope": "scope"},
		{"tag": "tag"},
	} {
		r, store := attemptRecorderFixture()
		call := OperationCall{Name: "resume", Args: args}
		attempt, _ := r.Begin(call.Name, lifecycleFailureTargetForCall(call))
		cause := &ResumeRefusal{Code: ResumeStarting, Diagnostic: "source is still starting"}
		_, result := attempt.Finish(nil, cause)
		history, err := store.Read()
		want := args["ref"]
		if want == "" {
			want = "unresolved resume"
		}
		if err != nil || len(history) != 1 || history[0].Address != (ThreadAddress{}) || history[0].Target != want || !errors.Is(result, cause) || attempt.DiagnosticWarning() != "" {
			t.Fatalf("unresolved refusal lost: args=%v history=%+v err=%v result=%v warning=%q", args, history, err, result, attempt.DiagnosticWarning())
		}
	}
}
