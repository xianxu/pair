package mockserve

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxFrontRequests = 64

func parseRealOrigin(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid real app origin: %w", err)
	}
	if u.Scheme != "http" || u.User != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return nil, fmt.Errorf("real app must be an HTTP loopback origin with an explicit port and no path, credentials, query or fragment")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, fmt.Errorf("real app origin must be loopback")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("real app origin needs a port from 1 to 65535")
	}
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u, nil
}

// developmentFront owns its transport, active requests and hijacked sockets.
// The external real application's process is never owned or stopped here.
type developmentFront struct {
	mock        *Server
	proxy       *httputil.ReverseProxy
	transport   *http.Transport
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	closed      bool
	active      int
	connections map[*frontConnection]struct{}
	joined      sync.WaitGroup
	closeOnce   sync.Once
}

func newDevelopmentFront(ctx context.Context, mock *Server, rawOrigin string) (*developmentFront, error) {
	origin, err := parseRealOrigin(rawOrigin)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	f := &developmentFront{mock: mock, ctx: ctx, cancel: cancel, connections: map[*frontConnection]struct{}{}}
	// localhost is pinned to loopback, never resolved from request-controlled data.
	dialHost := origin.Hostname()
	if dialHost == "localhost" {
		dialHost = "127.0.0.1"
	}
	dialAddress := net.JoinHostPort(dialHost, origin.Port())
	dialer := &net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}
	f.transport = &http.Transport{DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, dialAddress)
	}, ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 16 << 10, MaxIdleConns: 64, MaxIdleConnsPerHost: 64, IdleConnTimeout: 30 * time.Second, DisableCompression: true}
	f.proxy = &httputil.ReverseProxy{
		Transport: f.transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(origin)
			pr.SetXForwarded()
		},
		ModifyResponse: func(res *http.Response) error {
			location, err := res.Location()
			if err == nil && location.IsAbs() && location.Scheme == origin.Scheme && location.Host == origin.Host {
				location.Scheme, location.Host = "", ""
				res.Header.Set("Location", location.String())
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "Real app unavailable", http.StatusBadGateway)
		},
	}
	return f, nil
}

func validFrontPath(u *url.URL) bool {
	raw := strings.ToLower(u.EscapedPath())
	return strings.HasPrefix(u.Path, "/") && !strings.ContainsAny(u.Path, "\\\x00") && !strings.Contains(raw, "%2f") && !strings.Contains(raw, "%5c") && path.Clean(u.Path) == u.Path
}

func privateFrontPath(name string) bool {
	first, _, _ := strings.Cut(strings.TrimPrefix(name, "/"), "/")
	return strings.HasPrefix(first, ".") || first == "screens.json" || first == "design.md"
}

func readOnlyFront(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "read-only mock document", http.StatusMethodNotAllowed)
	return false
}

func (f *developmentFront) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	obs, catalogue := f.mock.frontObservation()
	decision := decideFrontRoute(r.URL, catalogue, obs.Version)
	switch decision.Kind {
	case routeNotFound:
		http.NotFound(w, r)
		return
	case routeInternal:
		f.mock.ServeHTTP(w, r)
		return
	}
	if obs.Error != "" || catalogue == nil {
		http.Error(w, "Repair mock project: "+obs.Error, 503)
		return
	}
	switch decision.Kind {
	case routeFixtures, routeAsset:
		f.mock.ServeHTTP(w, r)
	case routeReal:
		f.serveReal(w, r)
	case routeRealRedirect:
		target := url.URL{Path: decision.Screen.Route, RawQuery: r.URL.RawQuery}
		code := http.StatusTemporaryRedirect
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			code = http.StatusFound
		}
		http.Redirect(w, r, target.String(), code)
	case routeMockRedirect:
		if readOnlyFront(w, r) {
			f.mock.redirect(w, r, decision.Version, decision.Relative)
		}
	case routeMockDocument:
		if !readOnlyFront(w, r) {
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		f.mock.deliverFile(w, r, decision.Version+"/"+decision.Relative, decision.Relative, obs, catalogue)
	}
}

func (f *developmentFront) serveReal(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	if f.closed || f.active >= maxFrontRequests {
		f.mu.Unlock()
		http.Error(w, "Development front unavailable or busy", 503)
		return
	}
	f.active++
	f.joined.Add(1)
	f.mu.Unlock()
	defer func() { f.mu.Lock(); f.active--; f.mu.Unlock(); f.joined.Done() }()
	ctx, cancel := context.WithCancel(r.Context())
	callbackDone := make(chan struct{})
	stop := context.AfterFunc(f.ctx, func() {
		defer close(callbackDone)
		cancel()
		// Cancellation must unblock both incomplete uploads and slow writers.
		controller := http.NewResponseController(w)
		deadline := time.Now()
		_ = controller.SetReadDeadline(deadline)
		_ = controller.SetWriteDeadline(deadline)
	})
	defer func() {
		if !stop() {
			<-callbackDone
		}
		cancel()
	}()
	f.proxy.ServeHTTP(&frontWriter{ResponseWriter: w, front: f}, r.WithContext(ctx))
}

func (f *developmentFront) Close() error {
	f.closeOnce.Do(func() {
		f.mu.Lock()
		f.closed = true
		connections := make([]*frontConnection, 0, len(f.connections))
		for conn := range f.connections {
			connections = append(connections, conn)
		}
		f.mu.Unlock()
		f.cancel()
		for _, conn := range connections {
			_ = conn.Close()
		}
		f.transport.CloseIdleConnections()
		f.joined.Wait()
	})
	return nil
}

type frontWriter struct {
	http.ResponseWriter
	front *developmentFront
}

func (w *frontWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *frontWriter) Flush()                      { _ = http.NewResponseController(w.ResponseWriter).Flush() }
func (w *frontWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, err
	}
	tracked := &frontConnection{Conn: conn, front: w.front}
	w.front.mu.Lock()
	if w.front.closed {
		w.front.mu.Unlock()
		_ = conn.Close()
		return nil, nil, errors.New("development front closed")
	}
	w.front.connections[tracked] = struct{}{}
	w.front.mu.Unlock()
	return tracked, rw, nil
}

type frontConnection struct {
	net.Conn
	front *developmentFront
	once  sync.Once
	err   error
}

func (c *frontConnection) Close() error {
	c.once.Do(func() { c.err = c.Conn.Close(); c.front.mu.Lock(); delete(c.front.connections, c); c.front.mu.Unlock() })
	return c.err
}

func ServeWithReal(ctx context.Context, dir string, port int, out io.Writer, rawOrigin string) error {
	origin, err := parseRealOrigin(rawOrigin)
	if err != nil {
		return err
	}
	p, err := OpenProject(dir)
	if err != nil {
		return err
	}
	defer p.Close()
	s, err := newHybridServer(ctx, p, strconv.FormatInt(time.Now().UnixNano(), 36))
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return err
	}
	defer ln.Close()
	if origin.Port() == strconv.Itoa(ln.Addr().(*net.TCPAddr).Port) {
		return fmt.Errorf("real app origin points to the development front itself")
	}
	f, err := newDevelopmentFront(ctx, s, rawOrigin)
	if err != nil {
		return err
	}
	defer f.Close()
	s.progressOut = out
	return serveHandler(ctx, s, f, ln, out, 0)
}
