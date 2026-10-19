package couchcore

import (
	"context"
	"reflect"
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
		lifetimeLeaseEnvForTest(t, env.Dir, created.StartingPath, created.Address),
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

func TestWarmRecheckRefusesSessionReplacementBeforeHelper(t *testing.T) {
	t.Parallel()

	env, address := warmDetachedThread(t)
	calls := 0
	env.Artifacts.ListedSessionHook = func(ThreadAddress) error {
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

// Continue consumes the live terminal, not the cold-launch metadata saved for
// recreating its agent. Exercise the executor, including the child boundary.
func TestWarmResumeExecutorIgnoresColdMetadataDefects(t *testing.T) {
	for _, defect := range []string{"missing-profile", "incomplete-profile", "unsupported-agent", "missing-working-path", "missing-path-observer"} {
		t.Run(defect, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			env.Couch.Slots = &SlotCatalogFake{}
			record := warmDetachedRecord(t)
			record.StartingPath, record.WorkingPath = "/repo", "/repo"
			record.Reservation = false
			preference := PathLaunchPreference{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: "existing-repo", PhysicalPath: "/repo", LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {"--search"}}, Revision: 1}
			if err := writePathLaunchPreferenceForTest(env.Couch.Threads, preference); err != nil {
				t.Fatal(err)
			}
			switch defect {
			case "missing-profile":
				record.LatestLaunchProfile = nil
			case "incomplete-profile":
				// The store normalizes legacy null argv to configured-empty;
				// preserve that case without manufacturing an invalid record.
				record.LatestLaunchProfile.Argv = nil
			case "unsupported-agent":
				record.LatestLaunchProfile.Agent = "old-unsupported-agent"
			case "missing-working-path":
				env.Couch.Path.(*FakePathOps).Fail("/repo")
			case "missing-path-observer":
				env.Couch.Path = nil
			}
			created, err := env.Couch.Threads.CreateThread(record)
			if err != nil {
				t.Fatal(err)
			}
			session := "pair-" + string(created.Address.Tag)
			env.Artifacts.SetDetachedSession(created.Address, session)
			env.Runner.AfterAcknowledge = func(string) error { env.Artifacts.SetPairSession(created.Address, session, true); return nil }
			_, handle, err := env.Couch.ResumeContextWith(context.Background(), created.Address, ResumeOptions{WarmOnly: true})
			if err != nil {
				t.Fatalf("Continue refused saved %s: %v", defect, err)
			}
			if handle == nil {
				t.Fatal("Continue did not start attachment")
			}
			child := env.Runner.Child(handle.ID())
			if defect == "missing-working-path" && child.Dir != "/" {
				t.Fatalf("missing workspace helper cwd=%q, want safe host root", child.Dir)
			}
			if !slices.Equal(child.Argv, []string{"pair", "--couch-session-v1", "resume", string(created.Address.Tag)}) {
				t.Fatalf("not exact warm attachment: %q", child.Argv)
			}
			if env.Artifacts.BindingResolutions() != 0 {
				t.Fatal("Continue resolved native conversation")
			}
			for _, entry := range child.Env {
				if strings.HasPrefix(entry, launcher.CouchLaunchProfileEnv+"=") {
					t.Fatalf("Continue fabricated cold launch authority: %q", entry)
				}
			}
			after, err := env.Couch.Threads.GetThread(created.Address)
			if err != nil {
				t.Fatal(err)
			}
			if after.WorkingPath != created.WorkingPath {
				t.Fatalf("Continue rewrote saved working path: before=%q after=%q", created.WorkingPath, after.WorkingPath)
			}
			afterPreference, found, err := env.Couch.Threads.GetPathLaunchPreference(preference.RepoIdentity, preference.PhysicalPath)
			if err != nil || !found || !reflect.DeepEqual(afterPreference, preference) {
				t.Fatalf("Continue changed defaults: %+v found=%v err=%v", afterPreference, found, err)
			}
			if !reflect.DeepEqual(after, created) {
				t.Fatalf("Continue changed durable thread: before=%+v after=%+v", created, after)
			}
			if !reflect.DeepEqual(after.LatestLaunchProfile, created.LatestLaunchProfile) {
				t.Fatalf("Continue rewrote saved profile: before=%+v after=%+v", created.LatestLaunchProfile, after.LatestLaunchProfile)
			}
		})
	}
}

func (a warmSessionArtifacts) ListedPairSessionContext(ctx context.Context, address ThreadAddress, name string) (PairSessionBinding, error) {
	return a.PairSessionIO.(interface {
		ListedPairSessionContext(context.Context, ThreadAddress, string) (PairSessionBinding, error)
	}).ListedPairSessionContext(ctx, address, name)
}
