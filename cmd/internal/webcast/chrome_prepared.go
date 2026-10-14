package webcast

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func (c *Chrome) WritePrepared(ctx context.Context, admission board.AnnotationAdmission, reservation board.NoteReservation, check func() error, settle func(board.OperatorOutcome, error)) error {
	if c.rec == nil {
		return ErrNotRecording
	}
	return c.rec.WritePrepared(ctx, admission, reservation, check, settle)
}

func (c *Chrome) ObserveNote(ctx context.Context, id string) (*recording.Note, bool, error) {
	if c.rec == nil {
		return nil, false, ErrNotRecording
	}
	return c.rec.ObserveNote(ctx, id)
}
