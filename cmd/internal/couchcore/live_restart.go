package couchcore

import (
	"context"
	"fmt"
	"strings"
)

// Live-restart refusal codes (pair#421): why `couch --relaunch` or
// `couch --reload-context` declined a live slot. They reach the caller as the
// receipt's Code, next to the slot-operation codes above.
const (
	LiveRestartNotLive     = "not-live"     // nothing is running; --resume starts it
	LiveRestartBusyUnknown = "busy-unknown" // the wrapper cannot say (pre-#421, or no session)
	LiveRestartBusy        = "busy"         // a turn is open, the composer is occupied, or input is fresh
	LiveRestartDirty       = "dirty"        // the checkout has uncommitted changes
	LiveRestartStaleBinary = "stale-binary" // relaunch would run the binary the slot already runs
	LiveRestartUnavailable = "unavailable"  // this Couch cannot probe live slots
	OpRelaunch             = "relaunch"
	OpReloadContext        = "reload-context"
)

// LiveRestartFacts is everything the admission rule reads, gathered by a
// LiveRestartProbe at execution time on the console queue.
type LiveRestartFacts struct {
	// Live: the row has a running (occupied) incarnation.
	Live bool
	// Session: an admitted wrapper session stands for the row's thread.
	Session bool
	// SettledKnown: that session reports Settled (a hello-v2 wrapper that has
	// judged at least once); Settled is its latest claim.
	SettledKnown, Settled bool
	// GitKnown: the checkout's status could be read; Dirty is the result.
	GitKnown, Dirty bool
	Binary          BinaryFacts
}

// BinaryFacts compare the executable a relaunch would run with the one the
// slot runs now (relaunch only).
type BinaryFacts struct {
	// DevRebuild: Couch runs with PAIR_DEV, which slots inherit, so pair's own
	// dev_rebuild runs during the relaunch; the on-disk binary is not final.
	DevRebuild bool
	// RunningSHA is the slot wrapper's own content hash ("" when unknown).
	RunningSHA string
	// OnDiskPath is `pair` as Couch resolves it; OnDiskSHA "" when unreadable.
	OnDiskPath, OnDiskSHA, OnDiskRevision string
	OnDiskModified                        bool
	// Checkout is the git tree that owns OnDiskPath (its bin/ parent), and
	// CheckoutHEAD that tree's HEAD; both "" when not a git checkout.
	Checkout, CheckoutHEAD string
}

// LiveRestartOptions are the caller's explicit overrides, each recorded in the
// receipt note when used.
type LiveRestartOptions struct {
	// ForceUnknown admits a slot whose wrapper cannot report Settled
	// (pre-#421). It never overrides a known busy.
	ForceUnknown bool
	// SameBinary admits a relaunch onto the binary already running.
	SameBinary bool
}

// LiveRestartDecision: Code "" admits; Note is told to the caller either way.
type LiveRestartDecision struct {
	Code, Detail, Note string
}

// DecideLiveRestart is the one admission rule for both verbs. Pure. The checks
// run in a fixed order so the reported reason is the first thing the caller
// must fix.
func DecideLiveRestart(op string, f LiveRestartFacts, o LiveRestartOptions) LiveRestartDecision {
	var notes []string
	refuse := func(code, detail string) LiveRestartDecision {
		return LiveRestartDecision{Code: code, Detail: detail, Note: strings.Join(notes, "; ")}
	}
	switch {
	case !f.Live:
		return refuse(LiveRestartNotLive, "nothing is running in the slot; use --resume")
	case !f.Session || !f.SettledKnown:
		if !o.ForceUnknown {
			why := "its wrapper predates settle reporting (relaunch it once by hand, Alt+n)"
			if !f.Session {
				why = "no wrapper session is connected for it"
			}
			return refuse(LiveRestartBusyUnknown, "cannot tell whether the slot is idle: "+why+"; --force-unknown overrides")
		}
		notes = append(notes, "idle state unverified (--force-unknown)")
	case !f.Settled:
		return refuse(LiveRestartBusy, "the agent has a turn open, the composer holds text, or input was seen in the last few seconds")
	}
	switch {
	case !f.GitKnown:
		return refuse(LiveRestartDirty, "the checkout's status could not be read")
	case f.Dirty:
		return refuse(LiveRestartDirty, "the checkout has uncommitted changes")
	}
	if op == OpRelaunch {
		stale, detail, note := DecideBinaryFreshness(f.Binary, o.SameBinary)
		if note != "" {
			notes = append(notes, note)
		}
		if stale {
			return refuse(LiveRestartStaleBinary, detail)
		}
	}
	return LiveRestartDecision{Note: strings.Join(notes, "; ")}
}

// DecideBinaryFreshness: refuse a relaunch onto the very binary the slot runs
// (it would change nothing, which is how pair#418's fix missed a rollout), and
// name the fix. Content hash is the identity: two dirty builds at one revision
// differ. Pure.
func DecideBinaryFreshness(b BinaryFacts, sameBinaryOK bool) (stale bool, detail, note string) {
	fix := "rebuild it"
	if b.Checkout != "" {
		fix = "run make build in " + b.Checkout
	}
	switch {
	case b.DevRebuild:
		return false, "", "PAIR_DEV is set, so the relaunch rebuilds pair itself"
	case b.RunningSHA == "" || b.OnDiskSHA == "":
		return false, "", "binary freshness unverified (the slot or " + pairPathOrUnknown(b.OnDiskPath) + " reported no build identity)"
	case b.RunningSHA == b.OnDiskSHA && !sameBinaryOK:
		return true, fmt.Sprintf("%s is the binary this slot already runs, so a relaunch changes nothing; %s (or pass --same-binary)", pairPathOrUnknown(b.OnDiskPath), fix), ""
	case b.RunningSHA == b.OnDiskSHA:
		return false, "", "relaunching onto the same binary (--same-binary)"
	}
	var notes []string
	if b.OnDiskRevision != "" && b.CheckoutHEAD != "" && b.OnDiskRevision != b.CheckoutHEAD {
		notes = append(notes, fmt.Sprintf("%s was built from %s but its checkout is at %s; %s to pick up the rest",
			pairPathOrUnknown(b.OnDiskPath), short(b.OnDiskRevision), short(b.CheckoutHEAD), fix))
	}
	if b.OnDiskModified {
		notes = append(notes, "it was built from a dirty tree")
	}
	return false, "", strings.Join(notes, "; ")
}

func pairPathOrUnknown(path string) string {
	if path == "" {
		return "pair on Couch's PATH"
	}
	return path
}

func short(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

// LiveRestartProbe gathers facts about one live row (pair#421) and performs
// reload-context's restart. Couch's message service implements it, since only
// it holds the wrapper sessions; tests use a fake. path is the slot's checkout.
type LiveRestartProbe interface {
	LiveRestartFacts(ctx context.Context, op string, row ActionableThreadSummary, path string) (LiveRestartFacts, error)
	// RestartConversation signals the thread's broker-verified wrapper to start
	// a fresh agent conversation, re-checking the process identity first, and
	// returns once a new session proves the restart (ReloadUnconfirmed if not).
	RestartConversation(ctx context.Context, address ThreadAddress) error
}

// ReloadUnconfirmed: the signal was delivered but no new wrapper session was
// seen in time. The restart may still be under way, so the caller peeks before
// retrying; it is never reported as success.
type ReloadUnconfirmed struct{ Detail string }

func (e *ReloadUnconfirmed) Error() string { return "unconfirmed: " + e.Detail }

// ReloadContextResult is a confirmed reload-context.
type ReloadContextResult struct{ Address ThreadAddress }

// ReloadContext is Shift+Alt+N from outside the slot (pair#421): a fresh agent
// conversation in the same Pair process. Admission (prepareLiveRestart) already
// required a settled, clean, live slot; this re-checks liveness under the
// thread's hold, because the queue ran between admission and now.
func (c *Couch) ReloadContext(ctx context.Context, address ThreadAddress) (ReloadContextResult, error) {
	if c.LiveRestart == nil {
		return ReloadContextResult{}, &SlotOperationError{Code: LiveRestartUnavailable, Detail: "this Couch cannot observe live slots"}
	}
	ctx, release, err := c.hold(ctx, address, OpReloadContext)
	if err != nil {
		return ReloadContextResult{}, err
	}
	defer release()
	thread, err := c.Threads.GetThread(address)
	if err != nil {
		return ReloadContextResult{}, err
	}
	if !hasOccupiedIncarnation(thread) {
		return ReloadContextResult{}, &SlotOperationError{Code: LiveRestartNotLive, Detail: string(address.Tag) + " is no longer running"}
	}
	if err := c.LiveRestart.RestartConversation(ctx, address); err != nil {
		return ReloadContextResult{}, err
	}
	return ReloadContextResult{Address: address}, nil
}

// prepareLiveRestart admits relaunch or reload-context on a live row. The row
// is addressed by its thread (repo scope + exact tag), the dialect both
// operations' dispatch reads, never by the slot path.
func (c *Couch) prepareLiveRestart(ctx context.Context, op, target string, row ActionableThreadSummary, path string, opts LiveRestartOptions) (OperationCall, string, error) {
	if c.LiveRestart == nil {
		return OperationCall{}, "", &SlotOperationError{Code: LiveRestartUnavailable, Detail: "this Couch cannot observe live slots"}
	}
	facts, err := c.LiveRestart.LiveRestartFacts(ctx, op, row, path)
	if err != nil {
		return OperationCall{}, "", err
	}
	d := DecideLiveRestart(op, facts, opts)
	if d.Code != "" {
		detail := target + ": " + d.Detail
		if d.Note != "" {
			detail += " (" + d.Note + ")"
		}
		return OperationCall{}, "", &SlotOperationError{Code: d.Code, Detail: detail}
	}
	args := map[string]string{"repo-scope": row.Address.RepoScope, "tag": string(row.Address.Tag)}
	return OperationCall{Name: op, Args: args, Implicit: true, Context: ctx}, d.Note, nil
}
