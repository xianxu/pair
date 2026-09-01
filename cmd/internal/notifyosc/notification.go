// Package notifyosc owns Pair's normalized terminal-notification envelope.
package notifyosc

import (
	"strings"
	"unicode/utf8"
)

const MaxMessageBytes = 4 << 10

const Prefix = "\x1b]777;notify;pair;"

type Notification struct {
	Message string
}

// Sanitize converts arbitrary bytes to bounded terminal-safe UTF-8.
func Sanitize(raw []byte) string {
	valid := strings.ToValidUTF8(string(raw), "�")
	var out strings.Builder
	out.Grow(min(len(valid), MaxMessageBytes))
	for _, r := range valid {
		if r <= 0x1f || r >= 0x7f && r <= 0x9f {
			continue
		}
		n := utf8.RuneLen(r)
		if out.Len()+n > MaxMessageBytes {
			break
		}
		out.WriteRune(r)
	}
	return out.String()
}

// Encode returns the one Pair-owned notification envelope.
func Encode(message string) []byte {
	clean := Sanitize([]byte(message))
	out := make([]byte, 0, len(Prefix)+len(clean)+1)
	out = append(out, Prefix...)
	out = append(out, clean...)
	out = append(out, 0x07)
	return out
}

// DecodeOSC accepts one complete canonical Pair envelope and nothing else.
func DecodeOSC(sequence []byte) (Notification, bool) {
	if len(sequence) < len(Prefix)+1 || string(sequence[:len(Prefix)]) != Prefix {
		return Notification{}, false
	}
	end := len(sequence) - 1
	switch {
	case sequence[end] == 0x07:
	case end >= 1 && sequence[end-1] == 0x1b && sequence[end] == '\\':
		end--
	default:
		return Notification{}, false
	}
	return Notification{Message: Sanitize(sequence[len(Prefix):end])}, true
}
