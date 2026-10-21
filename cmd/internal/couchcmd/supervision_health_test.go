package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestSupervisionHealthWriteFailureChangesVisibleState(t *testing.T) {
	if got := factLogActive.advance(factLogWriteFailed); got == factLogActive {
		t.Fatal("failed sink write still reports an active logger")
	}
}

func TestSupervisionHealthLifecycleSequences(t *testing.T) {
	states := []factLogLifecycle{factLogOpening, factLogActive, factLogUnavailable, factLogWriteError, factLogClosed}
	events := []factLogEvent{factLogOpened, factLogOpenFailed, factLogWriteFailed, factLogWriteSucceeded, factLogClose}
	for _, initial := range states {
		for _, first := range events {
			for _, next := range events {
				closed := initial.advance(first).advance(factLogClose)
				if closed.advance(next) != factLogClosed {
					t.Fatal("closed owner acquired new authority")
				}
				if initial == factLogUnavailable && first != factLogClose && initial.advance(first) != factLogUnavailable {
					t.Fatal("failed open retried implicitly")
				}
			}
		}
	}
	if factLogWriteError.advance(factLogWriteSucceeded) != factLogActive || factLogActive.advance(factLogWriteFailed) != factLogWriteError {
		t.Fatal("independent-write recovery transition missing")
	}
	for state, want := range map[factLogLifecycle]string{factLogOpening: "opening", factLogActive: "working", factLogUnavailable: "unavailable", factLogWriteError: "failed-write", factLogClosed: "closed"} {
		if state.String() != want {
			t.Fatalf("%d projected as %q", state, state.String())
		}
	}
}

func TestSupervisionHealthOpenFailureAndOpeningClosure(t *testing.T) {
	for _, fail := range []bool{true, false} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			var opens atomic.Int32
			sink := &factSink{}
			log := newSupervisionFactLog(context.Background(), time.Now, func(context.Context) (io.WriteCloser, error) {
				opens.Add(1)
				close(entered)
				<-release
				if fail {
					return nil, errors.New("open denied\x1b[2J\n")
				}
				return sink, nil
			})
			t.Cleanup(log.Close)
			<-entered
			if h := log.health(); h.State != "opening" || h.Written != 0 {
				unblock()
				t.Fatalf("unresolved open reports completion: %+v", h)
			}
			if fail {
				unblock()
				<-log.done
				log.offer(supervisionFact{Event: "snapshot"})
				h := log.health()
				if h.State != "unavailable" || h.ErrorOperation != "open" || h.LastError != "open denied" || h.Dropped != 1 || h.Offered != 1 || h.LastErrorAt.IsZero() {
					t.Fatalf("open error lost: %+v", h)
				}
			} else {
				closed := make(chan struct{})
				go func() { log.Close(); close(closed) }()
				supervisionEventually(t, "close not admitted", func() bool { return log.health().State == "closed" })
				unblock()
				<-closed
				if log.health().State != "closed" {
					t.Fatal("late open reopened owner")
				}
			}
			if opens.Load() != 1 {
				t.Fatal("open was retried")
			}
		})
	}
}

func TestSupervisionHealthWriteOutcomesAndCountedReads(t *testing.T) {
	for _, count := range []int{16, 32} {
		for _, short := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/short=%v", count, short), func(t *testing.T) {
				sink := &factSink{fail: !short, short: short}
				log := testFactLog(t, sink)
				log.offer(supervisionFact{Event: "snapshot"})
				supervisionEventually(t, "failed write not observed", func() bool { return log.health().Dropped == 1 })
				h := log.health()
				if h.State != "failed-write" || h.Written != 0 || h.ErrorOperation != "write" || h.LastErrorAt.IsZero() || !h.LastWriteAt.IsZero() {
					t.Fatalf("write error lost: %+v", h)
				}
				if short && h.LastError != io.ErrShortWrite.Error() {
					t.Fatal(h.LastError)
				}
				for i := 0; i < count; i++ {
					copy := log.health()
					copy.State = "mutated caller copy"
				}
				sink.mu.Lock()
				if sink.attempts != 1 {
					t.Error("health reads retried a failed row")
				}
				sink.fail, sink.short = false, false
				sink.mu.Unlock()
				for i := 0; i < count; i++ {
					log.offer(supervisionFact{Event: "snapshot"})
				}
				supervisionEventually(t, "independent facts did not recover", func() bool { return log.health().Written == uint64(count) })
				h = log.health()
				if h.State != "working" || h.Offered != uint64(count+1) || h.Dropped != 1 || h.LastError == "" || h.LastWriteAt.IsZero() {
					t.Fatalf("acknowledged prefix or historical error lost: %+v", h)
				}
				sink.mu.Lock()
				attempts := sink.attempts
				sink.mu.Unlock()
				if attempts != count+1 {
					t.Fatalf("%d attempts for %d independent rows", attempts, count+1)
				}
				log.Close()
				log.offer(supervisionFact{Event: "snapshot"})
				if after := log.health(); after.State != "closed" || after.Offered != h.Offered {
					t.Fatalf("closed owner accepted a row: %+v", after)
				}
			})
		}
	}
}

func TestSupervisionHealthCloseDuringWrite(t *testing.T) {
	sink := &factSink{entered: make(chan struct{}), block: make(chan struct{})}
	log := testFactLog(t, sink)
	var once sync.Once
	unblock := func() { once.Do(func() { close(sink.block) }) }
	defer unblock()
	log.offer(supervisionFact{Event: "snapshot"})
	<-sink.entered
	closed := make(chan struct{})
	go func() { log.Close(); close(closed) }()
	supervisionEventually(t, "close not visible while sink held", func() bool { return log.health().State == "closed" })
	unblock()
	<-closed
	if h := log.health(); h.State != "closed" || h.Written != 1 {
		t.Fatalf("late complete write reopened owner or lost acknowledgment: %+v", h)
	}
}

func TestSupervisionHealthParentCancellationReportsStopped(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := newSupervisionFactLog(ctx, time.Now, func(context.Context) (io.WriteCloser, error) { return &factSink{}, nil })
	t.Cleanup(log.Close)
	supervisionEventually(t, "open not observed", func() bool { return log.health().State == "working" })
	cancel()
	<-log.done
	if h := log.health(); h.State != "closed" {
		t.Fatalf("stopped worker reports healthy: %+v", h)
	}
}

func TestSupervisionHealthExcludesFactPayload(t *testing.T) {
	log := testFactLog(t, &factSink{})
	secret := "distinctive private prompt"
	b := couchmessage.Binding{Slot: secret, Nonce: secret, Repository: secret}
	log.delivery(couchmessage.Message{From: b, To: b, ID: secret, Body: secret}, couchmessage.Receipt{Status: couchmessage.Submitted}, nil, false)
	supervisionEventually(t, "fact not acknowledged", func() bool { return log.health().Written == 1 })
	raw, err := json.Marshal(log.health())
	if err != nil || strings.Contains(string(raw), secret) {
		t.Fatalf("health includes payload: %s %v", raw, err)
	}
}

func FuzzSupervisionHealthErrorBound(f *testing.F) {
	for _, seed := range []string{"", "open denied\x1b[2J\r\n", strings.Repeat("界", 400), "\xff\xfe\x9b"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		log := &supervisionFactLog{now: time.Now}
		log.recordError("write", errors.New(raw))
		got := log.health().LastError
		if len(got) > 512 || !utf8.ValidString(got) || got == "" || strings.ContainsFunc(got, unicode.IsControl) {
			t.Fatalf("unsafe or unbounded error: %q", got)
		}
	})
}
