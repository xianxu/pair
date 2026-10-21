package artifactpath

import (
	"path/filepath"
	"testing"
)

func TestRemotePathsStayInOwnedCouchNamespace(t *testing.T) {
	base := t.TempDir()
	paths, err := ResolveCouchRemote(base)
	if err != nil {
		t.Fatal(err)
	}
	if paths.Latch() != filepath.Join(base, "remote-security", "latch.json") || paths.Origins() != filepath.Join(base, "remote-security", "origins.json") || paths.Audit() != filepath.Join(base, "remote-security", "audit.jsonl") || paths.PreviousAudit() != filepath.Join(base, "remote-security", "audit.previous.jsonl") {
		t.Fatalf("wrong authority paths: %+v", paths)
	}
	for _, bad := range []string{"", "relative", base + "/../unsafe"} {
		if _, err := ResolveCouchRemote(bad); err == nil {
			t.Fatalf("accepted unbound root %q", bad)
		}
	}
}

func TestRemoteSecurityNeverJoinsThreadCollection(t *testing.T) {
	root := t.TempDir()
	paths, err := ResolveCouchRemote(root)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := NewStorageOwner(root, "", "thread")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.Latch(), paths.Origins(), paths.Audit(), paths.PreviousAudit()} {
		if _, err := MatchArtifact(path, []StorageOwner{owner}, []string{"codex"}); err == nil {
			t.Fatalf("remote authority admitted to thread GC: %s", path)
		}
	}
	if rule := GCClassifications["remote-security"]; rule.Disposition != GCProtected {
		t.Fatal("kill authority collectable")
	}
}
