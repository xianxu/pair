//go:build darwin && cgo

package testdisk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

// Exercise cmd/go itself: a unit test cannot prove that a missing observation
// log refuses result-cache publication rather than caching unobserved success.
func TestGoCacheWriteBudgetConformance(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fixture := testpair.CacheTempDir(t)
	module := "module github.com/xianxu/pair/cmd/conformance\n\ngo 1.26.3\n\nrequire github.com/xianxu/pair v0.0.0\nreplace github.com/xianxu/pair => " + fmt.Sprintf("%q", repo) + "\n"
	writeCacheFixture(t, filepath.Join(fixture, "go.mod"), module)
	sums, err := os.ReadFile(filepath.Join(repo, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	writeCacheFixture(t, filepath.Join(fixture, "go.sum"), string(sums))
	for _, mode := range []string{"control", "guarded"} {
		dir := filepath.Join(fixture, mode)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		writeCacheFixture(t, filepath.Join(dir, "value.go"), "package fixture\nfunc Value() bool { return true }\n")
		guard := "os.Exit(m.Run())"
		if mode == "guarded" {
			guard = `os.Exit(testdisk.RunWithWriteBudget("fixture", 8<<20, m.Run))`
		}
		imports := ""
		if mode == "guarded" {
			imports = `"github.com/xianxu/pair/cmd/internal/testdisk"`
		}
		writeCacheFixture(t, filepath.Join(dir, "value_test.go"), fmt.Sprintf(`package fixture
import ("os"; "testing"; %s)
func TestMain(m *testing.M) { %s }
 func TestValue(t *testing.T) {
 f, err := os.OpenFile(os.Getenv("PAIR_CACHE_MARKER"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
 if err != nil { t.Fatal(err) }; if _, err := f.Write([]byte("x")); err != nil { t.Fatal(err) }; if err := f.Close(); err != nil { t.Fatal(err) }
 if !Value() { t.Fatal("value") }
}
`, imports, guard))
	}
	// The marker lives outside the fixture module: Go does not treat external
	// files as cache inputs, so the control can prove a real result-cache hit.
	markerRoot := t.TempDir()
	moduleCache, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	env := testpair.Environment(t, t.TempDir(),
		"GOCACHE="+t.TempDir(), "GOMODCACHE="+strings.TrimSpace(string(moduleCache)), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off",
		"GOFLAGS=-p=2 -trimpath -mod=mod")
	run := func(t *testing.T, mode, marker string, extra ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()
		args := append([]string{"test", "./" + mode}, extra...)
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir, cmd.Env = fixture, append(append([]string{}, env...), "PAIR_CACHE_MARKER="+marker)
		cmd.WaitDelay = 2 * time.Second
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	markerSize := func(t *testing.T, path string, want int) {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) != want {
			t.Fatalf("execution marker bytes=%d want=%d error=%v", len(raw), want, err)
		}
	}
	t.Run("control really caches", func(t *testing.T) {
		marker := filepath.Join(markerRoot, "control")
		if out := run(t, "control", marker); strings.Contains(out, "(cached)") {
			t.Fatalf("cold control cached: %s", out)
		}
		if out := run(t, "control", marker); !strings.Contains(out, "(cached)") {
			t.Fatalf("warm control did not cache: %s", out)
		}
		markerSize(t, marker, 1)
	})
	t.Run("guarded always executes", func(t *testing.T) {
		marker := filepath.Join(markerRoot, "guarded")
		for range 2 {
			if out := run(t, "guarded", marker); strings.Contains(out, "(cached)") {
				t.Fatalf("guarded result reused cache: %s", out)
			}
		}
		markerSize(t, marker, 2)
	})
	t.Run("coverage survives without result caching", func(t *testing.T) {
		marker := filepath.Join(markerRoot, "coverage")
		profile := filepath.Join(fixture, "coverage.out")
		for range 2 {
			if out := run(t, "guarded", marker, "-coverprofile="+profile); strings.Contains(out, "(cached)") {
				t.Fatalf("guarded coverage reused cache: %s", out)
			}
			raw, err := os.ReadFile(profile)
			if err != nil || !strings.HasPrefix(string(raw), "mode: ") || !strings.Contains(string(raw), "value.go:") {
				t.Fatalf("coverage not retained: %v\n%s", err, raw)
			}
		}
		markerSize(t, marker, 2)
	})
}

func writeCacheFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
