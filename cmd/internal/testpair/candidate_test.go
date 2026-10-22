package testpair

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPlaceCandidateUsesExplicitCheckoutAndEnvironment(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "tests/lib/pair-test-binary.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "pair")
	called := false
	err := PlaceCandidate(t.Context(), Placement{Source: root, Destination: destination, Bash: "/bin/bash", Env: []string{"PATH=/private/tools", "GOENV=off"}}, func(ctx context.Context, program string, args []string, dir string, env []string) (string, error) {
		called = true
		if program != "/bin/bash" || dir != root || len(args) != 2 || args[0] != script || args[1] != destination {
			t.Fatalf("unexpected placement: %s %v %s", program, args, dir)
		}
		for _, item := range env {
			if item == "PAIR_TEST_BINARY=hostile" {
				t.Fatal("ambient candidate leaked")
			}
		}
		if env[len(env)-1] != "PAIR_TEST_BINARY=" {
			t.Fatalf("candidate not explicitly cleared: %v", env)
		}
		return "", nil
	})
	if err != nil || !called {
		t.Fatalf("placement: called=%v err=%v", called, err)
	}
}
