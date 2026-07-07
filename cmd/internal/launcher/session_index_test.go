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

func TestSessionsForScopeFiltersAndAnnotatesIndexedSessions(t *testing.T) {
	pair := mustScope(t, "/Users/a/work/pair")
	other := mustScope(t, "/tmp/other/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{
		{SessionName: "pair-pair-work", ScopeKey: pair.Key, RepoRoot: pair.Root, RepoName: pair.DisplayName, Tag: "work"},
		{SessionName: "pair-pair-work-2", ScopeKey: other.Key, RepoRoot: other.Root, RepoName: other.DisplayName, Tag: "work"},
	}}
	sessions := []Session{
		{Name: "pair-pair-work", State: SessionDetached},
		{Name: "pair-pair-work-2", State: SessionDetached},
		{Name: "pair-legacy", State: SessionDetached},
	}

	got := SessionsForScope(sessions, index, pair)
	if len(got) != 1 {
		t.Fatalf("SessionsForScope returned %#v, want one current-scope session", got)
	}
	if got[0].Name != "pair-pair-work" || got[0].Tag != "work" || got[0].RepoName != "pair" {
		t.Fatalf("session = %#v, want annotated current-scope work", got[0])
	}
}

func TestSessionNameIndexRoundTripSkipsMalformedRows(t *testing.T) {
	entry := SessionNameEntry{
		SessionName: "pair-pair-work",
		ScopeKey:    "scope1",
		RepoRoot:    "/repo",
		RepoName:    "pair",
		Tag:         "work",
	}
	line, err := BuildSessionNameIndexLine(entry)
	if err != nil {
		t.Fatalf("BuildSessionNameIndexLine: %v", err)
	}
	for _, want := range []string{`"session_name"`, `"scope_key"`, `"repo_root"`, `"repo_name"`, `"tag"`} {
		if !strings.Contains(line, want) {
			t.Fatalf("index line = %s, want key %s", line, want)
		}
	}
	index := ParseSessionNameIndex(line + "\nnot-json\n")
	if len(index.Entries) != 1 {
		t.Fatalf("entries = %#v, want one valid entry", index.Entries)
	}
	if index.Entries[0] != entry {
		t.Fatalf("entry = %#v, want %#v", index.Entries[0], entry)
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
