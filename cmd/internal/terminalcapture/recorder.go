package terminalcapture

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

const (
	// MaxSessionBytes also bounds the directory reservation budget.
	MaxSessionBytes = 32 << 30
	maxQueueBytes   = 8 << 20
	maxQueueRecords = 8192
	maxFileBytes    = 256 << 20
	finalReserve    = 4096
)

var (
	ErrQueueFull    = errors.New("terminal capture queue limit reached")
	ErrFileLimit    = errors.New("terminal capture file limit reached")
	ErrCloseTimeout = errors.New("terminal capture drain timed out")
)

// Config bounds the complete session file, including its final status record.
// Zero selects the default 256 MiB; explicit limits range from 1 MiB to 32 GiB.
type Config struct {
	MaxBytes int64
}

// Phase describes recorder admission and shutdown. Failed is terminal even when
// the worker later completes its bounded drain.
type Phase uint8

const (
	Disabled Phase = iota
	Recording
	Draining
	Closed
	Failed
)

// Status is an immutable snapshot. WrittenBytes counts accepted writer receipts,
// including a partial write and the final marker; it does not promise fsync.
type Status struct {
	Phase                    Phase
	WrittenBytes, LimitBytes int64
	Err                      error
}

type options struct {
	queueBytes, queueRecords int
	fileBytes                int64
	closeTimeout             time.Duration
}

func defaultOptions() options {
	return options{maxQueueBytes, maxQueueRecords, maxFileBytes, 2 * time.Second}
}

// Recorder has one worker. Record performs no file IO and never waits for the
// worker; its short mutex orders admission, stamps, and ownership of copied data.
// A loss stops admission permanently. Only an explicit complete end record proves
// every admitted record was written; missing or incomplete ends are not replayable
// as a complete capture. Admission order does not establish physical paint time.
type Recorder struct {
	mu              sync.Mutex
	closeOnce       sync.Once
	closeErr        error
	dir             string
	start           time.Time
	seq             uint64
	opts            options
	queue           chan Record
	done            chan struct{}
	lifecycle       lifecycle
	changes         chan struct{}
	written         int64
	percent         int64
	retained, count int
}

// Open creates a unique private session under parent. The composition root is
// responsible for validating the opt-in destination against its isolation root.
func Open(parent string, config ...Config) (*Recorder, error) {
	opts := defaultOptions()
	if len(config) > 1 {
		return nil, errors.New("terminal capture accepts at most one configuration")
	}
	if len(config) == 1 && config[0].MaxBytes != 0 {
		limit := config[0].MaxBytes
		if limit < 1<<20 || limit > MaxSessionBytes {
			return nil, errors.New("terminal capture byte limit must be between 1 MiB and 32 GiB")
		}
		opts.fileBytes = limit
	}

	dir, f, err := openCaptureFile(parent, opts.fileBytes)
	if err != nil {
		return nil, fmt.Errorf("capture admission: %w", err)
	}
	return newRecorder(dir, f, opts), nil
}

func newRecorder(dir string, out io.WriteCloser, opts options) *Recorder {
	r := &Recorder{dir: dir, start: time.Now(), opts: opts, queue: make(chan Record, opts.queueRecords), done: make(chan struct{}), lifecycle: newLifecycle(), changes: make(chan struct{}, 1)}
	build := runtime.Version()
	if info, ok := debug.ReadBuildInfo(); ok {
		build += " " + info.Main.Path + "@" + info.Main.Version
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" || setting.Key == "vcs.modified" {
				build += " " + setting.Key + "=" + setting.Value
			}
		}
	}
	r.notifyLocked() // Publish the initial recording phase before returning.
	r.Record(Record{Kind: "capture-start", PID: os.Getpid(), Build: build})
	go r.run(out)
	return r
}

func (r *Recorder) Dir() string {
	if r == nil {
		return ""
	}
	return r.dir
}

func (r *Recorder) stamp(record *Record) {
	now := time.Now()
	r.seq++
	record.Version = 1
	record.Seq = r.seq
	record.Time = now.UTC()
	record.ElapsedNS = now.Sub(r.start).Nanoseconds()
}

// Status and Changes are nil-safe so disabled capture needs no special adapter.
func (r *Recorder) Status() Status {
	if r == nil {
		return Status{Phase: Disabled}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	state := r.lifecycle.snapshot()
	return Status{Phase: state.phase, WrittenBytes: r.written, LimitBytes: r.opts.fileBytes, Err: state.err}
}

// Changes coalesces wakeups; consumers read Status after waking. The channel is
// never closed, including at shutdown, so selecting it cannot become a busy loop.
func (r *Recorder) Changes() <-chan struct{} {
	if r == nil {
		return nil
	}
	return r.changes
}

func (r *Recorder) notifyLocked() {
	select {
	case r.changes <- struct{}{}:
	default:
	}
}

func (r *Recorder) transitionLocked(event lifecycleEvent) {
	effects := r.lifecycle.transition(event)
	if effects.closeAdmission {
		close(r.queue)
	}
	if effects.notify {
		r.notifyLocked()
	}
}

// receipt accounts for bytes only after Write acknowledges them. Usage wakeups
// occur at most 100 times over the finite file budget, never for each record.
func (r *Recorder) receipt(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.written += int64(n)
	percent := r.written * 100 / r.opts.fileBytes
	if percent != r.percent {
		r.percent = percent
		r.notifyLocked()
	}
}

// Record copies borrowed bytes before returning. Nil is the disabled hot path.
// Queue limits include the in-flight write so a blocked disk cannot defeat them.
func (r *Recorder) Record(record Record) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lifecycle.snapshot().phase != Recording {
		return
	}
	cost := record.retainedBytes()
	if cost > r.opts.queueBytes-r.retained || r.count >= r.opts.queueRecords {
		r.transitionLocked(lifecycleEvent{kind: captureFailed, err: ErrQueueFull})
		return
	}
	record = record.owned()
	r.stamp(&record)
	r.retained += cost
	r.count++
	r.queue <- record // count admits space before send; never waits on file IO.
}

func (r *Recorder) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transitionLocked(lifecycleEvent{kind: captureFailed, err: err})
}

func (r *Recorder) run(out io.WriteCloser) {
	defer close(r.done)
	var written int64
	writeFailed := false
	for record := range r.queue {
		if !writeFailed {
			data, err := json.Marshal(record)
			if err == nil && int64(len(data)+1) > r.opts.fileBytes-finalReserve-written {
				err = ErrFileLimit
			}
			if err == nil {
				data = append(data, '\n')
				var n int
				n, err = out.Write(data)
				written += int64(n)
				r.receipt(n)
				if err == nil && n != len(data) {
					err = io.ErrShortWrite
				}
			}
			if err != nil {
				r.fail(err)
				writeFailed = true
			}
		}
		r.mu.Lock()
		r.retained -= record.retainedBytes()
		r.count--
		r.mu.Unlock()
	}
	r.mu.Lock()
	end := Record{Kind: "capture-end", Status: "complete"}
	if failure := r.lifecycle.snapshot().err; failure != nil {
		end.Status = "incomplete"
		end.Error = failure.Error()
		if len(end.Error) > 512 {
			end.Error = end.Error[:512]
		}
	}
	r.stamp(&end)
	r.mu.Unlock()
	data, err := json.Marshal(end)
	if err == nil {
		data = append(data, '\n')
		if int64(len(data)) > r.opts.fileBytes-written {
			err = ErrFileLimit
		} else {
			var n int
			n, err = out.Write(data)
			r.receipt(n)
			if err == nil && n != len(data) {
				err = io.ErrShortWrite
			}
		}
	}
	if err != nil {
		r.fail(err)
	}
	if err = out.Close(); err != nil {
		r.fail(err)
	}
	r.mu.Lock()
	r.transitionLocked(lifecycleEvent{kind: workerCompleted})
	r.mu.Unlock()
}

// Close stops admission, drains admitted records, and closes the file. Calls are
// idempotent. Terminal restoration must precede this bounded diagnostic wait.
// Regular-file writes are not reliably cancellable: on timeout, the one worker
// can remain blocked until the OS returns or the process exits. It never resumes
// admission. An end record already being written may later finish; "complete"
// describes admitted data, while Close's error also reports shutdown failure.
func (r *Recorder) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		r.mu.Lock()
		r.transitionLocked(lifecycleEvent{kind: closeRequested})
		r.mu.Unlock()
		timer := time.NewTimer(r.opts.closeTimeout)
		defer timer.Stop()
		select {
		case <-r.done:
		case <-timer.C:
			r.mu.Lock()
			r.transitionLocked(lifecycleEvent{kind: drainTimedOut})
			r.mu.Unlock()
		}
		r.mu.Lock()
		r.closeErr = r.lifecycle.snapshot().err
		r.mu.Unlock()
	})
	return r.closeErr
}
