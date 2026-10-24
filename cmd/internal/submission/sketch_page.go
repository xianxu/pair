package submission

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
	"unicode/utf8"
)

// SketchPage owns the single full-viewport image for an indexed shared batch.
// Original ink and original begin context remain on each Sketch.
type SketchPage struct {
	PrivacyDecision
	File        string                   `json:"file"`
	Provenance  string                   `json:"provenance"`
	Source      sketchimage.PageScene    `json:"source"`
	Region      recording.Box            `json:"region"`
	Scale       float64                  `json:"scale"`
	Width       int                      `json:"width"`
	Height      int                      `json:"height"`
	Clipped     bool                     `json:"clipped"`
	Limitations []string                 `json:"limitations,omitempty"`
	Current     *sketchimage.CurrentView `json:"current,omitempty"`
}

func validateSourcePageLoad(s SketchSource) error {
	if err := s.PageLoad.Validate(); err != nil {
		return err
	}
	if s.PageLoad != (query.PageLoadFacts{}) && s.PageLoad.Target != s.Tab {
		return fmt.Errorf("page load target differs from source tab")
	}
	return nil
}

func validateSketchSource(s SketchSource) error {
	if err := validateSourcePageLoad(s); err != nil {
		return err
	}
	if s.Provenance != "recorded-dom" || s.T < 0 || uint64(s.T) > recording.MaxSketchInteger || s.RecordIndex < 0 || s.Doc < 1 || !sketchID(s.Tab) || len(s.URL) > MaxPageURLBytes || len(s.Title) > MaxPageTitleBytes || !utf8.ValidString(s.URL) || !utf8.ValidString(s.Title) {
		return fmt.Errorf("invalid sketch source facts")
	}
	_, err := sketchimage.PlanPage(sketchimage.PageScene{RecordingID: "validation", SketchSourceFacts: s.SketchSourceFacts})
	return err
}
func ValidateSketchPage(s Submission) error {
	p := s.SketchPage
	if p == nil {
		for _, sk := range s.Sketches {
			if sk.Image == nil {
				return fmt.Errorf("shared sketch requires page")
			}
		}
		return nil
	}
	if s.Version != 3 || s.Source.Kind != "dom" || len(s.Sketches) < 1 || len(s.Sketches) > MaxSubmissionSketches || p.Source.RecordingID != s.Source.RecordingID || p.File != "sketch-page.png" || (p.Provenance != "replay-rendered" || p.Current != nil) && (p.Provenance != "replay-rendered-current-view" || p.Current == nil) {
		return fmt.Errorf("invalid shared sketch page identity")
	}
	if err := validateSketchSource(SketchSource{SketchSourceFacts: p.Source.SketchSourceFacts, Provenance: "recorded-dom"}); err != nil {
		return err
	}
	plan, err := sketchimage.PlanPage(p.Source)
	if err != nil {
		return err
	}
	if p.Region != plan.Region || p.Scale != plan.Scale || p.Width != plan.Width || p.Height != plan.Height {
		return fmt.Errorf("shared page transform mismatch")
	}
	if err = p.PrivacyDecision.Validate(); err != nil {
		return err
	}
	if p.Privacy == "" || p.PrivacyOrigin != OriginOf(p.Source.URL) || p.PrivacyOrigin == "" && p.Privacy != Private || s.Privacy == Shareable && p.Privacy != Shareable {
		return fmt.Errorf("shared page privacy mismatch")
	}
	if len(p.Limitations) > 32 {
		return fmt.Errorf("too many page limitations")
	}
	for _, v := range p.Limitations {
		if len(v) > 256 {
			return fmt.Errorf("page limitation exceeds bound")
		}
	}
	latest := -1
	for i, sk := range s.Sketches {
		if sk.Image != nil || sk.Index != i+1 || sk.Finish == nil {
			return fmt.Errorf("invalid indexed shared sketch")
		}
		finish := sketchimage.PageScene{RecordingID: sk.RecordingID, SketchSourceFacts: sk.Finish.SketchSourceFacts}
		if p.Current == nil && !sketchimage.SamePageView(p.Source, finish) {
			return fmt.Errorf("incompatible sketch finish view")
		}
		if p.Privacy == Shareable && sk.Privacy != Shareable {
			return fmt.Errorf("page privacy is less restrictive than sketch")
		}
		if latest < 0 || sk.Finish.RecordIndex > s.Sketches[latest].Finish.RecordIndex {
			latest = i
		}
	}
	if p.Current == nil && p.Source.SketchSourceFacts != s.Sketches[latest].Finish.SketchSourceFacts {
		return fmt.Errorf("page is not latest chosen finish")
	}
	if p.Current != nil {
		paths := map[string][]sketchimage.Path{}
		for _, sk := range s.Sketches {
			if p.Source.RecordIndex < sk.HistoryRecordIndex || p.Source.T < sk.HistoryT {
				return fmt.Errorf("current page precedes selected history")
			}
			paths[sk.ID] = sk.Paths
		}
		if err := sketchimage.ValidateCurrentView(p.Source, *p.Current, paths); err != nil {
			return err
		}
	}
	return nil
}
