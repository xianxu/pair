package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pair-changelog")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// fakeClaude writes a PATH-shimmed `claude` that drains stdin, records that it
// ran (an "invoked" sentinel), and prints `body` — a process-level fake. It
// returns the dir holding the sentinel so a test can assert (non-)invocation.
func fakeClaude(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "body"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ntouch " + sh(filepath.Join(dir, "invoked")) +
		"\ncat >/dev/null\ncat " + sh(filepath.Join(dir, "body")) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func sh(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func invoked(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "invoked"))
	return err == nil
}

// run writes the cleaned/log/anchor fixtures, runs the binary, and returns the
// resulting log + anchor contents.
func run(t *testing.T, bin, cleaned, priorLog, priorAnchor, today string) (log, anchor string) {
	t.Helper()
	dir := t.TempDir()
	cleanedPath := filepath.Join(dir, "cleaned.txt")
	logPath := filepath.Join(dir, "changelog.md")
	anchorPath := filepath.Join(dir, "changelog.anchor")
	mustWrite(t, cleanedPath, cleaned)
	if priorLog != "" {
		mustWrite(t, logPath, priorLog)
	}
	if priorAnchor != "" {
		mustWrite(t, anchorPath, priorAnchor)
	}
	out, err := exec.Command(bin,
		"--cleaned", cleanedPath, "--log", logPath, "--anchor", anchorPath,
		"--agent", "claude", "--today", today,
	).CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	return readOr(logPath), readOr(anchorPath)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readOr(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func TestFirstRun(t *testing.T) {
	bin := buildBinary(t)
	fakeClaude(t, "- entry one\n\n- entry two\n")
	cleaned := "intro line\nwork\nLAST1\nLAST2\nLAST3\n"
	log, anchor := run(t, bin, cleaned, "", "", "2026-06-12")

	want := "## 2026-06-12\n\n- entry one\n\n- entry two\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
	if anchor != "LAST1\nLAST2\nLAST3\n" {
		t.Fatalf("anchor = %q", anchor)
	}
}

func TestIncrementalFreezesPrefixAndRevisesLast(t *testing.T) {
	bin := buildBinary(t)
	fakeClaude(t, "- two-revised\n\n- three\n")
	// anchor present mid-stream, with new content after it.
	cleaned := "intro\nwork\nANCHOR1\nANCHOR2\nANCHOR3\nnew work a\nnew work b\n"
	priorLog := "## 2026-06-12\n\n- one\n\n- two\n"
	priorAnchor := "ANCHOR1\nANCHOR2\nANCHOR3\n"
	log, anchor := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	frozen := "## 2026-06-12\n\n- one\n\n"
	if !strings.HasPrefix(log, frozen) {
		t.Fatalf("frozen prefix not byte-identical:\n%q", log)
	}
	want := "## 2026-06-12\n\n- one\n\n- two-revised\n\n- three\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
	// the anchor advanced to the last 3 cleaned lines.
	if anchor != "ANCHOR3\nnew work a\nnew work b\n" {
		t.Fatalf("anchor = %q", anchor)
	}
}

func TestReviseOnlyNeverDropsLast(t *testing.T) {
	bin := buildBinary(t)
	fakeClaude(t, "- two-revised\n") // only the revised last entry, no new
	cleaned := "intro\nANCHOR1\nANCHOR2\nANCHOR3\nnew tail\n"
	priorLog := "## 2026-06-12\n\n- one\n\n- two\n"
	priorAnchor := "ANCHOR1\nANCHOR2\nANCHOR3\n"
	log, _ := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	want := "## 2026-06-12\n\n- one\n\n- two-revised\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
}

func TestDateRollover(t *testing.T) {
	bin := buildBinary(t)
	fakeClaude(t, "- two-revised\n\n- three\n")
	cleaned := "intro\nANCHOR1\nANCHOR2\nANCHOR3\nnew tail\n"
	priorLog := "## 2026-06-11\n\n- one\n\n- two\n"
	priorAnchor := "ANCHOR1\nANCHOR2\nANCHOR3\n"
	log, _ := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	want := "## 2026-06-11\n\n- one\n\n- two-revised\n\n## 2026-06-12\n\n- three\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
}

func TestNoOpDoesNotCallModelOrChangeLog(t *testing.T) {
	bin := buildBinary(t)
	dir := fakeClaude(t, "- should not appear\n")
	// anchor IS the last 3 lines of cleaned → flush with the end → no-op.
	cleaned := "intro\nwork\nLAST1\nLAST2\nLAST3\n"
	priorLog := "## 2026-06-12\n\n- one\n\n- two\n"
	priorAnchor := "LAST1\nLAST2\nLAST3\n"
	log, _ := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	if log != priorLog {
		t.Fatalf("log changed on no-op:\n%q", log)
	}
	if invoked(dir) {
		t.Fatal("model was called on a no-op press")
	}
}
