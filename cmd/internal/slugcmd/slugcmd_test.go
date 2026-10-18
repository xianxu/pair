package slugcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// The body self-gates under Couch hosting (#650): the external Claude Stop
// hook reaches Run through bin/pair-slug without passing pair-wrap, so a
// hosted invocation must no-op before any session lookup or model call. The
// PAIR_SLUG_LOG skip line distinguishes the hosted gate from the no-session
// skip that sits behind it, so each case pins which guard fired.
func TestRunSkipsUnderCouchHosting(t *testing.T) {
	cases := []struct {
		name   string
		hosted bool
		slug   string
		tag    string
		want   string
	}{
		{"hosted skips before the session lookup", true, "", "T", "couch-hosted"},
		{"hosted with PAIR_SLUG=on passes the gate", true, "on", "", "no PAIR_TAG"},
		{"outside couch unchanged", false, "", "", "no PAIR_TAG"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Pin the couch vars in every case: the test process may itself be
			// hosted, and the ambient env must not leak in.
			scope := ""
			if tc.hosted {
				scope = "pair:8"
			}
			t.Setenv("COUCH_THREAD_SCOPE", scope)
			t.Setenv("COUCH_THREAD_TAG", "")
			t.Setenv("PAIR_SLUG", tc.slug)
			t.Setenv("PAIR_TAG", tc.tag)
			t.Setenv("PAIR_DATA_DIR", t.TempDir())
			log := filepath.Join(t.TempDir(), "slug.log")
			t.Setenv("PAIR_SLUG_LOG", log)
			if code := Run(); code != 0 {
				t.Fatalf("Run() = %d, want 0", code)
			}
			b, err := os.ReadFile(log)
			if err != nil {
				t.Fatalf("no skip line logged: %v", err)
			}
			if !strings.Contains(string(b), tc.want) {
				t.Fatalf("log = %q, want it to contain %q", b, tc.want)
			}
		})
	}
}

func TestInventoryTurnsNeverFallsBackFromProvisional(t *testing.T) {
	t.Parallel()
	runtime := slugRuntime(t, false)
	turns, status, err := inventoryTurns(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingProvisional || turns != nil {
		t.Fatalf("turns=%#v status=%s err=%v", turns, status, err)
	}
}

func TestInventoryTurnsUsesEstablishedRoot(t *testing.T) {
	t.Parallel()
	runtime := slugRuntime(t, true)
	turns, status, err := inventoryTurns(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingEstablished || len(turns) != 2 {
		t.Fatalf("turns=%#v status=%s err=%v", turns, status, err)
	}
}

func TestInventoryTurnsStreamsLongEstablishedRoot(t *testing.T) {
	const nativeID = "019e8178-79c2-7862-91db-e8fa1be3b162"
	padding := []byte(`{"type":"future_opaque_event","padding":"` + strings.Repeat("x", 1<<20) + `"}` + "\n")
	content := []byte(`{"timestamp":"2026-05-31T21:36:56Z","type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}` + "\n")
	content = append(content, bytes.Repeat(padding, 33)...)
	content = append(content, []byte(
		`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"late prompt"}]}}`+"\n"+
			`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"late reply"}]}}`+"\n")...)
	runtime := slugRuntimeContent(t, true, content)

	turns, status, err := inventoryTurns(runtime, "scope", "testtag", sessioninventory.AgentCodex)
	if err != nil || status != sessioninventory.BindingEstablished || len(turns) != 2 || turns[0].Text != "late prompt" || turns[1].Text != "late reply" {
		t.Fatalf("turns=%#v status=%s err=%v", turns, status, err)
	}
}

func slugRuntime(t *testing.T, established bool) *sessioninventorytest.FakeRuntime {
	t.Helper()
	const nativeID = "019e8178-79c2-7862-91db-e8fa1be3b162"
	content := []byte(
		`{"timestamp":"2026-05-31T21:36:56Z","type":"session_meta","payload":{"id":"` + nativeID + `","parent_thread_id":null,"source":"cli"}}` + "\n" +
			`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"prompt"}]}}` + "\n" +
			`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"reply"}]}}` + "\n")
	return slugRuntimeContent(t, established, content)
}

func slugRuntimeContent(t *testing.T, established bool, content []byte) *sessioninventorytest.FakeRuntime {
	t.Helper()
	const nativeID = "019e8178-79c2-7862-91db-e8fa1be3b162"
	runtime := sessioninventorytest.NewFakeRuntime()
	nativeRoot := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(nativeRoot)
	artifact := sessioninventory.Artifact{StorageRoot: nativeRoot.Name, RelativePath: "2026/05/31/rollout-test-" + nativeID + ".jsonl"}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", GenerationToken: "gen", MutationToken: "mutation"}, content)
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "testtag", Agent: "codex", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	if err != nil {
		t.Fatal(err)
	}
	ledger := append(launch, '\n')
	if established {
		state, _ := json.Marshal(sessioninventory.ScannerState{Version: 1, Agent: sessioninventory.AgentCodex, NativeID: nativeID, IdentityAnchor: nativeID, Role: sessioninventory.RoleRoot, ScannerSchema: "codex-v1", FirstRecordValidated: true})
		proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: nativeID, ScannerSchema: "codex-v1", ScannerState: state, Artifacts: []sessionledger.ArtifactProof{{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath, StableFileID: "stable", GenerationToken: "gen", MutationToken: "mutation", Size: int64(len(content)), ParserCompleteOffset: int64(len(content))}}}
		binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "testtag", Agent: "codex", LaunchOrdinal: 1, RootNativeID: nativeID, AuthorizationProof: &proof})
		if err != nil {
			t.Fatal(err)
		}
		ledger = append(ledger, append(binding, '\n')...)
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-testtag.jsonl"}}, ledger)
	return runtime
}
