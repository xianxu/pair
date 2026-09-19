package launcher

import (
	"errors"
	"testing"
)

// #288: once a create's pane has not appeared within the bound, one liveness
// snapshot decides. Only an answer that shows no running session is death; a
// failed question is not an answer.
func TestJudgeUnborn(t *testing.T) {
	const ours = "📁work-bugfix"
	other := Session{Name: "📁work-other", State: SessionLive}
	for _, tc := range []struct {
		name     string
		sessions []Session
		err      error
		want     birthVerdict
	}{
		{"the probe failed", nil, errors.New("zellij list-sessions: timeout"), birthUnknown},
		{"zellij's empty inventory", nil, nil, birthDead},
		{"only other sessions", []Session{other}, nil, birthDead},
		{"ours exited", []Session{other, {Name: ours, State: SessionExited}}, nil, birthDead},
		{"ours live", []Session{{Name: ours, State: SessionLive}}, nil, birthAlive},
		{"ours attached", []Session{{Name: ours, State: SessionAttached}}, nil, birthAlive},
		{"ours detached", []Session{{Name: ours, State: SessionDetached}}, nil, birthAlive},
	} {
		if got := judgeUnborn(tc.sessions, tc.err, ours); got != tc.want {
			t.Errorf("%s: judgeUnborn = %v, want %v", tc.name, got, tc.want)
		}
	}
}
