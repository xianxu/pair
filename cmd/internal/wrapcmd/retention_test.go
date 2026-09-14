package wrapcmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type retentionObserver struct {
	c        *storagegc.Coordinator
	owner    artifactpath.StorageOwner
	observed bool
}

func (w *retentionObserver) Write(p []byte) (int, error) {
	state, err := w.c.ReadOwner(w.owner)
	if err == nil {
		for _, r := range state.Processes {
			if r.Process.PID == os.Getpid() && r.Role == "wrapper" {
				w.observed = true
			}
		}
	}
	return len(p), nil
}

func TestRunProtectsActualWrapperLifetime(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", root)
	t.Setenv("PAIR_TAG", "retention-test")
	t.Setenv("PAIR_SCOPE_KEY", "")
	c, err := storagegc.NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := artifactpath.NewStorageOwner(c.Root, "", "retention-test")
	if err != nil {
		t.Fatal(err)
	}
	master, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer tty.Close()
	observer := &retentionObserver{c: c, owner: owner}
	var stderr bytes.Buffer
	if code := Run([]string{"sh", "-c", "printf ready"}, tty, observer, &stderr); code != 0 {
		t.Fatalf("run %d: %s", code, stderr.String())
	}
	if !observer.observed {
		t.Fatal("wrapper wrote output without lifetime registration")
	}
	state, err := c.ReadOwner(owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Processes) != 0 {
		t.Fatal("wrapper registration survived normal exit")
	}
}

func TestInvalidManagedOwnerRefusesWrapper(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	t.Setenv("PAIR_TAG", "../escape")
	t.Setenv("PAIR_SCOPE_KEY", "")
	master, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer tty.Close()
	var stderr bytes.Buffer
	if code := Run([]string{"sh", "-c", "exit 0"}, tty, io.Discard, &stderr); code == 0 {
		t.Fatal("invalid storage owner launched writer")
	}
}
