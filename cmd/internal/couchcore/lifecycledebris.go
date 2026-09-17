package couchcore

import "errors"

// ErrThreadRolledBack reports that clearing the debris removed the record
// itself, because the only thing it carried was a start that never completed.
//
// It is a FACT, not a verdict: rolling a never-started claim back deletes a
// husk with no launch profile, no park and no metadata (see
// ThreadStore.DeleteStart's final branch). Resume reads that as a refusal --
// there is nothing left to resume -- and archive reads it as success, because
// the row is gone, which is the whole of what archive promises. Guidance
// belongs at the consumer; the producer states what happened (#256 M1, round 3).
var ErrThreadRolledBack = errors.New("the thread was rolled back: it carried nothing but an unfinished start")

// clearLifecycleDebris clears the bookkeeping that would otherwise make a
// recoverable thread neither startable nor archivable, and refuses -- with a
// code -- whenever it cannot.
//
// It serves BOTH callers that need a record the store's structural guards will
// accept: re-adoption before a start (resume.go) and ArchiveThread. Those guards
// -- one incarnation at a time, no open park, no claim outstanding -- are
// structural invariants, not lifecycle opinions, so the rule for clearing them
// is written once and here.
//
// It is TOTAL over the record shapes `validateLifecycle` accepts. That totality
// is the rule, not a property of the shapes anyone has enumerated: round 2 wrote
// "every guard refusing on record.Incarnations or record.Park" as four SITES,
// and the shapes it had not thought of -- a foreign-owned park, then an open
// park with ZERO incarnations -- each reached the store and wedged `couch` in
// the whole tree with an uncoded refusal. Both are the same
// `replacementUnknown` escape in threadrecord/lifecycle.go, read at different
// incarnation counts.
//
// ARCH-SECURE: a record written by another version is untrusted input, so what
// the validator ACCEPTS is the domain, not what this codebase happens to write.
//
// Order is load-bearing in two directions:
//
//   - Every precondition is screened BEFORE any write. AbandonPark's tombstone
//     is permanent, so discovering afterwards that the retirement cannot proceed
//     destroys the park and leaves a thread that can be neither resumed nor
//     archived.
//   - Each write is authorized by a probe of the entity IT acts on. The park's
//     owner, the start's owner and the incarnation's process are three different
//     processes, and any of them can outlive the others.
func (c *Couch) clearLifecycleDebris(thread ThreadRecord) (*ThreadRecord, error) {
	if c == nil || c.Proc == nil || c.Threads == nil {
		return nil, nil
	}
	if thread.Park == nil && len(thread.Incarnations) == 0 {
		return nil, nil // nothing in the way
	}

	// ---- screen ----
	var incarnation *ThreadIncarnation
	var rollback string
	switch len(thread.Incarnations) {
	case 0:
	case 1:
		candidate := thread.Incarnations[0]
		if candidate.Start != nil {
			// A start is in flight while the couch that claimed it is alive or
			// unprovable -- the same rule the classifier applies to decide
			// `busy`, and the same direction ReconcileStart fails. A claim
			// whose owner is provably dead has no driver, and leaving it is
			// what made the row unarchivable as well as unstartable (#256 M2).
			//
			// The owner is NOT the helper. It is the couch process that began
			// the transaction, and probing the helper here would authorize a
			// rollback against a process nobody looked at.
			owner := ProcessIdentity{PID: candidate.Start.OwnerPID, Identity: candidate.Start.OwnerIdentity}
			if observeExactProcess(c.Proc, owner) != Dead {
				return nil, refuseResume(ResumeStarting,
					"a start claim is outstanding and the couch that made it could not be proved gone; let it finish")
			}
			if candidate.PID > 0 && observeExactProcess(c.Proc, ProcessIdentity{PID: candidate.PID, Identity: candidate.Identity}) != Dead {
				return nil, refuseResume(ResumeStarting,
					"the process this start forked could not be proved dead, so the claim must not be rolled back")
			}
			rollback = candidate.Start.Nonce
		} else {
			if candidate.PID <= 0 || candidate.Identity == "" {
				return nil, refuseResume(ResumeUnknown,
					"the recorded incarnation names no process, so it cannot be proved dead or retired")
			}
			if observeExactProcess(c.Proc, ProcessIdentity{PID: candidate.PID, Identity: candidate.Identity}) != Dead {
				return nil, refuseResume(ResumeUnknown,
					"recorded process could not be proved dead, so its incarnation cannot be retired; inspect it before acting on the thread")
			}
			if candidate.State != IncarnationLive {
				// RetireIncarnation refuses anything else, and finding that out
				// after abandoning the park is what destroys the park.
				return nil, refuseResume(ResumeUnknown,
					"recorded incarnation is "+string(candidate.State)+"; only a live one can be retired")
			}
			incarnation = &candidate
		}
	default:
		return nil, refuseResume(ResumeUnknown,
			"thread carries more than one recorded incarnation; ownership must be resolved before it can be resumed")
	}
	if thread.Park != nil {
		// The park's OWN owner, which validateLifecycle does not require to be
		// one of the incarnations: zero matches are permitted when the phase is
		// `unknown` and the transaction carries a replacement_incarnation
		// failure. Probing the incarnation here would write a permanent
		// tombstone about a process nothing looked at, one that may be alive and
		// mid-park -- its FinalizePark would then never match again. Pinned by
		// TestForeignOwnedParkIsRepresentableAndRefused.
		owner := ProcessIdentity{PID: thread.Park.Identity.PID, Identity: thread.Park.Identity.ProcessIdentity}
		if observeExactProcess(c.Proc, owner) != Dead {
			return nil, refuseResume(ResumeParking,
				"the park transaction's own owner could not be proved dead, so its record must not be abandoned")
		}
	}

	// ---- write ----
	//
	// Independent writes, not one transaction. Each is independently correct --
	// a park whose owner is dead, a claim whose claimant is dead, an incarnation
	// whose process is dead -- so a crash between them leaves a record the next
	// attempt repeats safely.
	if thread.Park != nil {
		abandoned, err := c.Threads.AbandonPark(thread.Address, thread.Revision, thread.Park.Identity)
		if err != nil {
			return nil, refuseResume(ResumeParking, "orphaned park could not be abandoned: "+err.Error())
		}
		thread = abandoned
	}
	if rollback != "" {
		if err := c.Threads.DeleteStart(thread.Address, thread.Revision, rollback); err != nil {
			return nil, refuseResume(ResumeStarting, "driverless start claim could not be rolled back: "+err.Error())
		}
		rolled, err := c.Threads.GetThread(thread.Address)
		if errors.Is(err, ErrThreadNotFound) {
			return nil, ErrThreadRolledBack
		}
		if err != nil {
			return nil, refuseResume(ResumeUnknown, "the rolled-back thread could not be re-read: "+err.Error())
		}
		return &rolled, nil
	}
	if incarnation == nil {
		return &thread, nil
	}
	retired, err := c.Threads.RetireIncarnation(thread.Address, thread.Revision,
		ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}, thread.LastActiveAt)
	if err != nil {
		return nil, refuseResume(ResumeUnknown, "stale incarnation could not be retired: "+err.Error())
	}
	return &retired, nil
}
