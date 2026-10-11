package wrapcmd

import (
	"bytes"
	"strings"

	"github.com/xianxu/pair/cmd/internal/ansi"
)

// orientationReplies recognizes only responses to queries actually forwarded
// by this wrapper. It does not consume or rewrite input bytes. Tracking partial
// framing prevents a split terminal response from masquerading as keystrokes.
type orientationReplies struct {
	queries       map[string]int
	output, input []byte
	inFlight      int
}

func (t *orientationReplies) observeQueries(data []byte) {
	t.output = append(t.output, data...)
	for len(t.output) > 0 {
		at := bytes.IndexByte(t.output, 0x1b)
		if at < 0 {
			t.output = nil
			return
		}
		t.output = t.output[at:]
		n, status := orientationEscape(t.output)
		if status == ansi.Incomplete {
			if len(t.output) > 4096 {
				t.output = nil
			}
			return
		}
		if n == 0 {
			t.output = t.output[1:]
			continue
		}
		query := string(t.output[:n])
		key := ""
		switch query {
		case "\x1b[6n", "\x1b[?6n":
			key = "cpr"
		case "\x1b[c", "\x1b[0c":
			key = "da1"
		case "\x1b[>c", "\x1b[>0c":
			key = "da2"
		case "\x1b[?u":
			key = "kitty"
		case "\x1b[5n":
			key = "status"
		case "\x1b[>q":
			key = "version"
		case "\x1b[18t":
			key = "window"
		}
		if strings.HasPrefix(query, "\x1b[?") && strings.HasSuffix(query, "$p") {
			mode := query[3 : len(query)-2]
			if mode == "2026" || mode == "2027" {
				key = "mode" + mode
			}
		}
		for _, color := range []string{"10", "11", "12"} {
			if query == "\x1b]"+color+";?\x07" || query == "\x1b]"+color+";?\x1b\\" {
				key = "color" + color
			}
		}
		if key != "" {
			if t.queries == nil {
				t.queries = map[string]int{}
			}
			if t.queries[key] < 16 {
				t.queries[key]++
			}
		}
		t.output = t.output[n:]
	}
}
func orientationEscape(data []byte) (int, ansi.Status) {
	if len(data) >= 2 && (data[1] == ']' || data[1] == 'P') {
		if at := bytes.Index(data[2:], []byte("\x1b\\")); at >= 0 {
			return at + 4, ansi.Complete
		}
		if data[1] == ']' {
			if at := bytes.IndexByte(data[2:], 7); at >= 0 {
				return at + 3, ansi.Complete
			}
		}
		return 0, ansi.Incomplete
	}
	return ansi.Frame(data)
}

// operatorData is orientation's conservative reading: any input that is not
// a reply to a query this wrapper forwarded is operator input.
func (t *orientationReplies) operatorData(data []byte) bool {
	t.input = append(t.input, data...)
	for len(t.input) > 0 {
		if t.input[0] != 0x1b {
			t.input = nil
			return true
		}
		n, status := orientationEscape(t.input)
		if status == ansi.Incomplete {
			if len(t.input) > 4096 {
				t.input = nil
				return true
			}
			return false
		}
		if n == 0 || !t.solicited(string(t.input[:n])) {
			t.input = nil
			return true
		}
		t.input = t.input[n:]
	}
	return false
}

// operatorKeystrokes is peer delivery's reading (pair#428): whether the input
// holds a real keystroke. Focus, mouse and terminal reports are not
// keystrokes, solicited or not; a solicited reply still settles its query.
func (t *orientationReplies) operatorKeystrokes(data []byte) bool {
	t.input = append(t.input, data...)
	for len(t.input) > 0 {
		if t.input[0] != 0x1b {
			t.input = nil
			return true
		}
		n, status := inputFrame(t.input)
		if status == ansi.Incomplete {
			if len(t.input) > 4096 {
				t.input = nil
				return true
			}
			return false
		}
		if n == 0 {
			t.input = nil
			return true
		}
		frame := string(t.input[:n])
		if !t.solicited(frame) && !terminalReport(frame) {
			t.input = nil
			return true
		}
		t.input = t.input[n:]
	}
	return false
}

// flushPartial resolves a frame the input stream left unfinished once no
// continuation arrived: a lone ESC, or Alt+] read as an OSC opener, is a key.
func (t *orientationReplies) flushPartial() bool {
	partial := len(t.input) > 0
	t.input = nil
	return partial
}

// solicited consumes one outstanding query when frame answers it.
func (t *orientationReplies) solicited(frame string) bool {
	key := orientationReplyKind(frame)
	if key == "" || t.queries[key] == 0 {
		return false
	}
	t.queries[key]--
	return true
}

// inputFrame is orientationEscape plus the X10 mouse report, whose three
// coordinate bytes follow a bare CSI M.
func inputFrame(data []byte) (int, ansi.Status) {
	if bytes.HasPrefix(data, []byte("\x1b[M")) {
		if len(data) < 6 {
			return 0, ansi.Incomplete
		}
		return 6, ansi.Complete
	}
	return orientationEscape(data)
}

// terminalReport is a complete input frame the terminal, not the operator,
// produced. A cursor position report (CSI r;c R) is absent on purpose:
// modified F3 has the same shape, so only a solicited CPR is a report.
func terminalReport(frame string) bool {
	switch {
	case frame == "\x1b[I" || frame == "\x1b[O":
		return true // focus
	case strings.HasPrefix(frame, "\x1b[M") && len(frame) == 6:
		return true // X10 mouse
	case strings.HasPrefix(frame, "\x1b]") || strings.HasPrefix(frame, "\x1bP"):
		return len(frame) > 2 // OSC or DCS reply, not a bare Alt+] / Alt+P
	case !strings.HasPrefix(frame, "\x1b[") || len(frame) < 4:
		return false
	}
	params, final := frame[2:len(frame)-1], frame[len(frame)-1]
	private := params[0] == '?' || params[0] == '>' || params[0] == '<'
	switch {
	case params[0] == '<' && (final == 'M' || final == 'm'):
		return decimalFields(params[1:], 3) // SGR mouse
	case final == 'M':
		return decimalFields(params, 3) // urxvt mouse
	case strings.HasSuffix(params, "$"):
		return final == 'y' // DECRPM / RPM
	case final == 'c' || (final == 'u' && params[0] == '?') || (final == 'R' && params[0] == '?'):
		return private // DA1/DA2, kitty flags, DECXCPR
	case final == 'n' || final == 't':
		return true // DSR, color-scheme notice, window reports
	}
	return false
}

func orientationReplyKind(reply string) string {
	// DECRPM is a response to DECRQM for one exact private mode. Mode
	// reports are terminal negotiation, but unsolicited reports remain input.
	// Track only the synchronized-output and grapheme modes observed at startup.
	if strings.HasPrefix(reply, "\x1b[?") && strings.HasSuffix(reply, "$y") {
		fields := strings.Split(reply[3:len(reply)-2], ";")
		if len(fields) == 2 && (fields[0] == "2026" || fields[0] == "2027") && len(fields[1]) == 1 && fields[1][0] >= '0' && fields[1][0] <= '4' {
			return "mode" + fields[0]
		}
		return ""
	}
	switch {
	case strings.HasPrefix(reply, "\x1b[?") && strings.HasSuffix(reply, "c") && decimalFields(reply[3:len(reply)-1], 1):
		return "da1"
	case strings.HasPrefix(reply, "\x1b[>") && strings.HasSuffix(reply, "c") && decimalFields(reply[3:len(reply)-1], 1):
		return "da2"
	case strings.HasPrefix(reply, "\x1b[?") && strings.HasSuffix(reply, "u") && decimalFields(reply[3:len(reply)-1], 1):
		return "kitty"
	case strings.HasPrefix(reply, "\x1b[") && strings.HasSuffix(reply, "R") && decimalFields(strings.TrimPrefix(reply[2:len(reply)-1], "?"), 2):
		return "cpr"
	case reply == "\x1b[0n":
		return "status"
	case strings.HasPrefix(reply, "\x1bP>|") && strings.HasSuffix(reply, "\x1b\\"):
		return "version"
	case strings.HasPrefix(reply, "\x1b[8;") && strings.HasSuffix(reply, "t") && decimalFields(reply[4:len(reply)-1], 2):
		return "window"
	}
	for _, color := range []string{"10", "11", "12"} {
		if strings.HasPrefix(reply, "\x1b]"+color+";rgb:") {
			return "color" + color
		}
	}
	return ""
}
func decimalFields(s string, minFields int) bool {
	fields := strings.Split(s, ";")
	if len(fields) < minFields {
		return false
	}
	for _, field := range fields {
		if field == "" {
			return false
		}
		for _, r := range field {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}
