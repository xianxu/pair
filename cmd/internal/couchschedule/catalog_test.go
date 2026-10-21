package couchschedule

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/osfs"
)

type catalogGit struct {
	names map[string]string
	calls int
}

func (g *catalogGit) RunContext(_ context.Context, dir string, args ...string) (string, error) {
	g.calls++
	if strings.Join(args, " ") != "config --get user.name" {
		return "", fmt.Errorf("unexpected Git call")
	}
	return g.names[dir], nil
}
func catalogRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}
func catalogTask(t *testing.T, root, name, extra string) {
	t.Helper()
	dir := filepath.Join(root, "scripts", "scheduled", name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for file, raw := range map[string]string{"main.go": "package main\nfunc main(){}", "schedule": `{"version":1,"cron":"0 7 * * *","timeout":"1m"` + extra + "}"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCatalogConfiguredHubsAndOwnerRevisions(t *testing.T) {
	a, b := catalogRoot(t), catalogRoot(t)
	catalogTask(t, a, "same", `,"scope":"team","owner":"xian-xu"`)
	catalogTask(t, b, "same", "")
	if err := os.WriteFile(filepath.Join(a, "team.md"), []byte("## Operators\n- xian-xu (Xian Xu): owner\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git := &catalogGit{names: map[string]string{a: "Xian Xu"}}
	c := Catalog{Git: git}
	got, err := c.Discover(t.Context(), []string{a, b})
	if err != nil || len(got) != 2 {
		t.Fatalf("discover: %v %v", got, err)
	}
	if !got[0].Allowed || !got[1].Allowed || got[0].Key == got[1].Key || git.calls != 1 {
		t.Fatalf("routing: %+v calls=%d", got, git.calls)
	}
	rev := got[0].Revision
	git.names[a] = "Other"
	got, err = c.Discover(t.Context(), []string{a, b})
	if err != nil || got[0].Allowed || got[0].Reason != "not-owner" || got[0].Revision == rev {
		t.Fatalf("owner change: %+v %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(a, "scripts/scheduled/same/schedule"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = c.Discover(t.Context(), []string{a, b})
	if err != nil || got[0].Allowed || !got[1].Allowed || got[0].Revision == rev {
		t.Fatalf("replacement: %+v %v", got, err)
	}
}

func TestCatalogTaskRefusalsAndNoFollow(t *testing.T) {
	root := catalogRoot(t)
	catalogTask(t, root, "machine", "")
	catalogTask(t, root, "team", `,"scope":"team","owner":"xian"`)
	c := Catalog{Git: &catalogGit{names: map[string]string{root: "Xian"}}}
	got, err := c.Discover(t.Context(), []string{root})
	if err != nil || !got[0].Allowed || got[1].Allowed {
		t.Fatalf("absent Operators: %+v %v", got, err)
	}
	if err := os.Remove(filepath.Join(root, "scripts/scheduled/machine/main.go")); err != nil {
		t.Fatal(err)
	}
	got, err = c.Discover(t.Context(), []string{root})
	if err != nil || got[0].Allowed {
		t.Fatalf("missing main: %+v %v", got, err)
	}
	outside := catalogRoot(t)
	catalogTask(t, outside, "escape", "")
	if err := os.Symlink(filepath.Join(outside, "scripts/scheduled/escape"), filepath.Join(root, "scripts/scheduled/link")); err != nil {
		t.Fatal(err)
	}
	got, err = c.Discover(t.Context(), []string{root})
	if err != nil || len(got) != 3 || got[0].Allowed {
		t.Fatalf("task symlink: %+v %v", got, err)
	}
	empty := catalogRoot(t)
	if got, err := c.Discover(t.Context(), []string{empty}); err != nil || len(got) != 0 {
		t.Fatalf("empty: %+v %v", got, err)
	}
	if err := os.Symlink(filepath.Join(outside, "scripts"), filepath.Join(empty, "scripts")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Discover(t.Context(), []string{empty}); err == nil {
		t.Fatal("followed scripts symlink")
	}
}

func TestCatalogCountsAndBounds(t *testing.T) {
	for _, n := range []int{8, 16} {
		root := catalogRoot(t)
		for i := 0; i < n; i++ {
			catalogTask(t, root, fmt.Sprintf("task-%02d", i), "")
		}
		reads := 0
		c := Catalog{ReadFile: func(ctx context.Context, f *os.File, max int64) ([]byte, error) {
			reads++
			return osfs.ReadOpenedRegularMetadata(ctx, f, max)
		}}
		got, err := c.Discover(t.Context(), []string{root})
		if err != nil || len(got) != n || reads != n {
			t.Fatalf("n=%d tasks=%d reads=%d err=%v", n, len(got), reads, err)
		}
	}
	root := catalogRoot(t)
	for i := 0; i < 257; i++ {
		catalogTask(t, root, fmt.Sprintf("task-%03d", i), "")
	}
	if _, err := (Catalog{}).Discover(t.Context(), []string{root}); err == nil {
		t.Fatal("accepted over cap")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := (Catalog{}).Discover(ctx, []string{root}); err == nil {
		t.Fatal("ignored cancellation")
	}
}

func TestCatalogCancellationIsNotTaskRefusal(t *testing.T) {
	root := catalogRoot(t)
	catalogTask(t, root, "one", "")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	c := Catalog{ReadFile: func(ctx context.Context, f *os.File, max int64) ([]byte, error) {
		cancel()
		return osfs.ReadOpenedRegularMetadata(ctx, f, max)
	}}
	if _, err := c.Discover(ctx, []string{root}); err != context.Canceled {
		t.Fatalf("cancellation became task refusal: %v", err)
	}
}
