package keyhelp

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// EVERY pair: keymap in the real init.lua must be CLASSIFIED — surfaced in the
// catalog or explicitly marked internal. A new user-facing key therefore cannot
// reach a release undocumented, and an editor internal cannot leak into user help.
//
// This is the anti-rot invariant #132 exists to install. #99 M5c had no such test,
// which is exactly why the help silently emptied when bin/pair-shell was retired.
//
// Reads the TREE copy, not the embedded bundle: assets/ is gitignored and `go test`
// never regenerates it, so classifying against the embedded copy would validate a
// stale snapshot (add a keymap, run go test, get green, ship undocumented).
func TestEveryNvimKeymapIsClassified(t *testing.T) {
	scan := ParseNvimKeymaps(mustReadTreeSource(t, "nvim/init.lua"))
	for _, km := range scan.Resolved {
		if !Catalog.Includes(km.Key) && !Catalog.IsInternal(km.Key) {
			t.Errorf("keymap %q (%q) is neither in the help catalog nor marked internal — classify it in catalog.go", km.Key, km.Desc)
		}
	}
	// Dynamic rows are keyed by their raw lhs text; they are real user-facing keys
	// (Alt+1..9 completion picks) and must be classified too.
	for _, km := range scan.Dynamic {
		if !Catalog.Includes(km.Raw) && !Catalog.IsInternal(km.Raw) {
			t.Errorf("dynamic keymap %q (%q) is unclassified", km.Raw, km.Desc)
		}
	}
}

// Global chords never appear in init.lua as literal keymap.set calls (they arrive
// via the generated workbench_actions.lua), so the test above cannot see them.
// Without this, adding a global chord would silently miss the help.
func TestEveryGlobalChordIsClassified(t *testing.T) {
	for _, b := range workbenchshortcut.GlobalBindings() {
		if !Catalog.Includes(b.NvimKey) && !Catalog.IsInternal(b.NvimKey) {
			t.Errorf("global chord %q (%q) is unclassified", b.NvimKey, b.Help)
		}
	}
}

// Both zellij-level Run binds must be documented; they are invisible to nvim, so
// nothing else would ever surface them. Alt+h documenting ITSELF is the point.
func TestZellijRunBindsAreDocumented(t *testing.T) {
	for _, z := range ParseZellijRunBinds(mustReadTreeSource(t, "zellij/config.kdl")) {
		if !Catalog.Includes(z.Key) {
			t.Errorf("zellij Run bind %q is undocumented", z.Key)
		}
	}
}

// The reverse direction: a catalog entry whose source row is gone is stale help —
// the same class of lie as missing help, pointing the other way.
func TestEveryCatalogEntryStillExists(t *testing.T) {
	live := map[string]bool{}
	scan := ParseNvimKeymaps(mustReadTreeSource(t, "nvim/init.lua"))
	for _, km := range scan.Resolved {
		live[km.Key] = true
	}
	for _, km := range scan.Dynamic {
		live[km.Raw] = true
	}
	for _, b := range workbenchshortcut.GlobalBindings() {
		live[b.NvimKey] = true
	}
	for _, rb := range workbenchshortcut.RoleBindings() {
		live[roleChordKey(rb.Chord)] = true
	}
	for _, z := range ParseZellijRunBinds(mustReadTreeSource(t, "zellij/config.kdl")) {
		live[z.Key] = true
	}
	for _, key := range Catalog.AllKeys() {
		if !live[key] {
			t.Errorf("catalog references %q (include or internal) but no source defines it any more — remove or repoint it", key)
		}
	}
}

// The classification tests above read the TREE copies on purpose. This is the only
// thing tying the shipped bundle to them: assets/ is gitignored and `go test` never
// regenerates it, so without this a stale embedded snapshot would silently weaken
// every assertion above while `pair keys` rendered yesterday's bindings.
func TestEmbeddedSourcesMatchTree(t *testing.T) {
	for _, path := range []string{"nvim/init.lua", "zellij/config.kdl"} {
		tree := mustReadTreeSource(t, path)
		embedded, err := runtimebundle.EmbeddedAsset(path)
		if err != nil {
			t.Fatalf("%s missing from the runtime bundle: %v", path, err)
		}
		if string(embedded) != tree {
			t.Errorf("%s: embedded bundle is stale — run `make build` to regenerate", path)
		}
	}
}
