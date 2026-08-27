package artifactpath

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompositeBindingsIsolateGoShellNeovimAndBothLayouts(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	dataDir := t.TempDir()
	first, err := Resolve(Address{DataDir: dataDir, RepoScope: "aaaaaaaaaaaaaaaa", Tag: "legacy"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Resolve(Address{DataDir: dataDir, RepoScope: "bbbbbbbbbbbbbbbb", Tag: "legacy"})
	if err != nil {
		t.Fatal(err)
	}
	for _, paths := range []Paths{first, second} {
		if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	// Representative Go mutation: the same legacy tag addresses independent
	// drafts once the selected repository scope differs.
	if err := os.WriteFile(first.Draft(), []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(second.Draft()); !os.IsNotExist(err) {
		t.Fatalf("Go mutation crossed scopes: second draft stat = %v", err)
	}

	// Representative shell mutation consumes only the exact exported path.
	shell := exec.Command("bash", "-c", `. bin/lib/adapt-log.sh; adapt_log test codex 1 scope isolated shell`)
	shell.Dir = repoRoot
	shell.Env = append(os.Environ(), "PAIR_TAG=legacy", "PAIR_ADAPT_LOG_PATH="+first.AdaptLog())
	if out, err := shell.CombinedOutput(); err != nil {
		t.Fatalf("shell consumer: %v: %s", err, out)
	}
	if _, err := os.Stat(second.AdaptLog()); !os.IsNotExist(err) {
		t.Fatalf("shell mutation crossed scopes: second adapt stat = %v", err)
	}

	// Representative Neovim mutation uses the other exact binding. It must not
	// append to the first scope even though PAIR_TAG is identical.
	lua := `local adapt = dofile('nvim/adapt.lua'); adapt.log(1, 'scope', 'isolated', 'nvim', 'test')`
	nvim := exec.Command("nvim", "-l", "-")
	nvim.Dir = repoRoot
	nvim.Stdin = strings.NewReader(lua)
	nvim.Env = append(os.Environ(), "PAIR_TAG=legacy", "PAIR_ADAPT_LOG_PATH="+second.AdaptLog())
	if out, err := nvim.CombinedOutput(); err != nil {
		t.Fatalf("Neovim consumer: %v: %s", err, out)
	}
	firstRaw, err := os.ReadFile(first.AdaptLog())
	if err != nil {
		t.Fatal(err)
	}
	secondRaw, err := os.ReadFile(second.AdaptLog())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(strings.TrimSpace(string(firstRaw)), "\n") != 0 || strings.Count(strings.TrimSpace(string(secondRaw)), "\n") != 0 {
		t.Fatalf("each scoped consumer should write exactly one record: first=%q second=%q", firstRaw, secondRaw)
	}
	if !strings.Contains(string(firstRaw), `"detail":"shell"`) || !strings.Contains(string(secondRaw), `"detail":"nvim"`) {
		t.Fatalf("scoped mutations observed wrong records: first=%q second=%q", firstRaw, secondRaw)
	}

	firstBindings := bindingMap(t, first, "codex")
	secondBindings := bindingMap(t, second, "codex")
	for _, name := range []string{"PAIR_DRAFT_PATH", "PAIR_AGENT_PANE_PATH", "PAIR_SCROLLBACK_RAW_PATH"} {
		if firstBindings[name] == secondBindings[name] {
			t.Fatalf("%s collided across scopes: %q", name, firstBindings[name])
		}
	}
	for _, layout := range []string{"main-2.kdl", "main-3.kdl"} {
		raw, err := os.ReadFile(filepath.Join(repoRoot, "zellij", "layouts", layout))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, binding := range []string{"$PAIR_DRAFT_PATH", "$PAIR_AGENT_PANE_PATH", "$PAIR_SCROLLBACK_RAW_PATH"} {
			if !strings.Contains(text, binding) {
				t.Errorf("%s does not consume exact binding %s", layout, binding)
			}
		}
		for _, reconstruction := range []string{"draft-$PAIR_TAG", "pane-$PAIR_TAG", "scrollback-$PAIR_TAG"} {
			if strings.Contains(text, reconstruction) {
				t.Errorf("%s reconstructs %q instead of consuming its scoped binding", layout, reconstruction)
			}
		}
	}
}

func bindingMap(t *testing.T, paths Paths, agent string) map[string]string {
	t.Helper()
	bindings, err := paths.EnvironmentBindings(agent)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		out[binding.Name] = binding.Path
	}
	return out
}
