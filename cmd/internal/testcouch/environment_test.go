package testcouch

import (
	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/internal/reporoot"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFactoryCreatesFreshConfinedRoots(t *testing.T) {
	base := t.TempDir()
	p, err := allocateRoots(base, []string{filepath.Join(base, "live")})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(p.Root)
	if err := ValidateRoots(p); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p.Root)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("unsafe root: %v %v", info, err)
	}
	if len(p.Mutable) < 10 {
		t.Fatal("incomplete runtime roots")
	}
	for name, path := range p.Mutable {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	q, err := allocateRoots(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(q.Root)
	if p.Root == q.Root {
		t.Fatal("factory reused a lease")
	}
}

func TestFactoryRefusesLiveParentBeforeAllocating(t *testing.T) {
	live := t.TempDir()
	if _, err := allocateRoots(live, []string{live}); err == nil {
		t.Fatal("allocated inside live root")
	}
	entries, err := os.ReadDir(live)
	if err != nil || len(entries) != 0 {
		t.Fatalf("refusal changed live root: %v %v", entries, err)
	}
}

func TestEnvironmentDoesNotInheritLiveAuthority(t *testing.T) {
	t.Setenv("COUCH_TREE", "/live/tree")
	t.Setenv("COUCH_CALLER_BINDING", "secret-binding")
	t.Setenv("ANTHROPIC_API_KEY", "secret-key")
	t.Setenv("ZELLIJ_SESSION_NAME", "live")
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(p.Root)
	source := runtimeSourceForTest(t)
	env, err := runtimeEnvironment(p, "/owned/bin", "claude", "run-one", source, "/owned/couch")
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		if _, duplicate := values[key]; duplicate {
			t.Fatalf("runtime environment repeats key %s", key)
		}
		values[key] = value
	}
	for _, key := range []string{"COUCH_TREE", "COUCH_CALLER_BINDING", "ANTHROPIC_API_KEY", "ZELLIJ_SESSION_NAME"} {
		if values[key] != "" {
			t.Fatalf("inherited %s=%q", key, values[key])
		}
	}
	for _, key := range []string{"COUCH_ISOLATED_ROOT", "COUCH_STORE_DIR", "COUCH_IDENTITY_DIR", "COUCH_PAIR_DATA_DIR", "HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR", "TMPDIR", "CODEX_HOME", "CLAUDE_CONFIG_DIR", "WF_SDLC_QUEUE_ROOT", "WF_TEST_LOCK_FILE", "WF_SDLC_QUEUE_FLEET_ROOT", "ZELLIJ_SOCKET_DIR"} {
		if values[key] == "" || !strings.HasPrefix(values[key], p.Root+string(filepath.Separator)) && values[key] != p.Root {
			t.Fatalf("unconfined %s=%q", key, values[key])
		}
	}
	if values["PATH"] != "/owned/bin" {
		t.Fatalf("ambient fallback PATH: %q", values["PATH"])
	}
	if values["PAIR_WRAP_LOG"] != startupLogPath(p) {
		t.Fatalf("wrapper diagnostics are not owned: %q", values["PAIR_WRAP_LOG"])
	}
}

func TestFactoryTerminalIdentityIsolation(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		roots, err := allocateRoots(t.TempDir(), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.RemoveAll(roots.Root); err != nil {
				t.Error(err)
			}
		})
		store := couchidentity.IdentityStore{HostDir: roots.Mutable["identity"], StoreDir: roots.Mutable["store"]}
		allocated, err := store.Allocate(t.Context(), couchidentity.AllocationRequest{Conversation: true, Terminal: true, RepositoryToken: "fixture"})
		if err != nil {
			t.Fatal(err)
		}
		if seen[allocated.SessionName] {
			t.Errorf("independent fleet authorities share native terminal name %q", allocated.SessionName)
		}
		seen[allocated.SessionName] = true
		if allocated.C>>63 != 1 || allocated.N != 1 || allocated.M != 1 {
			t.Errorf("first fleet allocation must have private high C and N=M=1: %+v", allocated)
		}
	}
}

func runtimeSourceForTest(t *testing.T) string {
	t.Helper()
	source, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func TestRuntimeEnvironmentUsesSelectedSourceOutsideCheckout(t *testing.T) {
	source := runtimeSourceForTest(t)
	foreign := t.TempDir()
	if err := os.WriteFile(filepath.Join(foreign, "go.mod"), []byte("module github.com/xianxu/pair\n"), 0600); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(foreign, "tests", "lib", "pair-test-env.sh")
	if err := os.MkdirAll(filepath.Dir(helper), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nprintf ran > \"$0.ran\"\nprintf 'FOREIGN_ENV=wrong\\000'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(p.Root)
	t.Chdir(foreign)
	env, err := runtimeEnvironment(p, "/owned/bin", "claude", "run-selected", source, "/owned/couch")
	if err != nil {
		t.Fatal(err)
	}
	selected := "PAIR_TEST_ENV_SCRIPT=" + filepath.Join(source, "tests", "lib", "pair-test-env.sh")
	found := false
	for _, entry := range env {
		if entry == selected {
			found = true
		}
		if strings.HasPrefix(entry, "FOREIGN_ENV=") {
			t.Fatalf("cwd helper selected: %v", env)
		}
	}
	if !found {
		t.Fatalf("selected helper missing: %v", env)
	}
	if _, err := os.Stat(helper + ".ran"); !os.IsNotExist(err) {
		t.Fatalf("foreign helper executed: %v", err)
	}
}

func TestRuntimeEnvironmentMissingSelectedHelperNeverFallsBack(t *testing.T) {
	p, err := allocateRoots(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(p.Root)
	if _, err := runtimeEnvironment(p, "/owned/bin", "claude", "run-missing", t.TempDir(), "/owned/couch"); err == nil {
		t.Fatal("missing selected helper fell back to current checkout")
	}
}
