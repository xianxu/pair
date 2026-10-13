package launcher

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecoveryLegacyTransferRefusesSpecialFiles(t *testing.T) {
	for _, kind := range []string{"fifo", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			src, dst := filepath.Join(dir, "source"), filepath.Join(dir, "destination")
			if kind == "fifo" {
				if err := unix.Mkfifo(src, 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				target := filepath.Join(dir, "target")
				if err := os.WriteFile(target, []byte("data"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, src); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan error, 1)
			go func() { done <- transferFile(src, dst, false) }()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("accepted special legacy source")
				}
			case <-time.After(250 * time.Millisecond):
				release := make(chan error, 1)
				go func() {
					f, e := os.OpenFile(src, os.O_WRONLY, 0)
					if e == nil {
						e = f.Close()
					}
					release <- e
				}()
				if err := <-release; err != nil {
					t.Fatal(err)
				}
				<-done
				t.Fatal("legacy transfer blocked on FIFO")
			}
		})
	}
}
func TestRecoveryTransferContextCancellationHasNoDestination(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "source"), filepath.Join(dir, "destination")
	if err := os.WriteFile(src, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := transferFileContext(ctx, src, dst, false); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled transfer wrote destination")
	}
}
