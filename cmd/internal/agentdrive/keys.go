// Package agentdrive holds pure, bounded agent input and screen decisions.
package agentdrive

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type KeyKind string

const (
	KeyText           KeyKind = "text"
	KeyEnter          KeyKind = "enter"
	KeySubmit         KeyKind = "submit"
	KeyUp             KeyKind = "up"
	KeyDown           KeyKind = "down"
	KeyEsc            KeyKind = "esc"
	KeyClearLine      KeyKind = "clear-line"
	KeyWait           KeyKind = "wait"
	MaxKeySteps               = 64
	MaxKeyTextBytes           = 8 * 1024
	MaxKeyWaitMS              = 1000
	MaxKeyTotalWaitMS         = 5000
)

type KeyStep struct {
	Kind   KeyKind
	Text   string `json:",omitempty"`
	WaitMS int    `json:",omitempty"`
}

type KeySequence []KeyStep

// StepDoc is one caller step. Steps is the single table the parser accepts and
// `couch --keys --help` and the couch skill describe (pair#716).
type StepDoc struct {
	Kind            KeyKind
	Syntax, Meaning string
}

var Steps = []StepDoc{
	{KeyText, "text:LITERAL", "type printable text; quote a step containing spaces"},
	{KeyEnter, "enter", "Return: newline in the composer, select in a picker"},
	{KeySubmit, "submit", "Alt+Return: submit the composer"},
	{KeyUp, "up", "arrow up"},
	{KeyDown, "down", "arrow down"},
	{KeyEsc, "esc", "Escape, always its own step"},
	{KeyClearLine, "clear-line", "Ctrl+U: clear the composer's current line"},
	{KeyWait, "wait:MS", fmt.Sprintf("pause 0–%d ms, at most %d ms in total", MaxKeyWaitMS, MaxKeyTotalWaitMS)},
}

// StepSyntax lists every step's syntax, space-separated, in table order.
func StepSyntax() string {
	syntax := make([]string, len(Steps))
	for i, step := range Steps {
		syntax[i] = step.Syntax
	}
	return strings.Join(syntax, " ")
}

func isStep(kind KeyKind) bool {
	for _, step := range Steps {
		if step.Kind == kind {
			return true
		}
	}
	return false
}

// ParseKeySequence accepts exact named inputs, text:literal and wait:milliseconds.
// Invalid input never returns a partially actionable sequence.
func ParseKeySequence(specs []string) (KeySequence, error) {
	if len(specs) == 0 || len(specs) > MaxKeySteps {
		return nil, fmt.Errorf("key sequence must have 1–%d steps; shorten the sequence", MaxKeySteps)
	}
	sequence := make(KeySequence, 0, len(specs))
	for i, spec := range specs {
		step := KeyStep{Kind: KeyKind(spec)}
		if text, ok := strings.CutPrefix(spec, "text:"); ok {
			step = KeyStep{Kind: KeyText, Text: text}
		} else if wait, ok := strings.CutPrefix(spec, "wait:"); ok {
			if wait == "" {
				return nil, fmt.Errorf("step %d: wait requires decimal milliseconds", i+1)
			}
			for _, r := range wait {
				if r < '0' || r > '9' {
					return nil, fmt.Errorf("step %d: wait requires decimal milliseconds", i+1)
				}
			}
			ms, err := strconv.Atoi(wait)
			if err != nil {
				return nil, fmt.Errorf("step %d: wait must be 0–%d milliseconds; shorten the sequence", i+1, MaxKeyWaitMS)
			}
			step = KeyStep{Kind: KeyWait, WaitMS: ms}
		}
		sequence = append(sequence, step)
	}
	if err := sequence.Validate(); err != nil {
		return nil, err
	}
	return sequence, nil
}

// Validate also checks directly constructed or decoded sequences before use.
func (s KeySequence) Validate() error {
	if len(s) == 0 || len(s) > MaxKeySteps {
		return fmt.Errorf("key sequence must have 1–%d steps; shorten the sequence", MaxKeySteps)
	}
	textBytes, waitMS := 0, 0
	for i, step := range s {
		switch step.Kind {
		case KeyText:
			if step.WaitMS != 0 || step.Text == "" || !utf8.ValidString(step.Text) {
				return fmt.Errorf("step %d: text requires nonempty printable UTF-8 and no wait field", i+1)
			}
			if len(step.Text) > MaxKeyTextBytes-textBytes {
				return fmt.Errorf("text exceeds %d aggregate bytes; shorten the sequence", MaxKeyTextBytes)
			}
			for _, r := range step.Text {
				if !unicode.IsPrint(r) {
					return fmt.Errorf("step %d: text contains nonprintable %U; use named keys for controls", i+1, r)
				}
			}
			textBytes += len(step.Text)
		case KeyWait:
			if step.Text != "" || step.WaitMS < 0 || step.WaitMS > MaxKeyWaitMS {
				return fmt.Errorf("step %d: wait requires 0–%d milliseconds and no text field; shorten the sequence", i+1, MaxKeyWaitMS)
			}
			if step.WaitMS > MaxKeyTotalWaitMS-waitMS {
				return fmt.Errorf("wait exceeds %d aggregate milliseconds; shorten the sequence", MaxKeyTotalWaitMS)
			}
			waitMS += step.WaitMS
		default:
			if !isStep(step.Kind) {
				return fmt.Errorf("step %d: unknown key %q; steps are %s", i+1, step.Kind, StepSyntax())
			}
			if step.Text != "" || step.WaitMS != 0 {
				return fmt.Errorf("step %d: %s accepts no extra fields", i+1, step.Kind)
			}
		}
	}
	return nil
}

// InputBytes returns one step's input. Callers preserve step boundaries, notably
// a standalone ESC before a subsequent Return; waits have no input bytes.
// The complete sequence must pass Validate before the first step is executed.
func (k KeyStep) InputBytes() []byte {
	switch k.Kind {
	case KeyText:
		return []byte(k.Text)
	case KeyEnter:
		return []byte{'\r'}
	case KeySubmit:
		return []byte{'\x1b', '\r'}
	case KeyUp:
		return []byte("\x1b[A")
	case KeyDown:
		return []byte("\x1b[B")
	case KeyLeft:
		return []byte("\x1b[D")
	case KeyRight:
		return []byte("\x1b[C")
	case KeyEsc:
		return []byte{'\x1b'}
	case KeyClearLine:
		return []byte{'\x15'}
	default:
		return nil
	}
}
