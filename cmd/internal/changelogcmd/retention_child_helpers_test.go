package changelogcmd

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDistillerDiagnosticBoundsExecCopy(t *testing.T) {
	var out distillerDiagnostic
	// Hide WriterTo as an exec pipe does: io.Copy must not bypass the cap via
	// a promoted bytes.Buffer.ReadFrom method on its destination.
	input := struct{ io.Reader }{strings.NewReader(strings.Repeat("x", 64*1024))}
	n, err := io.Copy(&out, input)
	if err != nil || n != 64*1024 || len(out.String()) != 32*1024 {
		t.Fatalf("drained=%d retained=%d err=%v", n, len(out.String()), err)
	}
	if n, err := out.Write([]byte("discarded")); err != nil || n != 9 || len(out.String()) != 32*1024 {
		t.Fatalf("later write: n=%d retained=%d err=%v", n, len(out.String()), err)
	}
}

func TestDistillerFIFOWriterAdmissionIsBoundedWithoutReader(t *testing.T) {
	_, _, fifo, _ := distillerFIFOFixture(t)
	// An unclosed completion channel represents a still-live child without
	// opening a reader; no command or background producer is created here.
	child := &distillerChild{done: make(chan struct{})}
	file, err := child.openWriter(fifo)
	if file != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("missing reader: file=%v err=%v", file, err)
	}
}

func TestDistillerFIFOWriterAdmissionPropagatesOtherErrors(t *testing.T) {
	child := &distillerChild{done: make(chan struct{})}
	file, err := child.openWriter(filepath.Join(t.TempDir(), "absent"))
	if file != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing FIFO: file=%v err=%v", file, err)
	}
}
