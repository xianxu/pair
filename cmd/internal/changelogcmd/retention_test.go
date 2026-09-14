package changelogcmd

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

func TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse(t *testing.T) {
	c, err := storagegc.NewCoordinator(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := artifactpath.NewStorageOwner(c.Root, "", "tag")
	c.Now = func() time.Time { return time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) }
	if err := c.Initialize(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	paths, _ := artifactpath.ResolveScoped(c.Root, "tag")
	log, _ := paths.ChangelogArtifacts("claude", "")
	if err := unix.Mkfifo(log.Cleaned, 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"PAIR_DATA_DIR": c.Root, "PAIR_TAG": "tag"}
	done := make(chan int, 1)
	var stderr bytes.Buffer
	go func() {
		done <- RunWithEnv([]string{"--cleaned", log.Cleaned, "--log", log.Log, "--anchor", log.Anchor, "--ready", log.Ready}, func(k string) string { return env[k] }, &stderr)
	}()
	found := false
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		state, err := c.ReadOwner(owner)
		if err == nil && len(state.Processes) == 1 {
			found = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !found {
		t.Fatal("distiller read before registering")
	}
	f, err := os.OpenFile(log.Cleaned, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if code := <-done; code != 0 {
		t.Fatal(stderr.String())
	}
	state, err := c.ReadOwner(owner)
	if err != nil || len(state.Processes) != 0 || !state.Activity.LastUse.Equal(c.Now()) {
		t.Fatalf("background distillation changed use: %+v %v", state, err)
	}
}
