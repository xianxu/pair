package drive

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestAnswer(t *testing.T) {
	for _, c := range []struct {
		err    error
		status int
	}{
		{nil, http.StatusNoContent},
		{ErrNoSeat, http.StatusUnauthorized},
		{ErrNotDriver, http.StatusForbidden},
		{fmt.Errorf("x: %w", ErrStale), http.StatusConflict},
		{ErrNotLive, http.StatusConflict},
		{fmt.Errorf("dispatch: %w", ErrDialog), http.StatusConflict},
		{fmt.Errorf("%w: paste", ErrRefused), http.StatusUnprocessableEntity},
		{fmt.Errorf("%w: twice", recording.ErrControl), http.StatusConflict},
		{ErrBatchLimit, http.StatusBadRequest},
		{ErrClosed, http.StatusGone},
		{fmt.Errorf("%w: Input.dispatchMouseEvent: closed", ErrInput), http.StatusBadGateway},
		{context.Canceled, http.StatusServiceUnavailable},
		{fmt.Errorf("ask: %w", context.DeadlineExceeded), http.StatusServiceUnavailable},
		{errors.New("disk full"), http.StatusInternalServerError},
	} {
		status, say := Answer(c.err)
		if status != c.status {
			t.Errorf("%v: status %d, want %d", c.err, status, c.status)
		}
		if c.err != nil && (say == "" || say == c.err.Error()) {
			t.Errorf("%v: says %q", c.err, say)
		}
	}
}

// The driver is told why a dialog stops their input, in the sentence the
// TL fixed (#467), not a bare failure.
func TestAnswerDialog(t *testing.T) {
	if _, say := Answer(ErrDialog); say != "The page is showing a dialog; the operator must answer it." {
		t.Fatalf("says %q", say)
	}
}
