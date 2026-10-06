package couchcore

// RebootRecord is what reboot found where the old conversation's record should
// be, after prepareRetirement has had its say.
type RebootRecord uint8

const (
	// RebootRecordNone is no record at all: an empty slot, or a :0 tag that
	// already left (a retry after a reboot whose launch failed).
	RebootRecordNone RebootRecord = iota + 1
	// RebootRecordReadable is a decoded record with a known path.
	RebootRecordReadable
	// RebootRecordUnreadable is a record couch cannot decode. Its bytes move
	// as-is; a :0 one has no readable path to start in.
	RebootRecordUnreadable
	// RebootRecordRolledBack is prepareRetirement's RolledBack: the record
	// held only an unfinished start and is already gone.
	RebootRecordRolledBack
)

// RebootFacts is everything DecideReboot reads. The IO shell (Couch.Reboot)
// gathers them; this file only decides.
type RebootFacts struct {
	Slot             bool
	Record           RebootRecord
	DirectoryPresent bool
}

// RebootPlan is what reboot does with the facts.
type RebootPlan uint8

const (
	RebootRefuse RebootPlan = iota + 1
	// RebootArchiveOnly retires the record and starts nothing: there is no
	// directory (or no readable path) to start a fresh agent in.
	RebootArchiveOnly
	// RebootArchiveAndStart retires the record and commits a fresh claimed
	// record in the same store journal.
	RebootArchiveAndStart
	// RebootStartOnly starts fresh where there is no record left to retire.
	RebootStartOnly
)

// RebootDirectoryMissing is a :1+ slot row's next step when its directory is
// gone: add slot reuses that leftover number and reconciles it back
// (pair#387). The switcher shows it; reboot itself reconciles the slot before
// it decides, so a slot reboot reaches this branch only if reconcile could not
// restore the directory, and reconcile's own refusal names the cause first.
const RebootDirectoryMissing = "directory missing — add slot recreates it"

// RebootCheckoutMissing is the same for a :0, whose checkout is the primary
// itself: add slot makes :1+ slots, never a primary, and a :0 row does not
// offer it there. Reboot archives the record; the checkout coming back is
// what lets a fresh agent start at that path again (pair#363 M2 review).
const RebootCheckoutMissing = "checkout missing — reboot archives this record; restore the checkout to start here again"

// rebootMissing is the no-directory reason for the kind reboot is acting on.
func rebootMissing(slot bool) string {
	if slot {
		return RebootDirectoryMissing
	}
	return RebootCheckoutMissing
}

const (
	rebootUnreadablePrimary = "record unreadable — start couch in its directory for a fresh agent"
	rebootNothing           = "nothing to reboot"
)

// DecideReboot is reboot's whole policy as one pure decision. The reason is
// non-empty exactly when the plan starts nothing (archive-only or refuse), so
// every outcome that leaves the operator without a live agent says why.
//
// The asymmetry between :0 and :1+ is where the path comes from. A slot's
// directory is known from the slot itself, so even an unreadable slot record
// can be archived (its bytes go to recovery/) and a fresh agent started. A :0
// record IS the path, so an unreadable one leaves nowhere to start.
func DecideReboot(f RebootFacts) (RebootPlan, string) {
	switch f.Record {
	case RebootRecordReadable:
		if !f.DirectoryPresent {
			return RebootArchiveOnly, rebootMissing(f.Slot)
		}
		return RebootArchiveAndStart, ""
	case RebootRecordUnreadable:
		if !f.Slot {
			return RebootArchiveOnly, rebootUnreadablePrimary
		}
		if !f.DirectoryPresent {
			return RebootRefuse, rebootMissing(f.Slot)
		}
		return RebootArchiveAndStart, ""
	case RebootRecordRolledBack:
		if !f.DirectoryPresent {
			return RebootRefuse, rebootMissing(f.Slot)
		}
		return RebootStartOnly, ""
	case RebootRecordNone:
		if !f.Slot {
			return RebootRefuse, rebootNothing
		}
		if !f.DirectoryPresent {
			return RebootRefuse, rebootMissing(f.Slot)
		}
		return RebootStartOnly, ""
	}
	return RebootRefuse, rebootNothing
}
