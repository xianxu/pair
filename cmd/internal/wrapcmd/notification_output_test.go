package wrapcmd

import (
	"bytes"
	"errors"
	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

func TestNotificationOutputEveryBoundary(t *testing.T) {
	for _, wire := range []string{"界", "\x1b[31m", "\x1b]0;title\x1b\\", "\x1bPqdata\x1b\\", "\x1b_apc\x1b\\", "\x1bXsos\x1b\\", "\x1b^pm\x1b\\"} {
		for split := 1; split < len(wire); split++ {
			var out bytes.Buffer
			p := newStdoutPump(&out)
			p.queue([]byte(wire[:split]))
			p.notify("ready", time.Now(), nil)
			p.flush("partial")
			if !bytes.Equal(out.Bytes(), []byte(wire[:split])) {
				t.Fatalf("%q split%d early injection %q", wire, split, out.Bytes())
			}
			p.queue([]byte(wire[split:]))
			p.flush("complete")
			want := append([]byte(wire), notifyosc.Encode("ready")...)
			if !bytes.Equal(out.Bytes(), want) {
				t.Fatalf("%q split%d got%q", wire, split, out.Bytes())
			}
		}
	}
}
func TestNotificationOutputMalformedBoundedExpiryEOF(t *testing.T) {
	var out bytes.Buffer
	p := newStdoutPump(&out)
	p.queue([]byte{0xe2, 0x1b, '['})
	if p.notify("unsafe", time.Now(), nil) == nil {
		t.Fatal("malformed stream accepted notification")
	}
	p.queue([]byte("31m"))
	p.flush("eof")
	if !bytes.Equal(out.Bytes(), []byte("\xe2\x1b[31m")) {
		t.Fatal("child bytes changed")
	}
	p = newStdoutPump(&out)
	p.queue([]byte("\x1b]0;"))
	now := time.Now()
	for i := 0; i < 32; i++ {
		if err := p.notify("pending", now, nil); err != nil {
			t.Fatal(err)
		}
	}
	if p.notify("overflow", now, nil) == nil {
		t.Fatal("unbounded queue")
	}
	p.expire(now.Add(2 * time.Second))
	if len(p.notifications) != 0 {
		t.Fatal("expiry retained notifications")
	}
	p.notify("eof", now, nil)
	p.finish()
	if len(p.notifications) != 0 {
		t.Fatal("EOF retained notification")
	}
}

type prefixWriter struct {
	bytes.Buffer
	limit int
	fail  error
}

func (w *prefixWriter) Write(b []byte) (int, error) {
	if len(b) > w.limit {
		b = b[:w.limit]
	}
	n, _ := w.Buffer.Write(b)
	return n, w.fail
}
func TestNotificationOutputPartialWrites(t *testing.T) {
	for _, terminalErr := range []error{nil, io.ErrClosedPipe} {
		w := &prefixWriter{limit: 3, fail: terminalErr}
		p := newStdoutPump(w)
		complete := false
		p.notify("ready", time.Now(), func() { complete = true })
		r := p.flush("test")
		if terminalErr == nil {
			if !complete || !bytes.Equal(w.Bytes(), notifyosc.Encode("ready")) || r.Err != nil {
				t.Fatalf("incomplete delivery: %+v %q", r, w.Bytes())
			}
		} else {
			if complete || !errors.Is(r.Err, terminalErr) || w.Len() != 3 {
				t.Fatalf("false completion %+v", r)
			}
			p.queue([]byte("later"))
			p.flush("again")
			if w.Len() != 3 {
				t.Fatal("replayed failed prefix")
			}
		}
	}
}
func TestNotificationRewriterContainsAndOrders(t *testing.T) {
	for _, wire := range []string{"\x1bPq\x1b]9;fake\x07\x1b\\", "\x1b_apc\x1b]9;fake\x07\x1b\\"} {
		var r NotificationRewriter
		got := r.Feed([]byte(wire), true)
		if len(got.Notifications) != 0 || string(got.Passthrough) != wire {
			t.Fatalf("nested notification escaped: %+v", got)
		}
	}
	for split := 0; split <= len("a\x1b]9;ready\x07b"); split++ {
		wire := "a\x1b]9;ready\x07b"
		var r NotificationRewriter
		var ordered string
		for _, chunk := range []string{wire[:split], wire[split:]} {
			for _, e := range r.Feed([]byte(chunk), true).Events {
				ordered += string(e.Passthrough)
				if e.Notification != nil {
					ordered += "<" + e.Notification.Message + ">"
				}
			}
		}
		if ordered != "a<ready>b" {
			t.Fatalf("split%d = %q", split, ordered)
		}
	}
}

// These lifecycle fixtures observe the production stdout stream and retain only
// decoded notification envelopes, so ordinary paint does not pollute assertions.
type notificationRecordingWriter struct {
	rewrite NotificationRewriter
	write   func([]byte) (int, error)
}

func notificationWriter(write func([]byte) (int, error)) io.Writer {
	return &notificationRecordingWriter{write: write}
}
func (w *notificationRecordingWriter) Write(data []byte) (int, error) {
	for _, n := range w.rewrite.Feed(data, true).Notifications {
		if _, err := w.write(notifyosc.Encode(n.Message)); err != nil {
			return 0, err
		}
	}
	return len(data), nil
}
func notificationFileWriter(t *testing.T, path string) io.Writer {
	t.Helper()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return notificationWriter(f.Write)
}

func TestNotificationOutputBoundsSafeReceiptsAndExpiresBeforeBoundary(t *testing.T) {
	p := newStdoutPump(io.Discard)
	for i := 0; i < 32; i++ {
		if err := p.notify("ready", time.Now(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if p.notify("overflow", time.Now(), nil) == nil || len(p.receipts) != 32 {
		t.Fatal("safe unflushed admission unbounded")
	}
	p.flush("test")
	if len(p.receipts) != 0 {
		t.Fatal("completed receipts retained")
	}
	var out bytes.Buffer
	p = newStdoutPump(&out)
	p.queue([]byte("\x1b["))
	p.notify("expired", time.Now().Add(-3*time.Second), nil)
	p.queue([]byte("m"))
	p.flush("test")
	if out.String() != "\x1b[m" {
		t.Fatalf("expired notification delivered: %q", out.String())
	}
}
func TestNotificationOutputControlCancellationAndMalformedUTF8(t *testing.T) {
	for _, abort := range []byte{0x18, 0x1a} {
		for _, start := range []string{"\x1b[12;", "\x1b]title", "\x1bPqpayload", "\x1b_apc"} {
			var out bytes.Buffer
			p := newStdoutPump(&out)
			p.queue([]byte(start))
			p.notify("ready", time.Now(), nil)
			p.queue([]byte{abort})
			p.flush("test")
			want := append(append([]byte(start), abort), notifyosc.Encode("ready")...)
			if !bytes.Equal(out.Bytes(), want) {
				t.Fatalf("abort %x %q: %q", abort, start, out.Bytes())
			}
		}
	}
	for _, invalid := range [][]byte{{0x80}, {0xc0, 0xaf}, {0xe0, 0x80, 0x80}, {0xf4, 0x90, 0x80, 0x80}, {0xe2, 0x1b, '['}} {
		var out bytes.Buffer
		p := newStdoutPump(&out)
		for _, b := range invalid {
			p.queue([]byte{b})
		}
		if p.notify("unsafe", time.Now(), nil) == nil {
			t.Fatalf("accepted %x", invalid)
		}
		p.flush("test")
		if !bytes.Equal(out.Bytes(), invalid) {
			t.Fatal("changed malformed bytes")
		}
	}
}
func TestNotificationObserverSeesExactQueuedWire(t *testing.T) {
	var wire, observed bytes.Buffer
	p := newStdoutPump(&wire)
	p.observe = func(b []byte) { observed.Write(b) }
	p.queue([]byte("\x1b["))
	p.notify("ready", time.Now(), nil)
	p.queue([]byte("m界"))
	p.flush("test")
	if !bytes.Equal(wire.Bytes(), observed.Bytes()) {
		t.Fatalf("observer diverged: %q vs %q", wire.Bytes(), observed.Bytes())
	}
}
func TestNotificationRewriterEOFAndHugeContainingString(t *testing.T) {
	for _, wire := range []string{"\x1b", "\x1b]9;incomplete", "\x1bPq" + string(bytes.Repeat([]byte("\x1b]9;fake\a"), 10000)) + "\x1b\\"} {
		var r NotificationRewriter
		var got []byte
		for start := 0; start < len(wire); start += 17 {
			end := start + 17
			if end > len(wire) {
				end = len(wire)
			}
			v := r.Feed([]byte(wire[start:end]), true)
			got = append(got, v.Passthrough...)
			if len(v.Notifications) != 0 {
				t.Fatal("contained notification escaped")
			}
			if len(r.pending) > notificationRewriteMaxPending {
				t.Fatal("unbounded pending")
			}
		}
		got = append(got, r.Finish()...)
		if string(got) != wire {
			t.Fatal("EOF changed child bytes")
		}
	}
}

func TestNotificationBrokerClosedWhileMasterLives(t *testing.T) {
	socketDir := isolateNotificationSockets(t)
	binding := filepath.Join(t.TempDir(), "pair-wrap-pid")
	broker, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { broker.Close() })
	entries, err := os.ReadDir(socketDir)
	if err != nil {
		t.Fatal(err)
	}
	foundSocket, foundLock := false, false
	for _, entry := range entries {
		if entry.Type()&os.ModeSocket != 0 {
			foundSocket = true
		}
		if entry.Name() == "lock" {
			foundLock = true
		}
	}
	if !foundSocket || !foundLock {
		t.Fatalf("broker escaped private namespace: socket=%t lock=%t", foundSocket, foundLock)
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	slugCalls := 0
	delivered := make(chan struct{}, 1)
	var out bytes.Buffer
	p := &proxy{ptmx: reader, notificationBroker: broker, spawnSlug: func() { slugCalls++ }, stdout: writerFunc(func(data []byte) (int, error) {
		n, e := out.Write(data)
		if bytes.Contains(data, notifyosc.Encode("hook ready")) {
			select {
			case delivered <- struct{}{}:
			default:
			}
		}
		return n, e
	})}
	done := make(chan struct{})
	go func() { p.masterPump(); close(done) }()
	t.Cleanup(func() {
		broker.Close()
		writer.Close()
		reader.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("master loop did not join")
		}
	})
	if err := notifytransport.Send(binding, "hook ready"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-delivered:
	case <-time.After(time.Second):
		t.Fatal("hook not delivered")
	}
	if err := broker.Close(); err != nil {
		t.Fatal(err)
	}
	writer.Write([]byte("after broker close"))
	writer.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("closed broker blocked output")
	}
	if !bytes.Contains(out.Bytes(), []byte("after broker close")) {
		t.Fatal("child output lost")
	}
	if slugCalls != 0 {
		t.Fatal("hook triggered model slug refresh")
	}
	if p.notificationLifecycle.Active || p.notificationLifecycle.Completed {
		t.Fatal("hook changed turn lifecycle")
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(b []byte) (int, error) { return f(b) }

func TestNotificationHookChild(t *testing.T) {
	if os.Getenv("PAIR_TEST_EARLY_NOTIFY") != "1" {
		return
	}
	if expected := os.Getenv("PAIR_TEST_NOTIFY_SOCKET_DIR"); expected == "" || os.Getenv("PAIR_NOTIFY_SOCKET_DIR") != expected {
		os.Exit(24)
	}
	if err := notifytransport.Send(os.Getenv("PAIR_PAIR_WRAP_PID_PATH"), "first child instruction"); err != nil {
		os.Exit(23)
	}
	os.Exit(0)
}
func TestNotificationBrokerBeforeExecAndCleanup(t *testing.T) {
	isolateNotificationSockets(t)
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	t.Setenv("PAIR_TAG", "notify-startup")
	t.Setenv("PAIR_SCOPE_KEY", "")
	t.Setenv("PAIR_WRAP_EVENTS", "0")
	var paths proxy
	paths.resolvePaths()
	t.Setenv("PAIR_TEST_EARLY_NOTIFY", "1")
	t.Setenv("PAIR_TEST_NOTIFY_SOCKET_DIR", os.Getenv("PAIR_NOTIFY_SOCKET_DIR"))
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", paths.capturePIDPath)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run([]string{executable, "-test.run=^TestNotificationHookChild$"}, strings.NewReader(""), &out, &stderr)
	if code != 0 {
		t.Fatalf("child code%d: %s", code, stderr.String())
	}
	if !bytes.Contains(out.Bytes(), notifyosc.Encode("first child instruction")) {
		t.Fatalf("startup hook lost: %q", out.Bytes())
	}
	if _, err := os.Stat(paths.capturePIDPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("binding retained: %v", err)
	}
	out.Reset()
	stderr.Reset()
	if code := Run([]string{filepath.Join(t.TempDir(), "missing-program")}, strings.NewReader(""), &out, &stderr); code == 0 {
		t.Fatal("missing child succeeded")
	}
	if _, err := os.Stat(paths.capturePIDPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("exec-failure binding retained: %v", err)
	}
}

func TestNotificationOutputFailureStopsRealPTYAndJoinsReader(t *testing.T) {
	for _, script := range []string{"printf trigger; exec sleep 30", "while :; do printf trigger; done"} {
		t.Run(script, func(t *testing.T) {
			cmd := exec.Command("/bin/sh", "-c", script)
			master, err := pty.Start(cmd)
			if err != nil {
				t.Fatal(err)
			}
			writer := &prefixWriter{limit: 3, fail: io.ErrClosedPipe}
			p := &proxy{ptmx: master, cmd: cmd, stdout: writer, stdoutFlushEvery: time.Millisecond}
			done := make(chan struct{})
			go func() { p.masterPump(); close(done) }()
			t.Cleanup(func() {
				_ = cmd.Process.Kill()
				_ = master.Close()
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					t.Error("master reader failed to join after cleanup")
				}
				_ = cmd.Wait()
			})
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("output failure did not stop master/reader")
			}
			if !errors.Is(p.stdoutPump.failure, io.ErrClosedPipe) || writer.Len() != 3 {
				t.Fatalf("failed-prefix accounting: err=%v n=%d", p.stdoutPump.failure, writer.Len())
			}
			if err := cmd.Wait(); err == nil {
				t.Fatal("failed output did not terminate live child")
			}
		})
	}
}

// Keep all broker sockets and arbitration locks away from the operator namespace.
// macOS t.TempDir paths exceed Unix socket limits, so own a short /tmp directory.
func isolateNotificationSockets(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "pair255-notify-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("remove owned notification fixture: %v", err)
		}
	})
	t.Setenv("PAIR_NOTIFY_SOCKET_DIR", dir)
	return dir
}
