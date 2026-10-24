package webcast

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/drive"
)

func TestDriveWarningRoutesKeepBusyInformational(t *testing.T) {
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, drive.ErrInput, errors.Join(drive.ErrInput, context.DeadlineExceeded), errors.New("unexpected failure")} {
		t.Run(err.Error(), func(t *testing.T) {
			logged, warned := 0, 0
			d := &driveRoutes{logf: func(string, ...any) { logged++ }, warnf: func(string, ...any) { warned++ }}
			for range 2 {
				d.answer(httptest.NewRecorder(), err)
			}
			if !errors.Is(err, drive.ErrInput) && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				if logged != 1 || warned != 0 {
					t.Fatalf("busy logs=%d warnings=%d", logged, warned)
				}
			} else if logged != 0 || warned != 1 {
				t.Fatalf("failure logs=%d warnings=%d", logged, warned)
			}
		})
	}
}
