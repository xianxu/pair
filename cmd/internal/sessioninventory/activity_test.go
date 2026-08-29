package sessioninventory_test

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestSessionActivityUsesOnlyEstablishedAuthorizedRoot(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions", Path: "/native/codex"}
	runtime.AddRoot(root)
	created := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	latest := created.Add(5 * time.Minute)
	transcript := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "root.jsonl", Kind: sessioninventory.ArtifactTranscript}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: transcript, ModTime: &latest}, []byte("{}\n"))
	node := sessioninventory.Node{Agent: sessioninventory.AgentCodex, Time: &sessioninventory.NativeTime{Value: created, Source: sessioninventory.TimeSourceMetadata}, Artifacts: []sessioninventory.Artifact{transcript}}

	for _, status := range []sessioninventory.BindingStatus{sessioninventory.BindingUnbound, sessioninventory.BindingProvisional, sessioninventory.BindingAmbiguous} {
		if got, ok, err := sessioninventory.ActivityForSession(runtime, sessioninventory.SessionQuery{Status: status, Root: &node}); err != nil || ok || got != (sessioninventory.SessionActivity{}) {
			t.Fatalf("%s activity = %#v,%v err=%v", status, got, ok, err)
		}
	}
	got, ok, err := sessioninventory.ActivityForSession(runtime, sessioninventory.SessionQuery{Status: sessioninventory.BindingEstablished, Root: &node})
	if err != nil || !ok || !got.CreatedAt.Equal(created) || !got.LastActivityAt.Equal(latest) || got.CreatedTimeSource != sessioninventory.TimeSourceMetadata || got.LastActivityTimeSource != sessioninventory.TimeSourceMTime {
		t.Fatalf("established activity = %#v,%v err=%v", got, ok, err)
	}
}
