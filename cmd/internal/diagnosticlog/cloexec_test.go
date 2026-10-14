package diagnosticlog

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func TestOpenRegularCloseOnExec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	for _, flags := range []int{syscall.O_RDWR | syscall.O_CREAT, syscall.O_RDONLY} {
		f, err := openRegular(path, flags)
		if err != nil {
			t.Fatal(err)
		}
		got, err := unix.FcntlInt(f.Fd(), unix.F_GETFD, 0)
		if err != nil {
			t.Error(err)
		} else if got&unix.FD_CLOEXEC == 0 {
			t.Error("regular descriptor can survive exec")
		}
		if err = f.Close(); err != nil {
			t.Error(err)
		}
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
