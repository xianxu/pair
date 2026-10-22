package broadcast

import (
	"github.com/xianxu/pair/cmd/internal/castlock"

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
	// Host is the endpoint shared with other casts; nil makes a private one
	// on Tunnel.
	Host   *Host
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
	// OnNotes receives the open notes and their version after every change
	// (#447): a note saved or resolved, or the board emptied because pointing
	// turned off. Versions only increase, so a receiver keeps the newest. It
	// is called on the changing goroutine, holding no session lock.
	OnNotes func(s *Session, notes []Note, version uint64)
	// AnchorNote says what a note placed on a cols×rows grid is about
	// (#488): the operator's program knows its own screen. It is called as
	// the note is accepted, before it opens, and must answer from memory
	// (a request's goroutine waits on it). nil leaves notes without an anchor.
	AnchorNote func(n Note, cols, rows int) NoteAnchor
	// Now stamps notes and rate-limit decisions; nil is time.Now. Couch passes the
	// clock it draws with, so flash timing has one source.
	Now func() time.Time
	// OnCodeRotated is told when a link's code rotates (pair#495): by
	// guessing, or by RotateCode. It runs on the rotating goroutine, holding
	// no session lock.
	OnCodeRotated func(link string, r castlock.Rotation)
}

// The links a cast locks, by the label Codes reports: a screen broadcast's
// view and pointer links, a web cast's view and drive links.
const (
	LinkView    = "view"
	LinkPointer = "pointer"
	LinkDrive   = "drive"
)

// DriveCodeDigits is the drive link's code length: it controls the
// operator's browser, so it is harder to guess than a view code.
const DriveCodeDigits = 6

// LinkCode is one link's current code. It is a second factor, not a link:
// Couch shows it to the operator only, never on the broadcast frame.
type LinkCode struct {
	Link string
	Code string
}

// Session is one broadcast: a hub, a server on the tunnel's listener, the
// tunnel, and the token that is the link's only credential. Everything it
// owns dies with it; nothing is written to disk.
type Session struct {
	token   string
	link    string
	base    string // the public base URL, for the pointer link
	cfg     Config
	locks   *LinkLocks // the view and pointer links' code locks (pair#495)
	pointer *PointerState
	hub     *Hub
	// startedAt and mode describe the session for Status (pair#413).
	startedAt time.Time
	mode      string
	mount     *Mount

	stopOnce sync.Once
	reason   error
	done     chan struct{}
}

// Start opens a broadcast and returns once its link answers. Cancelling ctx
// abandons the start; whatever was opened, including a tunnel that finishes
// opening after the cancel, is closed.
func Start(ctx context.Context, cfg Config) (*Session, error) {
	host := cfg.Host
	if host == nil {
		if cfg.Tunnel == nil {
			return nil, errors.New("broadcast: no tunnel configured")
		}
		host = NewHost(HostConfig{Tunnel: cfg.Tunnel, ProbeTimeout: cfg.ProbeTimeout, Resolve: cfg.Resolve})
	}
	token, err := NewToken()
	if err != nil {
		return nil, err
	}
	// The session is built before the hub and server, which call back into
	// it; the pointer can't be used until Start returns (no token is minted
	// before EnablePointer).
	// Each lock exists before its token is routable: the view lock before
	// Mount, the pointer lock in the same step as its token.
	locks := &LinkLocks{}
	if err := locks.Mint(LinkView, token, castlock.DefaultDigits, cfg.OnCodeRotated); err != nil {
		return nil, err
	}
	pointer := newPointerState(func(token string) error {
		return locks.Mint(LinkPointer, token, castlock.DefaultDigits, cfg.OnCodeRotated)
	})
	s := &Session{token: token, cfg: cfg, locks: locks, pointer: pointer, done: make(chan struct{})}
	hubOpts := cfg.Hub
	// The hub calls this on its own goroutine, which must not block.
	// It records the flip generation it fired under, so a report that
	// arrives after the operator flipped pointing again is ignored.
	hubOpts.OnPointerHidden = func() {
		gen := s.pointer.generation()
		go s.pointerHidden(gen)
	}
	hub := NewHub(hubOpts)
	server := NewServer(ServerOptions{Token: token, Locks: locks, Hub: hub, Ping: cfg.Ping, Theme: cfg.Theme,
		Pointer: s.pointer, Now: s.now, OnPoint: s.acceptPoint, OnNote: s.acceptNote, OnResolve: s.resolveNote, OnNoteEdit: s.editNote})
	mount, err := host.Mount(ctx, server, token+"/")
	if err != nil {
		hub.Close(err)
		locks.End()
		return nil, err
	}
	s.link, s.base, s.hub, s.mount = mount.Base()+"/"+token+"/", mount.Base(), hub, mount
	s.startedAt, s.mode = time.Now(), mount.Mode()
	go s.watch()
	return s, nil
}

// Broadcast modes, as Status reports them.
const (
	ModeTunnel    = "tunnel"
	ModeLocalOnly = "local-only"
)

// Status describes a running session for `couch --broadcast-list` (pair#413).
// It has no field that could carry the link or its token: the token is the
// broadcast's only credential, and a listing must never print it.
type Status struct {
	StartedAt time.Time
	Mode      string
	Viewers   int
}

// Status is the session's current description.
func (s *Session) Status() Status {
	return Status{StartedAt: s.startedAt, Mode: s.mode, Viewers: s.hub.Viewers()}
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

// Codes is every minted link's current code, view first.
func (s *Session) Codes() []LinkCode { return s.locks.Codes() }

// RotateCode gives link a new code and disconnects everyone who unlocked it
// with the old one.
func (s *Session) RotateCode(link string) error { return s.locks.Rotate(link) }

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

// watch ends the session when the hub ends (indicator hidden) or the Host
// lost its tunnel or server.
func (s *Session) watch() {
	select {
	case <-s.hub.Done():
		s.Stop(s.hub.Err())
	case <-s.mount.Exited():
		s.Stop(s.mount.Err())
	}
}

// teardown unmounts; the hub is closed, so every event stream is already
// returning. If this was the Host's last mount, it returns once the listener
// and tunnel are closed.
func (s *Session) teardown() {
	defer close(s.done)
	s.mount.Unmount()
	// The locks end after the mount, never before the hub's end event: ending
	// a lock revokes its sessions, which cuts their streams, and a viewer
	// would lose "Broadcast ended" to a bare disconnect.
	s.locks.End()
}

// NewToken mints a capability token: 32 random bytes, URL-safe. A cast's
// link carries it and it is the link's only credential.
func NewToken() (string, error) {
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

// EnsurePointerLink mints the stable code-locked link without enabling pointing.
func (s *Session) EnsurePointerLink() (string, error) {
	token, err := s.pointer.ensure(NewToken)
	if err != nil {
		return "", err
	}
	return s.base + "/" + token + "/", nil
}

// EnablePointer turns pointing on (#412), minting the pointer link on first
// use; it returns the link, the same one on every later call.
func (s *Session) EnablePointer() (string, error) {
	token, changed, err := s.pointer.set(true, NewToken)
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
	_, before := s.Notes()
	if _, changed, _ := s.pointer.set(false, nil); changed {
		s.hub.DisarmPointer()
		s.notesChangedSince(before)
	}
}

// PointerLink is the pointer link, or "" before it has been minted.
func (s *Session) PointerLink() string {
	if token := s.pointer.link(); token != "" {
		return s.base + "/" + token + "/"
	}
	return ""
}

// pointerHidden turns pointing off because the active marker stayed off the
// operator's screen (the hub has already disarmed its watch).
func (s *Session) pointerHidden(gen uint64) {
	_, before := s.Notes()
	if !s.pointer.offIfGeneration(gen) {
		return
	}
	s.notesChangedSince(before)
	if s.cfg.OnPointerOff != nil {
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
	if !landsOn(cur, b.Cols, b.Rows) {
		return
	}
	if s.cfg.OnPoints != nil {
		b.ChromeRows = cur.ChromeRows
		s.cfg.OnPoints(b)
	}
}

// landsOn checks against trusted operator geometry. Hub.Current waits on the
// hub loop, so callers retrieve it before holding pointer-state locks.
func landsOn(cur CurrentFrame, cols, rows int) bool {
	return cur.ChromeRows > 0 && cur.ChromeRows < rows && cur.OK && cur.Geometry == (terminal.Geometry{Cols: cols, Rows: rows}) && cur.Class == terminal.FramePublic && cur.PointerShown
}

// ErrScreenChanged refuses a note placed on a screen the operator no longer
// shows.
var ErrScreenChanged = errors.New("broadcast: the screen changed")

// acceptNote opens a note placed on what the operator sees now (#447). The
// generation is read before the screen check, which runs unlocked; the add
// refuses if pointing turned off or flipped since.
func (s *Session) acceptNote(req NoteRequest) error {
	gen := s.pointer.generation()
	if on, _ := s.pointer.On(); !on {
		return ErrPointerOff
	}
	cur := s.hub.Current()
	if !landsOn(cur, req.Cols, req.Rows) {
		return ErrScreenChanged
	}
	if !hasPin(req.Note.Box, req.Cols, req.Rows, cur.ChromeRows) {
		return ErrNoteRange
	}
	req.Note.At = s.now()
	if s.cfg.AnchorNote != nil {
		a := s.cfg.AnchorNote(req.Note, req.Cols, req.Rows)
		req.Note.Anchor = &a
	}
	if _, err := s.pointer.addNote(req.Note, req.Cols, req.Rows, cur.ChromeRows, gen); err != nil {
		return err
	}
	s.notesChanged()
	return nil
}

// ResolveNote resolves note id for the operator (the terminal's notes menu),
// as a viewer's Resolve does.
func (s *Session) ResolveNote(id int) error {
	return s.resolveNote(NoteResolutionRequest{ID: id, Author: "Operator"})
}

// resolveNote removes a note; an unknown id is already resolved.
func (s *Session) resolveNote(in NoteResolutionRequest) error {
	changed, err := s.pointer.resolveNote(NoteResolution{ID: in.ID, Author: in.Author, At: s.now()})
	if changed {
		s.notesChanged()
	}
	return err
}

func (s *Session) now() time.Time {
	if s.cfg.Now != nil {
		return s.cfg.Now()
	}
	return time.Now()
}

// Notes is the open notes and their version.
func (s *Session) Notes() ([]Note, uint64) {
	notes, version, _ := s.pointer.Notes()
	return notes, version
}

// notesChangedSince reports the notes if they changed since version: a flip
// to off reports only when it emptied the board.
func (s *Session) notesChangedSince(version uint64) {
	if _, now := s.Notes(); now != version {
		s.notesChanged()
	}
}

func (s *Session) notesChanged() {
	if s.cfg.OnNotes != nil {
		notes, version := s.Notes()
		s.cfg.OnNotes(s, notes, version)
	}
}
