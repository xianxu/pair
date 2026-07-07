package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// End-to-end render→distill SEAM test (#59, relocated from the deleted
// cmd/pair-changelog/e2e_test.go in #104 M3). The render's ⟦pair:ts DATE⟧ markers
// must survive through the REAL `pair scrollback render` → `pair changelog render`
// pipe — the wire contract `tsMarkerLine` (scrollbackcmd) ↔ `tsMarkerRe`
// (changelogcmd). The two sides are otherwise only guarded by independent
// hand-typed literals that never cross-check, so this is the one test that would
// catch a delimiter/format drift between them. Drives the single `pair` binary so
// it also guards the `scrollback render` / `changelog render` nested routes.
func TestChangelogSeamMarkerSurvival(t *testing.T) {
	bin := t.TempDir()
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairGo, ".")

	// Fake `claude` on PATH: `changelog render` shells the model, which must echo
	// a non-empty distilled body (the markers → dated headers happen in the Go
	// assembly around the model output, not in the model itself).
	fakeDir := t.TempDir()
	// The heredoc emits REAL newlines (a printf '%s' with \n would print them
	// literally and collapse the distiller's per-day segments into one line).
	script := "#!/bin/sh\ncat >/dev/null\ncat <<'EOF'\n- worked\n\n- more\nEOF\n"
	if err := os.WriteFile(filepath.Join(fakeDir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "PATH="+fakeDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	dir := t.TempDir()
	rawPath := filepath.Join(dir, "s.raw")
	evPath := filepath.Join(dir, "s.events.jsonl")
	cleanedPath := filepath.Join(dir, "s.cleaned")
	logPath := filepath.Join(dir, "changelog.md")
	anchorPath := filepath.Join(dir, "changelog.anchor")

	// 40 lines so scrollback forms (default 80x24); time events for two days at
	// offsets that land in different scrollback regions.
	var raw strings.Builder
	var off10, off30 int
	for i := 0; i < 40; i++ {
		if i == 10 {
			off10 = raw.Len()
		}
		if i == 30 {
			off30 = raw.Len()
		}
		fmt.Fprintf(&raw, "line %02d\r\n", i)
	}
	mustWrite(t, rawPath, raw.String())
	events := `{"type":"resize","offset":0,"cols":80,"rows":24}` + "\n" +
		fmt.Sprintf(`{"type":"time","offset":%d,"ts":"2026-06-13T09:00:00Z"}`, off10) + "\n" +
		fmt.Sprintf(`{"type":"time","offset":%d,"ts":"2026-06-14T09:00:00Z"}`, off30) + "\n"
	mustWrite(t, evPath, events)

	// Real render step, via the nested `pair scrollback render` route.
	r := runCommand(t, env, pairGo, "scrollback", "render", "--plain", "--max-lines", "0",
		"--with-timestamps", rawPath, evPath, cleanedPath)
	if r.code != 0 {
		t.Fatalf("scrollback render exit=%d stderr=%q", r.code, r.stderr)
	}
	cleaned := readFileOr(cleanedPath)
	if !strings.Contains(cleaned, "⟦pair:ts 2026-06-13⟧") || !strings.Contains(cleaned, "⟦pair:ts 2026-06-14⟧") {
		t.Fatalf("render did not emit both day markers:\n%s", cleaned)
	}

	// Real distill step over the rendered cleaned, via `pair changelog render`.
	d := runCommand(t, env, pairGo, "changelog", "render", "--cleaned", cleanedPath,
		"--log", logPath, "--anchor", anchorPath, "--agent", "claude")
	if d.code != 0 {
		t.Fatalf("changelog render exit=%d stderr=%q", d.code, d.stderr)
	}
	log := readFileOr(logPath)
	i13 := strings.Index(log, "## 2026-06-13")
	i14 := strings.Index(log, "## 2026-06-14")
	if i13 < 0 || i14 < 0 || i13 >= i14 {
		t.Fatalf("markers did not survive into dated headers (13=%d 14=%d):\n%s", i13, i14, log)
	}
	if strings.Contains(log, "⟦pair:ts") {
		t.Fatalf("ts marker leaked into the change log:\n%s", log)
	}
}

func readFileOr(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
