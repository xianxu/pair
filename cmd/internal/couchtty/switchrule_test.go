package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func addr(tag string) couchcore.ThreadAddress {
	return couchcore.ThreadAddress{RepoScope: "scope", Tag: couchcore.ThreadTag(tag)}
}

// The Spec's four consequences, driven as sequences rather than asserted one
// rule at a time: the rule is only interesting in composition.
func TestSwitchTrackerPrevious(t *testing.T) {
	a, n1, n2, c := addr("a"), addr("n1"), addr("n2"), addr("c")

	tests := []struct {
		name string
		// each hop is {target, viaNotification}
		hops []struct {
			target couchcore.ThreadAddress
			notify bool
		}
		want   couchcore.ThreadAddress
		wantOK bool
	}{
		{
			name: "no hop yet has nowhere to return to",
			want: couchcore.ThreadAddress{},
		},
		{
			name: "first landing is not yet a return target",
			hops: []struct {
				target couchcore.ThreadAddress
				notify bool
			}{{a, false}},
			want: couchcore.ThreadAddress{},
		},
		{
			name: "first hop from working actor A pins A",
			hops: []struct {
				target couchcore.ThreadAddress
				notify bool
			}{{a, false}, {n1, true}},
			want: a, wantOK: true,
		},
		{
			name: "N1 to N2 leaves A pinned",
			hops: []struct {
				target couchcore.ThreadAddress
				notify bool
			}{{a, false}, {n1, true}, {n2, true}},
			want: a, wantOK: true,
		},
		{
			name: "a manual detour out of a notification actor leaves A pinned",
			hops: []struct {
				target couchcore.ThreadAddress
				notify bool
			}{{a, false}, {n1, true}, {n2, true}, {c, false}},
			want: a, wantOK: true,
		},
		{
			name: "an ordinary switch between working actors advances previous",
			hops: []struct {
				target couchcore.ThreadAddress
				notify bool
			}{{a, false}, {c, false}},
			want: a, wantOK: true,
		},
		{
			name: "switching to the actor already current does not spend the slot",
			hops: []struct {
				target couchcore.ThreadAddress
				notify bool
			}{{a, false}, {c, false}, {c, false}},
			want: a, wantOK: true,
		},
		{
			name: "switching to previous itself leaves the slot on the actor left behind",
			hops: []struct {
				target couchcore.ThreadAddress
				notify bool
			}{{a, false}, {c, false}, {a, false}},
			want: c, wantOK: true,
		},
		{
			name: "returning home from a notification actor leaves previous == current",
			hops: []struct {
				target couchcore.ThreadAddress
				notify bool
			}{{a, false}, {n1, true}, {a, false}},
			want: a, wantOK: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var tracker SwitchTracker
			for _, hop := range test.hops {
				tracker.Switch(hop.target, hop.notify)
			}
			got, ok := tracker.Previous()
			if got != test.want || ok != test.wantOK {
				t.Fatalf("Previous() = (%v, %v), want (%v, %v)", got, ok, test.want, test.wantOK)
			}
		})
	}
}

// The Spec calls this out as intended, not as a bug: the operator is home and
// there is nowhere to bounce to. Asserted so a later "fix" has to argue with it.
func TestSwitchTrackerSecondReturnIsANoOp(t *testing.T) {
	a, n1 := addr("a"), addr("n1")

	var tracker SwitchTracker
	tracker.Switch(a, false)
	tracker.Switch(n1, true)

	previous, ok := tracker.Previous()
	if !ok || previous != a {
		t.Fatalf("Previous() = (%v, %v), want (%v, true)", previous, ok, a)
	}
	tracker.Switch(previous, false) // ctrl+backspace home

	again, ok := tracker.Previous()
	if !ok || again != a {
		t.Fatalf("after returning home Previous() = (%v, %v), want (%v, true) -- "+
			"previous stays A so the next ctrl+backspace is a no-op", again, ok, a)
	}
}

// Dropping is how onExit reports a landing that is not a switch: the operator
// lands on the panel, and the dead thread must not become a return target.
func TestSwitchTrackerDrop(t *testing.T) {
	a, b := addr("a"), addr("b")

	t.Run("dropping previous clears the return target", func(t *testing.T) {
		var tracker SwitchTracker
		tracker.Switch(a, false)
		tracker.Switch(b, false)
		tracker.Drop(a)
		if got, ok := tracker.Previous(); ok {
			t.Fatalf("Previous() = (%v, true), want no return target after A exited", got)
		}
	})

	t.Run("dropping current does not promote it to previous", func(t *testing.T) {
		var tracker SwitchTracker
		tracker.Switch(a, false)
		tracker.Switch(b, false)
		tracker.Drop(b)
		got, ok := tracker.Previous()
		if !ok || got != a {
			t.Fatalf("Previous() = (%v, %v), want (%v, true)", got, ok, a)
		}
		tracker.Switch(a, false)
		if got, ok := tracker.Previous(); ok {
			t.Fatalf("Previous() = (%v, true), want none -- the exited current must not "+
				"have advanced previous", got)
		}
	})

	t.Run("dropping an unrelated thread changes nothing", func(t *testing.T) {
		var tracker SwitchTracker
		tracker.Switch(a, false)
		tracker.Switch(b, false)
		tracker.Drop(addr("unrelated"))
		got, ok := tracker.Previous()
		if !ok || got != a {
			t.Fatalf("Previous() = (%v, %v), want (%v, true)", got, ok, a)
		}
	})
}
