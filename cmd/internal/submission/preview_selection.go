package submission

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"slices"
)

// SelectSketchPreview is the single oldest-first batch policy used by private
// preview and preparation. Only exact dispatched/resolved versions are omitted.
func SelectSketchPreview(candidates []PreviewItem, delivered map[PreviewItem]struct{}, stamp PreviewStamp) PreviewSelection {
	out := PreviewSelection{Stamp: stamp}
	for _, v := range candidates {
		if _, sent := delivered[v]; sent {
			continue
		}
		if len(out.Items) < 8 {
			out.Items = append(out.Items, v)
		} else {
			out.Remaining++
		}
	}
	return out
}

// SelectFeedbackPreview freezes all displayed note revisions and chooses the
// oldest compatible group of unsent sketches. Geometry never changes a delivery
// key; other views and overflow remain explicitly pending.
func SelectFeedbackPreview(candidates []PreviewSketchCandidate, notes []PreviewNote, delivered map[PreviewItem]struct{}, stamp PreviewStamp) (PreviewSelection, error) {
	out, _, err := selectFeedbackPreview(candidates, notes, delivered, stamp)
	return out, err
}

// selectFeedbackPreview returns its inspected-candidate count with the result,
// so growth checks exercise the production loop rather than a timing proxy.
func selectFeedbackPreview(candidates []PreviewSketchCandidate, notes []PreviewNote, delivered map[PreviewItem]struct{}, stamp PreviewStamp) (PreviewSelection, int, error) {
	visits := 0
	out := PreviewSelection{Stamp: stamp, Notes: slices.Clone(notes)}
	var group *sketchimage.PageScene
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		visits++
		if _, sent := delivered[c.Item]; sent {
			continue
		}
		if c.Item.RecordingID == "" || c.Item.SketchID == "" || seen[c.Item.SketchID] || c.Source.Doc < 1 || c.Source.Tab == "" || c.Source.RecordIndex < 0 || c.Finish.RecordIndex < c.Source.RecordIndex || c.Finish.T < c.Source.T {
			return PreviewSelection{}, visits, fmt.Errorf("feedback preview: missing or ambiguous sketch source/finish")
		}
		seen[c.Item.SketchID] = true
		scene := sketchimage.PageScene{RecordingID: c.Item.RecordingID, SketchSourceFacts: c.Finish}
		if _, err := sketchimage.PlanPage(scene); err != nil {
			return PreviewSelection{}, visits, err
		}
		if group == nil {
			copy := scene
			group = &copy
		}
		if len(out.Items) >= MaxSubmissionSketches || !sketchimage.SamePageView(*group, scene) {
			out.Remaining++
			continue
		}
		out.Items = append(out.Items, c.Item)
		out.Candidates = append(out.Candidates, c)
		if out.Page == nil || scene.RecordIndex > out.Page.RecordIndex {
			copy := scene
			out.Page = &copy
		}
	}
	return out, visits, nil
}
