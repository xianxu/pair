package notifytransport

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func startTestBroker(t *testing.T) (string, *Broker) {
	t.Helper()
	binding := filepath.Join(t.TempDir(), "wrapper-pid")
	b, err := Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := b.Close(); err != nil {
			t.Error(err)
		}
	})
	return binding, b
}
func receive(t *testing.T, b *Broker, want string) {
	t.Helper()
	select {
	case got := <-b.Messages():
		if got != want {
			t.Fatalf("message=%q want=%q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("no broker message")
	}
}
func TestBrokerRoundTripIsolationAndJoinedClose(t *testing.T) {
	a, ba := startTestBroker(t)
	raw, err := os.ReadFile(a)
	if err != nil || string(raw) != fmt.Sprint(os.Getpid()) {
		t.Fatalf("legacy PID binding=%q err=%v", raw, err)
	}
	b, bb := startTestBroker(t)
	for _, s := range []string{"ready", strings.Repeat("界", 1365) + "a", "bad\x1b\x07text"} {
		if err := Send(a, s); err != nil {
			t.Fatal(err)
		}
		want := s
		if strings.HasPrefix(s, "bad") {
			want = "badtext"
		}
		receive(t, ba, want)
	}
	if err := Send(b, "other"); err != nil {
		t.Fatal(err)
	}
	receive(t, bb, "other")
	addr, err := address(a, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := ba.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(addr); !os.IsNotExist(err) {
		t.Fatalf("socket retained: %v", err)
	}
	if _, err := os.Lstat(a); !os.IsNotExist(err) {
		t.Fatalf("binding retained: %v", err)
	}
	if _, ok := <-ba.Messages(); ok {
		t.Fatal("messages not closed after join")
	}
	if err := Send(a, "late"); err == nil {
		t.Fatal("closed broker accepted")
	}
}

func TestPureSocketAddressIdentity(t *testing.T) {
	base := socketAddress("/exact/binding", 123, 501)
	if base != socketAddress("/exact/binding", 123, 501) || len(base) > 100 {
		t.Fatalf("unstable/long address %q", base)
	}
	for _, other := range []string{socketAddress("/other/binding", 123, 501), socketAddress("/exact/binding", 124, 501), socketAddress("/exact/binding", 123, 502)} {
		if other == base {
			t.Fatalf("identity collision: %q", other)
		}
	}
	if !strings.HasPrefix(base, "/tmp/pair-notify-501/") {
		t.Fatalf("UID-private address=%q", base)
	}
}
func TestBrokerRejectsOversizeAndDiagnosesQueueOverflow(t *testing.T) {
	binding, b := startTestBroker(t)
	addr, _ := address(binding, os.Getpid())
	c, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: addr, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.SetWriteBuffer(8194); err != nil {
		t.Fatal(err)
	}
	c.SetWriteDeadline(time.Now().Add(time.Second))
	if _, err = c.Write([]byte(strings.Repeat("x", 4097))); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-b.Diagnostics():
		if err == nil {
			t.Fatal("missing oversized diagnostic")
		}
	case <-time.After(time.Second):
		t.Fatal("no oversized diagnostic")
	}
	if len(b.Messages()) != 0 {
		t.Fatal("oversize was truncated and delivered")
	}
	for i := 0; i < 33; i++ {
		if err := Send(binding, fmt.Sprint(i)); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-b.Diagnostics():
	case <-time.After(time.Second):
		t.Fatal("no overflow diagnostic")
	}
	if len(b.Messages()) != 32 {
		t.Fatalf("queue=%d", len(b.Messages()))
	}
	for i := 0; i < 32; i++ {
		receive(t, b, fmt.Sprint(i))
	}
	if err := Send(binding, "recovered"); err != nil {
		t.Fatal(err)
	}
	receive(t, b, "recovered")
}
func TestBrokerRefusesLiveBindingAndUnsafePaths(t *testing.T) {
	binding, b := startTestBroker(t)
	if other, err := Start(binding, os.Getpid()); err == nil {
		other.Close()
		t.Fatal("replaced live broker")
	}
	if err := Send(binding, "still-live"); err != nil {
		t.Fatal(err)
	}
	receive(t, b, "still-live")
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(binding, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(alias, os.Getpid()); err == nil {
		t.Fatal("symlink binding accepted")
	}
	if err := Send(alias, "wrong"); err == nil {
		t.Fatal("sender accepted symlink")
	}
	for _, pid := range []string{"", "0", "-1", "12 junk", strings.Repeat("1", 100)} {
		p := filepath.Join(t.TempDir(), "pid")
		os.WriteFile(p, []byte(pid), 0600)
		if err := Send(p, "x"); err == nil {
			t.Fatalf("PID %q accepted", pid)
		}
	}
	if _, err := Start(filepath.Join(t.TempDir(), "pid"), 0); err == nil {
		t.Fatal("zero PID accepted")
	}
}
func TestCloseDoesNotRemoveReplacementBinding(t *testing.T) {
	binding, b := startTestBroker(t)
	replacement := binding + ".new"
	os.WriteFile(replacement, []byte("replacement"), 0600)
	os.Rename(replacement, binding)
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(binding)
	if err != nil || string(got) != "replacement" {
		t.Fatalf("replacement removed: %q %v", got, err)
	}
}

func TestAddressCanonicalIsolationAndPrivateDirectory(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	a, err := address(filepath.Join(real, "pid"), os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	b, err := address(filepath.Join(alias, "pid"), os.Getpid())
	if err != nil || a != b {
		t.Fatalf("canonical address %q %q %v", a, b, err)
	}
	c, _ := address(filepath.Join(real, "other"), os.Getpid())
	if a == c || len(a) > 100 {
		t.Fatalf("address collision/length: %q %q", a, c)
	}
	unsafe := filepath.Join(root, "unsafe")
	os.Mkdir(unsafe, 0755)
	if err := privateDirectory(unsafe); err == nil {
		t.Fatal("public directory accepted")
	}
	if err := privateDirectory(alias); err == nil {
		t.Fatal("symlink directory accepted")
	}
}

func deadPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("/usr/bin/true")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	if alive(pid) {
		t.Fatal("completed subprocess still alive")
	}
	return pid
}
func TestBrokerReclaimsOnlyVerifiedDeadSocket(t *testing.T) {
	binding := filepath.Join(t.TempDir(), "pid")
	pid := deadPID(t)
	socket, err := address(binding, pid)
	if err != nil {
		t.Fatal(err)
	}
	old, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	old.Close()
	t.Cleanup(func() { os.Remove(socket) })
	if err := os.WriteFile(binding, []byte(fmt.Sprint(pid)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Send(binding, "dead"); err == nil {
		t.Fatal("dead PID accepted")
	}
	b, err := Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("dead socket retained: %v", err)
	}
	if err := Send(binding, "new"); err != nil {
		t.Fatal(err)
	}
	receive(t, b, "new")
}
func TestUnsafeStaleAndOccupiedPathsAreNeverUnlinked(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(fmt.Sprint(stale), func(t *testing.T) {
			binding := filepath.Join(t.TempDir(), "pid")
			pid := os.Getpid()
			if stale {
				pid = deadPID(t)
				os.WriteFile(binding, []byte(fmt.Sprint(pid)), 0600)
			}
			socket, _ := address(binding, pid)
			if err := os.WriteFile(socket, []byte("do not unlink"), 0600); err != nil {
				t.Fatal(err)
			}
			defer os.Remove(socket)
			if b, err := Start(binding, os.Getpid()); err == nil {
				b.Close()
				t.Fatal("occupied unsafe path accepted")
			}
			raw, err := os.ReadFile(socket)
			if err != nil || string(raw) != "do not unlink" {
				t.Fatalf("foreign path modified: %q %v", raw, err)
			}
			if !stale {
				if _, err := os.Lstat(binding); !os.IsNotExist(err) {
					t.Fatalf("failed Start published binding: %v", err)
				}
			}
		})
	}
}
func TestSendToUnavailableReaderIsBounded(t *testing.T) {
	binding := filepath.Join(t.TempDir(), "pid")
	socket, _ := address(binding, os.Getpid())
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	defer os.Remove(socket)
	conn.SetReadBuffer(8194)
	os.WriteFile(binding, []byte(fmt.Sprint(os.Getpid())), 0600)
	start := time.Now()
	var sendErr error
	for i := 0; i < 100; i++ {
		sendErr = Send(binding, strings.Repeat("x", 4096))
		if sendErr != nil {
			break
		}
	}
	if sendErr == nil {
		t.Fatal("unread socket never backpressured")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("sender exceeded bounded deadline: %v", elapsed)
	}
}

func TestConcurrentStartsKeepOnePublishedOwner(t *testing.T) {
	binding := filepath.Join(t.TempDir(), "pid")
	pid := deadPID(t)
	os.WriteFile(binding, []byte(fmt.Sprint(pid)), 0600)
	var wg sync.WaitGroup
	results := make(chan *Broker, 16)
	start := make(chan struct{})
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			b, err := Start(binding, os.Getpid())
			if err == nil {
				results <- b
			}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	var winner *Broker
	for b := range results {
		if winner != nil {
			b.Close()
			winner.Close()
			t.Fatal("multiple successful owners")
		}
		winner = b
	}
	if winner == nil {
		t.Fatal("no owner published")
	}
	defer winner.Close()
	if err := Send(binding, "winner"); err != nil {
		t.Fatal(err)
	}
	receive(t, winner, "winner")
}

func TestDirectoryLockBoundsStartupAndPreservesBinding(t *testing.T) {
	unlock, err := lockDirectory()
	if err != nil {
		t.Fatal(err)
	}
	binding := filepath.Join(t.TempDir(), "pid")
	start := time.Now()
	b, err := Start(binding, os.Getpid())
	unlock()
	if err == nil {
		b.Close()
		t.Fatal("Start ignored existing directory lock")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("lock wait unbounded: %v", elapsed)
	}
	if _, err := os.Lstat(binding); !os.IsNotExist(err) {
		t.Fatalf("timed out Start published: %v", err)
	}
	b, err = Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err = b.Close(); err != nil {
		t.Fatal(err)
	}
	// A fresh listener at the exact same binding is followed without cached PID/socket state.
	b, err = Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if err = Send(binding, "restarted"); err != nil {
		t.Fatal(err)
	}
	receive(t, b, "restarted")
}

func TestBrokerRejectsInvalidDatagramsWithoutPoisoningNext(t *testing.T) {
	binding, b := startTestBroker(t)
	socket, _ := address(binding, os.Getpid())
	c, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, raw := range [][]byte{{0xff}, []byte("bad\x1btext"), {}} {
		if _, err = c.Write(raw); err != nil {
			t.Fatal(err)
		}
		select {
		case <-b.Diagnostics():
		case <-time.After(time.Second):
			t.Fatalf("missing rejection: %q", raw)
		}
	}
	if len(b.Messages()) != 0 {
		t.Fatal("invalid messages delivered")
	}
	if err = Send(binding, "valid"); err != nil {
		t.Fatal(err)
	}
	receive(t, b, "valid")
}

func TestPublicationFailureRemovesReadySocket(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses directory write permission")
	}
	parent := t.TempDir()
	binding := filepath.Join(parent, "pid")
	socket, err := address(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(parent, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(parent, 0700)
	if b, err := Start(binding, os.Getpid()); err == nil {
		b.Close()
		t.Fatal("published in unwritable directory")
	}
	if _, err = os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("ready socket leaked on failed publication: %v", err)
	}
}
