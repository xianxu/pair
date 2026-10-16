package tailmarkup

import "testing"

func TestStrip(t *testing.T) {
	for _, tc := range []struct{ in, want string }{{" │‹rev›selected‹/rev›‹cursor›│ ", "selected"}, {"‹‹rev›literal‹‹/rev›", "‹rev›literal‹/rev›"}, {"‹dim›dim‹/dim›", "dim"}} {
		if got := Strip(tc.in); got != tc.want {
			t.Fatalf("got %q want %q", got, tc.want)
		}
	}
}
