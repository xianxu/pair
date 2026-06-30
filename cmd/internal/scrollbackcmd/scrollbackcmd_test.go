package scrollbackcmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUsage(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{}, io.Discard, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage: pair-scrollback-render") {
		t.Fatalf("stderr missing usage:\n%s", stderr.String())
	}
}

func TestRunWritesOutput(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "in.raw")
	evPath := filepath.Join(dir, "in.events.jsonl")
	outPath := filepath.Join(dir, "out.ansi")
	if err := os.WriteFile(rawPath, []byte("hello\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	events := `{"type":"resize","offset":0,"cols":20,"rows":5}` + "\n"
	if err := os.WriteFile(evPath, []byte(events), 0o644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := Run([]string{rawPath, evPath, outPath}, io.Discard, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr:\n%s", code, stderr.String())
	}
	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if !strings.Contains(string(body), "hello") {
		t.Fatalf("output missing rendered text:\n%s", string(body))
	}
}
