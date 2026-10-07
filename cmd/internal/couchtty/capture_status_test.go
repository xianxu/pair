package couchtty

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/terminalcapture"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestCaptureBadgeFirstPaintAndIdleFailure(t *testing.T) {
	for _, panel := range []bool{false, true} {
		name := "actor"
		if panel {
			name = "switcher"
		}
		t.Run(name, func(t *testing.T) {
			r, err := terminalcapture.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = r.Close() })
			f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
				c.SetCapture(r)
				if panel {
					c.focus = FocusPanel()
				}
				// Exclude unrelated periodic repaint opportunities.
				c.slotGitInterval, c.activityInterval = time.Hour, time.Hour
			})
			waitFor(t, "capture badge on first frame", func() bool {
				return strings.HasPrefix(f.screen.row(24), "REC 0%")
			})
			// Three MiB becomes four MiB of JSON base64, crossing one percent
			// of the default 256 MiB disk budget. No terminal event drives this.
			r.Record(terminalcapture.Record{Kind: "usage", Data: make([]byte, 3<<20)})
			waitUpTo(t, time.Second, "idle capture usage repaint", func() bool {
				return strings.HasPrefix(f.screen.row(24), "REC 1%")
			})
			r.Record(terminalcapture.Record{Kind: "oversized", Data: make([]byte, 9<<20)})
			waitUpTo(t, time.Second, "idle capture failure repaint", func() bool {
				return strings.HasPrefix(f.screen.row(24), "REC STOP:queue")
			})
		})
	}
}

func TestCaptureStatusBadgePriorityAndClickSpans(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status terminalcapture.Status
		badge  string
	}{
		{"disabled", terminalcapture.Status{}, ""},
		{"recording", terminalcapture.Status{Phase: terminalcapture.Recording, WrittenBytes: 25, LimitBytes: 100}, "REC 25%"},
		{"draining", terminalcapture.Status{Phase: terminalcapture.Draining}, "REC draining"},
		{"closed", terminalcapture.Status{Phase: terminalcapture.Closed}, "REC closed"},
		{"queue", terminalcapture.Status{Phase: terminalcapture.Failed, Err: errors.Join(errors.New("private path"), terminalcapture.ErrQueueFull)}, "REC STOP:queue"},
		{"full", terminalcapture.Status{Phase: terminalcapture.Failed, Err: terminalcapture.ErrFileLimit}, "REC STOP:full"},
		{"io", terminalcapture.Status{Phase: terminalcapture.Failed, Err: errors.New("private\x1b[2Jpath")}, "REC STOP:IO"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := StatusModel{Capture: tc.status, Actors: []StatusActor{{Label: "actor", Thread: menuAddress("actor")}}, Notice: "notice"}
			full := "actor  · notice"
			if tc.badge != "" {
				full = tc.badge + "  " + full
			}
			for width := 0; width <= 40; width++ {
				row := RenderStatusRow(width, model)
				plain := string(ansi.Strip([]byte(row.Body)))
				if textwidth.Width(plain) > width {
					t.Fatalf("width %d overflow: %q", width, plain)
				}
				wantPrefix := tc.badge
				if len(wantPrefix) > width {
					wantPrefix = wantPrefix[:width]
				}
				if !strings.HasPrefix(plain, wantPrefix) {
					t.Fatalf("width %d: %q lacks %q", width, plain, wantPrefix)
				}
				if width == 40 && plain != full {
					t.Fatalf("row = %q; want %q", plain, full)
				}
				for col := 0; col < len(tc.badge) && col < width; col++ {
					if _, ok := row.ColumnToActor(col); ok {
						t.Fatalf("badge column %d clickable", col)
					}
				}
				for _, chip := range row.Chips {
					if !strings.HasPrefix("actor", plain[chip.Start:chip.End]) {
						t.Fatalf("incorrect actor span %+v: %q", chip, plain)
					}
				}
			}
		})
	}
}

func TestCaptureFailureBeforeRunAppearsOnFirstFrame(t *testing.T) {
	r, err := terminalcapture.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	r.Record(terminalcapture.Record{Kind: "oversized", Data: make([]byte, 9<<20)})
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) { c.SetCapture(r) })
	waitFor(t, "pre-start capture failure", func() bool {
		return strings.HasPrefix(f.screen.row(24), "REC STOP:queue")
	})
}
