package hostty

import "testing"

func TestPrivateModesFormatsOneDECSETOrDECRST(t *testing.T) {
	for _, tt := range []struct {
		modes []int
		on    bool
		want  string
	}{
		{nil, true, ""},
		{nil, false, ""},
		{[]int{1002}, true, "\x1b[?1002h"},
		{[]int{1002, 1006}, false, "\x1b[?1002;1006l"},
	} {
		if got := PrivateModes(tt.modes, tt.on); got != tt.want {
			t.Fatalf("PrivateModes(%v, %v) = %q, want %q", tt.modes, tt.on, got, tt.want)
		}
	}
	// The click-only constant is one instance of the formatter, so the two
	// cannot drift.
	if got := PrivateModes([]int{1000, 1006}, true); got != EnableMouseClicks {
		t.Fatalf("PrivateModes(1000,1006) = %q, want EnableMouseClicks %q", got, EnableMouseClicks)
	}
}
