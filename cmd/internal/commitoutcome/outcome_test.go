package commitoutcome

import (
	"errors"
	"testing"
)

func TestOutcomeOf(t *testing.T) {
	t.Parallel()
	cause := errors.New("cause")
	for _, test := range []struct {
		name string
		err  error
		want Outcome
	}{
		{"success", nil, Committed},
		{"ordinary failure", cause, NotAuthoritative},
		{"indeterminate", Wrap(Indeterminate, cause), Indeterminate},
		{"committed cleanup", Wrap(Committed, cause), Committed},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Of(test.err); got != test.want || (test.err != nil && !errors.Is(test.err, cause)) {
				t.Fatalf("Of(%v)=%v want=%v", test.err, got, test.want)
			}
		})
	}
}
