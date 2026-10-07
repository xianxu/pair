package broadcast

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"net/http"
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
}

// Session is one broadcast: a hub, a server on the tunnel's listener, the
// tunnel, and the token that is the link's only credential. Everything it
// owns dies with it; nothing is written to disk.
type Session struct {
	token  string
	link   string
	hub    *Hub
	srv    *http.Server
	handle Handle

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
	hub := NewHub(cfg.Hub)
	srv := &http.Server{
		Handler:           NewServer(ServerOptions{Token: token, Hub: hub, Ping: cfg.Ping}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go srv.Serve(l)
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
	if err := probe(ctx, link, cfg.ProbeTimeout); err != nil {
		return abandon(handle, err)
	}
	s := &Session{token: token, link: link, hub: hub, srv: srv, handle: handle, done: make(chan struct{})}
	go s.watch()
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

func (s *Session) watch() {
	select {
	case <-s.hub.Done():
		s.Stop(s.hub.Err())
	case <-s.handle.Exited():
		s.Stop(ErrTunnelExited)
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
func probe(ctx context.Context, link string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := &http.Client{Timeout: 5 * time.Second}
	var last error
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
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
		case <-time.After(probeInterval):
		}
	}
}
