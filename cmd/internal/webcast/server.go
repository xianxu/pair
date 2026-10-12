package webcast

import (
	"embed"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

//go:embed web/index.html web/viewer.js web/viewer.css web/model.js web/render.js web/css.js
var webFS embed.FS

// contentSecurityPolicy: the page runs only its own script, talks only to
// its own origin, and loads nothing else. The page frame inherits it, so the
// recorded page's inline styles, the viewer's blob: stylesheets and the
// proxied images and fonts (same origin) are allowed; its scripts are not,
// twice over (this policy and the frame's sandbox).
const contentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline' blob:; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'"

var assets = map[string]struct{ file, contentType string }{
	"":           {"web/index.html", "text/html; charset=utf-8"},
	"viewer.js":  {"web/viewer.js", "text/javascript; charset=utf-8"},
	"viewer.css": {"web/viewer.css", "text/css; charset=utf-8"},
	"model.js":   {"web/model.js", "text/javascript; charset=utf-8"},
	"render.js":  {"web/render.js", "text/javascript; charset=utf-8"},
	"css.js":     {"web/css.js", "text/javascript; charset=utf-8"},
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
	token      string
	browser    Browser
	live       recording.Live
	proxy      *castassets.Proxy
	presence   Presence
	ping       time.Duration
	maxViewers int32
	viewers    atomic.Int32
	board      *boardRoutes // nil: no pointing or notes (WithBoard)
}

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

// Owns is the Host's routing: this cast answers only its token.
func (s *Server) Owns(segment string) bool { return broadcast.OwnsToken(segment, s.token) }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	broadcast.SetSecurityHeaders(w.Header(), contentSecurityPolicy)
	segment, rest, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	// The only routes that read a body are the board's POSTs (#470); every
	// other path is GET-only and never reads one.
	if r.Method == http.MethodPost && ok && s.Owns(segment) && s.boardRoute(w, r, rest) {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !ok || !s.Owns(segment) {
		http.NotFound(w, r)
		return
	}
	if rest == "events" {
		s.events(w, r)
		return
	}
	if hash, ok := strings.CutPrefix(rest, "assets/"); ok {
		s.proxy.Serve(w, r, hash)
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
	w.Header().Set("Content-Type", a.contentType)
	w.Write(body)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
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
	if id != "" && s.presence != nil {
		defer s.presence(id)()
	}
	states, cancel := s.browser.Subscribe()
	defer cancel()
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
	for {
		select {
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
