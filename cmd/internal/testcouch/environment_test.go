package testcouch

import (
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
	env, err := runtimeEnvironment(p, "/owned/bin", "claude", "run-one", "/owned/source", "/owned/couch")
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
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
