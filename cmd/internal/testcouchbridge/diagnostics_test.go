package testcouchbridge

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestDiagnosticsCapsCopyAndEmitsOneFinalMarker(t *testing.T) {
	var output bytes.Buffer
	d := NewDiagnostics(&output)
	payload := strings.Repeat("x", DiagnosticLimit+100)
	n, err := io.Copy(d, strings.NewReader(payload))
	if err != nil || n != int64(len(payload)) {
		t.Fatalf("copy: %d %v", n, err)
	}
	if output.Len() != DiagnosticLimit {
		t.Fatalf("copy bypassed cap: %d", output.Len())
	}
	if err = d.Finish("/private/owned/root"); err != nil {
		t.Fatal(err)
	}
	final := output.String()
	if !strings.Contains(final, "truncated") || !strings.Contains(final, "/private/owned/root") {
		t.Fatal("missing final diagnostic/root")
	}
	if err = d.Finish("/private/other"); err != nil {
		t.Fatal(err)
	}
	if output.String() != final {
		t.Fatal("duplicate final marker")
	}
	if _, err = d.Write([]byte("late")); err == nil {
		t.Fatal("write accepted after final marker")
	}
}

func TestDiagnosticsCleanFinishAndBoundedRoot(t *testing.T) {
	var output bytes.Buffer
	d := NewDiagnostics(&output)
	if _, err := d.Write([]byte("short\n")); err != nil {
		t.Fatal(err)
	}
	if err := d.Finish(""); err != nil {
		t.Fatal(err)
	}
	if output.String() != "short\n" {
		t.Fatalf("unneeded marker %q", output.String())
	}
	var long bytes.Buffer
	d = NewDiagnostics(&long)
	if err := d.Finish(strings.Repeat("\x00", DiagnosticLimit*2)); err != nil {
		t.Fatal(err)
	}
	if long.Len() > diagnosticMarkerLimit {
		t.Fatalf("unbounded root: %d", long.Len())
	}
}
