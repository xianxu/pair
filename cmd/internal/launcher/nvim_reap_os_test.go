package launcher

import (
	"context"
	"path/filepath"
	"slices"
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
)

func draftArgv(path string) []string {
	return []string{"nvim", "--embed", "-u", "/pair/nvim/init.lua", path}
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
		target = nvimSidecar{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectedArgPath: reapDraft}
	}
	sidecars := newFakeRuntime()
	sidecars.files[target.Path] = c.sidecar
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
		ExpectedAssetBase: "review.lua",
	}
	liveOwner := nvimSidecar{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectedArgPath: reapDraft, OwnerLive: true}
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
			name: "a review embed running pair's review config", sidecar: "4242\n",
			target:     review,
			table:      reapNvimTable(4242, []string{"nvim", "--embed", "-u", "/anywhere/nvim/review.lua", "/src/post.md"}),
			wantKill:   true,
			wantReason: reapReasonOwnerDead,
		},
		{
			name: "a bare embed sitting at a review record's pid", sidecar: "4242\n",
			target:     review,
			table:      reapNvimTable(4242, []string{"nvim", "--embed"}),
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
		{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectedArgPath: reapDraft},
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
		{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectedArgPath: reapDraft},
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
		{Path: reapSidecar, Kind: "draft", Owner: "work", ExpectedArgPath: reapDraft},
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

// Every kind in the closed set must carry an argv predicate. A kind added to
// artifactpath.NvimKinds() without one would be reaped on `nvim --embed` alone,
// which is the name-only kill #583 forbids.
func TestNvimSidecarsGiveEveryKindArgvEvidence(t *testing.T) {
	t.Parallel()
	paths, err := artifactpath.ResolveScoped(t.TempDir(), "work")
	if err != nil {
		t.Fatal(err)
	}
	targets := nvimSidecarsFor(paths, "work", paths.ScopeDir())
	if len(targets) != len(artifactpath.NvimKinds()) {
		t.Fatalf("targets = %+v, want one per kind %v", targets, artifactpath.NvimKinds())
	}
	for _, target := range targets {
		if target.Owner != "work" || target.Path == "" || target.Kind == "" || target.OwnerLive {
			t.Fatalf("target %+v: want the owner's record, no liveness claim", target)
		}
		if target.ExpectedArgPath == "" && !target.ExpectArgvTag && target.ExpectedAssetBase == "" {
			t.Fatalf("kind %q carries no argv predicate: %+v", target.Kind, target)
		}
		if target.Path != paths.NvimPID(target.Kind) {
			t.Fatalf("kind %q reads %q, want %q", target.Kind, target.Path, paths.NvimPID(target.Kind))
		}
	}
	// The draft's predicate is the exact artifact path; the review pane's is the
	// asset its launcher passes to `-u`, matched by base because pair's install
	// root differs per checkout (a live embed outlives the checkout that spawned
	// it, and its argv keeps that checkout's path).
	for _, target := range targets {
		switch target.Kind {
		case "draft":
			if target.ExpectedArgPath != paths.Draft() {
				t.Fatalf("draft expects %q, want %q", target.ExpectedArgPath, paths.Draft())
			}
		case "review":
			if want := filepath.Base(reviewLuaAsset); target.ExpectedAssetBase != want {
				t.Fatalf("review expects asset %q, want %q", target.ExpectedAssetBase, want)
			}
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
	targets := nvimSidecarsFor(paths, "work", paths.ScopeDir())
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
