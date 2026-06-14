package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// idleFooter is an idle claude footer (empty input box + rule + status),
// appended to test transcripts so trimLiveTail does its precise empty-box cut
// as on a real TTY (short fixtures would otherwise hit the coarse skip-4 path).
const idleFooter = "\u276f \n\u2500\u2500\u2500\u2500\n\u23f5\u23f5 bypass permissions\n"

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
	script := "#!/bin/sh\nprintf 'x\\n' >> " + sh(filepath.Join(dir, "calls")) +
		"\ncat > " + sh(filepath.Join(dir, "stdin")) +
		"\ncat " + sh(filepath.Join(dir, "body")) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func sh(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func invoked(dir string) bool { return callCount(dir) > 0 }

// callCount returns how many times the fake model was invoked.
func callCount(dir string) int {
	b, err := os.ReadFile(filepath.Join(dir, "calls"))
	if err != nil {
		return 0
	}
	return strings.Count(string(b), "\n")
}

// stdinLines returns how many lines the fake model received on stdin (the model
// input), or -1 if it was never invoked.
func stdinLines(dir string) int {
	b, err := os.ReadFile(filepath.Join(dir, "stdin"))
	if err != nil {
		return -1
	}
	return strings.Count(string(b), "\n")
}

// run writes the cleaned/log/anchor fixtures, runs the binary, and returns the
// resulting log + anchor contents.
func run(t *testing.T, bin, cleaned, priorLog, priorAnchor, today string) (log, anchor string) {
	t.Helper()
	return runIn(t, bin, t.TempDir(), cleaned, priorLog, priorAnchor, today)
}

// runIn is run() with an explicit dir so a test can inspect the sidecar files a
// run produces (e.g. the "<base>.ready" marker). run() wraps it with a fresh
// temp dir; the log/anchor live at changelog.md / changelog.anchor under `dir`.
func runIn(t *testing.T, bin, dir, cleaned, priorLog, priorAnchor, today string) (log, anchor string) {
	t.Helper()
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
	cleaned := "❯ start\nintro line\nLAST1\nLAST2\nLAST3\n" + idleFooter
	log, anchor := run(t, bin, cleaned, "", "", "2026-06-12")

	want := "## 2026-06-12\n\n- entry one\n\n- entry two\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
	if anchor != "turns:1\nLAST1\nLAST2\nLAST3\n" {
		t.Fatalf("anchor = %q", anchor)
	}
}

func TestIncrementalFreezesPrefixAndRevisesLast(t *testing.T) {
	bin := buildBinary(t)
	fakeClaude(t, "- two-revised\n\n- three\n")
	// A new turn (2 boundaries > prior 1) triggers the distill; the anchor is
	// present mid-stream with new content after it.
	cleaned := "❯ first\nintro\n❯ second\nANCHOR1\nANCHOR2\nANCHOR3\nnew work a\nnew work b\n" + idleFooter
	priorLog := "## 2026-06-12\n\n- one\n\n- two\n"
	priorAnchor := "turns:1\nANCHOR1\nANCHOR2\nANCHOR3\n"
	log, anchor := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	frozen := "## 2026-06-12\n\n- one\n\n"
	if !strings.HasPrefix(log, frozen) {
		t.Fatalf("frozen prefix not byte-identical:\n%q", log)
	}
	want := "## 2026-06-12\n\n- one\n\n- two-revised\n\n- three\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
	// the anchor advanced to the last 3 cleaned lines + the new turn count.
	if anchor != "turns:2\nANCHOR3\nnew work a\nnew work b\n" {
		t.Fatalf("anchor = %q", anchor)
	}
}

func TestReviseOnlyNeverDropsLast(t *testing.T) {
	bin := buildBinary(t)
	fakeClaude(t, "- two-revised\n") // only the revised last entry, no new
	cleaned := "❯ a\nintro\n❯ b\nANCHOR1\nANCHOR2\nANCHOR3\nnew tail\n" + idleFooter
	priorLog := "## 2026-06-12\n\n- one\n\n- two\n"
	priorAnchor := "turns:1\nANCHOR1\nANCHOR2\nANCHOR3\n"
	log, _ := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	want := "## 2026-06-12\n\n- one\n\n- two-revised\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
}

func TestDateRollover(t *testing.T) {
	bin := buildBinary(t)
	fakeClaude(t, "- two-revised\n\n- three\n")
	cleaned := "❯ a\nintro\n❯ b\nANCHOR1\nANCHOR2\nANCHOR3\nnew tail\n" + idleFooter
	priorLog := "## 2026-06-11\n\n- one\n\n- two\n"
	priorAnchor := "turns:1\nANCHOR1\nANCHOR2\nANCHOR3\n"
	log, _ := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	want := "## 2026-06-11\n\n- one\n\n- two-revised\n\n## 2026-06-12\n\n- three\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
}

// When the anchor is absent from the freshly-cleaned TTY (a redraw mangled it)
// but a prior log exists, locate → FullRedistill: main feeds the whole
// transcript as "new activity" yet still keeps the frozen prefix (only the last
// entry is model-revised). This pins that subtle seam behavior.
func TestFullRedistillWithPriorLogKeepsFrozenPrefix(t *testing.T) {
	bin := buildBinary(t)
	dir := fakeClaude(t, "- two-revised\n\n- three\n")
	// 2 boundaries > prior 1 → not a no-op; OLD1-3 absent → FullRedistill.
	cleaned := "❯ p\n❯ q\nfresh1\nfresh2\nfresh3\n" + idleFooter
	priorLog := "## 2026-06-12\n\n- one\n\n- two\n"
	priorAnchor := "turns:1\nOLD1\nOLD2\nOLD3\n" // absent in cleaned → FullRedistill
	log, anchor := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	if !invoked(dir) {
		t.Fatal("full-redistill should still call the model")
	}
	frozen := "## 2026-06-12\n\n- one\n\n"
	if !strings.HasPrefix(log, frozen) {
		t.Fatalf("frozen prefix not preserved on full-redistill:\n%q", log)
	}
	want := "## 2026-06-12\n\n- one\n\n- two-revised\n\n- three\n"
	if log != want {
		t.Fatalf("log = %q\nwant %q", log, want)
	}
	if anchor != "turns:2\nfresh1\nfresh2\nfresh3\n" {
		t.Fatalf("anchor = %q", anchor)
	}
}

// No new completed turn (cleaned has 1 boundary, prior recorded 1) → no-op: the
// model is not called and the log is untouched, even though the trailing lines
// churned. This is the turn-count no-op that replaces the brittle byte-flush one.
func TestNoOpWhenNoNewTurn(t *testing.T) {
	bin := buildBinary(t)
	dir := fakeClaude(t, "- should not appear\n")
	cleaned := "❯ a\nwork churned a bit\nLAST1\nLAST2\nLAST3\n" + idleFooter // still 1 boundary
	priorLog := "## 2026-06-12\n\n- one\n\n- two\n"
	// Realistic same-session anchor: it locates in the cleaned (committed tail).
	// A no-op requires the anchor to still be found — an absent anchor means the
	// session reset, which must re-distill (TestSessionResetDistillsNotNoOp).
	priorAnchor := "turns:1\nLAST1\nLAST2\nLAST3\n"
	log, _ := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	if log != priorLog {
		t.Fatalf("log changed on no-op:\n%q", log)
	}
	if invoked(dir) {
		t.Fatal("model was called on a no-op press")
	}
}

// After an agent restart (Alt+n) the screen re-renders as a fresh session whose
// turn count is BELOW the stale anchor's priorTurns. The turn-count no-op
// (len(boundaries) <= priorTurns) used to fire on that "fewer turns" reading and
// the new session never distilled. The anchor is a per-session marker, so when
// it no longer locates (FullRedistill) we must distill, not no-op (#58 follow-up).
func TestSessionResetDistillsNotNoOp(t *testing.T) {
	bin := buildBinary(t)
	dir := fakeClaude(t, "- new-session entry\n")
	// New session: one prompt boundary, content that does NOT contain the stale
	// anchor snippet below.
	cleaned := "❯ fresh prompt\nnew session work\nNEWLAST1\nNEWLAST2\nNEWLAST3\n" + idleFooter
	priorLog := "## 2026-06-12\n\n- old one\n\n- old two\n"
	// Stale anchor from the prior, longer session: high turn count + a snippet
	// that won't be found in the new cleaned → locate returns FullRedistill.
	priorAnchor := "turns:9\nOLD_SESSION_TAIL_A\nOLD_SESSION_TAIL_B\nOLD_SESSION_TAIL_C\n"
	log, anchor := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")

	if !invoked(dir) {
		t.Fatal("model NOT called after a session reset (no-op fired on a stale-anchor turn count)")
	}
	if !strings.Contains(log, "- new-session entry") {
		t.Fatalf("new session's entry not appended:\n%s", log)
	}
	if !strings.Contains(log, "- old one") {
		t.Fatalf("prior log dropped on reset (should append, not replace):\n%s", log)
	}
	// The fresh anchor now reflects the new session (committed content, lower count).
	if !strings.HasPrefix(anchor, "turns:1\n") {
		t.Fatalf("anchor not reset to the new session's turn count:\n%s", anchor)
	}
}

// On a real-change build the distiller drops a "<base>.ready" marker beside the
// log; the draft nvim fs-watches it to flash "change log ready" (#58). A no-op
// press (no new turn) must NOT drop it — the operator shouldn't be flashed for a
// build that produced nothing.
func TestReadyMarkerWrittenOnChangeOnly(t *testing.T) {
	bin := buildBinary(t)

	changeDir := t.TempDir()
	fakeClaude(t, "- entry\n")
	runIn(t, bin, changeDir, "❯ start\nL1\nL2\nL3\n"+idleFooter, "", "", "2026-06-12")
	if _, err := os.Stat(filepath.Join(changeDir, "changelog.ready")); err != nil {
		t.Fatalf("ready marker missing after a change build: %v", err)
	}

	noopDir := t.TempDir()
	fakeClaude(t, "- should not appear\n")
	runIn(t, bin, noopDir,
		"❯ a\nwork churned\nL1\nL2\nL3\n"+idleFooter,
		"## 2026-06-12\n\n- one\n", "turns:1\nL1\nL2\nL3\n", "2026-06-12")
	if _, err := os.Stat(filepath.Join(noopDir, "changelog.ready")); err == nil {
		t.Fatal("ready marker written on a no-op press")
	}
}

// A LATER press with a large gap (the agent did >800 lines of work since the
// anchor) is ALSO batched — 800 is the per-call batch size, not a first-run-only
// cap. And each batch's input stays bounded (~<= maxSliceLines) (#58).
func TestIncrementalBatchesLongGap(t *testing.T) {
	bin := buildBinary(t)
	dir := fakeClaude(t, "- entry\n")
	var b strings.Builder
	b.WriteString("❯ p1\nANCHOR1\nANCHOR2\nANCHOR3\n")
	for i := 0; i < 1000; i++ {
		b.WriteString("agent did work\n")
	}
	b.WriteString("❯ p2\n") // a new completed turn → not a no-op
	b.WriteString(idleFooter)
	priorLog := "## 2026-06-12\n\n- one\n"
	priorAnchor := "turns:1\nANCHOR1\nANCHOR2\nANCHOR3\n"
	run(t, bin, b.String(), priorLog, priorAnchor, "2026-06-12")

	if c := callCount(dir); c < 2 {
		t.Fatalf("incremental with a >800-line gap should batch; model called %d times", c)
	}
	if n := stdinLines(dir); n > 800+10 {
		t.Fatalf("a batch fed %d stdin lines, want ~<= 800 (batch size + wrapper)", n)
	}
}

// A long first-run transcript (> maxSliceLines) is distilled in MULTIPLE batches
// — not truncated to the last 800 — so the full session is covered. The model is
// called once per chunk with the accumulating log carried forward (#58).
func TestFirstRunBatchesLongTranscript(t *testing.T) {
	bin := buildBinary(t)
	dir := fakeClaude(t, "- batch entry\n")
	// 1701 committed lines after the footer is trimmed → ceil(1701/800) = 3 batches.
	var b strings.Builder
	b.WriteString("❯ start\n")
	for i := 0; i < 1700; i++ {
		b.WriteString("content line\n")
	}
	b.WriteString(idleFooter)
	log, _ := run(t, bin, b.String(), "", "", "2026-06-12")

	if c := callCount(dir); c != 3 {
		t.Fatalf("model called %d times, want 3 (1701 lines / 800 per batch)", c)
	}
	if !strings.Contains(log, "- batch entry") {
		t.Fatalf("batched log missing entries:\n%s", log)
	}
}

// Only the trailing footer changed (status-line churn) — committed content is
// identical, so trimLiveTail makes the turn count stable and the no-op fires.
// This is the #58 bug: footer churn used to break the anchor → re-distill every
// press.
func TestFooterChurnIsNoOp(t *testing.T) {
	bin := buildBinary(t)
	dir := fakeClaude(t, "- should not appear\n")
	stable := "❯ a prompt\nagent work\nANCHOR1\nANCHOR2\nANCHOR3"
	cleaned := stable + "\n❯ \n────────\n  ⏵⏵ bypass · 5 shells · NEW STATUS\n"
	priorLog := "## 2026-06-12\n\n- one\n\n- two\n"
	priorAnchor := "turns:1\nANCHOR1\nANCHOR2\nANCHOR3\n"
	log, _ := run(t, bin, cleaned, priorLog, priorAnchor, "2026-06-12")
	if invoked(dir) {
		t.Fatal("footer-only change triggered a model call (no-op should fire)")
	}
	if log != priorLog {
		t.Fatalf("log changed on no-op:\n%q", log)
	}
}
