package broadcast

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// Theme, when set, is read for each new viewer and sent before its
	// first frame.
	Theme func() Theme
	// Pointer is the broadcast's pointer capability (#412); nil serves no
	// pointer link. OnPoint receives each well-formed batch posted while
	// pointing is on; it decides whether the batch lands.
	Pointer *PointerState
	OnPoint func(PointBatch)
}

type asset struct{ file, contentType string }

// assets is every path served under /<token>/; nothing else is.
var assets = map[string]asset{
	"":           {"web/index.html", "text/html; charset=utf-8"},
	"viewer.js":  {"web/viewer.js", "text/javascript; charset=utf-8"},
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
	segment, rest, ok := splitPath(r.URL.Path)
	pointer := ok && s.opts.Pointer.Match(segment)
	// The one route that reads a body: POST /<pointer-token>/point. Every
	// other path is GET-only and never reads one (#395).
	if r.Method == http.MethodPost && pointer && rest == "point" {
		s.point(w, r)
		return
	}
	if r.Method != http.MethodGet {
		h.Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	view := ok && s.opts.Token != "" && subtle.ConstantTimeCompare([]byte(segment), []byte(s.opts.Token)) == 1
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

type wireCaps struct {
	Pointer bool `json:"pointer"`
}

// point takes one batch from a pointer page. Only well-formed input is
// considered; its errors are fixed text, never the request.
func (s *Server) point(w http.ResponseWriter, r *http.Request) {
	if on, _ := s.opts.Pointer.On(); !on {
		http.Error(w, "pointing is off", http.StatusForbidden)
		return
	}
	if mt := r.Header.Get("Content-Type"); mt != "application/json" && !strings.HasPrefix(mt, "application/json;") {
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}
	if !s.opts.Pointer.enter() {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	defer s.opts.Pointer.leave()
	if !s.opts.Pointer.allow(time.Now()) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	// A body must arrive promptly; a client trickling one doesn't get to
	// hold a slot.
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(pointReadBudget))
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxPointBody))
	if err != nil {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
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

type wireFrame struct {
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
	B    string `json:"b"`
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
	if s.opts.Theme != nil {
		theme, _ := json.Marshal(s.opts.Theme())
		if !send("event: theme\ndata: " + string(theme) + "\n\n") {
			return
		}
	}
	// A pointer page learns whether pointing is on when it joins and at
	// every flip; a view page never hears of it.
	var capsChanged <-chan struct{}
	sendCaps := func() bool {
		on, changed := s.opts.Pointer.On()
		capsChanged = changed
		caps, _ := json.Marshal(wireCaps{Pointer: on})
		return send("event: caps\ndata: " + string(caps) + "\n\n")
	}
	if pointer && !sendCaps() {
		return
	}
	ping := time.NewTicker(s.opts.Ping)
	defer ping.Stop()
	for {
		select {
		case m, open := <-sub.Messages():
			if !open {
				end, _ := json.Marshal(wireEnd{Reason: EndReason(s.opts.Hub.Err())})
				send("event: end\ndata: " + string(end) + "\n\n")
				return
			}
			frame, _ := json.Marshal(wireFrame{Cols: m.Cols, Rows: m.Rows, B: base64.StdEncoding.EncodeToString(m.Data)})
			if !send("event: frame\ndata: " + string(frame) + "\n\n") {
				return
			}
		case <-capsChanged:
			if !sendCaps() {
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
