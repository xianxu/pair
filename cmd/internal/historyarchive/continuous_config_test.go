package historyarchive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContinuousConfigAbsentStrictAndAbsolute(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ContinuousConfigName)
	if c, err := LoadContinuousConfig(path); c != nil || err != nil {
		t.Fatalf("absent config: %+v %v", c, err)
	}
	fp := strings.Repeat("A", 40)
	good := `{"version":1,"recipients":["` + fp + `"],"sink":"/s","catalog":"/c/catalog.jsonl","ops_root":"/ops","workspace":"/w"}`
	for name, body := range map[string]string{
		"good":     good,
		"unknown":  strings.Replace(good, `"version":1`, `"version":1,"extra":true`, 1),
		"relative": strings.Replace(good, `"/s"`, `"s"`, 1),
		"no-recip": strings.Replace(good, `["`+fp+`"]`, `[]`, 1),
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		c, err := LoadContinuousConfig(path)
		if name == "good" {
			if err != nil || c == nil || c.Sink != "/s" || c.OpsRoot != "/ops" {
				t.Fatalf("good config: %+v %v", c, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s config accepted", name)
		}
	}
}

func TestInventoryContinuousPatterns(t *testing.T) {
	inv, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	codex, _ := inv.Resolve("codex-transcripts")
	if !strings.Contains(strings.Join(codex.Patterns, ","), "history.jsonl") {
		t.Fatalf("codex patterns %v lack history.jsonl", codex.Patterns)
	}
	ops, _ := inv.Resolve("ops-documents")
	for _, p := range ops.Patterns {
		if p == "*" {
			t.Fatalf("ops patterns %v still archive every team file", ops.Patterns)
		}
	}
}
