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

func testNamespace(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "pnt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	t.Setenv("PAIR_NOTIFY_SOCKET_DIR", root)
	if got := rootDirectory(); got != root {
		t.Fatalf("namespace ignored: got %q want %q", got, root)
	}
	return root
}

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
	testNamespace(t)
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
	base := socketAddressIn("/tmp/pair-notify-501", "/exact/binding", 123)
	if base != socketAddressIn("/tmp/pair-notify-501", "/exact/binding", 123) || len(base) > 100 {
		t.Fatalf("unstable/long address %q", base)
	}
	for _, other := range []string{socketAddressIn("/tmp/pair-notify-501", "/other/binding", 123), socketAddressIn("/tmp/pair-notify-501", "/exact/binding", 124), socketAddressIn("/tmp/pair-notify-502", "/exact/binding", 123)} {
		if other == base {
			t.Fatalf("identity collision: %q", other)
		}
	}
	if !strings.HasPrefix(base, "/tmp/pair-notify-501/") {
		t.Fatalf("UID-private address=%q", base)
	}
}
func TestBrokerRejectsOversizeAndDiagnosesQueueOverflow(t *testing.T) {
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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
	testNamespace(t)
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

func TestBrokerCrashHelper(t *testing.T) {
	mode := os.Getenv("PAIR_NOTIFY_CRASH_HELPER")
	if mode == "" {
		return
	}
	if os.Getenv("PAIR_NOTIFY_SOCKET_DIR") == "" {
		os.Exit(30)
	}
	b, err := Start(os.Getenv("PAIR_NOTIFY_TEST_BINDING"), os.Getpid())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(31)
	}
	if mode == "failed-close" {
		unlock, err := lockDirectory()
		if err != nil {
			os.Exit(32)
		}
		err = b.Close()
		unlock()
		if err == nil {
			os.Exit(33)
		}
	}
	// Simulate process death without resource defers, including failed cleanup.
	os.Exit(0)
}

func TestSweepReclaimsCrashSocketWithoutPIDBinding(t *testing.T) {
	for _, mode := range []string{"crash", "failed-close"} {
		t.Run(mode, func(t *testing.T) {
			testNamespace(t)
			binding := filepath.Join(t.TempDir(), "pid")
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(exe, "-test.run=^TestBrokerCrashHelper$")
			cmd.Env = append(os.Environ(), "PAIR_NOTIFY_CRASH_HELPER="+mode, "PAIR_NOTIFY_TEST_BINDING="+binding)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("crash helper: %v %s", err, output)
			}
			pid := cmd.Process.Pid
			socket, err := address(binding, pid)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = os.Lstat(socket); err != nil {
				t.Fatalf("fixture has no residue: %v", err)
			}
			if err = os.Remove(binding); err != nil {
				t.Fatal(err)
			}
			_, b := startTestBroker(t)
			_ = b
			if _, err = os.Lstat(socket); !os.IsNotExist(err) {
				t.Fatalf("dead socket without binding survived: %v", err)
			}
		})
	}
}

func TestNamespacesIsolateLockSendAndClose(t *testing.T) {
	first := testNamespace(t)
	binding, b := startTestBroker(t)
	original, _ := address(binding, os.Getpid())
	unlock, err := lockDirectory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { unlock() }()
	second, err := os.MkdirTemp("/tmp", "pnt-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(second)
	t.Setenv("PAIR_NOTIFY_SOCKET_DIR", second)
	if err = Send(binding, "wrong namespace"); err == nil {
		t.Fatal("cross-namespace Send reached broker")
	}
	other, err := Start(filepath.Join(t.TempDir(), "pid"), os.Getpid())
	if err != nil {
		t.Fatalf("other namespace blocked by lock in %s: %v", first, err)
	}
	defer other.Close()
	unlock()
	unlock = func() {}
	if err = b.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Lstat(original); !os.IsNotExist(err) {
		t.Fatalf("Close used changed namespace: %v", err)
	}
	if _, err = os.Lstat(binding); !os.IsNotExist(err) {
		t.Fatalf("Close lost binding: %v", err)
	}
}

func TestSweepPreservesLiveAndForeignEntriesAndBoundsCapacity(t *testing.T) {
	root := testNamespace(t)
	pid := deadPID(t)
	owner := exec.Command("/bin/sleep", "30")
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = owner.Process.Kill(); _ = owner.Wait() }()
	otherLive := socketAddressIn(root, "/other-live-owner", owner.Process.Pid)
	liveSocket, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: otherLive, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer liveSocket.Close()
	foreign := filepath.Join(root, "foreign.sock")
	c, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: foreign, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	c.Close()
	regular := socketAddressIn(root, "/foreign-regular", pid)
	if err = os.WriteFile(regular, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	link := socketAddressIn(root, "/foreign-link", pid)
	if err = os.Symlink(foreign, link); err != nil {
		t.Fatal(err)
	}
	binding, live := startTestBroker(t)
	_, other := startTestBroker(t)
	_ = other
	for _, path := range []string{foreign, regular, link, otherLive} {
		if _, err = os.Lstat(path); err != nil {
			t.Fatalf("foreign entry removed: %s %v", path, err)
		}
	}
	if err = Send(binding, "still live"); err != nil {
		t.Fatal(err)
	}
	receive(t, live, "still live")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for i := len(entries); i < 1024; i++ {
		if err = os.WriteFile(filepath.Join(root, fmt.Sprintf("foreign-%d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if b, err := Start(filepath.Join(t.TempDir(), "pid"), os.Getpid()); err == nil {
		b.Close()
		t.Fatal("namespace admitted beyond1024entry capacity")
	}
}

func TestPureSocketFilenameOwnerGrammar(t *testing.T) {
	hash := strings.Repeat("a", 32)
	for _, test := range []struct {
		name  string
		pid   int
		valid bool
	}{
		{hash + "-1.sock", 1, true}, {hash + "-2147483647.sock", 2147483647, true},
		{hash + "-0.sock", 0, false}, {hash + "-01.sock", 0, false}, {hash + "-+1.sock", 0, false},
		{hash + "--1.sock", 0, false}, {hash + "-2147483648.sock", 0, false},
		{strings.Repeat("A", 32) + "-1.sock", 0, false}, {strings.Repeat("z", 32) + "-1.sock", 0, false},
		{hash + "-1.sock.extra", 0, false}, {"foreign.sock", 0, false},
	} {
		pid, valid := socketOwnerPID(test.name)
		if valid != test.valid || (valid && pid != test.pid) {
			t.Fatalf("%q = %d,%t", test.name, pid, valid)
		}
	}
}

func TestNamespaceCapacityIncludesInNamespacePIDBinding(t *testing.T) {
	root := testNamespace(t)
	for i := 0; i < 1022; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("foreign-%d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Lock is the1023rdentry; a socket AND its binding would exceed1024.
	if b, err := Start(filepath.Join(root, "pid"), os.Getpid()); err == nil {
		b.Close()
		t.Fatal("two-entry broker exceeded capacity")
	}
	if _, err := os.Stat(filepath.Join(root, "pid")); !os.IsNotExist(err) {
		t.Fatalf("capacity refusal published binding: %v", err)
	}
}
