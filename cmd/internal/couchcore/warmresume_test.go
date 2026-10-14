package couchcore

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func warmDetachedRecord(t *testing.T) ThreadRecord {
	t.Helper()
	record := actionableTestThread("couch-0000000000000001", time.Unix(100, 0).UTC())
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	return record
}

// The operator's pair-couch-24: the zellij session is alive with zero clients,
// so the agent is RUNNING and reattaching restarts nothing. The native session
// id is what a COLD resume needs to relaunch the agent with `--resume <id>`;
// the warm path consumes it nowhere, so demanding it refuses a thread that
// would reattach fine.
func TestDecideResumeAcceptsADetachedThreadWithNoNativeBinding(t *testing.T) {
	t.Parallel()

	for _, status := range []sessioninventory.BindingStatus{
		sessioninventory.BindingProvisional,
		sessioninventory.BindingUnbound,
		sessioninventory.BindingAmbiguous,
	} {
		t.Run(string(status), func(t *testing.T) {
			eligible, err := DecideResume(ResumeEligibilityInput{
				Thread:            warmDetachedRecord(t),
				WorkingPathExists: true,
				Detached:          true,
				Binding:           NativeBindingResolution{Status: status},
			})
			if err != nil {
				t.Fatalf("warm reattach refused: %v", err)
			}
			if eligible.RequiredSessionID != "" {
				t.Fatalf("warm reattach carries RequiredSessionID %q -- nothing on that path consumes it",
					eligible.RequiredSessionID)
			}
		})
	}
}

// The cold path is unchanged, and this is what says so: a verified park with
// the same binding states must still refuse, naming its diagnostic. Relaxing
// the warm path must not leak into the path that really does relaunch an agent.
func TestDecideResumeStillRefusesAColdResumeWithoutAnEstablishedBinding(t *testing.T) {
	t.Parallel()

	for status, want := range map[sessioninventory.BindingStatus]ResumeDiagnosticCode{
		sessioninventory.BindingProvisional: ResumeBindingProvisional,
		sessioninventory.BindingUnbound:     ResumeBindingUnbound,
		sessioninventory.BindingAmbiguous:   ResumeBindingAmbiguous,
	} {
		t.Run(string(status), func(t *testing.T) {
			record := warmDetachedRecord(t)
			markActionableParked(&record, record.LastActiveAt)
			_, err := DecideResume(ResumeEligibilityInput{
				Thread:            record,
				WorkingPathExists: true,
				Binding:           NativeBindingResolution{Status: status},
			})
			if got := ResumeDiagnosticOf(err); got != want {
				t.Fatalf("cold resume with %s = %q, want %q", status, got, want)
			}
		})
	}
}

// The launch shape of a warm reattach, asserted end to end through Resume.
//
// Both omissions are load-bearing and neither is visible from the pure layer:
// the COUCH_LAUNCH_PROFILE env must be ABSENT (not empty -- Pair distinguishes
// them), because it carries ResumeRequired, which Pair honours only at a create
// boundary and a live session is an attach boundary; and `--layout2` must not
// be sent, because a running session already has its layout and asking for a
// different one sends Pair down a path that offers to DELETE it.
func TestWarmReattachSendsNoResumeProfileAndNoLayout(t *testing.T) {
	t.Parallel()

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

	// Detached: a live zellij session with no client, and NO usable native
	// binding -- the operator's pair-couch-24 exactly.
	env.Artifacts.SetDetachedSession(created.Address, "pair-"+string(created.Address.Tag))
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(created.Address, "pair-"+string(created.Address.Tag), true)
		return nil
	}

	_, handle, err := env.Couch.Resume(created.Address)
	if err != nil {
		t.Fatalf("warm reattach refused: %v", err)
	}
	child := env.Runner.Child(handle.ID())

	if !slices.Equal(child.Argv, []string{"pair", "--couch-session-v1", "resume", string(created.Address.Tag)}) {
		t.Fatalf("warm argv = %q, want a bare `pair resume <tag>`", child.Argv)
	}
	for _, entry := range child.Env {
		if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
			t.Fatalf("warm reattach sent a resume profile: %q", entry)
		}
		if strings.HasPrefix(entry, "PAIR_USE_REPO_DEFAULT=") {
			t.Fatalf("warm reattach sent a repo-default flag: %q", entry)
		}
	}
	// The couch-ownership env still travels: it is what marks the child as
	// couch's, and it is independent of the resume authority.
	wantOwnership := []string{
		"COUCH_THREAD_SCOPE=" + created.Address.RepoScope,
		"COUCH_THREAD_TAG=" + string(created.Address.Tag),
		"COUCH_THREAD_RESUME=1",
	}
	for _, want := range wantOwnership {
		if !slices.Contains(child.Env, want) {
			t.Fatalf("warm env = %q, missing %q", child.Env, want)
		}
	}
}

// An explicit warm-only request never cold-starts a stopped agent.
func TestWarmOnlyResumeNeverStartsAnAgent(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		build func(t *testing.T, env *testEnv) ThreadAddress
	}{
		{"verified-parked", func(t *testing.T, env *testEnv) ThreadAddress {
			parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "codex", Argv: []string{"--saved"}})
			env.Artifacts.SetNativeBinding(parked.Address, "codex", sessioninventory.BindingEstablished, "native-1")
			return parked.Address
		}},
		{"verified-parked with a provisional binding", func(t *testing.T, env *testEnv) ThreadAddress {
			// A LATE refusal would surface the binding error instead of the
			// warm-only code, so this row is what proves the refusal is first.
			parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "codex", Argv: []string{"--saved"}})
			env.Artifacts.SetNativeBinding(parked.Address, "codex", sessioninventory.BindingProvisional, "native-1")
			return parked.Address
		}},
		{"detached-shaped with its session gone", func(t *testing.T, env *testEnv) ThreadAddress {
			// Resume-shaped -- no incarnation, no park, a usable profile -- but
			// nothing is running: the thread a pass reaches after the session
			// died on its own. Without warm-only this would cold-start it.
			profile := LaunchProfile{Agent: "claude", Argv: []string{}}
			record := validThreadRecord(t)
			record.StartingPath, record.WorkingPath = "/repo", "/repo"
			record.Reservation = false
			record.LatestLaunchProfile = &profile
			created, err := env.Couch.Threads.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			return created.Address
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			address := tt.build(t, env)
			before, err := env.Couch.Threads.GetThread(address)
			if err != nil {
				t.Fatal(err)
			}

			_, handle, err := env.Couch.ResumeContextWith(context.Background(), address, ResumeOptions{WarmOnly: true})
			if code := ResumeDiagnosticOf(err); code != ResumeNotDetached {
				t.Fatalf("diagnostic = %q (err %v), want %q", code, err, ResumeNotDetached)
			}
			if handle != nil {
				t.Fatal("a warm-only resume spawned a child")
			}
			after, err := env.Couch.Threads.GetThread(address)
			if err != nil {
				t.Fatal(err)
			}
			if after.Revision != before.Revision {
				t.Fatalf("revision %d -> %d: a warm-only refusal wrote the record", before.Revision, after.Revision)
			}
		})
	}
}

// And a thread that IS warm still reattaches under warm-only -- the flag
// narrows what may happen, it does not change the happy path.
func TestWarmOnlyResumeStillReattachesADetachedThread(t *testing.T) {
	t.Parallel()

	env, address := warmDetachedThread(t)
	env.Runner.AfterAcknowledge = func(string) error {
		env.Artifacts.SetPairSession(address, "pair-"+string(address.Tag), true)
		return nil
	}
	record, handle, err := env.Couch.ResumeContextWith(context.Background(), address, ResumeOptions{WarmOnly: true})
	if err != nil {
		t.Fatalf("warm-only refused a detached thread: %v", err)
	}
	if handle == nil || record.Shape != StartWarmReattach {
		t.Fatalf("record = %+v, want a warm reattach", record)
	}
}

// warm-only through the DECLARED operation, which is the only way the pass can
// ask for it: the console dispatches `resume` by name through the operation
// table. A test that calls ResumeContextWith directly would pass even if the
// dispatcher dropped the argument, and the pass would then cold-start a parked
// thread's agent -- exactly the outcome warm-only exists to prevent.
func TestWarmOnlyReachesTheResumeThroughTheOperationTable(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "codex", Argv: []string{"--saved"}})
	env.Artifacts.SetNativeBinding(parked.Address, "codex", sessioninventory.BindingEstablished, "native-1")

	_, err := DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch)}, OperationCall{
		Name: "resume", Implicit: true, Context: context.Background(),
		Args: map[string]string{
			"repo-scope": parked.Address.RepoScope,
			"tag":        string(parked.Address.Tag),
			"warm-only":  "true",
		},
	})
	if code := ResumeDiagnosticOf(err); code != ResumeNotDetached {
		t.Fatalf("dispatched warm-only resume of a parked thread: diagnostic %q (err %v), want %q", code, err, ResumeNotDetached)
	}
	if children := env.Runner.Ops; len(children) != 0 {
		t.Fatalf("runner ops %v: the parked thread's agent was started", children)
	}
}

// Native transcript resolution is not even an available capability here.
// The same portable session state still authorizes inventory and execution.
// warmSessionArtifacts is a couch whose artifacts expose only the session
// surfaces -- no native resolver. SessionPresenceResolver belongs here because
// after #256 the inventory's session question IS presence; a couch that cannot
// answer it reads every thread `unknown`, which is correct fail-closed behaviour
// but not what this test is about.
type warmSessionArtifacts struct {
	ThreadArtifactController
	DetachedSessionResolver
	PairSessionIO
	SessionPresenceResolver
}

func TestWarmInventoryAndResumeNeedNoNativeResolver(t *testing.T) {
	t.Parallel()

	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "failing-resolver", true: "missing-resolver"}[missing], func(t *testing.T) {
			env, address := warmDetachedThread(t)
			env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingUnbound, "")
			probe := &warmPathResolverProbe{FakeThreadArtifactCollisionChecker: env.Artifacts}
			env.Couch.Artifacts = probe
			if missing {
				env.Couch.Artifacts = warmSessionArtifacts{env.Artifacts, env.Artifacts, env.Artifacts, env.Artifacts}
			}
			rows, err := env.Couch.ActionableThreadInventory(nil)
			if err != nil || len(rows) != 1 || rows[0].State != ThreadDetached {
				t.Errorf("warm inventory = %+v, %v; want detached without a native resolver", rows, err)
			}
			record, handle, err := env.Couch.ResumeContextWith(t.Context(), address, ResumeOptions{WarmOnly: true})
			if err != nil || handle == nil || record.Shape != StartWarmReattach {
				t.Errorf("warm execution = %+v, %v; want warm reattachment", record, err)
			}
			if probe.calls != 0 || env.Artifacts.BindingResolutions() != 0 {
				t.Errorf("warm path resolved native binding: probe=%d fake=%d", probe.calls, env.Artifacts.BindingResolutions())
			}
		})
	}
}

func TestWarmProofMatcherAcceptsOnlyExactSessionEvidence(t *testing.T) {
	t.Parallel()

	record := warmDetachedRecord(t)
	valid := DetachedSessionObservation{Address: record.Address, SessionName: "pair-surviving", Agent: "claude"}
	for _, tt := range []struct {
		name         string
		observations []DetachedSessionObservation
		want         bool
	}{
		{"unbound", []DetachedSessionObservation{valid}, true},
		{"absent", nil, false},
		{"duplicate", []DetachedSessionObservation{valid, valid}, false},
		{"wrong-address", []DetachedSessionObservation{{Address: ThreadAddress{RepoScope: "other", Tag: record.Address.Tag}, SessionName: valid.SessionName, Agent: valid.Agent}}, false},
		{"saved-agent-does-not-gate-continue", []DetachedSessionObservation{{Address: record.Address, SessionName: valid.SessionName, Agent: "codex"}}, true},
		{"empty-session", []DetachedSessionObservation{{Address: record.Address, Agent: valid.Agent}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := detachedResumeProofMatches(record, tt.observations); got != tt.want {
				t.Fatalf("proof = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWarmRecheckRefusesSessionReplacementAfterClaim(t *testing.T) {
	t.Parallel()

	env, address := warmDetachedThread(t)
	calls := 0
	env.Artifacts.DetachedSessionsHook = func([]ThreadAddress) error {
		calls++
		if calls == 2 {
			env.Artifacts.SetDetachedSession(address, "pair-replacement")
		}
		return nil
	}
	_, handle, err := env.Couch.ResumeContextWith(t.Context(), address, ResumeOptions{WarmOnly: true})
	if ResumeDiagnosticOf(err) != ResumeSessionGone || handle != nil {
		t.Fatalf("replacement accepted: handle=%v err=%v", handle, err)
	}
	after, err := env.Couch.Threads.GetThread(address)
	if err != nil || len(after.Incarnations) != 0 || len(env.Runner.Ops) != 0 {
		t.Fatalf("failed recheck left effects: record=%+v ops=%v err=%v", after, env.Runner.Ops, err)
	}
}

func (a warmSessionArtifacts) NamedPairSessionContext(ctx context.Context, address ThreadAddress, name string) (PairSessionBinding, error) {
	return a.PairSessionIO.(interface {
		NamedPairSessionContext(context.Context, ThreadAddress, string) (PairSessionBinding, error)
	}).NamedPairSessionContext(ctx, address, name)
}
