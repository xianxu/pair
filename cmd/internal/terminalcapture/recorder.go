package terminalcapture

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

const (
	maxQueueBytes   = 8 << 20
	maxQueueRecords = 128
	maxFileBytes    = 256 << 20
	finalReserve    = 4096
)

var (
	ErrQueueFull    = errors.New("terminal capture queue limit reached")
	ErrFileLimit    = errors.New("terminal capture file limit reached")
	ErrCloseTimeout = errors.New("terminal capture drain timed out")
)

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
	stopped         bool
	retained, count int
	failure         error
}

// Open creates a unique private session under parent. The composition root is
// responsible for validating the opt-in destination against its isolation root.
func Open(parent string) (*Recorder, error) {
	if err := os.MkdirAll(parent, 0700); err != nil {
		return nil, fmt.Errorf("capture directory: %w", err)
	}
	dir, err := os.MkdirTemp(parent, "session-")
	if err != nil {
		return nil, fmt.Errorf("capture session: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		_ = os.Remove(dir)
		return nil, fmt.Errorf("capture events: %w", err)
	}
	return newRecorder(dir, f, defaultOptions()), nil
}

func newRecorder(dir string, out io.WriteCloser, opts options) *Recorder {
	r := &Recorder{dir: dir, start: time.Now(), opts: opts, queue: make(chan Record, opts.queueRecords), done: make(chan struct{})}
	build := runtime.Version()
	if info, ok := debug.ReadBuildInfo(); ok {
		build += " " + info.Main.Path + "@" + info.Main.Version
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" || setting.Key == "vcs.modified" {
				build += " " + setting.Key + "=" + setting.Value
			}
		}
	}
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

func (r *Recorder) stopLocked(err error) {
	if err != nil {
		r.failure = errors.Join(r.failure, err)
	}
	if !r.stopped {
		r.stopped = true
		close(r.queue)
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
	if r.stopped {
		return
	}
	cost := record.retainedBytes()
	if cost > r.opts.queueBytes-r.retained || r.count >= r.opts.queueRecords {
		r.stopLocked(ErrQueueFull)
		return
	}
	record.Data = append([]byte(nil), record.Data...)
	r.stamp(&record)
	r.retained += cost
	r.count++
	r.queue <- record // count admits space before send; never waits on file IO.
}

func (r *Recorder) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopLocked(err)
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
	if r.failure != nil {
		end.Status = "incomplete"
		end.Error = r.failure.Error()
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
		r.stopLocked(nil)
		r.mu.Unlock()
		timer := time.NewTimer(r.opts.closeTimeout)
		defer timer.Stop()
		select {
		case <-r.done:
		case <-timer.C:
			r.mu.Lock()
			if !errors.Is(r.failure, ErrCloseTimeout) {
				r.failure = errors.Join(r.failure, ErrCloseTimeout)
			}
			r.mu.Unlock()
		}
		r.mu.Lock()
		r.closeErr = r.failure
		r.mu.Unlock()
	})
	return r.closeErr
}
