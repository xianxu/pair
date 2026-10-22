package couchcmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Exercise the existing native clipboard path without touching the host pasteboard.
func noticeCopyAssertion(t *testing.T, console *couchtty.Console) func(io.Writer, string) {
	t.Helper()
	dir := t.TempDir()
	copied := filepath.Join(dir, "notice")
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\nexec /bin/cat > \"$PAIR_TEST_NOTICE_COPY\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("PAIR_TEST_NOTICE_COPY", copied)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	observations := make(chan couchtty.CopyObservation, 1)
	console.SetCopyObserver(func(o couchtty.CopyObservation) { observations <- o })
	return func(input io.Writer, want string) {
		t.Helper()
		if _, err := input.Write([]byte("\x19")); err != nil {
			t.Fatal(err)
		}
		select {
		case o := <-observations:
			if o.Outcome != "copied" {
				t.Fatalf("notice copy: %+v", o)
			}
		case <-testwait.After(t):
			t.Fatal("notice copy did not finish")
		}
		got, err := os.ReadFile(copied)
		if err != nil || string(got) != want {
			t.Fatalf("copied notice=%q, error=%v; want complete diagnostic %q", got, err, want)
		}
	}
}
