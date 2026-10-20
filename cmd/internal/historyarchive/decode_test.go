package historyarchive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The subprocess fixture uses real OS pipes and records readiness over a local
// socket. Tests never infer readiness from elapsed time or poll the filesystem.
func TestDecoderProcess(t *testing.T) {
	root := os.Getenv("PAIR_DECODER_FIXTURE")
	if root == "" {
		return
	}
	args := os.Args[1:]
	stage := "zstd"
	for _, arg := range args {
		if arg == "--decrypt" {
			stage = "gpg"
		}
	}
	encoded, _ := json.Marshal(args)
	if err := os.WriteFile(filepath.Join(root, stage+".args"), encoded, 0600); err != nil {
		os.Exit(91)
	}
	if err := os.WriteFile(filepath.Join(root, stage+".home"), []byte(os.Getenv("GNUPGHOME")), 0600); err != nil {
		os.Exit(92)
	}
	conn, err := net.Dial("tcp", os.Getenv("PAIR_DECODER_READY"))
	if err != nil {
		os.Exit(93)
	}
	fmt.Fprintf(conn, "%s %d\n", stage, os.Getpid())
	conn.Close()
	if _, err = os.Stat(filepath.Join(root, stage+".fail")); err == nil {
		os.Stderr.WriteString(strings.Repeat("invalid-stream ", 2000))
		os.Exit(7)
	}
	if _, err = os.Stat(filepath.Join(root, stage+".block")); err == nil {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGUSR1)
		_, _ = io.Copy(io.Discard, os.Stdin)
		ready, err := net.Dial("tcp", os.Getenv("PAIR_DECODER_READY"))
		if err != nil {
			os.Exit(94)
		}
		fmt.Fprintf(ready, "%s %d\n", stage+"-blocked", os.Getpid())
		ready.Close()
		<-signals
		os.Exit(0)
	}
	if _, err = io.Copy(os.Stdout, os.Stdin); err != nil {
		os.Exit(8)
	}
	if _, err = os.Stat(filepath.Join(root, stage+".late")); err == nil {
		os.Stdout.Close()
		if address := os.Getenv("PAIR_DECODER_LATE_ACK"); address != "" {
			conn, err := net.Dial("tcp", address)
			if err != nil {
				os.Exit(95)
			}
			if _, err = io.CopyN(io.Discard, conn, 1); err != nil {
				os.Exit(96)
			}
			conn.Close()
		}
		os.Stderr.WriteString("late integrity failure")
		os.Exit(7)
	}
	os.Exit(0)
}

type decoderEvent struct {
	stage string
	pid   int
}

func fixtureDecoder(t *testing.T) (CommandDecoder, string, <-chan decoderEvent) {
	t.Helper()
	root := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan decoderEvent, 4)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			var event decoderEvent
			_, err = fmt.Fscanf(conn, "%s %d\n", &event.stage, &event.pid)
			conn.Close()
			if err == nil {
				events <- event
			}
		}
	}()
	t.Cleanup(func() { listener.Close(); <-done })
	t.Setenv("PAIR_DECODER_FIXTURE", root)
	t.Setenv("PAIR_DECODER_READY", listener.Addr().String())
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(root, "tool")
	// The executable wrapper execs the test binary, so recorded PIDs are the
	// children whose process handles the production decoder must join.
	shellQuote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	if err = os.WriteFile(tool, []byte("#!/bin/sh\nexec "+shellQuote(exe)+" -test.run=^TestDecoderProcess$ -- \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return CommandDecoder{GPG: tool, Zstd: tool, Home: filepath.Join(root, "home")}, root, events
}
func decoderReady(t *testing.T, ctx context.Context, events <-chan decoderEvent, n int) []decoderEvent {
	t.Helper()
	var seen []decoderEvent
	for len(seen) < n {
		select {
		case event := <-events:
			seen = append(seen, event)
		case <-ctx.Done():
			t.Fatal("decoder child readiness: ", ctx.Err())
		}
	}
	return seen
}
func decoderExited(t *testing.T, events []decoderEvent) {
	t.Helper()
	for _, event := range events {
		if err := syscall.Kill(event.pid, 0); err != syscall.ESRCH {
			t.Fatalf("%s child %d remains: %v", event.stage, event.pid, err)
		}
	}
}
func decoderResult(t *testing.T, ctx context.Context, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		t.Fatal("decoder failed to join: ", ctx.Err())
		return nil
	}
}

func TestDecoderStreamAndPinentryFlags(t *testing.T) {
	c, root, events := fixtureDecoder(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	input := strings.Repeat("tar bytes", 100000)
	var output decoderObservedWriter
	if err := c.Decode(ctx, strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if output.closed {
		t.Fatal("successful decoder closed output")
	}
	if output.String() != input {
		t.Fatal("stream differs")
	}
	for _, stage := range []string{"gpg", "zstd"} {
		raw, err := os.ReadFile(filepath.Join(root, stage+".args"))
		if err != nil {
			t.Fatal(err)
		}
		var args []string
		if err = json.Unmarshal(raw, &args); err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(args, " ")
		if stage == "gpg" {
			for _, flag := range []string{"--no-options", "--no-auto-key-retrieve", "--auto-key-locate clear", "--output -", "--decrypt"} {
				if !strings.Contains(joined, flag) {
					t.Fatalf("gpg missing %s: %v", flag, args)
				}
			}
			for _, flag := range []string{"--pinentry-mode", "--batch", "--no-tty"} {
				if strings.Contains(joined, flag) {
					t.Fatalf("normal pinentry inhibited by %s", flag)
				}
			}
			home, _ := os.ReadFile(filepath.Join(root, stage+".home"))
			if string(home) != c.Home {
				t.Fatalf("GNUPGHOME=%q", home)
			}
		} else if !strings.Contains(joined, "--decompress") || !strings.Contains(joined, "--stdout") {
			t.Fatalf("zstd flags %v", args)
		}
	}
	decoderExited(t, decoderReady(t, ctx, events, 2))
}
func TestDecoderFailuresAndLateIntegrity(t *testing.T) {
	for _, mode := range []string{"gpg", "zstd", "gpg-start", "zstd-start", "input", "output", "late"} {
		t.Run(mode, func(t *testing.T) {
			c, root, events := fixtureDecoder(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var input io.Reader = strings.NewReader("complete plaintext")
			var output io.Writer = &bytes.Buffer{}
			switch mode {
			case "gpg", "zstd":
				if err := os.WriteFile(filepath.Join(root, mode+".fail"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "gpg-start":
				c.GPG = filepath.Join(root, "missing")
			case "zstd-start":
				c.Zstd = filepath.Join(root, "missing")
			case "input":
				input = codecBadReader{}
			case "output":
				output = codecBadWriter{}
			case "late":
				observed := &decoderObservedWriter{expected: len("complete plaintext"), ready: make(chan struct{})}
				output = observed
				decoderLateAck(t, observed.ready)
				if err := os.WriteFile(filepath.Join(root, "gpg.late"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := c.Decode(ctx, input, output)
			if err == nil {
				t.Fatal("failure accepted")
			}
			if len(err.Error()) > 10000 {
				t.Fatalf("unbounded diagnostic %d", len(err.Error()))
			}
			if mode == "late" {
				if !strings.Contains(err.Error(), "late integrity failure") {
					t.Fatal(err)
				}
				if output.(*decoderObservedWriter).String() != "complete plaintext" {
					t.Fatalf("late failure not exercised: %q", output.(*decoderObservedWriter).String())
				}
			}
			var seen []decoderEvent
			for {
				select {
				case event := <-events:
					seen = append(seen, event)
				default:
					decoderExited(t, seen)
					return
				}
			}
		})
	}
}

type decoderBlockedInput struct {
	started chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func (r *decoderBlockedInput) Read([]byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.closed
	return 0, io.ErrClosedPipe
}
func (r *decoderBlockedInput) Close() error {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
	return nil
}
func TestDecoderCancelJoinsChildrenAndBlockedInput(t *testing.T) {
	c, _, events := fixtureDecoder(t)
	watchdog, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	ctx, cancel := context.WithCancel(watchdog)
	defer cancel()
	input := &decoderBlockedInput{started: make(chan struct{}), closed: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- c.Decode(ctx, input, io.Discard) }()
	children := decoderReady(t, watchdog, events, 2)
	select {
	case <-input.started:
	case <-watchdog.Done():
		t.Fatal("input copy not started")
	}
	cancel()
	err := decoderResult(t, watchdog, done)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	select {
	case <-input.closed:
	default:
		t.Fatal("blocked input not closed")
	}
	decoderExited(t, children)
}
func TestDecoderFailedChildClosesBlockedInput(t *testing.T) {
	c, root, events := fixtureDecoder(t)
	if err := os.WriteFile(filepath.Join(root, "zstd.fail"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	input := &decoderBlockedInput{started: make(chan struct{}), closed: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- c.Decode(ctx, input, io.Discard) }()
	err := decoderResult(t, ctx, done)
	if err == nil {
		t.Fatal("child failure accepted")
	}
	select {
	case <-input.closed:
	default:
		t.Fatal("input copy leaked")
	}
	var children []decoderEvent
	for {
		select {
		case event := <-events:
			children = append(children, event)
		default:
			decoderExited(t, children)
			return
		}
	}
}

// decoderLateAck releases a late GPG failure only after the consumer observes
// the configured output boundary. Its listener goroutine is always joined.
func decoderLateAck(t *testing.T, ready <-chan struct{}) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		select {
		case <-ready:
			_, _ = conn.Write([]byte{1})
		case <-stop:
		}
	}()
	t.Cleanup(func() { close(stop); listener.Close(); <-done })
	t.Setenv("PAIR_DECODER_LATE_ACK", listener.Addr().String())
}

type decoderObservedWriter struct {
	buffer   bytes.Buffer
	expected int
	ready    chan struct{}
	once     sync.Once
	closed   bool
}

func (w *decoderObservedWriter) Write(p []byte) (int, error) {
	n, err := w.buffer.Write(p)
	if w.ready != nil && w.buffer.Len() >= w.expected {
		w.once.Do(func() { close(w.ready) })
	}
	return n, err
}
func (w *decoderObservedWriter) String() string { return w.buffer.String() }
func (w *decoderObservedWriter) Close() error   { w.closed = true; return nil }

type decoderBlockedOutput struct {
	started, closed      chan struct{}
	startOnce, closeOnce sync.Once
}

func (w *decoderBlockedOutput) Write([]byte) (int, error) {
	w.startOnce.Do(func() { close(w.started) })
	<-w.closed
	return 0, io.ErrClosedPipe
}
func (w *decoderBlockedOutput) Close() error { w.closeOnce.Do(func() { close(w.closed) }); return nil }
func TestDecoderBlockedOutputClosesOnFailureOrCancellation(t *testing.T) {
	for _, mode := range []string{"cancel", "internal-failure"} {
		t.Run(mode, func(t *testing.T) {
			c, root, events := fixtureDecoder(t)
			watchdog, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			ctx, cancel := context.WithCancel(watchdog)
			defer cancel()
			output := &decoderBlockedOutput{started: make(chan struct{}), closed: make(chan struct{})}
			if mode == "internal-failure" {
				if err := os.WriteFile(filepath.Join(root, "gpg.late"), nil, 0600); err != nil {
					t.Fatal(err)
				}
				decoderLateAck(t, output.started)
			}
			done := make(chan error, 1)
			go func() { done <- c.Decode(ctx, strings.NewReader("plaintext"), output) }()
			children := decoderReady(t, watchdog, events, 2)
			select {
			case <-output.started:
			case <-watchdog.Done():
				t.Fatal("output never blocked")
			}
			if mode == "cancel" {
				cancel()
			}
			err := decoderResult(t, watchdog, done)
			if err == nil {
				t.Fatal("blocked-output failure accepted")
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if mode == "internal-failure" && !strings.Contains(err.Error(), "late integrity failure") {
				t.Fatal(err)
			}
			select {
			case <-output.closed:
			default:
				t.Fatal("blocked output was not closed")
			}
			decoderExited(t, children)
		})
	}
}
func TestDecoderCancelBlockedChild(t *testing.T) {
	c, root, events := fixtureDecoder(t)
	if err := os.WriteFile(filepath.Join(root, "zstd.block"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	watchdog, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	ctx, cancel := context.WithCancel(watchdog)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Decode(ctx, strings.NewReader("ciphertext"), io.Discard) }()
	children := decoderReady(t, watchdog, events, 3)
	cancel()
	err := decoderResult(t, watchdog, done)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	decoderExited(t, children)
}
