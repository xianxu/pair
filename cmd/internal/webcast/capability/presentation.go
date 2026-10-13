package capability

// Presentation reduces internal phases to visible states and one operator
// action. A reason can accompany this action, but cannot replace it.
type Presentation struct {
	State   string `json:"state"`
	Action  string `json:"action"`
	Enabled bool   `json:"enabled"`
}

func Project(s CapabilityState) Presentation {
	if s.valid() {
		switch s.Phase {
		case Off:
			return Presentation{State: "off", Action: "enable"}
		case On:
			return Presentation{State: "on", Action: "disable", Enabled: true}
		case Ended:
			return Presentation{State: "ended", Action: "start-cast"}
		case Disabling:
			switch s.Outcome {
			case Running:
				return Presentation{State: "turning-off", Action: "wait"}
			case Retryable:
				return Presentation{State: "retry", Action: "retry"}
			}
		}
	}
	return Presentation{State: "broken", Action: "end-cast"}
}
