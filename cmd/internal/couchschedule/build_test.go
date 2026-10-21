package couchschedule

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestBuildCapturedSourceAndCache(t *testing.T) {
	root := t.TempDir()
	writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
	writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\nfunc main() {}\n")
	builds := 0
	b := Builder{CacheDir: t.TempDir()}
	b.Run = func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		if args[0] == "build" {
			builds++
		}
		return (Builder{}).command(ctx, dir, env, args...)
	}
	a, err := b.Build(context.Background(), root, "task")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Key == "" || a.Path == "" {
		t.Fatalf("empty artifact: %#v", a)
	}
	st, err := os.Stat(a.Path)
	if err != nil || st.Mode()&0111 == 0 {
		t.Fatalf("not executable: %v %v", st, err)
	}
	again, err := b.Build(context.Background(), root, "task")
	if err != nil || (again.Path != a.Path || again.Key != a.Key) {
		t.Fatalf("cache mismatch: %#v %v", again, err)
	}
	defer again.Close()
	if builds != 1 {
		t.Fatalf("cache compiled %d times, want 1", builds)
	}
	_ = a.Close()
	_ = again.Close()
	if err := os.WriteFile(a.Path, []byte("corrupted executable"), 0700); err != nil {
		t.Fatal(err)
	}
	rebuilt, e := b.Build(context.Background(), root, "task")
	if e != nil {
		t.Fatal(e)
	}
	defer rebuilt.Close()
	if builds != 2 {
		t.Fatalf("tampered binary not rebuilt: %d", builds)
	}
}

func TestBuildCapturedEditAndDependencyEmbed(t *testing.T) {
	root := t.TempDir()
	writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\nrequire fixture.local/lib v0.0.0\nreplace fixture.local/lib => ./lib\n")
	writeBuildFixture(t, root, "lib/go.mod", "module fixture.local/lib\n\ngo 1.23\n")
	writeBuildFixture(t, root, "lib/lib.go", "package lib\nimport _ \"embed\"\n//go:embed value.txt\nvar Value string\n")
	writeBuildFixture(t, root, "lib/value.txt", "old")
	writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\nimport (\"fmt\";\"fixture.local/lib\")\nfunc main(){fmt.Print(lib.Value)}\n")
	originalMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	b := Builder{CacheDir: t.TempDir(), AfterCapture: func() { writeBuildFixture(t, root, "lib/value.txt", "new") }}
	a, err := b.Build(context.Background(), root, "task")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	out, err := exec.Command(a.Path).Output()
	if err != nil || string(out) != "old" {
		t.Fatalf("captured artifact: %q %v", out, err)
	}
	b.AfterCapture = nil
	other, err := b.Build(context.Background(), root, "task")
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if other.Key == a.Key {
		t.Fatal("changed embed reused old key")
	}
	out, err = exec.Command(other.Path).Output()
	if err != nil || string(out) != "new" {
		t.Fatalf("changed artifact: %q %v", out, err)
	}
	after, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || string(after) != string(originalMod) {
		t.Fatal("live module file modified")
	}
	if _, err = os.Stat(filepath.Join(root, "go.sum")); !os.IsNotExist(err) {
		t.Fatalf("unexpected live sum: %v", err)
	}
}

func TestBuildRefusesUnsafeInputs(t *testing.T) {
	for _, kind := range []string{"symlink", "escape", "absolute-replace", "cache-symlink", "vendor", "workspace", "not-main"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
			writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\nfunc main(){}\n")
			cache := t.TempDir()
			switch kind {
			case "symlink":
				outside := filepath.Join(t.TempDir(), "main.go")
				if err := os.WriteFile(outside, []byte("package main\nfunc main(){}\n"), 0600); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(root, "scripts/scheduled/task/main.go")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			case "escape", "absolute-replace":
				lib := t.TempDir()
				if kind == "absolute-replace" {
					lib = filepath.Join(root, "lib")
				}
				writeBuildFixture(t, lib, "go.mod", "module fixture.local/lib\n\ngo 1.23\n")
				writeBuildFixture(t, lib, "lib.go", "package lib\nconst V=1\n")
				writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\nrequire fixture.local/lib v0.0.0\nreplace fixture.local/lib => "+lib+"\n")
				writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\nimport _ \"fixture.local/lib\"\nfunc main(){}\n")
			case "vendor":
				if err := os.Mkdir(filepath.Join(root, "vendor"), 0700); err != nil {
					t.Fatal(err)
				}
			case "workspace":
				writeBuildFixture(t, root, "go.work", "go 1.23\nuse .\n")
			case "not-main":
				writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package library\n")
			case "cache-symlink":
				cache = filepath.Join(t.TempDir(), "cache")
				if err := os.Symlink(t.TempDir(), cache); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := (Builder{CacheDir: cache}).Build(context.Background(), root, "task"); err == nil {
				t.Fatal("unsafe build accepted")
			}
		})
	}
}

func TestBuildEnvironmentAndCancellation(t *testing.T) {
	t.Setenv("GOFLAGS", "-race")
	t.Setenv("GOWORK", "host-workspace")
	t.Setenv("GOTOOLCHAIN", "auto")
	env := strings.Join(buildEnvironment(true), "\n")
	for _, want := range []string{"GOFLAGS=\n", "GOWORK=off\n", "GOTOOLCHAIN=local\n", "CGO_ENABLED=0\n", "GOPROXY=off\n"} {
		if !strings.Contains(env+"\n", want) {
			t.Errorf("environment lacks %q", want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Builder{CacheDir: t.TempDir()}).Build(ctx, t.TempDir(), "task"); err == nil {
		t.Fatal("cancelled build accepted")
	}
}

func TestBuildConcurrentSameKeyPublishesOnce(t *testing.T) {
	root := t.TempDir()
	writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
	writeBuildFixture(t, root, "scripts/scheduled/task/main.go", "package main\nfunc main(){}\n")
	var builds atomic.Int32
	b := Builder{CacheDir: t.TempDir()}
	b.Run = func(ctx context.Context, dir string, env, args []string) ([]byte, error) {
		if args[0] == "build" {
			builds.Add(1)
		}
		return (Builder{}).command(ctx, dir, env, args...)
	}
	var rendezvous sync.WaitGroup
	rendezvous.Add(2)
	b.AfterCapture = func() { rendezvous.Done(); rendezvous.Wait() }
	type result struct {
		a   Artifact
		err error
	}
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() { a, err := b.Build(t.Context(), root, "task"); results <- result{a, err} }()
	}
	one, two := <-results, <-results
	defer one.a.Close()
	defer two.a.Close()
	if one.err != nil || two.err != nil || (one.a.Path != two.a.Path || one.a.Key != two.a.Key) || builds.Load() != 1 {
		t.Fatalf("concurrent builds: %#v %#v count=%d", one, two, builds.Load())
	}
}

func TestBuildOutputBoundCannotBypassWrite(t *testing.T) {
	var output buildCapture
	source := struct{ io.Reader }{strings.NewReader(strings.Repeat("x", maxBuildCommandBytes+1))}
	if _, err := io.Copy(&output, source); err != nil {
		t.Fatal(err)
	}
	if !output.overflow || output.Len() != maxBuildCommandBytes {
		t.Fatalf("output bypassed cap: len=%d overflow=%v", output.Len(), output.overflow)
	}
}

func writeBuildFixture(t *testing.T, root, name, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
