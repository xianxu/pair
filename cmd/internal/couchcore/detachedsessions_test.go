package couchcore

import (
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
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
			bindings: []SessionNameBinding{{Address: one, SessionName: "pair-one", Agent: "claude", NativeID: "native-1"}},
			sessions: []launcher.Session{{Name: "pair-one", State: launcher.SessionDetached}},
			want:     []DetachedSessionObservation{{Address: one, SessionName: "pair-one", Agent: "claude", NativeID: "native-1"}},
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
				{Address: one, SessionName: "pair-one", Agent: "claude", NativeID: "native-1"},
				{Address: two, SessionName: "pair-two", Agent: "claude", NativeID: "native-2"},
			},
			sessions: []launcher.Session{
				{Name: "pair-one", State: launcher.SessionAttached},
				{Name: "pair-two", State: launcher.SessionDetached},
			},
			want: []DetachedSessionObservation{{Address: two, SessionName: "pair-two", Agent: "claude", NativeID: "native-2"}},
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

// The refresh's detached observation costs 2 + N zellij subprocesses
// (list-sessions twice, plus one list-clients per pair session), so what keeps
// it proportional is asking ONLY about records that could be detached. This
// pins that bound directly, because a benchmark of the pure reducer cannot see
// it -- the query lives on the refresh worker, not the keystroke path.
func TestActionableInventoryAsksOnlyAboutDetachCandidates(t *testing.T) {
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
	profile := &LaunchProfile{Agent: "claude", Argv: []string{}}

	newRecord := func(tag string, mutate func(*ThreadRecord)) ThreadAddress {
		t.Helper()
		seed := validThreadRecord(t)
		seed.Address.Tag = ThreadTag(tag)
		seed.StartingPath, seed.WorkingPath = ns.Dir(), ns.Dir()
		created, err := store.CreateThread(seed)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
			next.Reservation = false
			next.LatestLaunchProfile = profile
			mutate(next)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return created.Address
	}

	candidate := newRecord("couch-0000000000000001", func(*ThreadRecord) {})
	artifactsBinding := candidate
	// Occupied: it has an incarnation, so it cannot be detached.
	newRecord("couch-0000000000000002", func(r *ThreadRecord) {
		r.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 5, Identity: "id-5", StartedAt: time.Unix(2, 0).UTC()}}
	})
	// No profile: nothing to reattach with, so asking about it is wasted IO.
	newRecord("couch-0000000000000003", func(r *ThreadRecord) { r.LatestLaunchProfile = nil })

	artifacts := NewFakeThreadArtifactCollisionChecker()
	// Candidates must also clear the native-binding gate; without it the
	// inventory skips them before ever asking about sessions.
	artifacts.SetNativeBinding(artifactsBinding, "claude", sessioninventory.BindingEstablished, "native-root-1")
	var asked [][]ThreadAddress
	artifacts.DetachedSessionsHook = func(addresses []ThreadAddress) error {
		asked = append(asked, addresses)
		return nil
	}
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}

	if _, err := couch.ActionableThreadInventoryContext(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 {
		t.Fatalf("DetachedSessions called %d times, want exactly one batched query", len(asked))
	}
	if len(asked[0]) != 1 || asked[0][0] != candidate {
		t.Fatalf("asked about %+v, want only the candidate %+v", asked[0], candidate)
	}
}

// With no candidates at all, the refresh must not spawn the query.
func TestActionableInventorySkipsTheQueryWithNoCandidates(t *testing.T) {
	ns := testCouchNamespace(t)
	store := NewThreadStore(ns)
	seed := validThreadRecord(t)
	seed.StartingPath, seed.WorkingPath = ns.Dir(), ns.Dir()
	created, err := store.CreateThread(seed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateExistingThread(created.Address, created.Revision, func(next *ThreadRecord) error {
		next.Reservation = false
		next.Incarnations = []ThreadIncarnation{{State: IncarnationLive, PID: 5, Identity: "id-5", StartedAt: time.Unix(2, 0).UTC()}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	artifacts := NewFakeThreadArtifactCollisionChecker()
	called := 0
	artifacts.DetachedSessionsHook = func([]ThreadAddress) error {
		called++
		return nil
	}
	couch := &Couch{Threads: store, Artifacts: artifacts, Path: NewFakePathOps(nil)}
	if _, err := couch.ActionableThreadInventoryContext(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if called != 0 {
		t.Fatalf("DetachedSessions was called %d times with no candidates", called)
	}
}

// The pure function must emit observations the pure PROJECTOR accepts.
//
// It used to emit {Address, SessionName} only, which
// ProjectActionableThreads -- once it started enforcing the resume proof --
// always rejected. Production worked anyway because the IO shell patched Agent
// and NativeID onto the answer afterwards, which meant this function's own
// tests asserted a shape nothing downstream would take. Composing the two pure
// functions is the guard.
func TestProjectDetachedSessionsEmitsObservationsTheProjectorAccepts(t *testing.T) {
	address := ThreadAddress{RepoScope: "scope-a", Tag: "couch-0000000000000001"}
	observed := ProjectDetachedSessions(
		[]SessionNameBinding{{Address: address, SessionName: "pair-one", Agent: "claude", NativeID: "native-1"}},
		[]launcher.Session{{Name: "pair-one", State: launcher.SessionDetached}},
	)
	if len(observed) != 1 {
		t.Fatalf("ProjectDetachedSessions() = %+v, want one observation", observed)
	}

	record := ThreadRecord{
		SchemaVersion: ThreadSchemaVersion, Address: address,
		StartingPath: "/repo", WorkingPath: "/repo",
		CreatedAt: time.Unix(1, 0).UTC(), Revision: 1,
		LatestLaunchProfile: &LaunchProfile{Agent: "claude", Argv: []string{}},
	}
	rows := actionableRows([]ThreadRecord{record}, nil, nil, observed)
	if len(rows) != 1 || rows[0].State != ThreadDetached {
		t.Fatalf("rows = %+v, want the projector to accept its own upstream's output", rows)
	}
}
