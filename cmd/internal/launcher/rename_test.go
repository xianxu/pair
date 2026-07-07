package launcher

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenamePathsForZip(t *testing.T) {
	// The zip contract: renamePathsFor(old) and renamePathsFor(new) enumerate in
	// identical order, so index i pairs src→dst.
	old := renamePathsFor("brain", "/d")
	nw := renamePathsFor("mind", "/d")
	if len(old) != len(nw) {
		t.Fatalf("enumerations differ in length: %d vs %d", len(old), len(nw))
	}
	// Spot-check a few known members transform correctly by index.
	find := func(paths []string, want string) int {
		for i, p := range paths {
			if p == want {
				return i
			}
		}
		return -1
	}
	i := find(old, "/d/draft-brain.md")
	if i < 0 || nw[i] != "/d/draft-mind.md" {
		t.Fatalf("draft zip: old[%d]=? new=%q", i, nw[i])
	}
	i = find(old, "/d/config-brain-claude.json")
	if i < 0 || nw[i] != "/d/config-mind-claude.json" {
		t.Fatalf("config zip: new=%q", nw[i])
	}
	i = find(old, "/d/scrollback-brain-codex.events.jsonl")
	if i < 0 || nw[i] != "/d/scrollback-mind-codex.events.jsonl" {
		t.Fatalf("scrollback events zip: new=%q", nw[i])
	}
	// Exact-name enumeration must NOT include a `-2` sibling family.
	if find(old, "/d/draft-brain-2.md") >= 0 {
		t.Fatal("enumeration must be exact-name, not glob-like")
	}
}

func TestValidateRenameTags(t *testing.T) {
	if _, _, err := validateRenameTags("pair-x", "x"); err == nil {
		t.Fatal("pair-x and x normalize to the same tag → must refuse old==new")
	}
	if _, _, err := validateRenameTags("a", strings.Repeat("z", 257)); err == nil {
		t.Fatal(">256 tag must be refused")
	}
	old, nw, err := validateRenameTags("pair-old", "new")
	if err != nil || old != "old" || nw != "new" {
		t.Fatalf("validate = (%q,%q,%v)", old, nw, err)
	}
}

func TestRenamePlan(t *testing.T) {
	exists := map[string]bool{
		"/d/draft-old.md":           true,
		"/d/config-old-claude.json": true,
	}
	ex := func(p string) bool { return exists[p] }

	pairs, err := renamePlan("old", "new", "/d", ex)
	if err != nil {
		t.Fatalf("plan err: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("plan = %+v, want 2 pairs (only existing srcs)", pairs)
	}
	// Occupancy: any new-tag sidecar existing → refuse.
	exists["/d/draft-new.md"] = true
	if _, err := renamePlan("old", "new", "/d", ex); err == nil {
		t.Fatal("occupied new tag must refuse")
	}
	// No source files → refuse.
	if _, err := renamePlan("ghost", "fresh", "/d", func(string) bool { return false }); err == nil {
		t.Fatal("no source files must refuse")
	}
}

func TestParseRename(t *testing.T) {
	got, err := ParseArgs([]string{"rename", "old", "new"})
	if err != nil || got.Command != "rename" || got.RenameOld != "old" || got.RenameNew != "new" || got.RenameCheckOnly {
		t.Fatalf("plain rename = %+v err=%v", got, err)
	}
	got, err = ParseArgs([]string{"rename", "--restart-check", "old", "new"})
	if err != nil || !got.RenameCheckOnly {
		t.Fatalf("--restart-check = %+v err=%v", got, err)
	}
	if got, _ := ParseArgs([]string{"rename", "--", "old", "new"}); got.RenameOld != "old" {
		t.Fatalf("-- separator = %+v", got)
	}
	if _, err := ParseArgs([]string{"rename", "only"}); err == nil {
		t.Fatal("missing new tag must error")
	}
	if _, err := ParseArgs([]string{"rename", "a", "b", "c"}); err == nil {
		t.Fatal("extra positional must error")
	}
}

func renameFake(t *testing.T) *fakeRuntime {
	t.Helper()
	rt := newFakeRuntime()
	rt.files["/data"] = "" // the data dir exists
	rt.files["/data/draft-old.md"] = "draft"
	rt.files["/data/config-old-claude.json"] = "cfg"
	return rt
}

func TestRunRenameHappyPath(t *testing.T) {
	rt := renameFake(t)
	var out, errBuf bytes.Buffer
	code := runRename(rt, LaunchArgs{RenameOld: "old", RenameNew: "new"}, "/data", &out, &errBuf)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errBuf.String())
	}
	if _, ok := rt.files["/data/draft-new.md"]; !ok {
		t.Fatalf("draft not moved; files=%v", rt.files)
	}
	if _, ok := rt.files["/data/draft-old.md"]; ok {
		t.Fatal("old draft should be gone after move")
	}
	if _, ok := rt.files["/data/config-new-claude.json"]; !ok {
		t.Fatal("config not moved")
	}
	// Journal written then removed on success.
	if _, ok := rt.files["/data/.rename-old-to-new.journal"]; ok {
		t.Fatal("journal should be removed on success")
	}
	if !strings.Contains(out.String(), "ok") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRunRenameRestartCheckDoesNotMove(t *testing.T) {
	rt := renameFake(t)
	var out, errBuf bytes.Buffer
	code := runRename(rt, LaunchArgs{RenameOld: "old", RenameNew: "new", RenameCheckOnly: true}, "/data", &out, &errBuf)
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if len(rt.renamed) != 0 {
		t.Fatalf("--restart-check must not move: renamed=%v", rt.renamed)
	}
	if _, ok := rt.files["/data/draft-old.md"]; !ok {
		t.Fatal("--restart-check must leave originals in place")
	}
	if !strings.Contains(out.String(), "would move") {
		t.Fatalf("stdout = %q", out.String())
	}
}

// --restart-check is the in-session rename gesture's pre-kill validation — it
// runs while pair-<old> is STILL live in zellij, so it must skip the live-old
// gate (the whole reason the flag exists) while still refusing on a live new tag.
func TestRunRenameRestartCheckSkipsLiveOld(t *testing.T) {
	rt := renameFake(t)
	rt.sessions = []Session{{Name: "pair-old", State: SessionDetached}} // old still tracked
	var out, errBuf bytes.Buffer
	code := runRename(rt, LaunchArgs{RenameOld: "old", RenameNew: "new", RenameCheckOnly: true}, "/data", &out, &errBuf)
	if code != 0 {
		t.Fatalf("--restart-check must skip the live-old gate; code=%d stderr=%s", code, errBuf.String())
	}
	if len(rt.renamed) != 0 {
		t.Fatal("--restart-check must not move")
	}
	if !strings.Contains(out.String(), "would move") {
		t.Fatalf("out=%q", out.String())
	}
	// But a live NEW tag still refuses even under --restart-check.
	rt2 := renameFake(t)
	rt2.sessions = []Session{{Name: "pair-new", State: SessionAttached}}
	out.Reset()
	errBuf.Reset()
	if code := runRename(rt2, LaunchArgs{RenameOld: "old", RenameNew: "new", RenameCheckOnly: true}, "/data", &out, &errBuf); code != 1 {
		t.Fatalf("live new tag must refuse even with --restart-check; code=%d", code)
	}
}

func TestRunRenameSessionGates(t *testing.T) {
	// old still tracked (detached) → refuse.
	rt := renameFake(t)
	rt.sessions = []Session{{Name: "pair-old", State: SessionDetached}}
	var out, errBuf bytes.Buffer
	if code := runRename(rt, LaunchArgs{RenameOld: "old", RenameNew: "new"}, "/data", &out, &errBuf); code != 1 {
		t.Fatalf("live old should refuse; code=%d", code)
	}
	if len(rt.renamed) != 0 {
		t.Fatal("must not move when old is live")
	}
	// new exists in ANY state (EXITED counts — resurrectable contract) → refuse.
	rt2 := renameFake(t)
	rt2.sessions = []Session{{Name: "pair-new", State: SessionExited}}
	out.Reset()
	errBuf.Reset()
	if code := runRename(rt2, LaunchArgs{RenameOld: "old", RenameNew: "new"}, "/data", &out, &errBuf); code != 1 {
		t.Fatalf("exited new should refuse; code=%d", code)
	}
}

func TestRunRenameSessionGatesUseScopedSessionNames(t *testing.T) {
	scope := mustScope(t, "/work/pair")

	rt := renameFake(t)
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "pair-pair-new",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "new",
	}}}
	rt.sessions = []Session{{Name: "pair-pair-new", State: SessionDetached}}

	var out, errBuf bytes.Buffer
	if code := runRename(rt, LaunchArgs{RenameOld: "old", RenameNew: "new"}, "/data", &out, &errBuf); code != 1 {
		t.Fatalf("scoped live new tag should refuse; code=%d stdout=%s stderr=%s", code, out.String(), errBuf.String())
	}
	if len(rt.renamed) != 0 {
		t.Fatalf("must not move when scoped new tag is live: %v", rt.renamed)
	}
}

func TestRunRenameRollback(t *testing.T) {
	rt := renameFake(t)
	// draft-old.md moves first (enumeration order), config-old-claude.json second.
	rt.renameFailAt = "/data/config-old-claude.json"
	var out, errBuf bytes.Buffer
	code := runRename(rt, LaunchArgs{RenameOld: "old", RenameNew: "new"}, "/data", &out, &errBuf)
	if code != 1 {
		t.Fatalf("mv failure should exit 1; code=%d", code)
	}
	// The completed first move was rolled back — draft-old.md is restored, and
	// no half-moved draft-new.md survives.
	if _, ok := rt.files["/data/draft-old.md"]; !ok {
		t.Fatalf("rollback should restore draft-old.md; files=%v", rt.files)
	}
	if _, ok := rt.files["/data/draft-new.md"]; ok {
		t.Fatal("rollback should undo the draft move")
	}
	// Journal kept on rollback (forensic).
	if _, ok := rt.files["/data/.rename-old-to-new.journal"]; !ok {
		t.Fatal("journal must survive a rollback for diagnosis")
	}
	if !strings.Contains(errBuf.String(), "rolling back") {
		t.Fatalf("stderr = %q", errBuf.String())
	}
}
