package couchcore

import (
	"context"
	"errors"
	"fmt"
	"time"
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
// `pair resume <tag> --layout2` onto that surviving session.
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
	if c.Threads == nil || c.Proc == nil || c.Artifacts == nil {
		return ThreadRecord{}, errors.New("detach requires a thread store, process ops, and an artifact controller")
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
	after, err := sessions.PairSession(address)
	if err != nil {
		return ThreadRecord{}, fmt.Errorf("observe Pair session after detach: %w", err)
	}
	if !after.Present {
		return ThreadRecord{}, fmt.Errorf("thread %+v lost its Pair session during detach", address)
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
	for {
		current, err := c.Threads.GetThread(address)
		if err != nil {
			return ThreadRecord{}, fmt.Errorf("retire detached incarnation for %+v: %w", address, err)
		}
		detached, err := c.Threads.RetireIncarnation(address, current.Revision, identity)
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
	// to clear. The store's own guard is what refuses an occupied thread, and
	// it applies the same rule to a record it cannot read: unreadable means
	// unprovable, so it is moved rather than acted on.
	record, readErr := c.Threads.GetThread(address)

	// The guard runs BEFORE any effect. It used to run after Quiesce, inside
	// the store, so a park-in-flight thread had its session killed and was then
	// refused -- the agent dead, the record still listed.
	if readErr == nil {
		if err := archivableRecord(record); err != nil {
			return ArchiveResult{}, err
		}
		if err := c.Artifacts.Quiesce(address); err != nil {
			return ArchiveResult{}, fmt.Errorf("archive %s: its session could not be stopped: %w", address.Tag, err)
		}
	} else {
		// Unreadable: the operator can still remove the row -- that escape is
		// what keeps a corrupt record from locking its repository -- but couch
		// does NOT stop a session it cannot identify. `archivableRecord` needs a
		// decoded record to prove the thread is not live, so quiescing here
		// would kill an agent on the strength of a record we just failed to
		// read. Unknown stays conservative: the record is filed, the session is
		// left alone, and the caller is told.
		record = ThreadRecord{Address: address}
	}
	if err := c.Threads.ArchiveThread(address); err != nil {
		return ArchiveResult{}, err
	}
	return ArchiveResult{Record: record, SessionNotStopped: readErr != nil}, nil
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
