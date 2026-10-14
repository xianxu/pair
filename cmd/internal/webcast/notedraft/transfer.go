package notedraft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
)

type Asset struct {
	Evidence Evidence
	PNG      []byte
	Prepared board.PreparedNote
}

func unavailableAsset(reason string, p board.PreparedNote) Asset {
	return Asset{Prepared: p, Evidence: Evidence{Crop: core.Crop{Unavailable: reason}, BoxText: core.BoxText{Unavailable: reason}}}
}

// Claim durably pins each source before the caller can copy it into a bundle.
// It never captures the current page as a substitute for historical evidence.
func (s *Service) Claim(ctx context.Context, submissionID string, captureIDs []string) (map[string]Asset, error) {
	if !core.ValidID(submissionID) {
		return nil, errors.New("invalid submission ID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errors.New("cast has ended")
	}
	out := map[string]Asset{}
	for _, id := range captureIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, exists := out[id]; exists {
			continue
		}
		r := s.records[id]
		if r == nil || r.State.Phase == Retired {
			out[id] = unavailableAsset("not captured when drawn", board.PreparedNote{})
			continue
		}
		if r.State.Phase != Attached && r.State.Phase != Bundled && r.State.Phase != Transferring {
			out[id] = unavailableAsset("saved capture is not attached", r.Prepared)
			continue
		}
		if r.PendingSubmission != "" && r.PendingSubmission != submissionID {
			return nil, fmt.Errorf("capture %s has an unresolved publication", id)
		}
		next, _, err := Step(r.State, Event{Kind: Publish})
		if err != nil {
			return nil, err
		}
		copy := *r
		copy.State = next
		copy.PendingSubmission = submissionID
		if err = s.persistLocked(&copy); err != nil {
			return nil, err
		}
		r.State = next
		r.PendingSubmission = submissionID
		asset, err := s.assetLocked(r)
		if err != nil {
			asset = unavailableAsset("saved capture unavailable: "+err.Error(), r.Prepared)
		}
		out[id] = asset
	}
	return out, nil
}
func (s *Service) assetLocked(r *captureRecord) (Asset, error) {
	if r.BundlePath != "" {
		return readBundleAsset(r)
	}
	if r.State.Unavailable != "" {
		return unavailableAsset(r.State.Unavailable, r.Prepared), nil
	}
	if r.Ephemeral {
		return unavailableAsset("not captured when drawn", r.Prepared), nil
	}
	_, png, err := s.disk.read(r.ID)
	if err != nil {
		return Asset{}, err
	}
	if len(png) == 0 {
		return Asset{}, errors.New("draw-time pixels are missing")
	}
	if err = validateCaptureEvidence(r.Evidence, r.Prepared, png); err != nil {
		return Asset{}, err
	}

	return Asset{Evidence: r.Evidence, PNG: png, Prepared: r.Prepared}, nil
}

func openPublished(path string) (core.Submission, *os.Root, error) {
	var sub core.Submission
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || filepath.Base(path) != "submission.json" {
		return sub, nil, errors.New("invalid published manifest path")
	}
	for p := path; ; p = filepath.Dir(p) {
		st, err := os.Lstat(p)
		if err != nil {
			return sub, nil, err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return sub, nil, errors.New("symlink in published manifest path")
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return sub, nil, err
	}
	data, err := readBounded(root, "submission.json", core.MaxBundleBytes)
	if err == nil {
		err = json.Unmarshal(data, &sub)
	}
	if err == nil {
		err = core.Validate(sub)
	}
	if err == nil && sub.ID != filepath.Base(filepath.Dir(path)) {
		err = errors.New("published manifest identity mismatch")
	}
	if err != nil {
		root.Close()
		return sub, nil, err
	}
	return sub, root, nil
}
func noteAsset(root *os.Root, n core.Note, p board.PreparedNote) (Asset, error) {
	asset := Asset{Prepared: p, Evidence: Evidence{Crop: n.Crop, BoxText: n.BoxText}}
	if n.Page != nil {
		asset.Evidence.Page = *n.Page
	}
	if n.Crop.File != "" {
		png, err := readBounded(root, n.Crop.File, MaxPNGBytes)
		if err != nil {
			return Asset{}, err
		}
		if err = core.ValidatePNG(png); err != nil {
			return Asset{}, err
		}
		asset.PNG = png
	}
	return asset, nil
}
func readBundleAsset(r *captureRecord) (Asset, error) {
	sub, root, err := openPublished(r.BundlePath)
	if err != nil {
		return Asset{}, err
	}
	defer root.Close()
	for _, n := range sub.Notes {
		if n.CaptureID == r.ID && n.ID == r.NoteID {
			if n.SourceDocument != r.Prepared.Document {
				return Asset{}, errors.New("published capture document mismatch")
			}
			asset, err := noteAsset(root, n, r.Prepared)
			if err == nil && asset.Evidence.Crop.Unavailable == "" {
				err = validateCaptureEvidence(asset.Evidence, r.Prepared, asset.PNG)
			}
			return asset, err
		}
	}
	return Asset{}, errors.New("published capture reference is missing")
}

// ReconcilePublication consumes the publisher's durability observation. A
// reference is durable before source removal; unknown never releases ownership.
func (s *Service) ReconcilePublication(ctx context.Context, sub core.Submission, obs core.PublicationObservation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !core.ValidID(sub.ID) {
		return errors.New("invalid publication ID")
	}
	if obs.State == core.PublicationUnknown || obs.State == "" {
		return nil
	}
	if obs.State != core.PublicationAbsent && obs.State != core.PublicationConfirmed {
		return errors.New("invalid publication observation")
	}
	var pending []*captureRecord
	for _, r := range s.records {
		if r.PendingSubmission == sub.ID && r.State.Phase == Transferring {
			pending = append(pending, r)
		}
	}
	if obs.State == core.PublicationAbsent {
		for _, r := range pending {
			next, effects, err := Step(r.State, Event{Kind: BundleAbsent})
			if err != nil {
				return err
			}
			copy := *r
			copy.State = next
			copy.PendingSubmission = ""
			if err = s.finishTransferLocked(r, &copy, effects); err != nil {
				return err
			}
		}
		return nil
	}
	if err := core.Validate(sub); err != nil {
		return err
	}
	published, root, err := openPublished(obs.Path)
	if err != nil {
		return err
	}
	defer root.Close()
	published.Delivery = core.Prepared
	published.Superseded = false
	expected := sub
	expected.Delivery = core.Prepared
	expected.Superseded = false
	have, _ := json.Marshal(published)
	want, _ := json.Marshal(expected)
	if !bytes.Equal(have, want) {
		return errors.New("published manifest differs from transfer")
	}
	byCapture := map[string]core.Note{}
	for _, n := range sub.Notes {
		if n.CaptureID != "" {
			if _, exists := byCapture[n.CaptureID]; exists {
				return errors.New("duplicate capture in published submission")
			}
			byCapture[n.CaptureID] = n
		}
	}
	// Validate the whole publication before releasing any claimed source.
	for _, r := range pending {
		if err = ctx.Err(); err != nil {
			return err
		}
		n, exists := byCapture[r.ID]
		if !exists || n.ID != r.NoteID || n.SourceDocument != r.Prepared.Document {
			return errors.New("publication does not match capture owner")
		}
		asset, e := s.assetLocked(r)
		if e != nil {
			asset = unavailableAsset("saved capture unavailable: "+e.Error(), r.Prepared)
		}
		actual, e := noteAsset(root, n, r.Prepared)
		if e != nil {
			return e
		}
		if !sameCapturedAsset(asset, actual) {
			return errors.New("published pixels or evidence differ from capture")
		}
	}
	for _, r := range pending {
		next, effects, e := Step(r.State, Event{Kind: BundleConfirmed})
		if e != nil {
			return e
		}
		copy := *r
		copy.State = next
		copy.BundlePath = obs.Path
		copy.PendingSubmission = ""
		n := byCapture[r.ID]
		copy.Evidence = Evidence{Crop: n.Crop, BoxText: n.BoxText}
		if n.Page != nil {
			copy.Evidence.Page = *n.Page
		}
		copy.Ephemeral = false
		if e = s.finishTransferLocked(r, &copy, effects); e != nil {
			return e
		}
	}
	// A previous call may have persisted the reference and failed while deleting
	// the source. Repeat only those deletions, preserving the published bundle.
	for _, r := range s.records {
		if r.BundlePath == obs.Path && (r.State.Phase == Bundled || r.State.Phase == Retired) {
			if err = s.disk.remove(r.ID); err != nil {
				return err
			}
			if r.State.Phase == Retired {
				if err = s.disk.removeRef(r.ID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func sameCapturedAsset(a, b Asset) bool {
	if a.Evidence.Crop.Unavailable != "" || b.Evidence.Crop.Unavailable != "" {
		return a.Evidence.Crop.Unavailable != "" && b.Evidence.Crop.Unavailable != "" && len(a.PNG) == 0 && len(b.PNG) == 0
	}
	a.Evidence.Crop.File = b.Evidence.Crop.File
	return bytes.Equal(a.PNG, b.PNG) && reflect.DeepEqual(a.Evidence, b.Evidence)
}
func (s *Service) finishTransferLocked(original, next *captureRecord, effects []Effect) error {
	// Even an abandoned transfer first records its terminal reference. Recovery
	// can finish the ordered deletions if the process stops between these effects.
	if err := s.persistLocked(next); err != nil {
		return err
	}
	*original = *next
	for _, effect := range effects {
		switch effect {
		case DeleteSource:
			if err := s.disk.remove(next.ID); err != nil {
				return err
			}
		case DeleteReference:
			if err := s.disk.removeRef(next.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCaptureEvidence(e Evidence, p board.PreparedNote, png []byte) error {
	if err := core.ValidatePNG(png); err != nil {
		return err
	}
	if err := e.Page.Validate(); err != nil {
		return err
	}
	if err := core.ValidateCrop(e.Crop); err != nil {
		return err
	}
	if err := core.ValidateBoxText(e.BoxText); err != nil {
		return err
	}
	if err := core.ValidateBox(p.Region); err != nil {
		return err
	}
	if e.Crop.Document != p.Document || e.BoxText.Region != p.Region {
		return errors.New("capture document or original region mismatch")
	}
	if !p.Region.Contains(e.Crop.Box) || e.Crop.ClippedToViewport != (e.Crop.Box != p.Region) {
		return errors.New("capture crop does not match original region")
	}
	if p.At.Route != "" && e.Page.URL != p.At.Route {
		return errors.New("capture page differs from prepared route")
	}
	return nil
}
