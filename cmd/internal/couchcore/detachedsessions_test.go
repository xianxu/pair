package couchcore

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestProjectDetachedSessions(t *testing.T) {
	one := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000001"}
	two := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000002"}

	tests := []struct {
		name     string
		bindings []SessionNameBinding
		sessions []launcher.Session
		want     []DetachedSessionObservation
	}{
		{
			name:     "a live session with no client is detached",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionDetached}},
			want:     []DetachedSessionObservation{{Address: one, SessionName: "pair-one"}},
		},
		{
			name:     "an attached session is not detached",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionAttached}},
		},
		{
			name:     "an exited session is not detached",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionExited}},
		},
		{
			name:     "a bound name with no session at all yields nothing",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{{Name: "pair-other", State: launcher.SessionDetached}},
		},
		{
			name:     "an unbound session is not attributed to any thread",
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionDetached}},
		},
		{
			name: "each bound address is judged independently",
			bindings: []SessionNameBinding{
				{Address: one, SessionName: "pair-one"},
				{Address: two, SessionName: "pair-two"},
			},
			sessions: []launcher.Session{
				{Name: "pair-one", State: launcher.SessionAttached},
				{Name: "pair-two", State: launcher.SessionDetached},
			},
			want: []DetachedSessionObservation{{Address: two, SessionName: "pair-two"}},
		},
		{
			name:     "an empty session name is never a binding",
			bindings: []SessionNameBinding{{Address: one, SessionName: ""}},
			sessions: []launcher.Session{{Name: "", State: launcher.SessionDetached}},
		},
		{
			// Fail closed: two rows claiming one name cannot both be that
			// session, and couch cannot tell which is right.
			name: "an ambiguous session name yields nothing for either address",
			bindings: []SessionNameBinding{
				{Address: one, SessionName: "pair-shared"},
				{Address: two, SessionName: "pair-shared"},
			},
			sessions: []launcher.Session{{Name: "pair-shared", State: launcher.SessionDetached}},
		},
		{
			// Two zellij rows with one name is a state couch cannot resolve.
			name:     "a duplicated session row yields nothing",
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one"}},
			sessions: []launcher.Session{
				{Name: "pair-one", State: launcher.SessionDetached},
				{Name: "pair-one", State: launcher.SessionAttached},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ProjectDetachedSessions(test.bindings, test.sessions)
			if len(got) != len(test.want) {
				t.Fatalf("ProjectDetachedSessions() = %+v, want %+v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("ProjectDetachedSessions() = %+v, want %+v", got, test.want)
				}
			}
		})
	}
}
