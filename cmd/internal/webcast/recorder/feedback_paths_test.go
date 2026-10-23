package recorder

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

func TestResolveOperatorPathsUsesCurrentRecorderIdentity(t *testing.T) {
	h, b, _ := operatorHarness(t)
	doc := b.OperatorView().Doc
	var tab string
	if err := h.r.ask(context.Background(), func() { tab = h.r.m.st.Tab.ID }); err != nil {
		t.Fatal(err)
	}
	ids, err := h.r.ResolveOperatorPaths(context.Background(), "A", doc, tab, [][]int{{1, 2, 0}, {1, 2, 0}})
	if err != nil || len(ids) != 2 || ids[0] <= 0 || ids[0] != ids[1] {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	for _, tc := range []struct {
		session string
		doc     int
		tab     string
		paths   [][]int
	}{
		{"other", doc, tab, [][]int{{1, 2, 0}}}, {"A", doc + 1, tab, [][]int{{1, 2, 0}}}, {"A", doc, "other", [][]int{{1, 2, 0}}}, {"A", doc, tab, [][]int{{1, 2, 999999}}},
	} {
		if _, err := h.r.ResolveOperatorPaths(context.Background(), tc.session, tc.doc, tc.tab, tc.paths); !errors.Is(err, board.ErrStale) {
			t.Fatalf("accepted stale paths: %+v %v", tc, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.r.ResolveOperatorPaths(ctx, "A", doc, tab, [][]int{{1, 2, 0}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled=%v", err)
	}
}
