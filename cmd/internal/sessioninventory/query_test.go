package sessioninventory_test

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
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
	if query.Status != sessioninventory.BindingEstablished || query.Root == nil || query.Root.NativeID != nativeID {
		t.Fatalf("established query = %#v", query)
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
	if err != nil || query.Status != sessioninventory.BindingEstablished || query.Root == nil {
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
			`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[{"root_native_id":"a","root_native_id":"b","event_position":1}]}`+"\n"))

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
	if malformed != 6 {
		t.Fatalf("malformed=%d diagnostics=%#v", malformed, query.Diagnostics)
	}
}
