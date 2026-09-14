package storagegc

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func collectorFixture(t *testing.T) (*Collector, artifactpath.StorageOwner) {
	t.Helper()
	c, _ := coordinatorFixture(t)
	o, _ := artifactpath.NewStorageOwner(c.Root, "", "tag")
	os.WriteFile(filepath.Join(c.Root, "draft-tag.md"), []byte("draft"), 0600)
	ctx := context.Background()
	if err := c.Initialize(ctx, o); err != nil {
		t.Fatal(err)
	}
	if err := c.CompleteMigration(ctx, nil); err != nil {
		t.Fatal(err)
	}
	before := c.Now()
	c.Now = func() time.Time { return before.Add(61 * 24 * time.Hour) }
	return &Collector{Coordinator: c, Agents: []string{"codex"}, Legacy: func(context.Context, artifactpath.StorageOwner) (Liveness, error) { return ProcessDead, nil }}, o
}
func TestCollectorPreviewIsReadOnlyAndSeparatesCaptureAge(t *testing.T) {
	gc, o := collectorFixture(t)
	p, _ := artifactpath.ResolveScoped(o.Directory(), o.Tag)
	cap, _ := p.ParkedScrollbackArtifacts("20260913T000000")
	os.WriteFile(cap.Raw, []byte("capture"), 0600)
	now := gc.Coordinator.Now()
	os.Chtimes(cap.Raw, now.Add(-8*24*time.Hour), now.Add(-8*24*time.Hour))
	before, _ := os.ReadFile(gc.Coordinator.statePath(o))
	report, err := gc.Preview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Items) != 2 {
		t.Fatalf("%+v", report)
	}
	for _, item := range report.Items {
		if item.Decision.State != Eligible {
			t.Fatalf("%+v", item)
		}
	}
	after, _ := os.ReadFile(gc.Coordinator.statePath(o))
	if string(before) != string(after) {
		t.Fatal("preview changed metadata")
	}
}
func TestCollectorUnknownLegacyNeverEligible(t *testing.T) {
	gc, _ := collectorFixture(t)
	gc.Legacy = nil
	report, err := gc.Preview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range report.Items {
		if item.Decision.State == Eligible {
			t.Fatal("missing legacy evidence accepted")
		}
	}
}

func TestApplyInitializesBeforeMigrationAndNeverDeletes(t *testing.T) {
	c, _ := coordinatorFixture(t)
	o, _ := artifactpath.NewStorageOwner(c.Root, "", "tag")
	path := filepath.Join(c.Root, "draft-tag.md")
	os.WriteFile(path, []byte("keep"), 0600)
	gc := &Collector{Coordinator: c, Agents: []string{"codex"}, Legacy: func(context.Context, artifactpath.StorageOwner) (Liveness, error) { return ProcessDead, nil }}
	if _, err := gc.Apply(context.Background(), 100); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReadOwner(o); err != nil {
		t.Fatal("missing initial grace", err)
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != "keep" {
		t.Fatal("deleted before migration")
	}
}
func TestApplyDeletesEligibleSessionAndIsIdempotent(t *testing.T) {
	gc, o := collectorFixture(t)
	report, err := gc.Apply(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if report.Collected != 1 || report.CollectedBytes != 5 {
		t.Fatalf("%+v", report)
	}
	if _, err := os.Stat(filepath.Join(o.Directory(), "draft-tag.md")); !os.IsNotExist(err) {
		t.Fatal("eligible payload remains", err)
	}
	report, err = gc.Apply(context.Background(), 100)
	if err != nil || report.Collected != 0 {
		t.Fatalf("retry %+v %v", report, err)
	}
}

func TestCollectorUsesCapturePublicationClockAndRejectsChangedPayload(t *testing.T) {
	for _, mutation := range []string{"none", "raw", "events", "metadata"} {
		t.Run(mutation, func(t *testing.T) {
			gc, owner := collectorFixture(t)
			at := gc.Coordinator.Now().Add(-7*24*time.Hour - time.Hour)
			paths, _ := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
			capture, _ := paths.ParkedScrollbackArtifacts(at.UTC().Format("20060102T150405"))
			if err := os.WriteFile(capture.Raw, []byte("raw"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(capture.Events, []byte("events"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := gc.Coordinator.PublishCaptureMetadata(context.Background(), owner, artifactpath.ParkedCapture{Raw: capture.Raw, Events: capture.Events, Metadata: capture.Metadata}, at); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "raw":
				os.WriteFile(capture.Raw, []byte("changed raw"), 0600)
			case "events":
				os.Remove(capture.Events)
			case "metadata":
				os.WriteFile(capture.Metadata, []byte("{}"), 0600)
			}
			report, err := gc.Preview(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, item := range report.Items {
				if item.Bucket != artifactpath.CaptureRetention {
					continue
				}
				found = true
				want := Blocked
				if mutation == "none" {
					want = Eligible
				}
				if item.Decision.State != want {
					t.Fatalf("got %+v; want %s", item.Decision, want)
				}
			}
			if !found {
				t.Fatal("capture absent from inventory")
			}
		})
	}
}
