package launcher

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/reviewcmd"
)

// The record-side reasons: what a pid record can say that the process table
// cannot. ClassifyNvimReap owns the rest of the vocabulary.
const (
	reapReasonMalformed = "record does not name a pid"
	reapReasonGone      = "no process holds the pid"
	reapReasonRecycled  = "pid recycled before the signal"
)

// The closed kind set's members (artifactpath.NvimKinds), named here because
// the argv evidence a kind carries differs. TestNvimSidecarsGiveEveryKindArgvEvidence
// is the drift guard: a kind that arrives without a predicate fails it, rather
// than being reaped on `nvim --embed` alone.
const (
	nvimKindDraft      = "draft"
	nvimKindScrollback = "scrollback"
	nvimKindReview     = "review"
	// nvimKindChangelog labels the one embedded nvim that is not a kind: the
	// changelog viewer is per (tag, agent, session), so its record is keyed to
	// the artifact set it serves and its owner is the opener, not a tag.
	nvimKindChangelog = "changelog"
)

// reviewLuaAsset is the config every pair review pane passes to `-u`. It is the
// only field in a review embed's argv that is pair's: the rest names the
// reviewed file, which lives anywhere in the user's tree.
const reviewLuaAsset = reviewcmd.ReviewScript

// nvimSidecarIO is the pid-record IO the reap needs. OSRuntime satisfies it.
type nvimSidecarIO interface {
	ReadFile(path string) (string, error)
	Remove(path string)
}

// nvimSidecar is one embedded nvim to reap, with the argv attribution its
// evidence can make and whether its owner is live.
type nvimSidecar struct {
	// Path is the pid record to read and, once decided, remove. PID names the
	// process directly instead, for an embed the snapshot found with no record
	// behind it. A target carrying neither is nothing to decide.
	Path  string
	PID   int
	Kind  string
	Owner string
	// OwnerLive is false on a path that is itself the authorization to reap
	// (quit, orphan reap); the sweep sets it from the live-tag set.
	OwnerLive         bool
	ExpectedArgPath   string
	ExpectArgvTag     bool
	ArgvScopeDir      string
	ExpectedAssetBase string
}

// nvimReapOutcome is one record's decided result. Killed reports a SIGKILL the
// kernel accepted: the identity fence can refuse the signal after the decision,
// and so can the host. Removed reports whether the record is off disk.
type nvimReapOutcome struct {
	Kind, Owner, Path string
	PID               int
	Killed, Removed   bool
	Reason            string
}

// nvimSidecarsFor builds one target per kind of the closed set, so a kind cannot
// drift out of the reap (#583): the launch sweep enumerated {draft, scrollback}
// by hand, and a review pane's embed was never a candidate.
//
// ownerLive is the caller's evidence about the owner, not a property of the kind:
// a path that is itself the authorization to reap (quit) passes false, and the
// sweep passes what the live-tag set says, so a live tag's embed is only signalled
// on the classifier's reparented rule.
//
// scopeDir attributes a scrollback embed by tag; empty leaves it unattributable,
// which skips rather than kills.
func nvimSidecarsFor(paths artifactpath.Paths, owner, scopeDir string, ownerLive bool) []nvimSidecar {
	kinds := artifactpath.NvimKinds()
	targets := make([]nvimSidecar, 0, len(kinds))
	for _, kind := range kinds {
		target := nvimSidecar{Path: paths.NvimPID(kind), Kind: kind, Owner: owner, OwnerLive: ownerLive}
		switch kind {
		case nvimKindDraft:
			target.ExpectedArgPath = paths.Draft()
		case nvimKindScrollback:
			target.ExpectArgvTag, target.ArgvScopeDir = true, scopeDir
		case nvimKindReview:
			target.ExpectedAssetBase = filepath.Base(reviewLuaAsset)
		}
		targets = append(targets, target)
	}
	return targets
}

// sweepOrphanNvim reaps the embedded nvim processes whose owner is gone, one
// snapshot shared by every evidence source (#583). The sources disagree about
// what identifies an owner, so each states its own attribution and all three
// decide through the same ClassifyNvimReap row:
//
//   - a tag-keyed sidecar names a pid, and its owner is the tag in the record's
//     own name: dead once the live-tag set does not carry it, and for a live tag
//     only reparenting to launchd authorizes the kill — the pane that spawned the
//     embed closed while its session lived on;
//   - a changelog record's owner is the opener holding the sibling openlock, so
//     a viewer is reaped on the opener's death and not on its tag's;
//   - an embed that left no record is attributed by argv, the weakest evidence,
//     which only ever reaches a scope path pair's own parser recognizes.
//
// An embed argv cannot attribute stays out: that is the untagged orphan M2
// reports and reaps on confirmation, never a sweep kill.
func sweepOrphanNvim(ctx context.Context, table ProcessTable, records nvimSidecarIO, scopeDir string, liveTags []string) ([]nvimReapOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scope, err := artifactpath.ResolveSelectedScope(scopeDir)
	if err != nil {
		return nil, err
	}
	rows, err := table.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("nvim sweep: snapshot processes: %w", err)
	}
	live := make(map[string]bool, len(liveTags))
	for _, tag := range liveTags {
		live[tag] = true
	}
	byPID := indexByPID(rows)
	targets := tagSidecars(scope, scopeDir, live)
	targets = append(targets, changelogSidecars(scope, records, byPID)...)
	targets = append(targets, argvOnlyEmbeds(scope, table, scopeDir, live, rows)...)
	return reapNvimRows(ctx, table, records, targets, byPID)
}

// indexByPID is the snapshot's one index construction: the openlock liveness
// check and every decision read the same rows, so the map is built once per pass.
func indexByPID(rows []ProcessRow) map[int]ProcessRow {
	byPID := make(map[int]ProcessRow, len(rows))
	for _, row := range rows {
		byPID[row.PID] = row
	}
	return byPID
}

// tagSidecars enumerates the tag-keyed records in the scope and targets every
// kind of the closed set for each tag it finds, live or dead, carrying the
// liveness the sweep observed. An absent record decides nothing, so a tag's whole
// kind set is safe to target: it is how a kind that only one surface writes still
// gets reaped. A live tag's kinds are targets too, because its embed can outlive
// the pane that spawned it — the classifier then needs ppid 1 before it signals,
// and a healthy embed with its parent alive is kept.
func tagSidecars(scope artifactpath.ScopePaths, scopeDir string, live map[string]bool) []nvimSidecar {
	globs, err := scope.NvimPIDGlobs()
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, glob := range globs {
		matches, err := filepath.Glob(glob.Pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if tag, ok := scope.TagFromNvimPID(match, glob.Kind); ok {
				seen[tag] = true
			}
		}
	}
	tags := slices.Sorted(maps.Keys(seen))
	targets := make([]nvimSidecar, 0, len(tags)*len(artifactpath.NvimKinds()))
	for _, tag := range tags {
		paths, err := artifactpath.ResolveScoped(scopeDir, tag)
		if err != nil {
			continue
		}
		targets = append(targets, nvimSidecarsFor(paths, tag, scopeDir, live[tag])...)
	}
	return targets
}

// changelogSidecars targets the changelog viewers in the scope. The tag is not
// the owner here: one tag can have a viewer per session, and a live tag's viewer
// is still an orphan once the opener that held it is gone.
func changelogSidecars(scope artifactpath.ScopePaths, records nvimSidecarIO, byPID map[int]ProcessRow) []nvimSidecar {
	matches, err := filepath.Glob(scope.ChangelogNvimPIDGlob())
	if err != nil {
		return nil
	}
	slices.Sort(matches)
	targets := make([]nvimSidecar, 0, len(matches))
	for _, match := range matches {
		set, err := scope.ChangelogSetFromNvimPID(match)
		if err != nil {
			continue
		}
		targets = append(targets, nvimSidecar{
			Path: set.NvimPID, Kind: nvimKindChangelog, Owner: set.Log,
			OwnerLive:       openerLive(records, set.OpenLock, byPID),
			ExpectedArgPath: set.Log,
		})
	}
	return targets
}

// openerLive reads the lock the changelog opener holds for as long as it holds
// the viewer (RunViewer waits on it) and removes on a clean exit. So an absent
// lock is a dead owner, and a lock naming a pid the snapshot does not carry is
// one too; only a live pid keeps the viewer. A lock that does not name a pid is
// not the opener's, and the row's own predicates still gate the kill.
func openerLive(records nvimSidecarIO, openlock string, byPID map[int]ProcessRow) bool {
	raw, err := records.ReadFile(openlock)
	if err != nil {
		return false
	}
	pid, ok := sidecarPID(raw)
	if !ok {
		return false
	}
	_, present := byPID[pid]
	return present
}

// argvOnlyEmbeds attributes embeds that left no pid record. argv is the whole
// evidence, so the target states the predicate argv just satisfied: the draft's
// exact artifact path, or a tag attribution through the scope's one parser. A live
// tag's embed is attributed too, with the liveness the sweep observed, so the
// classifier decides it on ppid rather than the source dropping it.
func argvOnlyEmbeds(scope artifactpath.ScopePaths, table ProcessTable, scopeDir string, live map[string]bool, rows []ProcessRow) []nvimSidecar {
	var targets []nvimSidecar
	for _, row := range rows {
		if filepath.Base(row.Name) != "nvim" {
			continue
		}
		args, ok := table.Command(row.PID)
		if !ok {
			continue
		}
		tag := scope.TagFromNvimEmbedArgv(strings.Join(args, " "))
		if tag == "" {
			continue
		}
		paths, err := artifactpath.ResolveScoped(scopeDir, tag)
		if err != nil {
			continue
		}
		target := nvimSidecar{PID: row.PID, Owner: tag, OwnerLive: live[tag]}
		if argvCarriesField(args, paths.Draft()) {
			target.Kind, target.ExpectedArgPath = nvimKindDraft, paths.Draft()
		} else {
			target.Kind, target.ExpectArgvTag, target.ArgvScopeDir = nvimKindScrollback, true, scopeDir
		}
		targets = append(targets, target)
	}
	return targets
}

// reapNvimSidecars ends the embedded nvim processes a set of pid records name,
// taking the snapshot it decides against. Callers that already hold one — the
// sweep, whose evidence sources all read it — call reapNvimRows instead.
func reapNvimSidecars(ctx context.Context, table ProcessTable, records nvimSidecarIO, targets []nvimSidecar) ([]nvimReapOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := table.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("nvim reap: snapshot processes: %w", err)
	}
	return reapNvimRows(ctx, table, records, targets, indexByPID(rows))
}

// reapNvimRows is the whole reap path: one snapshot for pid/parent/identity/name,
// one argv read per named pid, ClassifyNvimReap's decision, and an identity
// re-read immediately before the signal (#399's fence). There is no subprocess
// seam here, so no reap can reach a process by name (#583) — the only signal it
// sends is SIGKILL to a pid whose kernel start stamp still matches the stamp the
// snapshot read.
//
// SIGKILL, not SIGTERM: every caller has already established that the owner is
// gone or is quitting, so no session is left to flush, and quit is time-bounded.
//
// Deciding against a snapshot already taken is what lets the sweep share one
// across its evidence sources: pid, parent, identity and the openlock liveness
// it reads all describe the same moment, and byPID is that snapshot's one index
// — the sweep hands the same map to its openlock check and to this pass.
//
// A pid is decided once: the sweep's sources overlap, and two records naming one
// pid are two pieces of evidence about the same process, so a second decision
// would only report a kill the first already made. Every record naming a decided
// pid still follows it off disk. Record-backed targets come first, so the source
// that owns a record is the one whose attribution decides it.
func reapNvimRows(ctx context.Context, table ProcessTable, records nvimSidecarIO, targets []nvimSidecar, byPID map[int]ProcessRow) ([]nvimReapOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	outcomes := make([]nvimReapOutcome, 0, len(targets))
	decided := make(map[int]string, len(targets))
	for _, target := range targets {
		pid, ok := target.PID, target.PID > 0
		if !ok {
			raw, err := records.ReadFile(target.Path)
			if err != nil {
				continue // no record: nothing to decide, nothing to remove
			}
			pid, ok = sidecarPID(raw)
		}
		if !ok {
			outcomes = append(outcomes, decideOne(table, records, target, pid, reapReasonMalformed, byPID))
			continue
		}
		// Two records can name one pid — a stale sidecar beside a current one —
		// and they are two pieces of evidence about the same process, not two
		// processes. The first decision stands, so the pid is signalled once; the
		// later record still follows it off disk, because it names a process the
		// pass has already accounted for.
		if reason, seen := decided[pid]; seen {
			if target.Path != "" && !reapKeepsRecord(reason) {
				records.Remove(target.Path)
			}
			continue
		}
		outcome := decideOne(table, records, target, pid, "", byPID)
		decided[pid] = outcome.Reason
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}

// decideOne is one target's decision, signal and record rule. A pre-set reason
// short-circuits the decision: it is what a record that names no pid gets.
func decideOne(table ProcessTable, records nvimSidecarIO, target nvimSidecar, pid int, reason string, byPID map[int]ProcessRow) nvimReapOutcome {
	outcome := nvimReapOutcome{Kind: target.Kind, Owner: target.Owner, Path: target.Path, PID: pid}
	kill := false
	if reason == "" {
		kill, reason = decideNvimSidecar(table, target, pid, byPID)
	}
	outcome.Reason = reason
	if kill {
		// A refused signal is not a kill: the pid raced away, or the kernel
		// will not let us. The record still goes — the decision was made and
		// the process it named is not this tag's to keep tracking.
		outcome.Killed = table.Signal(pid, syscall.SIGKILL) == nil
	}
	if target.Path != "" && !reapKeepsRecord(reason) {
		records.Remove(target.Path)
		outcome.Removed = true
	}
	return outcome
}

// decideNvimSidecar is one record's decision, identity fence included: the
// kernel start stamp is re-read before the signal is allowed, so a pid recycled
// since the snapshot is left alone -- and its record is dropped, because the
// process it named is gone.
func decideNvimSidecar(table ProcessTable, target nvimSidecar, pid int, byPID map[int]ProcessRow) (bool, string) {
	row, present := byPID[pid]
	if !present {
		return false, reapReasonGone
	}
	// argv is read here rather than carried in from the source that attributed the
	// row: the kill has to rest on evidence taken beside the identity fence below,
	// not on a read from an earlier moment in the pass.
	args, argsKnown := table.Command(pid)
	candidate := NvimReapRow{
		Kind: target.Kind, Owner: target.Owner, PID: pid, PPID: row.PPID,
		Identity: row.Identity, Name: row.Name,
		Args: args, ArgsKnown: argsKnown, OwnerLive: target.OwnerLive,
		ExpectedArgPath: target.ExpectedArgPath, ExpectArgvTag: target.ExpectArgvTag,
		ExpectedAssetBase: target.ExpectedAssetBase,
	}
	if candidate.ExpectArgvTag && argsKnown {
		candidate.ArgvTag = argvTag(target.ArgvScopeDir, args)
	}
	kill, reason := ClassifyNvimReap(candidate)
	if !kill {
		return false, reason
	}
	if table.Identity(pid) != row.Identity {
		return false, reapReasonRecycled
	}
	return true, reason
}

// argvTag attributes argv to a tag through artifactpath's one parser. A scope it
// cannot use attributes nothing, and an unattributed candidate is skipped.
func argvTag(scopeDir string, args []string) string {
	scope, err := artifactpath.ResolveSelectedScope(scopeDir)
	if err != nil {
		return ""
	}
	return scope.TagFromNvimEmbedArgv(strings.Join(args, " "))
}

// reapKeepsRecord reports whether an undecided outcome must leave the record on
// disk. Evidence that is merely missing -- a live owner, an unreadable identity
// or argv -- keeps it for the next pass; a decided negative does not, because
// the record then names a process that is not ours, or no process at all.
// Storage GC owns whatever the passes keep finding undecided
// (artifactpath: nvim-pid is collectable).
func reapKeepsRecord(reason string) bool {
	switch reason {
	case reapReasonOwnerLive, reapReasonIdentityUnknown, reapReasonArgvUnknown:
		return true
	}
	return false
}

// sidecarPID reads a record's pid. Anything but one positive integer is
// malformed, and the record is dropped rather than handed to a subprocess.
func sidecarPID(raw string) (int, bool) {
	pid, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

// processTable is the reap's host seam: the runtime's injected process source
// behind the table the server reaper already reads through.
func (r OSRuntime) processTable() ProcessTable { return OSProcessTable{Source: r.Processes} }
