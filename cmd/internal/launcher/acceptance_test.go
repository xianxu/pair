package launcher

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAcceptanceSameTagIsolationAcrossRepos(t *testing.T) {
	global := "/home/u/.local/share/pair"
	first := mustScope(t, "/home/u/work/pair")
	second := mustScope(t, "/tmp/other/pair")

	index := SessionNameIndex{}
	firstName, index, err := AssignSessionName(index, nil, first, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("first AssignSessionName: %v", err)
	}
	secondName, index, err := AssignSessionName(index, []Session{{Name: firstName, State: SessionDetached}}, second, "work", acceptAllSessionNames)
	if err != nil {
		t.Fatalf("second AssignSessionName: %v", err)
	}
	if firstName != "pair-pair-work" || secondName != "pair-pair-work-2" {
		t.Fatalf("session names = %q, %q; want readable disambiguated names", firstName, secondName)
	}
	if strings.Contains(firstName, first.Key) || strings.Contains(secondName, second.Key) {
		t.Fatalf("session names must not expose hidden scope keys: %q %q", firstName, secondName)
	}

	firstPaths := NewScopedPaths(global, first, "work")
	secondPaths := NewScopedPaths(global, second, "work")
	if firstPaths.Draft() == secondPaths.Draft() {
		t.Fatalf("same display tag in different repos must not share draft path: %q", firstPaths.Draft())
	}
	if filepath.Base(firstPaths.Draft()) != "draft-work.md" || filepath.Base(secondPaths.Draft()) != "draft-work.md" {
		t.Fatalf("scoped paths should keep repo-local display tag names: %q %q", firstPaths.Draft(), secondPaths.Draft())
	}

	live := []Session{
		{Name: firstName, State: SessionDetached},
		{Name: secondName, State: SessionDetached},
	}
	firstLive := SessionsForScope(live, index, first)
	secondLive := SessionsForScope(live, index, second)
	if len(firstLive) != 1 || firstLive[0].Name != firstName || firstLive[0].Tag != "work" {
		t.Fatalf("first scope live sessions = %#v", firstLive)
	}
	if len(secondLive) != 1 || secondLive[0].Name != secondName || secondLive[0].Tag != "work" {
		t.Fatalf("second scope live sessions = %#v", secondLive)
	}
}
