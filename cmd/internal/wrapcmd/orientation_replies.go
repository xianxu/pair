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
		if n == 0 {
			t.input = nil
			return true
		}
		key := orientationReplyKind(string(t.input[:n]))
		if key == "" || t.queries[key] == 0 {
			t.input = nil
			return true
		}
		t.queries[key]--
		t.input = t.input[n:]
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
