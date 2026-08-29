package ctxmeter

import (
	"testing"
)

func TestHumanize(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0"}, {999, "999"},
		{1000, "1k"}, {397556, "398k"}, // round half-up
		{999999, "1000k"},                    // k-branch can emit 4 digits
		{1000000, "1.0M"}, {1490000, "1.4M"}, // M-branch floors
	}
	for _, c := range cases {
		if got := Humanize(c.n); got != c.want {
			t.Errorf("Humanize(%d)=%q want %q", c.n, got, c.want)
		}
	}
}
