package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestRecoveryQueryCaptureExactBytesOwnershipAndClassification(t *testing.T) {
	capture := newRecoveryQueryCapture(t)
	deadline := time.Now().Add(time.Minute)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	start := time.Now()
	event := launcher.ZellijQueryObservation{Args: []string{"--session", "fixture", "action", "list-clients"}, Started: start, Finished: start.Add(time.Millisecond), Duration: time.Millisecond, Stdout: []byte{0, 255, '\n'}, Stderr: []byte("fixture diagnostic\n"), StdoutBytes: 6, StderrBytes: 19, StdoutTruncated: true, StderrTruncated: true, ExitCode: 7, Err: fmt.Errorf("fixture response: %w", launcher.ErrZellijNoReply)}
	capture.observe(ctx, event)
	wantOut, wantErr := bytes.Clone(event.Stdout), bytes.Clone(event.Stderr)
	event.Args[0] = "mutated"
	event.Stdout[0] = 'x'
	event.Stderr[0] = 'x'
	snapshot := capture.snapshot()
	if len(snapshot.Records) != 1 {
		t.Fatalf("records=%d", len(snapshot.Records))
	}
	got := snapshot.Records[0]
	if got.Args[0] != "--session" || !bytes.Equal(got.Stdout, wantOut) || !bytes.Equal(got.Stderr, wantErr) {
		t.Fatalf("capture lost exact owned response: %+v", got)
	}
	if got.StdoutBytes != 6 || got.StderrBytes != 19 || !got.StdoutTruncated || !got.StderrTruncated || got.ExitCode != 7 || !got.NoReply || got.Cancelled || got.Error == "" || got.CallerDeadline == nil || !got.CallerDeadline.Equal(deadline) || got.Started != start || got.Finished != start.Add(time.Millisecond) || got.Duration != time.Millisecond {
		t.Fatalf("classification/timing=%+v", got)
	}
	got.Args[0] = "snapshot-mutated"
	got.Stdout[0] = 'z'
	if again := capture.snapshot().Records[0]; again.Args[0] != "--session" || !bytes.Equal(again.Stdout, wantOut) {
		t.Fatal("snapshot aliases retained capture")
	}
}

func TestRecoveryQueryCaptureBoundsRecordsAndCopiedBytes(t *testing.T) {
	capture := &recoveryQueryCapture{}
	out, stderr := bytes.Repeat([]byte("o"), 4<<20+17), bytes.Repeat([]byte("e"), 4<<20+29)
	capture.observe(context.Background(), launcher.ZellijQueryObservation{Stdout: out, Stderr: stderr, StdoutBytes: len(out), StderrBytes: len(stderr)})
	for i := 0; i < 64; i++ {
		capture.Mark(fmt.Sprintf("mark-%d", i))
	}
	capture.observe(context.Background(), launcher.ZellijQueryObservation{Stdout: []byte("gone"), StdoutBytes: 4})
	got := capture.snapshot()
	if len(got.Records) != 64 || got.CopiedBytes != 8<<20 || got.DroppedRecords != 2 || got.DroppedBytes != 50 {
		t.Fatalf("bounds: records=%d bytes=%d dropped-records=%d dropped-bytes=%d", len(got.Records), got.CopiedBytes, got.DroppedRecords, got.DroppedBytes)
	}
	retainedCapacity := 0
	for _, record := range got.Records {
		retainedCapacity += cap(record.Stdout) + cap(record.Stderr)
	}
	if retainedCapacity > 8<<20 || !got.Records[0].StderrTruncated || got.Records[0].StdoutTruncated || !bytes.Equal(got.Records[0].Stdout, out) {
		t.Fatalf("bounded exact prefix/flags: capacity=%d first=%+v", retainedCapacity, got.Records[0].StderrTruncated)
	}
}

func TestRecoveryQueryCaptureConcurrentMarksAndContextHelper(t *testing.T) {
	capture := newRecoveryQueryCapture(t)
	markRecoveryQuery(context.Background(), "not captured")
	ctx := capture.Context(context.Background())
	markRecoveryQuery(ctx, "fixture boundary")
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 16; j++ {
				markRecoveryQuery(ctx, "concurrent fixture boundary")
			}
		}()
	}
	workers.Wait()
	got := capture.snapshot()
	if len(got.Records) != 64 || got.DroppedRecords != 65 || got.Records[0].Name != "fixture boundary" || got.Records[0].Started.IsZero() {
		t.Fatalf("concurrent lifecycle capture: records=%d dropped=%d", len(got.Records), got.DroppedRecords)
	}
}

func TestRecoveryQueryCaptureObserverDoesNotChangeQueryTrace(t *testing.T) {
	traces := map[int][]zellijtest.Query{}
	for _, count := range []int{8, 16} {
		var baseline []zellijtest.Query
		for _, observe := range []bool{false, true} {
			t.Run(fmt.Sprintf("operations-%d/observer-%v", count, observe), func(t *testing.T) {
				capture := newRecoveryQueryCapture(t)
				fixture := zellijtest.New(t)
				for _, setting := range fixture.Env() {
					key, value, _ := strings.Cut(setting, "=")
					t.Setenv(key, value)
				}
				fixture.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-fixture": {Clients: []zellijtest.Client{{ID: 1, PaneID: "terminal_0", RunningCommand: "fixture"}}}}})
				ctx := t.Context()
				if observe {
					ctx = capture.Context(ctx)
				}
				source := launcher.ZellijSource{Path: fixture.Path()}
				for operation := 0; operation < count; operation++ {
					got, err := source.ClientCountContext(ctx, "pair-fixture")
					if err != nil || got != 1 {
						t.Fatalf("operation %d: clients=%d err=%v", operation, got, err)
					}
				}
				queries := fixture.Queries()
				if len(queries) != count || len(fixture.Effects()) != 0 {
					t.Fatalf("observational calls changed work: queries=%d effects=%d", len(queries), len(fixture.Effects()))
				}
				if observe {
					if !reflect.DeepEqual(queries, baseline) {
						t.Fatalf("observer changed trace: observed=%+v baseline=%+v", queries, baseline)
					}
					if got := len(capture.snapshot().Records); got != count {
						t.Fatalf("captured=%d want=%d", got, count)
					}
				} else {
					baseline = queries
				}
			})
		}
		traces[count] = baseline
	}
	if len(traces[16]) != 2*len(traces[8]) {
		t.Fatalf("query work did not double: 8=%d 16=%d", len(traces[8]), len(traces[16]))
	}
}

func TestRecoveryQueryCapturePreservesCanceledFakeQuery(t *testing.T) {
	capture := newRecoveryQueryCapture(t)
	fixture := zellijtest.New(t)
	for _, setting := range fixture.Env() {
		key, value, _ := strings.Cut(setting, "=")
		t.Setenv(key, value)
	}
	lock, err := os.OpenFile(filepath.Join(fixture.Root(), "state.lock"), os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	ready := filepath.Join(fixture.Root(), "ready")
	wrapper := filepath.Join(fixture.Root(), "blocked-query")
	script := "#!/bin/sh\nprintf ready > '" + ready + "'\nexec '" + fixture.Path() + "' \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx = capture.Context(ctx)
	done := make(chan error, 1)
	go func() {
		_, err := (launcher.ZellijSource{Path: wrapper}).ClientCountContext(ctx, "pair-fixture")
		done <- err
	}()
	testwait.Until(t, "fake query process started", func() bool { _, err := os.Stat(ready); return err == nil })
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation classification=%v", err)
		}
	case <-testwait.After(t):
		t.Fatal("canceled fake query did not join")
	}
	got := capture.snapshot()
	if len(got.Records) != 1 || !got.Records[0].Cancelled || got.Records[0].ExitCode == 0 || got.Records[0].NoReply {
		t.Fatalf("canceled evidence=%+v", got)
	}
}

const recoveryQueryCaptureRecordLimit = 64
const recoveryQueryCaptureByteLimit = 8 << 20
const recoveryQueryCaptureMetadataLimit = 16 << 10

type recoveryQueryCaptureRecord struct {
	Kind, Name                                   string
	Args                                         []string
	Started, Finished                            time.Time
	Duration                                     time.Duration
	CallerDeadline                               *time.Time
	Stdout, Stderr                               []byte
	StdoutBytes, StderrBytes                     int
	StdoutTruncated, StderrTruncated             bool
	ExitCode                                     int
	Error                                        string
	Cancelled, DeadlineExceeded, NoReply         bool
	ArgsTruncated, NameTruncated, ErrorTruncated bool
}
type recoveryQueryCaptureSnapshot struct {
	Records                                                         []recoveryQueryCaptureRecord
	CopiedBytes, DroppedRecords, DroppedBytes, DroppedMetadataBytes int
}
type recoveryQueryCapture struct {
	mu    sync.Mutex
	state recoveryQueryCaptureSnapshot
}
type recoveryQueryCaptureContextKey struct{}

// Register before fixture cleanup: testing's LIFO order then emits the capture
// only after all later fixture cleanup and its lifecycle marks have completed.
func newRecoveryQueryCapture(t testing.TB) *recoveryQueryCapture {
	t.Helper()
	capture := &recoveryQueryCapture{}
	t.Cleanup(func() {
		raw, err := json.Marshal(capture.snapshot())
		if err != nil {
			t.Errorf("serialize recovery query capture: %v", err)
			return
		}
		t.Logf("recovery query capture: %s", raw)
	})
	return capture
}
func (r *recoveryQueryCapture) Context(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, recoveryQueryCaptureContextKey{}, r)
	return launcher.WithZellijQueryObserver(ctx, func(event launcher.ZellijQueryObservation) { r.observe(ctx, event) })
}
func markRecoveryQuery(ctx context.Context, name string) {
	if capture, ok := ctx.Value(recoveryQueryCaptureContextKey{}).(*recoveryQueryCapture); ok {
		capture.Mark(name)
	}
}
func (r *recoveryQueryCapture) Mark(name string) {
	now := time.Now()
	r.append(recoveryQueryCaptureRecord{Kind: "lifecycle", Name: name, Started: now, Finished: now})
}
func (r *recoveryQueryCapture) observe(ctx context.Context, event launcher.ZellijQueryObservation) {
	record := recoveryQueryCaptureRecord{Kind: "query", Args: event.Args, Started: event.Started, Finished: event.Finished, Duration: event.Duration, Stdout: event.Stdout, Stderr: event.Stderr, StdoutBytes: event.StdoutBytes, StderrBytes: event.StderrBytes, StdoutTruncated: event.StdoutTruncated, StderrTruncated: event.StderrTruncated, ExitCode: event.ExitCode, Cancelled: errors.Is(event.Err, context.Canceled), DeadlineExceeded: errors.Is(event.Err, context.DeadlineExceeded), NoReply: errors.Is(event.Err, launcher.ErrZellijNoReply)}
	if event.Err != nil {
		record.Error = event.Err.Error()
	}
	if deadline, ok := ctx.Deadline(); ok {
		record.CallerDeadline = &deadline
	}
	r.append(record)
}

// append owns the only mutation path. It does no IO, logging or extra query.
// Output capacity is exact; metadata and argv count have separate fixed bounds
// so even hostile fixture inputs cannot make a retained record unbounded.
func (r *recoveryQueryCapture) append(record recoveryQueryCaptureRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.state.Records) >= recoveryQueryCaptureRecordLimit {
		r.state.DroppedRecords++
		r.state.DroppedBytes += record.StdoutBytes + record.StderrBytes
		return
	}
	remaining := recoveryQueryCaptureByteLimit - r.state.CopiedBytes
	copyOutput := func(raw []byte, original int, truncated bool) ([]byte, bool) {
		n := min(len(raw), remaining)
		copied := make([]byte, n)
		copy(copied, raw[:n])
		remaining -= n
		r.state.CopiedBytes += n
		r.state.DroppedBytes += max(0, original-n)
		return copied, truncated || original > n || len(raw) > n
	}
	record.Stdout, record.StdoutTruncated = copyOutput(record.Stdout, record.StdoutBytes, record.StdoutTruncated)
	record.Stderr, record.StderrTruncated = copyOutput(record.Stderr, record.StderrBytes, record.StderrTruncated)
	metadataRemaining := recoveryQueryCaptureMetadataLimit
	copyText := func(raw string) (string, bool) {
		n := min(len(raw), metadataRemaining)
		metadataRemaining -= n
		r.state.DroppedMetadataBytes += len(raw) - n
		return strings.Clone(raw[:n]), n < len(raw)
	}
	args := make([]string, min(len(record.Args), recoveryQueryCaptureRecordLimit))
	for i := range args {
		value, truncated := copyText(record.Args[i])
		args[i] = value
		record.ArgsTruncated = record.ArgsTruncated || truncated
	}
	if len(args) != len(record.Args) {
		record.ArgsTruncated = true
		for _, arg := range record.Args[len(args):] {
			r.state.DroppedMetadataBytes += len(arg)
		}
	}
	record.Args = args
	record.Name, record.NameTruncated = copyText(record.Name)
	record.Error, record.ErrorTruncated = copyText(record.Error)
	if r.state.Records == nil {
		r.state.Records = make([]recoveryQueryCaptureRecord, 0, recoveryQueryCaptureRecordLimit)
	}
	r.state.Records = append(r.state.Records, record)
}
func (r *recoveryQueryCapture) snapshot() recoveryQueryCaptureSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	snapshot := r.state
	snapshot.Records = make([]recoveryQueryCaptureRecord, len(r.state.Records))
	for i, record := range r.state.Records {
		record.Args = append([]string(nil), record.Args...)
		stdout, stderr := make([]byte, len(record.Stdout)), make([]byte, len(record.Stderr))
		copy(stdout, record.Stdout)
		copy(stderr, record.Stderr)
		record.Stdout, record.Stderr = stdout, stderr
		if record.CallerDeadline != nil {
			deadline := *record.CallerDeadline
			record.CallerDeadline = &deadline
		}
		snapshot.Records[i] = record
	}
	return snapshot
}

type recoveryQueryCaptureLogTB struct {
	testing.TB
	cleanups []func()
	logs     []string
}

func (t *recoveryQueryCaptureLogTB) Cleanup(fn func()) { t.cleanups = append(t.cleanups, fn) }
func (t *recoveryQueryCaptureLogTB) Logf(format string, args ...any) {
	t.logs = append(t.logs, fmt.Sprintf(format, args...))
}

func TestRecoveryQueryCaptureDumpsOnlyAfterFixtureCleanup(t *testing.T) {
	sink := &recoveryQueryCaptureLogTB{TB: t}
	capture := newRecoveryQueryCapture(sink)
	capture.Mark("fixture body")
	sink.Cleanup(func() { capture.Mark("fixture cleanup") })
	if len(sink.logs) != 0 {
		t.Fatal("capture logged before cleanup")
	}
	for i := len(sink.cleanups) - 1; i >= 0; i-- {
		sink.cleanups[i]()
	}
	if len(sink.logs) != 1 {
		t.Fatalf("cleanup logs=%d want 1", len(sink.logs))
	}
	var got recoveryQueryCaptureSnapshot
	if err := json.Unmarshal([]byte(strings.TrimPrefix(sink.logs[0], "recovery query capture: ")), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Records) != 2 || got.Records[0].Name != "fixture body" || got.Records[1].Name != "fixture cleanup" {
		t.Fatalf("deferred dump preceded fixture cleanup: %+v", got)
	}
}
