package artifactpath

import (
	"path/filepath"
	"testing"
)

func TestFullscreenPathsAndGC(t *testing.T) {
	root := t.TempDir()
	owner, _ := NewStorageOwner(root, "repo", "thread")
	p, _ := ResolveScoped(owner.Directory(), owner.Tag)
	seen := map[string]bool{}
	for _, scope := range []string{"repo", "other"} {
		for _, tag := range []string{"thread", "neighbor"} {
			q, err := Resolve(Address{DataDir: root, RepoScope: scope, Tag: tag})
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{q.FullscreenReturn(), q.FullscreenLock(), q.FullscreenDiagnostics()} {
				if seen[path] {
					t.Fatal("path collision", path)
				}
				seen[path] = true
			}
		}
	}
	for _, tc := range []struct {
		path, family string
		retention    RetentionClass
	}{
		{p.FullscreenReturn(), "fullscreen-return", SessionRetention},
		{p.FullscreenLock(), "fullscreen-lock", SessionRetention},
		{p.FullscreenDiagnostics(), "fullscreen-diagnostics", DebugRetention},
	} {
		m, err := MatchArtifact(tc.path, []StorageOwner{owner}, nil)
		if err != nil || m.Family != tc.family || m.Retention != tc.retention {
			t.Fatalf("match %s = %+v, %v", tc.path, m, err)
		}
		if GCClassifications[tc.family].Disposition != GCCollectable {
			t.Fatal("not collectable", tc.family)
		}
	}
	for _, name := range []string{filepath.Base(p.FullscreenDiagnostics()), filepath.Base(p.FullscreenDiagnostics()) + ".pair-diagnostics.lock"} {
		owners, err := DiscoverStorageOwners(root, "repo", []string{name})
		if err != nil || len(owners) != 1 || owners[0] != owner {
			t.Fatalf("discover %s = %+v, %v", name, owners, err)
		}
	}
}

func TestFullscreenEnvironmentBindings(t *testing.T) {
	p, err := ResolveScoped(t.TempDir(), "thread")
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := p.EnvironmentBindings("codex")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, binding := range bindings {
		got[binding.Name] = binding.Path
	}
	for name, want := range map[string]string{
		"PAIR_FULLSCREEN_RETURN_PATH":      p.FullscreenReturn(),
		"PAIR_FULLSCREEN_LOCK_PATH":        p.FullscreenLock(),
		"PAIR_FULLSCREEN_DIAGNOSTICS_PATH": p.FullscreenDiagnostics(),
	} {
		if got[name] != want {
			t.Errorf("%s = %q, want %q", name, got[name], want)
		}
	}
}
