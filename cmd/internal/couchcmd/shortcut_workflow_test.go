package couchcmd

import (
	"os"
	"path"
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
