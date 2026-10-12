package broadcast

import (
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/castlock"
)

func TestLinkLocksOneLockPerToken(t *testing.T) {
	var ls LinkLocks
	var rotated []string
	on := func(label string, r castlock.Rotation) { rotated = append(rotated, label) }
	for _, l := range []struct {
		label, token string
		digits       int
	}{{LinkView, "tv", 4}, {LinkPointer, "tp", 4}, {LinkDrive, "td", DriveCodeDigits}} {
		if err := ls.Mint(l.label, l.token, l.digits, on); err != nil {
			t.Fatal(err)
		}
	}
	codes := ls.Codes()
	if len(codes) != 3 || codes[0].Link != LinkView || codes[1].Link != LinkPointer || codes[2].Link != LinkDrive || len(codes[2].Code) != 6 {
		t.Fatalf("codes %+v", codes)
	}
	view, pointer := ls.LockFor("tv"), ls.LockFor("tp")
	if view == nil || pointer == nil || view == pointer || ls.LockFor("nope") != nil {
		t.Fatal("LockFor does not route by token")
	}
	s, _ := view.Unlock(time.Now(), view.Code())
	if _, ok := pointer.Valid(s); ok {
		t.Fatal("a view session opened the pointer link")
	}
	if err := ls.Rotate(LinkPointer); err != nil {
		t.Fatal(err)
	}
	if err := ls.Rotate("no-such-link"); err == nil {
		t.Fatal("rotated an unknown link")
	}
	if len(rotated) != 1 || rotated[0] != LinkPointer {
		t.Fatalf("rotations told %v", rotated)
	}
	ls.End()
	if _, ok := view.Valid(s); ok {
		t.Fatal("a session outlived End")
	}
	if err := ls.Mint("late", "tl", 4, nil); err == nil {
		t.Fatal("minted after End")
	}
	var none *LinkLocks
	if none.LockFor("tv") != nil || none.Codes() != nil {
		t.Fatal("a nil registry is not empty")
	}
	none.End()
}
