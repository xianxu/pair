package broadcast

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/castlock"
)

// HostConfig opens a Host.
type HostConfig struct {
	Tunnel Tunnel
	// ProbeTimeout bounds waiting for a mount's link to answer.
	ProbeTimeout time.Duration
	// Resolve looks up the link's hostname for the probe; nil uses
	// PublicResolve.
	Resolve func(ctx context.Context, host string) ([]string, error)
}

// Mountable is a handler that answers its own first path segments (its
// capability tokens). LockFor names the lock guarding an owned segment, or nil
// for a link served without one (pair#495); a mount creates its locks with its
// tokens, before Mount, so no owned segment is ever served unlocked by
// accident, and they end with it.
type Mountable interface {
	http.Handler
	Owns(segment string) bool
	LockFor(segment string) *castlock.Lock
}

// Host is the public endpoint every cast shares (#454): one listener, one
// HTTP server and one tunnel, opened by the first Mount and closed after the
// last Unmount. A named tunnel can be open only once at a time, so a screen
// broadcast and a web cast running together must share it; each is mounted
// under its own tokens and the Host routes by the first path segment.
//
// Its states form one tagged value under mu: closed → opening → open →
// closing → closed. A Mount while opening waits for that open; while closing,
// for the close, then opens again. An open nobody waits for any more is
// cancelled, or closed as soon as it completes.
type Host struct {
	cfg HostConfig

	mu      sync.Mutex
	state   hostState
	attempt *hostAttempt // the open or close in progress
	waiting int          // Mounts waiting on an open
	mounts  map[*Mount]struct{}
	gen     uint64 // which open the server and handle belong to
	srv     *http.Server
	handle  Handle
	mode    string
}

type hostState uint8

const (
	hostClosed hostState = iota
	hostOpening
	hostOpen
	hostClosing
)

// hostAttempt is one open or close; done closes when it has settled, with
// err set for a failed open.
type hostAttempt struct {
	done   chan struct{}
	err    error
	cancel context.CancelFunc
	// abandoned: everyone gave up on this open. It can't be joined: its
	// context is cancelled, so its error would be another caller's.
	abandoned bool
}

// NewHost makes a closed Host; nothing is opened until the first Mount.
func NewHost(cfg HostConfig) *Host {
	if cfg.ProbeTimeout <= 0 {
		cfg.ProbeTimeout = DefaultProbeTimeout
	}
	return &Host{cfg: cfg, mounts: map[*Mount]struct{}{}}
}

// Mount serves m on the Host, opening it if needed, and returns once
// m's link answers at probePath (relative to the public base). Cancelling
// ctx abandons the mount.
func (h *Host) Mount(ctx context.Context, m Mountable, probePath string) (*Mount, error) {
	if h.cfg.Tunnel == nil {
		return nil, errors.New("broadcast: no tunnel configured")
	}
	h.mu.Lock()
	for {
		switch h.state {
		case hostOpen:
			mt := &Mount{host: h, m: m, base: h.handle.URL(), mode: h.mode, exited: make(chan struct{})}
			h.mounts[mt] = struct{}{}
			exited := h.handle.Exited()
			h.mu.Unlock()
			if err := probe(ctx, mt.base+"/"+probePath, h.cfg.ProbeTimeout, h.cfg.Resolve, exited); err != nil {
				mt.Unmount()
				return nil, err
			}
			return mt, nil
		case hostClosed:
			h.beginOpenLocked()
		}
		// Join an open in progress; wait out a close, or an abandoned open,
		// and try again.
		a, counted := h.attempt, h.state == hostOpening && !h.attempt.abandoned
		if counted {
			h.waiting++
		}
		h.mu.Unlock()
		select {
		case <-a.done:
			h.mu.Lock()
			if counted {
				h.waiting--
				if a.err != nil {
					h.mu.Unlock()
					return nil, a.err
				}
			}
		case <-ctx.Done():
			h.abandonWait(counted)
			return nil, ctx.Err()
		}
	}
}

// abandonWait is a waiter giving up. If it was the last interest in the
// Host, it returns only once whatever was opened is closed again: an open
// that completed meanwhile, or a tunnel that finishes opening after the
// cancel.
func (h *Host) abandonWait(counted bool) {
	h.mu.Lock()
	if counted {
		h.waiting--
	}
	a := h.releaseIfUnwantedLocked()
	h.mu.Unlock()
	for a != nil {
		<-a.done
		// An abandoned open that completed anyway is closed by open's own
		// release; wait for that close too.
		h.mu.Lock()
		a = nil
		if h.state == hostClosing && h.waiting == 0 && len(h.mounts) == 0 {
			a = h.attempt
		}
		h.mu.Unlock()
	}
}

// releaseIfUnwantedLocked is the one rule for every loss of interest (an
// open completing, a waiter abandoning, an Unmount): with nothing mounted and
// nobody waiting, an open Host closes and an opening one is abandoned. It
// returns the attempt to wait on, if any.
func (h *Host) releaseIfUnwantedLocked() *hostAttempt {
	if h.waiting > 0 || len(h.mounts) > 0 {
		return nil
	}
	switch h.state {
	case hostOpen:
		return h.beginCloseLocked()
	case hostOpening:
		h.attempt.abandoned = true
		h.attempt.cancel()
		return h.attempt
	}
	return nil
}

func (h *Host) beginOpenLocked() {
	ctx, cancel := context.WithCancel(context.Background())
	a := &hostAttempt{done: make(chan struct{}), cancel: cancel}
	h.state, h.attempt = hostOpening, a
	go h.open(ctx, a)
}

func (h *Host) open(ctx context.Context, a *hostAttempt) {
	defer a.cancel()
	srv, handle, served, err := h.start(ctx)
	h.mu.Lock()
	defer h.mu.Unlock()
	defer close(a.done)
	if err != nil {
		a.err = err
		h.state, h.attempt = hostClosed, nil
		return
	}
	h.gen++
	h.state, h.attempt, h.srv, h.handle = hostOpen, nil, srv, handle
	h.mode = ModeTunnel
	if _, local := h.cfg.Tunnel.(LocalOnly); local {
		h.mode = ModeLocalOnly
	}
	go h.watch(h.gen, served, handle)
	// Everyone who wanted it may have given up while it opened.
	h.releaseIfUnwantedLocked()
}

// start opens the listener, server and tunnel; whatever it opened is closed
// again if it fails. A tunnel that finishes opening after ctx was cancelled
// is a success here: open finds nobody wants it and closes it, one rule for
// every way interest can lapse while opening.
func (h *Host) start(ctx context.Context) (*http.Server, Handle, <-chan error, error) {
	l, err := h.cfg.Tunnel.Listen()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("broadcast: listen: %w", err)
	}
	srv := &http.Server{
		Handler: h,
		// Bounds on what an internet client can hold open: headers must
		// arrive promptly, and a body gets its own read deadline in its
		// handler. No server-wide ReadTimeout: it would also time out the
		// disconnect watch on long-lived event streams.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(l) }()
	handle, err := h.cfg.Tunnel.Open(ctx, l)
	if err != nil {
		srv.Close()
		return nil, nil, nil, err
	}
	return srv, handle, served, nil
}

// watch closes the Host when its tunnel or server dies on its own, telling
// every mount why. A later Mount opens it again.
func (h *Host) watch(gen uint64, served <-chan error, handle Handle) {
	var reason error
	select {
	case <-handle.Exited():
		reason = ErrTunnelExited
	case err := <-served:
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		reason = fmt.Errorf("%w: %v", ErrServerFailed, err)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.gen != gen || h.state != hostOpen {
		return
	}
	closing := h.beginCloseLocked()
	for mt := range h.mounts {
		mt.closeDone = closing.done
		mt.lose(reason)
		delete(h.mounts, mt)
	}
}

// beginCloseLocked tears down the open server and tunnel in the background.
func (h *Host) beginCloseLocked() *hostAttempt {
	a := &hostAttempt{done: make(chan struct{})}
	srv, handle := h.srv, h.handle
	h.state, h.attempt, h.srv, h.handle = hostClosing, a, nil, nil
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownBudget)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
		}
		handle.Close()
		h.mu.Lock()
		h.state, h.attempt = hostClosed, nil
		h.mu.Unlock()
		close(a.done)
	}()
	return a
}

// ServeHTTP routes /<segment>/… to the mount that owns segment, through the
// segment's lock when it has one.
func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	segment, rest, ok := splitPath(r.URL.Path)
	if ok {
		h.mu.Lock()
		var owner Mountable
		var base string
		for mt := range h.mounts {
			if mt.m.Owns(segment) {
				owner, base = mt.m, mt.base
				break
			}
		}
		h.mu.Unlock()
		if owner != nil {
			if l := owner.LockFor(segment); l != nil {
				admitted, done := gate(w, r, segment, rest, l, isSecureBase(base))
				if admitted == nil {
					return
				}
				defer done()
				r = admitted
			}
			owner.ServeHTTP(w, r)
			return
		}
	}
	// As a mount answers: only GET is ever allowed, and an unknown token
	// is not found.
	SetSecurityHeaders(w.Header(), contentSecurityPolicy)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.NotFound(w, r)
}

// Mount is one handler served on a Host.
type Mount struct {
	host       *Host
	m          Mountable
	base, mode string

	once   sync.Once
	err    error
	exited chan struct{}
	// Protected by host.mu; retained after routing removal so teardown joins
	// this generation's close, even if the Host has reopened meanwhile.
	closeDone <-chan struct{}
}

// Base is the Host's public base URL, without a trailing slash.
func (mt *Mount) Base() string { return mt.base }

// Mode is ModeTunnel or ModeLocalOnly.
func (mt *Mount) Mode() string { return mt.mode }

// Exited closes if the Host lost its tunnel or server under this mount; Err
// then says which.
func (mt *Mount) Exited() <-chan struct{} { return mt.exited }

// Err is why the mount was lost; nil while it serves or after Unmount.
func (mt *Mount) Err() error {
	select {
	case <-mt.exited:
		return mt.err
	default:
		return nil
	}
}

func (mt *Mount) lose(reason error) {
	mt.once.Do(func() {
		mt.err = reason
		close(mt.exited)
	})
}

// Unmount stops serving the handler. The last Unmount closes the Host and
// joins its tunnel/server closure. Lost mounts join the closure that removed
// them, even after the Host reopens. Idempotent, including concurrent calls.
func (mt *Mount) Unmount() {
	h := mt.host
	h.mu.Lock()
	if _, ok := h.mounts[mt]; ok {
		delete(h.mounts, mt)
		if closing := h.releaseIfUnwantedLocked(); closing != nil {
			mt.closeDone = closing.done
		}
	}
	done := mt.closeDone
	h.mu.Unlock()
	if done != nil {
		<-done
	}
}

// OwnsToken reports whether segment is token, in constant time: every cast
// answers its tokens this way.
func OwnsToken(segment, token string) bool {
	return token != "" && subtle.ConstantTimeCompare([]byte(segment), []byte(token)) == 1
}

// currentHandle is the open tunnel's handle, nil unless open.
func (h *Host) currentHandle() Handle {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.handle
}
