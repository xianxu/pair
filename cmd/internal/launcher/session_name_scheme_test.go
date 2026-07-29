package launcher

import (
	"strings"
	"testing"
)

// The 📁 session-name scheme (#130). These cover the pure core: the ownership
// predicate that keeps pair off foreign sessions, and the composer implementing
// spec rules 1-4.

func TestIsPairSessionName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"pair-x", true},             // legacy scheme
		{"pair-pair-pair", true},     // legacy, fully spelled
		{"📁x", true},                 // new scheme
		{"📁parley-nvim", true},       // new scheme with residual
		{"fabulous-aardvark", false}, // a real foreign session seen in the global list
		{"", false},
		{"repair-shop", false}, // must not match `pair-` mid-word
		{"folder📁", false},     // prefix only, not substring
	}
	for _, c := range cases {
		if got := isPairSessionName(c.name); got != c.want {
			t.Errorf("isPairSessionName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

// A foreign session must never be classified as pair's. This is the one
// predicate mistake that destroys someone else's state rather than confusing
// pair: SessionBlocksReuse force-deletes an EXITED session it believes is ours.
func TestIsPairSessionNameRejectsForeign(t *testing.T) {
	foreign := []string{
		"fabulous-aardvark",
		"nimble-narwhal",
		"main",
		"work",
		"scratch",
	}
	for _, name := range foreign {
		if isPairSessionName(name) {
			t.Errorf("isPairSessionName(%q) = true; a foreign session must never be a pair session", name)
		}
	}
}

func TestComposeSessionName(t *testing.T) {
	cases := []struct {
		what string
		repo string
		tag  string
		want string
	}{
		// The four worked examples from the spec.
		{"tag equals repo", "pair", "pair", "📁pair"},
		{"tag prefixed by repo", "pair", "pair-1", "📁pair-1"},
		{"dotted repo, underscored tag", "parley.nvim", "parley_nvim", "📁parley-nvim"},
		{"tag unrelated to repo", "parley.nvim", "work", "📁parley-work"},

		// Where the rules interact.
		{"repo token appears non-leading in tag", "pair", "my-pair", "📁pair-my-pair"},
		{"case differs, so no redundancy is dropped", "Pair", "pair", "📁Pair-pair"},
		{"empty tag falls back to the normalizer's default", "pair", "", "📁pair"},
		{"repo with no alphanumerics", "...", "work", "📁pair-work"},
		{"multi-token residual is fully preserved", "pair", "pair-feature-two", "📁pair-feature-two"},
		{"hyphenated repo keeps only its first token", "claude-code", "work", "📁claude-work"},
	}
	for _, c := range cases {
		scope := RepoScope{DisplayName: c.repo}
		if got := ComposeSessionName(scope, c.tag); got != c.want {
			t.Errorf("%s: ComposeSessionName(%q, %q) = %q, want %q", c.what, c.repo, c.tag, got, c.want)
		}
	}
}

// The headline outcomes the issue exists to produce, asserted directly so a
// regression names itself.
func TestComposeSessionNameRetiresTheRedundantSpellings(t *testing.T) {
	if got := ComposeSessionName(RepoScope{DisplayName: "pair"}, "pair"); got != "📁pair" {
		t.Errorf("pair-pair-pair should become 📁pair, got %q", got)
	}
	got := ComposeSessionName(RepoScope{DisplayName: "parley.nvim"}, "parley_nvim")
	if got != "📁parley-nvim" {
		t.Errorf("pair-parley_nvim-parley_nvim should become 📁parley-nvim untruncated, got %q", got)
	}
	// And it must fit: the whole point is that this no longer needs shortening.
	if len(got) > 24 {
		t.Errorf("%q is %d bytes; zellij's budget on this machine is 24", got, len(got))
	}
}

func TestSessionNameParts(t *testing.T) {
	repo, residual := sessionNameParts(RepoScope{DisplayName: "parley.nvim"}, "parley_nvim")
	if repo != "parley" {
		t.Errorf("repo = %q, want %q", repo, "parley")
	}
	if len(residual) != 1 || residual[0] != "nvim" {
		t.Errorf("residual = %v, want [nvim]", residual)
	}

	// Rule 4 drops only the LEADING matching run.
	_, residual = sessionNameParts(RepoScope{DisplayName: "pair"}, "pair-pair-x")
	if len(residual) != 2 || residual[0] != "pair" || residual[1] != "x" {
		t.Errorf("residual = %v, want [pair x] — only the leading run is dropped", residual)
	}
}

// The destructive path (#130). SessionBlocksReuse force-deletes a session it
// believes is a stale pair record, so a widened ownership predicate does not
// merely confuse pair — it destroys someone else's state. `fabulous-aardvark`
// below is a real session that was live in the global list while this was
// scoped.
func TestForeignSessionIsNeverADeletionCandidate(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessions = []Session{
		{Name: "fabulous-aardvark", State: SessionExited}, // a stranger's abandoned session
		{Name: "nimble-narwhal", State: SessionDetached},
	}
	rt.promptValue = "work"

	if _, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, deleted := range rt.deleted {
		if !isPairSessionName(deleted) {
			t.Fatalf("deleted a foreign session %q; deletions were %v", deleted, rt.deleted)
		}
	}
	if rt.launched != "" && !isPairSessionName(rt.launched) {
		t.Fatalf("launched a name outside pair's namespace: %q", rt.launched)
	}
}

// Migration must not reclaim a session someone can still get back to. Only an
// already-EXITED record is dead weight.
func TestSessionReclaimableOnlyForExitedRecords(t *testing.T) {
	live := []Session{
		{Name: "pair-work-attached", State: SessionAttached},
		{Name: "pair-work-detached", State: SessionDetached},
		{Name: "pair-work-exited", State: SessionExited},
	}
	cases := map[string]bool{
		"pair-work-attached": false, // someone is sitting in it
		"pair-work-detached": false, // resumable work; the picker still reaches it
		"pair-work-exited":   true,  // dead weight, and nothing will ever name it again
		"pair-work-absent":   true,  // zellij has no such session; delete is a no-op
	}
	for name, want := range cases {
		if got := sessionReclaimable(live, name); got != want {
			t.Errorf("sessionReclaimable(%q) = %v, want %v", name, got, want)
		}
	}
}

// Done-when's highest-risk clause: a legacy session and a new-format one must
// coexist in one snapshot, both discoverable with the right tag. Orphaning a
// live session costs the user real state, so this is pinned by a test rather
// than resting on the manual live check.
func TestMixedSchemeSnapshotStaysDiscoverable(t *testing.T) {
	scope := mustScope(t, "/Users/a/work/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{
		{SessionName: "pair-pair-legacy", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: "legacy"},
		{SessionName: "📁pair-fresh", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: "fresh"},
	}}
	sessions := []Session{
		{Name: "pair-pair-legacy", State: SessionDetached},
		{Name: "📁pair-fresh", State: SessionDetached},
		{Name: "fabulous-aardvark", State: SessionDetached},
	}

	scoped := SessionsForScope(sessions, index, scope)
	tags := map[string]string{}
	for _, s := range scoped {
		tags[s.Name] = s.Tag
	}
	if tags["pair-pair-legacy"] != "legacy" {
		t.Errorf("legacy session lost its tag: %v", tags)
	}
	if tags["📁pair-fresh"] != "fresh" {
		t.Errorf("new-format session lost its tag: %v", tags)
	}
	if _, ok := tags["fabulous-aardvark"]; ok {
		t.Errorf("a foreign session entered this repo's scope: %v", tags)
	}

	rows := buildListRowsForScope(
		[]string{"pair-pair-legacy", "📁pair-fresh"},
		"", index, scope.Key,
		func(tag string) string { return "claude" },
		func(string) int { return 0 },
	)
	if len(rows) != 2 {
		t.Fatalf("pair list dropped a row: %+v", rows)
	}

	// The picker must keep both rows distinct and selectable.
	for _, s := range scoped {
		if sessionTag(s) == "" {
			t.Errorf("picker would resolve %q to no tag", s.Name)
		}
	}
}

// `pair list` columns are terminal columns, not runes: 📁 is one rune and two
// columns, so a rune-padded table puts every new-format row a column off.
func TestListTableAlignsMixedSchemeRows(t *testing.T) {
	out := formatListTable([]ListRow{
		{Session: "pair-pair-legacy", Agent: "claude", State: SessionDetached},
		{Session: "📁pair-fresh", Agent: "claude", State: SessionDetached},
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want header + 2 rows, got %d lines:\n%s", len(lines), out)
	}
	// Measure COLUMNS, not bytes: strings.Index returns a byte offset, and 📁 is
	// four bytes wide but two columns — the very confusion this padding fixes.
	col := func(line string) int { return displayWidth(line[:strings.Index(line, "claude")]) }
	if col(lines[1]) != col(lines[2]) {
		t.Errorf("AGENT column misaligned between schemes (%d vs %d):\n%s",
			col(lines[1]), col(lines[2]), out)
	}
}

// Refuse-early (#130). The decision is pure and takes the limit as a parameter,
// so it is exercisable at a Linux-sized budget as well as the macOS one this was
// measured on — the whole reason the budget is not a package constant.
func TestSessionNameFits(t *testing.T) {
	if ok, _ := sessionNameFits("📁pair", 24); !ok {
		t.Error("📁pair is 8 bytes; it must fit a 24-byte budget")
	}
	// 📁 is 4 bytes, so this is 4+21 = 25.
	long := "📁" + strings.Repeat("z", 21)
	ok, msg := sessionNameFits(long, 24)
	if ok {
		t.Fatalf("%d-byte name must not fit a 24-byte budget", len(long))
	}
	for _, want := range []string{"25", "24"} {
		if !strings.Contains(msg, want) {
			t.Errorf("refusal must quote the real numbers, got %q (missing %q)", msg, want)
		}
	}
	// A larger budget (Linux's shorter socket path) accepts the same name.
	if ok, _ := sessionNameFits(long, 40); !ok {
		t.Error("the limit is a parameter; a 40-byte budget must accept a 25-byte name")
	}
}

func TestDiscoverSessionNameBudget(t *testing.T) {
	// A fake zellij that accepts names up to 30 bytes.
	probed := []string{}
	accepts := func(name string) bool {
		probed = append(probed, name)
		return len(name) <= 30
	}
	if got := discoverSessionNameBudget(accepts); got != 30 {
		t.Errorf("budget = %d, want 30", got)
	}
	// Every probe must be a synthetic pad, never a name that could belong to a
	// real session: list-clients SUCCEEDS against a foreign live session, which
	// would read as "fits" for the wrong reason.
	for _, name := range probed {
		if !strings.HasPrefix(name, sessionNameProbeMarker) {
			t.Errorf("probe %q is not a synthetic pad", name)
		}
	}
	// Falls back rather than looping when even the shortest probe is rejected.
	if got := discoverSessionNameBudget(func(string) bool { return false }); got != defaultSessionNameBudget {
		t.Errorf("unusable probe → %d, want the %d default", got, defaultSessionNameBudget)
	}
}
