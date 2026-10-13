package webcast

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// A driver's click that follows a link is recorded, attributed, before
// the navigation's segment (#467): the held observer's document goes
// before the take can reach it.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveDriveNavigation -v
func TestLiveDriveNavigation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rec")
	c := liveRecordingChrome(t, dir, -1)
	h := liveHarness{c}
	url := serveDriven(t)
	if err := c.call(h.session(t, ""), "Page.navigate", map[string]any{"url": url}, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	if err := c.SetLive(true); err != nil {
		t.Fatal(err)
	}
	for !c.Opened() {
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	ctrl := drive.NewMemController(liveGate{c}, drive.NewTabInjector(c), func() int64 { return 0 }, time.Minute)
	ctrl.WithControlReset(c.ResetControl)
	if err := ctrl.SetDriveAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	defer ctrl.Close()
	seat, _ := ctrl.Join("Ana")
	ctrl.Request(seat.Key)
	ctrl.Grant(seat.Viewer.ID)
	time.Sleep(300 * time.Millisecond)
	var link recording.NodeID
	var seg int
	c.WithTab(context.Background(), "", func(tab drive.Tab) error {
		link, seg = drive.ElementByID(tab.Page(), "next").ID, tab.Page().Seg
		return nil
	})
	x, y := h.pagePoint(t, "#next")
	if err := ctrl.Input(context.Background(), seat.Key, drive.Batch{Seg: seg, Acts: []drive.Act{{A: drive.ActClick, ID: link, X: x, Y: y, Button: drive.ButtonLeft, Count: 1}}}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond)
	c.SetLive(false)
	c.Close()
	<-c.Done()
	r, err := recording.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	clicked, navigated := false, false
	for _, rec := range r.Records {
		switch x := rec.(type) {
		case *recording.Click:
			if x.ID == link && x.By == seat.Viewer.ID && !navigated {
				clicked = true
			}
		case *recording.Seg:
			if x.Why == recording.SegNavigate {
				navigated = true
			}
		}
	}
	if !clicked || !navigated {
		t.Fatalf("the driver's navigating click: recorded before the navigation %v, navigated %v", clicked, navigated)
	}
}
