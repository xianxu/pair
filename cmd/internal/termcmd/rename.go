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
