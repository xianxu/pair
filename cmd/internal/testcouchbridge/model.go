package testcouchbridge

type Phase uint8

const (
	New Phase = iota
	Preparing
	Starting
	Checking
	Publishing
	Held
	Stopping
	Stopped
	Retained
)

type EventKind uint8

const (
	Begin EventKind = iota
	CandidateReady
	FleetReady
	EndpointReady
	ReadinessWritten
	EOF
	TERM
	INT
	Failed
	Deadline
	ProducerDied
	CleanupClean
	CleanupUncertain
)

type Effect uint8

const (
	None Effect = iota
	Prepare
	Start
	Check
	Publish
	Stop
)

type State struct {
	Phase   Phase
	Failure string
}
type Event struct {
	Kind    EventKind
	Failure string
}

// Reduce is the only phase writer. A stop owns cleanup through its terminal
// result; late operation completions cannot publish readiness or restart work.
func Reduce(s State, e Event) (State, Effect) {
	if s.Phase == Stopped || s.Phase == Retained {
		return s, None
	}
	if s.Phase == Stopping {
		switch e.Kind {
		case CleanupClean:
			s.Phase = Stopped
		case CleanupUncertain:
			s.Phase = Retained
			if s.Failure == "" {
				s.Failure = e.Failure
			}
		}
		return s, None
	}
	switch e.Kind {
	case EOF, TERM, INT, Failed, Deadline, ProducerDied:
		s.Phase = Stopping
		s.Failure = e.Failure
		return s, Stop
	}
	switch {
	case s.Phase == New && e.Kind == Begin:
		s.Phase = Preparing
		return s, Prepare
	case s.Phase == Preparing && e.Kind == CandidateReady:
		s.Phase = Starting
		return s, Start
	case s.Phase == Starting && e.Kind == FleetReady:
		s.Phase = Checking
		return s, Check
	case s.Phase == Checking && e.Kind == EndpointReady:
		s.Phase = Publishing
		return s, Publish
	case s.Phase == Publishing && e.Kind == ReadinessWritten:
		s.Phase = Held
	}
	return s, None
}
