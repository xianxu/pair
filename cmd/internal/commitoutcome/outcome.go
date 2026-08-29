// Package commitoutcome defines the result of publishing durable authority.
package commitoutcome

import (
	"errors"
	"fmt"
)

type Outcome uint8

const (
	NotAuthoritative Outcome = iota
	Indeterminate
	Committed
)

func (o Outcome) String() string {
	switch o {
	case NotAuthoritative:
		return "not-authoritative"
	case Indeterminate:
		return "indeterminate"
	case Committed:
		return "committed"
	default:
		return "unknown"
	}
}

// Error reports a failure after authority may have been published.
type Error struct {
	Outcome Outcome
	Err     error
}

func (e *Error) Error() string { return fmt.Sprintf("publication %s: %v", e.Outcome, e.Err) }
func (e *Error) Unwrap() error { return e.Err }

func Wrap(outcome Outcome, err error) error {
	return &Error{Outcome: outcome, Err: err}
}

// Of maps success, typed publication failures, and ordinary failures to the
// exhaustive authority result.
func Of(err error) Outcome {
	if err == nil {
		return Committed
	}
	var outcomeErr *Error
	if errors.As(err, &outcomeErr) {
		return outcomeErr.Outcome
	}
	return NotAuthoritative
}

// Join preserves the current authority result while adding a cleanup error.
func Join(outcome Outcome, err, cleanupErr error) error {
	if outcome == NotAuthoritative {
		return errors.Join(err, cleanupErr)
	}
	var outcomeErr *Error
	if errors.As(err, &outcomeErr) {
		err = outcomeErr.Err
	}
	return Wrap(outcome, errors.Join(err, cleanupErr))
}
