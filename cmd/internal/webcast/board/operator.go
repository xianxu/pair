package board

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// MaxOperatorBody bounds an entire binding message, including its envelope.
const MaxOperatorBody = 40 << 10

var ErrDriving = errors.New("board: a viewer is driving")
var ErrOperatorBusy = errors.New("board: an operator operation is outstanding")

type OperatorPrepare struct {
	Seg         int
	Path        []int
	Box, Region recording.Box
}
type OperatorSave struct {
	Draft string
	Color int
	Text  string
}
type OperatorResolve struct {
	Seg       int
	ID, State string
}
type OperatorEdit struct {
	Seg      int
	ID, Text string
}
type OperatorRequest struct {
	Op, Request string
	Generation  uint64
	Prepare     *OperatorPrepare
	Save        *OperatorSave
	Stroke      *StrokeIn
	Resolve     *OperatorResolve
	Edit        *OperatorEdit
	Draft       string
}

// ParseOperatorRequest accepts only the closed operator command union. Attribution
// is absent from the wire and is assigned by OperatorSession after CDP admission.
// Mutation generation is the canonical decimal uint64 the isolated client saw
// when it created the request, never an authority assigned on arrival.
func ParseOperatorRequest(body []byte) (OperatorRequest, error) {
	var raw struct {
		Op         *string         `json:"op"`
		Generation *string         `json:"generation"`
		Request    *string         `json:"request"`
		Body       json.RawMessage `json:"body"`
	}
	fail := func() (OperatorRequest, error) { return OperatorRequest{}, ErrShape }
	if len(body) > MaxOperatorBody || !decodeOperator(body, &raw, "op request generation body") || raw.Op == nil || raw.Request == nil || !idPattern.MatchString(*raw.Request) || len(raw.Body) == 0 || bytes.Equal(bytes.TrimSpace(raw.Body), []byte("null")) {
		return fail()
	}
	out := OperatorRequest{Op: *raw.Op, Request: *raw.Request}
	if raw.Generation != nil {
		generation, err := strconv.ParseUint(*raw.Generation, 10, 64)
		if err != nil || strconv.FormatUint(generation, 10) != *raw.Generation {
			return fail()
		}
		out.Generation = generation
	} else if out.Op != "status" && out.Op != "cancel-draft" {
		return fail()
	}
	switch out.Op {
	case "prepare-note":
		var v struct {
			Seg    *int        `json:"seg"`
			Path   *[]*int     `json:"path"`
			Box    *[]*float64 `json:"box"`
			Region *[]*float64 `json:"region"`
		}
		if !decodeOperator(raw.Body, &v, "seg path box region") || v.Seg == nil || !segOK(*v.Seg) || v.Path == nil || len(*v.Path) > 256 || v.Box == nil || v.Region == nil {
			return fail()
		}
		box, ok := boxOf(v.Box, 0)
		if !ok {
			return fail()
		}
		region, ok := boxOf(v.Region, 1)
		if !ok {
			return fail()
		}
		path := make([]int, len(*v.Path))
		for i, p := range *v.Path {
			if p == nil || *p < -1 || *p > 2147483647 {
				return fail()
			}
			path[i] = *p
		}
		out.Prepare = &OperatorPrepare{*v.Seg, path, box, region}
	case "save-note":
		var v struct {
			Draft *string `json:"draft"`
			Color *int    `json:"color"`
			Text  *string `json:"text"`
		}
		if !decodeOperator(raw.Body, &v, "draft color text") || v.Draft == nil || !draftHandle(*v.Draft) || v.Color == nil || !colorOK(*v.Color) || v.Text == nil || !noteText(*v.Text) {
			return fail()
		}
		out.Save = &OperatorSave{*v.Draft, *v.Color, *v.Text}
	case "stroke":
		var v struct {
			Seg   *int          `json:"seg"`
			ID    *string       `json:"id"`
			Color *int          `json:"color"`
			Pts   *[][]*float64 `json:"pts"`
		}
		if len(raw.Body) > MaxStrokeBody || !decodeOperator(raw.Body, &v, "seg id color pts") || v.Seg == nil || v.ID == nil || v.Color == nil || v.Pts == nil {
			return fail()
		}
		pts, ok := pointsOf(*v.Pts)
		if !ok {
			return fail()
		}
		in := StrokeIn{Seg: *v.Seg, ID: *v.ID, Color: *v.Color, Pts: pts}
		if !strokeShape(in) {
			return fail()
		}
		out.Stroke = &in
	case "edit-note":
		in, err := ParseNoteEdit(raw.Body)
		if err != nil {
			return fail()
		}
		out.Edit = &OperatorEdit{Seg: in.Seg, ID: in.ID, Text: in.Text}
	case "resolve":
		var v struct {
			Seg   *int    `json:"seg"`
			ID    *string `json:"id"`
			State *string `json:"state"`
		}
		if !decodeOperator(raw.Body, &v, "seg id state") || v.Seg == nil || !segOK(*v.Seg) || v.ID == nil || v.State == nil || !stateShape(StateIn{ID: *v.ID, State: *v.State}) {
			return fail()
		}
		out.Resolve = &OperatorResolve{*v.Seg, *v.ID, *v.State}
	case "cancel-draft", "draft-status":
		var v struct {
			Draft *string `json:"draft"`
		}
		if !decodeOperator(raw.Body, &v, "draft") || v.Draft == nil || !draftHandle(*v.Draft) {
			return fail()
		}
		out.Draft = *v.Draft
	case "status":
		var v struct{}
		if !decodeOperator(raw.Body, &v, "") {
			return fail()
		}
	default:
		return fail()
	}
	return out, nil
}
func draftHandle(s string) bool {
	return idPattern.MatchString(s) || (len(s) <= 32768 && strings.HasPrefix(s, "u.") && len(strings.Split(s, ".")) == 3)
}
func noteText(s string) bool { return broadcast.PlainText(s) && strings.TrimSpace(s) != "" }
func validBox(b recording.Box, min float64) bool {
	return coord(b[0]) && coord(b[1]) && extent(b[2]) && extent(b[3]) && b[2] >= min && b[3] >= min
}
func noteShape(in NoteIn) bool {
	return segOK(in.Seg) && in.Node > 0 && colorOK(in.Color) && noteText(in.Text) && validBox(in.Box, 0) && validBox(in.Region, 1)
}
func stateShape(in StateIn) bool {
	return idPattern.MatchString(in.ID) && (in.State == recording.NoteOpen || in.State == recording.NoteResolved)
}
func strokeShape(in StrokeIn) bool {
	if !segOK(in.Seg) || !idPattern.MatchString(in.ID) || !colorOK(in.Color) || len(in.Pts) == 0 || len(in.Pts) > broadcast.MaxPointsPerBatch {
		return false
	}
	for _, p := range in.Pts {
		if !coord(p[0]) || !coord(p[1]) || !(p[2] >= 0 && p[2] <= maxDT) {
			return false
		}
	}
	return true
}
func pointsOf(raw [][]*float64) ([][3]float64, bool) {
	if len(raw) == 0 || len(raw) > broadcast.MaxPointsPerBatch {
		return nil, false
	}
	pts := make([][3]float64, len(raw))
	for i, p := range raw {
		if len(p) != 3 {
			return nil, false
		}
		for j, n := range p {
			if n == nil {
				return nil, false
			}
			pts[i][j] = *n
		}
	}
	return pts, true
}
func (b *Board) annotationPageLocked(seg int, operator bool) (int, error) {
	doc, err := b.pageLocked(seg)
	if err == nil && operator && b.st.Control.Driver != nil {
		return 0, ErrDriving
	}
	return doc, err
}

// OperatorOutcome is evidence from the writer, not enqueue success. Unknown means
// the sink may have written part of a record; it cannot safely be submitted again.
type OperatorOutcome string

const (
	OperatorPending  OperatorOutcome = "pending"
	OperatorAccepted OperatorOutcome = "accepted"
	OperatorRefused  OperatorOutcome = "refused"
	OperatorUnknown  OperatorOutcome = "unknown"
)

type OperatorReply struct {
	Request string          `json:"request"`
	Outcome OperatorOutcome `json:"outcome"`
	ID      string          `json:"id,omitempty"`
	Error   string          `json:"error,omitempty"`
}
type OperatorEmit func(recording.Record, int, AnnotationAdmission, func(OperatorOutcome, error)) error

type operatorOperation struct {
	reply   OperatorReply
	payload string
}

// OperatorSession belongs to one trusted CDP context. IDs for all writes are
// positive, canonical decimal JS-safe integers increasing within that context.
// Two retained results (note/state and transient stroke) plus a high watermark
// bound memory without permitting an evicted request to write again.
type OperatorSession struct {
	mu           sync.Mutex
	board        *Board
	emit         OperatorEmit
	high         uint64
	last, stroke *operatorOperation
	active       *operatorOperation
	closed       bool
}

func (b *Board) NewOperator(emit OperatorEmit) *OperatorSession {
	return &OperatorSession{board: b, emit: emit}
}
func refused(request string, err error) OperatorReply {
	return OperatorReply{Request: request, Outcome: OperatorRefused, Error: err.Error()}
}
func (s *OperatorSession) Status(request string) OperatorReply {
	s.mu.Lock()
	var found *operatorOperation
	for _, op := range []*operatorOperation{s.last, s.stroke} {
		if op != nil && op.reply.Request == request {
			found = op
			break
		}
	}
	if found == nil {
		s.mu.Unlock()
		return OperatorReply{Request: request, Outcome: OperatorUnknown, Error: "board: result is no longer retained"}
	}
	reply := found.reply
	s.mu.Unlock()
	// A note's stable ID identifies this exact write. Board.Put follows the
	// production writer's disk sink, so its observed note is evidence even
	// when a later fanout failed. Never consult the recorder's ahead-of-sink
	// mirror, and never hold the session lock while reading the Board.
	if reply.Outcome == OperatorUnknown && reply.ID != "" && s.board.observedOperatorNote(reply.ID) {
		s.mu.Lock()
		if found.reply.Outcome == OperatorUnknown {
			found.reply.Outcome = OperatorAccepted
			found.reply.Error = ""
			if s.active == found {
				s.active = nil
			}
		}
		reply = found.reply
		s.mu.Unlock()
	}
	return reply
}

func (b *Board) observedOperatorNote(id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, note := range b.st.Notes {
		if note.ID == id && note.Author.ID == Operator {
			return true
		}
	}
	return false
}
func (s *OperatorSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.active != nil {
		s.active.reply.Outcome = OperatorUnknown
		s.active.reply.Error = ErrClosed.Error()
	}
}
func (s *OperatorSession) Note(request string, in NoteIn) OperatorReply {
	a, _ := s.board.admitOperatorAnnotation()
	return s.NoteAdmitted(a, request, in)
}
func (s *OperatorSession) NoteAdmitted(a AnnotationAdmission, request string, in NoteIn) OperatorReply {
	in.Viewer = Viewer{ID: Operator, Name: "Operator"}
	in.Owner = broadcast.OperatorNoteOwner
	return s.run(a, request, "note", in, func(emit Emit) (string, error) { return s.board.note(in, true, emit) })
}

func (s *OperatorSession) EditAdmitted(a AnnotationAdmission, request string, in EditIn) OperatorReply {
	in.Owner = broadcast.OperatorNoteOwner
	// The target already exists: only this request's writer acknowledgement
	// proves an edit, never observedOperatorNote(target).
	return s.run(a, request, "edit-note", in, func(emit Emit) (string, error) { return "", s.board.edit(in, true, emit) })
}
func (s *OperatorSession) Stroke(request string, in StrokeIn) OperatorReply {
	a, _ := s.board.admitOperatorAnnotation()
	return s.StrokeAdmitted(a, request, in)
}
func (s *OperatorSession) StrokeAdmitted(a AnnotationAdmission, request string, in StrokeIn) OperatorReply {
	in.Viewer = Viewer{ID: Operator, Name: "Operator"}
	return s.run(a, request, "stroke", in, func(emit Emit) (string, error) { return "", s.board.stroke(in, true, emit) })
}
func (s *OperatorSession) SetState(request string, seg int, in StateIn) OperatorReply {
	a, _ := s.board.admitOperatorAnnotation()
	return s.SetStateAdmitted(a, request, seg, in)
}
func (s *OperatorSession) SetStateAdmitted(a AnnotationAdmission, request string, seg int, in StateIn) OperatorReply {
	// A NoteState has no operation-specific record ID. Observing the target
	// state could reflect another person's resolve/reopen, so unknown state
	// writes remain conservative until their own writer confirms completion.
	in.Viewer = Viewer{ID: Operator, Name: "Operator"}
	return s.run(a, request, "state", struct {
		Seg   int
		State StateIn
	}{seg, in}, func(emit Emit) (string, error) { return "", s.board.setState(in, seg, true, emit) })
}
func (s *OperatorSession) run(a AnnotationAdmission, request, kind string, in any, write func(Emit) (string, error)) OperatorReply {
	data, err := json.Marshal(in)
	if err != nil {
		return refused(request, ErrShape)
	}
	payload := kind + string(data)
	s.mu.Lock()
	for _, op := range []*operatorOperation{s.last, s.stroke} {
		if op != nil && op.reply.Request == request {
			r := op.reply
			if op.payload != payload {
				r = refused(request, ErrShape)
			}
			s.mu.Unlock()
			return r
		}
	}
	n, err := strconv.ParseUint(request, 10, 64)
	if err != nil || n == 0 || n > 9007199254740991 || strconv.FormatUint(n, 10) != request || n <= s.high {
		s.mu.Unlock()
		return refused(request, ErrShape)
	}
	if s.closed {
		s.mu.Unlock()
		return refused(request, ErrClosed)
	}
	// Even a refused busy ID is consumed: it can never later become a write.
	s.high = n
	if s.active != nil {
		s.mu.Unlock()
		return refused(request, ErrOperatorBusy)
	}
	op := &operatorOperation{reply: OperatorReply{Request: request, Outcome: OperatorPending}, payload: payload}
	if kind == "stroke" {
		s.stroke = op
	} else {
		s.last = op
	}
	s.active = op
	s.mu.Unlock()
	var once sync.Once
	var reserved string
	settle := func(outcome OperatorOutcome, err error) {
		once.Do(func() {
			if outcome != OperatorAccepted && outcome != OperatorRefused && outcome != OperatorUnknown {
				outcome = OperatorUnknown
			}
			if outcome == OperatorRefused && reserved != "" {
				s.board.mu.Lock()
				delete(s.board.pending, reserved)
				s.board.mu.Unlock()
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			op.reply.Outcome = outcome
			if err != nil {
				op.reply.Error = err.Error()
			}
			if outcome != OperatorUnknown && s.active == op {
				s.active = nil
			}
		})
	}
	if a.board != s.board || a.authority.kind != ownerAnnotation {
		settle(OperatorRefused, ErrDisabled)
		return s.Status(request)
	}
	if err := a.Check(); err != nil {
		settle(OperatorRefused, err)
		return s.Status(request)
	}
	emitted := false
	id, err := write(func(r recording.Record, doc int) error {
		emitted = true
		if note, ok := r.(*recording.Note); ok {
			reserved = note.ID
			s.mu.Lock()
			op.reply.ID = note.ID
			s.mu.Unlock()
		}
		if s.emit == nil {
			return ErrClosed
		}
		err := s.emit(r, doc, a, settle)
		if err != nil {
			settle(OperatorRefused, err)
		}
		return err
	})
	if err != nil {
		settle(OperatorRefused, err)
	} else if !emitted {
		settle(OperatorAccepted, nil)
	}
	s.mu.Lock()
	if id != "" {
		op.reply.ID = id
	}
	reply := op.reply
	s.mu.Unlock()
	return reply
}

// decodeOperator additionally rejects duplicate keys and case aliases accepted
// by encoding/json. Each tagged command has a flat object; its arrays contain
// numbers only, and its nested envelope body is checked separately.
func decodeOperator(body []byte, v any, fields string) bool {
	dec := json.NewDecoder(bytes.NewReader(body))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return false
	}
	seen := map[string]bool{}
	allowed := " " + fields + " "
	for dec.More() {
		tok, err = dec.Token()
		if err != nil {
			return false
		}
		key, ok := tok.(string)
		if !ok || seen[key] || !strings.Contains(allowed, " "+key+" ") {
			return false
		}
		seen[key] = true
		var value json.RawMessage
		if dec.Decode(&value) != nil {
			return false
		}
	}
	return decode(body, v)
}
