package testcouch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildEnvironmentRejectsAmbientConfiguration(t *testing.T) {
	t.Setenv("PAIR_TEST_BINARY", "/foreign/pair")
	t.Setenv("GOFLAGS", "-toolexec=/foreign/tool")
	t.Setenv("HTTPS_PROXY", "https://foreign")
	root := t.TempDir()
	env, err := buildEnvironment(root, filepath.Join(root, "tools"), goCaches{GOCACHE: root, GOMODCACHE: root})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(env, "\n")
	for _, bad := range []string{"foreign", "HTTPS_PROXY", "PAIR_TEST_BINARY"} {
		if strings.Contains(got, bad) {
			t.Fatalf("leak %s: %s", bad, got)
		}
	}
	for _, want := range []string{"GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOFLAGS=-mod=readonly", "HOME=" + filepath.Join(root, "home")} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s: %s", want, got)
		}
	}
}

func TestBuildOwnedExplicitCheckoutAndPartialCleanup(t *testing.T) {
	source := t.TempDir()
	for _, name := range []string{"go.mod", "tests/lib/pair-test-binary.sh", "cmd/couch/main.go"} {
		p := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	scratch := t.TempDir()
	calls := 0
	failure := errors.New("offline dependency unavailable")
	deps := buildDependencies{resolve: func(string) (string, error) { return "/bin/sh", nil }, run: func(ctx context.Context, program string, args []string, dir string, env []string) (string, error) {
		calls++
		if dir != source {
			t.Fatalf("source substituted: %s", dir)
		}
		if len(args) > 0 && args[0] == "env" {
			if !strings.Contains(strings.Join(env, "\n"), "XDG_CONFIG_HOME="+scratch+string(filepath.Separator)) {
				t.Fatalf("metadata query config is not private: %v", env)
			}
			return `{"GOCACHE":"` + scratch + `","GOMODCACHE":"` + scratch + `"}`, nil
		}
		if !strings.Contains(strings.Join(env, "\n"), "GOPROXY=off") {
			t.Fatalf("online build: %v", env)
		}
		return "offline", failure
	}}
	owner, err := buildOwned(t.Context(), BuildConfig{Source: source, ScratchParent: scratch}, deps)
	if owner == nil || !errors.Is(err, failure) || calls != 2 {
		t.Fatalf("owner=%v err=%v calls=%d", owner, err, calls)
	}
	if _, err := os.Stat(owner.Binaries.Dir); err != nil {
		t.Fatalf("partial owner lost: %v", err)
	}
	if err := owner.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(owner.Binaries.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owner not removed: %v", err)
	}
}

func TestBuildOwnedUsesOnlySelectedCandidateAndOfflineCommands(t *testing.T) {
	t.Setenv("PAIR_TEST_BINARY", "/foreign/installed-pair")
	t.Setenv("GOFLAGS", "-toolexec=/foreign/compiler")
	t.Setenv("HTTPS_PROXY", "https://foreign")
	source := t.TempDir()
	for _, name := range []string{"go.mod", "tests/lib/pair-test-binary.sh", "cmd/couch/main.go"} {
		p := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	scratch := t.TempDir()
	override := filepath.Join(scratch, "exact-couch")
	if err := os.WriteFile(override, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	selectedTools := map[string]bool{}
	builds := []string{}
	pairBuilds := 0
	deps := buildDependencies{resolve: func(name string) (string, error) {
		selectedTools[name] = true
		if name == override {
			return override, nil
		}
		return "/bin/sh", nil
	}, run: func(ctx context.Context, program string, args []string, dir string, env []string) (string, error) {
		if dir != source {
			t.Fatalf("wrong checkout %s", dir)
		}
		text := strings.Join(env, "\n")
		if strings.Contains(text, "foreign") {
			t.Fatalf("ambient configuration leaked: %s", text)
		}
		if len(args) > 0 && args[0] == "env" {
			return `{"GOCACHE":"` + scratch + `","GOMODCACHE":"` + scratch + `"}`, nil
		}
		if !strings.Contains(text, "GOPROXY=off") || !strings.Contains(text, "GOFLAGS=-mod=readonly") {
			t.Fatalf("build not offline: %s", text)
		}
		output := ""
		if len(args) > 3 && args[0] == "build" {
			output = args[2]
			builds = append(builds, args[3])
		}
		if len(args) == 2 && args[0] == filepath.Join(source, "tests/lib/pair-test-binary.sh") {
			pairBuilds++
			output = args[1]
		}
		if output != "" {
			if err := os.WriteFile(output, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
				return "", err
			}
		}
		if len(args) > 0 && args[0] == "rev-parse" {
			return "selected-head\n", nil
		}
		return "fixture version\n", nil
	}}
	owner, err := buildOwned(t.Context(), BuildConfig{Source: source, ScratchParent: scratch, Couch: override}, deps)
	if owner != nil {
		t.Cleanup(func() {
			if err := owner.Close(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	if owner.Binaries.Couch != override || owner.Binaries.Nvim != "/bin/sh" || owner.Binaries.Evidence["source-head"] != "selected-head" {
		t.Fatalf("candidate provenance: %+v", owner.Binaries)
	}
	for _, name := range []string{"dirname", "cc", "c++"} {
		if !selectedTools[name] {
			t.Fatalf("build utility not selected once: %s", name)
		}
	}
	if strings.Contains(strings.Join(builds, " "), "./cmd/couch") || len(builds) != 2 || pairBuilds != 1 {
		t.Fatalf("wrong builders %v pair=%d", builds, pairBuilds)
	}
}

func TestBuildOwnedCancellationPreservesPreparationOwner(t *testing.T) {
	source := t.TempDir()
	for _, name := range []string{"go.mod", "tests/lib/pair-test-binary.sh", "cmd/couch/main.go"} {
		p := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	scratch := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	deps := buildDependencies{resolve: func(string) (string, error) { return "/bin/sh", nil }, run: func(ctx context.Context, program string, args []string, dir string, env []string) (string, error) {
		if len(args) > 0 && args[0] == "env" {
			return `{"GOCACHE":"` + scratch + `","GOMODCACHE":"` + scratch + `"}`, nil
		}
		cancel()
		return "", ctx.Err()
	}}
	owner, err := buildOwned(ctx, BuildConfig{Source: source, ScratchParent: scratch}, deps)
	if owner == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("partial cancelled owner=%v error=%v", owner, err)
	}
	if err := owner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBuildOwnedRejectsRelativeCouchOverrideBeforeToolLookup(t *testing.T) {
	source := t.TempDir()
	for _, name := range []string{"go.mod", "tests/lib/pair-test-binary.sh", "cmd/couch/main.go"} {
		p := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	lookedUp := false
	_, err := buildOwned(t.Context(), BuildConfig{Source: source, ScratchParent: t.TempDir(), Couch: "couch"}, buildDependencies{resolve: func(string) (string, error) { lookedUp = true; return "", errors.New("unexpected tool lookup") }})
	if err == nil || lookedUp {
		t.Fatalf("relative Couch override used invocation lookup: err=%v lookedUp=%v", err, lookedUp)
	}
}

func TestBuildOwnedRefusesScratchOverlapBeforeAnyToolOrCreation(t *testing.T) {
	source := t.TempDir()
	for _, name := range []string{"go.mod", "tests/lib/pair-test-binary.sh", "cmd/couch/main.go"} {
		p := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	live := t.TempDir()
	t.Setenv("HOME", live)
	scratch := filepath.Join(live, "scratch")
	if err := os.Mkdir(scratch, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "scratch-alias")
	if err := os.Symlink(scratch, alias); err != nil {
		t.Fatal(err)
	}
	for _, parent := range []string{scratch, alias, source} {
		looked := false
		_, err := buildOwned(t.Context(), BuildConfig{Source: source, ScratchParent: parent}, buildDependencies{resolve: func(string) (string, error) { looked = true; return "", errors.New("unexpected tool lookup") }})
		if err == nil || looked {
			t.Fatalf("unsafe scratch %s reached tool lookup: %v", parent, err)
		}
	}
	entries, err := os.ReadDir(scratch)
	if err != nil || len(entries) != 0 {
		t.Fatalf("scratch changed: %v %v", entries, err)
	}
}
