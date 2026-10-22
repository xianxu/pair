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
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
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
	// Once Claim starts, even an error may leave durable transfer intent. Keep
	// the submission identity so the publisher can reconcile absence or uncertainty.
	crops := map[string][]byte{}
	var assets map[string]notedraft.Asset
	var ids []string
	for _, n := range sub.Notes {
		if n.CaptureID != "" {
			ids = append(ids, n.CaptureID)
		}
	}
	if c.noteDrafts != nil && len(ids) > 0 {
		assets, err = c.noteDrafts.Claim(ctx, id, ids)
		if err != nil {
			return sub, crops, err
		}
	}
	sub.PrivacyDecision = core.PrivacyDecision{Privacy: core.Shareable}
	total := 0
	for i := range sub.Notes {
		if err = ctx.Err(); err != nil {
			return sub, crops, err
		}
		note := &sub.Notes[i]
		if asset, ok := assets[note.CaptureID]; ok {
			evidence := asset.Evidence
			if evidence.Crop.Unavailable != "" {
				if len(asset.PNG) != 0 {
					return sub, crops, fmt.Errorf("unavailable stored crop has pixels")
				}
				note.Crop = evidence.Crop
				note.BoxText = evidence.BoxText
			} else {
				evidence.Crop.File = fmt.Sprintf("crop-%d.png", i)
				if err = validateStoredNoteAsset(asset, *note, boxes[i].Region, evidence.Crop); err != nil {
					return sub, crops, err
				}
				if len(asset.PNG) > core.MaxPNGBytes || len(asset.PNG) > core.MaxBundleBytes-total {
					return sub, crops, fmt.Errorf("submission crops exceed bundle budget")
				}
				total += len(asset.PNG)
				page := evidence.Page
				note.Page = &page
				note.Crop = evidence.Crop
				note.BoxText = evidence.BoxText
				crops[note.Crop.File] = asset.PNG
			}
			if asset.Prepared.Document > 0 {
				if asset.Prepared.Document != note.SourceDocument {
					return sub, crops, fmt.Errorf("stored capture belongs to another note document")
				}
				drawNote, contextErr := dom.ContextAt(recs, *note, asset.Prepared.T)
				if contextErr != nil {
					return sub, crops, contextErr
				}
				*note = drawNote
			}
		}
		pageURL := ""
		if note.Page != nil {
			pageURL = note.Page.URL
		}
		note.PrivacyDecision = policy.Decide(pageURL)
		if note.Privacy != core.Shareable {
			sub.Privacy = core.Private
		}
		if replayBase != "" {
			note.Replay = ReplayLink(replayBase, sub.Source.RecordingID, note.T)
		}
	}
	if err = c.appendSubmissionSketches(ctx, &sub, crops, recs, selected, policy, &total, progress); err != nil {
		return sub, crops, err
	}
	if err = core.Validate(sub); err != nil {
		return sub, crops, err
	}
	manifest, err := json.Marshal(sub)
	if err != nil {
		return sub, crops, err
	}
	if len(manifest) > core.MaxBundleBytes-total {
		return sub, crops, fmt.Errorf("submission exceeds 64 MiB")
	}
	return sub, crops, nil
}

func validateStoredNoteAsset(asset notedraft.Asset, note core.Note, region recording.Box, crop core.Crop) error {
	if err := asset.Evidence.Page.Validate(); err != nil {
		return err
	}
	if err := core.ValidateCrop(crop); err != nil {
		return err
	}
	if err := core.ValidateBoxText(asset.Evidence.BoxText); err != nil {
		return err
	}
	if crop.Document != note.SourceDocument || asset.Prepared.Document != note.SourceDocument || asset.Prepared.Region != region || asset.Evidence.BoxText.Region != region {
		return fmt.Errorf("stored crop identity or text region changed")
	}
	if asset.Prepared.At.Route != "" && asset.Evidence.Page.URL != asset.Prepared.At.Route {
		return fmt.Errorf("stored capture route differs from original anchor")
	}
	if !region.Contains(crop.Box) || crop.ClippedToViewport != (crop.Box != region) {
		return fmt.Errorf("stored crop viewport clipping changed")
	}
	return core.ValidatePNG(asset.PNG)
}

// ReconcilePreparedSubmission applies independently confirmed publication
// evidence; errors and unknown outcomes never release a claimed source.
func (c *Cast) ReconcilePreparedSubmission(ctx context.Context, sub core.Submission, observation core.PublicationObservation) error {
	if c.noteDrafts == nil {
		return nil
	}
	if sub.Source.Kind != "dom" || sub.Source.RecordingPath != c.RecordingPath() {
		return fmt.Errorf("publication belongs to another recording")
	}
	return c.noteDrafts.ReconcilePublication(ctx, sub, observation)
}

// ResolveSubmission acknowledges only the frozen generation, after dispatch.
// An ended cast or a different recording requires no mutation. Historical
// documents in this recording remain eligible for conditional resolution.
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
	if len(selected.Notes) == 0 {
		return nil
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

func (c *Cast) ReconcilePendingSubmissions(ctx context.Context, observe func(context.Context, string) (core.Submission, core.PublicationObservation, error)) error {
	if c.noteDrafts == nil {
		return nil
	}
	return c.noteDrafts.ReconcilePending(ctx, observe)
}

// appendSubmissionSketches shares the already-read recording and immutable
// selected prefix with one batch renderer. It never captures the live browser.
func (c *Cast) appendSubmissionSketches(ctx context.Context, sub *core.Submission, files map[string][]byte, recs []recording.Record, frozen board.Selection, policy core.OriginPolicy, total *int, progress func(core.SubmissionProgress)) error {
	selected := frozen.Sketches
	if len(selected) == 0 {
		return nil
	}
	if len(selected) > core.MaxSubmissionSketches {
		return fmt.Errorf("submission sketch batch exceeds %d", core.MaxSubmissionSketches)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	wanted := make([]query.SketchSelection, len(selected))
	for i, v := range selected {
		wanted[i] = query.SketchSelection{ID: v.ID, NextSequence: v.NextSequence, Digest: v.Digest}
	}
	if len(recs) == 0 {
		return fmt.Errorf("submission recording has no sketch history")
	}
	views, err := query.New(recs, notes.Structural{}).SketchesAt(recs[len(recs)-1].Time(), wanted...)
	if err != nil {
		return err
	}
	byID := make(map[string]query.SketchView, len(views))
	for _, view := range views {
		byID[view.History.Begin.ID] = view
	}
	ordered := make([]query.SketchView, len(selected))
	for i, v := range selected {
		view, ok := byID[v.ID]
		if !ok || view.History.Label == "" || len(view.History.Visible) == 0 || (view.History.Phase != "complete" && view.History.Phase != "interrupted") {
			return fmt.Errorf("selected sketch %q is not a finished visible shape", v.ID)
		}
		ordered[i] = view
	}
	batch := sketchimage.PageBatch{Batch: sketchimage.Batch{RecordingDir: c.RecordingPath(), Records: recs, Sketches: ordered}, Finishes: make([]query.SketchSourceFacts, len(selected))}
	for i, v := range selected {
		if v.Source != ordered[i].Source {
			return fmt.Errorf("selected sketch %q source witness changed", v.ID)
		}
		batch.Finishes[i] = v.Finish
		if i == 0 || v.Finish.RecordIndex > batch.Scene.RecordIndex {
			batch.Scene = sketchimage.PageScene{RecordingID: sub.Source.RecordingID, SketchSourceFacts: v.Finish}
		}
	}
	if (frozen.CurrentPage == nil) != (frozen.Current == nil) {
		return fmt.Errorf("incomplete current submission view")
	}
	if frozen.CurrentPage != nil {
		batch.Scene = *frozen.CurrentPage
		batch.Current = frozen.Current
	}
	plan, err := sketchimage.ValidatePageBatch(batch)
	if err != nil {
		return err
	}
	renderer := c.sketchPages
	if renderer == nil {
		bin, e := FindChrome(os.Getenv, execLookPath, fileExists)
		if e != nil {
			return e
		}
		renderer = NewSketchPageImageRenderer(bin)
	}
	if progress != nil {
		progress(core.SubmissionProgress{Phase: "rendering-sketches", Total: len(ordered)})
	}
	image, err := renderer.RenderPage(ctx, batch)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = sketchimage.ValidatePageImage(batch, image); err != nil {
		return err
	}
	if len(image.PNG) > core.MaxBundleBytes-*total {
		return fmt.Errorf("submission images exceed bundle budget")
	}
	*total += len(image.PNG)
	pageDecision := policy.Decide(batch.Scene.URL)
	sub.SketchPage = &core.SketchPage{PrivacyDecision: pageDecision, File: "sketch-page.png", Provenance: "replay-rendered", Source: batch.Scene, Region: plan.Region, Scale: plan.Scale, Width: plan.Width, Height: plan.Height, Clipped: image.Clipped, Limitations: append([]string(nil), image.Limitations...)}
	if batch.Current != nil {
		sub.SketchPage.Provenance = "replay-rendered-current-view"
		sub.SketchPage.Current = batch.Current
	}
	if pageDecision.Privacy != core.Shareable {
		sub.Privacy = core.Private
	}
	for i, v := range ordered {
		decision := policy.Decide(v.Source.URL)
		if decision.Privacy != core.Shareable {
			sub.Privacy = core.Private
			sub.SketchPage.Privacy = core.Private
		}
		// Reuse the historical path validator/extractor. Its individual crop is not
		// part of the shared representation; original vectors remain unscaled.
		original, err := sketchimage.Plan(v.History, v.Source.VP, v.Source.Scroll, sketchimage.DefaultLimits())
		if err != nil {
			return err
		}
		for j := range original.Paths {
			original.Paths[j].At.Sel = ""
		}
		finish := core.SketchSource{SketchSourceFacts: batch.Finishes[i], Provenance: "recorded-dom"}
		item := core.Sketch{PrivacyDecision: decision, ID: v.History.Begin.ID, RecordingID: sub.Source.RecordingID, NextSequence: v.History.NextSequence, Digest: v.History.Receipts[len(v.History.Receipts)-1].Digest, Author: v.History.Begin.Author, Note: v.History.Label, HistoryT: v.HistoryT, HistoryRecordIndex: v.HistoryRecordIndex, Phase: v.History.Phase, Reason: v.History.Reason, Source: core.SketchSource{SketchSourceFacts: v.Source, Provenance: "recorded-dom"}, Elements: v.Elements, Index: i + 1, Paths: original.Paths, Finish: &finish}
		if err = core.ValidateSketch(item); err != nil {
			return err
		}
		sub.Sketches = append(sub.Sketches, item)
	}
	// A contributing private sketch makes the composite private. An origin's
	// shareable rule cannot certify that composite, so retain the page origin
	// while clearing that grant instead of attributing another origin's rule.
	if sub.SketchPage.Privacy != pageDecision.Privacy {
		sub.SketchPage.PrivacyRule = core.PolicyEvidence{}
	}
	files[sub.SketchPage.File] = append([]byte(nil), image.PNG...)
	return nil
}
