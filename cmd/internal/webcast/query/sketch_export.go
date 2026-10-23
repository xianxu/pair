package query

import (
	"context"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// SketchImageRequest carries the already-read recording and exact historical
// source boundary. Exporters render this input rather than rereading a live file.
// Input records are read-only; the selected view owns its projected values.
type SketchImageRequest struct {
	RecordingDir string
	OutputDir    string
	Records      []recording.Record
	Sketch       SketchView
}

// SketchImageExporter returns nil only after publishing the complete evidence.
// It owns output path admission, rendering bounds and cancellation cleanup.
type SketchImageExporter func(context.Context, SketchImageRequest) error

type RunOptions struct {
	Context           context.Context
	Link              Linker
	ExportSketchImage SketchImageExporter
}

func exportSketchImage(q *Querier, a args, options RunOptions) error {
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if options.ExportSketchImage == nil {
		return fmt.Errorf("query: sketch image exporter is unavailable")
	}
	if !a.atSet {
		a.at = q.End()
	}
	views, err := q.SketchesAt(a.at)
	if err != nil {
		return err
	}
	for _, view := range views {
		if view.History.Begin.ID == a.sketch {
			if err := ctx.Err(); err != nil {
				return err
			}
			return options.ExportSketchImage(ctx, SketchImageRequest{RecordingDir: a.dir, OutputDir: a.out, Records: q.recs, Sketch: view})
		}
	}
	return fmt.Errorf("query: sketch %q missing at history cutoff", a.sketch)
}
