package launcher

import "testing"

func TestLegacyImportPlanCopiesMissingScopedFilesOnly(t *testing.T) {
	exists := map[string]bool{
		"/global/draft-work.md":                true,
		"/global/config-work-claude.json":      true,
		"/global/scrollback-work-claude.raw":   true,
		"/scoped/config-work-claude.json":      true,
		"/global/draft-work-2.md":              true,
		"/global/scrollback-work-2-claude.raw": true,
	}
	pairs := legacyImportPlan("work", "/global", "/scoped", func(path string) bool {
		return exists[path]
	})
	got := map[string]string{}
	for _, pair := range pairs {
		got[pair.src] = pair.dst
	}
	if got["/global/draft-work.md"] != "/scoped/draft-work.md" {
		t.Fatalf("draft pair missing from plan: %#v", pairs)
	}
	if got["/global/scrollback-work-claude.raw"] != "/scoped/scrollback-work-claude.raw" {
		t.Fatalf("scrollback pair missing from plan: %#v", pairs)
	}
	if _, ok := got["/global/config-work-claude.json"]; ok {
		t.Fatalf("occupied scoped config must not be overwritten: %#v", pairs)
	}
	if _, ok := got["/global/draft-work-2.md"]; ok {
		t.Fatalf("exact tag import must not copy work-2 family: %#v", pairs)
	}
}

func TestImportLegacyFlatTagDoesNotOverwriteScopedQueueFiles(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/global/queue-work/000001.md"] = "legacy queued prompt"
	rt.files["/scoped/queue-work/000001.md"] = "scoped queued prompt"
	rt.files["/global/queue-work/000002.md"] = "second legacy prompt"

	if !importLegacyFlatTag(rt, "work", "/global", "/scoped") {
		t.Fatalf("legacy import should copy at least one missing queue file")
	}
	if got := rt.files["/scoped/queue-work/000001.md"]; got != "scoped queued prompt" {
		t.Fatalf("existing scoped queue file overwritten: %q", got)
	}
	if got := rt.files["/scoped/queue-work/000002.md"]; got != "second legacy prompt" {
		t.Fatalf("missing scoped queue file not copied: %q", got)
	}
}
