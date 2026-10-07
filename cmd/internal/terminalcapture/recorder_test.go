package terminalcapture

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func readRecords(t *testing.T, data []byte) []Record {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	var records []Record
	for {
		var record Record
		if err := dec.Decode(&record); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func TestPrivateSessionsRoundTripAndClose(t *testing.T) {
	parent := t.TempDir()
	a, err := Open(parent)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(parent)
	if err != nil {
		t.Fatal(err)
	}
	if a.Dir() == b.Dir() {
		t.Fatal("sessions reused a directory")
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	data := []byte{0, 27, 255, '\n', 128}
	a.Record(Record{Kind: "host-write", Data: data, Requested: len(data), Accepted: 0})
	data[0] = 'x'
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]os.FileMode{a.Dir(): 0700, filepath.Join(a.Dir(), "events.jsonl"): 0600} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != want {
			t.Fatalf("mode %s: %v %v", path, info, err)
		}
	}
	raw, err := os.ReadFile(filepath.Join(a.Dir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	records := readRecords(t, raw)
	if len(records) != 3 || records[0].Kind != "capture-start" || records[2].Status != "complete" {
		t.Fatalf("records: %+v", records)
	}
	if !bytes.Equal(records[1].Data, []byte{0, 27, 255, '\n', 128}) {
		t.Fatal("borrowed bytes mutated")
	}
	if !bytes.Contains(raw, []byte(`"accepted":0`)) {
		t.Fatal("zero write acceptance absent")
	}
	for i, r := range records {
		if r.Version != 1 || r.Seq != uint64(i+1) || r.Time.IsZero() || r.ElapsedNS < 0 {
			t.Fatalf("stamp: %+v", r)
		}
		if i > 0 && r.ElapsedNS < records[i-1].ElapsedNS {
			t.Fatal("monotonic admission time regressed")
		}
	}
	if records[0].PID != os.Getpid() || records[0].Build == "" {
		t.Fatal("missing initial process/build metadata")
	}
}

func TestNilRecorderAndOpenError(t *testing.T) {
	var r *Recorder
	r.Record(Record{Kind: "disabled"})
	if r.Dir() != "" || r.Close() != nil {
		t.Fatal("nil recorder not disabled")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(file); err == nil {
		t.Fatal("opened non-directory")
	}
}

type faultSink struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	block   <-chan struct{}
	started chan struct{}
	fail    error
	short   bool
	closed  bool
}

func (s *faultSink) Write(p []byte) (int, error) {
	if s.started != nil {
		select {
		case s.started <- struct{}{}:
		default:
		}
	}
	if s.block != nil {
		<-s.block
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail != nil {
		return 0, s.fail
	}
	if s.short {
		return s.buf.Write(p[:len(p)/2])
	}
	return s.buf.Write(p)
}
func (s *faultSink) Close() error { s.mu.Lock(); defer s.mu.Unlock(); s.closed = true; return nil }
func (s *faultSink) bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.buf.Bytes()...)
}

func TestConcurrentAdmissionsRemainOrdered(t *testing.T) {
	s := &faultSink{}
	o := defaultOptions()
	o.queueRecords = 2048
	r := newRecorder("", s, o)
	var wg sync.WaitGroup
	for producer := 0; producer < 8; producer++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				r.Record(Record{Kind: "endpoint-feed", Data: []byte("a")})
			}
		}()
	}
	wg.Wait()
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	records := readRecords(t, s.bytes())
	if len(records) != 802 {
		t.Fatalf("records=%d", len(records))
	}
	for i, r := range records {
		if r.Seq != uint64(i+1) {
			t.Fatalf("seq=%d at%d", r.Seq, i)
		}
	}
}

func TestConcurrentRecordAndClosePreserveFinalAdmissionOrder(t *testing.T) {
	for attempt := 0; attempt < 20; attempt++ {
		sink := &faultSink{}
		opts := defaultOptions()
		// The test exercises shutdown admission, independently of overflow.
		opts.queueRecords = 2048
		recorder := newRecorder("", sink, opts)
		start := make(chan struct{})
		admitted := make(chan struct{}, 8)
		var producers sync.WaitGroup
		for producer := 0; producer < 8; producer++ {
			producers.Add(1)
			go func() {
				defer producers.Done()
				<-start
				recorder.Record(Record{Kind: "endpoint-feed", Data: []byte("first")})
				admitted <- struct{}{}
				for i := 0; i < 100; i++ {
					recorder.Record(Record{Kind: "endpoint-feed", Data: []byte("during-close")})
				}
			}()
		}
		close(start)
		<-admitted
		closed := make(chan error, 4)
		for i := 0; i < cap(closed); i++ {
			go func() { closed <- recorder.Close() }()
		}
		producers.Wait()
		for i := 0; i < cap(closed); i++ {
			select {
			case err := <-closed:
				if err != nil {
					t.Fatalf("attempt %d close: %v", attempt, err)
				}
			case <-time.After(time.Second):
				t.Fatalf("attempt %d did not drain", attempt)
			}
		}
		recorder.Record(Record{Kind: "after-close"})
		records := readRecords(t, sink.bytes())
		if len(records) < 3 || records[0].Kind != "capture-start" {
			t.Fatalf("attempt %d lost initial admission: %+v", attempt, records)
		}
		for i, record := range records {
			if record.Seq != uint64(i+1) {
				t.Fatalf("attempt %d sequence %d at %d", attempt, record.Seq, i)
			}
			if record.Kind == "after-close" || record.Kind == "capture-end" && i != len(records)-1 {
				t.Fatalf("attempt %d admitted after end: %+v", attempt, record)
			}
		}
		end := records[len(records)-1]
		if end.Kind != "capture-end" || end.Status != "complete" {
			t.Fatalf("attempt %d final record: %+v", attempt, end)
		}
	}
}

func TestQueueOverflowStopsWithoutWaitingForDisk(t *testing.T) {
	for _, limit := range []string{"records", "bytes"} {
		t.Run(limit, func(t *testing.T) {
			unblock := make(chan struct{})
			s := &faultSink{block: unblock, started: make(chan struct{}, 1)}
			o := defaultOptions()
			if limit == "records" {
				o.queueRecords = 2
			} else {
				o.queueBytes = 1024
			}
			r := newRecorder("", s, o)
			<-s.started
			done := make(chan struct{})
			go func() {
				defer close(done)
				r.Record(Record{Kind: "feed", Data: bytes.Repeat([]byte("x"), 600)})
				r.Record(Record{Kind: "feed", Data: bytes.Repeat([]byte("x"), 600)})
				r.Record(Record{Kind: "must-not-resume"})
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("producer waited for disk")
			}
			close(unblock)
			if err := r.Close(); !errors.Is(err, ErrQueueFull) {
				t.Fatalf("close: %v", err)
			}
			records := readRecords(t, s.bytes())
			if records[len(records)-1].Status != "incomplete" {
				t.Fatal("overflow claimed complete")
			}
			for _, record := range records {
				if record.Kind == "must-not-resume" {
					t.Fatal("capture resumed after loss")
				}
			}
		})
	}
}

func TestDiskCapPreservesFinalIncompleteRecord(t *testing.T) {
	s := &faultSink{}
	o := defaultOptions()
	o.fileBytes = finalReserve + 1200
	r := newRecorder("", s, o)
	r.Record(Record{Kind: "feed", Data: bytes.Repeat([]byte("x"), 2000)})
	if err := r.Close(); !errors.Is(err, ErrFileLimit) {
		t.Fatalf("close: %v", err)
	}
	if int64(len(s.bytes())) > o.fileBytes {
		t.Fatal("file exceeded cap")
	}
	records := readRecords(t, s.bytes())
	if records[len(records)-1].Status != "incomplete" || records[len(records)-1].Error == "" {
		t.Fatalf("end: %+v", records)
	}
}

func TestDiskFailureCannotClaimComplete(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(map[bool]string{false: "failure", true: "short"}[short], func(t *testing.T) {
			s := &faultSink{short: short}
			if !short {
				s.fail = errors.New("disk unavailable")
			}
			r := newRecorder("", s, defaultOptions())
			r.Record(Record{Kind: "feed", Data: []byte("data")})
			if err := r.Close(); err == nil {
				t.Fatal("disk failure suppressed")
			}
			if bytes.Contains(s.bytes(), []byte(`"status":"complete"`)) {
				t.Fatal("disk failure claimed complete")
			}
		})
	}
}

func TestCloseTimeoutStopsAdmissionAndWorkerEventuallyExits(t *testing.T) {
	unblock := make(chan struct{})
	s := &faultSink{block: unblock, started: make(chan struct{}, 1)}
	o := defaultOptions()
	o.closeTimeout = 20 * time.Millisecond
	r := newRecorder("", s, o)
	<-s.started
	if err := r.Close(); !errors.Is(err, ErrCloseTimeout) {
		t.Fatalf("close: %v", err)
	}
	before := time.Now()
	if err := r.Close(); !errors.Is(err, ErrCloseTimeout) {
		t.Fatalf("repeated timed-out close: %v", err)
	}
	if time.Since(before) >= o.closeTimeout/2 {
		close(unblock)
		t.Fatal("repeated close restarted the diagnostic wait")
	}
	r.Record(Record{Kind: "late"})
	close(unblock)
	select {
	case <-r.done:
	case <-time.After(time.Second):
		t.Fatal("worker did not exit")
	}
	if err := r.Close(); !errors.Is(err, ErrCloseTimeout) {
		t.Fatalf("idempotent close: %v", err)
	}
	records := readRecords(t, s.bytes())
	for _, record := range records {
		if record.Kind == "late" {
			t.Fatal("admission resumed")
		}
	}
	if records[len(records)-1].Status != "incomplete" {
		t.Fatal("timed out drain claimed complete")
	}
}

// A paused disk must retain the observed startup envelope, including in-flight data.
func TestBlockedStartupBurstSurvives(t *testing.T) {
	unblock := make(chan struct{})
	sink := &faultSink{block: unblock, started: make(chan struct{}, 1)}
	r := newRecorder("", sink, defaultOptions())
	<-sink.started
	payload := bytes.Repeat([]byte("x"), 1024)
	for i := 0; i < 4000; i++ {
		payload[0] = byte(i)
		r.Record(Record{Kind: "endpoint-feed", Data: payload})
	}
	close(unblock)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	records := readRecords(t, sink.bytes())
	if len(records) != 4002 {
		t.Fatalf("records=%d, want4002", len(records))
	}
	for i, record := range records[1:4001] {
		payload[0] = byte(i)
		if !bytes.Equal(record.Data, payload) {
			t.Fatalf("payload %d changed", i)
		}
	}
}

func TestStatusNilAndConfiguration(t *testing.T) {
	var disabled *Recorder
	if got := disabled.Status(); got != (Status{Phase: Disabled}) || disabled.Changes() != nil {
		t.Fatalf("disabled=%+v", got)
	}
	for _, limit := range []int64{0, 1 << 20, 1 << 40} {
		r, err := Open(t.TempDir(), Config{MaxBytes: limit})
		if err != nil {
			t.Fatal(err)
		}
		want := limit
		if want == 0 {
			want = maxFileBytes
		}
		if got := r.Status(); got.Phase != Recording || got.LimitBytes != want || got.Err != nil {
			t.Fatalf("open=%+v", got)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(filepath.Join(r.Dir(), "events.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if got := r.Status(); got.Phase != Closed || got.WrittenBytes != info.Size() {
			t.Fatalf("closed=%+v size=%d", got, info.Size())
		}
	}
	parent := filepath.Join(t.TempDir(), "not-created")
	for _, limit := range []int64{-1, 1, (1 << 20) - 1, (1 << 40) + 1} {
		if _, err := Open(parent, Config{MaxBytes: limit}); err == nil {
			t.Fatalf("accepted %d", limit)
		}
	}
	if _, err := os.Stat(parent); !os.IsNotExist(err) {
		t.Fatalf("invalid configuration created directory: %v", err)
	}
}

func awaitPhase(t *testing.T, r *Recorder, want Phase) Status {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for {
		if got := r.Status(); got.Phase == want {
			return got
		}
		select {
		case <-r.Changes():
		case <-deadline.C:
			t.Fatalf("status=%+v, want phase %v", r.Status(), want)
		}
	}
}

func TestStatusFailureWakesBeforeClose(t *testing.T) {
	for _, kind := range []string{"queue", "disk", "cap", "short"} {
		t.Run(kind, func(t *testing.T) {
			unblock := make(chan struct{})
			sink := &faultSink{block: unblock, started: make(chan struct{}, 1)}
			opts := defaultOptions()
			want := ErrQueueFull
			switch kind {
			case "queue":
				opts.queueRecords = 1
			case "disk":
				want = errors.New("disk failure")
				sink.fail = want
			case "cap":
				opts.fileBytes = finalReserve
				want = ErrFileLimit
			case "short":
				sink.short = true
				want = io.ErrShortWrite
			}
			r := newRecorder("", sink, opts)
			if kind != "cap" {
				<-sink.started
			}
			if kind != "cap" {
				select {
				case <-r.Changes():
				default:
				}
			}
			if kind == "queue" {
				r.Record(Record{Kind: "overflow"})
			}
			close(unblock)
			select {
			case <-r.Changes():
			case <-time.After(time.Second):
				t.Fatal("failure did not wake status consumer before Close")
			}
			got := awaitPhase(t, r, Failed)
			if !errors.Is(got.Err, want) {
				t.Fatalf("status=%+v want=%v", got, want)
			}
			if err := r.Close(); !errors.Is(err, want) {
				t.Fatalf("close=%v", err)
			}
			got = r.Status()
			if got.Phase != Failed || got.WrittenBytes != int64(len(sink.bytes())) {
				t.Fatalf("final=%+v bytes=%d", got, len(sink.bytes()))
			}
			// Changes stays open and coalesced after worker exit, avoiding select spin.
			select {
			case _, ok := <-r.Changes():
				if !ok {
					t.Fatal("changes closed")
				}
			default:
			}
			select {
			case <-r.Changes():
				t.Fatal("extra notification or closed changes")
			default:
			}
		})
	}
}

func TestStatusDrainingAndTimeoutRemainFailed(t *testing.T) {
	unblock := make(chan struct{})
	sink := &faultSink{block: unblock, started: make(chan struct{}, 1)}
	opts := defaultOptions()
	opts.closeTimeout = 100 * time.Millisecond
	r := newRecorder("", sink, opts)
	<-sink.started
	select {
	case <-r.Changes():
	default:
	}
	closed := make(chan error, 1)
	go func() { closed <- r.Close() }()
	got := awaitPhase(t, r, Draining)
	if got.WrittenBytes != 0 {
		t.Fatalf("offered bytes counted: %+v", got)
	}
	if err := <-closed; !errors.Is(err, ErrCloseTimeout) {
		t.Fatalf("close=%v", err)
	}
	if got := r.Status(); got.Phase != Failed || !errors.Is(got.Err, ErrCloseTimeout) {
		t.Fatalf("timeout=%+v", got)
	}
	close(unblock)
	<-r.done
	if got := r.Status(); got.Phase != Failed || !errors.Is(got.Err, ErrCloseTimeout) {
		t.Fatalf("late worker=%+v", got)
	}
}

func TestUsageNotificationsOnlyCrossPercentages(t *testing.T) {
	sink := &faultSink{}
	opts := defaultOptions()
	opts.fileBytes = 1 << 20
	r := newRecorder("", sink, opts)
	// Observe each completed write independently, consuming changes as we go.
	waitWritten := func(previous int64) {
		t.Helper()
		deadline := time.Now().Add(time.Second)
		for r.Status().WrittenBytes <= previous {
			if time.Now().After(deadline) {
				t.Fatal("writer stalled")
			}
			time.Sleep(time.Millisecond)
		}
	}
	waitWritten(0)
	select {
	case <-r.Changes():
	default:
	}
	notifications := 0
	for i := 0; i < 200; i++ {
		before := r.Status().WrittenBytes
		r.Record(Record{Kind: "feed", Data: bytes.Repeat([]byte("x"), 100)})
		waitWritten(before)
		select {
		case <-r.Changes():
			notifications++
		default:
		}
	}
	status := r.Status()
	if notifications == 0 || int64(notifications) > status.WrittenBytes*100/status.LimitBytes {
		t.Fatalf("notifications=%d status=%+v", notifications, status)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNearLimitRecordIncludesBlockedInflightCost(t *testing.T) {
	unblock := make(chan struct{})
	sink := &faultSink{block: unblock, started: make(chan struct{}, 1)}
	r := newRecorder("", sink, defaultOptions())
	<-sink.started
	record := Record{Kind: "endpoint-feed"}
	r.mu.Lock()
	remaining := r.opts.queueBytes - r.retained - record.retainedBytes()
	r.mu.Unlock()
	record.Data = bytes.Repeat([]byte("x"), remaining)
	r.Record(record)
	if got := r.Status(); got.Phase != Recording {
		t.Fatalf("near-limit rejected: %+v", got)
	}
	r.mu.Lock()
	retained, count := r.retained, r.count
	r.mu.Unlock()
	if retained != maxQueueBytes || count != 2 {
		t.Fatalf("retained=%d count=%d", retained, count)
	}
	r.Record(Record{Kind: "overflow"})
	if got := r.Status(); got.Phase != Failed || !errors.Is(got.Err, ErrQueueFull) {
		t.Fatalf("overflow=%+v", got)
	}
	r.mu.Lock()
	after := r.retained
	r.mu.Unlock()
	if after != retained {
		t.Fatalf("overflow retained extra bytes: %d -> %d", retained, after)
	}
	close(unblock)
	if err := r.Close(); !errors.Is(err, ErrQueueFull) {
		t.Fatal(err)
	}
	records := readRecords(t, sink.bytes())
	if len(records) != 3 || !bytes.Equal(records[1].Data, record.Data) || records[2].Status != "incomplete" {
		t.Fatal("near-limit admitted data lost")
	}
}
