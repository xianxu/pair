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
