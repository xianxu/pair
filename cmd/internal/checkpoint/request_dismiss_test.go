package checkpoint

import (
	"strings"
	"testing"
)

// The dismissal rule on literal requests: only the exact FAILED request may be
// deleted; an empty id means the retained one (pair#280).
func TestCheckDismissible(t *testing.T) {
	failed := &Request{ID: "aa", Phase: Failed}
	for _, tc := range []struct {
		name string
		r    *Request
		id   string
		want string
	}{
		{"none", nil, "", "no continuation request"},
		{"obsolete", failed, "bb", "obsolete continuation request"},
		{"pending", &Request{ID: "aa", Phase: Pending}, "aa", "is pending; only a failed continuation can be dismissed"},
		{"running", &Request{ID: "aa", Phase: Running}, "aa", "is running; only a failed continuation can be dismissed"},
		{"complete", &Request{ID: "aa", Phase: Complete}, "", "is complete; only a failed continuation can be dismissed"},
	} {
		if err := CheckDismissible(tc.r, tc.id); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want %q", tc.name, err, tc.want)
		}
	}
	for _, id := range []string{"", "aa"} {
		if err := CheckDismissible(failed, id); err != nil {
			t.Errorf("exact failed request (id %q) refused: %v", id, err)
		}
	}
}

// A failed request always names BOTH exits; the CLI forms appear only with a tag.
func TestExitsNameBothWaysOutOfAFailedRequest(t *testing.T) {
	for _, phase := range AllPhases() {
		plain, tagged := Exits(phase, ""), Exits(phase, "couch-01")
		both := strings.Contains(plain, "Retry continuation") && strings.Contains(plain, "Dismiss continuation")
		if both != (phase == Failed) {
			t.Errorf("%s: exits %q; both exits must be named exactly for a failed request", phase, plain)
		}
		if strings.Contains(plain, "couch --internal") || !strings.Contains(tagged, "couch --internal retry-continuation couch-01") {
			t.Errorf("%s: CLI forms must follow the tag: %q / %q", phase, plain, tagged)
		}
		if phase == Failed && !strings.Contains(tagged, "couch --internal dismiss-continuation couch-01") {
			t.Errorf("failed request's CLI forms omit dismiss: %q", tagged)
		}
	}
}
