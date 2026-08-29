package sessioninventory_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestNoNativeAuthorityShadowInInteractiveConsumers(t *testing.T) {
	t.Parallel()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate shadow test")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
	if violations, err := nativeAuthorityShadowViolations(repo); err != nil {
		t.Fatal(err)
	} else if len(violations) != 0 {
		t.Fatalf("native authority shadows: %s", strings.Join(violations, "; "))
	}
}

func TestNativeAuthorityShadowSweepRejectsSyntheticConsumer(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, "cmd", "internal", "contextcmd", "bad.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package contextcmd\nfunc bad(home string) { _ = filepath.Join(home, \".codex\", \"sessions\"); InventoryWithRuntime(nil) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := nativeAuthorityShadowViolations(repo)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(violations, "\n")
	if !strings.Contains(joined, "Codex native path") || !strings.Contains(joined, "whole native inventory") {
		t.Fatalf("synthetic violations=%q", joined)
	}
}

func nativeAuthorityShadowViolations(repo string) ([]string, error) {
	inventorySubprocess := regexp.MustCompile(`(?:vim\.fn\.system|exec\.Command)[^\n]{0,300}session-inventory[^\n]*`)
	rules := []struct {
		name string
		re   *regexp.Regexp
	}{
		{name: "Codex native path", re: regexp.MustCompile(`(?:\.codex/sessions|["']\.codex["']\s*,\s*["']sessions["'])`)},
		{name: "Claude native path", re: regexp.MustCompile(`(?:\.claude/projects|["']\.claude["']\s*,\s*["']projects["'])`)},
		{name: "Agy native path", re: regexp.MustCompile(`antigravity-cli["'/,\s]+(?:brain|conversations)`)},
		{name: "Muse native path", re: regexp.MustCompile(`(?:\.local/share/muse/sessions|["']muse["']\s*,\s*["']sessions["'])`)},
		{name: "whole native inventory", re: regexp.MustCompile(`\b(?:InventoryWithRuntime|NativeEventsWithRuntime)\s*\(`)},
		{name: "direct lsof command", re: regexp.MustCompile(`exec\.Command\(["']lsof["']`)},
		{name: "retired transcript resolver", re: regexp.MustCompile(`(?:ReadCodexRootSessionID|CodexSessionIDFromPath|ResolveCodexSessionID|resolveLiveCodexTranscript)`)},
		{name: "config filename authority scan", re: regexp.MustCompile(`\.ConfigGlob\(\)`)},
		{name: "native transcript parser", re: regexp.MustCompile(`(?:parseTranscript|parseClaude|parseCodex|parseAgy|parseMuse|claudeEntry|codexEntry|agyEntry|museEnvelope)`)},
	}
	var violations []string
	for _, root := range []string{"cmd", "nvim", "bin"} {
		err := filepath.WalkDir(filepath.Join(repo, root), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			rel, err := filepath.Rel(repo, path)
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if rel == "cmd/internal/sessioninventory" || strings.Contains(rel, string(filepath.Separator)+"testdata") {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, "_test.go") || !governedShadowSource(path, root) {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if root == "bin" && !bytes.HasPrefix(raw, []byte("#!")) {
				return nil
			}
			for _, rule := range rules {
				if rel == "cmd/internal/procutil/procutil.go" && rule.name == "direct lsof command" {
					continue
				}
				if rule.re.Match(raw) {
					violations = append(violations, filepath.ToSlash(rel)+" contains "+rule.name)
				}
			}
			for _, match := range inventorySubprocess.FindAll(raw, -1) {
				if !bytes.Contains(match, []byte("--owner")) {
					violations = append(violations, filepath.ToSlash(rel)+" contains whole inventory subprocess")
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return violations, nil
}

func governedShadowSource(path, root string) bool {
	switch root {
	case "cmd":
		return strings.HasSuffix(path, ".go")
	case "nvim":
		return strings.HasSuffix(path, ".lua")
	case "bin":
		return filepath.Ext(path) == "" || strings.HasSuffix(path, ".sh")
	default:
		return false
	}
}
