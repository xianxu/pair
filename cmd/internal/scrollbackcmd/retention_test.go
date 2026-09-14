package scrollbackcmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

func TestManagedCaptureReaderRegistersExactTargetBeforeRead(t *testing.T) {
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
	capture, _ := paths.ParkedScrollbackArtifacts("20260913T010101")
	if err := unix.Mkfifo(capture.Raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(capture.Events, nil, 0600); err != nil {
		t.Fatal(err)
	}
	process, err := storagegc.CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	intent, err := c.BeginUse(context.Background(), owner, process, capture.Raw)
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"PAIR_DATA_DIR": c.Root, "PAIR_TAG": "tag"}
	out := filepath.Join(t.TempDir(), "out.txt")
	done := make(chan int, 1)
	var stderr bytes.Buffer
	go func() {
		done <- RunWithEnv([]string{"--plain", "--owner-intent", intent, capture.Raw, capture.Events, out}, func(k string) string { return env[k] }, io.Discard, &stderr)
	}()
	found := false
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		state, err := c.ReadOwner(owner)
		if err == nil && len(state.Processes) == 1 && len(state.Intents) == 0 {
			if state.Processes[0].Target != capture.Raw {
				t.Fatal("reader not exact", state.Processes)
			}
			found = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !found {
		t.Fatal("read began without registered capture")
	}
	f, err := os.OpenFile(capture.Raw, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("captured\n")
	_ = f.Close()
	if code := <-done; code != 0 {
		t.Fatal(stderr.String())
	}
	state, err := c.ReadOwner(owner)
	if err != nil || len(state.Processes) != 0 || !state.Activity.LastUse.Equal(c.Now()) {
		t.Fatalf("passive rendering changed use: %+v %v", state, err)
	}
}
func TestExplicitOwnerRejectsOtherCaptureBeforeOutput(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "out")
	var stderr bytes.Buffer
	code := RunWithEnv([]string{"--plain", "--owner-dir", root, "--owner-tag", "tag", filepath.Join(root, "parked-scrollback-other-20260913T010101.raw"), "", out}, func(string) string { return "" }, io.Discard, &stderr)
	if code == 0 {
		t.Fatal("wrong capture accepted")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("invalid input wrote output")
	}
}
