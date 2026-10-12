package webcast

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// CastConfig starts a Cast.
type CastConfig struct {
	// Launch starts the cast Chrome (real, or a FakeBrowser in tests),
	// recording it as rec says when rec is non-nil. A FakeBrowser records
	// nothing.
	Launch func(ctx context.Context, rec *RecordOptions) (Browser, error)
	// RecordingsDir, when set, records each cast to <dir>/<id>/ from its
	// first go-live (#458). A recording is kept until the operator deletes
	// its directory (atlas/webcast-format.md, Lifecycle).
	RecordingsDir string
	// NewLive makes a cast's viewer fan-out (#460's hub), fed the same
	// records as the directory; nil streams only the cast's state.
	NewLive func() (recording.Sink, recording.Live)
	// NewAssets makes a recording's asset store (#459), persisting bodies
	// through the recording's DirWriter; nil records no assets.
	NewAssets func(recording.AssetWriter) castassets.Served
	// Logf reports a recording's problems to the operator: one that can't
	// be opened or written, a resync, and at its end what it lost.
	Logf func(format string, args ...any)
	// Host is the endpoint shared with the screen broadcast.
	Host *broadcast.Host
	// Ping is the viewers' keepalive; zero is broadcast.DefaultPing.
	Ping time.Duration
	// Presence, when set, is told which viewers are watching (#466).
	Presence Presence
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
	// liveSink feeds the viewers' fan-out; the recorder closes it when its
	// recording ends, and the cast does when the browser has (a cast never
	// live). Closing twice is a no-op.
	liveSink recording.Sink
	// server and token: for tests.
	server *Server
	token  string
}

// StartCast launches the browser and returns once it runs; the viewer link
// opens in the background, so the operator has Chrome at once.
func StartCast(ctx context.Context, cfg CastConfig) (*Cast, error) {
	var live recording.Live
	var liveSink recording.Sink
	var ro *RecordOptions
	// The cast's asset proxy exists before Chrome does: the recording (and
	// its store, bound here) can open at a go-live before the Server is up.
	proxy := castassets.NewProxy()
	if cfg.RecordingsDir != "" {
		if cfg.NewLive != nil {
			liveSink, live = cfg.NewLive()
		}
		id := recordingID(time.Now())
		ro = &RecordOptions{ID: id, Out: recordingOutput(cfg, liveSink, id, proxy), Logf: cfg.Logf}
	}
	b, err := cfg.Launch(ctx, ro)
	if err != nil {
		if liveSink != nil {
			liveSink.Close()
		}
		return nil, err
	}
	token, err := broadcast.NewToken()
	if err != nil {
		b.Close()
		return nil, err
	}
	mountCtx, cancel := context.WithCancel(context.Background())
	c := &Cast{browser: b, pub: newPublisher(Update{State: b.State()}), cancel: cancel, done: make(chan struct{}), liveSink: liveSink}
	srv := NewServer(token, b, cfg.Ping, live, cfg.Presence)
	srv.proxy = proxy
	c.server, c.token = srv, token
	go c.run(mountCtx, cfg.Host, srv, token)
	return c, nil
}

// recordingID names a recording by its start, plus a random suffix.
func recordingID(at time.Time) string {
	var b [2]byte
	rand.Read(b[:])
	return at.Format("20060102T150405") + "-" + hex.EncodeToString(b[:])
}

// recordingOutput opens a recording at the first go-live: a DirWriter at
// <dir>/<id>, teed into the viewers' fan-out when there is one. The
// recorder closes both when the recording ends.
func recordingOutput(cfg CastConfig, live recording.Sink, id string, proxy *castassets.Proxy) recorder.Output {
	return func() (recorder.Opened, error) {
		if err := os.MkdirAll(cfg.RecordingsDir, 0o700); err != nil {
			return recorder.Opened{}, fmt.Errorf("webcast: recordings dir: %w", err)
		}
		w, err := recording.NewDirWriter(filepath.Join(cfg.RecordingsDir, id))
		if err != nil {
			return recorder.Opened{}, fmt.Errorf("webcast: recording: %w", err)
		}
		o := recorder.Opened{Sink: w, Frames: w}
		if live != nil {
			o.Sink = recording.Tee(w, live)
		}
		if cfg.NewAssets != nil {
			store := cfg.NewAssets(w)
			proxy.Bind(store) // viewers get the bodies the recording keeps
			o.Assets = store
		}
		return o, nil
	}
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
	if c.liveSink != nil {
		c.liveSink.Close()
	}
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
