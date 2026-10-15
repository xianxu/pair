package sessioninventory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestOwnerLaunchCacheRetainsOnlyConsumedEvidence(t *testing.T) {
	dir := t.TempDir()
	cache := NewOwnerLaunchCache()
	rt := NewOSRuntime(t.TempDir(), dir).WithOwnerLaunchCache(cache)
	const old = "11111111-1111-4111-8111-111111111111"
	const fresh = "22222222-2222-4222-8222-222222222222"
	boundary := sessionledger.LaunchArtifactBoundary{StorageRoot: "claude-projects", RelativePath: strings.Repeat("long-component/", 4096) + "old.jsonl", StableFileID: "inode", MutationToken: "ctime", RawSize: 100}
	launch := sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", RequestedNativeID: old, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{boundary}}
	// The previous binding has a valid scanner state and 4096 distinct artifact
	// proofs; resume consumes none of that historical authority.
	proof := largeOwnerCacheProof(t, old)

	binding := sessionledger.Record{Version: 3, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 1, RootNativeID: old, AuthorizationProof: proof, ConfirmationReason: sessionledger.ConfirmationReason("chosen-id")}
	var raw []byte
	appendRecord := func(r sessionledger.Record) {
		t.Helper()
		b, err := sessionledger.EncodeRecord(r)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, b...)
		raw = append(raw, '\n')
	}
	appendRecord(launch)
	appendRecord(binding)
	launch.RequestedNativeID = fresh
	appendRecord(launch)
	if err := os.WriteFile(filepath.Join(dir, "ledger-work.jsonl"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := QueryResumeTargetContext(context.Background(), rt, "scope", "work", AgentClaude)
	if err != nil || got.NativeID != old || got.FellBackFrom != 3 {
		t.Fatalf("large fixture fallback=%+v %v", got, err)
	}
	var retained int
	for _, entry := range cache.entries {
		b, _ := json.Marshal(entry.current)
		retained += len(b)
		b, _ = json.Marshal(entry.previous)
		retained += len(b)
	}
	if retained > 4096 {
		t.Fatalf("retained %d encoded bytes from %d-byte substantive history; fallback needs identity only", retained, len(raw))
	}
	t.Logf("source=%d retained=%d", len(raw), retained)
}

func TestCompactOwnerLaunchKeepsCurrentProofDropsUnusedPayloads(t *testing.T) {
	proof := &sessionledger.AuthorizationProof{ScannerState: json.RawMessage(`{"retained":"` + strings.Repeat("x", 256<<10) + `"}`)}
	baseline := []sessionledger.LaunchArtifactBoundary{{RelativePath: strings.Repeat("unused/", 16384)}}
	current := sessionledger.Current{Launch: sessionledger.Record{Ordinal: 9, Agent: "codex", RequestedNativeID: "requested", RequestOrigin: sessionledger.RequestOriginChosen, LaunchArtifactBoundaries: baseline, NativeWatermarks: []sessionledger.NativeWatermark{{RootNativeID: strings.Repeat("watermark", 16384)}}}, Binding: &sessionledger.Record{Version: 3, RootNativeID: "root", AuthorizationProof: proof}, Bindings: []sessionledger.Record{{AuthorizationProof: proof, LaunchArtifactBoundaries: baseline}}}
	compact := compactOwnerLaunch(current)
	if compact.Binding == nil || compact.Binding.AuthorizationProof != proof || compact.Binding.RootNativeID != "root" || compact.Launch.Ordinal != 9 {
		t.Fatal("lost consumed current authority")
	}
	if len(compact.Bindings) != 0 || len(compact.Launch.LaunchArtifactBoundaries) != 0 || len(compact.Launch.NativeWatermarks) != 0 {
		t.Fatal("unused current payload retained")
	}
	retained, _ := json.Marshal(compact)
	required, _ := json.Marshal(proof)
	if len(retained) > len(required)+4096 {
		t.Fatalf("retained %d exceeds necessary current proof %d", len(retained), len(required))
	}
	current.Conflict = true
	current.Binding = nil
	compact = compactOwnerLaunch(current)
	if !compact.Conflict || len(compact.Bindings) != 0 {
		t.Fatal("ambiguous status requires flag, not conflicting proof histories")
	}
}

func largeOwnerCacheProof(t *testing.T, id string) *sessionledger.AuthorizationProof {
	t.Helper()
	raw, err := json.Marshal(ScannerState{Version: ScannerStateVersion, Agent: AgentClaude, NativeID: id, IdentityAnchor: id, Role: RoleRoot, ScannerSchema: "claude-v1", FirstRecordValidated: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeScannerState(raw); err != nil {
		t.Fatal(err)
	}
	proof := &sessionledger.AuthorizationProof{Version: 1, RootNativeID: id, ScannerSchema: "claude-v1", ScannerState: raw}
	for i := 0; i < 4096; i++ {
		proof.Artifacts = append(proof.Artifacts, sessionledger.ArtifactProof{StorageRoot: "claude-projects", RelativePath: fmt.Sprintf("project/subagents/agent-%08d.jsonl", i), StableFileID: fmt.Sprintf("inode-%08d", i), MutationToken: "ctime", Size: 100, ParserCompleteOffset: 100})
	}
	if err := sessionledger.ValidateAuthorizationProof(*proof, id); err != nil {
		t.Fatal(err)
	}
	return proof
}
