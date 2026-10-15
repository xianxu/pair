package reporoot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFind(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/xianxu/pair\n\ngo 1.26\n"), 0600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "cmd", "internal", "example")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	for _, start := range []string{root, nested} {
		got, err := Find(start)
		if err != nil || got != root {
			t.Fatalf("Find(%q) = %q, %v; want %q", start, got, err, root)
		}
	}
	t.Chdir(nested)
	got, err := Current()
	if err != nil || got != root {
		t.Fatalf("Current = %q, %v; want %q", got, err, root)
	}
	// A nested foreign module must not borrow the enclosing checkout's assets.
	if err := os.WriteFile(filepath.Join(nested, "go.mod"), []byte("module example.com/foreign\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Find(nested); err == nil {
		t.Fatal("accepted foreign module")
	}
}

func TestFindRejectsMissingOrUnreadableModule(t *testing.T) {
	for _, mode := range []string{"missing", "directory", "wrong", "prefix", "comment"} {
		t.Run(mode, func(t *testing.T) {
			// Bound ancestry even when TMPDIR lives inside a Pair checkout.
			parent := t.TempDir()
			if err := os.WriteFile(filepath.Join(parent, "go.mod"), []byte("module example.com/foreign\n"), 0600); err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(parent, "nested")
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "go.mod")
			switch mode {
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "wrong", "prefix", "comment":
				content := map[string]string{"wrong": "module example.com/foreign\n", "prefix": "module github.com/xianxu/pair-extra\n", "comment": "// module github.com/xianxu/pair\n"}[mode]
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Find(root); err == nil {
				t.Fatal("accepted invalid checkout")
			}
		})
	}
	if _, err := Find(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("accepted nonexistent start")
	}
}
