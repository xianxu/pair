package broadcast

import (
	"context"
	"net"
	"sync"
	"time"
)

// FakeTunnel is a stateful Tunnel for tests across packages. It serves on
// loopback TCP like LocalOnly and records what was asked of it.
type FakeTunnel struct {
	// CloseError makes teardown uncertain without reporting a successful close.
	CloseError error
	// OpenStarted/OpenRelease allow a test to hold a connector handshake.
	OpenStarted  chan struct{}
	OpenRelease  <-chan struct{}
	CloseStarted chan struct{}
	// OpenDelay makes Open take this long.
	OpenDelay time.Duration
	// IgnoreCancel makes a delayed Open complete even after its context is
	// cancelled, as a slow real tunnel can.
	IgnoreCancel bool
	// CloseBlock, when set, holds every Close until it is closed.
	CloseBlock chan struct{}
	// Exclusive refuses an Open while another is open with ErrTunnelBusy,
	// as a named tunnel's claim does.
	Exclusive bool

	mu       sync.Mutex
	stats    FakeTunnelStats
	lastAddr string
	last     net.Listener
	handles  []*fakeHandle
}

// FakeTunnelStats counts calls. DoubleCloses counts Close on a closed handle;
// MaxOpen is the most handles open at once.
type FakeTunnelStats struct {
	Listens, Opens, Closes, DoubleCloses, MaxOpen int
}

func (f *FakeTunnel) Listen() (net.Listener, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.stats.Listens++
	f.lastAddr = l.Addr().String()
	f.last = l
	f.mu.Unlock()
	return l, nil
}

func (f *FakeTunnel) Open(ctx context.Context, l net.Listener) (Handle, error) {
	if f.OpenStarted != nil {
		select {
		case f.OpenStarted <- struct{}{}:
		default:
		}
	}
	if f.OpenRelease != nil {
		if f.IgnoreCancel {
			<-f.OpenRelease
		} else {
			select {
			case <-f.OpenRelease:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	if f.OpenDelay > 0 {
		t := time.NewTimer(f.OpenDelay)
		defer t.Stop()
		if f.IgnoreCancel {
			<-t.C
		} else {
			select {
			case <-t.C:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	h := &fakeHandle{f: f, localHandle: newLocalHandle("http://" + l.Addr().String())}
	f.mu.Lock()
	defer f.mu.Unlock()
	open := f.stats.Opens - f.stats.Closes
	if f.Exclusive && open > 0 {
		return nil, ErrTunnelBusy
	}
	f.stats.Opens++
	f.stats.MaxOpen = max(f.stats.MaxOpen, open+1)
	f.handles = append(f.handles, h)
	return h, nil
}

// Exit makes the most recently opened tunnel die on its own.
func (f *FakeTunnel) Exit() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if n := len(f.handles); n > 0 {
		h := f.handles[n-1]
		h.once.Do(func() { close(h.exited) })
	}
}

func (f *FakeTunnel) Stats() FakeTunnelStats {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stats
}

// BreakListener closes the most recent listener under its server, as a
// failing socket would.
func (f *FakeTunnel) BreakListener() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.last != nil {
		f.last.Close()
	}
}

// LastAddr is the address of the most recent listener.
func (f *FakeTunnel) LastAddr() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastAddr
}

type fakeHandle struct {
	*localHandle
	f      *FakeTunnel
	closed bool
}

func (h *fakeHandle) Close() error {
	if h.f.CloseStarted != nil {
		select {
		case h.f.CloseStarted <- struct{}{}:
		default:
		}
	}
	if h.f.CloseBlock != nil {
		<-h.f.CloseBlock
	}
	h.f.mu.Lock()
	defer h.f.mu.Unlock()
	if h.f.CloseError != nil {
		return h.f.CloseError
	}
	if h.closed {
		h.f.stats.DoubleCloses++
		return nil
	}
	h.closed = true
	h.f.stats.Closes++
	return nil
}
