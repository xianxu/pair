package runtimebundle

import "testing"

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
		"zellij/layouts/main.kdl",
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
