// Package agentfixture implements deterministic native-agent stand-ins for the
// disposable Couch harness. It never contacts a model or borrows actor identity.
package agentfixture

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxInputBytes = 16 << 10
const MaxAccepted = 128
const MaxEscapeBytes = 128

type Phase string

const (
	Idle    Phase = "idle"
	Working Phase = "working"
	Picker  Phase = "picker"
)

type AgentState struct {
	Agent    string `json:"agent"`
	Phase    Phase  `json:"phase"`
	Draft    string `json:"draft,omitempty"`
	Paste    bool   `json:"paste,omitempty"`
	Escape   string `json:"-"`
	Accepted uint64 `json:"accepted"`
}

// ReduceAgent accepts a body only at a submission outside bracketed paste.
// State is a value; incomplete escape and UTF-8 sequences survive byte splits.
func ReduceAgent(before AgentState, input []byte) (AgentState, []string, error) {
	s := before
	if s.Agent != "claude" && s.Agent != "codex" {
		return before, nil, errors.New("unsupported fixture agent")
	}
	var accepted []string
	for _, b := range input {
		if s.Escape != "" {
			s.Escape += string([]byte{b})
			if len(s.Escape) > MaxEscapeBytes {
				return before, nil, errors.New("terminal escape exceeds limit")
			}
			if strings.HasPrefix(s.Escape, "\x1b]") || strings.HasPrefix(s.Escape, "\x1bP") {
				// OSC and DCS replies are control strings, including native
				// Zellij XTVERSION. Their ST may arrive in a later read.
				if (s.Escape[1] == ']' && b == 7) || strings.HasSuffix(s.Escape, "\x1b\\") {
					s.Escape = ""
				}
				continue
			}
			if s.Escape == "\x1b[200~" {
				s.Paste = true
				s.Escape = ""
				continue
			}
			if s.Escape == "\x1b[201~" {
				s.Paste = false
				s.Escape = ""
				continue
			}
			// CSI terminal replies and keyboard reports are not authored input.
			if len(s.Escape) > 2 && s.Escape[1] == '[' && b >= 0x40 && b <= 0x7e {
				s.Escape = ""
				continue
			}
			if len(s.Escape) == 2 && b != '[' && b != ']' && b != 'P' {
				s.Escape = ""
			}
			continue
		}
		if b == 0x1b {
			if s.Phase == Picker {
				s.Phase = Idle
				s.Draft = ""
			}
			s.Escape = "\x1b"
			continue
		}
		if s.Phase == Picker {
			continue
		}
		if b == 3 {
			s.Phase = Idle
			s.Draft = ""
			continue
		}
		if !s.Paste {
			switch b {
			case 0x15:
				s.Draft = ""
				continue
			case 0x7f, 8:
				if len(s.Draft) > 0 {
					_, n := utf8.DecodeLastRuneInString(s.Draft)
					s.Draft = s.Draft[:len(s.Draft)-n]
				}
				continue
			case '\r':
				if s.Agent == "claude" && strings.HasSuffix(s.Draft, "\\") {
					s.Draft = s.Draft[:len(s.Draft)-1] + "\n"
					continue
				}
				if s.Draft != "" {
					if !utf8.ValidString(s.Draft) {
						return before, nil, errors.New("submitted input is not UTF-8")
					}
					if s.Accepted >= MaxAccepted {
						return before, nil, errors.New("accepted input limit reached")
					}
					accepted = append(accepted, s.Draft)
					s.Accepted++
				}
				s.Draft = ""
				s.Phase = Idle
				continue
			}
		}
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			continue
		}
		s.Draft += string([]byte{b})
		if len(s.Draft) > MaxInputBytes {
			return before, nil, errors.New("fixture input exceeds limit")
		}
	}
	return s, accepted, nil
}
