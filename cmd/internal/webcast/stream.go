package webcast

// wireState is what a viewer is told about the cast. While not live it says
// only that: nothing the operator does with the cast paused, or before going
// live, reaches a viewer.
type wireState struct {
	Live  bool   `json:"live"`
	URL   string `json:"url,omitempty"`
	Title string `json:"title,omitempty"`
}

func viewerState(s State) wireState {
	if s.Phase != PhaseLive {
		return wireState{}
	}
	return wireState{Live: true, URL: s.Tab.URL, Title: s.Tab.Title}
}

// wireEnd tells a viewer the cast is over. Its reason is a fixed string.
type wireEnd struct {
	Reason string `json:"reason"`
}
