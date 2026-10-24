package recorder

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type recorderReports struct {
	mu             sync.Mutex
	logs, warnings []string
}

func (r *recorderReports) logf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, fmt.Sprintf(format, args...))
}

func (r *recorderReports) warnf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.warnings = append(r.warnings, fmt.Sprintf(format, args...))
}

func (r *recorderReports) snapshot() ([]string, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.logs...), append([]string(nil), r.warnings...)
}

type failingRecorderSink struct {
	putErr, closeErr error
}

func (s failingRecorderSink) Put(rec.Record) error { return s.putErr }
func (s failingRecorderSink) Close() error         { return s.closeErr }

func TestRecorderActionableWarningRouting(t *testing.T) {
	for _, test := range []struct {
		name                      string
		openErr, putErr, closeErr error
		missingPage               bool
		want                      string
	}{
		{name: "open", openErr: errors.New("disk unavailable"), want: "recorder: open: disk unavailable"},
		{name: "write", putErr: errors.New("disk full"), want: "recorder: write: disk full"},
		{name: "snapshot", missingPage: true, want: "recorder: no snapshot"},
		{name: "close", closeErr: errors.New("flush failed"), want: "recorder: close: flush failed"},
	} {
		for _, legacy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/legacy=%t", test.name, legacy), func(t *testing.T) {
				reports := &recorderReports{}
				h := &harness{t: t, fc: newFakeChrome()}
				if !test.missingPage {
					h.page("A", "http://a/", "L1")
				}
				opts := Options{ID: "warning", Call: h.fc, ShotEvery: -1, Logf: reports.logf,
					Out: func() (Opened, error) {
						return Opened{Sink: failingRecorderSink{test.putErr, test.closeErr}}, test.openErr
					}}
				if !legacy {
					opts.Warnf = reports.warnf
				}
				h.r = Start(opts)
				t.Cleanup(func() { _ = h.r.Close() })
				h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
				h.settle()
				_ = h.r.Close()
				logs, warnings := reports.snapshot()
				got := warnings
				if legacy {
					got = logs
					if len(warnings) != 0 {
						t.Fatalf("legacy warnings = %v", warnings)
					}
				} else if len(logs) != 0 {
					t.Fatalf("warning duplicated in log: %v", logs)
				}
				if len(got) != 1 || !strings.Contains(got[0], test.want) {
					t.Fatalf("reports = %v, want one containing %q", got, test.want)
				}
			})
		}
	}
}

func TestRecorderSegmentFailureWarns(t *testing.T) {
	fc := newFakeChrome()
	h := &harness{t: t, fc: fc}
	h.page("A", "http://a/", "L1")
	reports := &recorderReports{}
	r := &Recorder{opts: Options{Call: fc, Logf: reports.logf, Warnf: reports.warnf},
		ctx: context.Background(), m: newMirror(), top: "A", t0: time.Now(),
		loader: map[string]string{}, frameIDs: map[string]string{}, snapSeq: map[string]uint64{}}
	// The CDP page exists, but its mirror session was lost: this is a segment
	// failure rather than a failed document call or recovered overflow.
	r.snapshotTop(time.Now(), rec.SegLive)
	logs, warnings := reports.snapshot()
	if len(logs) != 0 || len(warnings) != 1 || !strings.Contains(warnings[0], "not the cast page") {
		t.Fatalf("logs = %v, warnings = %v", logs, warnings)
	}
}
