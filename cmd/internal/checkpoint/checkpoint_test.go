package checkpoint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validBody = "---\ntype: continuation\nagent: codex\n---\n\n## NEXT ACTION\nContinue the exact task.\n"

func TestCheckpointValidation(t *testing.T) {
	c, err := New("/repo/checkpoint.md", validBody)
	if err != nil || c.Validate() != nil || c.Agent() != "codex" {
		t.Fatalf("valid checkpoint: %+v %v", c, err)
	}
	for name, mutate := range map[string]func(*Checkpoint){
		"digest":   func(c *Checkpoint) { c.Digest = strings.Repeat("0", 64) },
		"version":  func(c *Checkpoint) { c.Version++ },
		"relative": func(c *Checkpoint) { c.SourcePath = "relative.md" },
		"unclean":  func(c *Checkpoint) { c.SourcePath = "/repo/../checkpoint.md" },
		"control":  func(c *Checkpoint) { c.SourcePath = "/repo/line\nbreak.md" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := c
			mutate(&bad)
			if bad.Validate() == nil {
				t.Fatal("accepted invalid checkpoint")
			}
		})
	}
	for _, body := range []string{"", "\xff", strings.Repeat("x", MaxBytes+1), "## NEXT ACTION\nnot a continuation\n", "---\ntype: continuation\nagent: codex\n---\n## NEXT ACTION\n## empty\n"} {
		if _, err := New("/repo/checkpoint.md", body); err == nil {
			t.Fatalf("accepted invalid body length %d", len(body))
		}
	}
	if _, err := New("/repo/checkpoint.md", "---\ntype: continuation\n## NEXT ACTION\nagent: codex\n---\nno action section\n"); err == nil {
		t.Fatal("frontmatter comment counted as NEXT ACTION")
	}
	boundary := validBody + strings.Repeat("x", MaxBytes-len(validBody))
	if _, err := New("/repo/checkpoint.md", boundary); err != nil {
		t.Fatal(err)
	}
}

func TestCheckpointReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "checkpoint.md")
	if err := os.WriteFile(path, []byte(validBody), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := ReadFile(path)
	if err != nil || c.Body != validBody || c.SourcePath != path {
		t.Fatalf("read: %+v %v", c, err)
	}
	for _, invalid := range []string{dir, filepath.Join(dir, "absent")} {
		if _, err := ReadFile(invalid); err == nil {
			t.Fatalf("accepted %s", invalid)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", MaxBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFile(path); err == nil {
		t.Fatal("accepted oversized source")
	}
}
