package broadcast

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

const (
	DefaultProbeTimeout = 30 * time.Second
	probeInterval       = 500 * time.Millisecond
	shutdownBudget      = 2 * time.Second
)

// Config starts a Session.
type Config struct {
	Tunnel Tunnel
	Hub    HubOptions
	// Ping is the SSE keepalive interval; zero means DefaultPing.
	Ping time.Duration
	// ProbeTimeout bounds waiting for the public link to answer.
	ProbeTimeout time.Duration
	// Resolve looks up the link's hostname for the probe; nil uses a public
	// resolver, then the system one (see PublicResolve).
	Resolve func(ctx context.Context, host string) ([]string, error)
	// Theme supplies the operator's palette for each new viewer; nil sends
	// none and viewers keep xterm.js's colours.
	Theme func() Theme
	// OnPoints receives each pointer batch that lands: pointing on, the
	// operator's screen at the batch's grid, public, and showing the active
	// pointer marker (#412). It is called on a request goroutine, holding no
	// session lock.
	OnPoints func(PointBatch)
	// OnPointerOff is called when the pointer watch turned pointing off
	// because the active marker stayed hidden; not on the operator's own
	// DisablePointer.
	OnPointerOff func()
}

// Session is one broadcast: a hub, a server on the tunnel's listener, the
// tunnel, and the token that is the link's only credential. Everything it
// owns dies with it; nothing is written to disk.
type Session struct {
	token   string
	link    string
	base    string // the public base URL, for the pointer link
	cfg     Config
	pointer *PointerState
	hub     *Hub
	srv     *http.Server
	handle  Handle

	stopOnce sync.Once
	reason   error
	done     chan struct{}
}

// Start opens a broadcast and returns once its link answers. Cancelling ctx
// abandons the start; whatever was opened, including a tunnel that finishes
// opening after the cancel, is closed.
func Start(ctx context.Context, cfg Config) (*Session, error) {
	if cfg.Tunnel == nil {
		return nil, errors.New("broadcast: no tunnel configured")
	}
	if cfg.ProbeTimeout <= 0 {
		cfg.ProbeTimeout = DefaultProbeTimeout
	}
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	l, err := cfg.Tunnel.Listen()
	if err != nil {
		return nil, fmt.Errorf("broadcast: listen: %w", err)
	}
	// The session is built before the hub and server, which call back into
	// it; the pointer can't be used until Start returns (no token is minted
	// before EnablePointer).
	s := &Session{token: token, cfg: cfg, pointer: newPointerState(), done: make(chan struct{})}
	hubOpts := cfg.Hub
	// The hub calls this on its own goroutine, which must not block.
	// It records the flip generation it fired under, so a report that
	// arrives after the operator flipped pointing again is ignored.
	hubOpts.OnPointerHidden = func() {
		gen := s.pointer.generation()
		go s.pointerHidden(gen)
	}
	hub := NewHub(hubOpts)
	srv := &http.Server{
		Handler: NewServer(ServerOptions{Token: token, Hub: hub, Ping: cfg.Ping, Theme: cfg.Theme,
			Pointer: s.pointer, OnPoint: s.acceptPoint}),
		// Bounds on what an internet client can hold open: headers must
		// arrive promptly, and the one body (the pointer POST) gets its own
		// read deadline in its handler. No server-wide ReadTimeout: it would
		// also time out the disconnect watch on long-lived event streams.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(l) }()
	abandon := func(handle Handle, err error) (*Session, error) {
		hub.Close(err)
		srv.Close()
		if handle != nil {
			handle.Close()
		}
		return nil, err
	}
	handle, err := cfg.Tunnel.Open(ctx, l)
	if err != nil {
		return abandon(nil, err)
	}
	if err := ctx.Err(); err != nil {
		return abandon(handle, err)
	}
	link := handle.URL() + "/" + token + "/"
	if err := probe(ctx, link, cfg.ProbeTimeout, cfg.Resolve, handle.Exited()); err != nil {
		return abandon(handle, err)
	}
	s.link, s.base, s.hub, s.srv, s.handle = link, handle.URL(), hub, srv, handle
	go s.watch(served)
	return s, nil
}

// Link is the viewer URL. It carries the token: share it, never draw it.
func (s *Session) Link() string { return s.link }

// Offer hands the session a frame from the Presenter tap. Never blocks.
func (s *Session) Offer(f terminal.Frame, class terminal.FrameClass) { s.hub.Offer(f, class) }

// Activate starts the LIVE indicator watch; see Hub.Activate.
func (s *Session) Activate() { s.hub.Activate() }

// Stop ends the broadcast with reason (nil: the operator stopped it). Viewers
// are told at once; the listener and tunnel close in the background, and Done
// closes when they have. Idempotent: the first reason wins.
func (s *Session) Stop(reason error) {
	s.stopOnce.Do(func() {
		s.pointer.stop()
		s.hub.Close(reason)
		s.reason = s.hub.Err()
		go s.teardown()
	})
}

// Done closes once the broadcast has fully ended.
func (s *Session) Done() <-chan struct{} { return s.done }

// Err is why the broadcast ended; nil while it runs.
func (s *Session) Err() error {
	select {
	case <-s.done:
		return s.reason
	default:
		return nil
	}
}

// watch ends the session when any part it owns ends on its own: the hub
// (indicator hidden), the tunnel, or the HTTP server. Serve returns
// ErrServerClosed after our own teardown, which by then is a no-op Stop.
func (s *Session) watch(served <-chan error) {
	select {
	case <-s.hub.Done():
		s.Stop(s.hub.Err())
	case <-s.handle.Exited():
		s.Stop(ErrTunnelExited)
	case err := <-served:
		if !errors.Is(err, http.ErrServerClosed) {
			s.Stop(fmt.Errorf("%w: %v", ErrServerFailed, err))
		}
	}
}

func (s *Session) teardown() {
	defer close(s.done)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownBudget)
	defer cancel()
	// The hub is closed, so every event stream is already returning.
	if err := s.srv.Shutdown(ctx); err != nil {
		s.srv.Close()
	}
	s.handle.Close()
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("broadcast: token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// probe waits until the link answers 200, so "link copied" means it works.
// A quick tunnel's edge starts serving a few seconds after printing its URL.
//
// The hostname is resolved by resolve, not the system resolver: a new
// quick-tunnel hostname didn't resolve through macOS's resolver for over a
// minute while 1.1.1.1 had it in a second, and a failed system lookup may be
// cached, which would also delay the operator's own browser.
//
// It gives up at once if the tunnel exits meanwhile; probing a dead tunnel for
// the full timeout would hide why the start failed.
func probe(ctx context.Context, link string, timeout time.Duration, resolve func(context.Context, string) ([]string, error), exited <-chan struct{}) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if resolve == nil {
		resolve = PublicResolve
	}
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DialContext: resolvingDialer(resolve)}}
	var last error
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		// A *url.Error's text carries the link, and so the token; keep only
		// its cause, since this error reaches the operator's notice.
		var uerr *url.Error
		if errors.As(err, &uerr) {
			err = uerr.Err
		}
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			err = fmt.Errorf("status %d", resp.StatusCode)
		}
		last = err
		select {
		case <-ctx.Done():
			if parent := context.Cause(ctx); errors.Is(parent, context.Canceled) {
				return parent
			}
			return fmt.Errorf("broadcast: link never answered: %w", last)
		case <-exited:
			return ErrTunnelExited
		case <-time.After(probeInterval):
		}
	}
}

// publicResolvers are asked directly, bypassing the system's resolver.
var publicResolvers = []string{"1.1.1.1:53", "1.0.0.1:53"}

func resolverAt(server string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, server)
		},
	}
}

// PublicResolve resolves host through Cloudflare's public resolvers. Their
// "not found" stands: a quick-tunnel hostname is briefly unknown everywhere
// while it propagates (about 1.5s, measured), and asking the system then
// would only teach it a negative answer to cache. The system resolver is used
// only when no public resolver can be reached at all.
func PublicResolve(ctx context.Context, host string) ([]string, error) {
	var notFound error
	for _, server := range publicResolvers {
		addrs, err := resolverAt(server).LookupHost(ctx, host)
		if err == nil && len(addrs) > 0 {
			return addrs, nil
		}
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			notFound = err
		}
	}
	if notFound != nil {
		return nil, notFound
	}
	return net.DefaultResolver.LookupHost(ctx, host)
}

// resolvingDialer dials a hostname through resolve; IP literals are dialed
// as they are.
func resolvingDialer(resolve func(context.Context, string) ([]string, error)) func(context.Context, string, string) (net.Conn, error) {
	var d net.Dialer
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil || net.ParseIP(host) != nil {
			return d.DialContext(ctx, network, addr)
		}
		ips, err := resolve(ctx, host)
		if err != nil {
			return nil, err
		}
		var last error = errors.New("no addresses")
		for _, ip := range ips {
			conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip, port))
			if err == nil {
				return conn, nil
			}
			last = err
		}
		return nil, last
	}
}

// EnablePointer turns pointing on (#412), minting the pointer link on first
// use; it returns the link, the same one on every later call.
func (s *Session) EnablePointer() (string, error) {
	token, changed, err := s.pointer.set(true, newToken)
	if err != nil {
		return "", err
	}
	if changed {
		s.hub.ArmPointer()
	}
	return s.base + "/" + token + "/", nil
}

// DisablePointer turns pointing off: the pointer link stays valid as
// view-only, and its open pages are told.
func (s *Session) DisablePointer() {
	if _, changed, _ := s.pointer.set(false, nil); changed {
		s.hub.DisarmPointer()
	}
}

// PointerLink is the pointer link, or "" before EnablePointer.
func (s *Session) PointerLink() string {
	if token := s.pointer.link(); token != "" {
		return s.base + "/" + token + "/"
	}
	return ""
}

// pointerHidden turns pointing off because the active marker stayed off the
// operator's screen (the hub has already disarmed its watch).
func (s *Session) pointerHidden(gen uint64) {
	if s.pointer.offIfGeneration(gen) && s.cfg.OnPointerOff != nil {
		s.cfg.OnPointerOff()
	}
}

// acceptPoint passes on a batch only if it lands on what the operator sees
// now: pointing on, the same grid, a public screen, and the active pointer
// marker drawn. Anything else is dropped.
func (s *Session) acceptPoint(b PointBatch) {
	if on, _ := s.pointer.On(); !on {
		return
	}
	cur := s.hub.Current()
	if !cur.OK || cur.Geometry != (terminal.Geometry{Cols: b.Cols, Rows: b.Rows}) || cur.Class != terminal.FramePublic || !cur.PointerShown {
		return
	}
	if s.cfg.OnPoints != nil {
		s.cfg.OnPoints(b)
	}
}
