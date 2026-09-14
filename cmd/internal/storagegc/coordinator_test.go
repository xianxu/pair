package storagegc

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func coordinatorFixture(t *testing.T) (*Coordinator, artifactpath.StorageOwner) {
	t.Helper()
	root := t.TempDir()
	c, err := NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := artifactpath.NewStorageOwner(c.Root, "scope", "test")
	if err != nil {
		t.Fatal(err)
	}
	c.Now = func() time.Time { return time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC) }
	return c, owner
}

func TestUseIntentSurvivesUntilCompletion(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	process, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	token, err := c.BeginUse(ctx, o, process, "draft")
	if err != nil {
		t.Fatal(err)
	}
	state, err := c.ReadOwner(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Intents) != 1 {
		t.Fatal("missing durable intent before content effect")
	}
	before := state.Activity.LastUse
	c.Now = func() time.Time { return before.Add(time.Hour) }
	if err := c.CompleteUse(ctx, o, token); err != nil {
		t.Fatal(err)
	}
	state, err = c.ReadOwner(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Intents) != 0 || !state.Activity.LastUse.After(before) {
		t.Fatal("completion failed to replace intent with use")
	}
	if err := c.CompleteUse(ctx, o, token); err == nil {
		t.Fatal("unknown operation accepted")
	}
}

func TestIntentFailureRefusesContentEffect(t *testing.T) {
	c, o := coordinatorFixture(t)
	process, _ := CurrentProcessIdentity(os.Getpid())
	c.BeforePersist = func() error { return errors.New("disk fault") }
	wrote := false
	err := c.WriteChanged(context.Background(), o, process, "draft", func() (bool, error) { wrote = true; return true, nil })
	if err == nil || wrote {
		t.Fatal("content effect ran without durable protection")
	}
}

func TestFailedContentRetainsIntent(t *testing.T) {
	c, o := coordinatorFixture(t)
	process, _ := CurrentProcessIdentity(os.Getpid())
	err := c.WriteChanged(context.Background(), o, process, "draft", func() (bool, error) { return true, errors.New("interrupted after write") })
	if err == nil {
		t.Fatal("missing write error")
	}
	state, err := c.ReadOwner(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Intents) != 1 {
		t.Fatal("lost intent after indeterminate write")
	}
}

func TestUnchangedWriteDoesNotRefresh(t *testing.T) {
	c, o := coordinatorFixture(t)
	process, _ := CurrentProcessIdentity(os.Getpid())
	ctx := context.Background()
	if err := c.Initialize(ctx, o); err != nil {
		t.Fatal(err)
	}
	before, _ := c.ReadOwner(o)
	c.Now = func() time.Time { return before.Activity.LastUse.Add(time.Hour) }
	if err := c.WriteChanged(ctx, o, process, "draft", func() (bool, error) { return false, nil }); err != nil {
		t.Fatal(err)
	}
	after, _ := c.ReadOwner(o)
	if !after.Activity.LastUse.Equal(before.Activity.LastUse) || len(after.Intents) != 0 {
		t.Fatal("unchanged write counted as use")
	}
}

func TestReadOwnerDoesNotInitialize(t *testing.T) {
	c, o := coordinatorFixture(t)
	if _, err := c.ReadOwner(o); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(c.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("read-only lookup created metadata")
	}
}

func TestCoordinatorRejectsSymlinkState(t *testing.T) {
	c, o := coordinatorFixture(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, c.Root+"/.retention"); err != nil {
		t.Fatal(err)
	}
	if err := c.Initialize(context.Background(), o); err == nil {
		t.Fatal("followed metadata symlink")
	}
	files, _ := os.ReadDir(outside)
	if len(files) != 0 {
		t.Fatal("wrote through symlink")
	}
}

func TestRecoverUseKeepsLiveAndUnknownIntent(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	p, _ := CurrentProcessIdentity(os.Getpid())
	id, err := c.BeginUse(ctx, o, p, "draft")
	if err != nil {
		t.Fatal(err)
	}
	probe := &FakeProcessProbe{Processes: map[int]string{p.PID: p.Birth}, Unknown: map[int]bool{}}
	c.Probe = probe
	if err := c.RecoverUse(ctx, o); err != nil {
		t.Fatal(err)
	}
	s, _ := c.ReadOwner(o)
	if len(s.Intents) != 1 || s.Intents[0].ID != id {
		t.Fatal("live intent lost")
	}
	probe.Unknown[p.PID] = true
	if err := c.RecoverUse(ctx, o); err != nil {
		t.Fatal(err)
	}
	s, _ = c.ReadOwner(o)
	if len(s.Intents) != 1 {
		t.Fatal("unknown intent lost")
	}
	delete(probe.Unknown, p.PID)
	delete(probe.Processes, p.PID)
	old := s.Activity.LastUse
	c.Now = func() time.Time { return old.Add(61 * 24 * time.Hour) }
	if err := c.RecoverUse(ctx, o); err != nil {
		t.Fatal(err)
	}
	s, _ = c.ReadOwner(o)
	if len(s.Intents) != 0 || !s.Activity.LastUse.Equal(c.Now()) {
		t.Fatal("dead interrupted write needs recovery grace")
	}
}

func TestProcessRegistrationDoesNotRefreshExistingUse(t *testing.T) {
	c, o := coordinatorFixture(t)
	ctx := context.Background()
	p, _ := CurrentProcessIdentity(os.Getpid())
	if err := c.Initialize(ctx, o); err != nil {
		t.Fatal(err)
	}
	before, _ := c.ReadOwner(o)
	c.Now = func() time.Time { return before.Activity.LastUse.Add(time.Hour) }
	id, err := c.RegisterProcess(ctx, o, p, "background")
	if err != nil {
		t.Fatal(err)
	}
	s, _ := c.ReadOwner(o)
	if len(s.Processes) != 1 || !s.Activity.LastUse.Equal(before.Activity.LastUse) {
		t.Fatal("registration refreshed use")
	}
	if err := c.ReleaseProcess(ctx, o, id); err != nil {
		t.Fatal(err)
	}
	s, _ = c.ReadOwner(o)
	if len(s.Processes) != 0 || !s.Activity.LastUse.Equal(before.Activity.LastUse) {
		t.Fatal("release refreshed use")
	}
}

func TestBeginUseRejectsStaleProcess(t *testing.T) {
	c, o := coordinatorFixture(t)
	p, _ := CurrentProcessIdentity(os.Getpid())
	p.Birth = "wrong"
	if _, err := c.BeginUse(context.Background(), o, p, "draft"); err == nil {
		t.Fatal("stale writer authorized")
	}
}

func TestMetadataDirectorySyncFailureBlocksEffect(t *testing.T) {
	c, o := coordinatorFixture(t)
	p, _ := CurrentProcessIdentity(os.Getpid())
	c.SyncDirectory = func(path string) error {
		if path == c.Root {
			return errors.New("directory fsync failed")
		}
		return syncDirectory(path)
	}
	wrote := false
	if err := c.WriteChanged(context.Background(), o, p, "draft", func() (bool, error) { wrote = true; return true, nil }); err == nil || wrote {
		t.Fatal("undurable metadata directory authorized write")
	}
}

func TestReadOwnerRejectsFIFO(t *testing.T) {
	c, o := coordinatorFixture(t)
	if err := c.Initialize(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(c.statePath(o)); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(c.statePath(o), 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := c.ReadOwner(o); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO blocked metadata read")
	}
}
