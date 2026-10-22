package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/terminal"
	"testing"
	"time"
)

func TestCodesMenuReflowsAgainstComposedChrome(t *testing.T) {
	c := &Console{now: time.Now}
	rows := []codesMenuRow{{label: "view 123456", actions: []codesMenuWord{{text: "rotate", action: codesAction{link: "view"}}}}}
	c.codes.menu = castScreen
	c.codes.menuBox = layoutCodesMenu("LIVE codes", rows, 80, 24).Select(codesMenuSel{row: 0, word: 0})
	f := terminal.Frame{Geometry: terminal.Geometry{Cols: 60, Rows: 12}, Cells: make([]terminal.Cell, 60*12), ChromeRows: 2}
	c.codesLayer(f, terminal.FramePublic)
	box := c.codes.menuBox
	if !box.Open() || box.X+box.W != 60 || box.Y+box.H != 10 || box.sel != (codesMenuSel{row: 0, word: 0}) {
		t.Fatalf("retained menu did not follow composed chrome: %+v", box)
	}
}

func TestAttentionPanelReflowsAgainstComposedChrome(t *testing.T) {
	c := &Console{now: time.Now}
	c.attn.list = c.attn.list.Apply(hubNotify(menuAddress("one"), "pending review"), time.Now())
	c.attn.open = true
	c.layoutAttentionLocked(80, 25)
	f := terminal.Frame{Geometry: terminal.Geometry{Cols: 40, Rows: 8}, Cells: make([]terminal.Cell, 40*8), ChromeRows: 2}
	c.attentionLayer(f)
	box := c.attn.box
	if !box.Open() || box.X+box.W > 40 || box.Y+box.H > 6 {
		t.Fatalf("retained attention panel outside current body: %+v", box)
	}
}
