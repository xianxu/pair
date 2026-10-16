package titlepoller

type titlePhase uint8

const (
	titleAwaiting titlePhase = iota
	titleActive
	titleExited
)

type TitleEvent uint8

const (
	TitleLeaseReady TitleEvent = iota
	TitleOwnerEOF
	TitleSignalled
	TitlePollStopped
)

type TitleEffects uint8

const (
	TitleStartPoll TitleEffects = 1 << iota
	TitleExit
)

// TitleLifetime owns poll admission. EOF is terminal even before initial start.
type TitleLifetime struct{ phase titlePhase }

func (s TitleLifetime) stopped() bool { return s.phase == titleExited }
func AdvanceTitleLifetime(s TitleLifetime, event TitleEvent) (TitleLifetime, TitleEffects) {
	if s.phase == titleExited {
		return s, 0
	}
	switch event {
	case TitleLeaseReady:
		if s.phase == titleAwaiting {
			return TitleLifetime{titleActive}, TitleStartPoll
		}
	case TitleOwnerEOF, TitleSignalled, TitlePollStopped:
		return TitleLifetime{titleExited}, TitleExit
	}
	return s, 0
}
