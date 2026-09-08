package termcmd

import "strings"

type RenameEventKind int

const (
	RenameConsume RenameEventKind = iota
	RenameInsert
	RenameMoveLeft
	RenameMoveRight
	RenameHome
	RenameEnd
	RenameBackspace
	RenameDelete
	RenameDeleteToStart
	RenameCommit
	RenameCancel
)

type RenameEvent struct {
	Kind RenameEventKind
	Rune rune
}

type RenameOutcomeKind int

const (
	RenameOutcomeNone RenameOutcomeKind = iota
	RenameOutcomeCommit
	RenameOutcomeCancel
)

type RenameOutcome struct {
	Kind RenameOutcomeKind
	Name string
}

type RenameEditor struct {
	original string
	text     []rune
	cursor   int
}

func NewRenameEditor(name string) RenameEditor {
	text := []rune(name)
	return RenameEditor{original: name, text: text, cursor: len(text)}
}

func (e RenameEditor) Text() string {
	return string(e.text)
}

func (e RenameEditor) Cursor() int {
	return e.cursor
}

func (e RenameEditor) Original() string {
	return e.original
}

// Field is the editor as a SURFACE draws it: the text with the caret in it.
//
// ONE surface today -- the tab strip. It was written for two, when the pane
// TITLE also carried the field, and the caret was being placed independently in
// each; the title's copy was then deleted outright in the same milestone (#199
// M3), which is the better fix and leaves this with a single caller. It stays a
// method rather than folding back into RenderStrip because the clamp belongs
// with the editor's own invariants, and because the strip renders from a pure
// model that must not reach into an editor to compose a caret.
func (e RenameEditor) Field() string {
	cursor := e.cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(e.text) {
		cursor = len(e.text)
	}
	return string(e.text[:cursor]) + "│" + string(e.text[cursor:])
}

func (e RenameEditor) Apply(event RenameEvent) (RenameEditor, RenameOutcome) {
	e.text = append([]rune(nil), e.text...)
	switch event.Kind {
	case RenameInsert:
		e.text = append(e.text, 0)
		copy(e.text[e.cursor+1:], e.text[e.cursor:])
		e.text[e.cursor] = event.Rune
		e.cursor++
	case RenameMoveLeft:
		if e.cursor > 0 {
			e.cursor--
		}
	case RenameMoveRight:
		if e.cursor < len(e.text) {
			e.cursor++
		}
	case RenameHome:
		e.cursor = 0
	case RenameEnd:
		e.cursor = len(e.text)
	case RenameBackspace:
		if e.cursor > 0 {
			e.text = append(e.text[:e.cursor-1], e.text[e.cursor:]...)
			e.cursor--
		}
	case RenameDelete:
		if e.cursor < len(e.text) {
			e.text = append(e.text[:e.cursor], e.text[e.cursor+1:]...)
		}
	case RenameDeleteToStart:
		e.text = append([]rune(nil), e.text[e.cursor:]...)
		e.cursor = 0
	case RenameCommit:
		name := strings.TrimSpace(string(e.text))
		if name == "" {
			name = e.original
		}
		return e, RenameOutcome{Kind: RenameOutcomeCommit, Name: name}
	case RenameCancel:
		return e, RenameOutcome{Kind: RenameOutcomeCancel, Name: e.original}
	}
	return e, RenameOutcome{}
}
