package broadcast

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	ErrNoCloudflared = errors.New("broadcast: cloudflared is not installed (brew install cloudflared)")
	// ErrTunnelBusy refuses a second broadcaster on one named tunnel: two
	// connectors of a tunnel are replicas, and Cloudflare would split viewers'
	// requests between two Couches that each know only their own token.
	ErrTunnelBusy = errors.New("broadcast: another Couch is broadcasting on this tunnel")
)

const (
	defaultOpenTimeout = 30 * time.Second
	closeBudget        = 5 * time.Second
	privateDirPrefix   = "couch-broadcast-"
	// maxSocketPath stays under macOS's 104-byte sun_path.
	maxSocketPath = 100
)

var (
	// A quick tunnel's hostname is hyphen-joined random words. The pattern
	// requires the hyphen, so cloudflared's own API host, which appears in its
	// failure line ("failed to request quick Tunnel: Post
	// https://api.trycloudflare.com/tunnel"), can't pass for a tunnel.
	quickTunnelURL  = regexp.MustCompile(`https://[a-z0-9]+(?:-[a-z0-9]+)+\.trycloudflare\.com`)
	namedRegistered = "Registered tunnel connection"
)

// NamedTunnel is a tunnel created once on the operator's Cloudflare account
// (`cloudflared tunnel create`, `tunnel route dns`); cloudflared finds its
// credentials itself.
type NamedTunnel struct {
	Name     string
	Hostname string
}

// Cloudflared exposes a broadcast through cloudflared: a named tunnel over a
// private unix socket, or, with Named nil, an anonymous quick tunnel over a
// loopback TCP port (quick tunnels can't forward to a socket).
type Cloudflared struct {
	Named *NamedTunnel
	// Binary is the cloudflared to run; empty looks it up on PATH.
	Binary string
	// Guard is the argv that runs RunGuard (`couch __broadcast-guard`); the
	// tunnel runs under it so it can't outlive Couch. Nil runs cloudflared
	// directly, for tests of this type alone.
	Guard []string
	// RunDir is where private directories are made; empty is os.TempDir().
	RunDir string
	// Records holds run records (see ReapOrphans); empty disables records and
	// the named-tunnel lock.
	Records     string
	OpenTimeout time.Duration
}

// privateListener is a listener in its own 0700 directory; closing it
// removes the directory, so an abandoned start leaves nothing behind.
type privateListener struct {
	net.Listener
	dir    string
	socket string // empty for TCP
	once   sync.Once
}

func (l *privateListener) Close() error {
	err := l.Listener.Close()
	l.once.Do(func() { _ = os.RemoveAll(l.dir) })
	return err
}

func (c Cloudflared) Listen() (net.Listener, error) {
	dir, err := os.MkdirTemp(c.RunDir, privateDirPrefix)
	if err != nil {
		return nil, err
	}
	if socket := filepath.Join(dir, "s"); c.Named != nil && len(socket) <= maxSocketPath {
		l, err := net.Listen("unix", socket)
		if err == nil {
			return &privateListener{Listener: l, dir: dir, socket: socket}, nil
		}
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return &privateListener{Listener: l, dir: dir}, nil
}

func (c Cloudflared) Open(ctx context.Context, l net.Listener) (Handle, error) {
	pl, ok := l.(*privateListener)
	if !ok {
		return nil, errors.New("broadcast: cloudflared needs the listener its Listen made")
	}
	bin := c.Binary
	if bin == "" {
		bin = "cloudflared"
	}
	bin, err := exec.LookPath(bin)
	if err != nil {
		return nil, ErrNoCloudflared
	}
	var records *runRecords
	if c.Records != "" {
		records = &runRecords{dir: c.Records, runDir: c.RunDir}
	}
	key := ""
	if c.Named != nil {
		key = "named-" + c.Named.Name
	}
	rec, err := records.reapAndClaim(key, pl.dir)
	if err != nil {
		return nil, err
	}
	args, url, ready, err := c.tunnelArgs(pl)
	if err != nil {
		rec.release()
		return nil, err
	}
	argv := append([]string{bin}, args...)
	if c.Guard != nil {
		argv = append(append(append([]string{}, c.Guard...), "--remove", pl.dir, "--"), argv...)
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		rec.release()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		rec.release()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		rec.release()
		return nil, err
	}
	h := &cloudflaredHandle{cmd: cmd, stdin: stdin, guarded: c.Guard != nil, exited: make(chan struct{}), record: rec}
	found := make(chan string, 1)
	go h.watchStderr(stderr, c.Guard != nil, func(line string) {
		if u := quickTunnelURL.FindString(line); ready == "" && u != "" {
			select {
			case found <- u:
			default:
			}
		}
		if ready != "" && strings.Contains(line, ready) {
			select {
			case found <- url:
			default:
			}
		}
	})
	go func() {
		_ = cmd.Wait()
		close(h.exited)
	}()

	timeout := c.OpenTimeout
	if timeout <= 0 {
		timeout = defaultOpenTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case u := <-found:
		h.url = u
		if err := rec.record(h.cmd.Process.Pid, h.childPID()); err != nil {
			h.Close()
			return nil, fmt.Errorf("broadcast: run record: %w", err)
		}
		return h, nil
	case <-h.exited:
		h.Close()
		return nil, fmt.Errorf("broadcast: cloudflared exited while opening: %s", h.lastLine())
	case <-timer.C:
		h.Close()
		return nil, fmt.Errorf("broadcast: cloudflared opened no tunnel within %v: %s", timeout, h.lastLine())
	case <-ctx.Done():
		h.Close()
		return nil, ctx.Err()
	}
}

// tunnelArgs builds cloudflared's arguments, and what stderr says when the
// tunnel is ready: a quick tunnel prints its URL; a named tunnel logs a
// registered connection and serves at its fixed hostname.
func (c Cloudflared) tunnelArgs(pl *privateListener) (args []string, url, ready string, err error) {
	if c.Named == nil {
		return []string{"tunnel", "--no-autoupdate", "--url", "http://" + pl.Addr().String()}, "", "", nil
	}
	service := "http://" + pl.Addr().String()
	if pl.socket != "" {
		service = "unix:" + pl.socket
	}
	config := filepath.Join(pl.dir, "config.yml")
	body := fmt.Sprintf("tunnel: %s\ningress:\n  - hostname: %s\n    service: %s\n  - service: http_status:404\n", c.Named.Name, c.Named.Hostname, service)
	if err := os.WriteFile(config, []byte(body), 0o600); err != nil {
		return nil, "", "", err
	}
	return []string{"tunnel", "--no-autoupdate", "--config", config, "run", c.Named.Name}, "https://" + c.Named.Hostname, namedRegistered, nil
}

type cloudflaredHandle struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	guarded bool
	url     string
	exited  chan struct{}
	record  *runRecord

	mu    sync.Mutex
	child int
	last  string
	once  sync.Once
}

func (h *cloudflaredHandle) URL() string             { return h.url }
func (h *cloudflaredHandle) Exited() <-chan struct{} { return h.exited }

// watchStderr reads the guard's child line, then hands every line to look.
// It drains to EOF so the process never blocks on a full pipe.
func (h *cloudflaredHandle) watchStderr(r io.Reader, guarded bool, look func(string)) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	first := guarded
	for s.Scan() {
		line := s.Text()
		if first {
			first = false
			if pid, ok := ParseGuardChild(line); ok {
				h.mu.Lock()
				h.child = pid
				h.mu.Unlock()
				continue
			}
		}
		if t := strings.TrimSpace(line); t != "" {
			h.mu.Lock()
			h.last = t
			h.mu.Unlock()
		}
		look(line)
	}
}

func (h *cloudflaredHandle) childPID() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.child
}

func (h *cloudflaredHandle) lastLine() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.last == "" {
		return "no output"
	}
	return h.last
}

// Close stops the tunnel. Guarded, closing the pipe is the whole signal: the
// guard stops cloudflared's group and exits. (A signal to the guard itself
// would kill it before it could, orphaning cloudflared.) Unguarded,
// cloudflared gets SIGTERM. Past the budget, both groups are killed directly.
// Idempotent.
func (h *cloudflaredHandle) Close() error {
	h.once.Do(func() {
		_ = h.stdin.Close()
		if !h.guarded {
			_ = syscall.Kill(-h.cmd.Process.Pid, syscall.SIGTERM)
		}
		select {
		case <-h.exited:
		case <-time.After(closeBudget):
			_ = syscall.Kill(-h.cmd.Process.Pid, syscall.SIGKILL)
			if child := h.childPID(); child > 0 {
				_ = syscall.Kill(-child, syscall.SIGKILL)
			}
			<-h.exited
		}
		h.record.release()
	})
	return nil
}
