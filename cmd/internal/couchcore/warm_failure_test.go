package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// pair#230, at the seam: the six failure routes that reach quiescePostAckStart,
// each driven for a WARM reattach and for an OWNING start.
//
// The rule under test is one sentence -- a start never ends a session it did
// not create -- and the table is its class enumeration. A warm reattach
// attached to a session that was already running someone's agent; deleting it
// destroys exactly what the reattach existed to preserve.
//
// Every row's injection is at the fake seam, and none of them reaches a real
// zellij: FakeThreadArtifactCollisionChecker.Quiesce models the deletion, so
// "the session survived" is a claim these assertions can actually falsify.

// warmFailureRoute is one post-acknowledgement failure route.
type warmFailureRoute struct {
	name string
	// inject arms the failure. It runs after the thread exists and its session
	// state is set up. cancel cancels the context the resume runs under, so the
	// cancellation route can fire it from inside the launch.
	inject func(t *testing.T, env *testEnv, address ThreadAddress, cancel context.CancelFunc)
	// sessionSurvives is false for the one route whose session dies as the
	// cause of the failure rather than as a consequence of cleanup.
	sessionSurvives bool
	// abortAfterStart drives route 6, which fails only once the console has the
	// child and cannot attach it.
	abortAfterStart bool
}

func warmFailureRoutes() []warmFailureRoute {
	return []warmFailureRoute{
		{
			name: "1-acknowledge-failed",
			inject: func(_ *testing.T, env *testEnv, _ ThreadAddress, _ context.CancelFunc) {
				env.Runner.BeforeAcknowledge = func(string) error { return errors.New("ack transport closed") }
			},
			sessionSurvives: true,
		},
		{
			name: "2-cancelled-after-acknowledge",
			inject: func(_ *testing.T, env *testEnv, _ ThreadAddress, cancel context.CancelFunc) {
				env.Runner.AfterAcknowledge = func(string) error { cancel(); return nil }
			},
			sessionSurvives: true,
		},
		{
			// The session itself dies mid-reattach, so registration never
			// resolves. This is the one warm route that CAN reach the
			// registration timeout: a warm reattach's registration check is
			// satisfied by the session that already exists, so while the
			// session is there it returns at once.
			name: "3-registration-timed-out",
			inject: func(_ *testing.T, env *testEnv, address ThreadAddress, _ context.CancelFunc) {
				env.Runner.AfterAcknowledge = func(string) error {
					env.Artifacts.SetDetachedSession(address, "")
					env.Artifacts.SetPairSession(address, "pair-"+string(address.Tag), false)
					return nil
				}
			},
			sessionSurvives: false,
		},
		{
			name: "4-promotion-conflicted",
			inject: func(t *testing.T, env *testEnv, address ThreadAddress, _ context.CancelFunc) {
				env.Runner.AfterAcknowledge = func(string) error {
					current, err := env.Couch.Threads.GetThread(address)
					if err != nil {
						return err
					}
					_, err = env.Couch.Threads.UpdateExistingThread(address, current.Revision, func(next *ThreadRecord) error {
						next.Description = "changed under the start"
						return nil
					})
					return err
				}
			},
			sessionSurvives: true,
		},
		{
			name: "5-registry-persist-failed",
			inject: func(t *testing.T, env *testEnv, _ ThreadAddress, _ context.CancelFunc) {
				blocked := filepath.Join(t.TempDir(), "not-a-directory")
				if err := os.WriteFile(blocked, []byte("occupied"), 0o600); err != nil {
					t.Fatal(err)
				}
				env.Couch.Store = NewStore(blocked)
			},
			sessionSurvives: true,
		},
		{
			name:            "6-console-could-not-attach",
			inject:          func(*testing.T, *testEnv, ThreadAddress, context.CancelFunc) {},
			sessionSurvives: true,
			abortAfterStart: true,
		},
	}
}

// The headline: no warm route may delete the session it was reattaching.
func TestAFailedWarmReattachKeepsItsSession(t *testing.T) {
	for _, route := range warmFailureRoutes() {
		t.Run(route.name, func(t *testing.T) {
			env, address := warmDetachedThread(t)
			before, err := env.Couch.Threads.GetThread(address)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			route.inject(t, env, address, cancel)

			record, handle, err := env.Couch.ResumeContext(ctx, address)
			if route.abortAfterStart {
				if err != nil {
					t.Fatalf("warm reattach refused before the console saw it: %v", err)
				}
				err = env.Couch.AbortStarted(StartResult{Record: record, Handle: handle}, errors.New("attach terminal has already exited"))
			}
			if err == nil {
				t.Fatal("the route did not fail; the injection no longer reaches it")
			}

			if quiesced := env.Artifacts.Quiesces(); containsAddress(quiesced, address) {
				t.Fatalf("a failed WARM reattach quiesced %+v -- that deletes the session and kills the agent it was reattaching to", address)
			}
			if handle != nil && handle.Alive() {
				t.Fatal("the reattach's own helper was left running")
			}

			thread, getErr := env.Couch.Threads.GetThread(address)
			if getErr != nil {
				t.Fatal(getErr)
			}
			if len(thread.Incarnations) != 0 {
				t.Fatalf("thread = %+v, want no incarnation: the failed start's own write is undone", thread)
			}
			if thread.LastActiveAt != before.LastActiveAt {
				t.Fatalf("LastActiveAt = %v, want %v unchanged: a failed reattach is not activity",
					thread.LastActiveAt, before.LastActiveAt)
			}

			// The proof that matters to the operator: what the switcher will
			// say about this thread on the next refresh.
			rows, err := env.Couch.ActionableThreadInventoryContext(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			row, found := rowFor(rows, address)
			if !found {
				t.Fatalf("thread %+v vanished from the inventory", address)
			}
			if route.sessionSurvives {
				if row.State != ThreadDetached {
					t.Fatalf("row = %+v, want ThreadDetached: its session survived, so it is reattachable again", row)
				}
			} else if row.State == ThreadDetached {
				t.Fatalf("row = %+v claims detached, but this route's session died", row)
			}
		})
	}
}

// The other half of the class: an OWNING start still ends the session it
// created. pair#230 changes nothing here, and this is what proves it.
func TestAFailedOwningStartStillQuiescesItsSession(t *testing.T) {
	for _, route := range warmFailureRoutes() {
		if route.name == "3-registration-timed-out" {
			// Its injection is warm-specific: it kills a pre-existing session.
			continue
		}
		t.Run(route.name, func(t *testing.T) {
			env, parked := coldParkedThread(t)
			address := parked.Address
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			route.inject(t, env, address, cancel)

			record, handle, err := env.Couch.ResumeContext(ctx, address)
			if route.abortAfterStart {
				if err != nil {
					t.Fatalf("cold resume refused before the console saw it: %v", err)
				}
				err = env.Couch.AbortStarted(StartResult{Record: record, Handle: handle}, errors.New("attach terminal has already exited"))
			}
			if err == nil {
				t.Fatal("the route did not fail; the injection no longer reaches it")
			}
			if quiesced := env.Artifacts.Quiesces(); !containsAddress(quiesced, address) {
				t.Fatalf("a failed COLD resume did not quiesce %+v; it created that session and must not leave it behind", address)
			}
		})
	}
}

// warmDetachedThread is a thread whose zellij session is alive with no client --
// the state alt+d leaves behind, and the only state a warm reattach resumes
// from.
//
// No verified park: the surviving session is the sole RESUME authority, and
// ResumeContext never consults the binding resolver on this path. The native
// binding is set all the same, because the INVENTORY resolves one itself and
// its detached proof requires a native id -- without it the switcher would
// read binding-lost, which is a different row than the one this test is about.
func warmDetachedThread(t *testing.T) (*testEnv, ThreadAddress) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	profile := LaunchProfile{Agent: "claude", Argv: []string{}}
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = "/repo", "/repo"
	record.Reservation = false
	record.LatestLaunchProfile = &profile
	created, err := env.Couch.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	name := "pair-" + string(created.Address.Tag)
	env.Artifacts.SetDetachedSession(created.Address, name)
	env.Artifacts.SetPairSession(created.Address, name, true)
	env.Artifacts.SetNativeBinding(created.Address, "claude", sessioninventory.BindingEstablished, "native-warm-1")
	env.Couch.resumeRegistrationTimeout = 200 * time.Millisecond
	return env, created.Address
}

// coldParkedThread is the owning counterpart: a verified park, whose resume
// CREATES a session and so owns cleaning it up.
func coldParkedThread(t *testing.T) (*testEnv, ThreadRecord) {
	t.Helper()
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "codex", Argv: []string{"--saved"}})
	env.Artifacts.SetNativeBinding(parked.Address, "codex", sessioninventory.BindingEstablished, "native-root-1")
	env.Runner.AfterBlockedStart = func(string) {
		env.Artifacts.SetPairSession(parked.Address, "pair-"+string(parked.Address.Tag), true)
	}
	env.Couch.resumeRegistrationTimeout = 200 * time.Millisecond
	return env, parked
}

func containsAddress(addresses []ThreadAddress, want ThreadAddress) bool {
	for _, address := range addresses {
		if address == want {
			return true
		}
	}
	return false
}

func rowFor(rows []ActionableThreadSummary, address ThreadAddress) (ActionableThreadSummary, bool) {
	for _, row := range rows {
		if row.Address == address {
			return row, true
		}
	}
	return ActionableThreadSummary{}, false
}

// AbortStarted must take ownership from couch's OWN registry, not from the
// StartResult handed back to it (pair#230 plan gate PQ-3).
//
// The console relays a struct across a package boundary, and the destructive
// branch is that struct's ZERO value: a caller that rebuilds or copies the
// record without carrying Warm would silently ask couch to delete a session it
// borrowed. Every other test here relays the record couch itself built, so Warm
// is correct by accident and the protection is invisible -- this one clears the
// field to make the registry the only source that can answer.
func TestAbortStartedReadsOwnershipFromTheRegistryNotTheCaller(t *testing.T) {
	env, address := warmDetachedThread(t)

	record, handle, err := env.Couch.ResumeContext(context.Background(), address)
	if err != nil {
		t.Fatalf("warm reattach: %v", err)
	}
	if !record.Warm {
		t.Fatal("the reattach was not recorded warm; this test would prove nothing")
	}
	relayed := record
	relayed.Warm = false // a caller that dropped the field

	if err := env.Couch.AbortStarted(StartResult{Record: relayed, Handle: handle}, errors.New("attach terminal has already exited")); err == nil {
		t.Fatal("AbortStarted returned nil, want the abort cause")
	}
	if quiesced := env.Artifacts.Quiesces(); containsAddress(quiesced, address) {
		t.Fatalf("AbortStarted quiesced %+v on a caller-supplied zero value -- ownership must come from the registry", address)
	}
}
