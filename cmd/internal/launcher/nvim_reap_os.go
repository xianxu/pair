package launcher

import (
	"context"
	"fmt"
	"path/filepath"
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

// nvimSidecar is one pid record to reap, with the argv attribution its kind can
// make and whether its owner is live.
type nvimSidecar struct {
	Path  string
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
// scopeDir attributes a scrollback embed by tag; empty leaves it unattributable,
// which skips rather than kills.
func nvimSidecarsFor(paths artifactpath.Paths, owner, scopeDir string) []nvimSidecar {
	kinds := artifactpath.NvimKinds()
	targets := make([]nvimSidecar, 0, len(kinds))
	for _, kind := range kinds {
		target := nvimSidecar{Path: paths.NvimPID(kind), Kind: kind, Owner: owner}
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

// reapNvimSidecars ends the embedded nvim processes a set of pid records name,
// and it is the whole reap path: one snapshot for pid/parent/identity/name, one
// argv read per named pid, ClassifyNvimReap's decision, and an identity re-read
// immediately before the signal (#399's fence). There is no subprocess seam
// here, so no reap can reach a process by name (#583) -- the only signal it
// sends is SIGKILL to a pid whose kernel start stamp still matches the stamp the
// snapshot read.
//
// SIGKILL, not SIGTERM: every caller has already established that the owner is
// gone or is quitting, so no session is left to flush, and quit is time-bounded.
func reapNvimSidecars(ctx context.Context, table ProcessTable, records nvimSidecarIO, targets []nvimSidecar) ([]nvimReapOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := table.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("nvim reap: snapshot processes: %w", err)
	}
	byPID := make(map[int]ProcessRow, len(rows))
	for _, row := range rows {
		byPID[row.PID] = row
	}
	outcomes := make([]nvimReapOutcome, 0, len(targets))
	for _, target := range targets {
		raw, err := records.ReadFile(target.Path)
		if err != nil {
			continue // no record: nothing to decide, nothing to remove
		}
		pid, ok := sidecarPID(raw)
		outcome := nvimReapOutcome{Kind: target.Kind, Owner: target.Owner, Path: target.Path, PID: pid}
		kill, reason := false, reapReasonMalformed
		if ok {
			kill, reason = decideNvimSidecar(table, target, pid, byPID)
		}
		outcome.Reason = reason
		if kill {
			// A refused signal is not a kill: the pid raced away, or the kernel
			// will not let us. The record still goes — the decision was made and
			// the process it named is not this tag's to keep tracking.
			outcome.Killed = table.Signal(pid, syscall.SIGKILL) == nil
		}
		if !reapKeepsRecord(reason) {
			records.Remove(target.Path)
			outcome.Removed = true
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}

// decideNvimSidecar is one record's decision, identity fence included: the
// kernel start stamp is re-read before the signal is allowed, so a pid recycled
// since the snapshot is left alone -- and its record is dropped, because the
// process it named is gone.
func decideNvimSidecar(table ProcessTable, target nvimSidecar, pid int, rows map[int]ProcessRow) (bool, string) {
	row, present := rows[pid]
	if !present {
		return false, reapReasonGone
	}
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
