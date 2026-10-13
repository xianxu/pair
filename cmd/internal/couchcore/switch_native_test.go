package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"testing"
)

func TestSwitchRecoveryExactNativeSelectionDoesNotUseLatestLedger(t *testing.T) {
	rt := sessioninventorytest.NewFakeRuntime()
	rt.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"})
	old := "019d1111-1111-7111-8111-111111111111"
	artifact := sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/08/28/rollout-test-" + old + ".jsonl"}
	rt.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "old", GenerationToken: "old", MutationToken: "old"}, []byte(`{"type":"session_meta","payload":{"id":"`+old+`","parent_thread_id":null,"source":"cli"}}`+"\n"))
	resolver := SessionInventoryNativeBindingResolver{Runtime: rt}
	got, err := resolver.ResolveExact(context.Background(), "scope", "tag", "codex", old)
	if err != nil || got.NativeID != old || got.Status != sessioninventory.BindingEstablished {
		t.Fatalf("saved native unavailable without latest ledger: %+v %v", got, err)
	}
	if _, err := resolver.ResolveExact(context.Background(), "scope", "tag", "codex", "019d2222-2222-7222-8222-222222222222"); err == nil {
		t.Fatal("missing native session accepted")
	}
}
