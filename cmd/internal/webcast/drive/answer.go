package drive

import (
	"errors"
	"net/http"
	"time"
)

// The reverse channel's fixed parts (#466; #467 serves them).
const (
	// SeatHeader carries a seat's key on drive requests.
	SeatHeader = "X-Couch-Seat"
	// DefaultGrace is how long a seat outlives its last stream.
	DefaultGrace = 10 * time.Second
)

// What a viewer is told: fixed sentences that never echo the request.
const (
	sayNoSeat    = "Join the cast again."
	sayNotDriver = "Ask the operator for control first."
	sayStale     = "The page changed. Try again."
	sayNotLive   = "The cast is paused."
	sayDialog    = "The page is showing a dialog; the operator must answer it."
	sayRefused   = "That isn't allowed while driving."
	sayControl   = "Control has already changed."
	sayBad       = "That request isn't valid."
	sayEnded     = "The cast has ended."
)

// Answer is a drive route's response to err: its status and the sentence
// the viewer's shell shows. nil is 204.
func Answer(err error) (int, string) {
	switch {
	case err == nil:
		return http.StatusNoContent, ""
	case errors.Is(err, ErrNoSeat):
		return http.StatusUnauthorized, sayNoSeat
	case errors.Is(err, ErrNotDriver):
		return http.StatusForbidden, sayNotDriver
	case errors.Is(err, ErrStale):
		return http.StatusConflict, sayStale
	case errors.Is(err, ErrNotLive):
		return http.StatusConflict, sayNotLive
	case errors.Is(err, ErrDialog):
		return http.StatusConflict, sayDialog
	case errors.Is(err, ErrRefused):
		return http.StatusUnprocessableEntity, sayRefused
	case errors.Is(err, ErrControlOp):
		return http.StatusConflict, sayControl
	case errors.Is(err, ErrBatchShape), errors.Is(err, ErrBatchLimit):
		return http.StatusBadRequest, sayBad
	case errors.Is(err, ErrClosed):
		return http.StatusGone, sayEnded
	}
	return http.StatusInternalServerError, "Something went wrong."
}
