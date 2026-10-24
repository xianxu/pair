package couchtty

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestStatusSendWithoutPreviewAndDeliveredMask(t *testing.T) {
	r := newWebNotesCast(t, false) // no submission manager or browser preview
	r.note(t, "Ana", "send this note")
	waitFor(t, "send appears without preview", func() bool { r.con.mu.Lock(); defer r.con.mu.Unlock(); return r.con.statusWebSend != (ColumnSpan{}) })
	r.con.mu.Lock()
	cast := r.con.wcast.cast
	r.con.mu.Unlock()
	selected, err := cast.SubmissionSelection()
	if err != nil {
		t.Fatal(err)
	}
	n := selected.Notes[0]
	if err := cast.SetDeliveredFeedback(core.DeliveredFeedback{RecordingID: filepath.Base(cast.RecordingPath()), Notes: map[core.PreviewNote]struct{}{{ID: n.ID, Revision: n.Revision}: {}}}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "ack hides send without preview", func() bool { r.con.mu.Lock(); defer r.con.mu.Unlock(); return r.con.statusWebSend == (ColumnSpan{}) })
}

func TestStatusSendClickExactEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name    string
		button  int
		release bool
		stale   string
		want    int
	}{
		{name: "left", want: 1}, {name: "right", button: 2}, {name: "shift", button: 4}, {name: "release", release: true}, {name: "stale-endpoint", stale: "endpoint"},
		{name: "stale-cast", stale: "cast"},
		{name: "stale-pane", stale: "pane"},
		{name: "stale-thread", stale: "thread"},
		{name: "panel", stale: "panel"},
		{name: "wheel", button: 64},
		{name: "drag", button: 32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newWebNotesCast(t, false)
			r.note(t, "Ana", "send")
			var span ColumnSpan
			waitFor(t, "send span", func() bool {
				r.con.mu.Lock()
				defer r.con.mu.Unlock()
				span = r.con.statusWebSend
				return span != (ColumnSpan{})
			})
			before := sendChordWrites(r.child.Writes())
			err := r.con.runTerminalCommand(context.Background(), func() error {
				r.con.mu.Lock()
				row := int(r.con.size.Rows) - r.con.statusRows + 1
				switch tc.stale {
				case "endpoint":
					r.con.statusSendTarget.endpoint = nil
				case "cast":
					r.con.statusSendTarget.cast = nil
				case "pane":
					r.con.statusSendTarget.pane = nil
				case "thread":
					r.con.statusSendTarget.thread.Tag = "other"
				case "panel":
					r.con.focus = FocusPanel()
				}
				r.con.mu.Unlock()
				suffix := "M"
				if tc.release {
					suffix = "m"
				}
				var d terminal.Decoder
				events, err := d.Feed([]byte(fmt.Sprintf("\x1b[<%d;%d;%d%s", tc.button, span.Start+1, row, suffix)))
				if err != nil {
					return err
				}
				for _, event := range events {
					r.con.routeMouseEvent(event)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := r.child.Endpoint().Flush(context.Background()); err != nil {
				t.Fatal(err)
			}
			if got := sendChordWrites(r.child.Writes()) - before; got != tc.want {
				t.Fatalf("send admissions=%d want %d", got, tc.want)
			}
		})
	}
}
func sendChordWrites(writes [][]byte) int {
	n := 0
	for _, w := range writes {
		if bytes.Equal(w, []byte("\x1b[13;8u")) {
			n++
		}
	}
	return n
}
