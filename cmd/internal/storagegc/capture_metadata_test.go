package storagegc

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"os"
	"testing"
	"time"
)

func TestCapturePublicationCrashAndExactIdentity(t *testing.T) {
	c, _ := NewCoordinator(t.TempDir())
	o, _ := artifactpath.NewStorageOwner(c.Root, "", "work")
	p, _ := artifactpath.ResolveScoped(c.Root, "work")
	set, _ := p.ParkedScrollbackArtifacts("20260913T120000")
	member := artifactpath.ArtifactMember{Owner: o, Path: set.Raw, Family: "parked-scrollback"}
	capture, _ := artifactpath.ParseParkedCapture(member, time.UTC)
	if err := os.WriteFile(set.Raw, []byte("raw"), 0600); err != nil {
		t.Fatal(err)
	}
	c.BeforePersist = func() error { return errors.New("crash before publication") }
	if err := c.PublishCaptureMetadata(context.Background(), o, capture, time.Now()); err == nil {
		t.Fatal("expected failure")
	}
	if _, err := os.Stat(set.Raw); err != nil {
		t.Fatal("lost capture", err)
	}
	if _, err := os.Stat(set.Metadata); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("published incomplete evidence")
	}
	c.BeforePersist = nil
	if err := c.PublishCaptureMetadata(context.Background(), o, capture, time.Now()); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(set.Metadata)
	meta, err := artifactpath.DecodeCaptureMetadata(data, capture)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := CaptureIdentity(set.Raw)
	if err != nil || actual != meta.RawIdentity || meta.EventsIdentity != nil {
		t.Fatalf("identity mismatch: %+v %+v %v", actual, meta, err)
	}
	if err := c.PublishCaptureMetadata(context.Background(), o, capture, time.Now()); err == nil {
		t.Fatal("overwrote established timestamp")
	}
}
