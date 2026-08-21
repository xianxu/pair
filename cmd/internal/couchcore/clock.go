package couchcore

import "time"

// Clock is injected everywhere; the domain never calls time.Now directly.
// Controllable time is treated as architecture, not a test convenience.
type Clock interface{ Now() time.Time }

type SystemClock struct{}

var _ Clock = SystemClock{}

func (SystemClock) Now() time.Time { return time.Now() }

// FixedClock makes recorded timestamps assertable.
type FixedClock struct{ T time.Time }

var _ Clock = FixedClock{}

func (c FixedClock) Now() time.Time { return c.T }
