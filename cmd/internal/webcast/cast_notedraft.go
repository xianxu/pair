package webcast

import (
	"context"
	"fmt"
	"path/filepath"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

type preparedNoteBrowser interface {
	WritePrepared(context.Context, board.AnnotationAdmission, board.NoteReservation, func() error, func(board.OperatorOutcome, error)) error
	ObserveNote(context.Context, string) (*recording.Note, bool, error)
}
type castDraftCapture struct {
	browser func(context.Context) (Browser, error)
}

func (c castDraftCapture) Capture(ctx context.Context, p board.PreparedNote) (notedraft.Evidence, []byte, error) {
	b, err := c.browser(ctx)
	if err != nil {
		return notedraft.Evidence{}, nil, err
	}
	source, ok := b.(CropSource)
	if !ok {
		return notedraft.Evidence{}, nil, ErrNotRecording
	}
	crops, err := source.CaptureCrops(ctx, p.Document, []core.CaptureRegion{{Box: p.Region, Region: p.Region}}, nil)
	if err != nil {
		return notedraft.Evidence{}, nil, err
	}
	if len(crops) != 1 {
		return notedraft.Evidence{}, nil, fmt.Errorf("capture returned %d crops", len(crops))
	}
	v := crops[0]
	if v.Unavailable == "" && v.Document != p.Document {
		return notedraft.Evidence{}, nil, fmt.Errorf("capture document changed")
	}
	if v.Unavailable == "" && p.At.Route != "" && v.Page.URL != p.At.Route {
		return notedraft.Evidence{}, nil, fmt.Errorf("page route changed after note was drawn")
	}
	file := ""
	if v.Unavailable == "" {
		file = "crop-0.png"
	}
	return notedraft.Evidence{Page: v.Page, BoxText: v.BoxText, Crop: core.Crop{File: file, Box: v.Box, Document: v.Document, CapturedAt: v.CapturedAt, Unavailable: v.Unavailable, Settlement: v.Settlement, ClippedToViewport: v.ClippedToViewport}}, v.PNG, nil
}
func newCastNoteDrafts(recDir string, b *board.Board, browser func(context.Context) (Browser, error), observePublication func(context.Context, string) (core.Submission, core.PublicationObservation, error)) (*notedraft.Service, error) {
	path, err := filepath.Abs(recDir)
	if err != nil {
		return nil, err
	}
	return notedraft.New(notedraft.Options{ObservePublication: observePublication, Root: filepath.Dir(path) + ".note-captures", RecordingPath: path, Capture: castDraftCapture{browser}, Release: b.ReleaseNoteReservation,
		Write: func(ctx context.Context, a board.AnnotationAdmission, n board.NoteReservation, check func() error, settle func(board.OperatorOutcome, error)) error {
			raw, err := browser(ctx)
			if err != nil {
				return err
			}
			writer, ok := raw.(preparedNoteBrowser)
			if !ok {
				return ErrNotRecording
			}
			return writer.WritePrepared(ctx, a, n, check, settle)
		}, Observe: func(ctx context.Context, id string) (*recording.Note, bool, error) {
			raw, err := browser(ctx)
			if err != nil {
				return nil, false, err
			}
			writer, ok := raw.(preparedNoteBrowser)
			if !ok {
				return nil, false, ErrNotRecording
			}
			return writer.ObserveNote(ctx, id)
		}})
}

// draftEvidenceSink follows actual recorded transitions after the Board's leaf
// lock has been released. It never infers resolution from current visibility.
type draftEvidenceSink struct {
	logf  func(string, ...any)
	sink  recording.Sink
	notes *notedraft.Service
	ids   map[string]bool
}

func (s *draftEvidenceSink) Put(r recording.Record) error {
	if s.sink != nil {
		if err := s.sink.Put(r); err != nil {
			return err
		}
	}
	switch v := r.(type) {
	case *recording.Note:
		if v.Capture != "" {
			s.ids[v.ID] = true
		}
	case *recording.NoteState:
		if v.State == recording.NoteResolved && s.ids[v.ID] {
			s.resolve(v.ID)
		}
	case *recording.AnnotationClear:
		s.notes.RevokePreparations()
		for id := range s.ids {
			s.resolve(id)
		}
	case *recording.Seg:
		if v.Why != recording.SegResume {
			s.notes.RevokePreparations()
		}
	}
	return nil
}
func (s *draftEvidenceSink) Close() error {
	if s.sink != nil {
		return s.sink.Close()
	}
	return nil
}

// Cleanup failures retain the recorded resolution as durable retry evidence.
// They must not change a successful note write into a recorder failure.
func (s *draftEvidenceSink) resolve(id string) {
	if err := s.notes.Resolve(id); err != nil {
		if s.logf != nil {
			s.logf("capture cleanup deferred: %v", err)
		}
		return
	}
	delete(s.ids, id)
}
