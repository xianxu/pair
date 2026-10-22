//go:build unix

package testcouchbridge

import (
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func pipeIOFixture(t *testing.T) (*os.File, *os.File, *os.File, *os.File) {
	t.Helper()
	input, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	reader, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = input.Close(); _ = writer.Close(); _ = reader.Close(); _ = output.Close() })
	return input, writer, reader, output
}

func TestPipeIOStopCauses(t *testing.T) {
	for _, cause := range []EventKind{EOF, TERM, INT} {
		t.Run(map[EventKind]string{EOF: "EOF", TERM: "TERM", INT: "INT"}[cause], func(t *testing.T) {
			input, writer, _, output := pipeIOFixture(t)
			signals := make(chan os.Signal, 1)
			bridge, err := NewPipeIO(input, output, signals)
			if err != nil {
				t.Fatal(err)
			}
			defer bridge.Close()
			switch cause {
			case EOF:
				_ = writer.Close()
			case TERM:
				signals <- syscall.SIGTERM
			case INT:
				signals <- syscall.SIGINT
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			event, err := bridge.WaitStop(ctx)
			if err != nil || event.Kind != cause {
				t.Fatalf("stop=%+v err=%v", event, err)
			}
		})
	}
}

func TestPipeIORefusesInputAndSecondReadiness(t *testing.T) {
	input, writer, reader, output := pipeIOFixture(t)
	bridge, err := NewPipeIO(input, output, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	if _, err := writer.Write([]byte("not a command")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if _, err := bridge.WaitStop(ctx); err == nil {
		t.Fatal("accepted nonempty stdin")
	}
	line := []byte("{\"Version\":1}\n")
	if err := bridge.Publish(ctx, line); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(line))
	if _, err := io.ReadFull(reader, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(line) {
		t.Fatalf("readiness bytes=%q", got)
	}
	if err := bridge.Publish(ctx, line); err == nil {
		t.Fatal("published a second readiness line")
	}
}

func fillPipe(t *testing.T, file *os.File) {
	t.Helper()
	conn, err := file.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var operation error
	if err := conn.Control(func(fd uintptr) {
		flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
		if err != nil {
			operation = err
			return
		}
		if err = unix.SetNonblock(int(fd), true); err != nil {
			operation = err
			return
		}
		defer func() {
			_, restore := unix.FcntlInt(fd, unix.F_SETFL, flags)
			operation = errors.Join(operation, restore)
		}()
		for {
			_, err = unix.Write(int(fd), make([]byte, 4096))
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				break
			}
			if err != nil {
				operation = err
				break
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	if operation != nil {
		t.Fatal(operation)
	}
}

func TestPipeIOBlockedReadinessCancelsAndJoins(t *testing.T) {
	input, _, _, output := pipeIOFixture(t)
	fillPipe(t, output)
	bridge, err := NewPipeIO(input, output, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	err = bridge.Publish(ctx, []byte("{}\n"))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked write did not cancel: %v", err)
	}
	if err := bridge.Close(); err != nil {
		t.Fatal(err)
	}
	if err := bridge.Close(); err != nil {
		t.Fatalf("repeat close: %v", err)
	}
	if _, err := input.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("input not closed: %v", err)
	}
	if _, err := output.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("output not closed: %v", err)
	}
}

func TestPipeIOCloseJoinsWaitingInput(t *testing.T) {
	input, _, _, output := pipeIOFixture(t)
	bridge, err := NewPipeIO(input, output, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := bridge.WaitStop(context.Background()); result <- err }()
	// Observe admission, not a sleep: Close must join an already-owned call.
	deadline := time.Now().Add(time.Second)
	for {
		bridge.mu.Lock()
		admitted := bridge.waiting
		bridge.mu.Unlock()
		if admitted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("input waiter never admitted")
		}
		runtime.Gosched()
	}
	if err := bridge.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("closed input became clean EOF")
		}
	case <-time.After(time.Second):
		t.Fatal("Close abandoned waiter")
	}
}

func TestPipeIORefusesUnboundedOrMultilineReadiness(t *testing.T) {
	for _, line := range [][]byte{nil, []byte("{}"), []byte("{}\n{}\n"), []byte(strings.Repeat("x", ReadinessLimit) + "\n")} {
		input, _, _, output := pipeIOFixture(t)
		bridge, err := NewPipeIO(input, output, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := bridge.Publish(t.Context(), line); err == nil {
			t.Fatal("accepted invalid readiness line")
		}
		if err := bridge.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPipeIORefusesRegularFilesAndConsumesOwnership(t *testing.T) {
	input, _, _, output := pipeIOFixture(t)
	_ = output.Close()
	regular, err := os.CreateTemp(t.TempDir(), "output-")
	if err != nil {
		t.Fatal(err)
	}
	bridge, err := NewPipeIO(input, regular, nil)
	if err == nil || bridge != nil {
		t.Fatalf("accepted uncancellable file output: %v %v", bridge, err)
	}
	if _, err := regular.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("failed constructor retained output: %v", err)
	}
	if _, err := input.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("failed constructor retained input: %v", err)
	}
}

func TestPipeIOCloseJoinsBlockedPublication(t *testing.T) {
	input, _, _, output := pipeIOFixture(t)
	fillPipe(t, output)
	bridge, err := NewPipeIO(input, output, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- bridge.Publish(context.Background(), []byte("{}\n")) }()
	deadline := time.Now().Add(time.Second)
	for {
		bridge.mu.Lock()
		admitted := bridge.published
		bridge.mu.Unlock()
		if admitted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("publisher never admitted")
		}
		runtime.Gosched()
	}
	if err := bridge.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, os.ErrClosed) {
			t.Fatalf("blocked publication close=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close abandoned blocked publisher")
	}
}
