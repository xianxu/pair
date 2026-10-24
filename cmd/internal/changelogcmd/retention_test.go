package changelogcmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

// TestManagedDistillerProcess is selected only by the owned self-exec parent.
// Keeping the real command in a child lets every assertion failure cancel and
// join a blocked FIFO reader; a goroutine around os.ReadFile cannot be cancelled.
func TestManagedDistillerProcess(t *testing.T) {
	mode := os.Getenv("PAIR_TEST_CHANGELOG_CHILD")
	if mode == "" {
		return
	}
	if mode == "exit" {
		os.Exit(23)
	}
	root := os.Getenv("PAIR_TEST_CHANGELOG_ROOT")
	if mode != "render" || root == "" || !filepath.IsAbs(root) {
		t.Fatal("invalid distiller child selection")
	}
	paths, err := artifactpath.ResolveScoped(root, "tag")
	if err != nil {
		t.Fatal(err)
	}
	log, err := paths.ChangelogArtifacts("claude", "")
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"PAIR_DATA_DIR": root, "PAIR_TAG": "tag"}
	code := RunWithEnv([]string{"--cleaned", log.Cleaned, "--log", log.Log, "--anchor", log.Anchor, "--ready", log.Ready}, func(k string) string { return env[k] }, os.Stderr)
	if code != 0 {
		t.Fatalf("RunWithEnv returned %d", code)
	}
}

func distillerFIFOFixture(t *testing.T) (*storagegc.Coordinator, artifactpath.StorageOwner, string, string) {
	t.Helper()
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	owner, err := artifactpath.NewStorageOwner(c.Root, "", "tag")
	if err != nil {
		t.Fatal(err)
	}
	c.Now = func() time.Time { return time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) }
	if err := c.Initialize(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	paths, err := artifactpath.ResolveScoped(c.Root, "tag")
	if err != nil {
		t.Fatal(err)
	}
	log, err := paths.ChangelogArtifacts("claude", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(log.Cleaned, 0600); err != nil {
		t.Fatal(err)
	}
	// A future regression must never dispatch an operator's real model.
	modelDir := fakeClaude(t, "- unexpected model call\n")
	return c, owner, log.Cleaned, modelDir
}

func TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse(t *testing.T) {
	c, owner, fifo, modelDir := distillerFIFOFixture(t)
	child := startDistillerChild(t, context.Background(), c.Root, "render")
	if err := child.awaitLease(c, owner); err != nil {
		t.Fatal(err)
	}
	f, err := child.openWriter(fifo)
	if err != nil {
		t.Fatal(err)
	}
	// Always close the writer before the child's cleanup joins Wait.
	defer f.Close()
	// An empty open/close can strand os.ReadFile on Darwin (#679). Actual
	// payload survives a delayed reader; a newline still means no transcript.
	if n, err := f.WriteString("\n"); err != nil || n != 1 {
		t.Fatalf("FIFO payload: wrote %d: %v", n, err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := child.wait(); err != nil {
		t.Fatal(err)
	}
	state, err := c.ReadOwner(owner)
	if err != nil || len(state.Processes) != 0 || !state.Activity.LastUse.Equal(c.Now()) {
		t.Fatalf("background distillation changed use: %+v %v", state, err)
	}
	if invoked(modelDir) {
		t.Fatal("blank FIFO input invoked the model")
	}
	paths, _ := artifactpath.ResolveScoped(c.Root, "tag")
	log, _ := paths.ChangelogArtifacts("claude", "")
	for _, path := range []string{log.Log, log.Anchor, log.Ready} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("empty transcript created %s: %v", path, err)
		}
	}
}

func TestDistillerFIFOChildEarlyExitIsJoined(t *testing.T) {
	c, owner, fifo, _ := distillerFIFOFixture(t)
	child := startDistillerChild(t, context.Background(), c.Root, "exit")
	if err := child.awaitLease(c, owner); err == nil || !strings.Contains(err.Error(), "exited") {
		t.Fatalf("early exit was not reported: %v", err)
	}
	if _, err := child.openWriter(fifo); err == nil {
		t.Fatal("writer admitted after child exit")
	}
	assertDistillerJoined(t, child)
}

func TestDistillerFIFOChildCancellationJoinsBlockedRead(t *testing.T) {
	c, owner, _, modelDir := distillerFIFOFixture(t)
	child := startDistillerChild(t, context.Background(), c.Root, "render")
	if err := child.awaitLease(c, owner); err != nil {
		t.Fatal(err)
	}
	// With no writer the real RunWithEnv is blocked opening/reading the FIFO.
	// Cancellation must reap it rather than leave a syscall goroutine behind.
	child.stop()
	if err := child.wait(); err == nil {
		t.Fatal("cancelled blocked reader succeeded")
	}
	assertDistillerJoined(t, child)
	if invoked(modelDir) {
		t.Fatal("cancelled reader invoked the model")
	}
}
