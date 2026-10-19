package couchcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

const (
	bindScope = "scope"
	bindTag   = "work"
	bindRoot  = "claude-projects"
)

var bindNow = time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

func bindID(n int) string { return fmt.Sprintf("0b5c2a1e-9f1d-4c4e-8d5b-%012d", n) }

func bindRel(n int) string { return "-Users-x-workspace-pair/" + bindID(n) + ".jsonl" }

// bindFixture is a native runtime with n claude transcripts and the tag's
// ledger, whose launches bind, in order, to the conversations in roots.
func bindFixture(t *testing.T, transcripts int, roots ...int) *sessioninventorytest.FakeRuntime {
	t.Helper()
	rt := sessioninventorytest.NewFakeRuntime()
	rt.SetPairDataRoot(sessioninventory.StorageRoot{Name: "pair-data", Path: "/data/repos/" + bindScope})
	rt.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: bindRoot, Path: "/home/.claude/projects"})
	for i := 0; i < transcripts; i++ {
		rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: bindRoot, RelativePath: bindRel(i)}}, []byte("{}\n"))
	}
	var ledger []byte
	for i, n := range roots {
		launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: bindScope, Tag: bindTag, Agent: "claude", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
		if err != nil {
			t.Fatal(err)
		}
		state, err := json.Marshal(sessioninventory.ScannerState{Version: sessioninventory.ScannerStateVersion, Agent: sessioninventory.AgentClaude, NativeID: bindID(n), IdentityAnchor: bindID(n), Role: sessioninventory.RoleRoot, ScannerSchema: "claude-v1", FirstRecordValidated: true})
		if err != nil {
			t.Fatal(err)
		}
		proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: bindID(n), ScannerSchema: "claude-v1", ScannerState: state,
			Artifacts: []sessionledger.ArtifactProof{{StorageRoot: bindRoot, RelativePath: bindRel(n), StableFileID: "inode", MutationToken: "ctime", Size: 3, ParserCompleteOffset: 3}}}
		binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordBinding, ScopeKey: bindScope, Tag: bindTag, Agent: "claude", LaunchOrdinal: uint64(2*i + 1), RootNativeID: bindID(n), AuthorizationProof: &proof})
		if err != nil {
			t.Fatal(err)
		}
		ledger = append(ledger, launch...)
		ledger = append(ledger, '\n')
		ledger = append(ledger, binding...)
		ledger = append(ledger, '\n')
	}
	if len(roots) > 0 {
		rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "ledger-" + bindTag + ".jsonl"}}, ledger)
	}
	return rt
}

func bindThread(stored *threadrecord.TranscriptBinding) ThreadRecord {
	return ThreadRecord{
		Address:             ThreadAddress{RepoScope: bindScope, Tag: bindTag},
		LatestLaunchProfile: &LaunchProfile{Agent: "claude"},
		Transcript:          stored,
	}
}

func storedBinding(n int) *threadrecord.TranscriptBinding {
	return &threadrecord.TranscriptBinding{Agent: "claude", NativeID: bindID(n), StorageRoot: bindRoot, RelativePath: bindRel(n),
		BoundAt: bindNow.Add(-time.Hour), Source: threadrecord.TranscriptSourceLedger}
}

// listingResolver fails the test if the listing fallback runs.
type listingResolver struct{ t *testing.T }

func (r listingResolver) ResolveEstablished(context.Context, string, string, string) (NativeBindingResolution, error) {
	r.t.Fatal("the listing resolver ran for a binding that names its file")
	return NativeBindingResolution{}, nil
}

func assertBindCost(t *testing.T, rt *sessioninventorytest.FakeRuntime, transcript string) {
	t.Helper()
	for _, root := range []string{"pair-data", bindRoot} {
		if n := rt.OperationCountForRoot(sessioninventorytest.OperationListFiles, root); n != 0 {
			t.Errorf("%s listings = %d, want 0", root, n)
		}
	}
	if n := rt.OperationCountForRoot(sessioninventorytest.OperationReadAt, bindRoot) + rt.OperationCountForRoot(sessioninventorytest.OperationReadFile, bindRoot); n != 0 {
		t.Errorf("transcript body reads = %d, want 0", n)
	}
	if n := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-"+bindTag+".jsonl"); n != 1 {
		t.Errorf("ledger reads = %d, want 1", n)
	}
	if n := rt.OperationCountForRoot(sessioninventorytest.OperationStatFile, bindRoot); n != 1 {
		t.Errorf("transcript stats = %d, want 1", n)
	}
	if transcript != "" {
		if n := rt.OperationCount(sessioninventorytest.OperationStatFile, bindRoot+":"+transcript); n != 1 {
			t.Errorf("stat of %s = %d, want 1", transcript, n)
		}
	}
}

func TestResumeBindsByOneStat(t *testing.T) {
	t.Parallel()
	for _, transcripts := range []int{8, 16} {
		t.Run(fmt.Sprint(transcripts), func(t *testing.T) {
			rt := bindFixture(t, transcripts, 3)
			got, err := bindTranscriptWith(context.Background(), rt, listingResolver{t}, bindThread(storedBinding(3)))
			if err != nil {
				t.Fatal(err)
			}
			if got.Resolution.Status != sessioninventory.BindingEstablished || got.Resolution.NativeID != bindID(3) || got.Store != nil {
				t.Fatalf("bind = %+v", got)
			}
			assertBindCost(t, rt, bindRel(3))
		})
	}

	rt := bindFixture(t, 8, 3)
	rt.DeleteFile(sessioninventory.Artifact{StorageRoot: bindRoot, RelativePath: bindRel(3)})
	_, err := bindTranscriptWith(context.Background(), rt, listingResolver{t}, bindThread(storedBinding(3)))
	if ResumeDiagnosticOf(err) != ResumeTranscriptMissing {
		t.Fatalf("missing transcript err = %v, want %s", err, ResumeTranscriptMissing)
	}
	if n := rt.OperationCountForRoot(sessioninventorytest.OperationListFiles, bindRoot); n != 0 {
		t.Fatalf("a missing transcript was searched for: %d listings", n)
	}
}

// After pair's own context reset the agent is in a newer conversation the
// record never saw; the ledger names it, and Bind adopts it.
func TestBindAdoptsNewerLedgerRootAfterReset(t *testing.T) {
	t.Parallel()
	rt := bindFixture(t, 8, 3, 5)
	got, err := bindTranscriptWith(context.Background(), rt, listingResolver{t}, bindThread(storedBinding(3)))
	if err != nil {
		t.Fatal(err)
	}
	if got.Resolution.NativeID != bindID(5) || got.Store == nil || got.Store.NativeID != bindID(5) ||
		got.Store.RelativePath != bindRel(5) || got.Store.Source != threadrecord.TranscriptSourceLedger {
		t.Fatalf("adopted bind = %+v store=%+v", got, got.Store)
	}
	assertBindCost(t, rt, bindRel(5))

	for name, setup := range map[string]func(*sessioninventorytest.FakeRuntime){
		"ledger missing": func(rt *sessioninventorytest.FakeRuntime) {
			rt.DeleteFile(sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "ledger-" + bindTag + ".jsonl"})
		},
		"ledger unreadable": func(rt *sessioninventorytest.FakeRuntime) {
			rt.SetError(sessioninventorytest.OperationReadFile, "pair-data:ledger-"+bindTag+".jsonl", fs.ErrPermission)
		},
	} {
		t.Run(name, func(t *testing.T) {
			rt := bindFixture(t, 8, 3, 5)
			setup(rt)
			got, err := bindTranscriptWith(context.Background(), rt, listingResolver{t}, bindThread(storedBinding(3)))
			if err != nil {
				t.Fatal(err)
			}
			if got.Resolution.NativeID != bindID(3) || got.Store != nil {
				t.Fatalf("fallback to the stored binding = %+v", got)
			}
			for _, root := range []string{"pair-data", bindRoot} {
				if n := rt.OperationCountForRoot(sessioninventorytest.OperationListFiles, root); n != 0 {
					t.Fatalf("%s listings = %d, want 0", root, n)
				}
			}
			if n := rt.OperationCountForRoot(sessioninventorytest.OperationStatFile, bindRoot); n != 1 {
				t.Fatalf("transcript stats = %d, want 1", n)
			}
		})
	}
}

func TestLegacyRecordBindsFromLedgerOnceThenStores(t *testing.T) {
	t.Parallel()
	rt := bindFixture(t, 8, 4)
	first, err := bindTranscriptWith(context.Background(), rt, listingResolver{t}, bindThread(nil))
	if err != nil {
		t.Fatal(err)
	}
	if first.Store == nil || first.Store.Source != threadrecord.TranscriptSourceLegacyLedger || first.Store.NativeID != bindID(4) || !first.Store.Established() {
		t.Fatalf("legacy bind store = %+v", first.Store)
	}
	assertBindCost(t, rt, bindRel(4))

	second, err := bindTranscriptWith(context.Background(), rt, listingResolver{t}, bindThread(first.Store))
	if err != nil {
		t.Fatal(err)
	}
	if second.Store != nil || second.Resolution.NativeID != bindID(4) {
		t.Fatalf("second bind = %+v, want the stored binding unchanged", second)
	}
}

// fixedResolver is the listing resolver's answer, counted.
type fixedResolver struct {
	resolution NativeBindingResolution
	calls      *int
}

func (r fixedResolver) ResolveEstablished(context.Context, string, string, string) (NativeBindingResolution, error) {
	*r.calls++
	if code := bindingResumeDiagnostic(r.resolution); code != "" {
		return r.resolution, refuseBinding(code)
	}
	return r.resolution, nil
}

// With no file known yet (a session before its agent's first turn, or a
// binding that knows only its ID), the listing resolver answers once, exactly
// as before the field existed.
func TestBindFallsBackOnceWhenNoFileIsKnown(t *testing.T) {
	t.Parallel()
	rt := bindFixture(t, 8)
	idOnly := &threadrecord.TranscriptBinding{Agent: "claude", NativeID: bindID(6), BoundAt: bindNow.Add(-time.Minute), Source: threadrecord.TranscriptSourceLedger}
	calls := 0
	_, err := bindTranscriptWith(context.Background(), rt, fixedResolver{NativeBindingResolution{Status: sessioninventory.BindingProvisional}, &calls}, bindThread(idOnly))
	if calls != 1 || ResumeDiagnosticOf(err) != ResumeBindingProvisional {
		t.Fatalf("unanswered: calls=%d err=%v", calls, err)
	}
	got, err := bindTranscriptWith(context.Background(), rt, fixedResolver{NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: bindID(6)}, &calls}, bindThread(idOnly))
	if err != nil || got.Resolution.NativeID != bindID(6) || got.Store != nil {
		t.Fatalf("established by the resolver: %+v, %v", got, err)
	}
}

// invalidBinding names a root this build does not know, as a newer build
// adding an agent root would.
func invalidBinding() *threadrecord.TranscriptBinding {
	b := storedBinding(2)
	b.StorageRoot = "claude-projects-v2"
	return b
}

func TestPlanTranscriptBind(t *testing.T) {
	t.Parallel()
	located := ledgerTranscript{NativeID: bindID(2), StorageRoot: bindRoot, RelativePath: bindRel(2)}
	cases := []struct {
		name   string
		stored *threadrecord.TranscriptBinding
		agent  string
		ledger ledgerTranscript
		want   func(transcriptPlan) bool
	}{
		{"stored matches ledger", storedBinding(2), "claude", located, func(p transcriptPlan) bool { return p.Use != nil && !p.Store && p.Use.NativeID == bindID(2) }},
		{"ledger names a newer root", storedBinding(1), "claude", located, func(p transcriptPlan) bool { return p.Store && p.Use.NativeID == bindID(2) }},
		{"ledger conflict refuses", storedBinding(2), "claude", ledgerTranscript{Conflict: true}, func(p transcriptPlan) bool { return p.Refuse == ResumeBindingAmbiguous }},
		{"stored for another agent is ignored", storedBinding(1), "codex", ledgerTranscript{}, func(p transcriptPlan) bool { return p.Fallback }},
		{"ledger root without a proof falls back", nil, "claude", ledgerTranscript{NativeID: bindID(2)}, func(p transcriptPlan) bool { return p.Fallback }},
		{"ledger root without a proof keeps a matching stored file", storedBinding(2), "claude", ledgerTranscript{NativeID: bindID(2)}, func(p transcriptPlan) bool { return p.Use != nil && !p.Store }},
		{"unreadable ledger keeps the stored file", storedBinding(1), "claude", ledgerTranscript{Err: fs.ErrPermission}, func(p transcriptPlan) bool { return p.Use != nil && !p.Store }},
		{"invalid stored binding adopts a located ledger binding", invalidBinding(), "claude", located, func(p transcriptPlan) bool { return p.Store && p.Use.NativeID == bindID(2) }},
		{"invalid stored binding without a ledger refuses", invalidBinding(), "claude", ledgerTranscript{}, func(p transcriptPlan) bool { return p.Refuse == ResumeTranscriptInvalid }},
		{"invalid stored binding with a proofless ledger refuses", invalidBinding(), "claude", ledgerTranscript{NativeID: bindID(2)}, func(p transcriptPlan) bool { return p.Refuse == ResumeTranscriptInvalid }},
		{"nothing known falls back", nil, "claude", ledgerTranscript{}, func(p transcriptPlan) bool { return p.Fallback }},
	}
	for _, c := range cases {
		if got := planTranscriptBind(c.stored, c.agent, c.ledger); !c.want(got) {
			t.Errorf("%s: plan = %+v", c.name, got)
		}
	}
}

// A binding this build cannot open (here a newer build's root name) never
// makes the record unreadable: decode checks types only, and the resume is
// what refuses (pair#695 M1 review BR-5).
func TestInvalidStoredBindingKeepsTheRecordReadable(t *testing.T) {
	t.Parallel()
	store, ns := newTestThreadStore(t)
	record := validThreadRecord(t)
	record.StartingPath, record.WorkingPath = ns.Dir(), ns.Dir()
	record.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	record.Transcript = invalidBinding()
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatalf("CreateThread with a binding this build cannot open: %v", err)
	}
	read, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatalf("a record with an unopenable binding became unreadable: %v", err)
	}
	if read.Transcript == nil || read.Transcript.StorageRoot != "claude-projects-v2" {
		t.Fatalf("binding not kept as written: %+v", read.Transcript)
	}
	rt := bindFixture(t, 4)
	read.Address = ThreadAddress{RepoScope: bindScope, Tag: bindTag}
	if _, err := bindTranscriptWith(context.Background(), rt, listingResolver{t}, read); ResumeDiagnosticOf(err) != ResumeTranscriptInvalid {
		t.Fatalf("resume of an unopenable binding: err = %v, want %s", err, ResumeTranscriptInvalid)
	}
}
