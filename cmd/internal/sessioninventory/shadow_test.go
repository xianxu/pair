package sessioninventory_test

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
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
	path := filepath.Join(repo, "cmd", "internal", "sessioninventory", "bad.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`package sessioninventory
func direct(runtime Runtime) { InventoryWithRuntime(runtime) }
func alias(runtime Runtime) { scan := InventoryWithRuntime; scan(runtime) }
func selector(runtime Runtime) { scan := inventory.InventoryWithRuntime; scan(runtime) }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := nativeAuthorityShadowViolations(repo)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(violations, "\n")
	for _, want := range []string{
		"cmd/internal/sessioninventory/bad.go:direct:InventoryWithRuntime",
		"cmd/internal/sessioninventory/bad.go:alias:InventoryWithRuntime",
		"cmd/internal/sessioninventory/bad.go:selector:InventoryWithRuntime",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("synthetic violations=%q, missing %q", joined, want)
		}
	}
}

func nativeAuthorityShadowViolations(repo string) ([]string, error) {
	inventorySubprocess := regexp.MustCompile(`(?:vim\.fn\.system|exec\.Command)[^\n]{0,300}session-inventory[^\n]*`)
	allowedCalls := map[string]bool{
		"cmd/internal/sessioninventory/runcli.go:runCLIOptionsWithRenderers:InventoryWithRuntime":     true,
		"cmd/internal/sessioninventory/pair_inventory.go:RecoverPairBindings:NativeEventsWithRuntime": true,
	}
	seenCalls := make(map[string]bool, len(allowedCalls))
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
				if strings.Contains(rel, string(filepath.Separator)+"testdata") {
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
			if root == "cmd" {
				calls, err := wholeInventoryReferences(path, filepath.ToSlash(rel))
				if err != nil {
					return err
				}
				for _, call := range calls {
					if allowedCalls[call] {
						seenCalls[call] = true
					} else {
						violations = append(violations, call+" contains whole native inventory")
					}
				}
			}
			if strings.HasPrefix(filepath.ToSlash(rel), "cmd/internal/sessioninventory/") {
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
	for call := range allowedCalls {
		if !seenCalls[call] {
			violations = append(violations, "stale whole-inventory allowlist entry "+call)
		}
	}
	return violations, nil
}

func wholeInventoryReferences(path, relative string) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, err
	}
	var references []string
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if ok && (identifier.Name == "InventoryWithRuntime" || identifier.Name == "NativeEventsWithRuntime") {
				references = append(references, relative+":"+function.Name.Name+":"+identifier.Name)
			}
			return true
		})
	}
	return references, nil
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
