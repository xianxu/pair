package couchmessage

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func transportSocket(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "pair-msg-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	return filepath.Join(root, "rpc.sock")
}

func TestTransportRoundTripAndStrictDecode(t *testing.T) {
	type payload struct {
		Text string `json:"text"`
	}
	socket := transportSocket(t)
	var calls atomic.Int32
	server, err := StartServer(context.Background(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var request payload
		if err := strictjson.Decode(raw, &request); err != nil {
			return nil, err
		}
		calls.Add(1)
		return raw, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var got payload
	if err := Call(context.Background(), socket, payload{"hello"}, &got); err != nil {
		t.Fatal(err)
	}
	if got.Text != "hello" || calls.Load() != 1 {
		t.Fatalf("got %#v calls %d", got, calls.Load())
	}
	if err := Call(context.Background(), socket, map[string]string{"unknown": "x"}, &got); err == nil {
		t.Fatal("unknown request accepted")
	}
	var wrong struct {
		Other string `json:"other"`
	}
	if err := Call(context.Background(), socket, payload{"hello"}, &wrong); err == nil {
		t.Fatal("unknown response field accepted")
	}
	if calls.Load() != 2 {
		t.Fatal("transport retried a request")
	}
}

func TestTransportRejectsBadFrames(t *testing.T) {
	for _, tc := range []struct {
		name string
		size uint32
		body string
	}{
		{"oversize", 32769, ""}, {"zero", 0, ""}, {"truncated", 10, "{}"},
		{"invalid-json", 1, "x"}, {"duplicate", 13, `{"x":1,"x":2}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			socket := transportSocket(t)
			var calls atomic.Int32
			server, err := StartServer(context.Background(), socket, func(context.Context, []byte) ([]byte, error) { calls.Add(1); return []byte(`{}`), nil })
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socket, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(time.Second))
			var header [4]byte
			binary.BigEndian.PutUint32(header[:], tc.size)
			conn.Write(append(header[:], tc.body...))
			conn.CloseWrite()
			var b [1]byte
			if _, err = conn.Read(b[:]); err == nil {
				t.Fatal("bad frame received response")
			}
			if calls.Load() != 0 {
				t.Fatal("bad frame reached handler")
			}
		})
	}
}

func TestTransportCancellationJoinsHandlersAndClosesReaders(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	socket := transportSocket(t)
	entered, exited := make(chan struct{}), make(chan struct{})
	server, err := StartServer(ctx, socket, func(ctx context.Context, _ []byte) ([]byte, error) {
		close(entered)
		<-ctx.Done()
		close(exited)
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	idle, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer idle.Close()
	result := make(chan error, 1)
	go func() { var response any; result <- Call(context.Background(), socket, struct{}{}, &response) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler not started")
	}
	cancel()
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	default:
		t.Fatal("Close did not join handler")
	}
	if err := <-result; err == nil {
		t.Fatal("cancelled request succeeded")
	}
	idle.SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	if _, err := idle.Read(b[:]); !errors.Is(err, io.EOF) {
		t.Fatalf("idle connection not closed: %v", err)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("socket retained: %v", err)
	}
}

func TestTransportClientDeadlineAndCancellation(t *testing.T) {
	socket := transportSocket(t)
	server, err := StartServer(context.Background(), socket, func(ctx context.Context, _ []byte) ([]byte, error) { <-ctx.Done(); return nil, ctx.Err() })
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	var response any
	if err := Call(ctx, socket, struct{}{}, &response); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline: %v", err)
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2()
	if err := Call(ctx2, socket, struct{}{}, &response); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}

func TestTransportSocketOwnership(t *testing.T) {
	socket := transportSocket(t)
	handler := func(context.Context, []byte) ([]byte, error) { return []byte(`{}`), nil }
	server, err := StartServer(context.Background(), socket, handler)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if second, err := StartServer(context.Background(), socket, handler); err == nil {
		second.Close()
		t.Fatal("replaced live socket")
	}
	if err := os.Remove(socket); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(socket, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(socket); err != nil || string(body) != "replacement" {
		t.Fatalf("removed replacement: %q %v", body, err)
	}
}

func TestTransportRefusesUnsafeDirectory(t *testing.T) {
	socket := transportSocket(t)
	root := filepath.Dir(socket)
	for _, mode := range []os.FileMode{0755, 0777} {
		if err := os.Chmod(root, mode); err != nil {
			t.Fatal(err)
		}
		if s, err := StartServer(context.Background(), socket, func(context.Context, []byte) ([]byte, error) { return nil, nil }); err == nil {
			s.Close()
			t.Fatalf("accepted mode %o", mode)
		}
	}
	os.Chmod(root, 0700)
	link := filepath.Join(root, "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if s, err := StartServer(context.Background(), filepath.Join(link, "other.sock"), func(context.Context, []byte) ([]byte, error) { return nil, nil }); err == nil {
		s.Close()
		t.Fatal("accepted symlink directory")
	}
}

func TestTransportSocketPathIsPureBoundedAndScoped(t *testing.T) {
	a, err := SocketPath("/repo/namespace", "supervisor")
	if err != nil {
		t.Fatal(err)
	}
	b, err := SocketPath("/repo/other", "supervisor")
	if err != nil {
		t.Fatal(err)
	}
	c, err := SocketPath("/repo/namespace", "wrapper:123")
	if err != nil {
		t.Fatal(err)
	}
	long, err := SocketPath("/"+strings.Repeat("long/", 1000)+"namespace", "supervisor")
	if err != nil {
		t.Fatal(err)
	}
	if a == b || a == c || len(long) > 100 || !filepath.IsAbs(a) {
		t.Fatalf("bad paths %q %q %q %q", a, b, c, long)
	}
	if _, err := SocketPath("relative", "supervisor"); err == nil {
		t.Fatal("relative namespace accepted")
	}
	if _, err := SocketPath("/repo", ""); err == nil {
		t.Fatal("empty endpoint accepted")
	}
}

func TestTransportHandlerCapacity(t *testing.T) {
	socket := transportSocket(t)
	entered := make(chan struct{}, MaxTransportHandlers+1)
	server, err := StartServer(context.Background(), socket, func(ctx context.Context, _ []byte) ([]byte, error) {
		entered <- struct{}{}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	for i := 0; i < MaxTransportHandlers; i++ {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if _, err := conn.Write([]byte{0, 0, 0, 2, '{', '}'}); err != nil {
			t.Fatal(err)
		}
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatalf("handler %d did not start", i)
		}
	}
	extra, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer extra.Close()
	extra.SetDeadline(time.Now().Add(time.Second))
	extra.Write([]byte{0, 0, 0, 2, '{', '}'})
	var b [1]byte
	if _, err := extra.Read(b[:]); err == nil {
		t.Fatal("saturated server accepted extra request")
	}
	select {
	case <-entered:
		t.Fatal("handler cap exceeded")
	default:
	}
}

func TestTransportClientCancelsInFlight(t *testing.T) {
	socket := transportSocket(t)
	entered := make(chan struct{})
	server, err := StartServer(context.Background(), socket, func(ctx context.Context, _ []byte) ([]byte, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > TransportTimeout {
			t.Error("handler lacks bounded deadline")
		}
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { var response any; result <- Call(ctx, socket, struct{}{}, &response) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler not started")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("call ignored cancellation")
	}
}

func TestTransportRejectsOversizePayloads(t *testing.T) {
	socket := transportSocket(t)
	var calls atomic.Int32
	server, err := StartServer(context.Background(), socket, func(context.Context, []byte) ([]byte, error) {
		calls.Add(1)
		return []byte(`"` + strings.Repeat("x", MaxFrameBytes) + `"`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var response any
	if err := Call(context.Background(), socket, strings.Repeat("x", MaxFrameBytes), &response); err == nil {
		t.Fatal("oversize request accepted")
	}
	if calls.Load() != 0 {
		t.Fatal("oversize request reached handler")
	}
	if err := Call(context.Background(), socket, struct{}{}, &response); err == nil {
		t.Fatal("oversize reply accepted")
	}
	if calls.Load() != 1 {
		t.Fatal("unexpected retry")
	}
}
