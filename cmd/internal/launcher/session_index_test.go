package launcher

import (
	"strings"
	"testing"
)

func TestAssignSessionNameUsesReadableBaseWhenFree(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	name, _, err := AssignSessionName(SessionNameIndex{}, nil, scope, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if name != "pair-pair-work" {
		t.Fatalf("name = %q, want pair-pair-work", name)
	}
	if strings.Contains(name, scope.Key) {
		t.Fatalf("name %q exposed hidden key %q", name, scope.Key)
	}
}

func TestAssignSessionNameDisambiguatesSameRepoNameDifferentScope(t *testing.T) {
	first := mustScope(t, "/Users/a/work/pair")
	second := mustScope(t, "/tmp/other/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "pair-pair-work",
		ScopeKey:    first.Key,
		RepoRoot:    first.Root,
		RepoName:    first.DisplayName,
		Tag:         "work",
	}}}

	name, _, err := AssignSessionName(index, []Session{{Name: "pair-pair-work", State: SessionDetached}}, second, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if name != "pair-pair-work-2" {
		t.Fatalf("name = %q, want pair-pair-work-2", name)
	}
}

func TestAssignSessionNameReusesSameScopeBinding(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "pair-pair-work-2",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "work",
	}}}

	name, _, err := AssignSessionName(index, []Session{{Name: "pair-pair-work-2", State: SessionDetached}}, scope, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if name != "pair-pair-work-2" {
		t.Fatalf("name = %q, want prior binding", name)
	}
}

func TestAssignSessionNameShortensOverlongReadableName(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/repositorywithaverylongname")
	name, _, err := AssignSessionName(SessionNameIndex{}, nil, scope, "featurewithaverylongname", func(s string) bool {
		return len(s) <= 24
	})
	if err != nil {
		t.Fatalf("AssignSessionName returned error: %v", err)
	}
	if len(name) > 24 {
		t.Fatalf("name = %q len=%d, want <= 24", name, len(name))
	}
	if strings.Contains(name, scope.Key) {
		t.Fatalf("name %q exposed hidden key %q", name, scope.Key)
	}
	if !strings.HasPrefix(name, "pair-") {
		t.Fatalf("name = %q, want pair prefix", name)
	}
}

func TestAssignSessionNameErrorsWhenNoCandidateFits(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	if _, _, err := AssignSessionName(SessionNameIndex{}, nil, scope, "work", func(string) bool { return false }); err == nil {
		t.Fatal("AssignSessionName returned nil error")
	}
}

func acceptAllSessionNames(string) bool { return true }

func mustScope(t *testing.T, root string) RepoScope {
	t.Helper()
	scope, err := ResolveRepoScope(root)
	if err != nil {
		t.Fatalf("ResolveRepoScope(%q): %v", root, err)
	}
	return scope
}
