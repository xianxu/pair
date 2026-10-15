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
	"reflect"
	"strings"
	"testing"

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
		case checkpoint.Running:
			// A failed request is never in a record (#522): it reads as
			// the thread's line, so it routes as no request.
			return ResumeRouteContinuation
		case checkpoint.Pending:
			return ResumeRouteRecover
		}
	}
	if in.Slot {
		return ResumeRouteSlot
	}
	return ResumeRouteThread
}

func TestChooseResumeRoute(t *testing.T) {
	t.Parallel()

	phases := append(checkpoint.AllPhases(), "")
	for _, slot := range []bool{false, true} {
		for _, hasRecord := range []bool{false, true} {
			for _, phase := range phases {
				in := ResumeRouteInput{Slot: slot, HasRecord: hasRecord, Continuation: phase}
				if !hasRecord && phase != "" {
					continue // no record carries no continuation
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
// slot's own path preference. The listing proves an attachable terminal and
// the ledger proves the selected agent; adoption starts no native conversation.
func TestResumeOperationOnASlotPathAdoptsALostPointer(t *testing.T) {
	t.Parallel()

	env, local := slotRecoveryOperationFixture(t)
	writeSlotPathPreference(t, local, "claude")
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	address := ThreadAddress{RepoScope: scope.Key, Tag: "couch-1111111111111111"}
	env.Artifacts.SetPairSession(address, "pair-survivor", true)
	env.Artifacts.SetDetachedSession(address, "pair-survivor")
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-1")
	env.Couch.Workspaces = slotReadinessFunc(func(context.Context, ProvisionRequest) (ProvisionResult, error) {
		t.Fatal("warm adoption reconciled the workspace")
		return ProvisionResult{}, nil
	})
	value, err := dispatchResume(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": scope.Key})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := value.(StartResult)
	if !ok || result.Record.Thread != address || result.Handle == nil {
		t.Fatalf("resume did not adopt the survivor: %#v", value)
	}
	if result.Record.Shape != StartWarmReattach || childEnvValue(env.Runner.Child(result.Handle.ID()).Env, launcher.CouchLaunchProfileEnv) != "" {
		t.Fatal("lost pointer adoption launched a native agent")
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
// -- without matching native proof. The refusal says a session survives,
// not reboot, because reboot would
// refuse a slot with a live owner too.
func TestResumeAdoptionWithTheWrongAgentIsNotProved(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	env, source := switchEnvWithLiveThread(t)
	env.Proc.Kill(source.Incarnations[0].PID)
	env.Artifacts.SetDetachedSession(source.Address, "pair-exact")
	env.Artifacts.SetNativeBinding(source.Address, "claude", sessioninventory.BindingProvisional, "")
	value, err := dispatchResume(env.testEnv, map[string]string{"repo-scope": source.Address.RepoScope, "tag": string(source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	child, ok := value.(StartedChild)
	if !ok {
		t.Fatalf("unexpected result %#v", value)
	}
	started, ok := child.Started()
	if !ok || started.Record.Shape != StartWarmReattach || started.Handle == nil {
		t.Fatalf("not a warm reattach: %#v", value)
	}
	handle := started.Handle
	if childEnvValue(env.Runner.Child(handle.ID()).Env, launcher.CouchLaunchProfileEnv) != "" {
		t.Fatal("a fresh agent was started for a surviving session")
	}
}

// A compaction whose new conversation never started leaves the old one
// parked and one line (#522): resume brings the old conversation back through
// the ordinary resume -- the parked record's own resume authority -- with no
// launch prompt, and the line goes when it registers.
func TestResumeAfterAMissedContinuationReturnsTheOldConversation(t *testing.T) {
	t.Parallel()

	f := newContinuationFixture(t)
	f.env.Runner.FailNextStart(errors.New("fork unavailable"))
	if _, err := f.env.Couch.Continue(context.Background(), f.source.Address, f.status.RequestID); err == nil {
		t.Fatal("fork failure ignored")
	}
	before, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil || before.Continuation != nil || before.ContinuationMissed == nil || before.VerifiedPark == nil {
		t.Fatalf("fixture is not a missed continuation: %+v / %+v, %v", before.Continuation, before.ContinuationMissed, err)
	}
	f.env.Artifacts.SetNativeBinding(f.source.Address, "claude", sessioninventory.BindingEstablished, "native-old")
	value, err := dispatchResume(f.env.testEnv, map[string]string{"repo-scope": f.source.Address.RepoScope, "tag": string(f.source.Address.Tag)})
	if err != nil {
		t.Fatalf("resume refused after a missed continuation: %v", err)
	}
	child, ok := value.(StartedChild)
	if !ok {
		t.Fatalf("resume started nothing: %#v", value)
	}
	started, ok := child.Started()
	if !ok {
		t.Fatalf("resume started nothing: %#v", value)
	}
	var profile launcher.TrustedLaunchProfile
	if err := json.Unmarshal([]byte(childEnvValue(f.env.Runner.Child(started.Handle.ID()).Env, launcher.CouchLaunchProfileEnv)), &profile); err != nil {
		t.Fatal(err)
	}
	if !profile.ResumeRequired || profile.RequiredSessionID != "native-old" || profile.Prompt != "" {
		t.Fatalf("not the old conversation, or a prompt replayed: %+v", profile)
	}
}

// A parked :0 whose conversation no longer resolves cannot come back; resume
// says so with its exact refusal code and points at reboot.
func TestResumeThatCannotSucceedNamesReboot(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	env := newTestEnv(t, "/repo")
	record := archivableThread(t, env.Couch.Threads, "couch-0000000000000001")
	if _, err := env.Couch.Threads.CommitStartClaim(record.Address, record.Revision, "/repo/.git", env.Now, StartEvent{
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
	t.Parallel()

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
// warm-only resume: routing it through Continue or RecoverThread
// could start an agent behind the operator's back.
func TestResumeWarmOnlyIsUnrouted(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	env, local := slotRecoveryOperationFixture(t)
	_, err := env.Couch.spawnManagedResolution(context.Background(), StartResolution{
		Target: ThreadTarget{Kind: ThreadTargetSlot, Slot: *local.slot}, Action: StartOpen,
	})
	if ResumeDiagnosticOf(err) != ResumeNoSurvivor || !strings.Contains(err.Error(), "Tab → reboot") {
		t.Fatalf("open of an empty slot: diagnostic %q, err %v", ResumeDiagnosticOf(err), err)
	}
}

func TestResumeRecordlessColdEvidenceNeverLaunchesOrAdopts(t *testing.T) {
	env, local := slotRecoveryOperationFixture(t)
	writeSlotPathPreference(t, local, "claude")
	scope, _ := launcher.ResolveRepoScope(local.slot.WorktreeRoot)
	address := ThreadAddress{RepoScope: scope.Key, Tag: "couch-1111111111111111"}
	env.Artifacts.SetPairSession(address, "absent-old-session", false)
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "proven-native")
	_, err := dispatchResume(env, map[string]string{"path": local.slot.WorktreeRoot, "repo-scope": scope.Key})
	if ResumeDiagnosticOf(err) != ResumeNoSurvivor || !strings.Contains(strings.ToLower(err.Error()), "fresh") || len(env.Runner.Ops) != 0 {
		t.Fatalf("recordless cold launched: err=%v ops=%v", err, env.Runner.Ops)
	}
	if observed, e := local.observeSlotCurrent(); e != nil || observed.Record != nil {
		t.Fatalf("cold evidence adopted a pointer: %+v err=%v", observed, e)
	}
}

func TestResumeColdNativeRetainsPreparedConfirmationBoundary(t *testing.T) {
	f := newRecoveryExecutionFixture(t)
	f.Couch.RecoveryEvidence = nil
	f.Proc.Set(42, "old-helper")
	f.Artifacts.SetNativeBinding(f.record.Address, "codex", sessioninventory.BindingEstablished, "proven-native")
	retired := 0
	f.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		return RecoveryRetirement{Digest: "exact-owned", Stops: []RecoveryStopProcess{{PID: 42, Identity: "old-helper", Role: "wrapper"}}, Retire: func(context.Context) error { retired++; f.Proc.Kill(42); return nil }}, nil
	}
	_, err := dispatchResume(f.testEnv, map[string]string{"repo-scope": f.record.Address.RepoScope, "tag": string(f.record.Address.Tag)})
	var required *RecoveryConfirmationRequired
	if !errors.As(err, &required) || retired != 0 || len(f.Runner.Ops) != 0 {
		t.Fatalf("cold route bypassed confirmation: err=%v retired=%d ops=%v", err, retired, f.Runner.Ops)
	}
	result, err := f.Couch.ConfirmRecovery(context.Background(), required.Offer.Token)
	if err != nil || retired != 1 || result.Start.Handle == nil {
		t.Fatalf("confirmed cold route=%+v retired=%d err=%v", result, retired, err)
	}
}

func TestResumeUnknownListingNeverAttachesOrPreparesColdRecovery(t *testing.T) {
	env, record := peekEnv(t)
	env.Artifacts.SetPairSession(record.Address, "listed", true)
	env.Artifacts.ListedSessionHook = func(ThreadAddress) error { return errors.New("listing unanswered") }
	prepared := 0
	env.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		prepared++
		return RecoveryRetirement{}, nil
	}
	_, err := dispatchResume(env, map[string]string{"repo-scope": record.Address.RepoScope, "tag": string(record.Address.Tag)})
	if err == nil || prepared != 0 || len(env.Runner.Ops) != 0 {
		t.Fatalf("unknown listing acted: err=%v prepared=%d ops=%v", err, prepared, env.Runner.Ops)
	}
}

func TestResumePendingContinuationUsesItsRetainedCheckpointExecutor(t *testing.T) {
	f := newContinuationFixture(t)
	f.env.Proc.Kill(f.source.Incarnations[0].PID)
	f.env.Artifacts.SetPairSession(f.source.Address, "pair-exact", false)
	f.env.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		t.Fatal("retained continuation entered generic recovery retirement")
		return RecoveryRetirement{}, nil
	}
	value, err := dispatchResume(f.env.testEnv, map[string]string{"repo-scope": f.source.Address.RepoScope, "tag": string(f.source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	child, ok := value.(StartedChild)
	if !ok {
		t.Fatalf("continuation returned no start: %#v", value)
	}
	started, ok := child.Started()
	if !ok || started.Handle == nil || f.launches != 1 {
		t.Fatalf("continuation launch: value=%#v launches=%d", value, f.launches)
	}
	record, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil || record.Continuation == nil || record.Continuation.Phase != checkpoint.Complete || record.Continuation.SourceAbsence == nil {
		t.Fatalf("retained continuation not completed: %+v err=%v", record.Continuation, err)
	}
}

func TestResumePositiveListingOutranksPendingContinuation(t *testing.T) {
	f := newContinuationFixture(t)
	before, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	value, err := dispatchResume(f.env.testEnv, map[string]string{"repo-scope": f.source.Address.RepoScope, "tag": string(f.source.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	child, ok := value.(StartedChild)
	if !ok {
		t.Fatalf("Continue returned no client: %#v", value)
	}
	started, ok := child.Started()
	if !ok || started.Record.Shape != StartWarmReattach || childEnvValue(f.env.Runner.Child(started.Handle.ID()).Env, launcher.CouchLaunchProfileEnv) != "" {
		t.Fatalf("listing entered compaction launch: %#v", value)
	}
	after, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil || !reflect.DeepEqual(before, after) || len(f.env.Artifacts.TriggeredQuits()) != 0 {
		t.Fatalf("Continue changed retained pending request: before=%+v after=%+v err=%v", before, after, err)
	}
}

func TestResumeRunningContinuationRetriesItsExactRegisteredTarget(t *testing.T) {
	f := newContinuationFixture(t)
	first, err := f.env.Couch.Continue(context.Background(), f.source.Address, f.status.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	record, err := f.env.Couch.Threads.GetThread(f.source.Address)
	if err != nil {
		t.Fatal(err)
	}
	record, err = f.env.Couch.Threads.updateExistingThread(record.Address, record.Revision, func(next *ThreadRecord) error { unrecordTarget(next); return nil })
	if err != nil {
		t.Fatal(err)
	}
	f.env.Artifacts.SetPairSession(record.Address, record.SessionBinding.Name, false)
	f.env.Couch.RecoveryRetirement = func(context.Context, ThreadRecord) (RecoveryRetirement, error) {
		t.Fatal("running continuation entered generic native recovery")
		return RecoveryRetirement{}, nil
	}
	launches := f.launches
	value, err := dispatchResume(f.env.testEnv, map[string]string{"repo-scope": record.Address.RepoScope, "tag": string(record.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := value.(ContinuationResult)
	if !ok || result.Status.Phase != checkpoint.Complete || f.launches != launches || result.Status.RequestID != first.Status.RequestID {
		t.Fatalf("running continuation retry: value=%#v launches=%d want %d", value, f.launches, launches)
	}
}
