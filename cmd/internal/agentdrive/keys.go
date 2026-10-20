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
	KeyLeft           KeyKind = "left"
	KeyRight          KeyKind = "right"
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

// StepDoc is one caller step. Steps is the single table the parser accepts,
// InputBytes encodes, and `couch --keys --help` and the couch skill describe
// (pair#716, pair#717). Bytes are the classic xterm encodings, the family the
// drive already sends to Claude and Codex; text and wait carry none. A family
// row (ctrl-LETTER, alt-LETTER, f1-f12) is generated per member and shares a
// Group, which help and StepSyntax print once.
type StepDoc struct {
	Kind            KeyKind
	Syntax, Meaning string
	Bytes           string
	Group           string
}

var Steps = buildSteps()

// RefusedSteps are names that look like keys but are never a plain step, each
// with the reason the refusal states (pair#717).
var RefusedSteps = map[KeyKind]string{
	"ctrl-c": "an interrupt kills the turn; use couch --interrupt SLOT --confirm",
	"ctrl-d": "EOF; a second one exits the agent",
	"ctrl-z": "suspends the agent process under the wrapper",
	"ctrl-i": "the same byte as tab; use tab",
	"ctrl-m": "the same byte as enter; use enter",
	"ctrl-u": "the same byte as clear-line; use clear-line",
	"ctrl-s": "XON/XOFF flow control can freeze the agent's output",
	"ctrl-q": "XON/XOFF flow control can freeze the agent's output",
}

func buildSteps() []StepDoc {
	steps := []StepDoc{
		{Kind: KeyText, Syntax: "text:LITERAL", Meaning: "type printable text; quote a step containing spaces"},
		{Kind: KeyEnter, Syntax: "enter", Meaning: "Return: newline in the composer, select in a picker", Bytes: "\r"},
		{Kind: KeySubmit, Syntax: "submit", Meaning: "Alt+Return: submit the composer", Bytes: "\x1b\r"},
		{Kind: KeyUp, Syntax: "up", Meaning: "arrow up", Bytes: "\x1b[A"},
		{Kind: KeyDown, Syntax: "down", Meaning: "arrow down", Bytes: "\x1b[B"},
		{Kind: KeyLeft, Syntax: "left", Meaning: "arrow left", Bytes: "\x1b[D"},
		{Kind: KeyRight, Syntax: "right", Meaning: "arrow right", Bytes: "\x1b[C"},
		{Kind: "tab", Syntax: "tab", Meaning: "Tab: completion, focus", Bytes: "\t"},
		{Kind: "shift-tab", Syntax: "shift-tab", Meaning: "Shift+Tab: cycle Claude's mode", Bytes: "\x1b[Z"},
		{Kind: "backspace", Syntax: "backspace", Meaning: "delete the character before the cursor", Bytes: "\x7f"},
		{Kind: "delete", Syntax: "delete", Meaning: "delete the character under the cursor", Bytes: "\x1b[3~"},
		{Kind: "home", Syntax: "home", Meaning: "Home", Bytes: "\x1b[H"},
		{Kind: "end", Syntax: "end", Meaning: "End", Bytes: "\x1b[F"},
		{Kind: "pageup", Syntax: "pageup", Meaning: "Page Up", Bytes: "\x1b[5~"},
		{Kind: "pagedown", Syntax: "pagedown", Meaning: "Page Down", Bytes: "\x1b[6~"},
		{Kind: KeyEsc, Syntax: "esc", Meaning: "Escape, always its own step", Bytes: "\x1b"},
		{Kind: KeyClearLine, Syntax: "clear-line", Meaning: "Ctrl+U: clear the composer's current line", Bytes: "\x15"},
		{Kind: KeyWait, Syntax: "wait:MS", Meaning: fmt.Sprintf("pause 0–%d ms, at most %d ms in total", MaxKeyWaitMS, MaxKeyTotalWaitMS)},
	}
	functionKeys := []string{"\x1bOP", "\x1bOQ", "\x1bOR", "\x1bOS", "\x1b[15~", "\x1b[17~", "\x1b[18~", "\x1b[19~", "\x1b[20~", "\x1b[21~", "\x1b[23~", "\x1b[24~"}
	for i, bytes := range functionKeys {
		name := fmt.Sprintf("f%d", i+1)
		steps = append(steps, StepDoc{Kind: KeyKind(name), Syntax: name, Meaning: "function keys F1 to F12", Bytes: bytes, Group: "f1-f12"})
	}
	for c := 'a'; c <= 'z'; c++ {
		if _, refused := RefusedSteps[KeyKind("ctrl-"+string(c))]; !refused {
			steps = append(steps, StepDoc{Kind: KeyKind("ctrl-" + string(c)), Syntax: "ctrl-" + string(c), Meaning: "Ctrl+letter chord, except the refused ones below", Bytes: string(rune(c - 'a' + 1)), Group: "ctrl-LETTER"})
		}
	}
	for c := 'a'; c <= 'z'; c++ {
		steps = append(steps, StepDoc{Kind: KeyKind("alt-" + string(c)), Syntax: "alt-" + string(c), Meaning: "Alt+letter chord", Bytes: "\x1b" + string(c), Group: "alt-LETTER"})
	}
	return steps
}

// StepSyntax lists every step's syntax, a family once by its group, in table
// order.
func StepSyntax() string {
	var syntax []string
	seen := map[string]bool{}
	for _, step := range Steps {
		name := step.Syntax
		if step.Group != "" {
			name = step.Group
		}
		if !seen[name] {
			seen[name] = true
			syntax = append(syntax, name)
		}
	}
	return strings.Join(syntax, " ")
}

func stepDoc(kind KeyKind) (StepDoc, bool) {
	for _, step := range Steps {
		if step.Kind == kind {
			return step, true
		}
	}
	return StepDoc{}, false
}

func isStep(kind KeyKind) bool {
	_, ok := stepDoc(kind)
	return ok
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
			if reason, refused := RefusedSteps[step.Kind]; refused {
				return fmt.Errorf("step %d: %s is not a step: %s", i+1, step.Kind, reason)
			}
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
	if k.Kind == KeyText {
		return []byte(k.Text)
	}
	if step, ok := stepDoc(k.Kind); ok && step.Bytes != "" {
		return []byte(step.Bytes)
	}
	return nil
}
