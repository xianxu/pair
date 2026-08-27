package runtimebundle

import (
	"strings"
	"testing"
)

// Since #104 M3 the runtime bundle is config + shell shims ONLY — no helper
// binaries. Every former helper is a `pair <subcommand>` reached via the single
// `pair` on the session PATH; nothing named `pair-*` (a Go binary) is bundled.
func TestEmbeddedManifestIsConfigAndShimsOnly(t *testing.T) {
	manifest := EmbeddedManifest()
	paths := map[string]bool{}
	for _, asset := range manifest.Assets {
		paths[asset.Path] = true
	}
	for _, want := range []string{
		"bin/pair-help",   // shell shim (invoked by bare name in a session)
		"bin/pair-notify", // shell shim (Claude hooks)
		"bin/lib/dev-rebuild.sh",
		"nvim/init.lua",
		"nvim/review/init.lua",
		"zellij/config.kdl",
		"zellij/layouts/main-2.kdl",
		"zellij/layouts/main-3.kdl",
		"doctor/SKILL.md",
		"doctor/doctor.sh",
	} {
		if !paths[want] {
			t.Fatalf("EmbeddedManifest missing %q", want)
		}
	}
	for _, excluded := range []string{
		"bin/pair",    // never self-embed
		"bin/pair-go", // legacy twin, dropped #104 M3
		"bin/pair-dev",
		// #104 M3 — the helper binaries fold into `pair <sub>`; none are bundled.
		"bin/pair-wrap",
		"bin/pair-slug",
		"bin/pair-title",
		"bin/pair-session-watch",
		"bin/pair-context",
		"bin/pair-continuation",
		"bin/pair-scrollback-render",
		"bin/pair-scrollback-open",
		"bin/pair-changelog",
		"bin/pair-changelog-open",
		"bin/pair-review-open",
		"bin/pair-review-readiness",
		"bin/pair-review-target",
		"bin/copy-on-select",
		"bin/clipboard-to-pane",
		"bin/flash-pane",
		"bin/pair-scribe", // folds to `pair scribe`
		"nvim/init_test.lua",
	} {
		if paths[excluded] {
			t.Fatalf("EmbeddedManifest includes excluded path %q", excluded)
		}
	}
}

func TestEmbeddedMainLayoutsConsumeExactArtifactBindings(t *testing.T) {
	for _, path := range []string{"zellij/layouts/main-2.kdl", "zellij/layouts/main-3.kdl"} {
		data, err := EmbeddedAsset(path)
		if err != nil {
			t.Fatalf("EmbeddedAsset(%s): %v", path, err)
		}
		for _, binding := range []string{"$PAIR_DRAFT_PATH", "$PAIR_NVIM_DRAFT_PID_PATH", "$PAIR_AGENT_PANE_PATH", "$PAIR_SCROLLBACK_RAW_PATH"} {
			if !strings.Contains(string(data), binding) {
				t.Fatalf("%s must consume exact binding %s", path, binding)
			}
		}
	}
}

func TestEmbeddedNvimLayoutStateConsumesExactArtifactBinding(t *testing.T) {
	data, err := EmbeddedAsset("nvim/init.lua")
	if err != nil {
		t.Fatalf("EmbeddedAsset(nvim/init.lua): %v", err)
	}
	init := string(data)
	if !strings.Contains(init, "local LAYOUT_STATE_FILE = vim.env.PAIR_LAYOUT_MODE_PATH") {
		t.Fatalf("layout state must consume PAIR_LAYOUT_MODE_PATH")
	}
}
