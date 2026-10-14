package notedraft

import (
	"bytes"
	"context"
	"encoding/json"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func transferFixture(t *testing.T) (*Service, *captureRecord, []byte) {
	t.Helper()
	disk, err := newDiskStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := &captureRecord{ID: "capture", Owner: Owner{Scope: "owner"}, Request: "prepare", Created: time.Now(), Expires: time.Now().Add(DraftTTL), RecordingPath: "/recording", NoteID: "note", Prepared: board.PreparedNote{Document: 1, Region: recording.Box{0, 0, 2, 2}}, State: State{Phase: Attached}, Evidence: Evidence{BoxText: core.BoxText{Region: recording.Box{0, 0, 2, 2}}, Page: core.PageIdentity{URL: "https://example.test/", Provenance: "cdp"}, Crop: core.Crop{File: "crop-0.png", Document: 1, Box: recording.Box{0, 0, 2, 2}, CapturedAt: time.Now().UTC()}}}
	data, _ := json.Marshal(r)
	p := storePNG(t)
	if err = disk.put(r.ID, data, p); err != nil {
		t.Fatal(err)
	}
	return &Service{disk: disk, opts: Options{RecordingPath: "/recording"}, records: map[string]*captureRecord{r.ID: r}}, r, p
}
func publishCapture(t *testing.T, r *captureRecord, p []byte, id string) (core.Submission, core.PublicationObservation) {
	t.Helper()
	sub := core.Submission{Version: 3, ID: id, PrivacyDecision: core.PrivacyDecision{Privacy: core.Private}, Target: core.Target{Scope: "s", Tag: "t", Session: "session", Nonce: "n"}, Delivery: core.Prepared, Source: core.Source{Kind: "dom", RecordingID: "rec", RecordingPath: "/recording", TabID: "tab", Document: 1}, Notes: []core.Note{{ID: r.NoteID, CaptureID: r.ID, SourceDocument: 1, Page: &r.Evidence.Page, Crop: r.Evidence.Crop, BoxText: r.Evidence.BoxText, PrivacyDecision: core.PrivacyDecision{Privacy: core.Private, PrivacyOrigin: "https://example.test"}}}}
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	crops := map[string][]byte{sub.Notes[0].Crop.File: p}
	if _, err = store.Prepare(sub, crops); err != nil {
		t.Fatal(err)
	}
	obs, err := store.ConfirmPrepared(sub, crops)
	if err != nil {
		t.Fatal(err)
	}
	return sub, obs
}
func TestCaptureTransferRetainsUnknownAndReclaimsConfirmedSource(t *testing.T) {
	s, r, p := transferFixture(t)
	ctx := context.Background()
	assets, err := s.Claim(ctx, "submission", []string{r.ID})
	if err != nil || !bytes.Equal(assets[r.ID].PNG, p) || r.State.Phase != Transferring {
		t.Fatal(assets, err, r.State)
	}
	sub, obs := publishCapture(t, r, p, "submission")
	if err = s.ReconcilePublication(ctx, sub, core.PublicationObservation{State: core.PublicationUnknown}); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := s.disk.usage(); n != 1 {
		t.Fatal("unknown released source")
	}
	if err = s.ReconcilePublication(ctx, sub, obs); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := s.disk.usage(); n != 0 {
		t.Fatal("confirmed source retained")
	}
	if r.State.Phase != Bundled || r.BundlePath != obs.Path {
		t.Fatal(r)
	}
	if err = s.ReconcilePublication(ctx, sub, obs); err != nil {
		t.Fatal("confirmation retry", err)
	}
	assets, err = s.Claim(ctx, "second", []string{r.ID})
	if err != nil || !bytes.Equal(assets[r.ID].PNG, p) {
		t.Fatal("bundled retry lost exact bytes", err)
	}
	if err = s.ReconcilePublication(ctx, core.Submission{ID: "second"}, core.PublicationObservation{State: core.PublicationAbsent}); err != nil {
		t.Fatal(err)
	}
	if r.State.Phase != Bundled || r.BundlePath != obs.Path {
		t.Fatal("absence lost old bundle")
	}
}
func TestCaptureTransferRejectsMismatchedPublication(t *testing.T) {
	s, r, p := transferFixture(t)
	ctx := context.Background()
	if _, err := s.Claim(ctx, "submission", []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	sub, obs := publishCapture(t, r, p, "submission")
	sub.Notes[0].CaptureID = "other"
	if err := s.ReconcilePublication(ctx, sub, obs); err == nil {
		t.Fatal("mismatched publication accepted")
	}
	if n, _, _ := s.disk.usage(); n != 1 {
		t.Fatal("mismatch removed source")
	}
}
func TestCaptureTransferMissingEvidenceIsExplicitUnavailable(t *testing.T) {
	s, r, _ := transferFixture(t)
	if err := os.WriteFile(filepath.Join(s.disk.root, r.ID, "capture.png"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	assets, err := s.Claim(context.Background(), "submission", []string{r.ID, "missing"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{r.ID, "missing"} {
		if len(assets[id].PNG) != 0 || assets[id].Evidence.Crop.Unavailable == "" {
			t.Fatal("missing source fabricated", id, assets[id])
		}
	}
}
func TestCaptureTransferRecoveryFinishesReferenceBeforeSourceDeletion(t *testing.T) {
	s, r, p := transferFixture(t)
	sub, obs := publishCapture(t, r, p, "submission")
	_ = sub
	ref := *r
	ref.State = State{Phase: Bundled}
	ref.BundlePath = obs.Path
	data, _ := json.Marshal(ref)
	if err := s.disk.putRef(r.ID, data); err != nil {
		t.Fatal(err)
	}
	if err := s.recoverReferencesLocked(); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := s.disk.usage(); n != 0 {
		t.Fatal("recovery retained duplicate source")
	}
	assets, err := s.Claim(context.Background(), "second", []string{r.ID})
	if err != nil || !bytes.Equal(assets[r.ID].PNG, p) {
		t.Fatal(err)
	}
}
func TestCaptureTransferResolveWaitsForPublication(t *testing.T) {
	s, r, p := transferFixture(t)
	ctx := context.Background()
	if _, err := s.Claim(ctx, "submission", []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.Resolve(r.NoteID); err != nil {
		t.Fatal(err)
	}
	if !r.State.Abandoned {
		t.Fatal("resolve lost uncertain publication")
	}
	sub, obs := publishCapture(t, r, p, "submission")
	if err := s.ReconcilePublication(ctx, sub, obs); err != nil {
		t.Fatal(err)
	}
	if r.State.Phase != Retired {
		t.Fatal(r.State)
	}
	if refs, _ := s.disk.listRefs(); len(refs) != 0 {
		t.Fatal("resolved reference retained")
	}
	if _, err := os.Stat(obs.Path); err != nil {
		t.Fatal("published bundle deleted", err)
	}
}

func TestCaptureTransferRetriesInterruptedSourceRetirement(t *testing.T) {
	s, r, p := transferFixture(t)
	ctx := context.Background()
	if _, err := s.Claim(ctx, "submission", []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	sub, obs := publishCapture(t, r, p, "submission")
	foreign := filepath.Join(s.disk.root, r.ID, "foreign")
	if err := os.WriteFile(foreign, []byte("stop deletion"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.ReconcilePublication(ctx, sub, obs); err == nil {
		t.Fatal("expected interrupted source deletion")
	}
	if _, err := s.disk.readRef(r.ID); err != nil {
		t.Fatal("source deletion preceded durable reference", err)
	}
	if r.State.Phase != Bundled {
		t.Fatal("durable reference did not become owner", r.State)
	}
	if err := os.Remove(foreign); err != nil {
		t.Fatal(err)
	}
	if err := s.ReconcilePublication(ctx, sub, obs); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := s.disk.usage(); n != 0 {
		t.Fatal("retry did not reclaim source")
	}
}

func TestCaptureTransferRejectsChangedProvenanceAsUnavailable(t *testing.T) {
	for name, mutate := range map[string]func(*captureRecord){
		"route":    func(r *captureRecord) { r.Prepared.At.Route = "https://other.test/" },
		"region":   func(r *captureRecord) { r.Evidence.BoxText.Region = recording.Box{1, 1, 2, 2} },
		"outside":  func(r *captureRecord) { r.Evidence.Crop.Box = recording.Box{1, 1, 2, 2} },
		"clipping": func(r *captureRecord) { r.Evidence.Crop.ClippedToViewport = true },
		"box text": func(r *captureRecord) { r.Evidence.BoxText.Unavailable = "missing" },
	} {
		t.Run(name, func(t *testing.T) {
			s, r, _ := transferFixture(t)
			mutate(r)
			data, _ := json.Marshal(r)
			if err := s.disk.update(r.ID, data); err != nil {
				t.Fatal(err)
			}
			assets, err := s.Claim(context.Background(), "submission", []string{r.ID})
			if err != nil || assets[r.ID].Evidence.Crop.Unavailable == "" || len(assets[r.ID].PNG) != 0 {
				t.Fatal("invalid historical source accepted", assets, err)
			}
		})
	}
}
