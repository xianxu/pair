package testcouch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootPlanRejectsAuthorityOverlapBeforeEffects(t *testing.T) {
	valid := RootsPlan{Root: "/owned/run", Mutable: map[string]string{"home": "/owned/run/home", "queue": "/owned/run/queue"}, Denied: []string{"/live/home", "/live/store"}}
	if err := ValidateRoots(valid); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{"", ".", "/", "/owned/../run", "/live", "/live/home/run", "/live/home"} {
		p := valid
		p.Root = root
		if err := ValidateRoots(p); err == nil {
			t.Errorf("accepted root %q", root)
		}
	}
	for _, path := range []string{"", "relative", "/owned/runaway", "/live/store", "/owned/run/../escape"} {
		p := valid
		p.Mutable = map[string]string{"home": path}
		if err := ValidateRoots(p); err == nil {
			t.Errorf("accepted mutable path %q", path)
		}
	}
}

func TestPhysicalRootsRejectSymlinkEscapeWithoutCreatingAnything(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "home")); err != nil {
		t.Fatal(err)
	}
	p := RootsPlan{Root: root, Mutable: map[string]string{"home": filepath.Join(root, "home", "missing")}}
	if _, err := physicalRoots(p); err == nil {
		t.Fatal("accepted escaping symlink")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("refusal changed outside tree: %v %v", entries, err)
	}
}

func FuzzValidateRoots(f *testing.F) {
	f.Add("/owned/run/home")
	f.Add("/owned/run/../escape")
	f.Fuzz(func(t *testing.T, path string) {
		p := RootsPlan{Root: "/owned/run", Mutable: map[string]string{"home": path}, Denied: []string{"/live"}}
		if ValidateRoots(p) == nil {
			rel, err := filepath.Rel(p.Root, path)
			if err != nil || !filepath.IsLocal(rel) || !filepath.IsAbs(path) || filepath.Clean(path) != path {
				t.Fatalf("unsafe accepted: %q", path)
			}
		}
	})
}
