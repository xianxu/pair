package wrapcmd

import (
	"bytes"
	"strings"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

const notificationRewriteMaxPending = 64 << 10

type RewriteResult struct {
	Passthrough   []byte
	Notifications []notifyosc.Notification
	Observations  []TurnObservation
}

type NotificationRewriter struct {
	pending     []byte
	skippingOSC bool
	skipLastESC bool
}

func (r *NotificationRewriter) Feed(chunk []byte, normalize bool) RewriteResult {
	buf := append(append([]byte(nil), r.pending...), chunk...)
	r.pending = nil
	var result RewriteResult

	if r.skippingOSC {
		n := r.skipThroughOSC(buf)
		result.Passthrough = append(result.Passthrough, buf[:n]...)
		if r.skippingOSC {
			return result
		}
		buf = buf[n:]
	}

	for len(buf) > 0 {
		i := bytes.IndexByte(buf, 0x1b)
		if i < 0 {
			result.Passthrough = append(result.Passthrough, buf...)
			break
		}
		result.Passthrough = append(result.Passthrough, buf[:i]...)
		buf = buf[i:]
		if len(buf) == 1 {
			r.pending = append(r.pending, buf...)
			break
		}
		if buf[1] != ']' {
			result.Passthrough = append(result.Passthrough, buf[0])
			buf = buf[1:]
			continue
		}
		size, complete, malformed := rewriteOSCEnd(buf)
		if malformed {
			result.Passthrough = append(result.Passthrough, buf[:2]...)
			buf = buf[2:]
			continue
		}
		if !complete {
			if len(buf) > notificationRewriteMaxPending {
				result.Passthrough = append(result.Passthrough, buf...)
				r.skippingOSC = true
				r.skipLastESC = len(buf) > 0 && buf[len(buf)-1] == 0x1b
				break
			}
			r.pending = append(r.pending, buf...)
			break
		}
		seq := buf[:size]
		ps, body, ok := splitOSC(seq)
		if observation, progress := progressObservation(ps, body); ok && progress {
			result.Passthrough = append(result.Passthrough, seq...)
			result.Observations = append(result.Observations, observation)
		} else if message, actionable := nativeNotification(ps, body); ok && actionable {
			if normalize {
				result.Notifications = append(result.Notifications, notifyosc.Notification{Message: notifyosc.Sanitize([]byte(message))})
			} else {
				result.Passthrough = append(result.Passthrough, seq...)
			}
		} else {
			result.Passthrough = append(result.Passthrough, seq...)
		}
		buf = buf[size:]
	}
	return result
}

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

func rewriteOSCEnd(buf []byte) (size int, complete, malformed bool) {
	for i := 2; i < len(buf); i++ {
		switch buf[i] {
		case 0x07:
			return i + 1, true, false
		case 0x1b:
			if i+1 == len(buf) {
				return 0, false, false
			}
			if buf[i+1] == '\\' {
				return i + 2, true, false
			}
			return 0, false, true
		}
	}
	return 0, false, false
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

func (r *NotificationRewriter) skipThroughOSC(buf []byte) int {
	if r.skipLastESC {
		r.skipLastESC = false
		if len(buf) > 0 && buf[0] == '\\' {
			r.skippingOSC = false
			return 1
		}
	}
	for i := 0; i < len(buf); i++ {
		if buf[i] == 0x07 {
			r.skippingOSC = false
			return i + 1
		}
		if buf[i] == 0x1b {
			if i+1 < len(buf) && buf[i+1] == '\\' {
				r.skippingOSC = false
				return i + 2
			}
			if i+1 == len(buf) {
				r.skipLastESC = true
			}
		}
	}
	return len(buf)
}
