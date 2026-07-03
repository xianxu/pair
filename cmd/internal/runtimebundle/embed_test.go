package runtimebundle

import "testing"

func TestEmbeddedManifestContainsLaunchAssets(t *testing.T) {
	manifest := EmbeddedManifest()
	paths := map[string]bool{}
	for _, asset := range manifest.Assets {
		paths[asset.Path] = true
	}
	for _, want := range []string{
		"bin/pair-help",
		"bin/pair-title.sh",
		"bin/pair-session-watch.sh",
		"bin/lib/dev-rebuild.sh",
		"bin/pair-wrap",
		"bin/pair-slug",
		"bin/pair-context",
		"bin/pair-scrollback-render",
		"bin/pair-changelog",
		"bin/pair-continuation",
		"bin/pair-session-watch",
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
		"bin/pair",
		"bin/pair-go",
		"bin/pair-dev",
		"bin/pair-quit.sh",    // #94 M1 — ported to `pair quit`, no longer bundled
		"bin/pair-restart.sh", // #94 M1 — ported to `pair restart`, no longer bundled
		"nvim/init_test.lua",
	} {
		if paths[excluded] {
			t.Fatalf("EmbeddedManifest includes excluded path %q", excluded)
		}
	}
}
