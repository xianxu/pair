// Package notifyosc owns Pair's normalized terminal-notification envelope.
package notifyosc

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxMessageBytes = 4 << 10

const MaxTitleBytes = 256

const Prefix = "\x1b]777;notify;pair;"

// DecodeZellijOSC9 decodes the OSC payload (including its numeric command)
// produced by Zellij's osc9 host-notification protocol. The prefix is transport
// metadata and does not consume the application's message-byte allowance.
func DecodeZellijOSC9(payload []byte) (Notification, bool) {
	const prefix = "9;pair: "
	if !strings.HasPrefix(string(payload), prefix) || len(payload)-len(prefix) > MaxMessageBytes {
		return Notification{}, false
	}
	return Notification{Message: Sanitize(payload[len(prefix):])}, true
}

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

// ValidateDesktop validates raw byte budgets and requires visible sanitized text.
// It is shared by notification ingress and the final presenter boundary.
func ValidateDesktop(title, body string) error {
	if len(title) > MaxTitleBytes {
		return errors.New("notification title exceeds 256 bytes")
	}
	if len(body) > MaxMessageBytes {
		return errors.New("notification message exceeds 4096 bytes")
	}
	if strings.TrimSpace(Sanitize([]byte(title))) == "" {
		return errors.New("notification title is blank")
	}
	if strings.TrimSpace(Sanitize([]byte(body))) == "" {
		return errors.New("notification message is blank")
	}
	return nil
}

// EncodeDesktop returns one sanitized, ST-terminated OSC 777 notification.
// Semicolons delimit the title field but are ordinary text in the body.
func EncodeDesktop(title, body string) ([]byte, error) {
	if err := ValidateDesktop(title, body); err != nil {
		return nil, err
	}
	title = strings.ReplaceAll(Sanitize([]byte(title)), ";", ",")
	return []byte("\x1b]777;notify;" + title + ";" + Sanitize([]byte(body)) + "\x1b\\"), nil
}
