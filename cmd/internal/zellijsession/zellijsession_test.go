package zellijsession

import "testing"

func TestRowState(t *testing.T) {
	// Realistic --no-formatting rows (name + [created ...] + status suffix).
	raw := "pair-work [Created 1h ago]\n" +
		"pair-old [Created 9days ago] (EXITED - attach to resurrect)\n" +
		"📁1-153 [Created 4h ago] (current)\n" +
		"other-thing [Created 2h ago]\n"
	cases := []struct {
		session     string
		wantPresent bool
		wantExited  bool
		wantLive    bool
	}{
		{"pair-work", true, false, true},
		{"pair-old", true, true, false},
		{"📁1-153", true, false, true},
		{"pair-missing", false, false, false},
		{"other-thing", true, false, true},
		// A prefix of a real row must NOT match on the first field.
		{"pair", false, false, false},
	}
	for _, tc := range cases {
		present, exited := RowState(raw, tc.session)
		if present != tc.wantPresent || exited != tc.wantExited {
			t.Errorf("RowState(%q) = (%v,%v), want (%v,%v)", tc.session, present, exited, tc.wantPresent, tc.wantExited)
		}
		if got := Live(raw, tc.session); got != tc.wantLive {
			t.Errorf("Live(%q) = %v, want %v", tc.session, got, tc.wantLive)
		}
	}
}
