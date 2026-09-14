package artifactpath

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// StorageOwner names a single flat or repo-scoped namespace. Empty RepoScope
// explicitly selects legacy storage; it never means every repo.
// pair:m5-concept pure
type StorageOwner struct{ DataDir, RepoScope, Tag string }

func NewStorageOwner(root, scope, tag string) (StorageOwner, error) {
	o := StorageOwner{filepath.Clean(root), scope, tag}
	if scope == "" {
		if _, err := ResolveLegacyFlat(root, tag); err != nil {
			return StorageOwner{}, err
		}
	} else {
		if _, err := Resolve(Address{root, scope, tag}); err != nil {
			return StorageOwner{}, err
		}
	}
	return o, nil
}
func (o StorageOwner) Directory() string {
	if o.RepoScope == "" {
		return o.DataDir
	}
	return filepath.Join(o.DataDir, "repos", o.RepoScope)
}
func (o StorageOwner) Key() string {
	if o.RepoScope == "" {
		return filepath.Join("legacy", o.Tag)
	}
	return filepath.Join("repos", o.RepoScope, o.Tag)
}

// ArtifactMember is exact path authority, not a recursive deletion grant.
// Directory members require complete child inventory and lstat verification.
// pair:m5-concept pure
type ArtifactMember struct {
	Owner        StorageOwner
	Path, Family string
	Directory    bool
	Retention    RetentionClass
}

// pair:m5-concept pure
type ArtifactGroup struct {
	Owner    StorageOwner
	Members  []ArtifactMember
	Blockers []string
}

var captureAnchorPattern = regexp.MustCompile(`^parked-scrollback-(.+)-([0-9]{8}T[0-9]{6}(?:-[1-9][0-9]*)?)\.(raw|events\.jsonl|capture\.json)$`)

// DiscoverStorageOwners uses unambiguous history, diagnostic, and capture anchors.
// It does not infer tag/agent boundaries from agent-dependent filenames.
func DiscoverStorageOwners(root, scope string, names []string) ([]StorageOwner, error) {
	if _, err := NewStorageOwner(root, scope, "validation"); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []StorageOwner
	for _, name := range names {
		if filepath.Base(name) != name {
			continue
		}
		anchor := strings.TrimSuffix(name, ".pair-diagnostics.lock")
		tag, ok := TagFromHistorySidecar(name)
		if !ok {
			for _, f := range Families {
				if (f.Name == "wrap-events" || f.Name == "adapt") && strings.HasPrefix(anchor, f.Token) && strings.HasSuffix(anchor, ".jsonl") {
					tag = strings.TrimSuffix(strings.TrimPrefix(anchor, f.Token), ".jsonl")
					ok = true
				}
			}
		}
		if !ok {
			if parts := captureAnchorPattern.FindStringSubmatch(name); parts != nil {
				candidate, err := NewStorageOwner(root, scope, parts[1])
				if err == nil {
					member := ArtifactMember{Owner: candidate, Path: filepath.Join(candidate.Directory(), name), Family: "parked-scrollback"}
					if _, err := ParseParkedCapture(member, time.UTC); err == nil {
						tag, ok = candidate.Tag, true
					}
				}
			}
		}
		if !ok || seen[tag] {
			continue
		}
		o, err := NewStorageOwner(root, scope, tag)
		if err != nil {
			continue
		}
		seen[tag] = true
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out, nil
}

// MatchArtifact reconstructs every candidate using canonical constructors. Any
// competing owner interpretation blocks the path, including overlapping tags.
func MatchArtifact(path string, owners []StorageOwner, agents []string) (ArtifactMember, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ArtifactMember{}, fmt.Errorf("noncanonical artifact path %q", path)
	}
	for _, agent := range agents {
		if err := validateComponent("agent", agent); err != nil {
			return ArtifactMember{}, err
		}
	}
	var found *ArtifactMember
	for _, o := range owners {
		valid, err := NewStorageOwner(o.DataDir, o.RepoScope, o.Tag)
		if err != nil || valid != o {
			return ArtifactMember{}, fmt.Errorf("invalid storage owner %+v", o)
		}
		p, _ := ResolveScoped(o.Directory(), o.Tag)
		// History discovery cannot distinguish an agent draft from a base
		// draft whose tag ends in that agent. Absence of the parent anchor
		// does not resolve this ambiguity.
		if path == p.Draft() && ambiguousBaseDraft(o.Tag, agents) {
			return ArtifactMember{}, fmt.Errorf("ambiguous agent draft %q", path)
		}
		for _, m := range ownerCandidates(p, path, agents) {
			if m.Path != path {
				continue
			}
			m.Owner = o
			m.Retention = GCClassifications[m.Family].Retention
			if found != nil && found.Owner != o {
				return ArtifactMember{}, fmt.Errorf("ambiguous artifact %q", path)
			}
			copy := m
			found = &copy
		}
	}
	if found == nil {
		return ArtifactMember{}, fmt.Errorf("unrecognized artifact %q", path)
	}
	return *found, nil
}

func InventoryArtifacts(owner StorageOwner, owners []StorageOwner, agents []string, paths []string) (ArtifactGroup, error) {
	valid, err := NewStorageOwner(owner.DataDir, owner.RepoScope, owner.Tag)
	if err != nil || valid != owner {
		return ArtifactGroup{}, fmt.Errorf("invalid storage owner")
	}
	index, err := NewMatchIndex(owners, agents)
	if err != nil {
		return ArtifactGroup{}, err
	}
	group := ArtifactGroup{Owner: owner}
	seen := map[string]bool{}
	for _, path := range paths {
		relative, err := filepath.Rel(owner.Directory(), path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		if owner.RepoScope == "" && (relative == "repos" || strings.HasPrefix(relative, "repos"+string(filepath.Separator))) {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		if sharedArtifact(owner, path, agents) {
			continue
		}
		member, err := index.Match(path)
		if err != nil {
			group.Blockers = append(group.Blockers, err.Error())
			continue
		}
		if member.Owner == owner {
			group.Members = append(group.Members, member)
		}
	}
	sort.Slice(group.Members, func(i, j int) bool { return group.Members[i].Path < group.Members[j].Path })
	sort.Strings(group.Blockers)
	return group, nil
}

func sharedArtifact(o StorageOwner, path string, agents []string) bool {
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	s, _ := ResolveSelectedScope(o.Directory())
	for _, shared := range []string{p.Meta(), p.SessionBindings(), p.SessionInventoryCatalog(), p.SessionInventoryCatalog() + ".lock"} {
		if path == shared {
			return true
		}
	}
	for _, a := range agents {
		shared, err := s.AgentDefault(a)
		if err == nil && path == shared {
			return true
		}
	}
	return false
}

func ownerCandidates(p Paths, path string, agents []string) []ArtifactMember {
	var out []ArtifactMember
	add := func(family string, paths ...string) {
		for _, value := range paths {
			out = append(out, ArtifactMember{Path: value, Family: family})
		}
	}
	add("draft", p.Draft())
	add("ledger", p.Ledger())
	add("log", p.Log())
	out = append(out, ArtifactMember{Path: p.QueueDir(), Family: "queue", Directory: true})
	if filepath.Dir(path) == p.QueueDir() && queueMemberPattern.MatchString(filepath.Base(path)) {
		add("queue", path)
	}
	add("agent", p.Agent(), p.AgentOutput(), p.AgentPicks())
	add("agent-pid", p.AgentPID())
	add("outer-tty", p.OuterTTY())
	add("parked", p.Parked())
	add("adapt", p.AdaptLog())
	add("image-capture", p.ImageCapture(), p.ImageCaptureDone())
	add("continuation", p.Continuation())
	add("layout", p.WorkbenchLayout())
	add("layout-mode", p.LayoutMode())
	add("restart", p.Restart())
	add("picker", p.DraftPane())
	add("thread-claim", p.ThreadClaim())
	add("quote", p.Quote())
	add("slug", p.Slug(), p.SlugProposed())
	add("title-pid", p.TitlePID())
	add("pair-wrap-pid", p.PairWrapPID())
	add("wrap-events", p.WrapEvents())
	add("lifecycle", p.LifecycleJournal())
	add("scrollback-pending", p.ScrollbackPending())
	add("last-left-pane", p.LastLeftPane())
	add("last-terminal-pane", p.LastTerminalPane())
	add("terminal-panes", p.TerminalPanes())
	add("zellij-actions", p.ZellijActions())
	add("review", p.ReviewOpen(), p.ReviewMode(), p.ReviewTarget(), p.ReviewContext(), p.ReviewHandoff(), p.ReviewLanded(), p.ReviewDefinitionRequest(), p.ReviewDefinitionResult())
	add("codex-filter-kkp", p.CodexFilterKKP())
	add("config", p.LegacyCodexConfig())
	for _, kind := range []string{"draft", "scrollback", "review"} {
		add("nvim-pid", p.NvimPID(kind))
	}
	for _, a := range agents {
		add("config", p.Config(a))
		add("pane", p.Pane(a))
		add("agent-ready", p.AgentReady(a))
		add("draft", p.AgentDraft(a))
		scroll, _ := p.ScrollbackArtifacts(a)
		add("scrollback", scroll.Raw, scroll.Events, scroll.ANSI, scroll.Viewport, scroll.OpenLock)
		session := ""
		base, _ := p.ChangelogSessionChecked(a, "")
		for _, suffix := range []string{".md", ".anchor", ".cleaned", ".openlock", ".distill.lock", ".status"} {
			if strings.HasPrefix(path, base+"-") && strings.HasSuffix(path, suffix) {
				candidate := strings.TrimSuffix(strings.TrimPrefix(path, base+"-"), suffix)
				if validateComponent("session", candidate) == nil {
					session = candidate
					break
				}
			}
		}
		c, _ := p.ChangelogArtifacts(a, session)
		add("changelog", c.Log, c.Anchor, c.Cleaned, c.OpenLock, c.DistillLock, c.Status, c.Ready)
	}
	parkedBase, _ := p.ParkedScrollbackChecked("probe")
	prefix := strings.TrimSuffix(parkedBase, "probe")
	if strings.HasPrefix(path, prefix) {
		stamp := strings.TrimPrefix(path, prefix)
		for _, suffix := range []string{".events.jsonl", ".capture.json", ".raw"} {
			stamp = strings.TrimSuffix(stamp, suffix)
		}
		if set, err := p.ParkedScrollbackArtifacts(stamp); err == nil {
			add("parked-scrollback", set.Base, set.Raw, set.Events, set.Metadata)
		}
	}
	out = append(out, lifecycleCandidates(p, path)...)
	return out
}

func lifecycleCandidates(p Paths, path string) []ArtifactMember {
	probe, _ := p.Lifecycle("probe")
	root := filepath.Dir(probe.Dir())
	out := []ArtifactMember{{Path: root, Family: "lifecycle", Directory: true}}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return out
	}
	parts := strings.Split(relative, string(filepath.Separator))
	if len(parts) < 1 || len(parts) > 2 {
		return out
	}
	l, err := p.Lifecycle(parts[0])
	if err != nil {
		return out
	}
	out = append(out, ArtifactMember{Path: l.Dir(), Family: "lifecycle", Directory: true}, ArtifactMember{Path: l.Lock(), Family: "lifecycle-lock"})
	if len(parts) != 2 {
		return out
	}
	name := parts[1]
	for _, f := range Families {
		if f.Name != "lifecycle-request" && f.Name != "lifecycle-completion" && f.Name != "lifecycle-trigger" {
			continue
		}
		if !strings.HasPrefix(name, f.Token) || !strings.HasSuffix(name, ".json") {
			continue
		}
		value := strings.TrimSuffix(strings.TrimPrefix(name, f.Token), ".json")
		session := ""
		if f.Name == "lifecycle-trigger" {
			i := strings.LastIndexByte(value, '-')
			if i < 0 {
				continue
			}
			session, value = value[:i], value[i+1:]
		}
		attempt, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			continue
		}
		var expected string
		switch f.Name {
		case "lifecycle-request":
			expected, err = l.Request(attempt)
		case "lifecycle-completion":
			expected, err = l.Completion(attempt)
		case "lifecycle-trigger":
			expected, err = l.Trigger(session, attempt)
		}
		if err == nil {
			out = append(out, ArtifactMember{Path: expected, Family: f.Name})
		}
	}
	return out
}

var queueMemberPattern = regexp.MustCompile(`^[0-9]+\.md$`)

// MatchIndex amortizes static constructor work across an entire inventory.
// Dynamic prefix candidates are looked up by filename delimiters, so unrelated
// owners are never scanned on each path. It owns snapshots of caller inputs.
// pair:m5-concept pure
type MatchIndex struct {
	static    map[string][]ArtifactMember
	dynamic   map[string][]StorageOwner
	ambiguous map[string]bool
	agents    []string
}

func NewMatchIndex(owners []StorageOwner, agents []string) (*MatchIndex, error) {
	index := &MatchIndex{static: map[string][]ArtifactMember{}, dynamic: map[string][]StorageOwner{}, ambiguous: map[string]bool{}, agents: append([]string(nil), agents...)}
	for _, a := range agents {
		if err := validateComponent("agent", a); err != nil {
			return nil, err
		}
	}
	seen := map[StorageOwner]bool{}
	for _, o := range owners {
		valid, err := NewStorageOwner(o.DataDir, o.RepoScope, o.Tag)
		if err != nil || valid != o {
			return nil, fmt.Errorf("invalid storage owner %+v", o)
		}
		if seen[o] {
			continue
		}
		seen[o] = true
		p, _ := ResolveScoped(o.Directory(), o.Tag)
		for _, m := range ownerCandidates(p, "", agents) {
			m.Owner = o
			m.Retention = GCClassifications[m.Family].Retention
			index.static[m.Path] = append(index.static[m.Path], m)
		}
		if ambiguousBaseDraft(o.Tag, agents) {
			index.ambiguous[p.Draft()] = true
		}
		prefix := func(value string) { index.dynamic[value] = append(index.dynamic[value], o) }
		prefix(p.QueueDir() + string(filepath.Separator))
		l, _ := p.Lifecycle("probe")
		prefix(filepath.Dir(l.Dir()) + string(filepath.Separator))
		parked, _ := p.ParkedScrollbackChecked("probe")
		prefix(strings.TrimSuffix(parked, "probe"))
		for _, a := range agents {
			base, _ := p.ChangelogSessionChecked(a, "")
			prefix(base + "-")
		}
	}
	return index, nil
}

func (index *MatchIndex) dynamicOwners(path string) []StorageOwner {
	seen := map[StorageOwner]bool{}
	var out []StorageOwner
	for i := 0; i < len(path); i++ {
		if path[i] != '-' && path[i] != byte(filepath.Separator) {
			continue
		}
		for _, o := range index.dynamic[path[:i+1]] {
			if !seen[o] {
				seen[o] = true
				out = append(out, o)
			}
		}
	}
	return out
}

func (index *MatchIndex) Match(path string) (ArtifactMember, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ArtifactMember{}, fmt.Errorf("noncanonical artifact path %q", path)
	}
	if index.ambiguous[path] {
		return ArtifactMember{}, fmt.Errorf("ambiguous agent draft %q", path)
	}
	var found *ArtifactMember
	accept := func(m ArtifactMember) error {
		if found != nil && found.Owner != m.Owner {
			return fmt.Errorf("ambiguous artifact %q", path)
		}
		copy := m
		found = &copy
		return nil
	}
	for _, m := range index.static[path] {
		if err := accept(m); err != nil {
			return ArtifactMember{}, err
		}
	}
	for _, o := range index.dynamicOwners(path) {
		p, _ := ResolveScoped(o.Directory(), o.Tag)
		for _, m := range ownerCandidates(p, path, index.agents) {
			if m.Path != path {
				continue
			}
			m.Owner = o
			m.Retention = GCClassifications[m.Family].Retention
			if err := accept(m); err != nil {
				return ArtifactMember{}, err
			}
		}
	}
	if found == nil {
		return ArtifactMember{}, fmt.Errorf("unrecognized artifact %q", path)
	}
	return *found, nil
}

func ambiguousBaseDraft(tag string, agents []string) bool {
	for _, agent := range agents {
		if strings.HasSuffix(tag, "-"+agent) {
			parent := strings.TrimSuffix(tag, "-"+agent)
			if validateComponent("parent tag", parent) == nil {
				return true
			}
		}
	}
	return false
}

// ParkedCapture binds the raw/events pair to the producer's capture token.
// Legacy tokens contain local wall time; callers supply its known location.
// pair:m5-concept pure
type ParkedCapture struct {
	Token, Raw, Events, Metadata string
	CapturedAt                   time.Time
}

func ParseParkedCapture(member ArtifactMember, location *time.Location) (ParkedCapture, error) {
	if location == nil || member.Family != "parked-scrollback" {
		return ParkedCapture{}, fmt.Errorf("capture needs family and timezone")
	}
	o, err := NewStorageOwner(member.Owner.DataDir, member.Owner.RepoScope, member.Owner.Tag)
	if err != nil || o != member.Owner {
		return ParkedCapture{}, fmt.Errorf("invalid capture owner")
	}
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	probe, _ := p.ParkedScrollbackChecked("probe")
	prefix := strings.TrimSuffix(probe, "probe")
	if !strings.HasPrefix(member.Path, prefix) {
		return ParkedCapture{}, fmt.Errorf("capture outside owner")
	}
	token := strings.TrimPrefix(member.Path, prefix)
	token = strings.TrimSuffix(token, ".events.jsonl")
	token = strings.TrimSuffix(token, ".capture.json")
	token = strings.TrimSuffix(token, ".raw")
	stamp := token
	if i := strings.IndexByte(token, '-'); i >= 0 {
		stamp = token[:i]
		suffix := token[i+1:]
		n, err := strconv.ParseUint(suffix, 10, 64)
		if err != nil || n == 0 || strconv.FormatUint(n, 10) != suffix {
			return ParkedCapture{}, fmt.Errorf("invalid capture collision suffix")
		}
	}
	captured, err := time.ParseInLocation("20060102T150405", stamp, location)
	if err != nil || captured.Format("20060102T150405") != stamp {
		return ParkedCapture{}, fmt.Errorf("invalid capture timestamp")
	}
	set, err := p.ParkedScrollbackArtifacts(token)
	if err != nil || (member.Path != set.Base && member.Path != set.Raw && member.Path != set.Events && member.Path != set.Metadata) {
		return ParkedCapture{}, fmt.Errorf("noncanonical capture member")
	}
	return ParkedCapture{Token: token, Raw: set.Raw, Events: set.Events, Metadata: set.Metadata, CapturedAt: captured}, nil
}

// SharedStorageArtifact recognizes scope-owned metadata excluded from owner GC.
func SharedStorageArtifact(owner StorageOwner, path string, agents []string) bool {
	return sharedArtifact(owner, path, agents)
}

// DiagnosticLock recognizes the stable coordination inode for one exact debug
// artifact. Callers must additionally require an empty regular file; this is
// metadata exclusion authority, never permission to unlink the lock.
func (index *MatchIndex) DiagnosticLock(path string) bool {
	const suffix = ".pair-diagnostics.lock"
	if !strings.HasSuffix(path, suffix) {
		return false
	}
	member, err := index.Match(strings.TrimSuffix(path, suffix))
	return err == nil && member.Retention == DebugRetention && !member.Directory
}
