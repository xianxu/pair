package couchtty

// CopyFeedbackKind describes the outcome of an explicit clipboard attempt.
type CopyFeedbackKind uint8

const (
	CopyFeedbackNone CopyFeedbackKind = iota
	CopyFeedbackEmpty
	CopyFeedbackSuccess
	CopyFeedbackUnconfirmed
	CopyFeedbackFailed
)

// CopyFeedback is separate from operation notices. Only a failed or unconfirmed
// attempt retains its original input for an explicit retry; it is never logged.
type CopyFeedback struct {
	Kind    CopyFeedbackKind
	Text    string
	payload string
}

func ReduceCopyFeedback(state CopyFeedback, event MenuEvent) (CopyFeedback, bool) {
	if event.Kind == MenuEventCopyResult {
		next := event.CopyFeedback
		if !next.held() {
			next.payload = ""
		}
		return next, true
	}
	if state.Kind == CopyFeedbackNone {
		return state, false
	}
	if event.Kind == MenuEventKey && event.Key.Kind == KeyEscape {
		return CopyFeedback{}, true
	}
	if !state.held() {
		switch event.Kind {
		case MenuEventKey, MenuEventMouseSwitch, MenuEventMouseRowClick, MenuEventParkHotkey:
			return CopyFeedback{}, false
		}
	}
	return state, false
}

func (feedback CopyFeedback) held() bool {
	return feedback.Kind == CopyFeedbackFailed || feedback.Kind == CopyFeedbackUnconfirmed
}

// Separate lines keep the recovery keys discoverable even when the diagnostic
// itself is clipped at the minimum terminal width.
func renderedMenuCopyLines(state MenuState) []string {
	feedback := state.CopyFeedback
	if feedback.Kind == CopyFeedbackNone {
		return nil
	}
	lines := []string{feedback.Text}
	if feedback.held() {
		lines = append(lines, "Ctrl+Y retry · Esc dismiss")
	}
	return lines
}
