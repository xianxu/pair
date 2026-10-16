package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// A sidecar names a pid, and a pid is not a process. Every case below makes one
// predicate of the reap decision false and asserts that nothing is signalled;
// the fully-evidenced row is killed exactly once. The "never by name alone"
// half of #583 is structural: the core has no subprocess seam, so the fake
// table's signal list is the entire observable surface and no test could see a
// pkill even if one came back.

const (
	reapNvimExe  = "/opt/homebrew/bin/nvim"
	reapScopeDir = "/data/repos/scope"
	reapDraft    = reapScopeDir + "/draft-work.md"
	reapSidecar  = reapScopeDir + "/nvim-pid-work-draft"
	reapStamp    = "1700000000.1"
	// The review kind's owner-naming evidence: the file a review embed serves is
	// anywhere in the user's tree, so only the tag's own target state names it
	// (#583 BR-6).
	reapReviewState = reapScopeDir + "/review-target-work.json"
	reapReviewFile  = "/src/post.md"
)

// reviewStateJSON is the review-target document as it sits on disk, spelled out
// here rather than built by the writer so the reap is pinned to the contract the
// launcher reads.
func reviewStateJSON(file string) string {
	return `{"file":"` + file + `","status":"ready","session":"sid1"}`
}

func draftArgv(path string) []string {
	return []string{"nvim", "--embed", "-u", "/pair/nvim/init.lua", path}
}

// reviewArgv is a review embed's argv: the config its launcher passes to `-u`,
// and the arbitrary file it serves.
func reviewArgv(path string) []string {
	return []string{"nvim", "--embed", "-u", "/anywhere/nvim/review.lua", path}
}

// reapNvimTable is one embedded nvim at pid, reparented to launchd, with argv
// readable -- the row a dead owner leaves behind.
func reapNvimTable(pid int, args []string) *fakeProcessTable {
	table := newFakeProcessTable()
	row := table.add(pid, 1, reapStamp)
	row.name, row.args, row.argsOK = reapNvimExe, args, true
	return table
}

func scrollbackTarget(scopeDir string) nvimSidecar {
	return nvimSidecar{
		Path: reapScopeDir + "/nvim-pid-work-scrollback", Kind: "scrollback", Owner: "work",
		ExpectArgvTag: true, ArgvScopeDir: scopeDir,
	}
}

type nvimSidecarReapCase struct {
	name string
	// target defaults to the tag-keyed draft record.
	target  nvimSidecar
	sidecar string
	// reviewState, when set, is written to the target's review state path: the
	// review kind's owner-naming attribution is read from there.
	reviewState string
	// table defaults to reapNvimTable(4242, draftArgv(reapDraft)).
	table      *fakeProcessTable
	noProcess  bool
	row        func(*fakeProcess)
	wantKill   bool
	wantReason string
	wantKept   bool
}

func (c nvimSidecarReapCase) run(t *testing.T) nvimReapOutcome {
	t.Helper()
	target := c.target
	if target.Path == "" {
		target = nvimSidecar{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectArgvPath: true, ExpectedArgPath: reapDraft}
	}
	sidecars := newFakeRuntime()
	sidecars.files[target.Path] = c.sidecar
	if c.reviewState != "" {
		sidecars.files[target.ReviewState] = c.reviewState
	}
	table := c.table
	switch {
	case table != nil:
	case c.noProcess:
		table = newFakeProcessTable()
	default:
		table = reapNvimTable(4242, draftArgv(reapDraft))
	}
	if c.row != nil {
		for _, row := range table.procs {
			c.row(row)
		}
	}
	outcomes, err := reapNvimSidecars(context.Background(), table, sidecars, []nvimSidecar{target})
	if err != nil {
		t.Fatalf("reap: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("outcomes = %+v, want exactly one", outcomes)
	}
	outcome := outcomes[0]
	if outcome.Killed != c.wantKill || outcome.Reason != c.wantReason {
		t.Fatalf("killed=%v reason=%q, want killed=%v reason=%q", outcome.Killed, outcome.Reason, c.wantKill, c.wantReason)
	}
	var wantSignals []string
	if c.wantKill {
		wantSignals = []string{signalName(outcome.PID, syscall.SIGKILL)}
	}
	if !slices.Equal(table.signals, wantSignals) {
		t.Fatalf("signals = %v, want %v", table.signals, wantSignals)
	}
	if _, kept := sidecars.files[target.Path]; kept != c.wantKept {
		t.Fatalf("record still on disk = %v, want %v (removed %v)", kept, c.wantKept, sidecars.removed)
	}
	if outcome.Removed == c.wantKept {
		t.Fatalf("outcome.Removed = %v contradicts the record being kept=%v", outcome.Removed, c.wantKept)
	}
	return outcome
}

func TestReapNvimSidecarsKillsOnlyWhatTheEvidenceNames(t *testing.T) {
	t.Parallel()
	review := nvimSidecar{
		Path: reapScopeDir + "/nvim-pid-work-review", Kind: "review", Owner: "work",
		ExpectArgvPath: true, ReviewState: reapReviewState, ExpectedAssetBase: "review.lua",
	}
	liveOwner := nvimSidecar{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectArgvPath: true, ExpectedArgPath: reapDraft, OwnerLive: true}
	for _, tc := range []nvimSidecarReapCase{
		{
			name: "a fully evidenced embed whose owner is dead", sidecar: "4242\n",
			wantKill: true, wantReason: reapReasonOwnerDead,
		},
		{
			name: "the pid was recycled between the snapshot and the signal", sidecar: "4242\n",
			row:        func(p *fakeProcess) { p.identityAtSignal = "1700000009.9" },
			wantReason: reapReasonRecycled,
		},
		{
			name: "the pid now belongs to something that is not nvim", sidecar: "4242\n",
			row:        func(p *fakeProcess) { p.name = "/opt/homebrew/bin/node" },
			wantReason: reapReasonNotNvim,
		},
		{
			name: "the pid now belongs to a terminal nvim", sidecar: "4242\n",
			table:      reapNvimTable(4242, []string{"nvim", "-u", "/pair/nvim/init.lua", reapDraft}),
			wantReason: reapReasonNoEmbed,
		},
		{
			name: "the embed serves another tag's draft", sidecar: "4242\n",
			table:      reapNvimTable(4242, draftArgv(reapScopeDir+"/draft-other.md")),
			wantReason: reapReasonOtherPath,
		},
		{
			name: "the draft path appears only inside a longer field", sidecar: "4242\n",
			table:      reapNvimTable(4242, draftArgv(reapDraft+".bak")),
			wantReason: reapReasonOtherPath,
		},
		{
			name: "argv could not be read", sidecar: "4242\n",
			row:        func(p *fakeProcess) { p.argsOK = false },
			wantReason: reapReasonArgvUnknown, wantKept: true,
		},
		{
			name: "the kernel start stamp could not be read", sidecar: "4242\n",
			row:        func(p *fakeProcess) { p.identity = "" },
			wantReason: reapReasonIdentityUnknown, wantKept: true,
		},
		{
			name: "the owner is still alive", sidecar: "4242\n",
			table: func() *fakeProcessTable {
				table := reapNvimTable(4242, draftArgv(reapDraft))
				table.procs[4242].ppid = 4000
				return table
			}(),
			target: liveOwner, wantReason: reapReasonOwnerLive, wantKept: true,
		},
		{
			name: "a live owner's embed that reparented to launchd", sidecar: "4242\n",
			target: liveOwner, wantKill: true, wantReason: reapReasonReparented,
		},
		{
			name: "the record does not name a pid", sidecar: "not-a-pid\n",
			table:      reapNvimTable(4242, draftArgv(reapDraft)),
			wantReason: reapReasonMalformed,
		},
		{
			name: "no process holds the pid", sidecar: "4242\n",
			noProcess: true, wantReason: reapReasonGone,
		},
		{
			name: "a scrollback embed attributed to another tag", sidecar: "4242\n",
			target:     scrollbackTarget(reapScopeDir),
			table:      reapNvimTable(4242, draftArgv(reapScopeDir+"/scrollback-work-2-claude.ansi")),
			wantReason: reapReasonOtherTag,
		},
		{
			name: "a scrollback embed attributed to its owner", sidecar: "4242\n",
			target:     scrollbackTarget(reapScopeDir),
			table:      reapNvimTable(4242, draftArgv(reapScopeDir+"/scrollback-work-claude.ansi")),
			wantKill:   true,
			wantReason: reapReasonOwnerDead,
		},
		{
			name: "a review embed serving the file its own tag's state names", sidecar: "4242\n",
			target: review, reviewState: reviewStateJSON(reapReviewFile),
			table:      reapNvimTable(4242, reviewArgv(reapReviewFile)),
			wantKill:   true,
			wantReason: reapReasonOwnerDead,
		},
		{
			// #583 BR-6: this record outlived its embed, and the pid it names now
			// holds another tag's live review pane. Both argv carry review.lua, so
			// the config's base name says "a pair review pane" and not whose; only
			// the file this tag's own state names decides, and that pane serves
			// another file.
			name: "a dead tag's review record whose pid another tag's live pane holds", sidecar: "4242\n",
			target: review, reviewState: reviewStateJSON(reapReviewFile),
			table:      reapNvimTable(4242, reviewArgv("/src/theirs.md")),
			row:        func(p *fakeProcess) { p.ppid = 4000 },
			wantReason: reapReasonOtherPath,
		},
		{
			name: "a review record whose tag state names no file", sidecar: "4242\n",
			target: review, reviewState: `{"status":"ready"}`,
			table:      reapNvimTable(4242, reviewArgv(reapReviewFile)),
			wantReason: reapReasonNoAttribution,
			wantKept:   true,
		},
		{
			name:       "a review record whose tag state is gone",
			sidecar:    "4242\n",
			target:     review,
			table:      reapNvimTable(4242, reviewArgv(reapReviewFile)),
			wantReason: reapReasonNoAttribution,
			wantKept:   true,
		},
		{
			name: "a bare embed sitting at a review record's pid", sidecar: "4242\n",
			target: review, reviewState: reviewStateJSON(reapReviewFile),
			table:      reapNvimTable(4242, []string{"nvim", "--embed"}),
			wantReason: reapReasonOtherPath,
		},
		{
			name: "a review embed running another config", sidecar: "4242\n",
			target: review, reviewState: reviewStateJSON(reapReviewFile),
			table:      reapNvimTable(4242, []string{"nvim", "--embed", "-u", "/anywhere/nvim/init.lua", reapReviewFile}),
			wantReason: reapReasonOtherAsset,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			outcome := tc.run(t)
			if outcome.PID != 4242 && tc.wantReason != reapReasonMalformed {
				t.Fatalf("outcome pid = %d, want the recorded 4242", outcome.PID)
			}
		})
	}
}

// A missing record is not an outcome: there is nothing to decide and nothing to
// remove, so the sweep that reports reaped orphans stays quiet about it.
func TestReapNvimSidecarsIgnoresAnAbsentRecord(t *testing.T) {
	t.Parallel()
	table := reapNvimTable(4242, draftArgv(reapDraft))
	outcomes, err := reapNvimSidecars(context.Background(), table, newFakeRuntime(), []nvimSidecar{
		{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectArgvPath: true, ExpectedArgPath: reapDraft},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 0 || len(table.signals) != 0 {
		t.Fatalf("outcomes %+v signals %v, want none", outcomes, table.signals)
	}
}

// A cancelled context stops before the snapshot: the reap is part of quit, and
// quit reports the cancellation instead of half-reaping.
func TestReapNvimSidecarsRefusesACancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	table := reapNvimTable(4242, draftArgv(reapDraft))
	sidecars := newFakeRuntime()
	sidecars.files[reapSidecar] = "4242\n"
	outcomes, err := reapNvimSidecars(ctx, table, sidecars, []nvimSidecar{
		{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectArgvPath: true, ExpectedArgPath: reapDraft},
	})
	if err == nil {
		t.Fatal("reaped with a cancelled context")
	}
	if len(outcomes) != 0 || len(table.signals) != 0 || len(sidecars.removed) != 0 {
		t.Fatalf("outcomes %+v signals %v removed %v, want none", outcomes, table.signals, sidecars.removed)
	}
}

// A refused signal is no kill. The outcome is what the orphan report reads, so
// a pid that raced away between the fence and the signal must not be reported
// as reaped; its record still goes, because the decision was made.
func TestReapNvimSidecarsReportsARefusedSignalAsNotKilled(t *testing.T) {
	t.Parallel()
	table := reapNvimTable(4242, draftArgv(reapDraft))
	table.signalErr = syscall.EPERM
	sidecars := newFakeRuntime()
	sidecars.files[reapSidecar] = "4242\n"
	outcomes, err := reapNvimSidecars(context.Background(), table, sidecars, []nvimSidecar{
		{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectArgvPath: true, ExpectedArgPath: reapDraft},
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcomes[0].Killed || len(table.signals) != 0 {
		t.Fatalf("killed=%v signals=%v, want a refused signal to be no kill", outcomes[0].Killed, table.signals)
	}
	if outcomes[0].Reason != reapReasonOwnerDead {
		t.Fatalf("reason = %q, want the decision that was made", outcomes[0].Reason)
	}
	if _, kept := sidecars.files[reapSidecar]; !outcomes[0].Removed || kept {
		t.Fatal("a decided record should be gone even when the host refused the signal")
	}
}

// Every kind in the closed set must carry attribution that names its owner. A
// kind added to artifactpath.NvimKinds() without one would be reaped on evidence
// every tag's embed shares — `nvim --embed`, or the config's base name — which is
// the name-only kill #583 forbids.
func TestNvimSidecarsGiveEveryKindArgvEvidence(t *testing.T) {
	t.Parallel()
	paths, err := artifactpath.ResolveScoped(t.TempDir(), "work")
	if err != nil {
		t.Fatal(err)
	}
	targets := nvimSidecarsFor(paths, "work", paths.ScopeDir(), false)
	if len(targets) != len(artifactpath.NvimKinds()) {
		t.Fatalf("targets = %+v, want one per kind %v", targets, artifactpath.NvimKinds())
	}
	for _, target := range targets {
		if target.Owner != "work" || target.Path == "" || target.Kind == "" || target.OwnerLive {
			t.Fatalf("target %+v: want the owner's record, no liveness claim", target)
		}
		if !target.ExpectArgvPath && !target.ExpectArgvTag {
			t.Fatalf("kind %q carries no owner-naming attribution: %+v", target.Kind, target)
		}
		if target.Path != paths.NvimPID(target.Kind) {
			t.Fatalf("kind %q reads %q, want %q", target.Kind, target.Path, paths.NvimPID(target.Kind))
		}
	}
	// Each kind names its owner differently. The draft's is the exact artifact
	// path. The review pane serves a file anywhere in the user's tree, so its
	// owner-naming field comes from the tag's own review state, and the asset its
	// launcher passes to `-u` is only corroboration that argv is a pair pane: it
	// is matched by base because pair's install root differs per checkout (a live
	// embed outlives the checkout that spawned it, and its argv keeps that
	// checkout's path).
	for _, target := range targets {
		switch target.Kind {
		case "draft":
			if target.ExpectedArgPath != paths.Draft() {
				t.Fatalf("draft expects %q, want %q", target.ExpectedArgPath, paths.Draft())
			}
		case "review":
			if !target.ExpectArgvPath || target.ReviewState != paths.ReviewTarget() {
				t.Fatalf("review names no owner-naming state: %+v", target)
			}
			if want := filepath.Base(reviewLuaAsset); target.ExpectedAssetBase != want {
				t.Fatalf("review expects asset %q, want %q", target.ExpectedAssetBase, want)
			}
		}
	}
	// Liveness is the caller's evidence, so what the sweep observed must reach
	// every kind: a kind that dropped it would be reaped on its owner's death
	// alone, with no reparenting to justify the kill.
	for _, target := range nvimSidecarsFor(paths, "work", paths.ScopeDir(), true) {
		if !target.OwnerLive {
			t.Fatalf("kind %q lost the owner liveness it was given: %+v", target.Kind, target)
		}
	}
}

// One record per kind of the closed set, in its order: the derivation both the
// quit path and the orphan reaper build through, so a kind cannot be reaped by
// one and missed by the other.
func TestNvimSidecarsForDerivesFromTheClosedKindSet(t *testing.T) {
	t.Parallel()
	paths, err := artifactpath.ResolveScoped(t.TempDir(), "work")
	if err != nil {
		t.Fatal(err)
	}
	targets := nvimSidecarsFor(paths, "work", paths.ScopeDir(), false)
	var kinds, records []string
	for _, target := range targets {
		kinds = append(kinds, target.Kind)
		records = append(records, target.Path)
	}
	if !slices.Equal(kinds, artifactpath.NvimKinds()) {
		t.Fatalf("kinds = %v, want %v", kinds, artifactpath.NvimKinds())
	}
	if !slices.Equal(records, paths.NvimPIDs()) {
		t.Fatalf("records = %v, want %v", records, paths.NvimPIDs())
	}
}

// The scrollback attribution goes through artifactpath's one argv parser, so a
// tag that is a prefix of another is not mis-attributed (a bare prefix match
// would reap `work-2`'s viewer when `work` quits).
func TestReapNvimSidecarsAttributesScrollbackByTagNotPrefix(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		owner, ansi string
		wantKill    bool
	}{
		{"work", "scrollback-work-claude.ansi", true},
		{"work", "scrollback-work-2-claude.ansi", false},
		{"work-2", "scrollback-work-claude.ansi", false},
		{"work-2", "scrollback-work-2-claude.ansi", true},
	} {
		table := reapNvimTable(4242, draftArgv(filepath.Join(reapScopeDir, tc.ansi)))
		sidecars := newFakeRuntime()
		target := scrollbackTarget(reapScopeDir)
		target.Owner = tc.owner
		sidecars.files[target.Path] = "4242\n"
		if _, err := reapNvimSidecars(context.Background(), table, sidecars, []nvimSidecar{target}); err != nil {
			t.Fatal(err)
		}
		if killed := len(table.signals) == 1; killed != tc.wantKill {
			t.Fatalf("owner %q against %s: signals %v, want kill=%v", tc.owner, tc.ansi, table.signals, tc.wantKill)
		}
	}
}

// A target that needs tag attribution but has no scope to attribute with is
// never killed: missing evidence is a skip, not a licence.
func TestReapNvimSidecarsSkipsAttributionWithoutAScope(t *testing.T) {
	t.Parallel()
	table := reapNvimTable(4242, draftArgv(filepath.Join(reapScopeDir, "scrollback-work-claude.ansi")))
	sidecars := newFakeRuntime()
	target := scrollbackTarget("")
	sidecars.files[target.Path] = "4242\n"
	outcomes, err := reapNvimSidecars(context.Background(), table, sidecars, []nvimSidecar{target})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.signals) != 0 || outcomes[0].Reason != reapReasonOtherTag {
		t.Fatalf("signals %v reason %q, want no kill and %q", table.signals, outcomes[0].Reason, reapReasonOtherTag)
	}
	if _, kept := sidecars.files[target.Path]; !outcomes[0].Removed || kept {
		t.Fatal("a decided not-ours record should be gone")
	}
}

// sweepScope is a real scope directory on disk, because the sweep enumerates it:
// the records it finds are the evidence, and a map-backed fake cannot stand in
// for a glob.
type sweepScope struct {
	dir   string
	paths artifactpath.Paths
}

func newSweepScope(t *testing.T, tag string) *sweepScope {
	t.Helper()
	dir := t.TempDir()
	paths, err := artifactpath.ResolveScoped(dir, tag)
	if err != nil {
		t.Fatal(err)
	}
	return &sweepScope{dir: dir, paths: paths}
}

func (s *sweepScope) write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// record writes one tag-keyed sidecar naming pid.
func (s *sweepScope) record(t *testing.T, kind string, pid int) string {
	t.Helper()
	path := s.paths.NvimPID(kind)
	s.write(t, path, strconv.Itoa(pid)+"\n")
	return path
}

// reviewTarget writes the tag's review state naming file. It is the review
// kind's owner-naming evidence: a review pane serves a file anywhere in the
// user's tree, so only the tag's own state ties a pid to its owner (#583 BR-6).
func (s *sweepScope) reviewTarget(t *testing.T, file string) {
	t.Helper()
	s.write(t, s.paths.ReviewTarget(), reviewStateJSON(file))
}

// changelog writes the viewer's artifact-keyed record, and the opener's lock
// beside it when opener is positive.
func (s *sweepScope) changelog(t *testing.T, agent, session string, pid, opener int) artifactpath.ChangelogArtifactSet {
	t.Helper()
	set, err := s.paths.ChangelogArtifacts(agent, session)
	if err != nil {
		t.Fatal(err)
	}
	s.write(t, set.NvimPID, strconv.Itoa(pid)+"\n")
	if opener > 0 {
		s.write(t, set.OpenLock, strconv.Itoa(opener)+"\n")
	}
	return set
}

func (s *sweepScope) draftArgv() []string { return draftArgv(s.paths.Draft()) }

func (s *sweepScope) scrollbackArgv(t *testing.T, agent string) []string {
	t.Helper()
	scroll, err := s.paths.ScrollbackArtifacts(agent)
	if err != nil {
		t.Fatal(err)
	}
	return draftArgv(scroll.ANSI)
}

func (s *sweepScope) remainingRecords(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, pattern := range []string{"nvim-pid-*", "changelog-*.nvim-pid"} {
		matches, err := filepath.Glob(filepath.Join(s.dir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, matches...)
	}
	slices.Sort(out)
	return out
}

// nvimRow puts one nvim in the snapshot with the given argv.
func nvimRow(table *fakeProcessTable, pid, ppid int, args []string) *fakeProcess {
	row := table.add(pid, ppid, reapStamp)
	row.name, row.args, row.argsOK = reapNvimExe, args, true
	return row
}

// embed puts one reparented embedded nvim in the snapshot: the shape a dead
// owner leaves behind.
func embed(table *fakeProcessTable, pid int, args []string) *fakeProcess {
	return nvimRow(table, pid, 1, args)
}

// opener puts one live changelog opener in the snapshot: the process that holds
// the openlock for as long as it holds the viewer.
func opener(table *fakeProcessTable, pid int) *fakeProcess {
	row := table.add(pid, 1, reapStamp)
	row.name, row.args, row.argsOK = "/pair/bin/pair", []string{"pair", "changelog-open"}, true
	return row
}

// The sweep reads the whole scope at once, so its adversarial class is
// mis-attribution: three evidence sources over one snapshot, and a pid two
// sources name is signalled once.
func TestSweepOrphanNvimDecidesEachPidOnItsOwnEvidence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		liveTags []string
		setup    func(t *testing.T, scope *sweepScope, table *fakeProcessTable)
		// wantSignals is compared as a set: the snapshot's row order is not
		// part of the contract.
		wantSignals []string
		// wantKept names the records that must survive, by base name.
		wantKept []string
	}{
		{
			name: "a dead tag's draft embed",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "draft", 4242)
				embed(table, 4242, scope.draftArgv())
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			// A live tag's embed is a candidate: the pane that spawned it can close
			// while the session lives on, and the embed it leaves at ppid 1 is the
			// 13-day shape #583 exists for. Its healthy neighbour — a live parent —
			// is the case that must never be signalled.
			name:     "a live tag's draft embed whose parent is alive",
			liveTags: []string{"work"},
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "draft", 4242)
				nvimRow(table, 4242, 4000, scope.draftArgv())
			},
			wantKept: []string{"nvim-pid-work-draft"},
		},
		{
			name:     "a live tag's draft embed reparented to launchd",
			liveTags: []string{"work"},
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "draft", 4242)
				embed(table, 4242, scope.draftArgv())
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			name: "a dead tag's scrollback embed",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "scrollback", 4242)
				embed(table, 4242, scope.scrollbackArgv(t, "claude"))
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			// `scrollback-work-` is a prefix of `scrollback-work-2-`: a prefix
			// match would hand a live tag's viewer to its dead neighbour. The
			// dead tag's stale record is dropped, because the pid it names is
			// not its own. The embed's parent is alive so the case stays about
			// attribution: a reparented live-tag embed is its own case above.
			name:     "a live tag's scrollback embed found through a dead tag's record",
			liveTags: []string{"work-2"},
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "scrollback", 4242)
				other, err := artifactpath.ResolveScoped(scope.dir, "work-2")
				if err != nil {
					t.Fatal(err)
				}
				scroll, err := other.ScrollbackArtifacts("claude")
				if err != nil {
					t.Fatal(err)
				}
				nvimRow(table, 4242, 4000, draftArgv(scroll.ANSI))
			},
		},
		{
			name: "an unreaped review sidecar",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "review", 4242)
				scope.reviewTarget(t, "/src/post.md")
				embed(table, 4242, []string{"nvim", "--embed", "-u", "/anywhere/nvim/review.lua", "/src/post.md"})
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			// The sweep-level shape of #583 BR-6: a review record that outlived its
			// embed, and the pid it names now holds another tag's live pane. Both
			// argv carry review.lua, so the config's base name says "a pair review
			// pane" and not whose; only the file this tag's own state names decides,
			// and that pane serves another file. Nothing is signalled, and the stale
			// record goes: it names a process that is not this tag's.
			name: "a dead tag's review record whose pid another tag's live pane holds",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "review", 4242)
				scope.reviewTarget(t, "/src/post.md")
				nvimRow(table, 4242, 4000, []string{"nvim", "--embed", "-u", "/anywhere/nvim/review.lua", "/src/theirs.md"})
			},
		},
		{
			// A review state that names no file is missing evidence, not a pass:
			// the pane may well be the record's own, and the record stays for the
			// next pass rather than authorizing a kill nothing attributes.
			name: "a review sidecar whose state names no file",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "review", 4242)
				scope.write(t, scope.paths.ReviewTarget(), `{"status":"ready","session":"sid1"}`)
				embed(table, 4242, []string{"nvim", "--embed", "-u", "/anywhere/nvim/review.lua", "/src/post.md"})
			},
			wantKept: []string{"nvim-pid-work-review"},
		},
		{
			name: "a changelog viewer whose opener still holds the lock",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				set := scope.changelog(t, "claude", "sid1", 4242, 700)
				opener(table, 700)
				nvimRow(table, 600, 700, []string{"nvim", "-u", "/pair/nvim/changelog.lua", set.Log})
				nvimRow(table, 4242, 600, []string{"nvim", "--embed", "-u", "/pair/nvim/changelog.lua", set.Log})
			},
			wantKept: []string{"changelog-work-claude-sid1.nvim-pid"},
		},
		{
			name: "a changelog viewer whose opener died without clearing the lock",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				set := scope.changelog(t, "claude", "sid1", 4242, 700)
				embed(table, 4242, draftArgv(set.Log))
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			// A clean exit removes the openlock (opener/run.go's defer), so its
			// absence is the owner's death, not missing evidence.
			name: "a changelog viewer left behind by a clean exit",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				set := scope.changelog(t, "claude", "sid1", 4242, 0)
				embed(table, 4242, draftArgv(set.Log))
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			name: "a changelog viewer serving another log",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.changelog(t, "claude", "sid1", 4242, 0)
				embed(table, 4242, draftArgv(filepath.Join(scope.dir, "changelog-work-claude-sid2.md")))
			},
		},
		{
			// M2's classifier owns an unattributed embed; the sweep must not
			// reach it just because it is an orphan-shaped nvim.
			name: "a bare embed with no scope path at all",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				embed(table, 4242, []string{"nvim", "--embed"})
			},
		},
		{
			name: "one embed named by both a sidecar and its argv",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "draft", 4242)
				embed(table, 4242, scope.draftArgv())
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			// Two records naming one pid are two pieces of evidence about the
			// same process: one signal, and neither record left behind to name a
			// pid the sweep has already accounted for.
			name: "two sidecars naming the same embed",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.record(t, "draft", 4242)
				scope.record(t, "review", 4242)
				embed(table, 4242, scope.draftArgv())
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			name: "an argv-only embed of a dead tag",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				embed(table, 4242, scope.draftArgv())
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			name:     "an argv-only embed of a live tag whose parent is alive",
			liveTags: []string{"work"},
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				nvimRow(table, 4242, 4000, scope.draftArgv())
			},
		},
		{
			// No record, so nothing to remove: argv attribution plus ppid 1 is the
			// whole evidence, and it is enough for the classifier.
			name:     "an argv-only embed of a live tag reparented to launchd",
			liveTags: []string{"work"},
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				embed(table, 4242, scope.draftArgv())
			},
			wantSignals: []string{"4242:KILL"},
		},
		{
			name: "a record that does not name a pid",
			setup: func(t *testing.T, scope *sweepScope, table *fakeProcessTable) {
				scope.write(t, scope.paths.NvimPID("draft"), "not-a-pid\n")
				embed(table, 4242, scope.draftArgv())
			},
			wantSignals: []string{"4242:KILL"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scope := newSweepScope(t, "work")
			table := newFakeProcessTable()
			tc.setup(t, scope, table)
			if _, err := sweepOrphanNvim(context.Background(), table, NewOSRuntime(scope.dir, "/pair"), scope.dir, tc.liveTags); err != nil {
				t.Fatal(err)
			}
			signals := slices.Clone(table.signals)
			slices.Sort(signals)
			want := slices.Clone(tc.wantSignals)
			slices.Sort(want)
			if !slices.Equal(signals, want) {
				t.Fatalf("signals = %v, want %v", signals, want)
			}
			var kept []string
			for _, path := range scope.remainingRecords(t) {
				kept = append(kept, filepath.Base(path))
			}
			if !slices.Equal(kept, tc.wantKept) {
				t.Fatalf("records left = %v, want %v", kept, tc.wantKept)
			}
		})
	}
}

// The sweep is part of launch, so it reports a snapshot it cannot take instead
// of deciding on nothing.
func TestSweepOrphanNvimRefusesABrokenSnapshot(t *testing.T) {
	t.Parallel()
	scope := newSweepScope(t, "work")
	scope.record(t, "draft", 4242)
	table := newFakeProcessTable()
	table.snapshotErr = errors.New("sysctl failed")
	if _, err := sweepOrphanNvim(context.Background(), table, NewOSRuntime(scope.dir, "/pair"), scope.dir, nil); err == nil {
		t.Fatal("swept without a snapshot")
	}
	if len(scope.remainingRecords(t)) != 1 {
		t.Fatal("a failed snapshot must leave every record alone")
	}
}
