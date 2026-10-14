package webcast

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

// Viewers' pointing and notes (#470): POST <cast>/stroke, /note and
// /note-state, the only routes that read a body. The pointer and drive links
// may use them; the view-only link is refused before its body is read
// (pair#506). The viewer a request names is attribution, not authentication. Admission is the
// terminal pointer routes': a JSON content type, an in-flight slot, a rate
// budget per route family, a prompt body within its bound.

// Budgets, per cast. Strokes allow two viewers drawing at the shell's 50 ms
// cadence at once; notes and note states share the terminal notes' budget.
const (
	strokeRatePerSec = 2 * broadcast.PointRatePerSec
	bodyReadBudget   = 2 * time.Second
)

// Refusals the viewer shows: fixed sentences that never echo the request.
const (
	refuseBusy          = "Too many requests. Wait a moment."
	refuseColorFull     = "This color already has 8 notes. Resolve one first."
	refuseStale         = "The page changed. Try again."
	refuseEnded         = "The cast has ended."
	refuseNotLive       = "The cast isn't showing a page yet."
	refusePaused        = "The cast is paused."
	refuseUnknown       = "That note is gone."
	refuseInvalid       = "That isn't valid."
	refuseBroken        = "Notes aren't working in this cast."
	refuseViewOnly      = "This link is view-only."
	refuseAnnotationOff = "Annotation is off."
)

// boardRoutes is a cast's admission state for the board's routes.
type boardRoutes struct {
	board    *board.Board
	inFlight chan struct{}
	mu       sync.Mutex // guards the rate limits
	strokes  *broadcast.RateLimit
	notes    *broadcast.RateLimit
}

func newBoardRoutes(b *board.Board) *boardRoutes {
	return &boardRoutes{board: b, inFlight: make(chan struct{}, broadcast.MaxPointInFlight),
		strokes: broadcast.NewRateLimit(strokeRatePerSec, strokeRatePerSec),
		notes:   broadcast.NewRateLimit(broadcast.NoteRatePerSec, broadcast.NoteBurst)}
}

// WithBoard serves viewers' pointing and notes into b. Without a board the
// routes are not found.
func (s *Server) WithBoard(b *board.Board) *Server {
	if b != nil {
		s.board = newBoardRoutes(b)
	}
	return s
}

// boardRoute serves one POST under link; false when rest isn't a board route.
func (s *Server) boardRoute(w http.ResponseWriter, r *http.Request, rest string, link castLink) bool {
	var admission board.AnnotationAdmission
	var handle func([]byte) error
	limit, max := (*boardRoutes).allowNote, int64(board.MaxNoteBody)
	switch rest {
	case "stroke":
		limit, max = (*boardRoutes).allowStroke, board.MaxStrokeBody
		handle = func(body []byte) error {
			in, err := board.ParseStroke(body)
			if err != nil {
				return err
			}
			return admission.Stroke(in)
		}
	case "note":
		handle = func(body []byte) error {
			in, err := board.ParseNote(body)
			if err != nil {
				return err
			}
			in.Owner = broadcast.NoteOwnerFromContext(r.Context())
			_, err = admission.Note(in)
			return err
		}
	case "note-edit":
		handle = func(body []byte) error {
			in, err := board.ParseNoteEdit(body)
			if err != nil {
				return err
			}
			in.Owner = broadcast.NoteOwnerFromContext(r.Context())
			return admission.Edit(in)
		}
	case "note-state":
		handle = func(body []byte) error {
			in, err := board.ParseNoteState(body)
			if err != nil {
				return err
			}
			return admission.SetState(in)
		}
	default:
		return false
	}
	if s.board == nil {
		http.NotFound(w, r)
		return true
	}
	// Before admission: a view-only link spends no body read, in-flight
	// slot or rate budget.
	if !link.annotates() {
		http.Error(w, refuseViewOnly, http.StatusForbidden)
		return true
	}
	var err error
	generation, parseErr := strconv.ParseUint(r.Header.Get("X-Couch-Annotation-Generation"), 10, 64)
	if parseErr != nil {
		http.Error(w, refuseAnnotationOff, http.StatusForbidden)
		return true
	}
	admission, err = s.board.board.AdmitAnnotationGeneration(generation)
	if err != nil {
		status, text := boardAnswer(err)
		http.Error(w, text, status)
		return true
	}
	body, release, ok := s.board.admit(w, r, limit, max)
	if !ok {
		return true
	}
	defer release()
	status, text := boardAnswer(handle(body))
	if status == http.StatusNoContent {
		w.WriteHeader(status)
	} else {
		http.Error(w, text, status)
	}
	return true
}

// paletteAnswer is palette.json: the swatches' colors
// (broadcast.PointerPalette, the board's one source) and whether this link
// may annotate (pair#506). A view-only viewer still draws others' strokes
// and notes, read-only.
type paletteAnswer struct {
	Palette  []string `json:"palette"`
	Annotate bool     `json:"annotate"`
}

// boardAsset serves palette.json only when the cast takes notes, so the
// viewer mounts the notes module only then.
func (s *Server) boardAsset(w http.ResponseWriter, rest string, link castLink) bool {
	if rest != "palette.json" || s.board == nil {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(paletteAnswer{Palette: broadcast.PointerPalette(), Annotate: link.annotates()})
	return true
}

func (b *boardRoutes) allowStroke(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.strokes.Allow(now)
}

func (b *boardRoutes) allowNote(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.notes.Allow(now)
}

// admit takes one request: a JSON content type, an in-flight slot, the
// route's budget, then a body that arrives promptly and within max. The
// caller defers release. On false the response is written.
func (b *boardRoutes) admit(w http.ResponseWriter, r *http.Request, allow func(*boardRoutes, time.Time) bool, max int64) ([]byte, func(), bool) {
	return admitBody(w, r, b.inFlight, func(now time.Time) bool { return allow(b, now) }, max)
}

// admitBody is the body-reading routes' one admission (the board's, #470;
// remote driving's, #467): a JSON content type, a slot in inFlight, the
// budget allow, then a body that arrives promptly and within max. The
// caller defers release. On false the response is written.
func admitBody(w http.ResponseWriter, r *http.Request, inFlight chan struct{}, allow func(time.Time) bool, max int64) ([]byte, func(), bool) {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return nil, nil, false
	}
	select {
	case inFlight <- struct{}{}:
	default:
		http.Error(w, refuseBusy, http.StatusTooManyRequests)
		return nil, nil, false
	}
	release := func() { <-inFlight }
	if !allow(time.Now()) {
		release()
		http.Error(w, refuseBusy, http.StatusTooManyRequests)
		return nil, nil, false
	}
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(bodyReadBudget))
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, max))
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		release()
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return nil, nil, false
	case err != nil:
		release()
		http.Error(w, "bad request", http.StatusBadRequest)
		return nil, nil, false
	}
	return body, release, true
}

// boardAnswer is the one mapping from a board request's outcome to the
// status and the sentence the viewer shows.
func boardAnswer(err error) (int, string) {
	switch {
	case err == nil:
		return http.StatusNoContent, ""
	case errors.Is(err, board.ErrDisabled):
		return http.StatusForbidden, refuseAnnotationOff
	case errors.Is(err, board.ErrOwner):
		return http.StatusForbidden, "Only the author can edit this note in its original browser session."
	case errors.Is(err, board.ErrColorFull):
		return http.StatusConflict, refuseColorFull
	case errors.Is(err, board.ErrStale):
		return http.StatusConflict, refuseStale
	case errors.Is(err, board.ErrNotLive):
		return http.StatusConflict, refuseNotLive
	case errors.Is(err, board.ErrPaused):
		return http.StatusConflict, refusePaused
	case errors.Is(err, board.ErrUnknown):
		return http.StatusConflict, refuseUnknown
	case errors.Is(err, board.ErrEnded), errors.Is(err, board.ErrClosed):
		return http.StatusGone, refuseEnded
	case errors.Is(err, board.ErrBroken):
		return http.StatusServiceUnavailable, refuseBroken
	case errors.Is(err, board.ErrShape):
		return http.StatusBadRequest, refuseInvalid
	default: // the writer failed: nothing the viewer did
		return http.StatusServiceUnavailable, refuseBusy
	}
}
