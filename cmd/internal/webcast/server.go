package webcast

import (
	"embed"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

//go:embed web/index.html web/viewer.js web/viewer.css
var webFS embed.FS

// contentSecurityPolicy: the page runs only its own script, talks only to
// its own origin, and loads nothing else.
const contentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-ancestors 'none'"

var assets = map[string]struct{ file, contentType string }{
	"":           {"web/index.html", "text/html; charset=utf-8"},
	"viewer.js":  {"web/viewer.js", "text/javascript; charset=utf-8"},
	"viewer.css": {"web/viewer.css", "text/css; charset=utf-8"},
}

// endedReason is what viewers are told when the cast ends.
const endedReason = "ended"

// Server is the stub viewer endpoint (#454): GET only, under the cast's
// token, serving the page and an event stream of the cast's state. The
// recorder's DOM stream replaces the state events' content later; the route
// and the seam stay.
type Server struct {
	token      string
	browser    Browser
	ping       time.Duration
	maxViewers int32
	viewers    atomic.Int32
}

func NewServer(token string, b Browser, ping time.Duration) *Server {
	if ping <= 0 {
		ping = broadcast.DefaultPing
	}
	return &Server{token: token, browser: b, ping: ping, maxViewers: broadcast.DefaultMaxViewers}
}

// Owns is the Host's routing: this cast answers only its token.
func (s *Server) Owns(segment string) bool { return broadcast.OwnsToken(segment, s.token) }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	broadcast.SetSecurityHeaders(w.Header(), contentSecurityPolicy)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	segment, rest, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if !ok || !s.Owns(segment) {
		http.NotFound(w, r)
		return
	}
	if rest == "events" {
		s.events(w, r)
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
	if s.viewers.Add(1) > s.maxViewers {
		s.viewers.Add(-1)
		http.Error(w, "too many viewers", http.StatusServiceUnavailable)
		return
	}
	defer s.viewers.Add(-1)
	states, cancel := s.browser.Subscribe()
	defer cancel()
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
		case <-ping.C:
			if !stream.Comment("ping") {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}
