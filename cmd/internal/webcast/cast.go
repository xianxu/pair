package webcast

import (
	"github.com/xianxu/pair/cmd/internal/castlock"

	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recorder"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// CastConfig starts a Cast.
type CastConfig struct {
	// ReadOriginPolicy snapshots operator-owned sharing rules per submission.
	// A notice reports a private fallback without refusing the submission.
	ReadOriginPolicy func() (core.OriginPolicy, string)
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
	// Check, when set, runs once a recorded cast has ended, if its
	// recording has frames: the fidelity check (#478; couchcmd wires
	// CheckFidelity). It runs in the background, after the cast is done;
	// its summary, or its error, goes to Logf.
	Check func(ctx context.Context, dir string) (summary string, err error)
	// Logf reports a recording's problems to the operator: one that can't
	// be opened or written, a resync, and at its end what it lost.
	Logf func(format string, args ...any)
	// Host is the endpoint shared with the screen broadcast.
	Host *broadcast.Host
	// Ping is the viewers' keepalive; zero is broadcast.DefaultPing.
	Ping time.Duration
	// Presence, when set, is told which viewers are watching (#466).
	Presence Presence
	// OnCodeRotated is told when a link's code rotates (pair#495), on the
	// rotating goroutine; keep the highest Rotation.Gen per link.
	OnCodeRotated func(link string, r castlock.Rotation)
	// DriveGrace is how long a remote-driving seat outlives its last
	// stream (#467); zero is drive.DefaultGrace.
	DriveGrace time.Duration
}

// Update is what Couch shows of a cast: the browser's state and its links
// (pair#506). Link is the view-only link, "" until it answers; LinkErr says
// why there is none.
type Update struct {
	State State
	Link  string
	// PointerLink lets its viewers point and leave notes as well. "" until
	// the link answers, or when the cast takes no notes (not recorded).
	PointerLink string
	// DriveLink is the drive link (#467): the same viewer, offering a seat
	// to drive. "" until the link answers, or when the cast can't be driven.
	DriveLink string
	LinkErr   error
}

// LinkFor is the link of kind (broadcast.LinkView, LinkPointer or LinkDrive,
// the codes' labels): "" for a link the cast doesn't have, or any other kind.
// The one place a kind becomes a URL (pair#506).
func (u Update) LinkFor(kind string) string {
	switch kind {
	case broadcast.LinkView:
		return u.Link
	case broadcast.LinkPointer:
		return u.PointerLink
	case broadcast.LinkDrive:
		return u.DriveLink
	}
	return ""
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
	// board takes viewers' strokes and notes (#470); nil when the cast
	// isn't recorded. The recorder closes it with the recording (it is one
	// of the recording's sinks), the cast when the browser has ended.
	board *board.Board
	// drive is remote driving's controller (#467); nil when the browser
	// can't be driven (not recording, or a fake). Closed when the browser
	// has ended.
	drive drive.Controller
	// recDir is the recording's directory ("" when not recording); check
	// and logf run the fidelity check on it once the cast has ended.
	readOriginPolicy func() (core.OriginPolicy, string)
	recDir           string
	check            func(context.Context, string) (string, error)
	logf             func(string, ...any)
	// server and token: for tests. pointerToken is the pointer link's
	// (pair#506), "" when the cast takes no notes; driveToken is the drive
	// link's (#467), "" when the cast can't be driven.
	server       *Server
	token        string
	pointerToken string
	// locks are the links' code locks (pair#495): view, pointer when the
	// cast takes notes, and drive when it can be driven. They end after the
	// mount.
	locks      *broadcast.LinkLocks
	driveToken string
	// driveOff says why the cast can't be driven ("" when it can).
	driveOff string
}

// StartCast launches the browser and returns once it runs; the cast's links
// open in the background, so the operator has Chrome at once.
func StartCast(ctx context.Context, cfg CastConfig) (*Cast, error) {
	// The links' locks (pair#495), each minted with its token before the
	// token is routable; they end after the mount.
	locks := &broadcast.LinkLocks{}
	var live recording.Live
	var liveSink recording.Sink
	var ro *RecordOptions
	var bd *board.Board
	var b Browser // set once launched; the board's requests come after
	var recDir string
	// The cast's asset proxy exists before Chrome does: the recording (and
	// its store, bound here) can open at a go-live before the Server is up.
	proxy := castassets.NewProxy()
	if cfg.RecordingsDir != "" {
		if cfg.NewLive != nil {
			liveSink, live = cfg.NewLive()
		}
		// Viewers' strokes and notes go into the recorder's queue and the
		// board follows the recording, so it sees what viewers see.
		bd = board.New(notes.Structural{}, func(r recording.Record, doc int) error {
			if e, ok := b.(Emitter); ok {
				return e.Emit(r, doc)
			}
			return ErrNotRecording
		})
		bd.Logf = cfg.Logf
		id := recordingID(time.Now())
		recDir = filepath.Join(cfg.RecordingsDir, id)
		ro = &RecordOptions{Operator: bd, ID: id, Out: recordingOutput(cfg, liveSink, bd, id, proxy), Logf: cfg.Logf}
	}
	b, err := cfg.Launch(ctx, ro)
	if err != nil {
		if bd != nil {
			bd.Close()
		}
		if liveSink != nil {
			liveSink.Close()
		}
		return nil, err
	}
	token, err := broadcast.NewToken()
	if err == nil {
		// The view-only link's lock comes with its token (pair#495).
		err = locks.Mint(broadcast.LinkView, token, castlock.DefaultDigits, cfg.OnCodeRotated)
	}
	// A cast that takes notes has a pointer link, locked like the view-only link;
	// without its token the cast can't be what it was asked to be, so it
	// doesn't start (pair#506).
	var pointerToken string
	if err == nil && bd != nil {
		if pointerToken, err = broadcast.NewToken(); err == nil {
			err = locks.Mint(broadcast.LinkPointer, pointerToken, castlock.DefaultDigits, cfg.OnCodeRotated)
		}
	}
	if err != nil {
		b.Close()
		if bd != nil {
			bd.Close()
		}
		return nil, err
	}
	mountCtx, cancel := context.WithCancel(context.Background())
	c := &Cast{browser: b, pub: newPublisher(Update{State: b.State()}), cancel: cancel, done: make(chan struct{}), liveSink: liveSink,
		board: bd, readOriginPolicy: cfg.ReadOriginPolicy, recDir: recDir, check: cfg.Check, logf: cfg.Logf}
	if d, ok := b.(Driven); ok && ro != nil {
		grace := cfg.DriveGrace
		if grace <= 0 {
			grace = drive.DefaultGrace
		}
		// The recorder stamps each control record's time as it orders it.
		c.drive = drive.NewMemController(liveGate{d}, drive.NewTabInjector(d), func() int64 { return 0 }, grace)
		go followDriving(c.drive, d)
	}
	c.locks = locks
	var driveToken string
	if c.drive != nil {
		if driveToken, err = broadcast.NewToken(); err == nil {
			// The drive lock is minted with its token; without it there is
			// no drive link (pair#495).
			err = locks.Mint(broadcast.LinkDrive, driveToken, broadcast.DriveCodeDigits, cfg.OnCodeRotated)
		}
		if err != nil {
			driveToken = ""
			// Driving is off for this cast; the operator is told why.
			c.drive.Close()
			c.drive = nil
			c.driveOff = "couldn't make the drive link: " + err.Error()
			if cfg.Logf != nil {
				cfg.Logf("remote driving is off: %s", c.driveOff)
			}
		}
	} else {
		c.driveOff = "the cast isn't recorded"
	}
	srv := NewServer(token, b, cfg.Ping, live, cfg.Presence).WithBoard(bd).WithPointer(pointerToken).WithDrive(c.drive, driveToken).WithDriveLog(cfg.Logf).WithLocks(locks)
	srv.proxy = proxy
	c.server, c.token, c.pointerToken, c.driveToken = srv, token, pointerToken, driveToken
	go c.run(mountCtx, cfg.Host, srv, token)
	return c, nil
}

// linkURL is a link's URL under the Host's base; "" for a link the cast
// doesn't have (an empty token).
func linkURL(base, token string) string {
	if token == "" {
		return ""
	}
	return base + "/" + token + "/"
}

// recordingID names a recording by its start, plus a random suffix.
func recordingID(at time.Time) string {
	var b [2]byte
	rand.Read(b[:])
	return at.Format("20060102T150405") + "-" + hex.EncodeToString(b[:])
}

// recordingOutput opens a recording at the first go-live: a DirWriter at
// <dir>/<id>, teed into the viewers' fan-out when there is one and the
// board (#470). The recorder closes them all when the recording ends.
func recordingOutput(cfg CastConfig, live recording.Sink, bd *board.Board, id string, proxy *castassets.Proxy) recorder.Output {
	return func() (recorder.Opened, error) {
		if err := os.MkdirAll(cfg.RecordingsDir, 0o700); err != nil {
			return recorder.Opened{}, fmt.Errorf("webcast: recordings dir: %w", err)
		}
		w, err := recording.NewDirWriter(filepath.Join(cfg.RecordingsDir, id))
		if err != nil {
			return recorder.Opened{}, fmt.Errorf("webcast: recording: %w", err)
		}
		o := recorder.Opened{Sink: w, Frames: w}
		// The board before the viewers' fan-out: a viewer acts on what it
		// was shown, which the board must already know.
		sinks := []recording.Sink{w}
		if bd != nil {
			sinks = append(sinks, bd)
		}
		if live != nil {
			sinks = append(sinks, live)
		}
		if len(sinks) > 1 {
			o.Sink = recording.Tee(sinks...)
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
				base := r.m.Base()
				cur.Link, cur.PointerLink, cur.DriveLink = linkURL(base, token), linkURL(base, c.pointerToken), linkURL(base, c.driveToken)
			}
		case <-lost:
			lost = nil
			cur.Link, cur.PointerLink, cur.DriveLink, cur.LinkErr = "", "", "", mount.Err()
		}
		c.pub.publish(cur)
	}
	c.cancel()
	if c.liveSink != nil {
		c.liveSink.Close()
	}
	if c.board != nil {
		c.board.Close()
	}
	if c.drive != nil {
		c.drive.Close()
	}
	if mounted != nil {
		if r := <-mounted; r.m != nil {
			mount = r.m
		}
	}
	if mount != nil {
		mount.Unmount()
	}
	// After the mount: ending a lock revokes its sessions, and nothing may
	// cut a viewer's stream before the browser's end reached it.
	c.locks.End()
	c.pub.finish(Update{State: c.browser.State()})
	c.checkRecording()
}

// checkRecording starts the fidelity check on the cast's recording, if it
// has frames to check: the recorder closed it before the browser ended.
func (c *Cast) checkRecording() {
	if c.check == nil || c.recDir == "" {
		return
	}
	if _, err := os.Stat(filepath.Join(c.recDir, recording.FileFrames)); err != nil {
		return // never live, or no frame kept
	}
	logf := c.logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	go func() {
		summary, err := c.check(context.Background(), c.recDir)
		switch {
		case err != nil:
			logf("fidelity check: %v", err)
		case summary != "":
			logf("%s", summary)
		}
	}()
}

// Notes delivers the open notes viewers left on the current page (#470),
// whenever they change, for Couch's 💭N. A cast that isn't recorded has
// none: its channel holds one empty value and closes.
func (c *Cast) Notes() (<-chan []board.LiveNote, func()) {
	if c.board == nil {
		ch := make(chan []board.LiveNote, 1)
		ch <- nil
		close(ch)
		return ch, func() {}
	}
	return c.board.Subscribe()
}

// ResolveNote resolves a note as the operator.
func (c *Cast) ResolveNote(id string) error {
	if c.board == nil {
		return board.ErrUnknown
	}
	return c.board.SetState(board.StateIn{Viewer: board.Viewer{ID: board.Operator}, ID: id, State: recording.NoteResolved})
}

// Current is the latest Update.
func (c *Cast) Current() Update { return c.pub.get() }

// Subscribe delivers the latest Update, as Browser.Subscribe does.
func (c *Cast) Subscribe() (<-chan Update, func()) { return c.pub.subscribe() }

// ToggleLive goes live, or pauses.
func (c *Cast) ToggleLive() error {
	return c.browser.SetLive(c.browser.State().Phase != PhaseLive)
}

// Focus brings the cast tab and its Chrome window to the front (#501).
func (c *Cast) Focus() error { return c.browser.Focus() }

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

// Codes is each link's current code, view first. They are shown to the
// operator only, never on a broadcast frame.
func (c *Cast) Codes() []broadcast.LinkCode { return c.locks.Codes() }

// RotateCode gives link a new code and disconnects everyone who unlocked it
// with the old one.
func (c *Cast) RotateCode(link string) error { return c.locks.Rotate(link) }

// Drive is the cast's remote-driving controller (#467): Couch grants,
// denies and revokes through it and shows its state. nil when the cast
// can't be driven.
func (c *Cast) Drive() drive.Controller { return c.drive }

// Viewers is how many viewers are on the cast's links, by link (#467).
func (c *Cast) Viewers() (view, drive int) { return c.server.Viewers() }

// DriveOff says why the cast can't be driven, or "" when it can (#467).
func (c *Cast) DriveOff() string { return c.driveOff }
