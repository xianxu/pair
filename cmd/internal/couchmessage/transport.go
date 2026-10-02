package couchmessage

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const (
	// A discovery response contains at most MaxActors encoded bindings, each
	// with at most 512 bytes of candidate metadata, plus a 1024-byte envelope.
	// This also covers commit (3 bindings + 6*MaxBodyBytes + IDs/metadata)
	// and receipt (2 bindings + 6*(MaxBodyBytes+MaxReceiptDetailBytes) + metadata).
	MaxFrameBytes = MaxActors*(MaxBindingBytes+512) + 1024
	// Reserved framing code: response encoding failed before any payload write.
	transportResponseTooLarge = ^uint32(0)
	MaxTransportHandlers      = 128
	TransportTimeout          = 2 * time.Second
)

// SocketPath only derives an address; it does not create runtime files. The
// namespace must already be the caller's canonical absolute store path.
func SocketPath(namespace, endpoint string) (string, error) {
	if !filepath.IsAbs(namespace) || namespace != filepath.Clean(namespace) || endpoint == "" {
		return "", errors.New("message socket requires a canonical absolute namespace and endpoint")
	}
	digest := sha256.Sum256([]byte(namespace + "\x00" + endpoint))
	return filepath.Join("/tmp", fmt.Sprintf("pair-message-%d", os.Getuid()), fmt.Sprintf("%x.sock", digest[:20])), nil
}

// Server owns its listener, accepted connections, and handler goroutines.
// Handlers must honor their context and return within its deadline. Protocol
// refusals belong in response JSON; handler errors terminate the connection.
type Server struct {
	listener    *net.UnixListener
	socket      string
	identity    os.FileInfo
	ctx         context.Context
	cancel      context.CancelFunc
	handler     func(context.Context, []byte) ([]byte, error)
	mu          sync.Mutex
	connections map[net.Conn]struct{}
	handlers    sync.WaitGroup
	done        chan struct{}
	closeErr    error
}

// StartServer collects wrapper sockets with proven-dead owners. Existing broker
// sockets remain under the supervisor lease owner's cleanup authority.
func StartServer(ctx context.Context, socket string, handler func(context.Context, []byte) ([]byte, error)) (*Server, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, errors.New("message handler is required")
	}
	listener, info, err := transportListen(socket)
	if err != nil {
		return nil, err
	}
	serverCtx, cancel := context.WithCancel(ctx)
	s := &Server{listener: listener, socket: socket, identity: info, ctx: serverCtx, cancel: cancel,
		handler: handler, connections: make(map[net.Conn]struct{}), done: make(chan struct{})}
	go s.run()
	return s, nil
}

// transportListen binds a private, owner-only socket in the private message
// directory. The caller removes it with transportRemoveOwned(socket, info).
func transportListen(socket string) (*net.UnixListener, os.FileInfo, error) {
	if !filepath.IsAbs(socket) || socket != filepath.Clean(socket) || len(socket) > 103 {
		return nil, nil, errors.New("message socket must be a clean absolute path of at most 103 bytes")
	}
	if err := transportPrivateDirectory(filepath.Dir(socket)); err != nil {
		return nil, nil, err
	}
	if err := collectDeadEndpointSockets(filepath.Dir(socket), func(pid int) error { return syscall.Kill(pid, 0) }); err != nil {
		return nil, nil, err
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		return nil, nil, err
	}
	// Go's default unlink-on-close would remove a replacement generation.
	listener.SetUnlinkOnClose(false)
	info, err := os.Lstat(socket)
	if err != nil {
		listener.Close()
		return nil, nil, err
	}
	if err := os.Chmod(socket, 0600); err != nil {
		listener.Close()
		_ = transportRemoveOwned(socket, info)
		return nil, nil, err
	}
	return listener, info, nil
}

func transportPrivateDirectory(path string) error {
	if err := os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || info.Mode().Perm() != 0700 || !ok || stat.Uid != uint32(os.Getuid()) {
		return fmt.Errorf("unsafe message socket directory %q", path)
	}
	return nil
}

func transportRemoveOwned(path string, expected os.FileInfo) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !os.SameFile(info, expected) {
		return nil
	}
	return os.Remove(path)
}

func (s *Server) run() {
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		<-s.ctx.Done()
		s.listener.Close()
		s.mu.Lock()
		for conn := range s.connections {
			conn.Close()
		}
		s.mu.Unlock()
	}()
	for {
		conn, err := s.listener.AcceptUnix()
		if err != nil {
			break
		}
		s.mu.Lock()
		if s.ctx.Err() != nil || len(s.connections) >= MaxTransportHandlers {
			s.mu.Unlock()
			conn.Close()
			continue
		}
		s.connections[conn] = struct{}{}
		s.handlers.Add(1)
		s.mu.Unlock()
		go s.serve(conn)
	}
	s.cancel()
	<-watchDone
	s.handlers.Wait()
	s.closeErr = transportRemoveOwned(s.socket, s.identity)
	close(s.done)
}

func (s *Server) serve(conn net.Conn) {
	defer s.handlers.Done()
	defer func() { conn.Close(); s.mu.Lock(); delete(s.connections, conn); s.mu.Unlock() }()
	ctx, cancel := context.WithTimeout(s.ctx, TransportTimeout)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return
	}
	raw, err := transportReadFrame(conn)
	if err != nil {
		return
	}
	var value any
	if strictjson.Decode(raw, &value) != nil {
		return
	}
	response, err := s.handler(ctx, raw)
	if err != nil || ctx.Err() != nil {
		return
	}
	if strictjson.Decode(response, &value) != nil {
		return
	}
	if len(response) > MaxFrameBytes {
		// No response bytes have been sent. Report overflow explicitly; the
		// handler may already have admitted a send, so its outcome is uncertain.
		var header [4]byte
		binary.BigEndian.PutUint32(header[:], transportResponseTooLarge)
		_, _ = conn.Write(header[:])
		return
	}
	if err := transportWriteFrame(conn, response); err != nil {
		return // Partial writes cannot be replaced by another frame.
	}
}

// Close is idempotent. It cancels handlers, closes blocked socket reads, joins
// every server goroutine, and removes only this listener's socket inode.
func (s *Server) Close() error {
	s.cancel()
	<-s.done
	return s.closeErr
}

// Call performs exactly one exchange. A failed call can have been admitted by
// the remote handler; callers must query their request ID rather than retry it.
func Call(ctx context.Context, socket string, request any, response any) error {
	ctx, cancel := context.WithTimeout(ctx, TransportTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if len(raw) > MaxFrameBytes {
		return errors.New("message request exceeds frame limit")
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
	if err != nil {
		return transportContextError(ctx, err)
	}
	defer conn.Close()
	// Unlike socket deadlines, cancellation also interrupts a context without a
	// deadline. Join the callback before returning so Call owns all its work.
	closed := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { conn.Close(); close(closed) })
	defer func() {
		if !stop() {
			<-closed
		}
	}()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	if err := transportWriteFrame(conn, raw); err != nil {
		return transportContextError(ctx, err)
	}
	reply, err := transportReadFrame(conn)
	if err != nil {
		return transportContextError(ctx, err)
	}
	return strictjson.Decode(reply, response)
}

func transportContextError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return err
}

func transportReadFrame(in io.Reader) ([]byte, error) {
	return transportReadFrameLimit(in, MaxFrameBytes)
}

func transportReadFrameLimit(in io.Reader, limit int) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(in, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == transportResponseTooLarge {
		return nil, errors.New("message response exceeds frame limit; request outcome uncertain")
	}
	if size == 0 || int64(size) > int64(limit) {
		return nil, errors.New("invalid message frame length")
	}
	body := make([]byte, int(size))
	_, err := io.ReadFull(in, body)
	return body, err
}

func transportWriteFrame(out io.Writer, body []byte) error {
	if len(body) == 0 || len(body) > MaxFrameBytes {
		return errors.New("invalid message frame length")
	}
	frame := make([]byte, 4+len(body))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(body)))
	copy(frame[4:], body)
	n, err := out.Write(frame)
	if err != nil {
		return err
	}
	if n != len(frame) {
		return io.ErrShortWrite
	}
	return nil
}
