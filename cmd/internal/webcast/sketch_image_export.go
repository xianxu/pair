package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

// ExportSketchImage publishes a private, complete image/facts directory. The
// renderer is shared with submission preparation; the query never dispatches.
func ExportSketchImage(ctx context.Context, renderer sketchimage.ImageRenderer, request query.SketchImageRequest) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}
	if renderer == nil {
		return errors.New("sketch image: renderer unavailable")
	}
	recordingDir, err := filepath.EvalSymlinks(request.RecordingDir)
	if err != nil {
		return err
	}
	recordingDir, err = filepath.Abs(recordingDir)
	if err != nil {
		return err
	}
	recordingInfo, err := os.Stat(recordingDir)
	if err != nil {
		return err
	}
	output, err := filepath.Abs(request.OutputDir)
	if err != nil {
		return err
	}
	parentPath, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return err
	}
	output = filepath.Join(parentPath, filepath.Base(output))
	if err := sketchExportOutsideRecording(parentPath, recordingInfo); err != nil {
		return err
	}
	rel, err := filepath.Rel(recordingDir, output)
	if err != nil {
		return err
	}
	if rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("sketch image: output must be outside the recording")
	}
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		return err
	}
	defer parent.Close()
	name := filepath.Base(output)
	if _, err := parent.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("sketch image: output already exists or is inaccessible: %s", output)
	}
	b := sketchimage.Batch{RecordingDir: request.RecordingDir, Records: request.Records, Sketches: []query.SketchView{request.Sketch}}
	plans, err := sketchimage.ValidateBatch(b)
	if err != nil {
		return err
	}
	images, err := renderer.Render(ctx, b)
	if err != nil {
		return err
	}
	if err = sketchimage.ValidateImages(plans, images); err != nil {
		return err
	}
	im := images[0]
	metadata, err := json.MarshalIndent(struct {
		Provenance  string                 `json:"provenance"`
		Sketch      query.SketchView       `json:"sketch"`
		Image       string                 `json:"image"`
		Transform   sketchimage.RenderPlan `json:"transform"`
		Limitations []string               `json:"limitations"`
	}{"replay-rendered", request.Sketch, "sketch.png", im.Plan, im.Limitations}, "", "  ")
	if err != nil {
		return err
	}
	if len(metadata) > sketchimage.MaxBatchImageBytes-len(im.PNG) {
		return errors.New("sketch image: metadata and image budget exceeded")
	}
	token, err := broadcast.NewToken()
	if err != nil {
		return err
	}
	stage := ".sketch-image-" + token
	if err = parent.Mkdir(stage, 0700); err != nil {
		return err
	}
	published := false
	defer func() {
		if published {
			return
		}
		if e := parent.RemoveAll(stage); e != nil {
			err = errors.Join(err, e)
		}
	}()
	for _, file := range []struct {
		name string
		body []byte
	}{{"sketch.png", im.PNG}, {"metadata.json", metadata}} {
		f, e := parent.OpenFile(filepath.Join(stage, file.name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		_, writeErr := f.Write(file.body)
		syncErr := f.Sync()
		closeErr := f.Close()
		if e = errors.Join(writeErr, syncErr, closeErr); e != nil {
			return e
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	parentFile, err := parent.Open(".")
	if err != nil {
		return err
	}
	defer parentFile.Close()
	opened, err := parentFile.Stat()
	if err != nil {
		return err
	}
	current, err := os.Stat(parentPath)
	if err != nil {
		return err
	}
	if !os.SameFile(opened, current) {
		return errors.New("sketch image: output parent moved during preparation")
	}
	if err := sketchExportOutsideRecording(parentPath, recordingInfo); err != nil {
		return err
	}
	// Kernel no-replace arbitration protects an export created during rendering.
	err = publishSketchImageExclusive(parentFile, stage, name)
	published = err == nil
	return err
}

// SameFile also detects case aliases on case-insensitive filesystems; lexical
// relative paths and EvalSymlinks alone do not establish physical ancestry.
func sketchExportOutsideRecording(parent string, recording os.FileInfo) error {
	for {
		st, err := os.Stat(parent)
		if err != nil {
			return err
		}
		if os.SameFile(st, recording) {
			return errors.New("sketch image: output must be outside the recording")
		}
		next := filepath.Dir(parent)
		if next == parent {
			return nil
		}
		parent = next
	}
}
