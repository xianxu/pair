package historyarchive_test

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func holdLock(t *testing.T, f *os.File) {
	t.Helper()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
}
