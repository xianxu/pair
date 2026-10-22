package broadcast

import (
	"github.com/xianxu/pair/cmd/internal/castlock"

	"encoding/base64"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

// contentSecurityPolicy confines the viewer page to its own origin. Scripts
// are strict; styles allow inline because xterm.js's DOM renderer inserts
// <style> elements, and inline style can neither run code nor reach another
// origin.
const contentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self'; font-src 'self'; frame-ancestors 'none'"

// DefaultPing keeps an idle stream open through Cloudflare, which closes
// streams quiet for 100s, and reveals viewers that went away.
const DefaultPing = 15 * time.Second

type ServerOptions struct {
	// Now supplies logical rate-limit time. Socket deadlines use real time.
	Now   func() time.Time
	Token string
	// Locks guards the view and pointer links (pair#495); nil serves them
	// unlocked.
	Locks *LinkLocks
	Hub   *Hub
	Ping  time.Duration
	// Theme, when set, is read for each new viewer and sent before its
	// first frame.
	Theme func() Theme
	// Pointer is the broadcast's pointer capability (#412); nil serves no
	// pointer link. OnPoint receives each well-formed batch posted while
	// pointing is on; it decides whether the batch lands.
	Pointer *PointerState
	OnPoint func(PointBatch)
	// OnNote and OnResolve decide whether a well-formed note or resolve
	// request lands (#447); their errors map to fixed refusal text.
	OnNote     func(NoteRequest) error
	OnResolve  func(NoteResolutionRequest) error
	OnNoteEdit func(NoteEditRequest) error
}

type asset struct{ file, contentType string }

// assets is every path served under /<token>/; nothing else is.
var assets = map[string]asset{
	"":           {"web/index.html", "text/html; charset=utf-8"},
	"viewer.js":  {"web/viewer.js", "text/javascript; charset=utf-8"},
	"notes.js":   {"web/notes.js", "text/javascript; charset=utf-8"},
	"collab.js":  {"web/collab.js", "text/javascript; charset=utf-8"},
	"viewer.css": {"web/viewer.css", "text/css; charset=utf-8"},
	"xterm.js":   {"web/vendor/xterm/xterm.js", "text/javascript; charset=utf-8"},
	"xterm.css":  {"web/vendor/xterm/xterm.css", "text/css; charset=utf-8"},
	// Unicode 11 widths, so emoji are two columns in the viewer as in Couch.
	"addon-unicode11.js": {"web/vendor/xterm/addon-unicode11.js", "text/javascript; charset=utf-8"},
	// JetBrains Mono, the font the operator's terminal draws (OFL; see
	// web/vendor/fonts/VENDOR.md).
	"fonts/JetBrainsMono-Regular.woff2":    {"web/vendor/fonts/JetBrainsMono-Regular.woff2", "font/woff2"},
	"fonts/JetBrainsMono-Bold.woff2":       {"web/vendor/fonts/JetBrainsMono-Bold.woff2", "font/woff2"},
	"fonts/JetBrainsMono-Italic.woff2":     {"web/vendor/fonts/JetBrainsMono-Italic.woff2", "font/woff2"},
	"fonts/JetBrainsMono-BoldItalic.woff2": {"web/vendor/fonts/JetBrainsMono-BoldItalic.woff2", "font/woff2"},
	// Symbols JetBrains Mono lacks, which iPad Safari would draw as emoji
	// (#415; web/vendor/fonts/symbols.py).
	"fonts/NotoSansSymbols2-Couch.woff": {"web/vendor/fonts/NotoSansSymbols2-Couch.woff", "font/woff"},
}

// Server is the view-only endpoint. It serves GET only, never reads a request
// body, and answers only under the capability token's path.
type Server struct{ opts ServerOptions }

func NewServer(opts ServerOptions) *Server {
	if opts.Ping <= 0 {
		opts.Ping = DefaultPing
	}
	return &Server{opts: opts}
}

// SetSecurityHeaders sets what every cast response carries: csp, no caching,
// no referrer, no sniffing.
func SetSecurityHeaders(h http.Header, csp string) {
	h.Set("Content-Security-Policy", csp)
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
}

// Owns reports whether segment is one of this broadcast's tokens, for the
// Host's routing.
func (s *Server) Owns(segment string) bool {
	return OwnsToken(segment, s.opts.Token) || s.opts.Pointer.Match(segment)
}

// LockFor is the lock guarding segment, the view link's or the pointer
// link's.
func (s *Server) LockFor(segment string) *castlock.Lock { return s.opts.Locks.LockFor(segment) }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	SetSecurityHeaders(h, contentSecurityPolicy)
	segment, rest, ok := splitPath(r.URL.Path)
	pointer := ok && s.opts.Pointer.Match(segment)
	// The only routes that read a body: POST /<pointer-token>/point, note
	// and resolve. Every other path is GET-only and never reads one (#395).
	if r.Method == http.MethodPost && pointer {
		switch rest {
		case "point":
			s.point(w, r)
			return
		case "note":
			s.note(w, r)
			return
		case "note-edit":
			s.noteEdit(w, r)
			return
		case "resolve":
			s.resolve(w, r)
			return
		}
	}
	if r.Method != http.MethodGet {
		h.Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	view := ok && OwnsToken(segment, s.opts.Token)
	if !view && !pointer {
		http.NotFound(w, r)
		return
	}
	if rest == "events" {
		s.events(w, r, pointer)
		return
	}
	a, ok := assets[rest]
	if !ok {
		http.NotFound(w, r)
		return
	}
	body, err := webFS.ReadFile(a.file)
	if err != nil {
		http.Error(w, "unavailable", http.StatusInternalServerError)
		return
	}
	h.Set("Content-Type", a.contentType)
	w.Write(body)
}

// splitPath splits "/<token>/<rest>".
func splitPath(path string) (segment, rest string, ok bool) {
	if !strings.HasPrefix(path, "/") {
		return "", "", false
	}
	segment, rest, ok = strings.Cut(path[1:], "/")
	return segment, rest, ok && segment != ""
}

// wireCaps tells a pointer page whether pointing is on, the palette its
// swatches offer, and the color Couch assigned this stream, which the page
// uses until its viewer picks one (#446).
type wireCaps struct {
	Pointer bool     `json:"pointer"`
	Color   int      `json:"color"`
	Palette []string `json:"palette"`
}

// Refusals a pointer page may show its viewer: a closed set of fixed
// sentences that never echo the request.
const (
	refusePointerOff = "Pointing is off."
	refuseBusy       = "Too many requests. Wait a moment."
	refuseColorFull  = "This color already has 8 notes. Resolve one first."
	refuseOverlap    = "A note is already there."
	refuseScreen     = "The screen changed. Try again."
	refuseNote       = "That note isn't valid."
)

// readPointerBody admits one POST from a pointer page and reads its body:
// pointing on, a JSON content type, an in-flight slot, the route's rate
// budget, then a body that arrives promptly and within max. The caller
// defers release: the slot is held while the request is read and handled.
// On false the response is written.
func (s *Server) readPointerBody(w http.ResponseWriter, r *http.Request, allow func(time.Time) bool, max int64) ([]byte, func(), bool) {
	if on, _ := s.opts.Pointer.On(); !on {
		http.Error(w, refusePointerOff, http.StatusForbidden)
		return nil, nil, false
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return nil, nil, false
	}
	if !s.opts.Pointer.enter() {
		http.Error(w, refuseBusy, http.StatusTooManyRequests)
		return nil, nil, false
	}
	release := s.opts.Pointer.leave
	now := time.Now()
	if s.opts.Now != nil {
		now = s.opts.Now()
	}
	if !allow(now) {
		release()
		http.Error(w, refuseBusy, http.StatusTooManyRequests)
		return nil, nil, false
	}
	// A body must arrive promptly; a client trickling one doesn't get to
	// hold a slot.
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(pointReadBudget))
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

// point takes one batch from a pointer page. Only well-formed input is
// considered; its errors are fixed text, never the request.
func (s *Server) point(w http.ResponseWriter, r *http.Request) {
	body, release, ok := s.readPointerBody(w, r, s.opts.Pointer.allow, MaxPointBody)
	if !ok {
		return
	}
	defer release()
	batch, err := ParsePointBatch(body)
	if err != nil {
		http.Error(w, "bad point batch", http.StatusBadRequest)
		return
	}
	if s.opts.OnPoint != nil {
		s.opts.OnPoint(batch)
	}
	// Accepted or dropped (stale grid, private screen), the page is told the
	// same: whether a point landed isn't its business.
	w.WriteHeader(http.StatusNoContent)
}

// note opens one sticky note (#447). Unlike a point, its author is told
// whether it was saved, and why not.
func (s *Server) note(w http.ResponseWriter, r *http.Request) {
	if s.opts.OnNote == nil {
		http.NotFound(w, r)
		return
	}
	body, release, ok := s.readPointerBody(w, r, s.opts.Pointer.allowNote, MaxNoteBody)
	if !ok {
		return
	}
	defer release()
	n, err := ParseNote(body)
	if err == nil {
		n.Note.Owner = NoteOwnerFromContext(r.Context())
		err = s.opts.OnNote(n)
	}
	refuseNoteErr(w, err)
}

func (s *Server) noteEdit(w http.ResponseWriter, r *http.Request) {
	if s.opts.OnNoteEdit == nil {
		http.NotFound(w, r)
		return
	}
	body, release, ok := s.readPointerBody(w, r, s.opts.Pointer.allowNote, MaxNoteBody)
	if !ok {
		return
	}
	defer release()
	in, err := ParseNoteEditRequest(body)
	if err == nil {
		in.Owner = NoteOwnerFromContext(r.Context())
		err = s.opts.OnNoteEdit(in)
	}
	refuseNoteErr(w, err)
}

// resolve removes one note; an unknown id is already resolved.
func (s *Server) resolve(w http.ResponseWriter, r *http.Request) {
	if s.opts.OnResolve == nil {
		http.NotFound(w, r)
		return
	}
	body, release, ok := s.readPointerBody(w, r, s.opts.Pointer.allowNote, MaxNoteBody)
	if !ok {
		return
	}
	defer release()
	in, err := ParseNoteResolution(body)
	if err == nil {
		err = s.opts.OnResolve(in)
	}
	refuseNoteErr(w, err)
}

// refuseNoteErr answers a note or resolve request: 204, or a fixed refusal.
func refuseNoteErr(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrPointerOff):
		http.Error(w, refusePointerOff, http.StatusForbidden)
	case errors.Is(err, ErrNoteOwner):
		http.Error(w, "Only the author can edit this note in its original browser session.", http.StatusForbidden)
	case errors.Is(err, ErrNoteGone):
		http.Error(w, "That note is gone.", http.StatusConflict)
	case errors.Is(err, ErrNoteColorFull):
		http.Error(w, refuseColorFull, http.StatusConflict)
	case errors.Is(err, ErrNoteOverlap):
		http.Error(w, refuseOverlap, http.StatusConflict)
	case errors.Is(err, ErrScreenChanged):
		http.Error(w, refuseScreen, http.StatusConflict)
	default:
		http.Error(w, refuseNote, http.StatusBadRequest)
	}
}

// wireNotes is the notes event: every open note, and the width of the
// broadcast's controls on the last row, which no note covers. The viewer
// uses it for hit-testing instead of hardcoding it.
type wireNotes struct {
	Guard int    `json:"guard"`
	Notes []Note `json:"notes"`
}

type wireFrame struct {
	ChromeRows int    `json:"chrome_rows"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
	B          string `json:"b"`
}

type wireEnd struct {
	Reason string `json:"reason"`
}

func (s *Server) events(w http.ResponseWriter, r *http.Request, pointer bool) {
	sub, err := s.opts.Hub.Subscribe()
	switch {
	case errors.Is(err, ErrTooManyViewers):
		http.Error(w, "too many viewers", http.StatusServiceUnavailable)
		return
	case err != nil:
		http.Error(w, "broadcast ended", http.StatusGone)
		return
	}
	defer sub.Close()
	stream := OpenEventStream(w, s.opts.Ping)
	if stream == nil {
		return
	}
	if s.opts.Theme != nil {
		if !stream.Event("theme", s.opts.Theme()) {
			return
		}
	}
	// A pointer page learns whether pointing is on when it joins and at
	// every flip; a view page never hears of it.
	var capsChanged <-chan struct{}
	color := 0
	if pointer {
		color = s.opts.Pointer.claimColor()
		defer s.opts.Pointer.releaseColor(color)
	}
	sendCaps := func() bool {
		on, changed := s.opts.Pointer.On()
		capsChanged = changed
		return stream.Event("caps", wireCaps{Pointer: on, Color: color, Palette: PointerPalette()})
	}
	if pointer && !sendCaps() {
		return
	}
	// Every page, view and pointer alike, sees the notes (#447): the list
	// on join, and again after every change.
	var notesChanged <-chan struct{}
	sendNotes := func() bool {
		notes, _, changed := s.opts.Pointer.Notes()
		notesChanged = changed
		return stream.Event("notes", wireNotes{Guard: StatusGuardCols, Notes: notes})
	}
	if s.opts.Pointer != nil && !sendNotes() {
		return
	}
	ping := time.NewTicker(s.opts.Ping)
	defer ping.Stop()
	for {
		select {
		case m, open := <-sub.Messages():
			if !open {
				stream.Event("end", wireEnd{Reason: EndReason(s.opts.Hub.Err())})
				return
			}
			if !stream.Event("frame", wireFrame{Cols: m.Cols, Rows: m.Rows, ChromeRows: m.ChromeRows, B: base64.StdEncoding.EncodeToString(m.Data)}) {
				return
			}
		case <-capsChanged:
			if !sendCaps() {
				return
			}
		case <-notesChanged:
			if !sendNotes() {
				return
			}
		case <-ping.C:
			// A failed ping ends this viewer, so a vanished one frees its slot.
			if !stream.Comment("ping") {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

// endReasons is the whole viewer-facing vocabulary for why a broadcast ended.
// The reason crosses to remote viewers, so it is a closed set of fixed
// strings: error text, which can carry local addresses and socket paths, never
// leaves the machine.
var endReasons = []struct {
	err  error
	text string
}{
	{ErrHubClosed, "the operator stopped broadcasting"},
	{ErrIndicatorHidden, "the LIVE indicator was not visible on the operator's screen"},
	{ErrTunnelExited, "the tunnel closed"},
	{ErrServerFailed, "the broadcast server stopped"},
}

// EndReason maps why the hub ended to its viewer-facing text.
func EndReason(err error) string {
	if err == nil {
		return endReasons[0].text
	}
	for _, r := range endReasons {
		if errors.Is(err, r.err) {
			return r.text
		}
	}
	return "the broadcast ended"
}
