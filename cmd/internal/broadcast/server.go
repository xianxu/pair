package broadcast

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	Token string
	Hub   *Hub
	Ping  time.Duration
}

type asset struct{ file, contentType string }

// assets is every path served under /<token>/; nothing else is.
var assets = map[string]asset{
	"":           {"web/index.html", "text/html; charset=utf-8"},
	"viewer.js":  {"web/viewer.js", "text/javascript; charset=utf-8"},
	"viewer.css": {"web/viewer.css", "text/css; charset=utf-8"},
	"xterm.js":   {"web/vendor/xterm/xterm.js", "text/javascript; charset=utf-8"},
	"xterm.css":  {"web/vendor/xterm/xterm.css", "text/css; charset=utf-8"},
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

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Security-Policy", contentSecurityPolicy)
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodGet {
		h.Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest, ok := s.authorized(r.URL.Path)
	if !ok {
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
	h.Set("Content-Type", a.contentType)
	w.Write(body)
}

// authorized splits "/<token>/<rest>" and checks the token in constant time.
func (s *Server) authorized(path string) (rest string, ok bool) {
	segment, rest, found := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	if !found || !strings.HasPrefix(path, "/") || s.opts.Token == "" {
		return "", false
	}
	if subtle.ConstantTimeCompare([]byte(segment), []byte(s.opts.Token)) != 1 {
		return "", false
	}
	return rest, true
}

type wireFrame struct {
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
	B    string `json:"b"`
}

type wireEnd struct {
	Reason string `json:"reason"`
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	send := func(chunk string) bool {
		if _, err := fmt.Fprint(w, chunk); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	if !send(": connected\n\n") {
		return
	}
	ping := time.NewTicker(s.opts.Ping)
	defer ping.Stop()
	for {
		select {
		case m, open := <-sub.Messages():
			if !open {
				end, _ := json.Marshal(wireEnd{Reason: endReason(s.opts.Hub.Err())})
				send("event: end\ndata: " + string(end) + "\n\n")
				return
			}
			frame, _ := json.Marshal(wireFrame{Cols: m.Cols, Rows: m.Rows, B: base64.StdEncoding.EncodeToString(m.Data)})
			if !send("event: frame\ndata: " + string(frame) + "\n\n") {
				return
			}
		case <-ping.C:
			// A failed ping ends this viewer, so a vanished one frees its slot.
			if !send(": ping\n\n") {
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

// endReason maps why the hub ended to its viewer-facing text.
func endReason(err error) string {
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
