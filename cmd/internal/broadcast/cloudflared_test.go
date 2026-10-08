package broadcast

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// shortDir is a directory this test creates and removes, short enough for a
// unix socket path (t.TempDir can exceed macOS's 104-byte sun_path).
func shortDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cb")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// fakeCloudflared writes an executable named cloudflared that runs body,
// with $@ being the real arguments. It records them in $DIR/argv.
func fakeCloudflared(t *testing.T, body string) (bin, dir string) {
	t.Helper()
	dir = shortDir(t)
	bin = filepath.Join(dir, "cloudflared")
	script := "#!/bin/sh\necho \"$@\" > " + filepath.Join(dir, "argv") + "\n" + body + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, dir
}

const (
	quickBanner = `echo "INF |  https://brave-test-tunnel.trycloudflare.com  |" >&2; exec sleep 300`
	namedReady  = `echo "INF Registered tunnel connection connIndex=0" >&2; exec sleep 300`
)

func guardArgv(t *testing.T) []string {
	t.Setenv("BROADCAST_GUARD_HELPER", "1")
	return []string{os.Args[0], "-test.run=^TestHelperGuardProcess$", "--"}
}

func openTunnel(t *testing.T, c Cloudflared) (net.Listener, Handle) {
	t.Helper()
	l, err := c.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	h, err := c.Open(context.Background(), l)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.Close() })
	return l, h
}

func TestQuickTunnelParsesURLOverTCP(t *testing.T) {
	bin, dir := fakeCloudflared(t, quickBanner)
	l, h := openTunnel(t, Cloudflared{Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t)})
	if h.URL() != "https://brave-test-tunnel.trycloudflare.com" {
		t.Fatalf("URL %q", h.URL())
	}
	if _, ok := l.Addr().(*net.TCPAddr); !ok {
		t.Fatalf("quick tunnel origin %T, want TCP (quick tunnels refuse sockets)", l.Addr())
	}
	argv := waitFile(t, filepath.Join(dir, "argv"))
	if argv != "tunnel --no-autoupdate --url http://"+l.Addr().String() {
		t.Fatalf("argv %q", argv)
	}
}

// needUnixSockets skips where unix-socket listeners are refused, as in a
// sandboxed run; Listen then falls back to TCP by design.
func needUnixSockets(t *testing.T) {
	t.Helper()
	path := filepath.Join(shortDir(t), "probe")
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Skipf("unix sockets unavailable here: %v", err)
	}
	l.Close()
}

// needPS skips where ps can't list processes (a sandbox); reaping checks
// commands through it.
func needPS(t *testing.T) {
	t.Helper()
	if procutil.Command(strconv.Itoa(os.Getpid())) == "" {
		t.Skip("ps unavailable here")
	}
}

func TestNamedTunnelServesFromPrivateSocket(t *testing.T) {
	needUnixSockets(t)
	bin, dir := fakeCloudflared(t, namedReady)
	named := &NamedTunnel{Name: "couch-broadcast", Hostname: "live.example.com"}
	l, h := openTunnel(t, Cloudflared{Named: named, Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t)})
	if h.URL() != "https://live.example.com" {
		t.Fatalf("URL %q", h.URL())
	}
	pl := l.(*privateListener)
	if pl.socket == "" || l.Addr().Network() != "unix" {
		t.Fatalf("named tunnel origin %v, want a unix socket", l.Addr())
	}
	info, err := os.Stat(pl.dir)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("private dir mode %v %v", info.Mode(), err)
	}
	config := filepath.Join(pl.dir, "config.yml")
	if argv := waitFile(t, filepath.Join(dir, "argv")); argv != "tunnel --no-autoupdate --config "+config+" run couch-broadcast" {
		t.Fatalf("argv %q", argv)
	}
	body, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	want := "tunnel: couch-broadcast\ningress:\n  - hostname: live.example.com\n    service: unix:" + pl.socket + "\n  - service: http_status:404\n"
	if string(body) != want {
		t.Fatalf("config:\n%s\nwant:\n%s", body, want)
	}
}

func TestListenFallsBackToTCPWhenSocketPathTooLong(t *testing.T) {
	long := filepath.Join(shortDir(t), strings.Repeat("d", 90))
	if err := os.MkdirAll(long, 0o700); err != nil {
		t.Fatal(err)
	}
	l, err := Cloudflared{Named: &NamedTunnel{Name: "n", Hostname: "h"}, RunDir: long}.Listen()
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if _, ok := l.Addr().(*net.TCPAddr); !ok {
		t.Fatalf("origin %v, want the TCP fallback", l.Addr())
	}
}

func TestListenerCloseRemovesPrivateDir(t *testing.T) {
	needUnixSockets(t)
	l, err := Cloudflared{Named: &NamedTunnel{Name: "n", Hostname: "h"}, RunDir: shortDir(t)}.Listen()
	if err != nil {
		t.Fatal(err)
	}
	dir := l.(*privateListener).dir
	l.Close()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("private dir survived Close: %v", err)
	}
}

func TestCloudflaredOpenFailures(t *testing.T) {
	t.Run("not installed", func(t *testing.T) {
		c := Cloudflared{Binary: filepath.Join(shortDir(t), "cloudflared"), RunDir: shortDir(t)}
		l, _ := c.Listen()
		defer l.Close()
		if _, err := c.Open(context.Background(), l); !errors.Is(err, ErrNoCloudflared) {
			t.Fatalf("err %v", err)
		}
	})
	t.Run("exits while opening", func(t *testing.T) {
		bin, _ := fakeCloudflared(t, `echo "ERR failed to connect: boom" >&2; exit 1`)
		c := Cloudflared{Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t)}
		l, _ := c.Listen()
		defer l.Close()
		_, err := c.Open(context.Background(), l)
		if err == nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("err %v", err)
		}
	})
	t.Run("opens nothing in time", func(t *testing.T) {
		bin, _ := fakeCloudflared(t, `exec sleep 300`)
		c := Cloudflared{Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t), OpenTimeout: 300 * time.Millisecond}
		l, _ := c.Listen()
		defer l.Close()
		if _, err := c.Open(context.Background(), l); err == nil || !strings.Contains(err.Error(), "no tunnel within") {
			t.Fatalf("err %v", err)
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		bin, _ := fakeCloudflared(t, `exec sleep 300`)
		c := Cloudflared{Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t)}
		l, _ := c.Listen()
		defer l.Close()
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(200*time.Millisecond, cancel)
		if _, err := c.Open(ctx, l); !errors.Is(err, context.Canceled) {
			t.Fatalf("err %v", err)
		}
	})
}

// Close through the guard stops cloudflared, by closing the pipe alone.
func TestCloudflaredCloseStopsTunnel(t *testing.T) {
	bin, _ := fakeCloudflared(t, namedReady)
	c := Cloudflared{Named: &NamedTunnel{Name: "t", Hostname: "h"}, Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t)}
	_, h := openTunnel(t, c)
	child := h.(*cloudflaredHandle).childPID()
	if child <= 0 {
		t.Fatal("guard reported no child")
	}
	start := time.Now()
	h.Close()
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("Close took %v", d)
	}
	waitDead(t, strconv.Itoa(child), 2*time.Second)
	select {
	case <-h.Exited():
	default:
		t.Fatal("Exited not closed after Close")
	}
}

func TestCloudflaredExitedWhenTunnelDies(t *testing.T) {
	bin, dir := fakeCloudflared(t, `echo "INF Registered tunnel connection" >&2; while [ ! -e `+"$0.die"+` ]; do sleep 0.05; done; exit 3`)
	_ = dir
	c := Cloudflared{Named: &NamedTunnel{Name: "t", Hostname: "h"}, Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t)}
	_, h := openTunnel(t, c)
	if err := os.WriteFile(bin+".die", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-h.Exited():
	case <-time.After(5 * time.Second):
		t.Fatal("Exited not closed when cloudflared died")
	}
}

func TestNamedTunnelIsExclusive(t *testing.T) {
	bin, _ := fakeCloudflared(t, namedReady)
	records := shortDir(t)
	c := Cloudflared{Named: &NamedTunnel{Name: "shared", Hostname: "h"}, Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t), Records: records}
	_, first := openTunnel(t, c)
	l, _ := c.Listen()
	defer l.Close()
	if _, err := c.Open(context.Background(), l); !errors.Is(err, ErrTunnelBusy) {
		t.Fatalf("second broadcaster on one tunnel: %v", err)
	}
	first.Close()
	l2, _ := c.Listen()
	defer l2.Close()
	h, err := c.Open(context.Background(), l2)
	if err != nil {
		t.Fatalf("after the first closed: %v", err)
	}
	h.Close()
	if left, _ := filepath.Glob(filepath.Join(records, "*.json")); len(left) != 0 {
		t.Fatalf("records left behind: %v", left)
	}
}

// startOrphan runs a process named cloudflared in its own group, standing in
// for a tunnel whose owner and guard are gone.
func startOrphan(t *testing.T) int {
	t.Helper()
	bin, _ := fakeCloudflared(t, `exec sleep 300`)
	cmd := exec.Command(bin)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go cmd.Wait()
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(procutil.Command(strconv.Itoa(cmd.Process.Pid)), "cloudflared") {
		if time.Now().After(deadline) {
			t.Fatal("orphan never showed its command")
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cmd.Process.Pid
}

// deadPID is a PID that was just alive and has exited.
func deadPID(t *testing.T) (int, string) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	id := identity(pid)
	_ = cmd.Wait()
	return pid, id
}

func writeRecord(t *testing.T, dir, name string, d recordData) string {
	t.Helper()
	b, _ := json.Marshal(d)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReapOrphans(t *testing.T) {
	if identity(os.Getpid()) == "" {
		t.Skip("no process identity on this platform; reaping destroys nothing")
	}
	needPS(t)
	t.Run("dead owner: orphan killed, record and dir removed", func(t *testing.T) {
		records := shortDir(t)
		orphan := startOrphan(t)
		runDir := shortDir(t)
		private, _ := os.MkdirTemp(runDir, privateDirPrefix)
		owner, ownerID := deadPID(t)
		rec := writeRecord(t, records, "named-x.json", recordData{Owner: owner, OwnerID: ownerID, Tunnel: orphan, TunnelID: identity(orphan), PrivateDir: private})
		ReapOrphans(records, runDir)
		waitDead(t, strconv.Itoa(orphan), 2*time.Second)
		for _, p := range []string{rec, private} {
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				t.Fatalf("%s survived reaping", p)
			}
		}
	})
	t.Run("live owner: untouched", func(t *testing.T) {
		records := shortDir(t)
		orphan := startOrphan(t)
		rec := writeRecord(t, records, "named-y.json", recordData{Owner: os.Getpid(), OwnerID: identity(os.Getpid()), Tunnel: orphan, TunnelID: identity(orphan)})
		ReapOrphans(records, "")
		time.Sleep(100 * time.Millisecond)
		if !procutil.Alive(strconv.Itoa(orphan)) {
			t.Fatal("reaped a live owner's tunnel")
		}
		if _, err := os.Stat(rec); err != nil {
			t.Fatal("removed a live owner's record")
		}
	})
	t.Run("recycled pid: identity mismatch spares the process", func(t *testing.T) {
		records := shortDir(t)
		orphan := startOrphan(t)
		owner, ownerID := deadPID(t)
		writeRecord(t, records, "named-z.json", recordData{Owner: owner, OwnerID: ownerID, Tunnel: orphan, TunnelID: "darwin:0.0"})
		ReapOrphans(records, "")
		time.Sleep(100 * time.Millisecond)
		if !procutil.Alive(strconv.Itoa(orphan)) {
			t.Fatal("killed a process whose identity doesn't match the record")
		}
	})
	t.Run("wrong command: spared", func(t *testing.T) {
		records := shortDir(t)
		cmd := exec.Command("sleep", "300")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
		owner, ownerID := deadPID(t)
		writeRecord(t, records, "named-w.json", recordData{Owner: owner, OwnerID: ownerID, Tunnel: cmd.Process.Pid, TunnelID: identity(cmd.Process.Pid)})
		ReapOrphans(records, "")
		time.Sleep(100 * time.Millisecond)
		if !procutil.Alive(strconv.Itoa(cmd.Process.Pid)) {
			t.Fatal("killed a process that isn't cloudflared")
		}
	})
	t.Run("only a plain private dir in the run dir is removed", func(t *testing.T) {
		runDir := shortDir(t)
		elsewhere := shortDir(t)
		foreignName, _ := os.MkdirTemp(runDir, "other-")
		wrongParent, _ := os.MkdirTemp(elsewhere, privateDirPrefix)
		openMode, _ := os.MkdirTemp(runDir, privateDirPrefix)
		os.Chmod(openMode, 0o755)
		target, _ := os.MkdirTemp(elsewhere, "target-")
		link := filepath.Join(runDir, privateDirPrefix+"link")
		os.Symlink(target, link)
		for i, dir := range []string{foreignName, wrongParent, openMode, link} {
			records := shortDir(t)
			owner, ownerID := deadPID(t)
			writeRecord(t, records, "named-"+strconv.Itoa(i)+".json", recordData{Owner: owner, OwnerID: ownerID, PrivateDir: dir})
			ReapOrphans(records, runDir)
			if _, err := os.Lstat(dir); err != nil {
				t.Errorf("removed %s, which isn't plainly a broadcast private dir", dir)
			}
		}
		if _, err := os.Stat(target); err != nil {
			t.Fatal("followed a symlink out of the run dir")
		}
	})
	t.Run("no owner identity: record cleared when the owner is gone, nothing killed", func(t *testing.T) {
		records := shortDir(t)
		orphan := startOrphan(t)
		owner, _ := deadPID(t)
		rec := writeRecord(t, records, "named-u.json", recordData{Owner: owner, Tunnel: orphan, TunnelID: identity(orphan)})
		live := writeRecord(t, records, "named-l.json", recordData{Owner: os.Getpid()})
		ReapOrphans(records, "")
		if _, err := os.Stat(rec); !os.IsNotExist(err) {
			t.Fatal("a dead owner's identity-less record stayed, keeping its tunnel busy")
		}
		if _, err := os.Stat(live); err != nil {
			t.Fatal("removed a live owner's identity-less record")
		}
		time.Sleep(100 * time.Millisecond)
		if !procutil.Alive(strconv.Itoa(orphan)) {
			t.Fatal("killed a process on an identity-less record")
		}
	})
}

// BR-21: when cloudflared can't reach its API, its failure line names the
// API host; that must not pass for a tunnel URL.
func TestQuickTunnelAPIFailureIsNotAURL(t *testing.T) {
	bin, _ := fakeCloudflared(t, `echo 'ERR failed to request quick Tunnel: Post "https://api.trycloudflare.com/tunnel": dial tcp: lookup api.trycloudflare.com: no such host' >&2; exit 1`)
	c := Cloudflared{Binary: bin, Guard: guardArgv(t), RunDir: shortDir(t)}
	l, _ := c.Listen()
	defer l.Close()
	h, err := c.Open(context.Background(), l)
	if err == nil {
		h.Close()
		t.Fatalf("opened %q from a failure line", h.URL())
	}
	if !strings.Contains(err.Error(), "failed to request quick Tunnel") {
		t.Fatalf("err %v, want cloudflared's own failure", err)
	}
	for _, line := range []string{"https://api.trycloudflare.com/tunnel", "https://trycloudflare.com"} {
		if u := quickTunnelURL.FindString(line); u != "" {
			t.Errorf("%q matched as a tunnel URL: %q", line, u)
		}
	}
	if u := quickTunnelURL.FindString("|  https://brave-test-tunnel.trycloudflare.com  |"); u != "https://brave-test-tunnel.trycloudflare.com" {
		t.Errorf("banner URL not found: %q", u)
	}
}

// BR-22: two Couches judge the same stale named-tunnel lock. A reads it and
// is held there (afterRecordRead) while B runs; without one lock around
// reap+claim, B claims and A then deletes B's fresh lock and claims too.
// With it, B waits for A and finds the tunnel taken. Deterministic.
func TestRunRecordsConcurrentClaimHasOneWinner(t *testing.T) {
	records := shortDir(t)
	owner, ownerID := deadPID(t)
	writeRecord(t, records, "named-shared.json", recordData{Owner: owner, OwnerID: ownerID})
	aRead := make(chan struct{})
	release := make(chan struct{})
	// Only the first reader (A) is held. Not sync.Once: its Do makes later
	// callers wait for the first, which would serialize B behind A and hide
	// the race this test exists to show.
	var first atomic.Bool
	afterRecordRead = func() {
		if first.CompareAndSwap(false, true) {
			close(aRead)
			<-release
		}
	}
	t.Cleanup(func() { afterRecordRead = nil })
	results := make(chan error, 2)
	go func() {
		_, err := (&runRecords{dir: records}).reapAndClaim("named-shared", "")
		results <- err
	}()
	<-aRead
	bDone := make(chan error, 1)
	go func() {
		_, err := (&runRecords{dir: records}).reapAndClaim("named-shared", "")
		bDone <- err
	}()
	// Give B every chance to run to completion if nothing stops it.
	select {
	case err := <-bDone:
		results <- err
		close(release)
	case <-time.After(300 * time.Millisecond):
		close(release)
		results <- <-bDone
	}
	won := 0
	for range 2 {
		if err := <-results; err == nil {
			won++
		} else if !errors.Is(err, ErrTunnelBusy) {
			t.Fatal(err)
		}
	}
	if won != 1 {
		t.Fatalf("%d claimers took one tunnel", won)
	}
}

func TestRunRecordWriteFailureIsAnError(t *testing.T) {
	rec := &runRecord{path: filepath.Join(shortDir(t), "missing", "named-x.json")}
	if err := rec.record(1, 2); err == nil {
		t.Fatal("record into a missing directory reported success")
	}
}

// A crash between a record's write and rename leaves a .tmp; the next reap
// removes it, and a live record beside it stays.
func TestReapRemovesLeftoverTempRecords(t *testing.T) {
	records := shortDir(t)
	tmp := filepath.Join(records, "named-x.json.tmp")
	if err := os.WriteFile(tmp, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	live := writeRecord(t, records, "named-y.json", recordData{Owner: os.Getpid(), OwnerID: identity(os.Getpid())})
	ReapOrphans(records, "")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatal("leftover .tmp survived reaping")
	}
	if _, err := os.Stat(live); err != nil {
		t.Fatal("live record removed")
	}
}
