package broadcast

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AccessEnforcement pins the Cloudflare Access team and exact application audience.
type AccessEnforcement struct{ TeamName, Audience string }

// ErrProtectedCleanup marks unconfirmed transport teardown independently of
// whether opening returned a mount. It must never be treated as a clean cancel.
var ErrProtectedCleanup = errors.New("broadcast: protected transport cleanup unconfirmed")

// ProtectedTunnel is a named, Access-enforced tunnel with a private Unix origin.
// Configuration is copied at construction and cannot be weakened by callers.
type ProtectedTunnel struct{ cloudflared Cloudflared }

type protectedTransport interface {
	Tunnel
	protectedTransport()
}

func (*ProtectedTunnel) protectedTransport() {}

var safeTunnelName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,99}$`)
var safeTeam = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
var safeAudience = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// NewProtectedTunnel refuses incomplete protection rather than selecting a fallback.
func NewProtectedTunnel(c Cloudflared, a AccessEnforcement) (*ProtectedTunnel, error) {
	if c.Named == nil || !safeTunnelName.MatchString(c.Named.Name) || !validProtectedHostname(c.Named.Hostname) || len(c.Guard) == 0 || strings.TrimSpace(c.Guard[0]) == "" || strings.TrimSpace(c.Records) == "" || !safeTeam.MatchString(a.TeamName) || !safeAudience.MatchString(a.Audience) {
		return nil, errors.New("broadcast: protected tunnel requires named hostname, guard, records and explicit Access team/audience")
	}
	named := *c.Named
	c.Named = &named
	c.Guard = append([]string(nil), c.Guard...)
	c.access = &a
	return &ProtectedTunnel{cloudflared: c}, nil
}
func validProtectedHostname(s string) bool {
	if len(s) > 253 || !strings.Contains(s, ".") || s != strings.ToLower(s) {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if !safeTeam.MatchString(label) {
			return false
		}
	}
	return net.ParseIP(s) == nil
}
func (p *ProtectedTunnel) Listen() (net.Listener, error) {
	return listenProtected(p.cloudflared.RunDir)
}
func listenProtected(runDir string) (net.Listener, error) {
	dir, err := os.MkdirTemp(runDir, privateDirPrefix)
	if err != nil {
		return nil, err
	}
	cleanup := func(err error) (net.Listener, error) { _ = os.RemoveAll(dir); return nil, err }
	if err := os.Chmod(dir, 0700); err != nil {
		return cleanup(err)
	}
	socket := filepath.Join(dir, "s")
	if len(socket) > maxSocketPath {
		return cleanup(errors.New("broadcast: protected Unix socket path too long"))
	}
	l, err := net.Listen("unix", socket)
	if err != nil {
		return cleanup(err)
	}
	if err := os.Chmod(socket, 0600); err != nil {
		_ = l.Close()
		return cleanup(err)
	}
	return &privateListener{Listener: l, dir: dir, socket: socket, protected: true}, nil
}
func (p *ProtectedTunnel) Open(ctx context.Context, l net.Listener) (Handle, error) {
	pl, ok := l.(*privateListener)
	if !ok || !pl.protected || pl.socket == "" {
		return nil, errors.New("broadcast: protected tunnel requires its private Unix listener")
	}
	return p.cloudflared.Open(ctx, l)
}

// NewProtectedHost creates an exclusive Host with connector/local readiness only.
// The private marker prevents ordinary tunnels from accidentally exposing remote control.
func NewProtectedHost(t Tunnel) (*Host, error) {
	switch p := t.(type) {
	case *ProtectedTunnel:
		if p == nil || p.cloudflared.access == nil {
			return nil, errors.New("broadcast: invalid protected tunnel")
		}
	case *FakeProtectedTunnel:
		if p == nil || p.FakeTunnel == nil {
			return nil, errors.New("broadcast: invalid protected fake")
		}
	default:
		return nil, errors.New("broadcast: protected transport required")
	}
	h := NewHost(HostConfig{Tunnel: t})
	h.protected = true
	return h, nil
}

// MountProtected admits exactly one owner, including while opening or closing.
func (h *Host) MountProtected(ctx context.Context, m Mountable) (*Mount, error) {
	h.mu.Lock()
	if h.closeErr != nil {
		err := errors.Join(ErrProtectedCleanup, h.closeErr)
		h.mu.Unlock()
		return nil, err
	}
	if !h.protected || h.protectedBusy {
		h.mu.Unlock()
		return nil, fmt.Errorf("broadcast: protected host unavailable: %w", ErrTunnelBusy)
	}
	h.protectedBusy = true
	h.protectedGeneration = h.gen
	h.mu.Unlock()
	mt, err := h.mount(ctx, m, "")
	if err != nil {
		h.mu.Lock()
		if h.closeErr == nil {
			h.protectedBusy = false
		} else {
			err = errors.Join(err, ErrProtectedCleanup, h.closeErr)
		}
		h.mu.Unlock()
	}
	return mt, err
}

// FakeProtectedTunnel exercises the protected Host lifecycle using FakeTunnel's
// stateful connector, forwarding only from a private Unix listener.
type FakeProtectedTunnel struct {
	*FakeTunnel
	// PublicURL is advertised without being probed. Empty uses a fixed HTTPS fixture.
	PublicURL string
	// RunDir selects the private origin directory; empty uses os.TempDir.
	RunDir string
}

func (*FakeProtectedTunnel) protectedTransport() {}
func (f *FakeProtectedTunnel) Listen() (net.Listener, error) {
	l, err := listenProtected(f.RunDir)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.stats.Listens++
	f.last = l
	f.lastAddr = l.Addr().String()
	f.mu.Unlock()
	return l, nil
}

func (f *FakeProtectedTunnel) Open(ctx context.Context, l net.Listener) (Handle, error) {
	pl, ok := l.(*privateListener)
	if !ok || !pl.protected || pl.socket == "" {
		return nil, errors.New("broadcast: protected fake requires private Unix listener")
	}
	h, err := f.FakeTunnel.Open(ctx, l)
	if err != nil {
		return nil, err
	}
	base := f.PublicURL
	if base == "" {
		base = "https://remote.example.com"
	}
	h.(*fakeHandle).url = base
	return h, nil
}
