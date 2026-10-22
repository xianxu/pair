package couchtty

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/textwidth"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCastControlIntent(t *testing.T) {
	for _, action := range []string{"enable", "disable", "retry", "wait", "end-cast", "start-cast", ""} {
		if got := castControlIntent(true, capability.Presentation{Action: action}); got != "copy" {
			t.Fatalf("right %s: %s", action, got)
		}
		want := action
		if action == "enable" {
			want = "enable-copy"
		}
		if got := castControlIntent(false, capability.Presentation{Action: action}); got != want {
			t.Fatalf("left %s: %s", action, got)
		}
	}
}
func TestWebCapabilityRowStatesAndRecovery(t *testing.T) {
	for _, a := range []bool{false, true} {
		for _, d := range []bool{false, true} {
			row := RenderStatusRow(120, StatusModel{Webcast: WebcastLive, WebAnnotationOn: a, WebDrivingOn: d})
			for _, v := range []struct {
				label string
				on    bool
			}{{broadcast.PointerLabel, a}, {WebcastDriveLinkLabel, d}} {
				sgr := dimSGR
				if v.label == WebcastDriveLinkLabel {
					sgr = ""
				}
				if v.on {
					sgr = broadcast.PointerSGR
				}
				if !strings.Contains(row.Body, sgr+v.label) {
					t.Fatalf("a=%v d=%v %q lacks %q", a, d, row.Body, sgr+v.label)
				}
			}
		}
	}
	row := RenderStatusRow(65, StatusModel{Webcast: WebcastLive, CapabilityNotice: "Click 👆 to retry: couldn't clear marks", Actors: []StatusActor{{Label: strings.Repeat("actor", 30)}}})
	if !strings.Contains(rowPlain(row), "Click 👆 to retry") {
		t.Fatalf("recovery hidden: %q", row.Body)
	}
	for width := 1; width < 40; width++ {
		row := RenderStatusRow(width, StatusModel{Webcast: WebcastLive, WebAnnotationOn: true, WebDrivingOn: true})
		for _, s := range []ColumnSpan{row.WebPointerLink, row.WebDriveLink} {
			if s.End > width {
				t.Fatalf("width %d span %+v", width, s)
			}
		}
	}
}

func TestWebCapabilityClicksRevokeAndPreserveLinks(t *testing.T) {
	r := newDriveCast(t)
	a := r.viewer(t, "A")
	r.viewer(t, "B")
	if err := r.cast().Drive().Grant(a); err != nil {
		t.Fatal(err)
	}
	link := r.link
	before := copies(r.consoleFixture, link)
	r.con.onWebCapabilityClick("drive", false)
	waitFor(t, "drive off", func() bool { return r.cast().Capabilities().Drive.Phase == capability.Off })
	if got := r.control(); got.Driver != nil || len(got.Pending) != 0 {
		t.Fatalf("not revoked: %+v", got)
	}
	r.con.onWebCapabilityClick("drive", true)
	waitFor(t, "off copied", func() bool { return copies(r.consoleFixture, link) == before+1 })
	if r.cast().Capabilities().Drive.Phase != capability.Off {
		t.Fatal("copy enabled")
	}
	r.con.onWebCapabilityClick("drive", false)
	waitFor(t, "enabled and copied", func() bool {
		return r.cast().Capabilities().Drive.Phase == capability.On && copies(r.consoleFixture, link) == before+2
	})
	if r.cast().Current().DriveLink != link || r.control().Driver != nil {
		t.Fatal("link changed or driver restored")
	}
}

type cleanupBrowser struct {
	*webcast.FakeDriven
	calls            atomic.Int32
	entered, release chan struct{}
	terminal         bool
}

func (b *cleanupBrowser) ClearAnnotations(ctx context.Context, id uint64) error {
	n := b.calls.Add(1)
	if n == 1 {
		close(b.entered)
		select {
		case <-b.release:
		case <-ctx.Done():
		}
		outcome := capability.Retryable
		if b.terminal {
			outcome = capability.Broken
		}
		return &capability.Failure{Outcome: outcome, Cause: errors.New("recorder unavailable")}
	}
	return nil
}
func TestWebCapabilityRecoveryIsVisibleAndClickRetries(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(fmt.Sprint(broken), func(t *testing.T) {
			var b *cleanupBrowser
			r := newWebcastRig(t, func(r *webcastRig) {
				r.wrap = func(f *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser {
					d, err := webcast.NewFakeDriven(f, drive.DrivenPage(100), &recording.MemSink{})
					if err != nil {
						t.Fatal(err)
					}
					b = &cleanupBrowser{FakeDriven: d, entered: make(chan struct{}), release: make(chan struct{}), terminal: broken}
					return b
				}
			})
			r.open(t)
			r.con.mu.Lock()
			cast := r.con.wcast.cast
			r.con.mu.Unlock()
			if err := cast.SetCapability(context.Background(), webcast.CapabilityAnnotation, true); err != nil {
				t.Fatal(err)
			}
			r.con.onWebCapabilityClick(webcast.CapabilityAnnotation, false)
			<-b.entered
			if cast.Capabilities().Project(webcast.CapabilityAnnotation).Enabled {
				t.Fatal("cleanup remains enabled")
			}
			r.con.onWebCapabilityClick(webcast.CapabilityAnnotation, false)
			if b.calls.Load() != 1 {
				t.Fatal("duplicate cleanup")
			}
			close(b.release)
			want := "Click 👆 to retry"
			if broken {
				want = "Ctrl+click Chrome to end cast"
			}
			waitFor(t, "recovery visible", func() bool { return strings.Contains(r.statusRow(), want) })
			r.con.setNotice("Unrelated notice")
			if !strings.Contains(r.statusRow(), want) {
				t.Fatal("transient notice hid recovery")
			}
			r.con.onWebCapabilityClick(webcast.CapabilityAnnotation, true)
			if b.calls.Load() != 1 {
				t.Fatal("copy retried cleanup")
			}
			r.con.onWebCapabilityClick(webcast.CapabilityAnnotation, false)
			if broken {
				if b.calls.Load() != 1 {
					t.Fatal("broken retried")
				}
				r.clickCell(t, 16)
				waitFor(t, "broken cast ended by advertised action", func() bool {
					select {
					case <-cast.Done():
						return true
					default:
						return false
					}
				})
				return
			}
			waitFor(t, "explicit retry completed off", func() bool { return cast.Capabilities().Annotation.Phase == capability.Off })
			if b.calls.Load() != 2 {
				t.Fatalf("retry calls %d", b.calls.Load())
			}
		})
	}
}

func TestWebCapabilityStaleCompletionDoesNotCopy(t *testing.T) {
	for _, end := range []bool{false, true} {
		t.Run(fmt.Sprint(end), func(t *testing.T) {
			r := newDriveCast(t)
			cast := r.cast()
			if err := cast.SetCapability(context.Background(), webcast.CapabilityDrive, false); err != nil {
				t.Fatal(err)
			}
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
			}()
			go func() {
				_ = r.con.runTerminalCommand(r.con.lifetime, func() error { close(entered); <-release; return nil })
				close(done)
			}()
			<-entered
			before := copies(r.consoleFixture, r.link)
			completion := r.con.onWebCapabilityClick(webcast.CapabilityDrive, false)
			waitFor(t, "enabled before UI completion", func() bool { return cast.Capabilities().Drive.Phase == capability.On })
			if end {
				r.con.endWebcast()
			} else {
				// Authority publishes On before the enable worker releases its slot.
				// Wait for admission of the newer operation, not that earlier phase.
				waitFor(t, "newer disable admitted", func() bool {
					err := cast.SetCapability(context.Background(), webcast.CapabilityDrive, false)
					if errors.Is(err, webcast.ErrCapabilityBusy) {
						return false
					}
					if err != nil {
						t.Fatal(err)
					}
					return true
				})
			}
			close(release)
			<-done
			<-completion
			_ = r.con.runTerminalCommand(r.con.lifetime, func() error { return nil })
			if copies(r.consoleFixture, r.link) != before {
				t.Fatal("stale enable copied")
			}
		})
	}
}

func TestDriveChipNameClicksToggleAndCopy(t *testing.T) {
	r := newDriveCast(t)
	a := r.viewer(t, "Ana")
	r.viewer(t, "Bo")
	if err := r.cast().Drive().Grant(a); err != nil {
		t.Fatal(err)
	}
	var span ColumnSpan
	waitFor(t, "named drive chip", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		row := RenderStatusRow(120, r.con.statusModelLocked())
		span = r.con.statusWebDriveLink
		return span == row.WebDriveLink && strings.Contains(row.Body, broadcast.PointerSGR+"👽 Ana")
	})
	// Final visible name cell belongs to the same capability target.
	before := copies(r.consoleFixture, r.link)
	r.mouse(2, span.End-1, 23, false)
	r.mouse(2, span.End-1, 23, true)
	waitFor(t, "drive name copied", func() bool { return copies(r.consoleFixture, r.link) == before+1 })
	if got := r.control(); got.Driving() != a || len(got.Pending) != 1 {
		t.Fatalf("right copy changed control: %+v", got)
	}
	r.mouse(0, span.End-1, 23, false)
	r.mouse(0, span.End-1, 23, true)
	waitFor(t, "name click disables drive", func() bool { return r.cast().Capabilities().Drive.Phase == capability.Off })
	if got := r.control(); got.Driver != nil || len(got.Pending) != 0 {
		t.Fatalf("name click did not clear control: %+v", got)
	}
}

func TestDriveOffChipHasNoDimStyle(t *testing.T) {
	row := RenderStatusRow(120, StatusModel{Webcast: WebcastLive})
	if strings.Contains(row.Body, dimSGR+"👽") || strings.Contains(row.Body, broadcast.PointerSGR+"👽") {
		t.Fatalf("off drive has colour: %q", row.Body)
	}
}

func TestDriveChipThreeStatesAndClipping(t *testing.T) {
	for _, tc := range []struct {
		name                string
		on                  bool
		driver, want, style string
	}{
		{"off", false, "Ana", "👽", ""},
		{"idle", true, "", "👽", broadcast.PointerSGR},
		{"occupied", true, "Ana", "👽 Ana", broadcast.PointerSGR},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := RenderStatusRow(120, StatusModel{Webcast: WebcastLive, WebDrivingOn: tc.on, WebDriverName: tc.driver})
			suffix := tc.style + tc.want
			if tc.style != "" {
				suffix += "\x1b[0m"
			}
			if !strings.HasSuffix(row.Body, suffix) {
				t.Fatalf("wrong chip/style: %q want suffix %q", row.Body, suffix)
			}
			if row.WebDrive != (ColumnSpan{}) {
				t.Fatal("driver appeared in asks span")
			}
			if row.WebDriveLink.End-row.WebDriveLink.Start != textwidth.Width(tc.want) {
				t.Fatalf("chip target %+v does not cover %q", row.WebDriveLink, tc.want)
			}
		})
	}
	for width := 1; width < 70; width++ {
		row := RenderStatusRow(width, StatusModel{Webcast: WebcastLive, WebDrivingOn: true, WebDriverName: strings.Repeat("Ana", 30), WebDrive: "✋Bo"})
		if row.WebDriveLink.End > width {
			t.Fatalf("width%d target%+v", width, row.WebDriveLink)
		}
		if row.WebDriveLink != (ColumnSpan{}) && row.WebDriveLink.End != textwidth.Width(rowPlain(row)) {
			t.Fatalf("clipped name target not drawn end width%d: %+v %q", width, row.WebDriveLink, rowPlain(row))
		}
	}
	row := RenderStatusRow(120, StatusModel{Webcast: WebcastLive, WebDrivingOn: true, WebDriverName: "A\x1b[2Jna\n"})
	if strings.Contains(row.Body, "\x1b[2J") || strings.Contains(row.Body, "\n") {
		t.Fatalf("raw driver name reached terminal: %q", row.Body)
	}
}
