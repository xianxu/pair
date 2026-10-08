package broadcast

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func startSession(t *testing.T, ft *FakeTunnel, hub HubOptions) *Session {
	t.Helper()
	s, err := Start(context.Background(), Config{Tunnel: ft, Hub: hub, Ping: 20 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.Stop(nil)
		<-s.Done()
	})
	return s
}

func get(t *testing.T, url string) (int, error) {
	t.Helper()
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

var tokenShape = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func TestSessionLinkServesViewer(t *testing.T) {
	ft := &FakeTunnel{}
	a := startSession(t, ft, HubOptions{})
	b := startSession(t, ft, HubOptions{})
	for _, s := range []*Session{a, b} {
		if !strings.HasPrefix(s.Link(), s.handle.URL()+"/") || !strings.HasSuffix(s.Link(), "/") {
			t.Fatalf("link %q", s.Link())
		}
		if code, err := get(t, s.Link()); err != nil || code != http.StatusOK {
			t.Fatalf("GET link: %d %v", code, err)
		}
		if !tokenShape.MatchString(s.token) {
			t.Fatalf("token %q is not 32 bytes of base64url", s.token)
		}
	}
	if a.token == b.token {
		t.Fatal("two sessions share a token")
	}
	if st := ft.Stats(); st.Listens != 2 || st.Opens != 2 {
		t.Fatalf("fake stats %+v", st)
	}
}

// readEnd waits for the SSE end event on an open stream.
func readEnd(t *testing.T, events *sseReader) sseEvent {
	t.Helper()
	for {
		ev := events.next()
		if ev.name == "end" {
			return ev
		}
	}
}

func openSessionEvents(t *testing.T, s *Session) *sseReader {
	t.Helper()
	resp, err := http.Get(s.Link() + "events")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return &sseReader{t: t, r: bufio.NewReader(resp.Body)}
}

func TestSessionStopEndsViewersAndRevokesToken(t *testing.T) {
	release := make(chan struct{})
	ft := &FakeTunnel{CloseBlock: release}
	s := startSession(t, ft, HubOptions{})
	s.Offer(live(t, "on air"), terminal.FramePublic)
	events := openSessionEvents(t, s)
	link := s.Link()

	stopped := make(chan struct{})
	go func() { s.Stop(nil); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop waited for the tunnel to close")
	}
	if ev := readEnd(t, events); !strings.Contains(ev.data, "operator stopped") {
		t.Fatalf("end event %q", ev.data)
	}
	select {
	case <-s.Done():
		t.Fatal("Done closed before the tunnel finished closing")
	default:
	}
	close(release)
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("teardown never finished")
	}
	if _, err := get(t, link); err == nil {
		t.Fatal("link still answers after the broadcast ended")
	}
	if st := ft.Stats(); st.Closes != 1 || st.DoubleCloses != 0 {
		t.Fatalf("fake stats %+v", st)
	}
	s.Stop(errors.New("again")) // idempotent
	if !errors.Is(s.Err(), ErrHubClosed) {
		t.Fatalf("Err() = %v", s.Err())
	}
}

func TestSessionCancelDuringStart(t *testing.T) {
	for _, late := range []bool{false, true} {
		name := "open honours cancel"
		if late {
			name = "open completes after cancel"
		}
		t.Run(name, func(t *testing.T) {
			ft := &FakeTunnel{OpenDelay: 50 * time.Millisecond, IgnoreCancel: late}
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(10*time.Millisecond, cancel)
			s, err := Start(ctx, Config{Tunnel: ft})
			if !errors.Is(err, context.Canceled) || s != nil {
				t.Fatalf("Start = %v, %v; want canceled", s, err)
			}
			st := ft.Stats()
			if st.Closes != st.Opens {
				t.Fatalf("a late open leaked: %+v", st)
			}
			if _, err := net.DialTimeout("tcp", ft.LastAddr(), time.Second); err == nil {
				t.Fatal("listener still accepting after a cancelled start")
			}
		})
	}
}

func TestSessionTunnelExitEndsSession(t *testing.T) {
	ft := &FakeTunnel{}
	s := startSession(t, ft, HubOptions{})
	events := openSessionEvents(t, s)
	ft.Exit()
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session outlived its tunnel")
	}
	if !errors.Is(s.Err(), ErrTunnelExited) {
		t.Fatalf("Err() = %v", s.Err())
	}
	if ev := readEnd(t, events); !strings.Contains(ev.data, "tunnel") {
		t.Fatalf("end event %q", ev.data)
	}
}

func TestSessionEndsWhenIndicatorHidden(t *testing.T) {
	s := startSession(t, &FakeTunnel{}, HubOptions{Grace: 20 * time.Millisecond})
	s.Offer(hidden(t, "no indicator"), terminal.FramePublic)
	s.Activate()
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session kept broadcasting with the indicator hidden")
	}
	if !errors.Is(s.Err(), ErrIndicatorHidden) {
		t.Fatalf("Err() = %v", s.Err())
	}
}

// Render and forget: a broadcast writes no frame content anywhere a process
// commonly writes (home, temp, XDG dirs, the working directory).
func TestSessionPersistsNoFrameData(t *testing.T) {
	root := t.TempDir()
	dirs := map[string]string{}
	for _, name := range []string{"HOME", "TMPDIR", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CONFIG_HOME", "cwd"} {
		d := filepath.Join(root, name)
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
		dirs[name] = d
		if name != "cwd" {
			t.Setenv(name, d)
		}
	}
	t.Chdir(dirs["cwd"])
	const marker = "BROADCAST-MARKER-395"
	s := startSession(t, &FakeTunnel{}, HubOptions{})
	events := openSessionEvents(t, s)
	for i := range 5 {
		s.Offer(live(t, marker+" "+strings.Repeat("x", i)), terminal.FramePublic)
		for ev := events.next(); ev.name != "frame"; ev = events.next() {
		}
	}
	s.Stop(nil)
	<-s.Done()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(b, []byte(marker)) {
			t.Errorf("frame content persisted in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A listener that breaks makes Serve return; the session must end rather
// than report live with nothing serving viewers.
func TestSessionEndsWhenServerFails(t *testing.T) {
	ft := &FakeTunnel{}
	s := startSession(t, ft, HubOptions{})
	ft.BreakListener()
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("session kept reporting live after its server stopped")
	}
	if !errors.Is(s.Err(), ErrServerFailed) {
		t.Fatalf("Err() = %v", s.Err())
	}
}

// hostTunnel serves on loopback but publishes a hostname the system can't
// resolve, as a fresh quick tunnel does before local DNS catches up.
type hostTunnel struct{ FakeTunnel }

func (h *hostTunnel) Open(ctx context.Context, l net.Listener) (Handle, error) {
	inner, err := h.FakeTunnel.Open(ctx, l)
	if err != nil {
		return nil, err
	}
	_, port, _ := net.SplitHostPort(l.Addr().String())
	return renamed{Handle: inner, url: "http://broadcast-probe.invalid:" + port}, nil
}

type renamed struct {
	Handle
	url string
}

func (r renamed) URL() string { return r.url }

func TestSessionProbeResolvesThroughItsResolver(t *testing.T) {
	var asked []string
	resolve := func(_ context.Context, host string) ([]string, error) {
		asked = append(asked, host)
		return []string{"127.0.0.1"}, nil
	}
	s, err := Start(context.Background(), Config{Tunnel: &hostTunnel{}, Resolve: resolve, ProbeTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { s.Stop(nil); <-s.Done() }()
	if len(asked) == 0 || asked[0] != "broadcast-probe.invalid" {
		t.Fatalf("probe asked %v", asked)
	}
}

func TestSessionProbeErrorCarriesNoToken(t *testing.T) {
	resolve := func(context.Context, string) ([]string, error) { return nil, errors.New("no such host") }
	_, err := Start(context.Background(), Config{Tunnel: &hostTunnel{}, Resolve: resolve, ProbeTimeout: 1200 * time.Millisecond})
	if err == nil {
		t.Fatal("start succeeded with an unresolvable link")
	}
	if strings.Contains(err.Error(), "broadcast-probe.invalid:") || regexp.MustCompile(`/[A-Za-z0-9_-]{43}/`).MatchString(err.Error()) {
		t.Fatalf("probe error leaks the link: %v", err)
	}
}
