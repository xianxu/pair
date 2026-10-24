package webcast

import (
	"github.com/xianxu/pair/cmd/internal/castlock"

	"embed"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

//go:embed web/index.html web/viewer.js web/viewer.css web/model.js web/sketch-model.js web/render.js web/css.js web/annotate.js web/floating-panel.js web/feedback-entries.js web/feedback-layout.js web/feedback-glow.js web/feedback-panel.js web/feedback-panel.css web/cast-tools.js web/cast-tools.css web/annotation-policy.js web/annotation-anchor.js web/annotation-placement.js web/capture-draft.js web/annotate.css web/placeholders.js web/drive.js web/drive.css web/replay.js web/replay.css web/replayer.js web/fidelity.html web/fidelity.js web/sketch-image.html web/sketch-image.js web/sketch-page-image.js web/sketch-policy.js web/sketch-render.js web/sketch-delivery.js web/sketch-scene.js web/sketch.js web/sketch-notes.js web/sketch.css web/feedback-mask.js web/feedback-visibility.js
var webFS embed.FS

// contentSecurityPolicy: the page runs only its own script, talks only to
// its own origin, and loads nothing else. The page frame inherits it, so the
// recorded page's inline styles, the viewer's blob: stylesheets and the
// proxied images and fonts (same origin) are allowed; its scripts are not,
// twice over (this policy and the frame's sandbox).
const contentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline' blob:; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'"

var assets = map[string]struct{ file, contentType string }{
	"feedback-mask.js":     {"web/feedback-mask.js", "text/javascript; charset=utf-8"},
	"sketch-image.html":    {"web/sketch-image.html", "text/html; charset=utf-8"},
	"sketch-image.js":      {"web/sketch-image.js", "text/javascript; charset=utf-8"},
	"sketch-page-image.js": {"web/sketch-page-image.js", "text/javascript; charset=utf-8"},
	"sketch-policy.js":     {"web/sketch-policy.js", "text/javascript; charset=utf-8"},
	"sketch-render.js":     {"web/sketch-render.js", "text/javascript; charset=utf-8"},
	"sketch-delivery.js":   {"web/sketch-delivery.js", "text/javascript; charset=utf-8"},
	"sketch-scene.js":      {"web/sketch-scene.js", "text/javascript; charset=utf-8"},
	"sketch-notes.js":      {"web/sketch-notes.js", "text/javascript; charset=utf-8"},
	"sketch.js":            {"web/sketch.js", "text/javascript; charset=utf-8"},
	"sketch.css":           {"web/sketch.css", "text/css; charset=utf-8"},

	"":                {"web/index.html", "text/html; charset=utf-8"},
	"viewer.js":       {"web/viewer.js", "text/javascript; charset=utf-8"},
	"viewer.css":      {"web/viewer.css", "text/css; charset=utf-8"},
	"model.js":        {"web/model.js", "text/javascript; charset=utf-8"},
	"sketch-model.js": {"web/sketch-model.js", "text/javascript; charset=utf-8"},
	"render.js":       {"web/render.js", "text/javascript; charset=utf-8"},
	"css.js":          {"web/css.js", "text/javascript; charset=utf-8"},
	// Pointing and notes (#470): this package's module and style, and the
	// swatches and dialog shared with the terminal viewer.
	"annotation-anchor.js":    {"web/annotation-anchor.js", "text/javascript; charset=utf-8"},
	"capture-draft.js":        {"web/capture-draft.js", "text/javascript; charset=utf-8"},
	"annotation-placement.js": {"web/annotation-placement.js", "text/javascript; charset=utf-8"},
	"annotation-policy.js":    {"web/annotation-policy.js", "text/javascript; charset=utf-8"},
	"floating-panel.js":       {"web/floating-panel.js", "text/javascript; charset=utf-8"},
	"feedback-entries.js":     {"web/feedback-entries.js", "text/javascript; charset=utf-8"},
	"feedback-layout.js":      {"web/feedback-layout.js", "text/javascript; charset=utf-8"},
	"feedback-glow.js":        {"web/feedback-glow.js", "text/javascript; charset=utf-8"},
	"feedback-panel.js":       {"web/feedback-panel.js", "text/javascript; charset=utf-8"},
	"feedback-panel.css":      {"web/feedback-panel.css", "text/css; charset=utf-8"},
	"cast-tools.js":           {"web/cast-tools.js", "text/javascript; charset=utf-8"},
	"cast-tools.css":          {"web/cast-tools.css", "text/css; charset=utf-8"},
	"annotate.js":             {"web/annotate.js", "text/javascript; charset=utf-8"},
	"annotate.css":            {"web/annotate.css", "text/css; charset=utf-8"},
	// The marks for what the recording can't hold (#513).
	"placeholders.js": {"web/placeholders.js", "text/javascript; charset=utf-8"},
	// Remote driving (#467).
	"drive.js":   {"web/drive.js", "text/javascript; charset=utf-8"},
	"drive.css":  {"web/drive.css", "text/css; charset=utf-8"},
	"replay.js":  {"web/replay.js", "text/javascript; charset=utf-8"},
	"replay.css": {"web/replay.css", "text/css; charset=utf-8"},
	// The fidelity check's page (#478): the replay unscaled, no shell.
	"replayer.js":   {"web/replayer.js", "text/javascript; charset=utf-8"},
	"fidelity.html": {"web/fidelity.html", "text/html; charset=utf-8"},
	"fidelity.js":   {"web/fidelity.js", "text/javascript; charset=utf-8"},
}

// endedReason is what viewers are told when the cast ends.
const endedReason = "ended"

// Server is the viewer endpoint, under the cast's token, serving
// the viewer and an event stream of the cast's state ("state", "end") and,
// with a Live, of its recording ("rec": each record line, the catch-up
// first, #460), and the recording's asset bodies at assets/<hash> (#459).
// With a board (WithBoard) it also takes viewers' strokes and notes by POST
// (#470); everything else is GET only.
type Server struct {
	sketches      *board.SketchService
	sketchAsset   func(string, int, int) ([]byte, int, error)
	noteDrafts    *notedraft.Service
	noteOwner     func(*http.Request) string
	noteStreamsMu sync.Mutex
	noteStreams   map[string]int
	token         string
	browser       Browser
	live          recording.Live
	proxy         *castassets.Proxy
	presence      Presence
	ping          time.Duration
	maxViewers    int32
	viewers       atomic.Int32
	// viewViewers and driveViewers: the streams on each link (#467), each
	// its own counter, so neither is derived from two reads.
	viewViewers, driveViewers atomic.Int32
	board                     *boardRoutes // nil: no pointing or notes (WithBoard)
	// pointerToken is the pointer link's (pair#506, WithPointer): "" means
	// the cast has none.
	pointerToken string
	drive        *driveRoutes // nil: no remote driving (WithDrive)
	// locks guard the links (pair#495; WithLocks).
	locks *broadcast.LinkLocks
}

// WithLocks sets the links' code locks; call it before the server is mounted.
func (s *Server) WithLocks(locks *broadcast.LinkLocks) *Server {
	s.locks = locks
	return s
}

// LockFor is the lock guarding segment: the view-only, pointer or drive link's.
func (s *Server) LockFor(segment string) *castlock.Lock { return s.locks.LockFor(segment) }

// Presence is told that viewer id's stream opened; leave is called when it
// closes. It is how remote driving (#466, #467) knows who is watching.
type Presence func(id string) (leave func())

// viewerID is the shape of an events stream's ?viewer=.
var viewerID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// NewServer serves b's state and, when live is set, its records. presence
// may be nil.
func NewServer(token string, b Browser, ping time.Duration, live recording.Live, presence Presence) *Server {
	if ping <= 0 {
		ping = broadcast.DefaultPing
	}
	return &Server{token: token, browser: b, live: live, presence: presence, ping: ping, maxViewers: broadcast.DefaultMaxViewers, proxy: castassets.NewProxy()}
}

// Assets is the cast's asset proxy. The cast is to bind the recording's
// store to it when the recording opens; that wiring lands with #458's
// recorder, and until then the proxy answers 404.
func (s *Server) Assets() *castassets.Proxy { return s.proxy }

// Owns is the Host's routing: this cast answers only its links' tokens
// (view, pointer, drive; linkOf).
func (s *Server) Owns(segment string) bool {
	return s.linkOf(segment) != linkNone
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	broadcast.SetSecurityHeaders(w.Header(), contentSecurityPolicy)
	segment, rest, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	link := s.linkOf(segment)
	// The only routes that read a body are the board's (#470) and remote
	// driving's (#467) POSTs; every
	// other path is GET-only and never reads one.
	if r.Method == http.MethodPost && ok && link != linkNone && (s.sketchRoute(w, r, rest, link) || s.boardRoute(w, r, rest, link) || s.driveRoute(w, r, rest, link.drives())) {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !ok || link == linkNone {
		http.NotFound(w, r)
		return
	}
	if rest == "events" {
		// A pointer-link stream counts as a view stream: only the drive
		// link's streams can hold a seat (pair#506).
		s.events(w, r, link)
		return
	}
	if hash, ok := strings.CutPrefix(rest, "assets/"); ok {
		s.proxy.Serve(w, r, hash)
		return
	}
	if s.boardAsset(w, rest, link) {
		return
	}
	serveViewerFile(w, r, rest)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request, link castLink) {
	driveLink := link.drives()
	id := r.URL.Query().Get("viewer")
	if id != "" && !viewerID.MatchString(id) {
		http.Error(w, "bad viewer id", http.StatusBadRequest)
		return
	}
	if s.viewers.Add(1) > s.maxViewers {
		s.viewers.Add(-1)
		http.Error(w, "too many viewers", http.StatusServiceUnavailable)
		return
	}
	defer s.viewers.Add(-1)
	if link.annotates() {
		defer s.attachNoteStream(r)()
	}
	byLink := &s.viewViewers
	if driveLink {
		byLink = &s.driveViewers
	}
	byLink.Add(1)
	defer byLink.Add(-1)
	if id != "" && s.presence != nil {
		defer s.presence(id)()
	}
	// A seat's stream keeps the seat (#467). An id that is no seat still
	// watches: watching needs none.
	if id != "" && driveLink {
		if detach, err := s.drive.ctrl.Attach(id); err == nil {
			defer detach()
		}
	}
	states, cancel := s.browser.Subscribe()
	defer cancel()
	var annotations, driving <-chan capability.Snapshot
	var feedback <-chan board.LiveFeedbackMask
	if s.board != nil {
		ch, cancel := s.board.board.SubscribeAnnotationCapability()
		annotations = ch
		defer cancel()
		feedback, cancel = s.board.board.SubscribeFeedbackMask()
		defer cancel()
	}
	if s.drive != nil {
		ch, cancel := s.drive.ctrl.SubscribeDriveCapability()
		driving = ch
		defer cancel()
	}
	// When the record channel closes, the fan-out says why (Ended): a viewer
	// dropped as too slow has its stream ended, and reconnects for a fresh
	// catch-up; once the fan-out has ended, the viewer keeps what it has and
	// the stream goes on with state and end.
	var recs, sub <-chan recording.Record
	if s.live != nil {
		var unsubscribe func()
		recs, unsubscribe = s.live.Subscribe()
		sub = recs
		defer unsubscribe()
	}
	stream := broadcast.OpenEventStream(w, s.ping)
	if stream == nil {
		return
	}
	ping := time.NewTicker(s.ping)
	defer ping.Stop()
	var last *wireState
	var lastCapabilities *wireCapabilities
	publishCapabilities := func() bool {
		next := s.viewerCapabilities(link)
		if lastCapabilities != nil && *lastCapabilities == next {
			return true
		}
		lastCapabilities = &next
		return stream.Event("capabilities", next)
	}
	// The subscription owns an initial full mask. Deliver it before any
	// recorded catch-up can paint submitted ink on a reconnecting viewer.
	if feedback != nil {
		mask, open := <-feedback
		if !open {
			feedback = nil
		} else if !stream.Event("feedback", mask) {
			return
		}
	}
	for {
		select {
		case mask, open := <-feedback:
			if !open {
				feedback = nil
				continue
			}
			if !stream.Event("feedback", mask) {
				return
			}
		case _, open := <-annotations:
			if !open {
				annotations = nil
				continue
			}
			if !publishCapabilities() {
				return
			}
		case _, open := <-driving:
			if !open {
				driving = nil
				continue
			}
			if !publishCapabilities() {
				return
			}
		case st, open := <-states:
			if !open || st.Phase == PhaseEnded {
				stream.Event("end", wireEnd{Reason: endedReason})
				return
			}
			next := viewerState(st)
			if last != nil && *last == next {
				continue
			}
			if !stream.Event("state", next) {
				return
			}
			last = &next
		case rec, open := <-recs:
			if !open {
				if !s.live.Ended(sub) {
					return
				}
				recs = nil
				continue
			}
			line, err := recording.Encode(rec)
			if err != nil {
				continue // a record the recorder built can always encode
			}
			if !stream.Event("rec", json.RawMessage(line)) {
				return
			}
		case <-ping.C:
			if !stream.Comment("ping") {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

// Viewers is how many streams are open, by link: the view-only and pointer
// links' together (pair#506), and the drive link's (#467, `couch --broadcast-list`).
func (s *Server) Viewers() (view, drive int) {
	return int(s.viewViewers.Load()), int(s.driveViewers.Load())
}
