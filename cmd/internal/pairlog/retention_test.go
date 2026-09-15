package pairlog

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func managedLogFixture(t *testing.T) (*storagegc.Coordinator, artifactpath.StorageOwner, func(string) string, string) {
	t.Helper()
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	o, _ := artifactpath.NewStorageOwner(c.Root, "", "tag")
	p, _ := artifactpath.ResolveScoped(c.Root, "tag")
	env := map[string]string{"PAIR_DATA_DIR": c.Root, "PAIR_TAG": "tag", "PAIR_LOG_PATH": p.Log()}
	c.Now = func() time.Time { return time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) }
	if err := c.Initialize(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	return c, o, func(k string) string { return env[k] }, p.Log()
}
func TestManagedLogChangesPublishUseButRetriesDoNot(t *testing.T) {
	c, o, getenv, _ := managedLogFixture(t)
	args := []string{"--append-id", "attempt-managed"}
	if code := RunCLI(args, strings.NewReader("authored"), getenv, time.Now(), io.Discard); code != 0 {
		t.Fatal(code)
	}
	first, err := c.ReadOwner(o)
	if err != nil || !first.Activity.LastUse.After(c.Now()) || len(first.Intents) != 0 || len(first.Processes) != 0 {
		t.Fatalf("first write unprotected: %+v %v", first, err)
	}
	if code := RunCLI(args, strings.NewReader("authored"), getenv, time.Now(), io.Discard); code != 0 {
		t.Fatal(code)
	}
	second, err := c.ReadOwner(o)
	if err != nil || !first.Activity.LastUse.Equal(second.Activity.LastUse) {
		t.Fatalf("no-op prepare refreshed use: %+v %v", second, err)
	}
	if code := RunCommitCLI(args, getenv, io.Discard); code != 0 {
		t.Fatal(code)
	}
	third, err := c.ReadOwner(o)
	if err != nil || !third.Activity.LastUse.After(second.Activity.LastUse) {
		t.Fatalf("submission untracked: %+v %v", third, err)
	}
	if code := RunCommitCLI(args, getenv, io.Discard); code != 0 {
		t.Fatal(code)
	}
	fourth, err := c.ReadOwner(o)
	if err != nil || !third.Activity.LastUse.Equal(fourth.Activity.LastUse) {
		t.Fatal("no-op commit refreshed use", err)
	}
}
func TestManagedLogRejectsWrongOwnerBeforeWriting(t *testing.T) {
	c, _, getenv, _ := managedLogFixture(t)
	foreign := filepath.Join(t.TempDir(), "foreign.md")
	env := func(k string) string {
		if k == "PAIR_LOG_PATH" {
			return foreign
		}
		return getenv(k)
	}
	if code := RunCLI([]string{"--append-id", "attempt-x"}, strings.NewReader("authored"), env, time.Now(), io.Discard); code == 0 {
		t.Fatal("foreign owner accepted")
	}
	if _, err := os.Stat(foreign); !os.IsNotExist(err) {
		t.Fatal("foreign log written")
	}
	_ = c
}

func TestManagedLogAcceptsSelectedDirectoryAliasRejectsLeafSymlink(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	env := func(k string) string {
		switch k {
		case "PAIR_DATA_DIR":
			return alias
		case "PAIR_TAG":
			return "work"
		}
		return ""
	}
	path := filepath.Join(alias, "log-work.md")
	called := false
	if err := managedLogWrite(env, path, func(SessionLogStore) error { called = true; return nil }); err != nil || !called {
		t.Fatalf("alias: %v %v", called, err)
	}
	target := filepath.Join(root, "elsewhere")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if err := managedLogWrite(env, path, func(SessionLogStore) error { t.Fatal("followed final symlink"); return nil }); err == nil {
		t.Fatal("accepted final symlink")
	}
}
