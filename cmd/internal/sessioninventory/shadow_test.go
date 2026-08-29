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

func TestShadowSweep(t *testing.T) {
	t.Parallel()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate shadow test")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", ".."))
	rules := []struct {
		name string
		re   *regexp.Regexp
	}{
		{name: "Codex native path", re: regexp.MustCompile(`(?:\.codex/sessions|["']\.codex["']\s*,\s*["']sessions["'])`)},
		{name: "Claude native path", re: regexp.MustCompile(`(?:\.claude/projects|["']\.claude["']\s*,\s*["']projects["'])`)},
		{name: "Agy native path", re: regexp.MustCompile(`antigravity-cli["'/,\s]+(?:brain|conversations)`)},
		{name: "Muse native path", re: regexp.MustCompile(`(?:\.local/share/muse/sessions|["']muse["']\s*,\s*["']sessions["'])`)},
		{name: "direct lsof command", re: regexp.MustCompile(`exec\.Command\(["']lsof["']`)},
		{name: "retired transcript resolver", re: regexp.MustCompile(`(?:ReadCodexRootSessionID|CodexSessionIDFromPath|ResolveCodexSessionID|resolveLiveCodexTranscript)`)},
		{name: "config filename authority scan", re: regexp.MustCompile(`\.ConfigGlob\(\)`)},
		{name: "native transcript parser", re: regexp.MustCompile(`(?:parseTranscript|parseClaude|parseCodex|parseAgy|parseMuse|claudeEntry|codexEntry|agyEntry|museEnvelope)`)},
	}

	for _, root := range []string{"cmd", "nvim", "bin"} {
		err := filepath.WalkDir(filepath.Join(repo, root), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
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
					t.Errorf("%s contains %s outside sessioninventory", rel, rule.name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
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
