package sessioninventory_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestQuerySessionRequiresEstablishedBinding(t *testing.T) {
	t.Parallel()
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	runtime := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(nativeRoot)
	transcript := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/08/28/rollout-test-" + nativeID + ".jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: transcript}, []byte(
		`{"timestamp":"2026-08-28T10:04:00Z","type":"session_meta","payload":{"id":"`+nativeID+`","parent_thread_id":null,"source":"cli"}}`+"\n"))
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}`+"\n"))

	query, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil {
		t.Fatal(err)
	}
	if query.Status != sessioninventory.BindingProvisional || query.Root != nil {
		t.Fatalf("provisional query = %#v", query)
	}

	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}`+"\n"+
			`{"v":1,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"`+nativeID+`"}`+"\n"))
	query, err = sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil {
		t.Fatal(err)
	}
	if query.Status != sessioninventory.BindingProvisional || query.Root != nil || !hasDiagnostic(query.Diagnostics, sessioninventory.DiagnosticBindingStale) {
		t.Fatalf("proofless query = %#v", query)
	}
}

func TestQuerySessionEstablishesProofAfterCatalogLossWithoutTranscriptRead(t *testing.T) {
	t.Parallel()
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	runtime, transcript, ledger := proofBackedCodexFixture(t, nativeID, nil)

	query, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || query.Status != sessioninventory.BindingEstablished || query.Root == nil || query.Root.NativeID != nativeID {
		t.Fatalf("query=%#v err=%v", query, err)
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadAt, transcript.StorageRoot+":"+transcript.RelativePath); got != 0 {
		t.Fatalf("transcript ReadAt count=%d, want 0", got)
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadFile, ledger.StorageRoot+":"+ledger.RelativePath); got != 1 {
		t.Fatalf("ledger ReadFile count=%d, want 1", got)
	}
}

func TestQuerySessionPersistsAppendAdvancementAcrossQueries(t *testing.T) {
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	runtime, transcript, _ := proofBackedCodexFixture(t, nativeID, nil)
	runtime.AppendFile(transcript, []byte(
		`{"timestamp":"2026-08-28T10:05:00Z","type":"event_msg","payload":{"type":"user_message","message":"next"}}`+"\n"), "ctime:2")

	first, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || first.Status != sessioninventory.BindingEstablished || first.Root == nil {
		t.Fatalf("first query=%#v err=%v", first, err)
	}
	readsAfterFirst := runtime.OperationCount(sessioninventorytest.OperationReadAt, transcript.StorageRoot+":"+transcript.RelativePath)
	if readsAfterFirst == 0 {
		t.Fatal("first appended query performed no suffix read")
	}

	second, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || second.Status != sessioninventory.BindingEstablished || second.Root == nil {
		t.Fatalf("second query=%#v err=%v", second, err)
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadAt, transcript.StorageRoot+":"+transcript.RelativePath); got != readsAfterFirst {
		t.Fatalf("second unchanged query repeated body reads: before=%d after=%d", readsAfterFirst, got)
	}
}

func TestQuerySessionRevalidatesAndCachesGrowthWithoutGenerationToken(t *testing.T) {
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	runtime, transcript, _ := proofBackedCodexFixtureWithGeneration(t, nativeID, nil, "")
	primed, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || primed.Status != sessioninventory.BindingEstablished || primed.Root == nil {
		t.Fatalf("prime query=%#v err=%v", primed, err)
	}
	runtime.AppendFile(transcript, []byte(
		`{"timestamp":"2026-08-28T10:05:00Z","type":"event_msg","payload":{"type":"agent_message","message":"done"}}`+"\n"), "ctime:2")

	first, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || first.Status != sessioninventory.BindingEstablished || first.Root == nil || first.Root.NativeID != nativeID {
		t.Fatalf("first query=%#v err=%v", first, err)
	}
	readsAfterFirst := runtime.OperationCount(sessioninventorytest.OperationReadAt, transcript.StorageRoot+":"+transcript.RelativePath)
	if readsAfterFirst == 0 {
		t.Fatal("first grown query did not revalidate the exact transcript")
	}

	second, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || second.Status != sessioninventory.BindingEstablished || second.Root == nil || second.Root.NativeID != nativeID {
		t.Fatalf("second query=%#v err=%v", second, err)
	}
	if got := runtime.OperationCount(sessioninventorytest.OperationReadAt, transcript.StorageRoot+":"+transcript.RelativePath); got != readsAfterFirst {
		t.Fatalf("cached current validation repeated body reads: before=%d after=%d", readsAfterFirst, got)
	}
}

func TestOwnerCLIUsesBoundedPersistentQuery(t *testing.T) {
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	runtime, _, _ := proofBackedCodexFixture(t, nativeID, nil)
	getenv := func(key string) string {
		if key == "PAIR_SCOPE_KEY" {
			return "scope"
		}
		return ""
	}
	var stdout, stderr strings.Builder
	code := sessioninventory.RunCLIWithRuntime([]string{"--agent", "codex", "--scope", "current", "--owner", "work"}, getenv, runtime, &stdout, &stderr)
	if code != 0 || stdout.String() != nativeID+"\n" || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if reads := runtime.OperationCount(sessioninventorytest.OperationReadAt, ""); reads != 0 {
		t.Fatalf("owner query body reads=%d, want 0 for unchanged proof", reads)
	}
}

func TestQuerySessionCatalogLossProofClassCoversEveryAgentWithoutBodyReads(t *testing.T) {
	t.Parallel()
	const id = "55555555-5555-4555-8555-555555555555"
	for _, test := range []struct {
		agent     sessioninventory.Agent
		schema    string
		artifacts []sessioninventory.Artifact
	}{
		{sessioninventory.AgentClaude, "claude-v1", []sessioninventory.Artifact{{StorageRoot: "claude-projects", RelativePath: "-repo/" + id + ".jsonl"}}},
		{sessioninventory.AgentCodex, "codex-v1", []sessioninventory.Artifact{{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-test-" + id + ".jsonl"}}},
		{sessioninventory.AgentMuse, "muse-v1", []sessioninventory.Artifact{{StorageRoot: "muse-sessions", RelativePath: "2026/08/28/" + id + "/session.jsonl"}}},
		{sessioninventory.AgentAgy, "agy-v1", []sessioninventory.Artifact{
			{StorageRoot: "agy-conversations", RelativePath: id + ".db"},
			{StorageRoot: "agy-brain", RelativePath: id + "/.system_generated/logs/transcript.jsonl"},
		}},
	} {
		test := test
		t.Run(string(test.agent), func(t *testing.T) {
			runtime := sessioninventorytest.NewFakeRuntime()
			for _, rootName := range []string{"claude-projects", "codex-sessions", "muse-sessions", "agy-conversations", "agy-brain"} {
				for _, artifact := range test.artifacts {
					if artifact.StorageRoot == rootName {
						runtime.AddRoot(sessioninventory.StorageRoot{Agent: test.agent, Name: rootName, Path: "/native/" + rootName})
					}
				}
			}
			state, err := json.Marshal(sessioninventory.ScannerState{Version: 1, Agent: test.agent, NativeID: id, IdentityAnchor: id, Role: sessioninventory.RoleRoot, ScannerSchema: test.schema, FirstRecordValidated: true})
			if err != nil {
				t.Fatal(err)
			}
			proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: id, ScannerSchema: test.schema, ScannerState: state}
			for i, artifact := range test.artifacts {
				content := []byte("already validated\n")
				stable := sessioninventory.StableFileID("stable-" + string(rune('a'+i)))
				generation := sessioninventory.GenerationToken("gen-" + string(rune('a'+i)))
				mutation := sessioninventory.MutationToken("mutation-" + string(rune('a'+i)))
				runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: stable, GenerationToken: generation, MutationToken: mutation}, content)
				proof.Artifacts = append(proof.Artifacts, sessionledger.ArtifactProof{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath, StableFileID: string(stable), GenerationToken: string(generation), MutationToken: string(mutation), Size: int64(len(content)), ParserCompleteOffset: int64(len(content))})
			}
			pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
			runtime.SetPairDataRoot(pairRoot)
			ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
			launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: string(test.agent), LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
			if err != nil {
				t.Fatal(err)
			}
			binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: string(test.agent), LaunchOrdinal: 1, RootNativeID: id, AuthorizationProof: &proof})
			if err != nil {
				t.Fatal(err)
			}
			runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, append(append(launch, '\n'), append(binding, '\n')...))

			query, err := sessioninventory.QuerySession(runtime, "scope", "work", test.agent)
			if err != nil || query.Status != sessioninventory.BindingEstablished || query.Root == nil || query.Root.NativeID != id {
				t.Fatalf("query=%#v err=%v", query, err)
			}
			if reads := runtime.OperationCount(sessioninventorytest.OperationReadAt, ""); reads != 0 {
				t.Fatalf("body reads=%d, want 0", reads)
			}
			if queries := runtime.OperationCount(sessioninventorytest.OperationSQLite, ""); queries != 0 {
				t.Fatalf("sqlite queries=%d, want 0", queries)
			}
		})
	}
}

func TestSessionForOwnerPreservesAmbiguousAndUnbound(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		inventory  sessioninventory.Inventory
		wantStatus sessioninventory.BindingStatus
	}{
		{"unbound", sessioninventory.Inventory{}, sessioninventory.BindingUnbound},
		{"ambiguous", sessioninventory.Inventory{Bindings: []sessioninventory.Binding{{ScopeKey: "scope", Tag: "work", Agent: sessioninventory.AgentCodex, Status: sessioninventory.BindingAmbiguous}}}, sessioninventory.BindingAmbiguous},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := sessioninventory.SessionForOwner(test.inventory, "scope", "work", sessioninventory.AgentCodex)
			if got.Status != test.wantStatus || got.Root != nil {
				t.Fatalf("query = %#v", got)
			}
		})
	}
}

func TestTokenUsageForRootReadsOnlyAuthorizedTranscript(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(root)
	transcript := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "root.jsonl", Kind: sessioninventory.ArtifactTranscript}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: transcript}, []byte(
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":40}}}}`+"\n"+
			`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":60}}}}`+"\n"))
	node := sessioninventory.Node{Agent: sessioninventory.AgentCodex, Artifacts: []sessioninventory.Artifact{
		{StorageRoot: root.Name, RelativePath: "state.json", Kind: sessioninventory.ArtifactMetadata},
		transcript,
	}}
	usage, ok, err := sessioninventory.TokenUsageForRoot(runtime, node)
	if err != nil || !ok || usage.InputTokens != 60 {
		t.Fatalf("usage = %#v, %v, err=%v", usage, ok, err)
	}
}

func TestTokenUsageForRootReadsRecordsAfterThirtyTwoMiB(t *testing.T) {
	runtime := sessioninventorytest.NewFakeRuntime()
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "long.jsonl", Kind: sessioninventory.ArtifactTranscript}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, longCodexTranscript(
		`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":77}}}}`+"\n"))
	node := sessioninventory.Node{Agent: sessioninventory.AgentCodex, Artifacts: []sessioninventory.Artifact{artifact}}

	usage, ok, err := sessioninventory.TokenUsageForRoot(runtime, node)
	if err != nil || !ok || usage.InputTokens != 77 {
		t.Fatalf("usage=%#v ok=%v err=%v", usage, ok, err)
	}
}

func TestQuerySessionPreservesValidPairFilesFromPartialListing(t *testing.T) {
	const nativeID = "019d1111-1111-7111-8111-111111111111"
	runtime := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(nativeRoot)
	transcript := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/08/28/rollout-test-" + nativeID + ".jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: transcript}, []byte(
		`{"timestamp":"2026-08-28T10:04:00Z","type":"session_meta","payload":{"id":"`+nativeID+`","parent_thread_id":null,"source":"cli"}}`+"\n"))
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}`+"\n"+
			`{"v":1,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"`+nativeID+`"}`+"\n"))
	rejected := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "config-bad-codex.json"}
	runtime.SetError(sessioninventorytest.OperationListFiles, pairRoot.Name, &sessioninventory.ListingIssuesError{Artifacts: []sessioninventory.Artifact{rejected}})

	query, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || query.Status != sessioninventory.BindingProvisional || query.Root != nil {
		t.Fatalf("query=%#v err=%v", query, err)
	}
	found := false
	for _, diagnostic := range query.Diagnostics {
		if diagnostic.Code == sessioninventory.DiagnosticArtifactPathInvalid {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics=%#v", query.Diagnostics)
	}
}

func hasDiagnostic(diagnostics []sessioninventory.Diagnostic, code sessioninventory.DiagnosticCode) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func proofBackedCodexFixture(t *testing.T, nativeID string, modTime *time.Time) (*sessioninventorytest.FakeRuntime, sessioninventory.Artifact, sessioninventory.Artifact) {
	return proofBackedCodexFixtureWithGeneration(t, nativeID, modTime, "gen:1")
}

func proofBackedCodexFixtureWithGeneration(t *testing.T, nativeID string, modTime *time.Time, generation sessioninventory.GenerationToken) (*sessioninventorytest.FakeRuntime, sessioninventory.Artifact, sessioninventory.Artifact) {
	t.Helper()
	runtime := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(nativeRoot)
	transcript := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/08/28/rollout-test-" + nativeID + ".jsonl"}
	content := []byte(`{"timestamp":"2026-08-28T10:04:00Z","type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}` + "\n")
	entry := sessioninventory.FileEntry{Artifact: transcript, StableFileID: "dev:1/ino:1", GenerationToken: generation, MutationToken: "ctime:1", ModTime: modTime}
	runtime.PutFile(entry, content)

	state, err := json.Marshal(sessioninventory.ScannerState{
		Version: sessioninventory.ScannerStateVersion, Agent: sessioninventory.AgentCodex, NativeID: nativeID,
		IdentityAnchor: nativeID, Role: sessioninventory.RoleRoot, ScannerSchema: "codex-v1", FirstRecordValidated: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: nativeID, ScannerSchema: "codex-v1", ScannerState: state, Artifacts: []sessionledger.ArtifactProof{{
		StorageRoot: transcript.StorageRoot, RelativePath: transcript.RelativePath, StableFileID: "dev:1/ino:1", GenerationToken: string(generation), MutationToken: "ctime:1", Size: int64(len(content)), ParserCompleteOffset: int64(len(content)),
	}}}
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 1, RootNativeID: nativeID, AuthorizationProof: &proof})
	if err != nil {
		t.Fatal(err)
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, append(append(launch, '\n'), append(binding, '\n')...))
	return runtime, transcript, ledger
}

func TestQuerySessionDoesNotDiagnoseCompatibilityLedgerRowsAsMalformed(t *testing.T) {
	runtime := sessioninventorytest.NewFakeRuntime()
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(
		`{"agent":"codex","args":[],"session_id":"legacy","started":"2026-08-28T00:00:00Z","last_active":"2026-08-28T00:00:00Z","repo_root":"/repo","repo_name":"pair"}`+"\n"+
			`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}`+"\n"))

	query, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range query.Diagnostics {
		if diagnostic.Code == sessioninventory.DiagnosticPairRecordMalformed {
			t.Fatalf("compatibility row diagnosed as malformed: %#v", query.Diagnostics)
		}
	}
}

func TestQuerySessionDiagnosesInvalidCompatibilityRows(t *testing.T) {
	runtime := sessioninventorytest.NewFakeRuntime()
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	valid := `{"agent":"codex","args":[],"session_id":"legacy","started":"2026-08-28T00:00:00Z","last_active":"2026-08-28T00:00:00Z","repo_root":"/repo","repo_name":"pair"}`
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}}, []byte(
		valid+"\n"+
			`{"agent":"codex","session_id":"partial"}`+"\n"+
			strings.TrimSuffix(valid, "}")+`,"extra":true}`+"\n"+
			strings.Replace(valid, `"codex"`, `"unsupported"`, 1)+"\n"+
			strings.Replace(valid, `"agent":"codex"`, `"agent":"future","agent":"codex"`, 1)+"\n"+
			`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"future","agent":"codex","pair_log_offset":0,"native_watermarks":[]}`+"\n"+
			`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[{"root_native_id":"a","root_native_id":"b","event_position":1}]}`+"\n"+
			`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[{"root_native_id":"a"}]}`+"\n"+
			`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[{"root_native_id":"a","event_position":null}]}`+"\n"+
			`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[],"root_native_id":null}`+"\n"+
			strings.TrimSuffix(valid, "}")+`,"legacy_import":null}`+"\n"))

	query, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil {
		t.Fatal(err)
	}
	malformed := 0
	for _, diagnostic := range query.Diagnostics {
		if diagnostic.Code == sessioninventory.DiagnosticPairRecordMalformed {
			malformed++
		}
	}
	if malformed != 10 {
		t.Fatalf("malformed=%d diagnostics=%#v", malformed, query.Diagnostics)
	}
}
