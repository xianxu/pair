package webcast

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"io"
	"os"
	"path/filepath"
)

// Read only the requested bytes. Repeated transfers never load the entire
// asset, and the service separately proves this hash belongs to the source.
func sketchAssetReader(dir string) func(string, int, int) ([]byte, int, error) {
	return func(hash string, offset, length int) ([]byte, int, error) {
		if dir == "" || !recording.ValidHash(hash) || offset < 0 || length <= 0 || length > recording.SketchSourceChunkSize {
			return nil, 0, errors.New("invalid sketch asset chunk")
		}
		f, err := os.Open(filepath.Join(dir, recording.AssetName(hash)))
		if err != nil {
			return nil, 0, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return nil, 0, err
		}
		if !info.Mode().IsRegular() || info.Size() < 0 || uint64(info.Size()) > recording.MaxSketchInteger || int64(offset) > info.Size() {
			return nil, 0, errors.New("invalid sketch asset offset")
		}
		part := make([]byte, min(int64(length), info.Size()-int64(offset)))
		_, err = f.ReadAt(part, int64(offset))
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, 0, err
		}
		return part, int(info.Size()), nil
	}
}

type sketchSourceProvider interface {
	CandidateSketchSource(*recording.SketchBegin, int, int) (*recording.SketchSource, error)
}

// Reuse the actual viewer log. Custom/no-live configurations get one private
// source hub on the same ordered tee, rather than silently losing drawing.
func sketchSourceOutput(sink recording.Sink) (recording.Sink, sketchSourceProvider) {
	if source, ok := sink.(sketchSourceProvider); ok {
		return sink, source
	}
	h := recording.NewHub(0)
	if sink == nil {
		return h, h
	}
	return recording.Tee(sink, h), h
}

type sketchWriter interface {
	WriteSketch(context.Context, board.SketchWrite) (board.SketchWriteResult, error)
	ObserveSketch(context.Context, string, string) (board.SketchObservation, error)
}

func (c *Chrome) WriteSketch(ctx context.Context, in board.SketchWrite) (board.SketchWriteResult, error) {
	if c.rec == nil {
		return board.SketchWriteResult{Outcome: board.SketchRefused, Err: ErrNotRecording}, ErrNotRecording
	}
	return c.rec.WriteSketch(ctx, in)
}
func (c *Chrome) ObserveSketch(ctx context.Context, id, operation string) (board.SketchObservation, error) {
	if c.rec == nil {
		return board.SketchObservation{}, ErrNotRecording
	}
	return c.rec.ObserveSketch(ctx, id, operation)
}
