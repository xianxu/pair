package webcast

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The driven fixture's pages (#467): a search box, a save button and a link
// to a second page with its own box and button.
const (
	drivenPage1 = `<!doctype html><title>Invoices</title><body style="margin:0;font:16px sans-serif;padding:24px">
<h1>Invoices</h1>
<input id=q placeholder="Search invoices" style="width:300px;height:32px">
<button id=save style="height:36px" onclick="this.textContent='Saved'">Save</button>
<p><a id=next href="/detail">Open invoice 42</a></p>
<ul><li>Invoice 41</li><li>Invoice 42</li><li>Invoice 43</li></ul>
</body>`
	drivenPage2 = `<!doctype html><title>Invoice 42</title><body style="margin:0;font:16px sans-serif;padding:24px">
<h1>Invoice 42</h1>
<label>Note <input id=memo style="width:300px;height:32px"></label>
<button id=approve style="height:36px" onclick="this.textContent='Approved'">Approve</button>
</body>`
)

func serveDriven(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, drivenPage1) })
	mux.HandleFunc("/detail", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, drivenPage2) })
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })
	return "http://" + l.Addr().String() + "/"
}

func init() {
	// driven: a real remote-driving session through the production pieces
	// (MemController behind liveGate, TabInjector over the recorder): Ana
	// asks and is granted, clicks and types, the operator clicks too, Ana
	// follows a link (a navigation while driving), a pause and resume,
	// Ana gives control back; Bo asks and is turned down; Ana is granted
	// again and the operator revokes; Cy asks and goes away (a leave).
	scenarios["driven"] = func(t *testing.T, h liveHarness, s string) {
		c := h.c
		pause := func(ms int) { time.Sleep(time.Duration(ms) * time.Millisecond) }
		for deadline := time.Now().Add(10 * time.Second); !c.Opened(); time.Sleep(20 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatal("the recorder never opened")
			}
		}
		ctrl := drive.NewMemController(liveGate{c}, drive.NewTabInjector(c), func() int64 { return 0 }, time.Second)
		defer ctrl.Close()
		go followDriving(ctrl, c)
		must := func(err error) {
			t.Helper()
			if err != nil {
				t.Fatal(err)
			}
		}
		seat := func(name string) (drive.Seat, func()) {
			st, err := ctrl.Join(name)
			must(err)
			detach, err := ctrl.Attach(st.Viewer.ID)
			must(err)
			return st, detach
		}
		// at is the element with that id as the recording knows it, and the
		// segment shown.
		at := func(id string) (recording.NodeID, int) {
			var n recording.NodeID
			var seg int
			must(c.WithTab(context.Background(), "", func(tab drive.Tab) error {
				if el := drive.ElementByID(tab.Page(), id); el != nil {
					n = el.ID
				}
				seg = tab.Page().Seg
				return nil
			}))
			if n == 0 {
				t.Fatalf("no #%s in the recording", id)
			}
			return n, seg
		}
		click := func(id string) (drive.Act, int) {
			n, seg := at(id)
			x, y := h.pagePoint(t, "#"+id)
			return drive.Act{A: drive.ActClick, ID: n, X: x, Y: y, Button: drive.ButtonLeft, Count: 1}, seg
		}
		input := func(key string, seg int, acts ...drive.Act) {
			t.Helper()
			must(ctrl.Input(context.Background(), key, drive.Batch{Seg: seg, Acts: acts}))
		}

		pause(800)
		ana, _ := seat("Ana")
		must(ctrl.Request(ana.Key))
		pause(600)
		must(ctrl.Grant(ana.Viewer.ID))
		pause(600)
		a, seg := click("q")
		input(ana.Key, seg, a)
		for _, part := range []string{"inv", "oice", " 42"} {
			pause(150)
			input(ana.Key, seg, drive.Act{A: drive.ActText, V: part})
		}
		input(ana.Key, seg, drive.Act{A: drive.ActKey, Type: drive.KeyDown, Key: "Enter"}, drive.Act{A: drive.ActKey, Type: drive.KeyUp, Key: "Enter"})
		pause(800)
		h.click(t, s, "#save") // the operator, while Ana drives
		pause(800)
		a, seg = click("next")
		input(ana.Key, seg, a) // a navigation while driving
		pause(1500)
		must(c.SetLive(false))
		pause(700)
		must(c.SetLive(true))
		pause(1000)
		a, seg = click("memo")
		input(ana.Key, seg, a, drive.Act{A: drive.ActText, V: "approved by phone"})
		pause(600)
		must(ctrl.Release(ana.Key))
		pause(600)
		bo, _ := seat("Bo")
		must(ctrl.Request(bo.Key))
		pause(500)
		must(ctrl.Deny(bo.Viewer.ID))
		pause(500)
		must(ctrl.Request(ana.Key))
		pause(400)
		must(ctrl.Grant(ana.Viewer.ID))
		pause(500)
		a, seg = click("approve")
		input(ana.Key, seg, a)
		pause(600)
		must(ctrl.Revoke())
		pause(500)
		cy, cyGone := seat("Cy")
		must(ctrl.Request(cy.Key))
		pause(400)
		cyGone() // Cy's stream goes: after the grace period, a leave
		pause(1600)
		if p := ctrl.Control().Pending; len(p) != 0 {
			t.Fatalf("Cy never left: %+v", p)
		}
	}
}
