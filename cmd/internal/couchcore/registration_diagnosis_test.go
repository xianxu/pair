package couchcore

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// A registration deadline must name WHICH half of startup did not finish (#215).
//
// The bare error -- `await Pair registration {RepoScope:… Tag:…}: context
// deadline exceeded` -- says neither what was waited for nor whether Pair
// started, and diagnosing it cost hours. The discriminator is an observation
// couch already had and did not make.
func TestARegistrationTimeoutSaysWhetherPairStarted(t *testing.T) {
	address := ThreadAddress{RepoScope: "e108517d46ab4575", Tag: "couch-7bd0c2986975082c"}
	const budget = 15 * time.Second

	for _, tc := range []struct {
		name    string
		present bool
		session string
		want    []string
		absent  string
	}{
		{
			name: "pair started but never registered", present: true, session: "📁pair-couch-26",
			want:   []string{"15s", "📁pair-couch-26", "STARTED", "Pair's startup"},
			absent: "never started",
		},
		{
			// The fake reports an absent binding as an ERROR, exactly as
			// production does -- a missing index entry means Pair never recorded
			// a name. That is an unreadable observation, so the diagnosis must
			// stop short of a verdict rather than assert one (ARCH-SECURE).
			name: "pair's binding cannot be read", present: false, session: "",
			want:   []string{"15s", "could NOT determine", "not a verdict", "zellij list-sessions"},
			absent: "IS live",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := NewFakeThreadArtifactCollisionChecker()
			if tc.present {
				fake.SetPairSession(address, tc.session, true)
			}
			c := &Couch{Artifacts: fake}

			got := c.diagnoseRegistrationFailure(context.DeadlineExceeded, address, budget)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("diagnosis %q does not mention %q", got, want)
				}
			}
			if strings.Contains(got, tc.absent) {
				t.Errorf("diagnosis %q claims %q, which is the other case", got, tc.absent)
			}
		})
	}

	// No session observer installed at all -- the branch nothing covered.
	bare := &Couch{}
	if got := bare.diagnoseRegistrationFailure(context.DeadlineExceeded, address, budget); got == "" {
		t.Error("a console with no session observer produced no diagnosis at all; the " +
			"operator still needs to know the wait happened and why nothing could be said")
	} else if !strings.Contains(got, "no session observer") {
		t.Errorf("diagnosis %q does not say WHY it cannot answer", got)
	}

	// A non-timeout error already says what it is; do not append noise, and do
	// not spend the observation.
	fake := NewFakeThreadArtifactCollisionChecker()
	c := &Couch{Artifacts: fake}
	if got := c.diagnoseRegistrationFailure(errors.New("evidence is unknown"), address, budget); got != "" {
		t.Errorf("non-timeout error got a startup diagnosis it did not need: %q", got)
	}
}
