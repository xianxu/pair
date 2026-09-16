package wrapcmd

import (
	"bytes"
	"strings"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

const notificationRewriteMaxPending = 64 << 10

type RewriteEvent struct {
	Passthrough  []byte
	Notification *notifyosc.Notification
	Observation  *TurnObservation
}
type RewriteResult struct {
	Passthrough   []byte
	Notifications []notifyosc.Notification
	Observations  []TurnObservation
	Events        []RewriteEvent
}

func (r *RewriteResult) pass(data []byte) {
	if len(data) == 0 {
		return
	}
	r.Passthrough = append(r.Passthrough, data...)
	if len(r.Events) > 0 && r.Events[len(r.Events)-1].Passthrough != nil {
		r.Events[len(r.Events)-1].Passthrough = append(r.Events[len(r.Events)-1].Passthrough, data...)
	} else {
		r.Events = append(r.Events, RewriteEvent{Passthrough: append([]byte(nil), data...)})
	}
}

type NotificationRewriter struct {
	pending  []byte
	boundary outputBoundary
}

func (r *NotificationRewriter) Feed(chunk []byte, normalize bool) RewriteResult {
	var result RewriteResult
	pass := func(data []byte) {
		result.pass(data)
		for _, c := range data {
			r.boundary.advance(c)
		}
	}
	for _, c := range chunk {
		if len(r.pending) == 0 {
			if c == 0x1b && r.boundary.safe() {
				r.pending = append(r.pending, c)
			} else {
				pass([]byte{c})
			}
			continue
		}
		if len(r.pending) == 1 && c != ']' {
			pass(r.pending)
			r.pending = nil
			pass([]byte{c})
			continue
		}
		r.pending = append(r.pending, c)
		if len(r.pending) == 2 {
			continue
		}
		n := len(r.pending)
		complete := c == 7 || (c == '\\' && r.pending[n-2] == 0x1b)
		if c == 0x18 || c == 0x1a || n > notificationRewriteMaxPending {
			pass(r.pending)
			r.pending = nil
			continue
		}
		if !complete {
			continue
		}
		seq := r.pending
		r.pending = nil
		ps, body, ok := splitOSC(seq)
		// Bare ESC inside a string is ambiguous; preserve it without interpreting
		// nested OSC payload as an independent notification.
		end := len(seq) - 1
		if seq[end] == '\\' {
			end--
		}
		if bytes.IndexByte(seq[2:end], 0x1b) >= 0 {
			pass(seq)
			continue
		}
		if observation, progress := progressObservation(ps, body); ok && progress {
			pass(seq)
			result.Observations = append(result.Observations, observation)
			result.Events = append(result.Events, RewriteEvent{Observation: &observation})
		} else if message, actionable := nativeNotification(ps, body); ok && actionable && normalize {
			notification := notifyosc.Notification{Message: notifyosc.Sanitize([]byte(message))}
			result.Notifications = append(result.Notifications, notification)
			result.Events = append(result.Events, RewriteEvent{Notification: &notification})
		} else {
			pass(seq)
		}
	}
	return result
}
func (r *NotificationRewriter) Finish() []byte { out := r.pending; r.pending = nil; return out }

func progressObservation(ps, body []byte) (TurnObservation, bool) {
	if string(ps) != "9" || !bytes.HasPrefix(body, []byte("4;")) {
		return TurnObservation{}, false
	}
	state := body[len("4;"):]
	if i := bytes.IndexByte(state, ';'); i >= 0 {
		state = state[:i]
	}
	switch string(state) {
	case "3":
		return TurnObservation{Kind: ObservationWorking}, true
	case "0":
		return TurnObservation{Kind: ObservationStopped}, true
	default:
		return TurnObservation{}, false
	}
}

// progressOSCAuthorized keeps Claude's iTerm progress protocol from becoming
// lifecycle authority for arbitrary wrapped programs. The sequence remains
// transparent terminal output for every agent; only Claude may drive Reduce.
func progressOSCAuthorized(agent string) bool {
	return agent == "claude"
}

func splitOSC(seq []byte) (ps, body []byte, ok bool) {
	end := len(seq) - 1
	if end > 0 && seq[end-1] == 0x1b && seq[end] == '\\' {
		end--
	}
	content := seq[2:end]
	i := bytes.IndexByte(content, ';')
	if i < 0 {
		return nil, nil, false
	}
	return content[:i], content[i+1:], true
}

func nativeNotification(ps, body []byte) (string, bool) {
	switch string(ps) {
	case "9":
		if bytes.HasPrefix(body, []byte("4;")) {
			return "", false
		}
		if len(body) == 0 {
			return "agent attention", true
		}
		return string(body), true
	case "777":
		parts := strings.SplitN(string(body), ";", 3)
		if len(parts) != 3 || parts[0] != "notify" {
			return "", false
		}
		if message := notifyosc.Sanitize([]byte(parts[2])); message != "" {
			return message, true
		}
		if title := notifyosc.Sanitize([]byte(parts[1])); title != "" {
			return title, true
		}
		return "agent attention", true
	default:
		return "", false
	}
}
