package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	bridge "github.com/xianxu/pair/cmd/internal/testcouchbridge"
)

type inertOwner struct{}

func (inertOwner) Prepare(context.Context) error { return nil }
func (inertOwner) Start(context.Context) error   { return nil }
func (inertOwner) Readiness(context.Context) (bridge.Readiness, error) {
	return bridge.Readiness{}, nil
}

type loggingFailureOwner struct {
	inertOwner
	diagnostics      io.Writer
	closed, asserted bool
}

func (o *loggingFailureOwner) Prepare(context.Context) error {
	_, _ = io.WriteString(o.diagnostics, "fixture failed before readiness\n")
	return errors.New("fixture preparation failed")
}
func (o *loggingFailureOwner) Close(context.Context) error { o.closed = true; return nil }
func (o *loggingFailureOwner) AssertCleaned() error        { o.asserted = true; return nil }

func TestCLIFullStderrPipeCannotPreventJoinedOwnerCleanup(t *testing.T) {
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer errReader.Close()
	defer errWriter.Close()
	sink, err := bridge.NewDiagnosticPipe(errWriter)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := errWriter.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	fd := -1
	if err = conn.Control(func(raw uintptr) { fd = int(raw) }); err != nil {
		t.Fatal(err)
	}
	filled := false
	for i := 0; i < 1024; i++ {
		_, writeErr := unix.Write(fd, make([]byte, 4096))
		if errors.Is(writeErr, unix.EAGAIN) {
			filled = true
			break
		}
		if writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if !filled {
		t.Fatal("bounded fixture could not fill stderr")
	}
	input, control, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	reader, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	owner := &loggingFailureOwner{}
	deps := dependencies{newOwner: func(_ bridge.Options, d io.Writer) bridge.Owner { owner.diagnostics = d; return owner }}
	done := make(chan int, 1)
	go func() { done <- run([]string{"--scratch-parent", "/scratch"}, input, output, sink, nil, deps) }()
	select {
	case code := <-done:
		if code == 0 || !owner.closed || !owner.asserted {
			t.Fatalf("code=%d closed=%v asserted=%v", code, owner.closed, owner.asserted)
		}
	case <-time.After(3 * time.Second):
		// Release a regressed blocking writer before failing, so the fixture
		// itself never leaves an owned stage behind.
		_ = errReader.Close()
		<-done
		t.Fatal("full stderr blocked cleanup")
	}
}
func (inertOwner) Done() <-chan struct{}       { return nil }
func (inertOwner) Close(context.Context) error { return nil }
func (inertOwner) AssertCleaned() error        { return nil }
func (inertOwner) RetainedRoot() string        { return "/private/retained" }

type inertPipes struct{}

func (inertPipes) WaitStop(context.Context) (bridge.Event, error) { return bridge.Event{}, nil }
func (inertPipes) Publish(context.Context, []byte) error          { return nil }
func (inertPipes) Close() error                                   { return nil }

func TestCLIRejectsArgumentsBeforeEffects(t *testing.T) {
	for _, args := range [][]string{nil, {"--scratch-parent", "relative"}, {"--scratch-parent", "/scratch", "extra"}, {"--scratch-parent", "/scratch", "--couch", "relative"}, {"--unknown"}} {
		var diagnostics bytes.Buffer
		out, err := os.CreateTemp(t.TempDir(), "stdout")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = out.Close() })
		called := false
		deps := dependencies{newPipes: func(*os.File, *os.File, <-chan os.Signal) (bridge.BridgeIO, error) {
			called = true
			return inertPipes{}, nil
		}}
		code := run(args, nil, out, &diagnostics, nil, deps)
		if code == 0 || called || diagnostics.Len() == 0 {
			t.Fatalf("args=%v code=%d called=%v diag=%q", args, code, called, diagnostics.String())
		}
		info, err := out.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() != 0 {
			t.Fatal("argument failure contaminated stdout")
		}
	}
}

func TestCLIErrorOutputIsBounded(t *testing.T) {
	var diagnostics bytes.Buffer
	deps := dependencies{
		newPipes: func(*os.File, *os.File, <-chan os.Signal) (bridge.BridgeIO, error) { return inertPipes{}, nil },
		newOwner: func(bridge.Options, io.Writer) bridge.Owner { return inertOwner{} },
		run: func(context.Context, bridge.Owner, bridge.BridgeIO) error {
			return errors.New(string(bytes.Repeat([]byte{'x'}, bridge.DiagnosticLimit*2)))
		},
	}
	if code := run([]string{"--scratch-parent", "/scratch"}, nil, nil, &diagnostics, nil, deps); code == 0 {
		t.Fatal("error exited clean")
	}
	if diagnostics.Len() > bridge.DiagnosticLimit+1024 || !bytes.Contains(diagnostics.Bytes(), []byte("diagnostics truncated")) || !bytes.Contains(diagnostics.Bytes(), []byte("/private/retained")) {
		t.Fatalf("error output lacks bound or evidence location: %d", diagnostics.Len())
	}
}

func TestCLIUsesNativeOwnerAndReturnsRunnerResult(t *testing.T) {
	for _, fail := range []bool{false, true} {
		var diagnostics bytes.Buffer
		var received bridge.Options
		var stderr io.Writer
		ran := false
		deps := dependencies{
			newPipes: func(*os.File, *os.File, <-chan os.Signal) (bridge.BridgeIO, error) { return inertPipes{}, nil },
			newOwner: func(o bridge.Options, d io.Writer) bridge.Owner { received = o; stderr = d; return inertOwner{} },
			run: func(_ context.Context, o bridge.Owner, p bridge.BridgeIO) error {
				ran = true
				_, _ = io.WriteString(stderr, "fixture diagnostic\n")
				if fail {
					return errors.New("runner failed")
				}
				return nil
			},
		}
		code := run([]string{"--scratch-parent", "/scratch", "--couch", "/candidate/couch"}, nil, nil, &diagnostics, nil, deps)
		if !ran || received.Couch != "/candidate/couch" || received.ScratchParent != "/scratch" || (code != 0) != fail {
			t.Fatalf("code=%d ran=%v options=%+v", code, ran, received)
		}
		if fail && !bytes.Contains(diagnostics.Bytes(), []byte("/private/retained")) {
			t.Fatal("failure lost retained root")
		}
	}
}

func TestCLIPipeFailureNeverStartsOwner(t *testing.T) {
	var diagnostics bytes.Buffer
	deps := dependencies{newPipes: func(*os.File, *os.File, <-chan os.Signal) (bridge.BridgeIO, error) {
		return nil, errors.New("not pipes")
	}, newOwner: func(bridge.Options, io.Writer) bridge.Owner { t.Fatal("owner created after pipe refusal"); return nil }}
	if code := run([]string{"--scratch-parent", "/scratch"}, nil, nil, &diagnostics, nil, deps); code == 0 {
		t.Fatal("pipe error exited clean")
	}
}
