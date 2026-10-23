package couchtty

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	core "github.com/xianxu/pair/cmd/internal/submission"
)

type captureSubmissionCast struct {
	*submissionCastFake
	observations   []core.PublicationObservation
	reconcileErr   error
	pendingErr     error
	pendingChecked bool
}

func (f *captureSubmissionCast) ReconcilePendingSubmissions(ctx context.Context, observe func(context.Context, string) (core.Submission, core.PublicationObservation, error)) error {
	f.pendingChecked = true
	_, obs, err := observe(ctx, "old-capture")
	if err != nil {
		return err
	}
	if obs.State != core.PublicationAbsent {
		return errors.New("unexpected old bundle")
	}
	return f.pendingErr
}

func TestSubmissionRecoversClaimsBeforeStartingNewBundle(t *testing.T) {
	m, base, target := newSubmissionManagerTest(t)
	f := &captureSubmissionCast{submissionCastFake: base, pendingErr: errors.New("old publication unknown")}
	m.current = func() (submissionCast, error) { return f, nil }
	acknowledgeSubmissionFeedback(t, m, base)
	run := submissionWorkQueue(t, m)
	if _, err := m.request(context.Background(), "prepare", "new-capture", target); err != nil {
		t.Fatal(err)
	}
	run()
	if !f.pendingChecked || f.prepared != 0 || m.job.status.State != "failed" {
		t.Fatalf("pending=%v prepared=%d status=%+v", f.pendingChecked, f.prepared, m.job.status)
	}
}

func (f *captureSubmissionCast) ReconcilePreparedSubmission(ctx context.Context, sub core.Submission, obs core.PublicationObservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.observations = append(f.observations, obs)
	return f.reconcileErr
}

func TestSubmissionCapturePublicationReconciled(t *testing.T) {
	for _, failure := range []string{"", "guard", "publication", "reconcile"} {
		t.Run(failure, func(t *testing.T) {
			m, base, target := newSubmissionManagerTest(t)
			f := &captureSubmissionCast{submissionCastFake: base}
			m.current = func() (submissionCast, error) { return f, nil }
			acknowledgeSubmissionFeedback(t, m, base)
			run := submissionWorkQueue(t, m)
			if _, err := m.request(context.Background(), "prepare", "capture", target); err != nil {
				t.Fatal(err)
			}
			if failure == "guard" {
				m.guard = func(core.Target) error { return errors.New("target changed") }
			}
			if failure == "publication" {
				// A foreign directory makes publication refuse before creating the bundle.
				if err := os.Mkdir(filepath.Join(filepath.Dir(filepath.Dir(m.store.ManifestPath("capture"))), "alien"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if failure == "reconcile" {
				f.reconcileErr = errors.New("reference persistence failed")
			}
			run()
			if len(f.observations) != 1 {
				t.Fatalf("publication observations=%v", f.observations)
			}
			want := core.PublicationConfirmed
			if failure == "guard" || failure == "publication" {
				want = core.PublicationAbsent
			}
			if f.observations[0].State != want {
				t.Fatalf("observation=%+v want %s", f.observations[0], want)
			}
			if failure != "" && m.job.status.State != "failed" {
				t.Fatalf("failure hidden: %+v", m.job.status)
			}
		})
	}
}
