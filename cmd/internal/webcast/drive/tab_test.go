package drive

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestOnTarget(t *testing.T) {
	// top doc > body(1) > div(2) > button(3) > span(4); div > iframe(5),
	// whose document(6) > input(7); button's shadow root(8) > b(9).
	el := func(id recording.NodeID, name string) *recording.DOMNode {
		return &recording.DOMNode{ID: id, Y: recording.YEl, N: name}
	}
	nodes := map[recording.NodeID]*recording.DOMNode{}
	add := func(n, parent *recording.DOMNode) *recording.DOMNode {
		nodes[n.ID] = n
		if parent != nil {
			n.Parent = parent
			parent.Kids = append(parent.Kids, n)
		}
		return n
	}
	body := add(el(1, "body"), nil)
	div := add(el(2, "div"), body)
	button := add(el(3, "button"), div)
	add(el(4, "span"), button)
	frame := add(el(5, "iframe"), div)
	doc := add(&recording.DOMNode{ID: 6}, nil)
	doc.Host, frame.Frame = frame, doc
	add(el(7, "input"), doc)
	shadow := add(&recording.DOMNode{ID: 8}, nil)
	shadow.Host, button.Shadow = button, shadow
	add(el(9, "b"), shadow)

	node := func(id recording.NodeID) *recording.DOMNode { return nodes[id] }
	for _, c := range []struct {
		hit, id recording.NodeID
		want    bool
		why     string
	}{
		{3, 3, true, "the element itself"},
		{4, 3, true, "a child of the target"},
		{9, 3, true, "inside the target's shadow root"},
		{7, 5, true, "inside the target frame's document"},
		{3, 4, false, "the target's parent"},
		{2, 3, false, "an ancestor of the target"},
		{4, 7, false, "an unrelated element"},
		{5, 7, true, "an out-of-process frame's owner, the target inside it"},
		{2, 7, false, "a non-frame ancestor of the target"},
		{0, 3, false, "nothing at the point"},
		{3, 99, false, "an id the page doesn't have"},
	} {
		if got := onTarget(node, c.hit, c.id); got != c.want {
			t.Errorf("%s: onTarget(%d, %d) = %v, want %v", c.why, c.hit, c.id, got, c.want)
		}
	}
}
