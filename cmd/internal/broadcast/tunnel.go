package broadcast

import (
	"context"
	"errors"
	"net"
	"sync"
)

// ErrTunnelExited ends a broadcast whose tunnel died on its own.
var ErrTunnelExited = errors.New("broadcast: the tunnel exited")

// Tunnel exposes the broadcast's local listener. It owns both halves because
// the listener's kind depends on what exposes it: a browser on this machine
// needs TCP, while cloudflared can forward to a private unix socket.
type Tunnel interface {
	// Listen creates the local listener this tunnel forwards to.
	Listen() (net.Listener, error)
	// Open exposes l and returns once the public base URL is known.
	Open(ctx context.Context, l net.Listener) (Handle, error)
}

// Handle is one open tunnel.
type Handle interface {
	// URL is the public base URL, without a trailing slash.
	URL() string
	// Exited closes if the tunnel dies on its own.
	Exited() <-chan struct{}
	// Close stops the tunnel and removes whatever Listen created. Idempotent.
	Close() error
}

// LocalOnly serves on loopback TCP with no tunnel: the link works only on
// this machine (COUCH_BROADCAST_TUNNEL=off).
type LocalOnly struct{}

func (LocalOnly) Listen() (net.Listener, error) { return net.Listen("tcp", "127.0.0.1:0") }

func (LocalOnly) Open(_ context.Context, l net.Listener) (Handle, error) {
	return newLocalHandle("http://" + l.Addr().String()), nil
}

type localHandle struct {
	url    string
	exited chan struct{}
	once   sync.Once
}

func newLocalHandle(url string) *localHandle {
	return &localHandle{url: url, exited: make(chan struct{})}
}

func (h *localHandle) URL() string             { return h.url }
func (h *localHandle) Exited() <-chan struct{} { return h.exited }
func (h *localHandle) Close() error            { return nil }
