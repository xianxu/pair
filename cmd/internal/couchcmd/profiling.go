package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"
)

const profileArtifactLimit = 32 << 20
const profileMetadataLimit = 64 << 10

// ProfileRequest is an explicit launch-only diagnostic destination.
type ProfileRequest struct{ Dir string }

// ParseProfileRequest validates syntax without touching the filesystem.
func ParseProfileRequest(raw string) (ProfileRequest, error) {
	if raw != "" && (!filepath.IsAbs(raw) || filepath.Clean(raw) != raw) {
		return ProfileRequest{}, errors.New("COUCH_PROFILE_DIR requires a clean absolute path")
	}
	return ProfileRequest{Dir: raw}, nil
}

type profileBackend interface {
	StartCPU(io.Writer) error
	StopCPU()
	Allocations(io.Writer) error
}
type runtimeProfileBackend struct{}

func (runtimeProfileBackend) StartCPU(w io.Writer) error { return pprof.StartCPUProfile(w) }
func (runtimeProfileBackend) StopCPU()                   { pprof.StopCPUProfile() }
func (runtimeProfileBackend) Allocations(w io.Writer) error {
	return pprof.Lookup("allocs").WriteTo(w, 0)
}

type profileClock interface {
	Wait(context.Context, time.Duration) bool
}
type systemProfileClock struct{}

func (systemProfileClock) Wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type profileStorage interface {
	Reserve(string) error
	Create(string, string) (io.WriteCloser, error)
}
type osProfileStorage struct{}

func (osProfileStorage) Reserve(dir string) error { return os.Mkdir(dir, 0700) }
func (osProfileStorage) Create(dir, name string) (io.WriteCloser, error) {
	return os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}

type profileDependencies struct {
	backend profileBackend
	clock   profileClock
	storage profileStorage
}

// profileWriter bounds diagnostic disk growth and preserves errors hidden by
// runtime/pprof's asynchronous CPU writer. StopCPU must join before Close.
type profileWriter struct {
	mu        sync.Mutex
	file      io.WriteCloser
	remaining int
	first     error
	closed    bool
}

func (w *profileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.first != nil {
		return 0, w.first
	}
	if w.closed {
		return 0, os.ErrClosed
	}
	over := len(p) > w.remaining
	if over {
		p = p[:w.remaining]
	}
	n, err := w.file.Write(p)
	w.remaining -= n
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err == nil && over {
		err = errors.New("profile artifact byte limit exceeded")
	}
	if err != nil {
		w.first = err
	}
	return n, err
}
func (w *profileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.closed {
		w.closed = true
		if err := w.file.Close(); w.first == nil {
			w.first = err
		}
	}
	return w.first
}

type profileMetadata struct {
	Status         string           `json:"status"`
	Failure        string           `json:"failure,omitempty"`
	Started        time.Time        `json:"started"`
	CaptureStarted time.Time        `json:"capture_started,omitempty"`
	Finished       time.Time        `json:"finished"`
	WarmupSeconds  int              `json:"warmup_seconds"`
	CPUSeconds     int              `json:"cpu_seconds"`
	PID            int              `json:"pid"`
	Build          *debug.BuildInfo `json:"build,omitempty"`
}

// ProfileWindow owns one serialized interpreter and all its output files.
// Close cancels and joins; it never races the worker to release the profiler.
type ProfileWindow struct {
	cancel  context.CancelFunc
	done    chan struct{}
	failure error
}

func (w *ProfileWindow) Close() error { w.cancel(); <-w.done; return w.failure }

// StartProfileWindow reserves new private outputs, then arms a single finite
// profiling window. A disabled request creates no files or goroutines.
func StartProfileWindow(parent context.Context, request ProfileRequest, deps profileDependencies, notify func(string)) (*ProfileWindow, error) {
	if request.Dir == "" {
		return nil, nil
	}
	if _, err := ParseProfileRequest(request.Dir); err != nil {
		return nil, err
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	if deps.backend == nil {
		deps.backend = runtimeProfileBackend{}
	}
	if deps.clock == nil {
		deps.clock = systemProfileClock{}
	}
	if deps.storage == nil {
		deps.storage = osProfileStorage{}
	}
	if err := deps.storage.Reserve(request.Dir); err != nil {
		return nil, fmt.Errorf("profile reserve: %w", err)
	}
	outputs := make(map[string]*profileWriter, 4)
	for _, name := range []string{"cpu.pprof", "allocs-before.pprof", "allocs-after.pprof", "metadata.json"} {
		file, err := deps.storage.Create(request.Dir, name)
		if err != nil {
			var cleanup error
			for _, w := range outputs {
				cleanup = errors.Join(cleanup, w.Close())
			}
			return nil, errors.Join(fmt.Errorf("profile create %s: %w", name, err), cleanup)
		}
		limit := profileArtifactLimit
		if name == "metadata.json" {
			limit = profileMetadataLimit
		}
		outputs[name] = &profileWriter{file: file, remaining: limit}
	}
	ctx, cancel := context.WithCancel(parent)
	window := &ProfileWindow{cancel: cancel, done: make(chan struct{})}
	if notify == nil {
		notify = func(string) {}
	}
	notify("profile armed: " + request.Dir + "; capture after 60s for 30s")
	go window.run(ctx, request.Dir, deps, outputs, notify)
	return window, nil
}

func (w *ProfileWindow) run(ctx context.Context, dir string, deps profileDependencies, outputs map[string]*profileWriter, notify func(string)) {
	defer close(w.done)
	defer w.cancel()
	state := ProfileLifecycle{}
	metadata := profileMetadata{Started: time.Now(), WarmupSeconds: 60, CPUSeconds: 30, PID: os.Getpid()}
	metadata.Build, _ = debug.ReadBuildInfo()
	step := func(event profileEvent) profileEffect {
		next, effect, err := StepProfile(state, event)
		if err != nil {
			panic(err)
		} // interpreter/model mismatch; not an IO outcome
		state = next
		return effect
	}
	event := profileEvent{kind: profileCancel}
	if deps.clock.Wait(ctx, time.Minute) {
		event.kind = profileWarmupElapsed
	}
	effect := step(event)
	if effect == profileAcquire {
		err := deps.backend.Allocations(outputs["allocs-before.pprof"])
		err = errors.Join(err, outputs["allocs-before.pprof"].Close())
		switch {
		case err != nil:
			effect = step(profileEvent{kind: profileAcquireFailed, err: err})
		case ctx.Err() != nil:
			effect = step(profileEvent{kind: profileCancel})
		default:
			err = deps.backend.StartCPU(outputs["cpu.pprof"])
			switch {
			case err != nil:
				effect = step(profileEvent{kind: profileAcquireFailed, err: err})
			case ctx.Err() != nil:
				effect = step(profileEvent{kind: profileAcquiredCancelled})
			default:
				step(profileEvent{kind: profileAcquired})
				metadata.CaptureStarted = time.Now()
				notify("profile capturing: " + dir)
				event = profileEvent{kind: profileCancel}
				if deps.clock.Wait(ctx, 30*time.Second) {
					event.kind = profileCaptureElapsed
				}
				effect = step(event)
			}
		}
	}
	var failure error
	if effect == profileReleaseOwned {
		deps.backend.StopCPU()
		failure = outputs["cpu.pprof"].Close()
		failure = errors.Join(failure, deps.backend.Allocations(outputs["allocs-after.pprof"]))
	}
	for _, name := range []string{"cpu.pprof", "allocs-before.pprof", "allocs-after.pprof"} {
		failure = errors.Join(failure, outputs[name].Close())
	}
	metadata.Status = "complete"
	if state.cancelled {
		metadata.Status = "cancelled"
	}
	cause := state.failure
	if cause == nil {
		cause = failure
	}
	if cause != nil {
		metadata.Status = "failed"
		metadata.Failure = cause.Error()
	}
	metadata.Finished = time.Now()
	failure = errors.Join(failure, json.NewEncoder(outputs["metadata.json"]).Encode(metadata), outputs["metadata.json"].Close())
	step(profileEvent{kind: profileReleased, err: failure})
	w.failure = state.failure
	if w.failure != nil {
		notify("profile failed/incomplete: " + dir + ": " + w.failure.Error())
	} else {
		notify("profile " + metadata.Status + ": " + dir)
	}
}

// runProfiledConsole binds the optional worker to the existing Console lifetime.
// It reports diagnostic failure through that owner and preserves its exit code.
func runProfiledConsole(getenv func(string) string, notify func(string), run func() int) (code int, profileErr error) {
	request, err := ParseProfileRequest(getenv("COUCH_PROFILE_DIR"))
	if err != nil {
		notify("profile unavailable: " + err.Error())
		return run(), nil
	}
	window, err := StartProfileWindow(context.Background(), request, profileDependencies{}, notify)
	if err != nil {
		notify("profile unavailable: " + err.Error())
	} else if window != nil {
		defer func() { profileErr = window.Close() }()
	}
	return run(), nil
}
