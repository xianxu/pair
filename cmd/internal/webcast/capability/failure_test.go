package capability

import (
	"errors"
	"fmt"
	"testing"
)

func TestFailureOutcomeDefaultsClosed(t *testing.T) {
	cause := errors.New("queue unavailable")
	for _, test := range []struct {
		err  error
		want CleanupOutcome
	}{
		{cause, Broken},
		{&Failure{Outcome: Retryable, Cause: cause}, Retryable},
		{fmt.Errorf("wrapped: %w", &Failure{Outcome: Retryable, Cause: cause}), Retryable},
		{&Failure{Outcome: Running, Cause: cause}, Broken},
		{&Failure{Outcome: Broken, Cause: cause}, Broken},
	} {
		if got := FailureOutcome(test.err); got != test.want {
			t.Fatalf("%v => %v, want %v", test.err, got, test.want)
		}
	}
	err := &Failure{Outcome: Retryable, Cause: cause}
	if !errors.Is(err, cause) || err.Error() != cause.Error() {
		t.Fatal("failure lost its evidence")
	}
}
