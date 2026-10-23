//go:build unix

package testcouchbridge

import (
	"errors"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func fillDiagnosticPipe(t *testing.T, f *os.File) {
	t.Helper()
	conn, err := f.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	fd := -1
	if err = conn.Control(func(raw uintptr) { fd = int(raw) }); err != nil {
		t.Fatal(err)
	}
	if err = unix.SetNonblock(fd, true); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 4096)
	for i := 0; i < 1024; i++ {
		_, err := unix.Write(fd, payload)
		if errors.Is(err, unix.EAGAIN) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Fatal("fixture pipe did not fill within bounded setup")
}

func TestDiagnosticPipeFullReturnsWithoutReader(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	sink, err := NewDiagnosticPipe(writer)
	if err != nil {
		t.Fatal(err)
	}
	fillDiagnosticPipe(t, writer)
	if _, err := sink.Write([]byte("unread")); !errors.Is(err, unix.EAGAIN) {
		t.Fatalf("full pipe: %v", err)
	}
}

func TestDiagnosticPipeRejectsNonPipe(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := NewDiagnosticPipe(file); err == nil {
		t.Fatal("accepted potentially blocking regular-file diagnostics")
	}
}

func TestDiagnosticPipeClosedReaderReturnsBrokenPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	sink, err := NewDiagnosticPipe(writer)
	if err != nil {
		t.Fatal(err)
	}
	if err = reader.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = sink.Write([]byte("lost reader")); !errors.Is(err, unix.EPIPE) {
		t.Fatalf("closed reader: %v", err)
	}
}
