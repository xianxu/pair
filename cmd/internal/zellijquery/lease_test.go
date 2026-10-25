package zellijquery

import (
	"context"
	"golang.org/x/sys/unix"
	"path/filepath"
	"testing"
)

func TestFileLeaseCloseOnExec(t *testing.T) {
	lease, err := AcquireFileLease(context.Background(), filepath.Join(t.TempDir(), "lease"))
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	l := lease.(*fileLease)
	flags, err := unix.FcntlInt(l.file.Fd(), unix.F_GETFD, 0)
	if err != nil || flags&unix.FD_CLOEXEC == 0 {
		t.Fatalf("descriptor inheritance: %d %v", flags, err)
	}
}
