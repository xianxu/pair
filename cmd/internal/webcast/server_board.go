package webcast

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

// Viewers' pointing and notes (#470): POST <cast>/stroke, /note and
// /note-state, the only routes that read a body. Anyone with the cast link
// may use them (one always-on link, internal viewers, TL 10-12); the viewer
// a request names is attribution, not authentication. Admission is the
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
	refuseBusy      = "Too many requests. Wait a moment."
	refuseColorFull = "This color already has 8 notes. Resolve one first."
	refuseStale     = "The page changed. Try again."
	refuseEnded     = "The cast has ended."
	refuseNotLive   = "The cast isn't showing a page yet."
	refuseUnknown   = "That note is gone."
	refuseInvalid   = "That isn't valid."
	refuseBroken    = "Notes aren't working in this cast."
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

// boardRoute serves one POST; false when rest isn't a board route.
func (s *Server) boardRoute(w http.ResponseWriter, r *http.Request, rest string) bool {
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
			return s.board.board.Stroke(in)
		}
	case "note":
		handle = func(body []byte) error {
			in, err := board.ParseNote(body)
			if err != nil {
				return err
			}
			_, err = s.board.board.Note(in)
			return err
		}
	case "note-state":
		handle = func(body []byte) error {
			in, err := board.ParseNoteState(body)
			if err != nil {
				return err
			}
			return s.board.board.SetState(in)
		}
	default:
		return false
	}
	if s.board == nil {
		http.NotFound(w, r)
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
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return nil, nil, false
	}
	select {
	case b.inFlight <- struct{}{}:
	default:
		http.Error(w, refuseBusy, http.StatusTooManyRequests)
		return nil, nil, false
	}
	release := func() { <-b.inFlight }
	if !allow(b, time.Now()) {
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
	case errors.Is(err, board.ErrColorFull):
		return http.StatusConflict, refuseColorFull
	case errors.Is(err, board.ErrStale):
		return http.StatusConflict, refuseStale
	case errors.Is(err, board.ErrNotLive):
		return http.StatusConflict, refuseNotLive
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
