package webcast

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	dom "github.com/xianxu/pair/cmd/internal/webcast/submission"
)

const MaxSubmissionRecordingBytes int64 = 128 << 20

func (c *Cast) SubmissionSelection() (board.Selection, error) {
	if c.board == nil || c.recDir == "" {
		return board.Selection{}, ErrNotRecording
	}
	select {
	case <-c.done:
		return board.Selection{}, ErrEnded
	default:
	}
	return c.board.Selection()
}
func (c *Cast) RecordingPath() string {
	if c.recDir == "" {
		return ""
	}
	path, err := filepath.Abs(c.recDir)
	if err != nil {
		return ""
	}
	return path
}

// PrepareSubmission produces evidence only. Publication and delivery are owned
// by the submission store; this operation never resolves notes or sends text.
func (c *Cast) PrepareSubmission(parent context.Context, id string, target core.Target, selected board.Selection, replayBase string, progress func(core.SubmissionProgress)) (core.Submission, map[string][]byte, error) {
	var empty core.Submission
	if c.board == nil || c.recDir == "" {
		return empty, nil, ErrNotRecording
	}
	capture, ok := c.browser.(CropSource)
	if !ok {
		return empty, nil, fmt.Errorf("cast cannot capture submission crops")
	}
	select {
	case <-c.done:
		return empty, nil, ErrEnded
	default:
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return empty, nil, err
	}
	var policy core.OriginPolicy
	if c.readOriginPolicy != nil {
		var notice string
		policy, notice = c.readOriginPolicy()
		if notice != "" && c.logf != nil {
			c.logf("%s", notice)
		}
	}
	if progress != nil {
		progress(core.SubmissionProgress{Phase: "context"})
	}
	recs, err := readSubmissionRecording(ctx, c.RecordingPath())
	if err != nil {
		return empty, nil, err
	}
	sub, boxes, err := dom.Build(recs, selected, id, target, c.RecordingPath(), nil)
	if err != nil {
		return empty, nil, err
	}
	captured, err := capture.CaptureCrops(ctx, selected.Document, boxes, progress)
	if err != nil {
		return empty, nil, err
	}
	if err = ctx.Err(); err != nil {
		return empty, nil, err
	}
	select {
	case <-c.done:
		return empty, nil, ErrEnded
	default:
	}
	if len(captured) != len(sub.Notes) {
		return empty, nil, fmt.Errorf("capture did not return every selected note")
	}
	// Every available crop names the same admitted page. Unknown identity is
	// represented by unavailable evidence, never by pixels with guessed metadata.
	var page *core.PageIdentity
	pageUnavailable := false
	for _, crop := range captured {
		if crop.Unavailable != "" {
			// Off-screen notes lack pixels, not document identity. A page change
			// still cannot be mixed with captures from an admitted document.
			if crop.Unavailable != "outside viewport" {
				pageUnavailable = true
			}
			continue
		}
		if err := crop.Page.Validate(); err != nil {
			return empty, nil, err
		}
		if page == nil {
			value := crop.Page
			page = &value
		} else if *page != crop.Page {
			return empty, nil, fmt.Errorf("capture page identity changed within batch")
		}
	}
	if pageUnavailable && page != nil {
		return empty, nil, fmt.Errorf("capture batch mixes available and unavailable pages")
	}
	sub.Source.Page = page
	pageURL := ""
	if page != nil {
		pageURL = page.URL
	}
	sub.PrivacyDecision = policy.Decide(pageURL)

	total := 0
	for _, crop := range captured {
		if len(crop.PNG) > core.MaxPNGBytes || len(crop.PNG) > core.MaxBundleBytes-total {
			return empty, nil, fmt.Errorf("submission crops exceed bundle budget")
		}
		total += len(crop.PNG)
	}
	crops := make(map[string][]byte)
	for i, crop := range captured {
		evidence := core.Crop{Box: crop.Box, CapturedAt: crop.CapturedAt, Document: crop.Document,
			Unavailable: crop.Unavailable, Settlement: crop.Settlement, ClippedToViewport: crop.ClippedToViewport}
		if crop.Unavailable != "" {
			if len(crop.PNG) != 0 {
				return empty, nil, fmt.Errorf("unavailable crop has pixels")
			}
			if err = core.ValidateCrop(evidence); err != nil {
				return empty, nil, err
			}
		} else {
			evidence.File = fmt.Sprintf("crop-%d.png", i)
			// Validate finite geometry before Contains performs coordinate arithmetic.
			if err = core.ValidateCrop(evidence); err != nil {
				return empty, nil, err
			}
			requested := boxes[i].Box
			if crop.Document != selected.Document || !requested.Contains(crop.Box) || crop.ClippedToViewport != (crop.Box != requested) {
				return empty, nil, fmt.Errorf("crop capture identity or viewport clipping changed")
			}
			if err = core.ValidatePNG(crop.PNG); err != nil {
				return empty, nil, err
			}
			crops[evidence.File] = crop.PNG
		}
		sub.Notes[i].Crop = evidence
		sub.Notes[i].BoxText = crop.BoxText
		if crop.Unavailable == "" && crop.BoxText.Region != boxes[i].Region {
			return empty, nil, fmt.Errorf("box text region changed")
		}
		if replayBase != "" {
			sub.Notes[i].Replay = ReplayLink(replayBase, sub.Source.RecordingID, sub.Notes[i].T)
		}
	}
	if err = core.Validate(sub); err != nil {
		return empty, nil, err
	}
	manifest, err := json.Marshal(sub)
	if err != nil {
		return empty, nil, err
	}
	if len(manifest) > core.MaxBundleBytes-total {
		return empty, nil, fmt.Errorf("submission exceeds 64 MiB")
	}
	return sub, crops, nil
}

// ResolveSubmission acknowledges only the frozen generation, after dispatch.
// An ended cast or a different recording/document is historical evidence and
// requires no mutation of the current board.
func (c *Cast) ResolveSubmission(ctx context.Context, sub core.Submission) error {
	if sub.Delivery != core.Dispatched && sub.Delivery != core.Resolved {
		return fmt.Errorf("submission has not been dispatched")
	}
	if err := core.Validate(sub); err != nil {
		return err
	}
	if c.board == nil || c.recDir == "" || sub.Source.Kind != "dom" || sub.Source.RecordingPath != c.RecordingPath() {
		return nil
	}
	select {
	case <-c.done:
		return nil
	default:
	}
	selected := board.Selection{Document: sub.Source.Document, TabID: sub.Source.TabID}
	for _, note := range sub.Notes {
		selected.Notes = append(selected.Notes, board.SelectedNote{ID: note.ID, Revision: note.Revision})
	}
	err := c.board.ResolveSelection(ctx, selected)
	if errors.Is(err, board.ErrClosed) {
		return nil
	}
	return err
}

// readSubmissionRecording checks every known input's size before allocating
// its bytes. Open handles and bounded reads also cover growth after stat.
func readSubmissionRecording(ctx context.Context, path string) ([]recording.Record, error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	type input struct {
		name string
		file *os.File
	}
	var inputs []input
	defer func() {
		for _, in := range inputs {
			in.file.Close()
		}
	}()
	var size int64
	for _, name := range []string{recording.FileEvents, recording.FileNotes, recording.FilePointer, recording.FileFrames} {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		// os.Root contains traversal; additionally refuse internal aliases.
		missing := false
		parts := strings.Split(name, "/")
		for i := range parts {
			info, e := root.Lstat(strings.Join(parts[:i+1], "/"))
			if os.IsNotExist(e) {
				missing = true
				break
			}
			if e != nil {
				return nil, e
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("symlink in recording input")
			}
		}
		if missing {
			continue
		}
		file, e := root.Open(name)
		if e != nil {
			return nil, e
		}
		inputs = append(inputs, input{name, file})
		info, e := file.Stat()
		if e != nil {
			return nil, e
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("recording input is not a regular file")
		}
		if info.Size() > MaxSubmissionRecordingBytes-size {
			return nil, fmt.Errorf("recording exceeds 128 MiB submission input budget")
		}
		size += info.Size()
	}
	files := map[string][]recording.Located{}
	remaining := MaxSubmissionRecordingBytes
	for _, in := range inputs {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		data, e := io.ReadAll(io.LimitReader(in.file, remaining+1))
		if e != nil {
			return nil, e
		}
		if int64(len(data)) > remaining {
			return nil, fmt.Errorf("recording exceeds 128 MiB submission input budget")
		}
		remaining -= int64(len(data))
		// Match the live reader: complete malformed lines refuse; torn tails wait.
		data = data[:bytes.LastIndexByte(data, '\n')+1]
		// Decode incrementally: DecodeFile splits its input, and a file of
		// blank lines must not allocate a slice entry for every byte.
		lineNumber := 0
		for len(data) > 0 {
			line, rest, _ := bytes.Cut(data, []byte{'\n'})
			data = rest
			lineNumber++
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			located, bad := recording.DecodeFile(in.name, line)
			if len(bad) > 0 {
				bad[0].Line = lineNumber
				return nil, errors.New(bad[0].String())
			}
			for i := range located {
				located[i].Line = lineNumber
			}
			files[in.name] = append(files[in.name], located...)
		}
	}
	return recording.Records(recording.Merge(files)), nil
}
