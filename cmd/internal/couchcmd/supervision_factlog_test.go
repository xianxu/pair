package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type factSink struct {
	mu sync.Mutex
	bytes.Buffer
	block, entered chan struct{}
	once           sync.Once
	fail           bool
	short          bool
	attempts       int
	closed         bool
}

func (s *factSink) Write(p []byte) (int, error) {
	if s.entered != nil {
		s.once.Do(func() { close(s.entered) })
	}
	if s.block != nil {
		<-s.block
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts++
	if s.fail {
		return 0, errors.New("disk failed")
	}
	if s.short {
		return s.Buffer.Write(p[:len(p)/2])
	}
	return s.Buffer.Write(p)
}
func (s *factSink) Close() error { s.mu.Lock(); defer s.mu.Unlock(); s.closed = true; return nil }
func (s *factSink) rows(t *testing.T) []supervisionFact {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	var rows []supervisionFact
	for _, line := range bytes.Split(bytes.TrimSpace(s.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var f supervisionFact
		if err := json.Unmarshal(line, &f); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, f)
	}
	return rows
}
func testFactLog(t *testing.T, s *factSink) *supervisionFactLog {
	t.Helper()
	l := newSupervisionFactLog(context.Background(), time.Now, func(context.Context) (io.WriteCloser, error) { return s, nil })
	t.Cleanup(l.Close)
	return l
}
func TestSupervisionFactHints(t *testing.T) {
	for _, c := range []struct{ body, want string }{{"Landed #530.", "landed"}, {"#543: merged PR #324", "landed"}, {"Waiting for operator smoke", "waiting-for-operator"}, {"Progress: tests running", "progress"}, {"secret /Users/operator/file\x1b]9;alert\a", "unknown"}, {strings.Repeat("x", 10000) + " landed", "unknown"}} {
		if got := factKindHint(c.body); got != c.want {
			t.Errorf("hint=%s want %s", got, c.want)
		}
	}
}
func TestSupervisionFactLifecycle(t *testing.T) {
	for _, prefix := range [][]factLogEvent{{}, {factLogOpened}, {factLogOpenFailed}, {factLogOpened, factLogWriteFailed}} {
		state := factLogOpening
		for _, e := range prefix {
			state = state.advance(e)
		}
		state = state.advance(factLogClose)
		for _, e := range []factLogEvent{factLogOpened, factLogOpenFailed, factLogWriteFailed, factLogClose} {
			if got := state.advance(e); got != factLogClosed {
				t.Fatal("late event reopened closed log")
			}
		}
	}
}
func TestSupervisionFactLogBoundedNonblockingAndGaps(t *testing.T) {
	sink := &factSink{block: make(chan struct{}), entered: make(chan struct{})}
	l := testFactLog(t, sink)
	var release sync.Once
	unblock := func() { release.Do(func() { close(sink.block) }) }
	defer unblock()
	l.offer(supervisionFact{Event: "snapshot", Slots: []supervisionFactSlot{}})
	select {
	case <-sink.entered:
	case <-time.After(time.Second):
		t.Fatal("writer did not start")
	}
	offered := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			l.offer(supervisionFact{Event: "snapshot", Slots: []supervisionFactSlot{}})
		}
		close(offered)
	}()
	select {
	case <-offered:
	case <-time.After(time.Second):
		t.Fatal("producer blocked behind sink")
	}
	l.mu.Lock()
	dropped := l.dropped
	l.mu.Unlock()
	if dropped == 0 {
		t.Fatal("unbounded buffer")
	}
	unblock()
	supervisionEventually(t, "buffer not drained", func() bool { return len(l.rows) == 0 })
	l.offer(supervisionFact{Event: "snapshot", Slots: []supervisionFactSlot{}})
	supervisionEventually(t, "gap not observable", func() bool {
		for _, r := range sink.rows(t) {
			if r.Dropped > 0 {
				return true
			}
		}
		return false
	})
	l.Close()
	l.offer(supervisionFact{Event: "snapshot"})
	sink.mu.Lock()
	closed := sink.closed
	sink.mu.Unlock()
	if !closed {
		t.Fatal("sink not closed")
	}
}
func TestSupervisionFactLogPrivacyAndClock(t *testing.T) {
	sink := &factSink{}
	now := time.Now()
	l := newSupervisionFactLog(context.Background(), func() time.Time { return now }, func(context.Context) (io.WriteCloser, error) { return sink, nil })
	defer l.Close()
	b := couchmessage.Binding{Slot: "secret:7", Nonce: "secret nonce", Repository: "/secret/path"}
	subject := l.subject(b)
	if len(subject) != 32 || subject == l.incarnation(b) {
		t.Fatal("bad opaque domain identities")
	}
	l.offer(supervisionFact{Event: "session_write", Subject: subject, Incarnation: l.incarnation(b)})
	l.mu.Lock()
	now = now.Add(-time.Hour)
	l.mu.Unlock()
	l.offer(supervisionFact{Event: "session_write", Subject: subject, Incarnation: l.incarnation(b)})
	supervisionEventually(t, "rows not written", func() bool { return len(sink.rows(t)) == 2 })
	rows := sink.rows(t)
	if rows[1].ElapsedMS < rows[0].ElapsedMS || rows[1].Seq != rows[0].Seq+1 {
		t.Fatal("ordering regressed")
	}
	sink.mu.Lock()
	raw := sink.String()
	sink.mu.Unlock()
	if strings.Contains(raw, "secret") {
		t.Fatal("content leaked")
	}
}
func TestSupervisionFactLogOpenFailureAndCloseWhileOpening(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	sink := &factSink{}
	l := newSupervisionFactLog(context.Background(), time.Now, func(context.Context) (io.WriteCloser, error) { close(started); <-release; return sink, nil })
	<-started
	done := make(chan struct{})
	go func() { l.Close(); close(done) }()
	supervisionEventually(t, "close not admitted", func() bool { l.mu.Lock(); defer l.mu.Unlock(); return l.state == factLogClosed })
	l.offer(supervisionFact{Event: "snapshot"})
	close(release)
	<-done
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if !sink.closed || sink.Len() != 0 {
		t.Fatal("late open was not closed without writes")
	}
}

func TestSupervisionFactLogUnavailableAndWriteFailure(t *testing.T) {
	l := newSupervisionFactLog(context.Background(), time.Now, func(context.Context) (io.WriteCloser, error) { return nil, errors.New("open failed") })
	defer l.Close()
	<-l.done
	l.offer(supervisionFact{Event: "snapshot"})
	l.mu.Lock()
	state, dropped := l.state, l.dropped
	l.mu.Unlock()
	if state != factLogUnavailable || dropped == 0 {
		t.Fatal("unavailable writer did not account for dropped fact")
	}
	sink := &factSink{fail: true}
	active := testFactLog(t, sink)
	active.offer(supervisionFact{Event: "snapshot"})
	supervisionEventually(t, "write failure invisible", func() bool { active.mu.Lock(); defer active.mu.Unlock(); return active.dropped > 0 })
	sink.mu.Lock()
	sink.fail = false
	sink.mu.Unlock()
	active.offer(supervisionFact{Event: "snapshot"})
	supervisionEventually(t, "later independent event did not recover", func() bool { return len(sink.rows(t)) == 1 })
	if row := sink.rows(t)[0]; row.Seq != 2 || row.Dropped != 1 {
		t.Fatalf("lost gap: %+v", row)
	}
}

func TestSupervisionFactSnapshotBoundAndUnknowns(t *testing.T) {
	yes := true
	quiet := int64(1<<63 - 1)
	submission := uint64(1<<64 - 1)
	slots := make([]supervisionFactSlot, 128)
	for i := range slots {
		slots[i] = supervisionFactSlot{Subject: strings.Repeat("a", 32), Incarnation: strings.Repeat("b", 32), Known: true, Resting: &yes, Settled: &yes, Picker: &yes, Pending: &yes, QuietMS: &quiet, Submission: &submission}
	}
	_, err := encodeFact(supervisionFact{Event: "snapshot", Slots: slots})
	if err != nil {
		t.Fatalf("maximum supported snapshot does not fit: %v", err)
	}
	raw, err := encodeFact(supervisionFact{Event: "snapshot", Slots: []supervisionFactSlot{{Subject: strings.Repeat("a", 32), Incarnation: strings.Repeat("b", 32)}}})
	if err != nil || !bytes.Contains(raw, []byte(`"quiet_ms":null`)) || !bytes.Contains(raw, []byte(`"settled":null`)) {
		t.Fatalf("unknown facts collapsed: %s %v", raw, err)
	}
	if _, err := encodeFact(supervisionFact{Event: strings.Repeat("x", maxFactBytes)}); err == nil {
		t.Fatal("oversized row accepted")
	}
}
