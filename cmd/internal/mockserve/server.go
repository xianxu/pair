package mockserve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const statePath = "/__couch_mock/state"
const reloadPath = "/__couch_mock/reload.js"

// Server publishes one scan observation to every browser. Requests never scan.
type Server struct {
	project  *Project
	lifetime string
	scan     func(context.Context) (Snapshot, error)
	mu       sync.RWMutex
	state    RevisionState
}

func NewServer(ctx context.Context, p *Project, lifetime string) (*Server, error) {
	s := &Server{project: p, lifetime: lifetime, scan: p.Scan}
	snap, err := s.scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("repair mock project: %w", err)
	}
	s.state.Observe(snap.Version, snap.Digest, nil)
	return s, nil
}
func (s *Server) refresh(ctx context.Context) {
	snap, err := s.scan(ctx)
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Observe(snap.Version, snap.Digest, err)
}
func (s *Server) observation() Observation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Snapshot(s.lifetime)
}
func (s *Server) watch(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refresh(ctx)
		}
	}
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "read-only mock server", http.StatusMethodNotAllowed)
		return
	}
	obs := s.observation() // baseline precedes document IO, including first-poll races.
	if r.URL.Path == statePath {
		w.Header().Set("Content-Type", "application/json")
		if obs.Error != "" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		if r.Method != http.MethodHead {
			_ = json.NewEncoder(w).Encode(obs)
		}
		return
	}
	if r.URL.Path == reloadPath {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		if r.Method != http.MethodHead {
			_, _ = w.Write(ReloadScript)
		}
		return
	}
	if obs.Error != "" {
		http.Error(w, "Repair mock project: "+obs.Error, http.StatusServiceUnavailable)
		return
	}
	raw := strings.ToLower(r.URL.EscapedPath())
	name := strings.TrimPrefix(r.URL.Path, "/")
	if strings.Contains(raw, "%2f") || strings.Contains(raw, "%5c") || strings.ContainsAny(name, "\\\x00") || path.Clean("/"+name) != "/"+name {
		http.NotFound(w, r)
		return
	}
	if name == "" {
		s.redirect(w, r, obs.Version, "index.html")
		return
	}
	if name == "fixtures.json" {
		s.file(w, r, name, "", obs)
		return
	}
	version, screen, ok := strings.Cut(name, "/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, err := ParseVersion(version); err != nil {
		http.NotFound(w, r)
		return
	}
	if screen == "" {
		s.redirect(w, r, obs.Version, "index.html")
		return
	}
	if version != obs.Version {
		if !s.project.existsScreen(version, "index.html") {
			http.NotFound(w, r)
			return
		}
		if !s.project.existsScreen(obs.Version, screen) {
			screen = "index.html"
		}
		s.redirect(w, r, obs.Version, screen)
		return
	}
	if strings.HasSuffix(screen, ".html") && !s.project.existsScreen(version, screen) {
		if screen == "index.html" {
			http.NotFound(w, r)
			return
		}
		s.redirect(w, r, version, "index.html")
		return
	}
	s.file(w, r, version+"/"+screen, screen, obs)
}
func (s *Server) redirect(w http.ResponseWriter, r *http.Request, version, screen string) {
	u := url.URL{Path: "/" + version + "/" + screen, RawQuery: r.URL.RawQuery}
	http.Redirect(w, r, u.String(), http.StatusFound)
}
func (s *Server) file(w http.ResponseWriter, r *http.Request, name, screen string, obs Observation) {
	limit := MaxBytes
	if strings.HasSuffix(screen, ".html") {
		limit = MaxHTMLBytes
	}
	body, err := s.project.readBounded(name, limit)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	contentType := mime.TypeByExtension(path.Ext(name))
	if strings.HasPrefix(contentType, "text/html") && !strings.HasSuffix(screen, ".html") {
		http.Error(w, "mock screens must use the .html extension", http.StatusUnsupportedMediaType)
		return
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if screen != "" && strings.HasSuffix(screen, ".html") {
		id, err := ScreenFromPath(screen)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		body, err = TransformPage(body, Bootstrap{Version: obs.Version, Screen: id, Revision: obs.Revision})
		if err != nil {
			http.Error(w, "cannot transform mock page: "+err.Error(), 500)
			return
		}
		contentType = "text/html; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

// Serve owns the listener, scanner and project handle until all have stopped.
func Serve(ctx context.Context, dir string, port int, out io.Writer) error {
	p, err := OpenProject(dir)
	if err != nil {
		return err
	}
	defer p.Close()
	lifetime := strconv.FormatInt(time.Now().UnixNano(), 36)
	s, err := NewServer(ctx, p, lifetime)
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return err
	}
	return serveListener(ctx, s, ln, out)
}
func serveListener(ctx context.Context, s *Server, ln net.Listener, out io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	srv := &http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	served := make(chan error, 1)
	serveDone := make(chan struct{})
	go func() { defer close(serveDone); served <- srv.Serve(ln) }()
	watched := make(chan struct{})
	go func() { defer close(watched); s.watch(ctx) }()
	_, writeErr := fmt.Fprintln(out, "http://"+ln.Addr().String()+"/")
	var runErr error
	if writeErr != nil {
		runErr = writeErr
	} else {
		select {
		case <-ctx.Done():
		case runErr = <-served:
		}
	}
	cancel()
	shutdown, stop := context.WithTimeout(context.Background(), 2*time.Second)
	if err := srv.Shutdown(shutdown); err != nil {
		_ = srv.Close()
		if runErr == nil {
			runErr = err
		}
	}
	stop()
	_ = ln.Close()
	<-watched
	<-serveDone
	// Serve's completion is buffered even when already received above.
	if errors.Is(runErr, http.ErrServerClosed) {
		return nil
	}
	return runErr
}
