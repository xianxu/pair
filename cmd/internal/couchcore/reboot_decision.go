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

// RebootDirectoryMissing is the operator's next step when reboot has no
// directory to start in: archive is all reboot can do, and add slot is what
// brings a slot directory back (pair#387 repairs the leftover registration).
const RebootDirectoryMissing = "directory missing — add slot recreates it"

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
			return RebootArchiveOnly, RebootDirectoryMissing
		}
		return RebootArchiveAndStart, ""
	case RebootRecordUnreadable:
		if !f.Slot {
			return RebootArchiveOnly, rebootUnreadablePrimary
		}
		if !f.DirectoryPresent {
			return RebootRefuse, RebootDirectoryMissing
		}
		return RebootArchiveAndStart, ""
	case RebootRecordRolledBack:
		if !f.DirectoryPresent {
			return RebootRefuse, RebootDirectoryMissing
		}
		return RebootStartOnly, ""
	case RebootRecordNone:
		if !f.Slot {
			return RebootRefuse, rebootNothing
		}
		if !f.DirectoryPresent {
			return RebootRefuse, RebootDirectoryMissing
		}
		return RebootStartOnly, ""
	}
	return RebootRefuse, rebootNothing
}
