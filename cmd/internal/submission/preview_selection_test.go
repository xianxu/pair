package submission

import (
	"fmt"
	"testing"
)

func TestSelectSketchPreviewOldestExactVersionsAcrossTargets(t *testing.T) {
	var all []PreviewItem
	for i := 0; i < 12; i++ {
		all = append(all, PreviewItem{RecordingID: "r", SketchID: fmt.Sprint(i), Sequence: 5, Digest: "a"})
	}
	sent := map[PreviewItem]struct{}{all[0]: {}, all[2]: {}}
	newer := all[0]
	newer.Sequence++
	all = append(all, newer)
	got := SelectSketchPreview(all, sent, PreviewStamp{Board: 7, Delivered: 3})
	if len(got.Items) != 8 || got.Remaining != 3 || got.Items[0] != all[1] || got.Items[1] != all[3] || got.Stamp.Board != 7 {
		t.Fatalf("preview %+v", got)
	}
	for _, v := range got.Items {
		sent[v] = struct{}{}
	}
	next := SelectSketchPreview(all, sent, PreviewStamp{})
	if len(next.Items) != 3 || next.Items[2] != newer {
		t.Fatalf("next %+v", next)
	}
	got.Items[0].SketchID = "mutated"
	if all[1].SketchID != "1" {
		t.Fatal("selector borrowed candidates")
	}
}
