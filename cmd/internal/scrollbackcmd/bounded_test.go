package scrollbackcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

func TestRecoveryReplayContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RenderBoundedLinesContext(ctx, "/missing", "/missing", 100); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled replay = %v", err)
	}
	// Interrupt while feeding a recording that repeatedly redraws one screen,
	// so history capacity cannot be the reason replay stops.
	counted := &cancelDuringReplay{Context: context.Background(), remaining: 4}
	raw := []byte(strings.Repeat("working\r", 8192))
	_, err := replayDataWithLimitsContext(counted, raw, nil, true, 101, 100, boundedReplayLimits(100))
	if !errors.Is(err, context.Canceled) || counted.remaining > 0 {
		t.Fatalf("mid-replay cancellation = %v, checks remaining %d", err, counted.remaining)
	}
}

type cancelDuringReplay struct {
	context.Context
	remaining int
}

func (c *cancelDuringReplay) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		return context.Canceled
	}
	return nil
}

func TestOwnedCaptureLeaseProtectsReplayAndReleases(t *testing.T) {
	for _, failReplay := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure=%t", failReplay), func(t *testing.T) {
			ctx := context.Background()
			c, err := storagegc.NewCoordinator(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			now := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
			c.Now = func() time.Time { return now }
			scope, tag := "0123456789abcdef", "couch-0123456789abcdef"
			owner, err := artifactpath.NewStorageOwner(c.Root, scope, tag)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(owner.Directory(), 0700); err != nil {
				t.Fatal(err)
			}
			if err := c.Initialize(ctx, owner); err != nil {
				t.Fatal(err)
			}
			if err := c.CompleteMigration(ctx, nil); err != nil {
				t.Fatal(err)
			}
			paths, err := artifactpath.ResolveScoped(owner.Directory(), tag)
			if err != nil {
				t.Fatal(err)
			}
			selected, _ := paths.ParkedScrollbackArtifacts("20260913T010101")
			other, _ := paths.ParkedScrollbackArtifacts("20260913T020202")
			for _, capture := range []artifactpath.ParkedScrollbackArtifactSet{selected, other} {
				events := `{"type":"resize","cols":20,"rows":2}`
				if failReplay && capture.Raw == selected.Raw {
					events = `{"type":"resize","cols":1001,"rows":2}`
				}
				for path, body := range map[string]string{capture.Raw: "selected history", capture.Events: events} {
					if err := os.WriteFile(path, []byte(body), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err := c.PublishCaptureMetadata(ctx, owner, artifactpath.ParkedCapture{Raw: capture.Raw, Events: capture.Events, Metadata: capture.Metadata}, now.Add(-storagegc.CaptureRetentionPeriod-time.Hour)); err != nil {
					t.Fatal(err)
				}
			}
			gc := &storagegc.Collector{Coordinator: c, Agents: []string{"claude"}, Legacy: func(context.Context, artifactpath.StorageOwner) (storagegc.Liveness, error) {
				return storagegc.ProcessDead, nil
			}}
			collect := func(want int) {
				t.Helper()
				report, err := gc.Apply(ctx, 100)
				if err != nil || report.Collected != want {
					t.Fatalf("collection: %+v, %v; want %d collected", report, err, want)
				}
			}
			assertProtected := func() {
				t.Helper()
				state, err := c.ReadOwner(owner)
				if err != nil || len(state.Processes) != 1 || state.Processes[0].Target != selected.Raw || state.Processes[0].Role != "scrollback-reader" {
					t.Fatalf("replay lacks exact selected capture lease: %+v, %v", state, err)
				}
				for _, path := range []string{selected.Raw, selected.Events, selected.Metadata} {
					if _, err := os.Stat(path); err != nil {
						t.Fatalf("selected capture removed during replay: %s: %v", path, err)
					}
				}
			}
			original := replayOwnedCapture
			t.Cleanup(func() { replayOwnedCapture = original })
			called := false
			replayOwnedCapture = func(raw, events string, maxLines int) ([]string, error) {
				called = true
				if raw != selected.Raw || events != selected.Events || maxLines != 0 {
					t.Fatalf("wrong replay selection: %s %s %d", raw, events, maxLines)
				}
				assertProtected()
				collect(1) // The other old capture is collectible while this one is leased.
				assertProtected()
				if _, err := os.Stat(other.Raw); !os.IsNotExist(err) {
					t.Fatalf("unselected capture retained: %v", err)
				}
				lines, err := original(raw, events, maxLines)
				collect(0)
				assertProtected()
				return lines, err
			}
			lines, err := RenderOwnedCaptureLines(c.Root, scope, tag, "claude", "20260913T010101", 0)
			if !called || (err != nil) != failReplay {
				t.Fatalf("replay called=%t lines=%q error=%v", called, lines, err)
			}
			if failReplay {
				if !strings.Contains(err.Error(), "resize 1001x2") {
					t.Fatalf("unexpected replay failure: %v", err)
				}
			} else if strings.Join(lines, "\n") != "selected history" {
				t.Fatalf("wrong replay: %q", lines)
			}
			state, err := c.ReadOwner(owner)
			if err != nil || len(state.Processes) != 0 {
				t.Fatalf("replay leaked its lease: %+v, %v", state, err)
			}
			collect(1)
			for _, path := range []string{selected.Raw, selected.Events, selected.Metadata} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("capture retained after replay returned: %s: %v", path, err)
				}
			}
		})
	}
}

func boundedFixture(t *testing.T, raw, events string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	r, e := filepath.Join(dir, "capture.raw"), filepath.Join(dir, "capture.events.jsonl")
	for p, b := range map[string]string{r: raw, e: events} {
		if err := os.WriteFile(p, []byte(b), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return r, e
}
func TestBoundedMatchesReplay(t *testing.T) {
	r, e := boundedFixture(t, "\x1b[31mfirst\x1b[0m\r\nsecond\r\nthird\r\nfourth", `{"type":"resize","cols":20,"rows":2}`)
	for _, n := range []int{0, -1, 1, 100} {
		want, err := RenderLines(r, e, n)
		if err != nil {
			t.Fatal(err)
		}
		got, err := RenderBoundedLines(r, e, n)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("max %d: %q %v, want %q", n, got, err, want)
		}
	}
}
func TestBoundedRejectsUnsafeSources(t *testing.T) {
	for _, kind := range []string{"symlink", "fifo", "directory", "raw-size", "events-size", "missing-events"} {
		t.Run(kind, func(t *testing.T) {
			r, e := boundedFixture(t, "safe", "")
			source := r
			switch kind {
			case "symlink":
				source = r + ".link"
				if err := os.Symlink(r, source); err != nil {
					t.Fatal(err)
				}
				r = source
			case "fifo":
				source = r + ".fifo"
				if err := unix.Mkfifo(source, 0600); err != nil {
					t.Fatal(err)
				}
				r = source
			case "directory":
				r = filepath.Dir(r)
				source = r
			case "raw-size":
				if err := os.Truncate(r, 64<<20+1); err != nil {
					t.Fatal(err)
				}
			case "events-size":
				source = e
				if err := os.Truncate(e, 8<<20+1); err != nil {
					t.Fatal(err)
				}
			case "missing-events":
				e += ".missing"
				source = e
			}
			got, err := RenderBoundedLines(r, e, 0)
			if err == nil || len(got) != 0 || !strings.Contains(err.Error(), source) {
				t.Fatalf("got %q, %v; want rejection naming %s", got, err, source)
			}
		})
	}
}
func TestBoundedRejectsResizeBeforeReplay(t *testing.T) {
	for _, size := range [][2]int{{1001, 1}, {1, 1001}, {0, 5}, {5, -1}} {
		r, e := boundedFixture(t, "x", fmt.Sprintf(`{"type":"resize","cols":20,"rows":2}`+"\n"+`{"type":"resize","offset":1,"cols":%d,"rows":%d}`, size[0], size[1]))
		if _, err := RenderBoundedLines(r, e, 0); err == nil || !strings.Contains(err.Error(), e) {
			t.Fatalf("size %v: %v", size, err)
		}
	}
}
func TestBoundedRowsRejectInsteadOfReturningTail(t *testing.T) {
	// A small injected envelope exercises overflow without a large fixture.
	for _, suffix := range []string{""} {
		r, e := boundedFixture(t, "first\r\nsecond\r\nthird\r\nfourth"+suffix, `{"type":"resize","cols":20,"rows":1}`)
		got, err := renderBoundedLines(r, e, 0, 2)
		if err == nil || len(got) != 0 || !strings.Contains(err.Error(), r) || !strings.Contains(err.Error(), e) {
			t.Fatalf("got %q, %v", got, err)
		}
	}
}
func TestOwnedCaptureSelectsExactToken(t *testing.T) {
	root := t.TempDir()
	scope, tag := "0123456789abcdef", "couch-0123456789abcdef"
	p, err := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: scope, Tag: tag})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(p.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	live, _ := p.ScrollbackArtifacts("claude")
	old, _ := p.ParkedScrollbackArtifacts("20260913T010101")
	for path, body := range map[string]string{live.Raw: "current", live.Events: "", old.Raw: "previous", old.Events: ""} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for token, want := range map[string]string{"": "current", "20260913T010101": "previous"} {
		got, err := RenderOwnedCaptureLines(root, scope, tag, "claude", token, 0)
		if err != nil || strings.Join(got, "\n") != want {
			t.Fatalf("token %q got %q %v", token, got, err)
		}
	}
	if _, err := RenderOwnedCaptureLines(root, scope, tag, "claude", "20260913T020202", 0); err == nil {
		t.Fatal("missing previous fell back")
	}
}

func TestBoundedBlankHistoryTail(t *testing.T) {
	r, e := boundedFixture(t, strings.Repeat("\r\n", 10), `{"type":"resize","cols":20,"rows":2}`)
	want, err := RenderLines(r, e, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RenderBoundedLines(r, e, 1)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q %v, want %q", got, err, want)
	}
}

func TestBoundedRowsAtLimitAreComplete(t *testing.T) {
	r, e := boundedFixture(t, "first\r\nsecond", `{"type":"resize","cols":20,"rows":1}`)
	got, err := renderBoundedLines(r, e, 0, 2)
	if err != nil || !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestBoundedFullRetainsTwoThousandWideLines(t *testing.T) {
	var raw strings.Builder
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&raw, "%04d%s\r\n", i, strings.Repeat("x", 76))
	}
	r, e := boundedFixture(t, raw.String(), `{"type":"resize","cols":80,"rows":24}`)
	got, err := RenderBoundedLines(r, e, 0)
	if err != nil || len(got) != 2000 || !strings.HasPrefix(got[0], "0000") {
		t.Fatalf("got %d lines, error %v; want all 2000", len(got), err)
	}
}

func TestBoundedRejectsScreenArea(t *testing.T) {
	for _, initial := range []bool{true, false} {
		t.Run(fmt.Sprintf("initial=%t", initial), func(t *testing.T) {
			events := `{"type":"resize","cols":1000,"rows":1000}`
			if !initial {
				events = `{"type":"resize","cols":80,"rows":24}` + "\n" + `{"type":"resize","offset":1,"cols":1000,"rows":1000}`
			}
			r, e := boundedFixture(t, "text", events)
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("untrusted dimensions panicked: %v", p)
				}
			}()
			got, err := RenderBoundedLines(r, e, 0)
			if err == nil || len(got) != 0 || !strings.Contains(err.Error(), e) {
				t.Fatalf("got %q, %v; want visible area refusal naming events", got, err)
			}
		})
	}
}

func TestBoundedCapacityLossIsVisible(t *testing.T) {
	for _, kind := range []string{"cells", "bytes", "oversized-row", "oversized-byte-row", "rows"} {
		for _, clearAfter := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/clear=%t", kind, clearAfter), func(t *testing.T) {
				limits := boundedReplayLimits(100)
				switch kind {
				case "cells":
					limits.HistoryCells = 6 // two four-cell rows overflow cells alone
				case "bytes":
					limits.HistoryBytes = 600 // one row fits, two overflow bytes alone
				case "oversized-byte-row":
					limits.HistoryBytes = 1
				case "oversized-row":
					limits.HistoryCells = 3
				case "rows":
					limits.HistoryLines = 1
				}
				if kind == "cells" || kind == "bytes" {
					prefix, err := replayDataWithLimits([]byte("aaaa\r\n"), []scrollbackEvent{{Type: "resize", Cols: 10, Rows: 1}}, true, limits.HistoryLines, 100, limits)
					if err != nil || !reflect.DeepEqual(prefix.lines, []string{"aaaa"}) {
						t.Fatalf("a single row must fit: %q %v", prefix.lines, err)
					}
				}
				raw := "aaaa\r\nbbbb\r\ncccc"
				if clearAfter {
					raw += "\x1b[3J"
				}
				got, err := replayDataWithLimits([]byte(raw), []scrollbackEvent{{Type: "resize", Cols: 10, Rows: 1}}, true, limits.HistoryLines, 100, limits)
				if err == nil || len(got.lines) != 0 || !strings.Contains(err.Error(), "capacity") {
					t.Fatalf("got %q, %v; want visible capacity refusal", got.lines, err)
				}
			})
		}
	}
}

func TestBoundedIntentionalHistoryClear(t *testing.T) {
	r, e := boundedFixture(t, "before\r\nold\r\n\x1b[3Jafter", `{"type":"resize","cols":20,"rows":1}`)
	got, err := RenderBoundedLines(r, e, 0)
	if err != nil || !reflect.DeepEqual(got, []string{"after"}) {
		t.Fatalf("got %q %v", got, err)
	}
}

func TestReplayCheckedGeometryErrors(t *testing.T) {
	for _, initial := range []bool{true, false} {
		t.Run(fmt.Sprintf("initial=%t", initial), func(t *testing.T) {
			limits := boundedReplayLimits(100)
			limits.MaxCells = 100
			events := []scrollbackEvent{{Type: "resize", Cols: 11, Rows: 10}}
			if !initial {
				events = append([]scrollbackEvent{{Type: "resize", Cols: 10, Rows: 10}}, events...)
			}
			got, err := replayDataWithLimits(nil, events, true, 101, 100, limits)
			if err == nil || len(got.lines) != 0 || !strings.Contains(err.Error(), "resize 11x10") {
				t.Fatalf("got %q %v", got.lines, err)
			}
		})
	}
}

func TestBoundedRejectsMalformedResizeEvidence(t *testing.T) {
	for _, events := range []string{
		`{"type":"resize","cols":20,"rows":1`,
		`{"type":"resize","cols":9999999999999999999999999,"rows":1}`,
		`{"type":"resize","cols":20,"rows":1,"offset":-1}`,
		`{"type":"resize","cols":20,"rows":1,"offset":6}`,
		`{"type":"resize","cols":20,"rows":1,"offset":3}` + "\n" + `{"type":"resize","cols":10,"rows":1,"offset":1}`,
	} {
		t.Run(events, func(t *testing.T) {
			r, e := boundedFixture(t, "hello", events)
			got, err := RenderBoundedLines(r, e, 0)
			if err == nil || len(got) != 0 || !strings.Contains(err.Error(), e) {
				t.Fatalf("got %q %v", got, err)
			}
			if _, err := RenderLines(r, e, 0); err != nil {
				t.Fatalf("ordinary viewer lost tolerant decoding: %v", err)
			}
		})
	}
}

func TestBoundedAcceptsScreenAreaBoundary(t *testing.T) {
	for _, initial := range []bool{true, false} {
		events := `{"type":"resize","cols":512,"rows":512}`
		if !initial {
			events = `{"type":"resize","cols":80,"rows":24}` + "\n" + `{"type":"resize","cols":512,"rows":512}`
		}
		r, e := boundedFixture(t, "accepted", events)
		got, err := RenderBoundedLines(r, e, 0)
		if err != nil || !reflect.DeepEqual(got, []string{"accepted"}) {
			t.Fatalf("initial=%t: %q %v", initial, got, err)
		}
	}
}

func TestBoundedFeedStopsAfterCapacityLoss(t *testing.T) {
	limits := boundedReplayLimits(1)
	em, err := vt.NewEmulatorWithLimits(10, 1, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer em.Close()
	// A later clear/title would hide evidence of continuing through the whole
	// recording. Stop soon after overflow instead of repeatedly evicting history.
	raw := strings.Repeat("x\r\n", 2000) + "\x1b[3J"
	_, err = feedSegmentsChecked(em, []byte(raw), nil, true)
	if err == nil || em.HistorySnapshot().ClearEpoch != 0 {
		t.Fatalf("feed did not stop at capacity loss: %v", err)
	}
}

func TestBoundedResizeCapacityLoss(t *testing.T) {
	limits := boundedReplayLimits(100)
	limits.HistoryCells = 3
	raw := []byte("abcdefgh")
	events := []scrollbackEvent{{Type: "resize", Cols: 10, Rows: 1}, {Type: "resize", Offset: int64(len(raw)), Cols: 2, Rows: 1}}
	got, err := replayDataWithLimits(raw, events, true, 101, 100, limits)
	if err == nil || len(got.lines) != 0 || !strings.Contains(err.Error(), "capacity") {
		t.Fatalf("resize lost history without refusal: %q %v", got.lines, err)
	}
}
