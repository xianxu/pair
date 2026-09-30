package artifactpath

import (
	"path/filepath"
	"testing"
)

func TestSessionArtifactOwnerExactPaths(t *testing.T) {
	root := "/tmp/Pair data"
	scope := "0123456789abcdef"
	p, err := Resolve(Address{DataDir: root, RepoScope: scope, Tag: "1-repo-10"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{p.Draft(), p.ScrollbackRaw("claude"), p.ScrollbackRaw("codex")} {
		got, ok := SessionArtifactOwner(root, path, []string{"claude", "codex"})
		if !ok || got.Tag != "1-repo-10" || got.RepoScope != scope {
			t.Fatalf("%s: %+v %v", path, got, ok)
		}
	}
	for _, path := range []string{p.Draft() + ".bak", filepath.Join(root+"-other", "repos", scope, "draft-1-repo-10.md"), filepath.Join(root, "repos", scope, "nested", "draft-1-repo-10.md"), root + "/repos/" + scope + "/../" + scope + "/draft-1-repo-10.md", p.ScrollbackRaw("unsupported")} {
		if got, ok := SessionArtifactOwner(root, path, []string{"claude", "codex"}); ok {
			t.Fatalf("invalid path %s accepted: %+v", path, got)
		}
	}
}
