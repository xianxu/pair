package notedraft

import (
	"context"
	"encoding/json"
	"fmt"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"

	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecoveryReopensCorruptPixelsAndPublishedReferences(t *testing.T) {
	s, r, p := transferFixture(t)
	path := t.TempDir()
	r.Owner = Owner{Scope: "owner"}
	r.Request = "prepare"
	r.Created = time.Now()
	r.Expires = r.Created.Add(DraftTTL)
	r.RecordingPath = path
	s.opts.RecordingPath = path
	data, _ := json.Marshal(r)
	if err := s.disk.update(r.ID, data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.disk.root, r.ID, "capture.png"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(Options{Root: s.disk.root, RecordingPath: path, Capture: &heldCapture{}})
	if err != nil {
		t.Fatal("corrupt pixels blocked metadata recovery", err)
	}
	defer reopened.Close()
	assets, err := reopened.Claim(context.Background(), "missing", []string{r.ID})
	if err != nil || assets[r.ID].Evidence.Crop.Unavailable == "" {
		t.Fatal(assets, err)
	}
	if err = reopened.ReconcilePublication(context.Background(), core.Submission{ID: "missing"}, core.PublicationObservation{State: core.PublicationAbsent}); err != nil {
		t.Fatal(err)
	}
	// Repair only the test fixture before publishing its original bytes.
	if err = os.WriteFile(filepath.Join(s.disk.root, r.ID, "capture.png"), p, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.Claim(context.Background(), "submission", []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	sub, obs := publishCapture(t, r, p, "submission")
	if err = reopened.ReconcilePublication(context.Background(), sub, obs); err != nil {
		t.Fatal(err)
	}
	if err = reopened.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := New(Options{Root: s.disk.root, RecordingPath: path, Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if again.records[r.ID] == nil || again.records[r.ID].BundlePath != obs.Path {
		t.Fatal("reference not restored")
	}
	if err = again.Resolve(r.NoteID); err != nil {
		t.Fatal(err)
	}
	if refs, _ := again.disk.listRefs(); len(refs) != 0 {
		t.Fatal("resolved reference retained")
	}
	if _, err = os.Stat(obs.Path); err != nil {
		t.Fatal("resolve deleted published bundle")
	}
}
func TestRecoveryGlobalReservationsAreAtomic(t *testing.T) {
	root := t.TempDir()
	one, err := newDiskStore(root)
	if err != nil {
		t.Fatal(err)
	}
	two, err := newDiskStore(root)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			disk := one
			if i%2 == 1 {
				disk = two
			}
			r := captureRecord{ID: fmt.Sprint(i), State: State{Phase: Preparing}}
			data, _ := json.Marshal(r)
			results <- disk.reserve(r.ID, data)
		}(i)
	}
	wg.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		} else if err.Error() != StoreFull {
			t.Fatal(err)
		}
	}
	if accepted != 31 {
		t.Fatalf("worst-case reservations admitted %d, want 31", accepted)
	}
	n, _, err := one.usage()
	if err != nil || n != 31 {
		t.Fatal(n, err)
	}
}
func TestRecoverySignedStatusUsesConfirmedSave(t *testing.T) {
	s, err := New(Options{Root: t.TempDir(), RecordingPath: t.TempDir(), Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := Owner{Scope: "owner", Generation: 1}
	r := &captureRecord{ID: "unavailable", Owner: owner, RecordingPath: s.opts.RecordingPath, Request: "prepare", State: State{Phase: Ready, Unavailable: StoreFull}, Expires: time.Now().Add(DraftTTL)}
	token, err := s.token(r)
	if err != nil {
		t.Fatal(err)
	}
	r.State = State{Phase: Attached, Unavailable: StoreFull}
	r.NoteID = "saved"
	r.Ephemeral = true
	s.records[r.ID] = r
	got, err := s.Status(owner, token)
	if err != nil || got.State != "saved" || got.NoteID != "saved" {
		t.Fatal(got, err)
	}
}
func TestRecoveryAdmissionBoundReturnsSaveReadyToken(t *testing.T) {
	root := t.TempDir()
	s, err := New(Options{Root: root, RecordingPath: t.TempDir(), Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	foreign := t.TempDir()
	for i := 0; i < 31; i++ {
		r := captureRecord{ID: fmt.Sprint(i), RecordingPath: foreign, State: State{Phase: Preparing}, Expires: time.Now().Add(DraftTTL)}
		data, _ := json.Marshal(r)
		if err = s.disk.reserve(r.ID, data); err != nil {
			t.Fatal(err)
		}
	}
	a, p := preparation(t)
	status, err := s.Prepare(context.Background(), Owner{Scope: "owner", Generation: 1}, "last", a, p, false)
	if err != nil || status.State != "ready" || status.Unavailable != StoreFull || !strings.HasPrefix(status.Draft, "u.") {
		t.Fatal(status, err)
	}
}
func TestRecoverySweepKeepsUncertainIntentAndCleansMissingRecording(t *testing.T) {
	s, err := New(Options{Root: t.TempDir(), RecordingPath: t.TempDir(), Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	missing := filepath.Join(t.TempDir(), "gone")
	r := captureRecord{ID: "orphan", Owner: Owner{Scope: "owner"}, Request: "prepare", Prepared: board.PreparedNote{Document: 1}, Created: time.Now(), Expires: time.Now().Add(DraftTTL), RecordingPath: missing, NoteID: "note", State: State{Phase: Attached}}
	data, _ := json.Marshal(r)
	if err = s.disk.put(r.ID, data, nil); err != nil {
		t.Fatal(err)
	}
	if err = s.Sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := s.disk.usage(); n != 0 {
		t.Fatal("orphan retained")
	}
}
func TestRecoveryPendingPublicationPreservesUnknown(t *testing.T) {
	s, r, p := transferFixture(t)
	if _, err := s.Claim(context.Background(), "submission", []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReconcilePending(context.Background(), func(context.Context, string) (core.Submission, core.PublicationObservation, error) {
		return core.Submission{ID: "submission"}, core.PublicationObservation{State: core.PublicationUnknown}, nil
	}); err == nil {
		t.Fatal("unknown publication reported reconciled")
	}
	if r.State.Phase != Transferring {
		t.Fatal("unknown released ownership")
	}
	sub, obs := publishCapture(t, r, p, "submission")
	if err := s.ReconcilePending(context.Background(), func(context.Context, string) (core.Submission, core.PublicationObservation, error) {
		return sub, obs, nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryUsesCompleteRecordedEvidence(t *testing.T) {
	for _, scenario := range []struct {
		name                    string
		present, resolved, torn bool
		want                    Phase
	}{{"confirmed", true, false, false, Attached}, {"absent", false, false, false, Retired}, {"resolved", true, true, false, Retired}, {"torn", true, false, true, Attaching}} {
		t.Run(scenario.name, func(t *testing.T) {
			s, r, _ := transferFixture(t)
			dir := filepath.Join(t.TempDir(), "recording")
			writer, err := recording.NewDirWriter(dir)
			if err != nil {
				t.Fatal(err)
			}
			b := recording.NewBuilder("recording")
			el := b.El("p", nil, b.Txt("original"))
			b.Page(recording.SegLive, "https://example.test/", el)
			if scenario.present {
				box := recording.Box{0, 0, 2, 2}
				b.At(10).Add(&recording.Note{ID: r.NoteID, Capture: r.ID, Author: recording.Author{ID: "owner"}, Text: "saved", At: recording.NoteAt{Node: el.ID, Box: &box}, Region: &box})
			}
			if scenario.resolved {
				b.At(20).Add(&recording.NoteState{ID: r.NoteID, State: recording.NoteResolved})
			}
			for _, rec := range b.Recs {
				if err = writer.Put(rec); err != nil {
					t.Fatal(err)
				}
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			if scenario.torn {
				file := filepath.Join(dir, recording.FileNotes)
				data, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(file, append(data, []byte(`{"kind":`)...), 0600); err != nil {
					t.Fatal(err)
				}
			}
			r.RecordingPath = dir
			r.State.Phase = Attaching
			data, _ := json.Marshal(r)
			if err = s.disk.update(r.ID, data); err != nil {
				t.Fatal(err)
			}
			reopened, err := New(Options{Root: s.disk.root, RecordingPath: dir, Capture: &heldCapture{}})
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			if got := reopened.records[r.ID].State.Phase; got != scenario.want {
				t.Fatalf("recovered %s, want %s", got, scenario.want)
			}
		})
	}
}
func TestRecoverySkipsInvalidPersistedOwnership(t *testing.T) {
	for name, mutate := range map[string]func(*captureRecord){"phase": func(r *captureRecord) { r.State.Phase = "impossible" }, "owner": func(r *captureRecord) { r.Owner.Scope = "" }, "path": func(r *captureRecord) { r.RecordingPath = "relative" }, "publication": func(r *captureRecord) { r.State.Phase = Transferring }, "bundle": func(r *captureRecord) { r.State.Phase = Bundled }} {
		t.Run(name, func(t *testing.T) {
			s, r, _ := transferFixture(t)
			mutate(r)
			data, _ := json.Marshal(r)
			if err := s.disk.update(r.ID, data); err != nil {
				t.Fatal(err)
			}
			got, err := New(Options{Root: s.disk.root, RecordingPath: t.TempDir(), Capture: &heldCapture{}})
			if err != nil {
				t.Fatal(err)
			}
			defer got.Close()
			if got.records[r.ID] != nil {
				t.Fatal("invalid ownership loaded")
			}
			if _, err = os.Stat(filepath.Join(s.disk.root, r.ID, "metadata.json")); err != nil {
				t.Fatal("invalid ownership evidence deleted", err)
			}
		})
	}
}
func TestRecoveryOrdinaryCaptureFailureOffersRetry(t *testing.T) {
	s := &Service{}
	for _, reason := range []string{CaptureBusy, StoreFull, "crop unavailable: deadline exceeded"} {
		status := s.statusLocked(&captureRecord{State: State{Phase: Ready, Unavailable: reason}})
		want := "ready"
		if reason != CaptureBusy && reason != StoreFull {
			want = "failed"
		}
		if status.State != want {
			t.Fatal(status, want)
		}
	}
}
func TestRecoveryStartupObservesPublicationBeforeDeletingSource(t *testing.T) {
	s, r, p := transferFixture(t)
	r.RecordingPath = t.TempDir()
	data, _ := json.Marshal(r)
	if err := s.disk.update(r.ID, data); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Claim(context.Background(), "submission", []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	sub, obs := publishCapture(t, r, p, "submission")
	called := false
	reopened, err := New(Options{Root: s.disk.root, RecordingPath: r.RecordingPath, Capture: &heldCapture{}, ObservePublication: func(ctx context.Context, id string) (core.Submission, core.PublicationObservation, error) {
		called = true
		if id != "submission" {
			t.Fatal(id)
		}
		return sub, obs, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if !called || reopened.records[r.ID].State.Phase != Bundled {
		t.Fatal("startup did not reconcile publication")
	}
	if n, _, _ := reopened.disk.usage(); n != 0 {
		t.Fatal("startup retained confirmed source")
	}
}

func TestRecoverySweepsForeignOrphanReferenceWithoutDeletingBundle(t *testing.T) {
	s, r, p := transferFixture(t)
	sub, obs := publishCapture(t, r, p, "submission")
	_ = sub
	r.RecordingPath = filepath.Join(t.TempDir(), "removed")
	r.State = State{Phase: Bundled}
	r.BundlePath = obs.Path
	data, _ := json.Marshal(r)
	if err := s.disk.putRef(r.ID, data); err != nil {
		t.Fatal(err)
	}
	if err := s.disk.remove(r.ID); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(Options{Root: s.disk.root, RecordingPath: t.TempDir(), Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	refs, err := reopened.disk.listRefs()
	if err != nil || len(refs) != 0 {
		t.Fatal("foreign orphan reference retained", refs, err)
	}
	if _, err = os.Stat(obs.Path); err != nil {
		t.Fatal("orphan cleanup deleted bundle", err)
	}
}

func TestRecoveryNewRecordingReconcilesInactivePreviousCast(t *testing.T) {
	s, r, p := transferFixture(t)
	oldPath := t.TempDir()
	r.RecordingPath = oldPath
	data, _ := json.Marshal(r)
	if err := s.disk.update(r.ID, data); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Claim(context.Background(), "submission", []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	sub, obs := publishCapture(t, r, p, "submission")
	old, err := New(Options{Root: s.disk.root, RecordingPath: oldPath, Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	if err = old.Close(); err != nil {
		t.Fatal(err)
	}
	fresh, err := New(Options{Root: s.disk.root, RecordingPath: t.TempDir(), Capture: &heldCapture{}, ObservePublication: func(context.Context, string) (core.Submission, core.PublicationObservation, error) {
		return sub, obs, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	if len(fresh.records) != 0 {
		t.Fatal("historical notes exposed to new cast")
	}
	if n, _, _ := fresh.disk.usage(); n != 0 {
		t.Fatal("old transfer was not recovered")
	}
	ref, err := fresh.disk.readRef(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	var got captureRecord
	if err = json.Unmarshal(ref, &got); err != nil || got.BundlePath != obs.Path {
		t.Fatal(got, err)
	}
}
func TestRecoveryDoesNotObserveLivePeerPublication(t *testing.T) {
	s, r, _ := transferFixture(t)
	r.RecordingPath = t.TempDir()
	data, _ := json.Marshal(r)
	if err := s.disk.update(r.ID, data); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Claim(context.Background(), "submission", []string{r.ID}); err != nil {
		t.Fatal(err)
	}
	live, err := New(Options{Root: s.disk.root, RecordingPath: r.RecordingPath, Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	if err = os.Remove(r.RecordingPath); err != nil {
		t.Fatal(err)
	}
	calls := 0
	peer, err := New(Options{Root: s.disk.root, RecordingPath: t.TempDir(), Capture: &heldCapture{}, ObservePublication: func(context.Context, string) (core.Submission, core.PublicationObservation, error) {
		calls++
		return core.Submission{ID: "submission"}, core.PublicationObservation{State: core.PublicationAbsent}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	if calls != 0 {
		t.Fatal("live peer publication observed")
	}
	if live.records[r.ID].State.Phase != Transferring {
		t.Fatal("live claim released")
	}
}

func TestRecoveryMalformedForeignMetadataDoesNotBlockNewNotes(t *testing.T) {
	disk, err := newDiskStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = disk.put("corrupt", []byte(`{}`), nil); err != nil {
		t.Fatal(err)
	}
	meta := filepath.Join(disk.root, "corrupt", "metadata.json")
	if err = os.WriteFile(meta, []byte(`{"broken":`), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := New(Options{Root: disk.root, RecordingPath: t.TempDir(), Capture: &heldCapture{}})
	if err != nil {
		t.Fatal("malformed old capture blocked cast", err)
	}
	defer s.Close()
	a, p := preparation(t)
	got, err := s.Prepare(context.Background(), Owner{Scope: "owner", Generation: 1}, "new", a, p, false)
	if err != nil || got.State != "pending" && !(got.State == "ready" && got.Unavailable == StoreFull) {
		t.Fatal("new note blocked", got, err)
	}
	if data, err := os.ReadFile(meta); err != nil || string(data) != `{"broken":` {
		t.Fatal("corrupt evidence changed", string(data), err)
	}
	assets, err := s.Claim(context.Background(), "submission", []string{"corrupt"})
	if err != nil || assets["corrupt"].Evidence.Crop.Unavailable == "" {
		t.Fatal(assets, err)
	}
}
func TestRecoveryUnmeasurableStoreStillOffersSaveReadyNote(t *testing.T) {
	disk, err := newDiskStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = disk.put("corrupt", []byte(`{}`), nil); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(disk.root, "corrupt", "metadata.json")); err != nil {
		t.Fatal(err)
	}
	s, err := New(Options{Root: disk.root, RecordingPath: t.TempDir(), Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, p := preparation(t)
	got, err := s.Prepare(context.Background(), Owner{Scope: "owner", Generation: 1}, "new", a, p, false)
	if err != nil || got.State != "ready" || got.Unavailable != StoreFull || got.Error != "" || !strings.HasPrefix(got.Draft, "u.") {
		t.Fatal(got, err)
	}
}

func TestRecoveryCorruptNeighborDoesNotBlockExistingReadySave(t *testing.T) {
	s, r, _ := transferFixture(t)
	a, p := preparation(t)
	r.Prepared = p
	r.State = State{Phase: Ready, Unavailable: CaptureBusy}
	r.NoteID = ""
	r.Owner = Owner{Scope: "owner", Generation: 1}
	r.Expires = time.Now().Add(DraftTTL)
	s.ctx = context.Background()
	s.opts.Now = time.Now
	s.opts.Write = func(_ context.Context, _ board.AnnotationAdmission, _ board.NoteReservation, _ func() error, settle func(board.OperatorOutcome, error)) error {
		settle(board.OperatorAccepted, nil)
		return nil
	}
	data, _ := json.Marshal(r)
	if err := s.disk.update(r.ID, data); err != nil {
		t.Fatal(err)
	}
	if err := s.disk.put("corrupt", []byte(`{}`), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.disk.root, "corrupt", "metadata.json")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Save(context.Background(), r.Owner, "save", r.ID, a, board.Viewer{ID: "owner"}, 0, "keep authored note", nil)
	if err != nil || got.State != "saved" {
		t.Fatal("corrupt neighbor blocked save", got, err)
	}
	persisted, err := s.disk.metadata(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	var saved captureRecord
	if err = json.Unmarshal(persisted, &saved); err != nil || saved.State.Phase != Attached || saved.NoteID != got.NoteID || saved.SaveRequest != "save" {
		t.Fatal("save ownership was not persisted", saved, err)
	}
}

func TestRevokeScopeRetiresOnlyUnsavedOwnedPreparations(t *testing.T) {
	s, err := New(Options{Root: t.TempDir(), RecordingPath: t.TempDir(), Capture: &heldCapture{}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, p := preparation(t)
	_ = a
	now := time.Now()
	for _, item := range []struct {
		id, scope  string
		phase      Phase
		generation uint64
	}{{"ready", "owner", Ready, 1}, {"pending", "owner", Preparing, 2}, {"uncertain", "owner", Attaching, 1}, {"saved", "owner", Attached, 1}, {"foreign", "other", Ready, 1}} {
		r := &captureRecord{ID: item.id, Owner: Owner{Scope: item.scope, Generation: item.generation}, Request: item.id, RecordingPath: s.opts.RecordingPath, Prepared: p, Created: now, Expires: now.Add(DraftTTL), State: State{Phase: item.phase}}
		if item.phase == Attaching || item.phase == Attached {
			r.NoteID = item.id + "-note"
		}
		data, _ := json.Marshal(r)
		if err = s.disk.put(r.ID, data, nil); err != nil {
			t.Fatal(err)
		}
		s.records[r.ID] = r
	}
	s.RevokeScope("owner")
	for id, want := range map[string]Phase{"ready": Retired, "pending": Retired, "uncertain": Attaching, "saved": Attached, "foreign": Ready} {
		if r := s.records[id]; r.State.Phase != want {
			t.Fatal(id, r.State)
		}
	}
	if !s.records["uncertain"].State.Abandoned {
		t.Fatal("unknown save lost abandonment")
	}
	for _, id := range []string{"uncertain", "saved", "foreign"} {
		if _, err = s.disk.metadata(id); err != nil {
			t.Fatal("retained owner lost evidence", id, err)
		}
	}
}

func TestRevokeScopeCancelsOldCapacityTokenButAllowsReconnect(t *testing.T) {
	now := time.Now()
	s, err := New(Options{Root: t.TempDir(), RecordingPath: t.TempDir(), Capture: &heldCapture{}, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := Owner{Scope: "owner", Generation: 1}
	r := &captureRecord{ID: "capacity", Owner: owner, RecordingPath: s.opts.RecordingPath, Request: "prepare", Created: now, Expires: now.Add(DraftTTL), State: State{Phase: Ready, Unavailable: StoreFull}}
	old, err := s.token(r)
	if err != nil {
		t.Fatal(err)
	}
	s.RevokeScope(owner.Scope)
	if _, err := s.Status(owner, old); err == nil {
		t.Fatal("disconnected capacity token survived")
	}
	now = now.Add(time.Second)
	r.Created = now
	r.Expires = now.Add(DraftTTL)
	fresh, err := s.token(r)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.Status(owner, fresh); err != nil || got.State != "ready" {
		t.Fatal(got, err)
	}
}
