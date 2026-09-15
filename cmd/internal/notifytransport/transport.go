package notifytransport

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
)

const QueueSize = 32
const SendTimeout = 200 * time.Millisecond

type Broker struct {
	conn                    *net.UnixConn
	binding, socket         string
	bindingInfo, socketInfo os.FileInfo
	messages                chan string
	diagnostics             chan error
	done                    chan struct{}
	once                    sync.Once
	closeErr                error
}

// Start publishes the exact existing wrapper PID binding only after listening.
// The process owning this broker must also own the supplied PID.
func Start(binding string, pid int) (*Broker, error) {
	if pid <= 0 || pid != os.Getpid() {
		return nil, errors.New("notification broker requires its own positive process PID")
	}
	canonical, err := canonicalBinding(binding)
	if err != nil {
		return nil, err
	}
	unlock, err := lockDirectory()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err = clearDeadBinding(canonical); err != nil {
		return nil, err
	}
	socket, err := address(canonical, pid)
	if err != nil {
		return nil, err
	}
	// ListenUnixgram refuses occupied paths; it never unlinks another listener.
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(socket)
	if err != nil {
		conn.Close()
		return nil, err
	}
	b := &Broker{conn: conn, binding: canonical, socket: socket, socketInfo: info, messages: make(chan string, QueueSize), diagnostics: make(chan error, 1), done: make(chan struct{})}
	fail := func(err error) (*Broker, error) { conn.Close(); _ = removeOwned(socket, info); return nil, err }
	if err = conn.SetReadBuffer(4 * (notifyosc.MaxMessageBytes + 1)); err != nil {
		return fail(err)
	}
	// Exclusive creation prevents competing wrappers from overwriting a winner.
	f, err := os.OpenFile(canonical, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fail(err)
	}
	b.bindingInfo, err = f.Stat()
	if err != nil {
		f.Close()
		return fail(err)
	}
	// Preserve the legacy binding's exact decimal format (capture/reexec
	// consumers parse it directly, without trimming a trailing newline).
	_, writeErr := io.WriteString(f, strconv.Itoa(pid))
	err = errors.Join(writeErr, f.Close())
	if err != nil {
		_ = removeOwned(canonical, b.bindingInfo)
		return fail(err)
	}
	go b.receive()
	return b, nil
}

func (b *Broker) Messages() <-chan string { return b.messages }

// Diagnostics coalesces transport failures when the wrapper is busy. Consumers
// log these errors; no unbounded log queue or per-message goroutine is created.
func (b *Broker) Diagnostics() <-chan error { return b.diagnostics }
func (b *Broker) diagnose(err error) {
	select {
	case b.diagnostics <- err:
	default:
	}
}
func (b *Broker) receive() {
	defer close(b.done)
	defer close(b.messages)
	defer close(b.diagnostics)
	buffer := make([]byte, notifyosc.MaxMessageBytes+1)
	for {
		n, _, err := b.conn.ReadFromUnix(buffer)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				b.diagnose(err)
			}
			return
		}
		if n > notifyosc.MaxMessageBytes {
			b.diagnose(errors.New("notification datagram exceeds 4096 bytes"))
			continue
		}
		message := string(buffer[:n])
		if message == "" || notifyosc.Sanitize(buffer[:n]) != message {
			b.diagnose(errors.New("invalid notification datagram"))
			continue
		}
		select {
		case b.messages <- message:
		default:
			b.diagnose(errors.New("notification broker queue full"))
		}
	}
}

// Close interrupts and joins the sole reader before removing owned resources.
func (b *Broker) Close() error {
	b.once.Do(func() {
		err := b.conn.Close()
		<-b.done
		unlock, lockErr := lockDirectory()
		if lockErr != nil {
			b.closeErr = errors.Join(err, lockErr)
			return
		}
		defer unlock()
		b.closeErr = errors.Join(err, removeOwned(b.binding, b.bindingInfo), removeOwned(b.socket, b.socketInfo))
	})
	return b.closeErr
}

// Send rereads the binding each time: a surviving hook follows its current
// wrapper without caching a stale attachment or falling back to terminal IO.
func Send(binding, message string) error {
	deadline := time.Now().Add(SendTimeout)
	canonical, err := canonicalBinding(binding)
	if err != nil {
		return err
	}
	pid, _, err := readPID(canonical)
	if err != nil {
		return err
	}
	if !alive(pid) {
		return errors.New("notification wrapper is no longer live")
	}
	socket, err := address(canonical, pid)
	if err != nil {
		return err
	}
	info, err := os.Lstat(socket)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 || !owned(info) {
		return errors.New("unsafe notification socket")
	}
	clean := notifyosc.Sanitize([]byte(message))
	if clean == "" {
		return errors.New("notification message is empty after sanitizing")
	}
	conn, err := (&net.Dialer{Deadline: deadline}).Dial("unixgram", socket)
	if err != nil {
		return err
	}
	defer conn.Close()
	// macOS defaults AF_UNIX datagrams to a 2048-byte send buffer. Reserve a
	// bounded buffer large enough for the complete 4096-byte protocol message.
	if err = conn.(*net.UnixConn).SetWriteBuffer(2 * (notifyosc.MaxMessageBytes + 1)); err != nil {
		return err
	}
	if err = conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	n, err := conn.Write([]byte(clean))
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	if n != len(clean) {
		return io.ErrShortWrite
	}
	return nil
}
