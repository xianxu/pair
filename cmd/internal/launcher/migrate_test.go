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
