package historyarchive

import "errors"

type DriveFolderPhase uint8

const (
	FolderPlanned DriveFolderPhase = iota
	FolderObserving
	FolderMissing
	FolderCreating
	FolderReobserving
	FolderReady
	FolderConflict
	FolderUnknown
	FolderCancelled
)

type DriveFolderEventKind uint8

const (
	FolderBegin DriveFolderEventKind = iota
	FolderAbsent
	FolderUnique
	FolderAmbiguous
	FolderCreate
	FolderCreated
	FolderFailure
	FolderCancel
)

type DriveFolderEffect uint8

const (
	FolderNoEffect DriveFolderEffect = iota
	FolderObserve
	FolderMake
)

// DriveFolderState fences one folder creation attempt and its confirming read.
// pair:m5-concept pure
type DriveFolderState struct {
	phase   DriveFolderPhase
	creates uint8
	id      string
}

// pair:m5-concept pure
type DriveFolderEvent struct {
	Kind DriveFolderEventKind
	ID   string
}

func (s DriveFolderState) Phase() DriveFolderPhase { return s.phase }
func (s DriveFolderState) FolderID() string        { return s.id }

func StepDriveFolder(s DriveFolderState, e DriveFolderEvent) (DriveFolderState, DriveFolderEffect, error) {
	next := s
	if e.Kind == FolderCancel {
		next.phase = FolderCancelled
		return next, FolderNoEffect, nil
	}
	if s.phase == FolderReady || s.phase == FolderConflict || s.phase == FolderUnknown || s.phase == FolderCancelled {
		return s, FolderNoEffect, errors.New("folder attempt is complete")
	}
	if e.Kind == FolderFailure {
		next.phase = FolderUnknown
		return next, FolderNoEffect, nil
	}
	switch {
	case s.phase == FolderPlanned && e.Kind == FolderBegin:
		next.phase = FolderObserving
		return next, FolderObserve, nil
	case (s.phase == FolderObserving || s.phase == FolderReobserving) && e.Kind == FolderUnique && validRemoteIdentity(e.ID):
		next.phase = FolderReady
		next.id = e.ID
		return next, FolderNoEffect, nil
	case (s.phase == FolderObserving || s.phase == FolderReobserving) && e.Kind == FolderAmbiguous:
		next.phase = FolderConflict
		return next, FolderNoEffect, nil
	case s.phase == FolderObserving && e.Kind == FolderAbsent:
		next.phase = FolderMissing
		return next, FolderNoEffect, nil
	case s.phase == FolderReobserving && e.Kind == FolderAbsent:
		next.phase = FolderUnknown
		return next, FolderNoEffect, nil
	case s.phase == FolderMissing && s.creates == 0 && e.Kind == FolderCreate:
		next.phase = FolderCreating
		next.creates = 1
		return next, FolderMake, nil
	case s.phase == FolderCreating && e.Kind == FolderCreated:
		next.phase = FolderReobserving
		return next, FolderObserve, nil
	}
	return s, FolderNoEffect, errors.New("invalid folder state/event")
}
