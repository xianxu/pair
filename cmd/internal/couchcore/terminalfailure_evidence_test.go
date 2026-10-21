package couchcore

import (
	"errors"
	"testing"
)

type terminalEvidenceError struct{ evidence LifecycleFailure }

func (e *terminalEvidenceError) Error() string                             { return "attach rejected" }
func (e *terminalEvidenceError) TerminalFailureEvidence() LifecycleFailure { return e.evidence }

func TestOperationAttemptAdoptsTerminalFailureWithoutRebinding(t *testing.T) {
	r, store := attemptRecorderFixture()
	address := ThreadAddress{RepoScope: "scope", Tag: "exact"}
	a, _ := r.Begin("reboot", LifecycleFailureTarget{Target: "fixture:3", Address: address})
	code := 23
	cause := &terminalEvidenceError{LifecycleFailure{Nonce: "foreign", Operation: "foreign", Address: ThreadAddress{RepoScope: "foreign", Tag: "foreign"}, Stage: "attach-precommit", Stream: "pty stdout/stderr", Output: []byte("last bytes"), BeforeExit: &code, LaunchError: "endpoint=real input_ended=true termination_cause=unavailable"}}
	_, err := a.Finish(nil, errors.Join(errors.New("outer"), cause))
	history, readErr := store.Read()
	if readErr != nil || len(history) != 1 {
		t.Fatalf("history=%+v error=%v", history, readErr)
	}
	f := history[0]
	if !errors.Is(err, cause) || f.Nonce != a.Nonce() || f.Operation != "reboot" || f.Address != address || f.Target != "fixture:3" || f.LaunchNonce != "" {
		t.Fatalf("terminal evidence changed authority: %+v %v", f, err)
	}
	if f.Stage != "attach-precommit" || string(f.Output) != "last bytes" || f.BeforeExit == nil || *f.BeforeExit != 23 || f.AfterExit != nil || f.LaunchError != cause.evidence.LaunchError {
		t.Fatalf("lost terminal evidence: %+v", f)
	}
	cause.evidence.Output[0] = 'X'
	code = 99
	if string(f.Output) != "last bytes" || *f.BeforeExit != 23 {
		t.Fatal("record aliases evidence producer")
	}
}
