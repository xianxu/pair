package couchschedule

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type ExecutionRequest struct {
	Namespace, HubRoot, Task, RunID, Revision, CacheDir, PermitsDir, TestLockHelper, TestLockFile, GoBinary string
	Args                                                                                                    []string
	Spec                                                                                                    TaskSpec
	Deadline                                                                                                time.Time
}
type ExecutionResult struct {
	Exit            int
	Outcome         string
	Output          []byte
	OutputBytes     int64
	BuildKey, Error string
}
type OwnerIdentity struct {
	PID   int
	Start string
}

// ExecutionError identifies whether effects may have started before a result
// was lost. Such a run retains its recorded owner and must never be replayed.
type ExecutionError struct {
	Owner          OwnerIdentity
	MayHaveStarted bool
	Err            error
}

func (e *ExecutionError) Error() string { return e.Err.Error() }
func (e *ExecutionError) Unwrap() error { return e.Err }

type Executor struct {
	Program    string
	PrefixArgs []string
	OnOwner    func(OwnerIdentity) error
	OnOutput   func([]byte)
}

type TaskExecutor interface {
	Execute(context.Context, ExecutionRequest, func(OwnerIdentity) error) (ExecutionResult, error)
}

var _ TaskExecutor = Executor{}

func (e Executor) Execute(ctx context.Context, r ExecutionRequest, onOwner func(OwnerIdentity) error) (ExecutionResult, error) {
	e.OnOwner = onOwner
	return e.Run(ctx, r)
}

const ownerFlag = "--internal-schedule-owner"
const ownerMaxFrame = 1 << 20

type ownerFrame struct {
	Kind         string
	Owner        *OwnerIdentity    `json:",omitempty"`
	Request      *ExecutionRequest `json:",omitempty"`
	Result       *ExecutionResult  `json:",omitempty"`
	StreamOutput bool              `json:",omitempty"`
}

func (e Executor) Run(ctx context.Context, r ExecutionRequest) (ExecutionResult, error) {
	if err := validateExecution(r); err != nil {
		return ExecutionResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, err
	}
	if e.OnOwner == nil {
		return ExecutionResult{}, errors.New("execution requires durable owner acknowledgement")
	}
	program := e.Program
	var err error
	if program == "" {
		program, err = os.Executable()
		if err != nil {
			return ExecutionResult{}, err
		}
	}
	cmd := exec.Command(program, append(append([]string(nil), e.PrefixArgs...), ownerFlag)...)
	input, err := cmd.StdinPipe()
	if err != nil {
		return ExecutionResult{}, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return ExecutionResult{}, err
	}
	diagnostics := operationalrun.NewOutputBuffer()
	cmd.Stderr = diagnostics
	stream, err := newOwnerOutputReader(e.OnOutput)
	if err != nil {
		input.Close()
		return ExecutionResult{}, err
	}
	if stream != nil {
		cmd.ExtraFiles = []*os.File{stream.writer}
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		stream.abort()
		return ExecutionResult{}, err
	}
	stream.start()
	// Never CommandContext-kill the controller: EOF asks it to stop and join
	// its work groups before its process releases either admission lease.
	stop := context.AfterFunc(ctx, func() {
		if parentEndCancelsOwner(time.Now(), r.Deadline) {
			_ = input.Close()
		}
	})
	defer stop()
	defer input.Close()
	waited := false
	wait := func() error {
		if waited {
			return nil
		}
		waited = true
		return cmd.Wait()
	}
	var owner OwnerIdentity
	authorized := false
	fail := func(err error) (ExecutionResult, error) {
		input.Close()
		_ = wait()
		stream.finish()
		return ExecutionResult{}, &ExecutionError{Owner: owner, MayHaveStarted: authorized, Err: err}
	}
	if err = writeOwnerFrame(input, ownerFrame{Kind: "request", Request: &r, StreamOutput: stream != nil}); err != nil {
		return fail(err)
	}
	ready, err := readOwnerFrame(output)
	if err != nil {
		return fail(fmt.Errorf("controller readiness: %w: %s", err, diagnostics.Bytes()))
	}
	if ready.Kind != "ready" || ready.Owner == nil || ready.Owner.PID != cmd.Process.Pid || ready.Owner.Start == "" || procutil.StrictIdentity(strconv.Itoa(ready.Owner.PID)) != ready.Owner.Start {
		return fail(errors.New("controller identity unavailable or changed"))
	}
	owner = *ready.Owner
	if err = e.OnOwner(*ready.Owner); err != nil {
		return fail(err)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	authorized = true
	if err = writeOwnerFrame(input, ownerFrame{Kind: "authorize"}); err != nil {
		return fail(err)
	}
	result, err := readOwnerFrame(output)
	if err != nil {
		return fail(fmt.Errorf("controller result uncertain: %w: %s", err, diagnostics.Bytes()))
	}
	if result.Kind != "result" || result.Result == nil || !validExecutionResult(*result.Result) {
		return fail(errors.New("invalid controller result"))
	}
	completeOutput := func() error {
		prefix := stream.finish()
		if prefix > len(result.Result.Output) {
			return errors.New("stream prefix exceeds retained output")
		}
		if e.OnOutput != nil && prefix < len(result.Result.Output) {
			e.OnOutput(append([]byte(nil), result.Result.Output[prefix:]...))
		}
		return nil
	}
	if result.Result.Outcome == "uncertain" {
		// The exact owner remains alive holding leases while group cleanup is
		// unknown. Retain its durable identity; a background waiter reaps it if
		// cleanup later succeeds. No later execution is authorized here.
		go func() { _ = cmd.Wait() }()
		waited = true
		if err = completeOutput(); err != nil {
			return ExecutionResult{}, &ExecutionError{Owner: owner, MayHaveStarted: true, Err: err}
		}
		return *result.Result, &ExecutionError{Owner: owner, MayHaveStarted: true, Err: errors.New(result.Result.Error)}
	}
	input.Close()
	if err = wait(); err != nil {
		stream.finish()
		return *result.Result, fmt.Errorf("controller exit: %w", err)
	}
	if err = completeOutput(); err != nil {
		return ExecutionResult{}, &ExecutionError{Owner: owner, MayHaveStarted: true, Err: err}
	}
	return *result.Result, nil
}

func validateExecution(r ExecutionRequest) error {
	if !ValidTaskName(r.Task) || !ValidTaskName(r.RunID) || r.Revision == "" || len(r.Revision) > 256 || strings.IndexFunc(r.Revision, func(c rune) bool { return c <= 32 || c == 127 }) >= 0 {
		return errors.New("invalid execution task/run/revision")
	}
	if r.Spec.Timeout <= 0 || r.Spec.Timeout > 24*time.Hour || r.Deadline.IsZero() || !time.Now().Before(r.Deadline) || r.Deadline.After(time.Now().Add(24*time.Hour)) {
		return errors.New("invalid execution deadline")
	}
	for _, p := range []string{r.Namespace, r.HubRoot, r.CacheDir, r.PermitsDir} {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return errors.New("execution requires canonical absolute directories")
		}
		if _, err := os.Lstat(p); err == nil {
			if err = buildSafePath(p, p); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if r.Spec.Heavy && (!filepath.IsAbs(r.TestLockHelper) || filepath.Clean(r.TestLockHelper) != r.TestLockHelper) {
		return errors.New("heavy execution requires absolute admission helper")
	}
	if r.TestLockFile != "" && (!filepath.IsAbs(r.TestLockFile) || filepath.Clean(r.TestLockFile) != r.TestLockFile) {
		return errors.New("invalid admission test lock path")
	}
	if len(r.Args) > 4096 {
		return errors.New("too many execution arguments")
	}
	total := 0
	for _, a := range r.Args {
		total += len(a)
		if strings.ContainsRune(a, 0) {
			return errors.New("NUL execution argument")
		}
	}
	if total > 256<<10 {
		return errors.New("execution arguments exceed bound")
	}
	return nil
}

func validExecutionResult(r ExecutionResult) bool {
	switch r.Outcome {
	case "quiet", "wake", "failed", "cancelled", "timeout", "uncertain":
	default:
		return false
	}
	return r.Exit >= -1 && r.Exit <= 255 && len(r.Output) <= operationalrun.OutputLimit && r.OutputBytes >= int64(len(r.Output)) && len(r.Error) <= 8192
}

func writeOwnerFrame(w io.Writer, f ownerFrame) error {
	raw, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if len(raw) > ownerMaxFrame {
		return errors.New("owner frame exceeds bound")
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(raw)))
	if _, err = w.Write(header[:]); err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}
func readOwnerFrame(r io.Reader) (ownerFrame, error) {
	var f ownerFrame
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return f, err
	}
	n := binary.BigEndian.Uint32(header[:])
	if n == 0 || n > ownerMaxFrame {
		return f, errors.New("invalid owner frame length")
	}
	raw := make([]byte, n)
	if _, err := io.ReadFull(r, raw); err != nil {
		return f, err
	}
	if err := strictjson.Decode(raw, &f); err != nil {
		return f, err
	}
	return f, nil
}

// StopOwner asks the exact controller incarnation to cancel; only that
// controller has authority to signal and join its work groups.
func StopOwner(owner OwnerIdentity) error {
	if owner.PID <= 0 || owner.Start == "" || procutil.StrictIdentity(strconv.Itoa(owner.PID)) != owner.Start {
		return errors.New("execution owner identity changed or unavailable")
	}
	return syscall.Kill(owner.PID, syscall.SIGTERM)
}

// RunOwner is a private subprocess entrypoint. Its inherited stdin is both the
// authorization channel and the parent-lifetime fence.
func RunOwner(stdin io.Reader, stdout, stderr io.Writer) int {
	frame, err := readOwnerFrame(stdin)
	if err != nil || frame.Kind != "request" || frame.Request == nil {
		return 2
	}
	r := *frame.Request
	if err = validateExecution(r); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	identity := OwnerIdentity{PID: os.Getpid(), Start: procutil.StrictIdentity(strconv.Itoa(os.Getpid()))}
	stream, err := inheritedOwnerOutput(frame.StreamOutput)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if stream != nil {
		defer stream.Close()
	}
	if identity.Start == "" {
		return 2
	}
	base, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer stop()
	deadline := r.Deadline
	if timeout := time.Now().Add(r.Spec.Timeout); timeout.Before(deadline) {
		deadline = timeout
	}
	ctx, cancel := context.WithDeadline(base, deadline)
	defer cancel()
	// One reader owns stdin. Closing the pipe at any point, including while
	// waiting for admission, cancels the entire controller lifetime.
	ack := make(chan bool, 1)
	parentGone := func() {
		if parentEndCancelsOwner(time.Now(), deadline) {
			cancel()
		}
	}
	go func() {
		f, e := readOwnerFrame(stdin)
		if e != nil || f.Kind != "authorize" {
			ack <- false
			parentGone()
			return
		}
		ack <- true
		var b [1]byte
		_, _ = stdin.Read(b[:])
		parentGone()
	}()
	if writeOwnerFrame(stdout, ownerFrame{Kind: "ready", Owner: &identity}) != nil {
		return 2
	}
	select {
	case ok := <-ack:
		if !ok {
			return 2
		}
	case <-ctx.Done():
		return 2
	}
	result, hold := executeOwned(ctx, r, stream)
	if stream != nil {
		_ = stream.Close()
	}
	if len(result.Error) > 8192 {
		result.Error = result.Error[:8192]
	}
	if writeOwnerFrame(stdout, ownerFrame{Kind: "result", Result: &result}) != nil {
		fmt.Fprintln(stderr, "execution result publication failed")
	}
	if hold != nil {
		hold()
	}
	return 0
}

// A parent ending before the declared deadline cancels the run. At or after
// the deadline, the controller's own timer owns the cause. Cancelling its
// context from EOF would race that timer and mislabel a timeout as cancellation.
func parentEndCancelsOwner(now, deadline time.Time) bool { return now.Before(deadline) }

type ownerOutputReader struct {
	reader, writer *os.File
	done           chan struct{}
	prefix         int
	output         func([]byte)
}

func newOwnerOutputReader(output func([]byte)) (*ownerOutputReader, error) {
	if output == nil {
		return nil, nil
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	return &ownerOutputReader{reader: reader, writer: writer, done: make(chan struct{}), output: output}, nil
}
func (s *ownerOutputReader) abort() {
	if s != nil {
		s.reader.Close()
		s.writer.Close()
	}
}
func (s *ownerOutputReader) start() {
	if s == nil {
		return
	}
	_ = s.writer.Close()
	go func() {
		defer close(s.done)
		defer s.reader.Close()
		buffer := make([]byte, 8192)
		for s.prefix < operationalrun.OutputLimit/2 {
			n, err := s.reader.Read(buffer)
			if n > 0 {
				n = min(n, operationalrun.OutputLimit/2-s.prefix)
				s.output(append([]byte(nil), buffer[:n]...))
				s.prefix += n
			}
			if err != nil {
				return
			}
		}
	}()
}
func (s *ownerOutputReader) finish() int {
	if s == nil {
		return 0
	}
	<-s.done
	return s.prefix
}
