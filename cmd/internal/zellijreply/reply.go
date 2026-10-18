// Package zellijreply validates native observations used to authorize actions.
// Unlike zellijpane.Parse, it never turns missing decision facts into defaults.
package zellijreply

import (
	"context"
	"errors"
	"time"
)

// ErrMismatch means two valid observations disagree on the caller's facts.
var ErrMismatch = errors.New("Zellij reply observations disagree")

// AdmissionContext gives all observations of one operation a shared budget.
// context.WithTimeout preserves an earlier parent deadline.
func AdmissionContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}

// Confirm reads exactly twice after a valid first observation. The pure project
// function validates and canonicalizes only decision facts. No retry is made.
// Success returns the second observation for planning without a third read.
func Confirm(ctx context.Context, read func(context.Context) ([]byte, error), project func([]byte) (string, error)) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	first, err := read(ctx)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	candidate, err := project(first)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	second, err := read(ctx)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	confirmed, err := project(second)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if candidate != confirmed {
		return nil, ErrMismatch
	}
	return second, nil
}
