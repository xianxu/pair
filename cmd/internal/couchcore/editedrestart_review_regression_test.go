package couchcore

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// After one exact replacement observation, even an occupied fallback claim
// observes absence. This prevents a second replacement observation from hiding
// an incorrectly emitted fresh fallback by refusing that fallback incidentally.
type onceReplacingEditedRestartArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	a, b     string
	replaced bool
}

func (f *onceReplacingEditedRestartArtifacts) BindTranscript(_ context.Context, thread ThreadRecord) (TranscriptBind, error) {
	if !f.replaced && hasOccupiedIncarnation(thread) {
		f.replaced = true
		return TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: f.b}}, nil
	}
	if f.replaced {
		return TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingUnbound}}, refuseBinding(ResumeBindingUnbound)
	}
	return TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: f.a}}, nil
}

func TestSwitchAgentEditedRestartObservedReplacementThenAbsenceIsTerminal(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: agent, Argv: []string{}})
			env.Artifacts.SetNativeBinding(source.Address, agent, sessioninventory.BindingEstablished, bindID(3))
			artifacts := &onceReplacingEditedRestartArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, a: bindID(3), b: bindID(5)}
			env.Couch.Artifacts = artifacts
			argv := []string{}
			p, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, agent, &argv)
			if err != nil {
				t.Fatal(err)
			}
			env.Couch.FreshRegistration = func(context.Context, ThreadAddress, string, string) (bool, error) { return true, nil }
			env.Runner.AfterAcknowledge = func(id string) error {
				env.Artifacts.SetPairSession(source.Address, continuationChildSession(t, env.Runner, id), true)
				env.Artifacts.SetPaneSidecar(source.Address, agent)
				return nil
			}
			got, err := env.Couch.SwitchAgent(context.Background(), SwitchAgentRequest{Restart: p.AcceptAgentSwitch(false).restart, Address: source.Address, Agent: agent, Argv: argv, AcceptedFingerprint: p.Fingerprint})
			if !artifacts.replaced {
				t.Fatal("post-claim replacement not observed")
			}
			if !errors.Is(err, ErrSwitchResolutionChanged) || got.Outcome == SwitchStarted || countStarts(env.Runner) != 0 {
				t.Fatalf("forgot authoritative A-to-B replacement after B disappeared: outcome=%s starts=%d err=%v", got.Outcome, countStarts(env.Runner), err)
			}
		})
	}
}
