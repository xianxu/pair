package couchcmd

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// Keep the real protocol check scheduled when any shortcut input owner changes.
func TestShortcutConformanceWorkflowSourceTriggers(t *testing.T) {
	data, err := os.ReadFile("../../../.github/workflows/couch-zellij-conformance.yml")
	if err != nil {
		t.Fatal(err)
	}
	sources := []string{
		"cmd/internal/wrapcmd/wrap.go",
		"cmd/internal/couchtty/keys.go",
		"cmd/internal/couchtty/console.go",
		"nvim/init.lua",
		"nvim/workbench_actions.lua",
		"cmd/internal/workbenchshortcut/framing.go",
		"zellij/config.kdl",
		"cmd/internal/pairlifecycletest/live_zellij.go",
		"cmd/internal/couchcmd/shortcut_conformance_live_test.go",
	}
	// Read the workflow's block-style event/path lists. Fail closed if the
	// syntax changes, so a refactor cannot silently disable this assertion.
	events := map[string][]string{}
	event := ""
	inPaths := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") {
			event = strings.TrimSuffix(strings.TrimSpace(line), ":")
			inPaths = false
		}
		if line == "    paths:" {
			inPaths = true
			continue
		}
		if inPaths && strings.HasPrefix(line, "      - \"") {
			events[event] = append(events[event], strings.TrimSuffix(strings.TrimPrefix(line, "      - \""), "\""))
		}
	}
	for _, event := range []string{"pull_request", "push"} {
		t.Run(event, func(t *testing.T) {
			for _, source := range sources {
				matched := false
				for _, pattern := range events[event] {
					// These selectors use literal paths, filename globs and subtree /**.
					if strings.HasSuffix(pattern, "/**") {
						matched = matched || strings.HasPrefix(source, strings.TrimSuffix(pattern, "**"))
					} else {
						ok, err := path.Match(pattern, source)
						if err != nil {
							t.Fatal(err)
						}
						matched = matched || ok
					}
				}
				if !matched {
					t.Errorf("%s does not trigger conformance", source)
				}
			}
		})
	}
}

// Hosted CI checks out Pair without its sibling ariadne repository. Exercise
// the actual workflow command in that shape, without starting live processes.
func TestConformanceWorkflowRunsWithoutSiblingMakefile(t *testing.T) {
	workflow, err := os.ReadFile("../../../.github/workflows/couch-zellij-conformance.yml")
	if err != nil {
		t.Fatal(err)
	}
	var argv []string
	for _, line := range strings.Split(string(workflow), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "run: ") || !strings.Contains(line, "test-couch-zellij-live") {
			continue
		}
		if argv != nil {
			t.Fatal("multiple conformance commands; update the fixture")
		}
		argv = strings.Fields(strings.TrimPrefix(line, "run: "))
	}
	if len(argv) < 2 || argv[0] != "make" || argv[len(argv)-1] != "test-couch-zellij-live" {
		t.Fatalf("unsupported conformance workflow command: %q", argv)
	}
	// Only the recipe file exists. Makefile has the same deliberately dangling
	// link as a checkout without ../ariadne; do not repair it in the fixture.
	fixture := filepath.Join(t.TempDir(), "pair")
	if err := os.Mkdir(fixture, 0700); err != nil {
		t.Fatal(err)
	}
	local, err := os.ReadFile("../../../Makefile.local")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "Makefile.local"), local, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../ariadne/Makefile", filepath.Join(fixture, "Makefile")); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(argv[0], append([]string{"-n"}, argv[1:]...)...)
	command.Dir = fixture
	// Inherited make flags/includes must not supply the missing sibling or run
	// unrelated targets. All other environment remains available for make itself.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key != "MAKEFLAGS" && key != "GNUMAKEFLAGS" && key != "MAKEFILES" {
			command.Env = append(command.Env, entry)
		}
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("standalone conformance dry-run: %v\n%s", err, output)
	}
	expected := map[string]string{
		"./cmd/internal/launcher":  "TestSessionQuiescenceLive",
		"./cmd/internal/couchcore": "TestRecoveryRealHelperAndSessionConformanceLive",
		"./cmd/internal/couchcmd":  "TestAgentShortcutInputConformanceLive",
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if !strings.HasPrefix(line, "PAIR_LIVE_COUCH=1 go test ") {
			t.Fatalf("unexpected dry-run command %q", line)
		}
		count++
		matched := false
		for pkg, selector := range expected {
			if strings.Contains(line, " "+pkg+" ") && strings.Contains(line, selector) {
				delete(expected, pkg)
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("unexpected or duplicate conformance command %q", line)
		}
	}
	if count != 3 || len(expected) != 0 {
		t.Fatalf("expected three conformance commands, got %d; missing %v", count, expected)
	}
}

// Hosted macOS images need not provide Go. The workflow must provision the
// repository's declared toolchain before any recipe invokes it.
func TestConformanceWorkflowProvisionsModuleToolchain(t *testing.T) {
	workflow, err := os.ReadFile("../../../.github/workflows/couch-zellij-conformance.yml")
	if err != nil {
		t.Fatal(err)
	}
	checkedOut, setup, versionFile := false, false, ""
	for _, block := range strings.Split(string(workflow), "\n      - ") {
		if strings.Contains(block, "uses: actions/checkout@") {
			checkedOut = true
		}
		if strings.Contains(block, "uses: actions/setup-go@") {
			if !checkedOut {
				t.Fatal("Go setup cannot read module version before checkout")
			}
			setup = true
			for _, line := range strings.Split(block, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "go-version-file:") {
					versionFile = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "go-version-file:")), "\"'")
				}
			}
		}
		if strings.Contains(block, "run:") && strings.Contains(block, "test-couch-zellij-live") {
			if !setup || versionFile != "go.mod" {
				t.Fatalf("conformance recipe needs Go setup from go.mod first: setup=%v versionFile=%q", setup, versionFile)
			}
			module, err := os.ReadFile(filepath.Join("../../..", versionFile))
			if err != nil || !strings.Contains(string(module), "\ngo ") {
				t.Fatalf("Go setup version source lacks module toolchain: %v", err)
			}
			return
		}
	}
	t.Fatal("conformance execution step missing")
}
