package artifactpath

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestStorageOwnerNamespaces(t *testing.T) {
	root := t.TempDir()
	legacy, err := NewStorageOwner(root, "", "thread")
	if err != nil {
		t.Fatal(err)
	}
	scoped, err := NewStorageOwner(root, "repo", "thread")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Directory() != root || scoped.Directory() != filepath.Join(root, "repos", "repo") || legacy.Key() == scoped.Key() {
		t.Fatal("namespaces collided")
	}
	for _, args := range [][3]string{{"relative", "repo", "tag"}, {root, "../repo", "tag"}, {root, "repo", "../tag"}, {root, "", ""}} {
		if _, err := NewStorageOwner(args[0], args[1], args[2]); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestGCExactMatchesAndAmbiguity(t *testing.T) {
	root := t.TempDir()
	a, _ := NewStorageOwner(root, "repo", "a")
	ab, _ := NewStorageOwner(root, "repo", "a-b")
	legacy, _ := NewStorageOwner(root, "", "a")
	p, _ := ResolveScoped(a.Directory(), a.Tag)
	owners := []StorageOwner{a, ab, legacy}
	agents := []string{"codex", "b-codex"}
	for _, path := range []string{p.Draft(), p.QueueDir(), p.ScrollbackRaw("codex"), p.ScrollbackEvents("codex"), p.Config("codex")} {
		m, err := MatchArtifact(path, owners, agents)
		if err != nil || m.Owner != a {
			t.Fatalf("%s: %+v %v", path, m, err)
		}
	}
	for _, path := range []string{p.ScrollbackRaw("b-codex"), p.ScrollbackRaw("codex") + ".bak", filepath.Join(a.Directory(), "unknown-a"), p.Draft() + "/child"} {
		if _, err := MatchArtifact(path, owners, agents); err == nil {
			t.Fatalf("accepted ambiguous/unknown %s", path)
		}
	}
	parked, _ := p.ParkedScrollbackArtifacts("20260913-120000")
	m, err := MatchArtifact(parked.Events, owners, agents)
	if err != nil || m.Family != "parked-scrollback" {
		t.Fatalf("parked: %+v %v", m, err)
	}
	change, _ := p.ChangelogArtifacts("codex", "session-id")
	if _, err := MatchArtifact(change.Anchor, owners, agents); err != nil {
		t.Fatal(err)
	}
	group, err := InventoryArtifacts(a, owners, agents, []string{p.Draft(), filepath.Join(a.Directory(), "unknown-a")})
	if err != nil || len(group.Members) != 1 || len(group.Blockers) != 1 {
		t.Fatalf("group %+v %v", group, err)
	}
}

func TestGCManifestExhaustive(t *testing.T) {
	for _, f := range Families {
		rule, ok := GCClassifications[f.Name]
		if !ok || rule.Authority == "" {
			t.Errorf("missing GC authority for %s", f.Name)
		}
		switch rule.Disposition {
		case GCCollectable, GCShared, GCProtected:
		default:
			t.Errorf("missing disposition for %s", f.Name)
		}
	}
	if len(GCClassifications) != len(Families) {
		t.Fatal("unknown GC family")
	}
}

func TestDiscoverOwnersOnlyValidatedAnchors(t *testing.T) {
	owners, err := DiscoverStorageOwners(t.TempDir(), "repo", []string{"draft-a.md", "ledger-a.jsonl", "log-b.md", "scrollback-c-codex.raw", "draft-../oops.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(owners) != 2 || owners[0].Tag != "a" || owners[1].Tag != "b" {
		t.Fatalf("owners %+v", owners)
	}
}

func TestGCLifecycleQueueAndSharedEntries(t *testing.T) {
	o, _ := NewStorageOwner(t.TempDir(), "repo", "a")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	l, _ := p.Lifecycle("nonce")
	req, _ := l.Request(1)
	done, _ := l.Completion(2)
	trigger, _ := l.Trigger("session-with-hyphens", 3)
	for _, path := range []string{filepath.Dir(l.Dir()), l.Dir(), l.Lock(), req, done, trigger, filepath.Join(p.QueueDir(), "000001.md")} {
		if _, err := MatchArtifact(path, []StorageOwner{o}, nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{filepath.Join(l.Dir(), "quit-request-01.json"), filepath.Join(l.Dir(), "quit-completion-0.json"), filepath.Join(l.Dir(), "unknown"), filepath.Join(p.QueueDir(), "other.md"), filepath.Join(p.QueueDir(), "nested", "000001.md")} {
		if _, err := MatchArtifact(path, []StorageOwner{o}, nil); err == nil {
			t.Fatalf("accepted %s", path)
		}
	}
	s, _ := ResolveSelectedScope(o.Directory())
	def, _ := s.AgentDefault("codex")
	g, err := InventoryArtifacts(o, []StorageOwner{o}, []string{"codex"}, []string{p.Meta(), p.SessionBindings(), p.SessionInventoryCatalog(), def, p.Draft()})
	if err != nil || len(g.Blockers) != 0 || len(g.Members) != 1 {
		t.Fatalf("shared files block group: %+v %v", g, err)
	}
}

func TestGCDiagnosticsHaveIndependentRetention(t *testing.T) {
	o, _ := NewStorageOwner(t.TempDir(), "", "a")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	for _, c := range []struct {
		path string
		want RetentionClass
	}{{p.AdaptLog(), DebugRetention}, {p.Log(), SessionRetention}, {p.Ledger(), SessionRetention}, {p.Changelog("codex"), SessionRetention}, {p.WrapEvents(), DebugRetention}, {p.Draft(), SessionRetention}, {p.ScrollbackRaw("codex"), SessionRetention}} {
		m, err := MatchArtifact(c.path, []StorageOwner{o}, []string{"codex"})
		if err != nil || m.Retention != c.want {
			t.Fatalf("%s: %+v %v", c.path, m, err)
		}
	}
	owners, err := DiscoverStorageOwners(o.DataDir, "", []string{filepath.Base(p.WrapEvents())})
	if err != nil || len(owners) != 1 || owners[0] != o {
		t.Fatalf("diagnostic owner %+v %v", owners, err)
	}
}

func TestGCReviewEditorPID(t *testing.T) {
	o, _ := NewStorageOwner(t.TempDir(), "repo", "a")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	if _, err := MatchArtifact(p.NvimPID("review"), []StorageOwner{o}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGCLegacyInventoryExcludesScopedNamespace(t *testing.T) {
	root := t.TempDir()
	legacy, _ := NewStorageOwner(root, "", "a")
	scoped, _ := NewStorageOwner(root, "repo", "a")
	p, _ := ResolveScoped(scoped.Directory(), scoped.Tag)
	g, err := InventoryArtifacts(legacy, []StorageOwner{legacy, scoped}, nil, []string{p.Draft(), filepath.Join(scoped.Directory(), "unknown")})
	if err != nil || len(g.Members) != 0 || len(g.Blockers) != 0 {
		t.Fatalf("legacy sees scoped children %+v %v", g, err)
	}
}

func TestGCCollectableFamiliesHaveExactCandidates(t *testing.T) {
	o, _ := NewStorageOwner(t.TempDir(), "repo", "a")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	l, _ := p.Lifecycle("nonce")
	req, _ := l.Request(1)
	done, _ := l.Completion(1)
	trigger, _ := l.Trigger("session", 1)
	parked, _ := p.ParkedScrollbackArtifacts("123")
	seen := map[string]bool{}
	for _, input := range []string{p.Draft(), req, done, trigger, parked.Raw} {
		for _, candidate := range ownerCandidates(p, input, []string{"codex"}) {
			seen[candidate.Family] = true
			if _, err := MatchArtifact(candidate.Path, []StorageOwner{o}, []string{"codex"}); err != nil {
				t.Errorf("candidate %s: %v", candidate.Path, err)
			}
		}
	}
	for _, f := range Families {
		if GCClassifications[f.Name].Disposition == GCCollectable && !seen[f.Name] {
			t.Errorf("collectable family %s has no exact constructor candidate", f.Name)
		}
	}
}

func TestGCAgentDraftAnchorCannotAuthorizeWrongOwner(t *testing.T) {
	root := t.TempDir()
	o, _ := NewStorageOwner(root, "repo", "a")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	owners, err := DiscoverStorageOwners(root, "repo", []string{filepath.Base(p.Draft()), filepath.Base(p.AgentDraft("codex"))})
	if err != nil {
		t.Fatal(err)
	}
	if len(owners) != 2 {
		t.Fatalf("expected conservative ambiguous anchors: %+v", owners)
	}
	if _, err := MatchArtifact(p.AgentDraft("codex"), owners, []string{"codex"}); err == nil {
		t.Fatal("agent draft misattributed to inferred owner")
	}
	for _, owner := range owners {
		g, err := InventoryArtifacts(owner, owners, []string{"codex"}, []string{p.AgentDraft("codex")})
		if err != nil || len(g.Members) != 0 || len(g.Blockers) != 1 {
			t.Fatalf("ambiguous group %+v %v", g, err)
		}
	}
}

func TestGCLoneAgentDraftAnchorRetains(t *testing.T) {
	root := t.TempDir()
	o, _ := NewStorageOwner(root, "repo", "a")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	owners, err := DiscoverStorageOwners(root, "repo", []string{filepath.Base(p.AgentDraft("codex"))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MatchArtifact(p.AgentDraft("codex"), owners, []string{"codex"}); err == nil {
		t.Fatal("lone agent draft authorized as unrelated owner")
	}
}

func TestGCMatchIndexAgreesWithExactMatcher(t *testing.T) {
	root := t.TempDir()
	a, _ := NewStorageOwner(root, "repo", "a")
	ab, _ := NewStorageOwner(root, "repo", "a-b")
	owners := []StorageOwner{a, ab}
	agents := []string{"codex", "b-codex"}
	p, _ := ResolveScoped(a.Directory(), a.Tag)
	index, err := NewMatchIndex(owners, agents)
	if err != nil {
		t.Fatal(err)
	}
	l, _ := p.Lifecycle("nonce")
	req, _ := l.Request(1)
	parked, _ := p.ParkedScrollbackArtifacts("123")
	c, _ := p.ChangelogArtifacts("codex", "session-id")
	for _, path := range []string{p.Draft(), p.ScrollbackRaw("codex"), p.ScrollbackRaw("b-codex"), filepath.Join(p.QueueDir(), "123.md"), filepath.Join(p.QueueDir(), "unknown"), req, parked.Raw, parked.Events, c.Log, c.Anchor, p.Config("codex"), p.Draft() + ".bak"} {
		want, werr := MatchArtifact(path, owners, agents)
		got, gerr := index.Match(path)
		if (werr == nil) != (gerr == nil) || (werr == nil && got != want) {
			t.Fatalf("%s indexed %+v %v, exact %+v %v", path, got, gerr, want, werr)
		}
	}
	// A built index owns its input snapshot.
	owners[0].Tag = "mutated"
	agents[0] = "mutated"
	if _, err := index.Match(p.Draft()); err != nil {
		t.Fatal(err)
	}
}

func TestGCMatchIndexBoundsCandidatesFor100000Names(t *testing.T) {
	root := t.TempDir()
	owners := make([]StorageOwner, 1000)
	for i := range owners {
		owners[i], _ = NewStorageOwner(root, "repo", fmt.Sprintf("thread%04d", i))
	}
	index, err := NewMatchIndex(owners, []string{"codex"})
	if err != nil {
		t.Fatal(err)
	}
	checks := 0
	for i := 0; i < 100000; i++ {
		o := owners[i%len(owners)]
		p, _ := ResolveScoped(o.Directory(), o.Tag)
		c, _ := p.ChangelogArtifacts("codex", fmt.Sprintf("session-%d", i))
		candidates := index.dynamicOwners(c.Log)
		checks += len(candidates)
		if len(candidates) != 1 || candidates[0] != o {
			t.Fatalf("unbounded/wrong candidate set at %d: %+v", i, candidates)
		}
	}
	if checks != 100000 {
		t.Fatalf("candidate owner checks=%d", checks)
	}
}

func TestGCMatchIndexValidatesAndRetainsAmbiguousAnchors(t *testing.T) {
	root := t.TempDir()
	o, _ := NewStorageOwner(root, "repo", "a-codex")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	index, err := NewMatchIndex([]StorageOwner{o}, []string{"codex"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := index.Match(p.Draft()); err == nil {
		t.Fatal("ambiguous inferred owner was collectible")
	}
	if _, err := NewMatchIndex([]StorageOwner{{DataDir: root, RepoScope: "../escape", Tag: "a"}}, nil); err == nil {
		t.Fatal("invalid owner accepted")
	}
	if _, err := NewMatchIndex([]StorageOwner{o}, []string{"../escape"}); err == nil {
		t.Fatal("invalid agent accepted")
	}
	if _, err := index.Match("relative"); err == nil {
		t.Fatal("relative path accepted")
	}
}

func TestGCParkedCaptureHasIndependentClock(t *testing.T) {
	o, _ := NewStorageOwner(t.TempDir(), "repo", "a")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	set, _ := p.ParkedScrollbackArtifacts("20260913T120000-2")
	m, err := MatchArtifact(set.Raw, []StorageOwner{o}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Retention != CaptureRetention {
		t.Fatal("capture tied to owner activity")
	}
	capture, err := ParseParkedCapture(m, time.UTC)
	if err != nil || capture.Token != "20260913T120000-2" || !capture.CapturedAt.Equal(time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)) || capture.Raw != set.Raw || capture.Events != set.Events {
		t.Fatalf("capture %+v %v", capture, err)
	}
	for _, token := range []string{"20260931T120000", "20260913T120000-0", "20260913T120000-01", "123"} {
		bad, _ := p.ParkedScrollbackArtifacts(token)
		m.Path = bad.Raw
		if _, err := ParseParkedCapture(m, time.UTC); err == nil {
			t.Fatalf("invalid capture timestamp %s", token)
		}
	}
}

func TestGCDiscoversCaptureOnlyOwnerFromValidatedTimestamp(t *testing.T) {
	root := t.TempDir()
	o, _ := NewStorageOwner(root, "repo", "tag-with-20260101T010101")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	set, _ := p.ParkedScrollbackArtifacts("20260913T120000-2")
	for _, path := range []string{set.Raw, set.Events} {
		owners, err := DiscoverStorageOwners(root, "repo", []string{filepath.Base(path)})
		if err != nil || len(owners) != 1 || owners[0] != o {
			t.Fatalf("capture owner %v %v", owners, err)
		}
	}
	for _, stamp := range []string{"20260931T120000", "20260913T120000-01", "20260913T120000-0", "20260913T120000-junk"} {
		bad, _ := p.ParkedScrollbackArtifacts(stamp)
		owners, err := DiscoverStorageOwners(root, "repo", []string{filepath.Base(bad.Raw)})
		if err != nil || len(owners) != 0 {
			t.Fatalf("invalid anchor %v %v", owners, err)
		}
	}
}
