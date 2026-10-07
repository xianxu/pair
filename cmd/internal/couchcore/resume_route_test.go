package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// declaredResumeCodes is every constant of type ResumeDiagnosticCode written in
// resume.go, found by parsing the file rather than by a hand list -- a code
// added later reaches this test without anyone remembering to add it here.
func declaredResumeCodes(t *testing.T) map[ResumeDiagnosticCode]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "resume.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[ResumeDiagnosticCode]bool{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value := spec.(*ast.ValueSpec)
			ident, ok := value.Type.(*ast.Ident)
			if !ok || ident.Name != "ResumeDiagnosticCode" {
				continue
			}
			for _, lit := range value.Values {
				basic, ok := lit.(*ast.BasicLit)
				if !ok || basic.Kind != token.STRING {
					t.Fatalf("ResumeDiagnosticCode constant with a non-literal value: %#v", lit)
				}
				codes[ResumeDiagnosticCode(strings.Trim(basic.Value, `"`))] = true
			}
		}
	}
	if len(codes) == 0 {
		t.Fatal("parsed no ResumeDiagnosticCode constants; the enumeration proves nothing")
	}
	return codes
}

func TestResumeRebootAdviceClassifiesEveryDeclaredCode(t *testing.T) {
	declared := declaredResumeCodes(t)
	for code := range declared {
		if _, ok := ResumeRebootAdvice[code]; !ok {
			t.Errorf("%s is declared but ResumeRebootAdvice does not say whether it names reboot", code)
		}
	}
	for code := range ResumeRebootAdvice {
		if !declared[code] {
			t.Errorf("ResumeRebootAdvice classifies %s, which resume.go does not declare", code)
		}
	}
}

// The Spec's split, as an independent literal: reboot is named only when the
// transcript cannot come back; a transient refusal says to retry instead.
func TestResumeRebootAdviceValues(t *testing.T) {
	want := map[ResumeDiagnosticCode]bool{
		ResumePathMissing:        true,
		ResumeProfileMissing:     true,
		ResumeProfileInvalid:     true,
		ResumeAgentUnsupported:   true,
		ResumeBindingUnbound:     true,
		ResumeBindingRootMissing: true,
		ResumeBindingAmbiguous:   true,
		ResumeTombstoned:         true,
		ResumeSessionGone:        true,
		ResumeNoSurvivor:         true,
		ResumeLive:               false,
		ResumeUnknown:            false,
		ResumeParking:            false,
		ResumeStarting:           false,
		ResumeBindingProvisional: false,
		ResumeNotDetached:        false,
		ResumeNotRunning:         false,
		ResumeSurvivorsAmbiguous: false,
		ResumeSurvivorUnproven:   false,
		// The conversation is still running behind an orphaned server; the
		// advice is reap, never reboot (#399).
		ResumeOrphanedServer: false,
	}
	if len(want) != len(ResumeRebootAdvice) {
		t.Errorf("advice has %d codes, the Spec lists %d", len(ResumeRebootAdvice), len(want))
	}
	for code, names := range want {
		got, ok := ResumeRebootAdvice[code]
		if !ok || got != names {
			t.Errorf("%s: advice %v (present %v), want %v", code, got, ok, names)
		}
	}
}

func TestWithRebootAdviceWrapsAndKeepsTheCode(t *testing.T) {
	cannot := withRebootAdvice(&ResumeRefusal{Code: ResumeBindingUnbound, Diagnostic: "no conversation"})
	if !strings.Contains(cannot.Error(), "reboot") || ResumeDiagnosticOf(cannot) != ResumeBindingUnbound {
		t.Fatalf("unrecoverable refusal: %v (code %q)", cannot, ResumeDiagnosticOf(cannot))
	}
	transient := withRebootAdvice(fmt.Errorf("wrapped: %w", &ResumeRefusal{Code: ResumeStarting, Diagnostic: "busy"}))
	if strings.Contains(transient.Error(), "reboot") {
		t.Fatalf("transient refusal names reboot: %v", transient)
	}
	plain := errors.New("plain")
	if withRebootAdvice(plain) != plain || withRebootAdvice(nil) != nil {
		t.Fatal("an error without a resume code must pass through untouched")
	}
}

// expectedResumeRoute is the plan's route table written as its own literal.
func expectedResumeRoute(in ResumeRouteInput) ResumeRoute {
	if in.Slot && !in.HasRecord {
		return ResumeRouteSlot
	}
	if in.HasRecord {
		switch in.Continuation {
		case checkpoint.Failed, checkpoint.Running:
			return ResumeRouteContinuation
		case checkpoint.Pending:
			return ResumeRouteRecover
		}
	}
	if in.Slot {
		return ResumeRouteSlot
	}
	if in.HasRecord && in.RecoveryRecover {
		return ResumeRouteRecover
	}
	return ResumeRouteThread
}

func TestChooseResumeRoute(t *testing.T) {
	phases := append(checkpoint.AllPhases(), "")
	for _, slot := range []bool{false, true} {
		for _, hasRecord := range []bool{false, true} {
			for _, phase := range phases {
				for _, recover := range []bool{false, true} {
					in := ResumeRouteInput{Slot: slot, HasRecord: hasRecord, Continuation: phase, RecoveryRecover: recover}
					if !hasRecord && (phase != "" || recover) {
						continue // no record carries no continuation and no recovery verdict
					}
					want := expectedResumeRoute(in)
					if want == 0 {
						t.Fatalf("%+v: the expected table has no route", in)
					}
					if got := ChooseResumeRoute(in); got != want {
						t.Errorf("%+v: route %v, want %v", in, got, want)
					}
				}
			}
		}
	}
}

// dispatchResume drives `resume` through the declared operation and the
// live-owner executor -- the production boundary the switcher and the console
// both reach -- so a dispatcher that dropped or misrouted an argument fails
// here rather than only in the terminal.
func dispatchResume(env *testEnv, args map[string]string) (any, error) {
	return DispatchOperation(OperationExecutors{LiveOwner: CouchLiveOwnerExecutor(env.Couch), DirectStore: DirectStoreExecutor(env.Couch)},
		OperationCall{Name: "resume", Args: args, Implicit: true, Context: context.Background()})
}

func writeSlotPathPreference(t *testing.T, local *ThreadStore, agent string) {
	t.Helper()
	saved := PathLaunchPreference{SchemaVersion: 1, RepoIdentity: local.slot.RepoIdentity, PhysicalPath: local.slot.WorktreeRoot, LastAgent: agent, ArgvByAgent: map[string][]string{agent: {}}, Revision: 1}
	raw, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(local.root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local.root, "preferences.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

// A slot whose thread pointer couch lost, with its conversation still bound
// in the native ledger. Resume carries no agent argument -- the operator only
// pressed resume -- so the agent the adoption is proved against comes from the
// slot's own path preference, and the ledger proves it per agent.
func TestResumeOperationOnASlotPathAdoptsALostPointer(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	writeSlotPathPreference(t, local, "claude")
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	address := ThreadAddress{RepoScope: scope.Key, Tag: "couch-1111111111111111"}
	env.Artifacts.SetPairSession(address, "pair-survivor", false)
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-1")
	// The cold resume's session coming up births its agent pane.
	env.Runner.AfterAcknowledge = func(id string) error {
		env.Artifacts.SetPairSession(address, continuationChildSession(t, env.Runner, id), true)
		return nil
	}
	value, err := dispatchResume(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": scope.Key})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := value.(StartResult)
	if !ok || result.Record.Thread != address || result.Handle == nil {
		t.Fatalf("resume did not adopt the survivor: %#v", value)
	}
	current, err := local.GetThread(address)
	if err != nil || current.LatestLaunchProfile == nil || current.LatestLaunchProfile.Agent != "claude" {
		t.Fatalf("thread.json does not name the survivor: %+v, %v", current, err)
	}
}

// An adoption is only as good as the evidence for its agent. Resume's agent is
// a GUESS from the slot's preference, so it may adopt only where the world
// checks that guess.
//
// Cold: the native ledger binds per agent, so a record-less survivor bound for
// claude is not proved by a codex guess; there is no conversation to adopt, and
// the refusal names reboot.
//
// Warm: the detached-session proof echoes whatever agent it is asked about and
// observes nothing about which agent runs in the session (pane sidecars name
// agents but keep stale twins, and nothing reads them as identity). So a
// record-less detached survivor is never adopted on a guess -- right or wrong
// -- and the refusal says a session survives, not reboot, because reboot would
// refuse a slot with a live owner too.
func TestResumeAdoptionWithTheWrongAgentIsNotProved(t *testing.T) {
	t.Run("cold", func(t *testing.T) {
		env, local := slotRecoveryOperationFixture(t)
		writeSlotPathPreference(t, local, "codex")
		scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
		address := ThreadAddress{RepoScope: scope.Key, Tag: "couch-1111111111111111"}
		env.Artifacts.SetPairSession(address, "pair-survivor", false)
		env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-1")
		_, err := dispatchResume(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": scope.Key})
		if code := ResumeDiagnosticOf(err); code != ResumeNoSurvivor {
			t.Fatalf("diagnostic %q (err %v), want %q", code, err, ResumeNoSurvivor)
		}
		if !strings.Contains(err.Error(), "reboot") {
			t.Fatalf("a refusal with no way back does not name reboot: %v", err)
		}
		if len(env.Runner.Ops) != 0 {
			t.Fatalf("runner ops %v: something was spawned", env.Runner.Ops)
		}
		if _, err := local.GetThread(address); err == nil {
			t.Fatal("an unproved survivor was adopted into thread.json")
		}
	})
	for _, guess := range []string{"codex", "claude"} {
		t.Run("warm/"+guess, func(t *testing.T) {
			env, local := slotRecoveryOperationFixture(t)
			writeSlotPathPreference(t, local, guess)
			scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
			address := ThreadAddress{RepoScope: scope.Key, Tag: "couch-1111111111111111"}
			env.Artifacts.SetPairSession(address, "pair-survivor", true)
			env.Artifacts.SetDetachedSession(address, "pair-survivor")
			_, err := dispatchResume(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": scope.Key})
			if code := ResumeDiagnosticOf(err); code != ResumeSurvivorUnproven {
				t.Fatalf("diagnostic %q (err %v), want %q", code, err, ResumeSurvivorUnproven)
			}
			if strings.Contains(err.Error(), "reboot") {
				t.Fatalf("a live survivor's refusal names reboot, which would refuse too: %v", err)
			}
			if len(env.Runner.Ops) != 0 {
				t.Fatalf("runner ops %v: something was spawned", env.Runner.Ops)
			}
			if _, err := local.GetThread(address); err == nil {
				t.Fatal("a survivor was adopted on an unproven agent guess")
			}
		})
	}
}

// The switcher's row said session-gone when it was drawn; by the time resume
// runs, the agent is back behind a detached session and its helper is dead.
// Resume decides from fresh evidence, not the row, so it reattaches warm --
// today's Enter→recover-thread outcome -- and starts no new agent.
func TestResumeReattachesASurvivorOfAnUnusablePrimary(t *testing.T) {
	env, source := switchEnvWithLiveThread(t)
	env.Proc.Kill(source.Incarnations[0].PID)
	env.Artifacts.SetDetachedSession(source.Address, "pair-exact")
	env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingProvisional, "")
	value, err := dispatchResume(env.testEnv, map[string]string{"repo-scope": source.Address.RepoScope, "tag": string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	var shape StartShape
	var handle Handle
	switch result := value.(type) {
	case StartResult:
		shape, handle = result.Record.Shape, result.Handle
	case ContinuationResult:
		shape, handle = result.Record.Shape, result.Handle
	default:
		t.Fatalf("unexpected result %#v", value)
	}
	if shape != StartWarmReattach || handle == nil {
		t.Fatalf("not a warm reattach: shape %q", shape)
	}
	if childEnvValue(env.Runner.Child(handle.ID()).Env, launcher.CouchLaunchProfileEnv) != "" {
		t.Fatal("a fresh agent was started for a surviving session")
	}
}

// A parked thread whose continuation failed is retried where it stands: the
// request moves on rather than the generic resume refusing it with the
// continuation guard.
func TestResumeOperationRetriesAFailedContinuation(t *testing.T) {
	f := newContinuationFixture(t)
	f.env.Runner.FailNextStart(errors.New("fork unavailable"))
	if _, err := f.env.Couch.Continue(context.Background(), f.source.Address, f.status.RequestID); err == nil {
		t.Fatal("fork failure ignored")
	}
	before, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil || before.Continuation == nil || before.Continuation.Phase != checkpoint.Failed {
		t.Fatalf("fixture is not a failed continuation: %+v, %v", before.Continuation, err)
	}
	value, err := dispatchResume(f.env.testEnv, map[string]string{"repo-scope": f.source.Address.RepoScope, "tag": string(f.source.Address.Tag)})
	if err != nil {
		t.Fatalf("resume refused a failed continuation: %v", err)
	}
	if _, ok := value.(ContinuationResult); !ok {
		t.Fatalf("resume did not take the continuation retry: %#v", value)
	}
	after, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil || after.Continuation == nil || after.Continuation.Phase == checkpoint.Failed {
		t.Fatalf("the request did not advance: %+v, %v", after.Continuation, err)
	}
	if f.launches != 1 {
		t.Fatalf("retry launches %d, want 1", f.launches)
	}
}

// A parked :0 whose conversation no longer resolves cannot come back; resume
// says so with its exact refusal code and points at reboot.
func TestResumeThatCannotSucceedNamesReboot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	_, err := dispatchResume(env, map[string]string{"repo-scope": parked.Address.RepoScope, "tag": string(parked.Address.Tag)})
	if code := ResumeDiagnosticOf(err); code != ResumeBindingUnbound {
		t.Fatalf("diagnostic %q (err %v), want %q", code, err, ResumeBindingUnbound)
	}
	if !strings.Contains(err.Error(), "reboot") {
		t.Fatalf("an unrecoverable refusal does not name reboot: %v", err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("runner ops %v", env.Runner.Ops)
	}
}

// A start another couch is still driving is transient: the answer is to wait,
// and reboot would be the wrong advice.
func TestResumeTransientRefusalDoesNotNameReboot(t *testing.T) {
	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
	if _, err := env.Couch.Threads.CommitStartClaim(record.Address, record.Revision, "/repo/.git", time.Unix(200, 0).UTC(), StartEvent{
		Kind: StartClaimed, Nonce: "start-0123456789abcdef", Shape: StartFreshExisting,
		Owner:   SupervisorOwner{PID: 77, Identity: "owner-couch"},
		Profile: record.LatestLaunchProfile,
	}); err != nil {
		t.Fatal(err)
	}
	env.Proc.Set(77, "owner-couch")
	// The conversation itself resolves, so the only refusal left is the
	// start in flight.
	env.Artifacts.SetNativeBinding(record.Address, "claude", sessioninventory.BindingEstablished, "native-1")
	_, err := dispatchResume(env, map[string]string{"repo-scope": record.Address.RepoScope, "tag": string(record.Address.Tag)})
	if code := ResumeDiagnosticOf(err); code != ResumeStarting {
		t.Fatalf("diagnostic %q (err %v), want %q", code, err, ResumeStarting)
	}
	if strings.Contains(err.Error(), "reboot") {
		t.Fatalf("a transient refusal names reboot: %v", err)
	}
}

// OpenSlot's survivor refusals are typed, so resume can tell "nothing to
// adopt" (reboot) from "several candidates" (stop one, then retry).
func TestOpenSlotZeroSurvivorsIsATypedRefusal(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	if _, err := env.Couch.OpenSlot(context.Background(), local.slot.WorktreeRoot, "claude"); ResumeDiagnosticOf(err) != ResumeNoSurvivor {
		t.Fatalf("zero survivors: diagnostic %q (err %v)", ResumeDiagnosticOf(err), err)
	}
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	for _, tag := range []ThreadTag{"couch-1111111111111111", "couch-2222222222222222"} {
		address := ThreadAddress{RepoScope: scope.Key, Tag: tag}
		env.Artifacts.SetPairSession(address, "pair-"+string(tag), false)
		env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-"+string(tag))
	}
	if _, err := env.Couch.OpenSlot(context.Background(), local.slot.WorktreeRoot, "claude"); ResumeDiagnosticOf(err) != ResumeSurvivorsAmbiguous {
		t.Fatalf("two survivors: diagnostic %q (err %v)", ResumeDiagnosticOf(err), err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("runner ops %v", env.Runner.Ops)
	}
}

// The background reattach pass asks for warm-only, and it must stay a direct
// warm-only resume: routing it through RetryContinuation or RecoverThread
// could start an agent behind the operator's back.
func TestWarmOnlyResumeIsUnrouted(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "codex", Argv: []string{"--saved"}})
	env.Artifacts.SetNativeBinding(parked.Address, "codex", sessioninventory.BindingEstablished, "native-1")
	_, err := dispatchResume(env, map[string]string{"repo-scope": parked.Address.RepoScope, "tag": string(parked.Address.Tag), "warm-only": "true"})
	if code := ResumeDiagnosticOf(err); code != ResumeNotDetached {
		t.Fatalf("diagnostic %q (err %v), want %q", code, err, ResumeNotDetached)
	}
	if strings.Contains(err.Error(), "reboot") {
		t.Fatalf("a warm-only skip names reboot: %v", err)
	}
	if len(env.Runner.Ops) != 0 {
		t.Fatalf("runner ops %v", env.Runner.Ops)
	}
}

// The start form's open reaches OpenSlot without resume's top, and still names
// reboot when nothing can come back (pair#363 M1 review: OpenSlot's refusals
// had lost their next step).
func TestStartFormOpenOfAnEmptySlotNamesReboot(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	_, err := env.Couch.spawnManagedResolution(context.Background(), StartResolution{
		Target: ThreadTarget{Kind: ThreadTargetSlot, Slot: *local.slot}, Action: StartOpen,
	})
	if ResumeDiagnosticOf(err) != ResumeNoSurvivor || !strings.Contains(err.Error(), "Tab → reboot") {
		t.Fatalf("open of an empty slot: diagnostic %q, err %v", ResumeDiagnosticOf(err), err)
	}
}
