package couchcore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// detachExitPoll is how often Detach re-observes the client it asked to leave.
// Couch has no wait seam -- Wait belongs to PairLifecycleController -- so the
// bounded wait is a poll, the same shape awaitThreadRegistration already uses.
const detachExitPoll = 10 * time.Millisecond

// detachExitTimeout bounds that wait. A client that has not gone by then is not
// killed; detach fails and the thread stays live.
const detachExitTimeout = 15 * time.Second

// Detach stops a thread's Pair client and leaves its zellij session running.
//
// This is the warm counterpart to park. Park writes a quit intent, tears the
// zellij session down, and records a verified park as the resume authority --
// which kills the agent. Detach kills nothing that matters: the pair client and
// the zellij client it hosts go, the session-watcher and title-poller sidecars
// sharing its process group go with them, and the zellij SERVER session plus the
// agent running inside it survive. Reattaching is a fresh
// `pair resume <tag>` -- with NO layout flag -- onto that surviving session:
// the session already has its layout, and asking for a different one is the
// path that offers to DELETE it (#179).
//
// It deliberately does not reuse handleCleanup, whose own comment says "this
// path is rollback, not graceful actor shutdown": that path escalates to an
// unconditional SIGKILL, and detach is the everyday gesture -- and, once leaving
// couch detaches rather than parks, the gesture applied to every thread on the
// way out. Truncating an agent mid-write is the outcome detach exists to avoid,
// so SIGTERM is the only signal sent and a client that ignores it makes the
// operation FAIL rather than escalate. Nothing was destroyed, so failing is
// safe and needs no recovery mode.
//
// Two proofs are required before any durable write, because a record whose
// incarnation is retired without a surviving session is worse than one left
// occupied: the switcher would offer a reattach that cannot work.
func (c *Couch) Detach(ctx context.Context, address ThreadAddress) (ThreadRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ThreadRecord{}, err
	}
	if err := validateThreadAddress(address); err != nil {
		return ThreadRecord{}, err
	}
	// Clock joins the list because detach now RECORDS when the thread was last
	// active (pair#187). A missing dependency must refuse here, where the
	// message names it, rather than segfault deep in the retry loop -- which is
	// what a nil clock did the moment this precondition was added and not
	// declared.
	if c.Threads == nil || c.Proc == nil || c.Artifacts == nil || c.Clock == nil {
		return ThreadRecord{}, errors.New("detach requires a thread store, process ops, an artifact controller, and a clock")
	}
	sessions, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return ThreadRecord{}, errors.New("detach requires Pair session observation")
	}

	record, err := c.Threads.GetThread(address)
	if err != nil {
		return ThreadRecord{}, err
	}
	if record.Park != nil {
		return ThreadRecord{}, errors.New("cannot detach a thread with an open park transaction")
	}
	if len(record.Incarnations) != 1 || record.Incarnations[0].State != IncarnationLive {
		return ThreadRecord{}, fmt.Errorf("thread %+v has no live incarnation to detach", address)
	}
	incarnation := record.Incarnations[0]
	identity := ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}

	// Observe the session BEFORE signalling: if it is not there now, detaching
	// would leave a thread with no view and nothing to reattach to.
	before, err := sessions.PairSession(address)
	if err != nil {
		return ThreadRecord{}, fmt.Errorf("observe Pair session before detach: %w", err)
	}
	if !before.Present {
		return ThreadRecord{}, fmt.Errorf("thread %+v has no live Pair session to detach from", address)
	}

	if err := c.Proc.SignalGroup(identity.PID, TermSignal); err != nil {
		return ThreadRecord{}, fmt.Errorf("detach %+v: %w", address, err)
	}
	if err := c.awaitExactProcessExit(ctx, identity); err != nil {
		return ThreadRecord{}, err
	}

	// And after: the whole point is that it survived its client.
	detached, err := c.retireDetachedIncarnation(ctx, address, identity, c.Clock.Now())
	if err != nil {
		return ThreadRecord{}, err
	}
	return detached, nil
}

// retireDetachedIncarnation is the shared half of "the client is gone, the
// session is not": prove the session present, then retire that exact
// incarnation so the thread reads detached again.
//
// Detach owns this rule, and pair#230's warm-reattach cleanup needs exactly the
// same one -- a reattach that failed after its helper was acknowledged has to
// hand the thread back to the session it borrowed, under the same proofs. One
// copy, because two would drift on the retry policy first.
//
// Its messages are operation-neutral: start cleanup reaches it too, and an
// operator who never pressed detach should not be told a detach failed.
//
// detachedAt is the caller's, read ONCE before any attempt: reading the clock
// per attempt would make the recorded activity time a function of how much
// revision contention there was, which measures the store rather than the
// thread. A cleanup caller passes the thread's EXISTING LastActiveAt, since a
// failed reattach is not activity.
func (c *Couch) retireDetachedIncarnation(
	ctx context.Context, address ThreadAddress, identity ProcessIdentity, detachedAt time.Time,
) (ThreadRecord, error) {
	sessions, ok := c.Artifacts.(PairSessionIO)
	if !ok {
		return ThreadRecord{}, errors.New("retiring a detached incarnation requires Pair session observation")
	}
	after, err := sessions.PairSession(address)
	if err != nil {
		return ThreadRecord{}, fmt.Errorf("observe Pair session before retiring its incarnation: %w", err)
	}
	if !after.Present {
		return ThreadRecord{}, fmt.Errorf("thread %+v has no live Pair session to retire onto", address)
	}

	// Retry on a revision conflict rather than giving up. The revision was read
	// BEFORE a SIGTERM, a bounded wait and two zellij observations, so anything
	// touching the record in that window -- a metadata edit, a refresh-driven
	// write -- would otherwise abandon a thread whose client is already dead,
	// leaving exactly the stale-IncarnationLive state pair#171 describes,
	// reached from an ordinary failure path rather than a crash.
	//
	// The loop shape is MarkIncarnationUnknown's: re-read, re-attempt, and let
	// RetireIncarnation's own preconditions refuse if the record genuinely
	// stopped being retirable.
	//
	// Bounded, and it checks the context. The loop retries a revision conflict,
	// which is a contended-store condition and not one that resolves by trying
	// forever: an unbounded retry against a store that keeps losing the race
	// spins a detach that has already SIGTERMed its client, with no way for the
	// operator to interrupt it. The cap is generous because a real conflict
	// clears in one attempt. A CLEANUP caller passes a fresh context.Background
	// rather than its own, because cleanup must complete precisely when the
	// thing that failed was a cancellation.
	const maxRetireAttempts = 32
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return ThreadRecord{}, fmt.Errorf("retire detached incarnation for %+v: %w", address, err)
		}
		if attempt > maxRetireAttempts {
			return ThreadRecord{}, fmt.Errorf(
				"retire detached incarnation for %+v: gave up after %d revision conflicts", address, maxRetireAttempts)
		}
		current, err := c.Threads.GetThread(address)
		if err != nil {
			return ThreadRecord{}, fmt.Errorf("retire detached incarnation for %+v: %w", address, err)
		}
		detached, err := c.Threads.RetireIncarnation(address, current.Revision, identity, detachedAt)
		var conflict *ThreadRevisionError
		if errors.As(err, &conflict) {
			continue
		}
		if err != nil {
			return ThreadRecord{}, fmt.Errorf("retire detached incarnation for %+v: %w", address, err)
		}
		return detached, nil
	}
}

// awaitExactProcessExit waits for one exact process to be gone, bounded.
//
// Unknown liveness is not exit: a process couch cannot observe has not been
// proved to have left, and treating it as gone would retire an incarnation that
// might still be running.
func (c *Couch) awaitExactProcessExit(ctx context.Context, identity ProcessIdentity) error {
	sleep := c.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	deadline := detachExitTimeout
	for waited := time.Duration(0); ; waited += detachExitPoll {
		switch observeExactProcess(c.Proc, identity) {
		case Dead:
			return nil
		case Unknown:
			return fmt.Errorf("cannot observe whether pid %d exited", identity.PID)
		}
		if waited >= deadline {
			return fmt.Errorf("pid %d did not exit within %s of SIGTERM; thread left running", identity.PID, deadline)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		sleep(detachExitPoll)
	}
}

// ArchiveThread is the operator's delete, complete: the session it names stops,
// then the record leaves the working set.
//
// The store's ArchiveThread is bookkeeping only, and bookkeeping alone leaves a
// running agent nothing tracks -- exactly the forgotten thread couch exists to
// prevent. Park cannot do the stopping: it drives a transaction through
// PairLifecycle and needs a live incarnation, which the debris this action is
// FOR does not have. Quiesce works one layer down (`zellij delete-session
// --force`, polled until the session is verifiably gone), so it reaches the
// case park cannot.
//
// Quiesce runs FIRST and a failure refuses the archive. The other order would
// produce the precise state this exists to remove: a record in the archive with
// a live session behind it. Quiesce is idempotent -- it returns nil when there
// is no session bound to the address at all, which is the common debris case --
// so a refused archive is safe to retry.
func (c *Couch) ArchiveThread(ctx context.Context, address ThreadAddress) (ArchiveResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateThreadAddress(address); err != nil {
		return ArchiveResult{}, err
	}
	if c == nil || c.Threads == nil || c.Artifacts == nil {
		return ArchiveResult{}, errors.New("archive requires a thread store and an artifact controller")
	}
	if err := ctx.Err(); err != nil {
		return ArchiveResult{}, err
	}
	// Read for the RESULT, not as a precondition. An undecodable record is
	// exactly what the operator most wants gone, so failing here would leave a
	// row that can be neither used nor removed -- the shape this action exists
	// to clear. What refuses a thread couch is hosting is the admission rule
	// below, and it needs the decoded record to classify: an unreadable record
	// is unprovable, so it is moved rather than acted on.
	record, readErr := c.Threads.GetThread(address)

	// The guard runs BEFORE any effect. It used to run after Quiesce, inside
	// the store, so a park-in-flight thread had its session killed and was then
	// refused -- the agent dead, the record still listed.
	if readErr == nil {
		// ADMISSION FIRST, through the same door the switcher's offer is
		// compared against (#256 M3): ArchivableState over the classification.
		//
		// `archivableRecord` used to answer this, and it asked the RECORD. Since
		// M1 the record cannot answer it: couch's own hosting is the live proof,
		// so a thread couch is hosting may carry no incarnation at all -- and
		// that row passed the occupancy rule and went straight to Quiesce, which
		// kills the session its agent is running in. The record keeps a
		// narrower, record-only guard below; this is the one that knows what the
		// thread IS.
		//
		// It is read-only, so it costs nothing to be wrong about and precedes
		// every write. Its cost is one evidence round -- one host-wide
		// `list-sessions`, and the ledger only for this address, because the
		// `ask` predicate narrows it -- on an operator keypress, the same bill
		// PrepareAgentSwitch pays for the same reason.
		state, reason, classifyErr := c.classifyForAction(ctx, address)
		if classifyErr != nil {
			return ArchiveResult{}, fmt.Errorf("archive %s: its state could not be classified; inspect and retry: %w", address.Tag, classifyErr)
		}
		if !ArchivableState(state, reason) {
			return ArchiveResult{}, fmt.Errorf("archive %s: %s", address.Tag, archiveRefusal(state, reason))
		}
		// ORDER, in two steps, and the order is the guard (#256 M2).
		//
		// 1. Ask the session. It is read-only, and it is the one refusal that
		//    is nobody's fault and may change on its own -- an unanswerable
		//    zellij, a client that is still attached. Refusing here has written
		//    nothing, which is what makes a retry honest.
		// 2. Then clear the debris, on the same terms resume does.
		//
		// The other guards below -- DecideRecovery's park and incarnation-shape
		// gates, archivableRecord's unfinished-transaction rule -- are
		// structural: they say what the recovery may safely act on, and each
		// protects a real precondition downstream, so none of them is removed.
		// They stopped being the OPERATOR's wall instead, because by the time
		// they run, a park whose owner is provably gone and a start claim whose
		// couch is provably gone are no longer there.
		first, observeErr := c.observeRecovery(ctx, record)
		switch {
		case observeErr != nil && !errors.Is(observeErr, ErrPairSessionBindingAbsent):
			// An unanswerable session is not an absent one, and nothing has
			// been written yet, so the honest move is to say so and stop.
			return ArchiveResult{}, fmt.Errorf("archive %s: %w", address.Tag, observeErr)
		case observeErr == nil:
			if refusal := RecoverySessionRefusal(first); refusal != "" {
				return ArchiveResult{}, fmt.Errorf("archive %s: %s", address.Tag, refusal)
			}
		}
		// clearLifecycleDebris refuses, with a code, whenever it cannot prove
		// the debris orphaned -- and it screens every precondition before its
		// first write, so a refusal there has changed nothing either. Quiesce,
		// the irreversible act on the world, is still far below.
		if cleared, clearErr := c.clearLifecycleDebris(record); clearErr != nil {
			if errors.Is(clearErr, ErrThreadRolledBack) {
				// The record carried nothing but an unfinished start, so
				// rolling it back removed the row -- which is the whole of what
				// archive promises the operator.
				return ArchiveResult{Record: record}, nil
			}
			return ArchiveResult{}, fmt.Errorf("archive %s: %w", address.Tag, clearErr)
		} else if cleared != nil {
			record = *cleared
		}
		reconciled, evidence, err := c.reconcileRecoveryHelper(ctx, address)
		if err != nil {
			// A pre-session launch failure can leave a readable empty record
			// with no binding. This is a non-signalling bookkeeping escape,
			// not proof of session absence for recovery or a retained request.
			if !errors.Is(err, ErrPairSessionBindingAbsent) || len(record.Incarnations) != 0 || record.Park != nil || record.Continuation != nil {
				return ArchiveResult{}, err
			}
			// Recheck the exact index before the revision-guarded move. A
			// newly published binding requires normal ownership checks.
			if _, err := c.recoverySession(ctx, address); !errors.Is(err, ErrPairSessionBindingAbsent) {
				if err != nil {
					return ArchiveResult{}, err
				}
				return ArchiveResult{}, fmt.Errorf("archive %s: session binding appeared before archive", address.Tag)
			}
			if err := ctx.Err(); err != nil {
				return ArchiveResult{}, err
			}
			if err := c.Threads.ArchiveThreadExpected(address, record.Revision); err != nil {
				return ArchiveResult{}, err
			}
			return ArchiveResult{Record: record}, nil
		}
		record = reconciled
		decision := DecideRecovery(evidence)
		if !decision.Archive {
			return ArchiveResult{}, fmt.Errorf("archive %s: %s", address.Tag, decision.Diagnosis)
		}
		if err := archivableRecord(record); err != nil {
			return ArchiveResult{}, err
		}
		if err := c.archiveContinuationVacant(record, evidence); err != nil {
			return ArchiveResult{}, err
		}
		latest, err := c.observeRecovery(ctx, record)
		if err != nil {
			return ArchiveResult{}, err
		}
		// Compared against BOTH earlier looks, not just the reconciler's. The
		// window that matters runs from the FIRST observation -- a session that
		// appears after it and settles before the reconciler's would otherwise
		// look stable across the last two and be quiesced (#256 M2).
		if latest.Session != evidence.Session || latest.Presence != evidence.Presence ||
			(observeErr == nil && (latest.Session != first.Session || latest.Presence != first.Presence)) ||
			!DecideRecovery(latest).Archive {
			return ArchiveResult{}, fmt.Errorf("archive %s: helper or session ownership changed before stop", address.Tag)
		}
		if err := c.archiveContinuationVacant(record, latest); err != nil {
			return ArchiveResult{}, err
		}
		if err := ctx.Err(); err != nil {
			return ArchiveResult{}, err
		}
		if err := c.Artifacts.Quiesce(address); err != nil {
			return ArchiveResult{}, fmt.Errorf("archive %s: its session could not be stopped: %w", address.Tag, err)
		}
		// Quiesce may cross an external failure/retry boundary. Refuse if a
		// session appeared again or durable request state changed meanwhile.
		presence, err := c.observeSessionPresenceContext(ctx, address)
		if err != nil {
			return ArchiveResult{}, err
		}
		if presence != PresenceAbsent {
			return ArchiveResult{}, fmt.Errorf("archive %s: session absence is no longer proved", address.Tag)
		}
		if err := c.Threads.ArchiveThreadExpected(address, record.Revision); err != nil {
			return ArchiveResult{}, err
		}
		return ArchiveResult{Record: record}, nil
	} else {
		// Unreadable: the operator can still remove the row -- that escape is
		// what keeps a corrupt record from locking its repository -- but couch
		// does NOT stop a session it cannot identify. Classifying the thread --
		// which is what says couch is not hosting it -- needs a decoded record,
		// so quiescing here would kill an agent on the strength of a record we
		// just failed to read. Unknown stays conservative: the record is filed, the session is
		// left alone, and the caller is told.
		record = ThreadRecord{Address: address}
	}
	if err := c.Threads.ArchiveThreadExpected(address, 0); err != nil {
		return ArchiveResult{}, err
	}
	return ArchiveResult{Record: record, SessionNotStopped: readErr != nil}, nil
}

// archiveRefusal says what to DO about a row archive will not take. It is called
// only for a classification ArchivableState refuses, and only for one
// ClassifyThread can produce -- so its arms are `live`, `busy` and
// `unusable/unknown`, pinned by TestArchiveRefusalCoversEveryRefusedClassification.
//
// `archived` gets no arm of its own on purpose. The classifying projection never
// emits it (TestProjectionNeverProducesArchived), and a specially worded message
// for a state production cannot reach is the drift that once gave `invalid` a
// label, an Enter notice and an archive exit no real store could produce. The
// default says the true thing for it.
//
// Guidance lives here, at the consumer, rather than as a field every producer
// carries (#256 M1, round 3): the classification says what the thread IS, and
// archive is the only caller that needs to say what to do about it instead.
func archiveRefusal(state ActionableThreadState, reason ThreadReason) string {
	switch state {
	case ThreadLive:
		return "it is live -- couch is hosting its agent; detach or park it first"
	case ThreadBusy:
		return "it is busy -- a start is in flight; let it finish or be released"
	case ThreadUnusable:
		return "its state is unresolved (" + reason.Label() + "); nothing is known well enough to stop it, so retry"
	}
	return "it is " + string(state) + " and cannot be archived"
}

// A retained request may name a live source/target even after its incarnation
// was retired. Archive preserves the request only when those actors are proved
// absent; a checkpoint is never permission to stop an unfinished conversation.
func (c *Couch) archiveContinuationVacant(record ThreadRecord, evidence RecoveryEvidence) error {
	return withContinuationExits(record, c.checkArchiveContinuationVacant(record, evidence))
}

func (c *Couch) checkArchiveContinuationVacant(record ThreadRecord, evidence RecoveryEvidence) error {
	request := record.Continuation
	if request == nil || request.Phase == checkpoint.Complete {
		return nil
	}
	if evidence.Presence != PresenceAbsent {
		return fmt.Errorf("archive %s: continuation source or target session is still occupied", record.Address.Tag)
	}
	identities := []ProcessIdentity{{PID: request.Source.Helper.PID, Identity: request.Source.Helper.Identity}}
	if request.Target != nil {
		identities = append(identities, ProcessIdentity{PID: request.Target.PID, Identity: request.Target.Identity})
	}
	for _, identity := range identities {
		if identity.PID == 0 && identity.Identity == "" {
			continue
		}
		if c.Proc == nil || observeExactProcess(c.Proc, identity) != Dead {
			return fmt.Errorf("archive %s: continuation source or target helper is not proved dead", record.Address.Tag)
		}
	}
	return nil
}

// ArchiveResult is what archiving did, including what it deliberately did NOT
// do.
//
// The warning used to travel as an error, which made every consumer read a
// completed archive as a failed one: the CLI exited 1 while the row was gone,
// and the switcher took its failure branch -- a red notice, the confirmation
// frame left open, no projection refresh -- so the recovery path the start
// refusal names appeared to fail. An operation that mutated is a success; what
// it could not do belongs in the result.
type ArchiveResult struct {
	Record ThreadRecord
	// SessionNotStopped means couch could not read the record and so left its
	// session alone rather than stopping something it could not identify.
	SessionNotStopped bool
}

// Warning is the operator-facing note, empty when there is nothing to say.
func (r ArchiveResult) Warning() string {
	if !r.SessionNotStopped {
		return ""
	}
	return fmt.Sprintf(
		"couch could not read %s, so it archived the record without stopping its session; "+
			"check `zellij list-sessions` if an agent is still running", r.Record.Address.Tag)
}
