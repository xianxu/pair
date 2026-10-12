package webcast

import (
	"context"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// CastConfig starts a Cast.
type CastConfig struct {
	// Launch starts the cast Chrome (real, or a FakeBrowser in tests).
	Launch func(ctx context.Context) (Browser, error)
	// Host is the endpoint shared with the screen broadcast.
	Host *broadcast.Host
	// Ping is the viewers' keepalive; zero is broadcast.DefaultPing.
	Ping time.Duration
}

// Update is what Couch shows of a cast: the browser's state and the viewer
// link. Link is "" until it answers; LinkErr says why there is none.
type Update struct {
	State   State
	Link    string
	LinkErr error
}

// Cast is one web cast: a Browser, served to viewers under a token on the
// shared Host. It ends when its Browser does.
type Cast struct {
	browser Browser
	pub     *publisher[Update]
	cancel  context.CancelFunc
	done    chan struct{}
}

// StartCast launches the browser and returns once it runs; the viewer link
// opens in the background, so the operator has Chrome at once.
func StartCast(ctx context.Context, cfg CastConfig) (*Cast, error) {
	b, err := cfg.Launch(ctx)
	if err != nil {
		return nil, err
	}
	token, err := broadcast.NewToken()
	if err != nil {
		b.Close()
		return nil, err
	}
	mountCtx, cancel := context.WithCancel(context.Background())
	c := &Cast{browser: b, pub: newPublisher(Update{State: b.State()}), cancel: cancel, done: make(chan struct{})}
	go c.run(mountCtx, cfg.Host, NewServer(token, b, cfg.Ping), token)
	return c, nil
}

type mountResult struct {
	m   *broadcast.Mount
	err error
}

// run owns the cast's Update. It ends when the browser does: viewers have
// been told by then (their streams follow the browser), so the mount goes
// last.
func (c *Cast) run(ctx context.Context, host *broadcast.Host, srv *Server, token string) {
	defer close(c.done)
	states, unsubscribe := c.browser.Subscribe()
	defer unsubscribe()
	mounted := make(chan mountResult, 1)
	go func() {
		m, err := host.Mount(ctx, srv, token+"/")
		mounted <- mountResult{m, err}
	}()
	var mount *broadcast.Mount
	var lost <-chan struct{}
	cur := c.pub.get()
	for running := true; running; {
		select {
		case st, open := <-states:
			if !open {
				running = false
				continue
			}
			cur.State = st
		case r := <-mounted:
			mounted = nil
			if r.err != nil {
				cur.LinkErr = r.err
			} else {
				mount, lost = r.m, r.m.Exited()
				cur.Link = r.m.Base() + "/" + token + "/"
			}
		case <-lost:
			lost = nil
			cur.Link, cur.LinkErr = "", mount.Err()
		}
		c.pub.publish(cur)
	}
	c.cancel()
	if mounted != nil {
		if r := <-mounted; r.m != nil {
			mount = r.m
		}
	}
	if mount != nil {
		mount.Unmount()
	}
	c.pub.finish(Update{State: c.browser.State()})
}

// Current is the latest Update.
func (c *Cast) Current() Update { return c.pub.get() }

// Subscribe delivers the latest Update, as Browser.Subscribe does.
func (c *Cast) Subscribe() (<-chan Update, func()) { return c.pub.subscribe() }

// ToggleLive goes live, or pauses.
func (c *Cast) ToggleLive() error {
	return c.browser.SetLive(c.browser.State().Phase != PhaseLive)
}

// End ends the cast and closes Chrome; it returns once the link is closed.
func (c *Cast) End() {
	c.browser.Close()
	<-c.done
}

// Done closes when the cast has ended and its link is closed.
func (c *Cast) Done() <-chan struct{} { return c.done }

// Err is why the cast ended (the browser's reason); nil while it runs and
// for End.
func (c *Cast) Err() error {
	select {
	case <-c.done:
		return c.browser.Err()
	default:
		return nil
	}
}
