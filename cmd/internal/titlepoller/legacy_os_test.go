package titlepoller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"strings"
	"testing"
	"time"
)

func TestObserveLegacyTitlesUsesOneCompleteSnapshot(t *testing.T) {
	calls, resources := 0, 0
	sample, err := captureLegacyTitles(context.Background(), "darwin", func(context.Context) ([]procutil.Observation, error) {
		calls++
		return []procutil.Observation{legacyTestTitle(42, "work"), legacyTestWrapper("/data/scrollback-other-codex.raw")}, nil
	}, func(context.Context, procutil.Process) (LegacyResource, error) {
		resources++
		return LegacyResource{RSSBytes: 123, RSSKnown: true}, nil
	}, func() time.Time { return time.Unix(300, 0) })
	if err != nil || !sample.Complete || calls != 1 || resources != 1 {
		t.Fatalf("snapshot=%+v err=%v calls=%d resources=%d", sample, err, calls, resources)
	}
	var out bytes.Buffer
	if err := WriteLegacyTitleReport(&out, sample, legacyTestPolicy()); err != nil {
		t.Fatal(err)
	}
	var report LegacyReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Titles) != 1 || report.Titles[0].Disposition != LegacyEligible || !report.Titles[0].RSSKnown || report.Titles[0].RSSBytes != 123 || !report.Titles[0].AgeKnown || report.Titles[0].AgeSeconds <= 199 || report.Cutoff.Start != "200.0" {
		t.Fatalf("report=%+v", report)
	}
}
func TestObserveLegacyTitlesFailsClosedWithoutCompleteTable(t *testing.T) {
	for _, platform := range []string{"linux", "windows"} {
		calls := 0
		s, err := captureLegacyTitles(context.Background(), platform, func(context.Context) ([]procutil.Observation, error) { calls++; return nil, nil }, nil, time.Now)
		if !errors.Is(err, ErrLegacySampleUnsupported) || s.Complete || calls != 0 {
			t.Fatalf("platform=%s sample=%+v err=%v calls=%d", platform, s, err, calls)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s, err := captureLegacyTitles(ctx, "darwin", func(context.Context) ([]procutil.Observation, error) {
		t.Fatal("canceled collector read table")
		return nil, nil
	}, nil, time.Now)
	if !errors.Is(err, context.Canceled) || s.Complete {
		t.Fatalf("sample=%+v err=%v", s, err)
	}
}
func TestLegacyTitleReportWriterFailure(t *testing.T) {
	if err := WriteLegacyTitleReport(legacyFailWriter{}, legacyTestSample(), legacyTestPolicy()); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("writer error=%v", err)
	}
}

type legacyFailWriter struct{}

func (legacyFailWriter) Write([]byte) (int, error) { return 0, errors.New("closed report") }
func TestLegacyCleanupCancellationAndInvalidPIDHaveNoSignals(t *testing.T) {
	for _, pid := range []int{-1, 0, 1, 1 << 31} {
		fake := &legacyCleanupFake{identities: []string{"100.1"}}
		CleanupLegacyTitle(context.Background(), LegacyTitleDecision{PID: pid, Start: "100.1", Disposition: LegacyEligible}, fake, time.Millisecond)
		if len(fake.events) != 0 {
			t.Fatalf("invalid pid=%d effects=%v", pid, fake.events)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fake := &legacyCleanupFake{identities: []string{"100.1"}}
	CleanupLegacyTitle(ctx, LegacyTitleDecision{PID: 42, Start: "100.1", Disposition: LegacyEligible}, fake, time.Millisecond)
	if len(fake.events) != 0 {
		t.Fatalf("canceled effects=%v", fake.events)
	}
}
