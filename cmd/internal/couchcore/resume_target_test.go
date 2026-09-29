package couchcore

import (
	"context"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

// Exercise the production resolver: admission must not depend on a native file
// being readable, or on optional transcript proof surviving a reboot.
func TestNativeResolverRetainsProbationTargetWithoutTranscript(t *testing.T) {
	runtime := sessioninventorytest.NewFakeRuntime()
	runtime.SetPairDataRoot(sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"})
	artifact := sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "ledger-work.jsonl"}
	launch := `{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"A","request_origin":"resume","baseline_complete":false}` + "\n"
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, []byte(launch))
	resolver := SessionInventoryNativeBindingResolver{Runtime: runtime}
	got, err := resolver.ResolveEstablished(context.Background(), "scope", "work", "codex")
	if err != nil || got.Status != sessioninventory.BindingProvisional || got.NativeID != "A" {
		t.Fatalf("probation target=%+v error=%v", got, err)
	}
	runtime.AppendFile(artifact, []byte(`{"v":3,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"D","confirmation_reason":"correlation"}`+"\n"), "new")
	got, err = resolver.ResolveEstablished(context.Background(), "scope", "work", "codex")
	if err != nil || got.Status != sessioninventory.BindingEstablished || got.NativeID != "D" {
		t.Fatalf("confirmed target=%+v error=%v", got, err)
	}
}
