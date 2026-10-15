package storagegc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"golang.org/x/sys/unix"
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

func TestCaptureWitnessArchiveAndReplacement(t *testing.T) {
	t.Run("legacy-producer-v1", func(t *testing.T) { testCaptureWitnessArchiveAndReplacement(t, 1) })
	t.Run("producer-epoch-v2", func(t *testing.T) { testCaptureWitnessArchiveAndReplacement(t, 2) })
}
func testCaptureWitnessArchiveAndReplacement(t *testing.T, version int) {
	c, _ := NewCoordinator(t.TempDir())
	o, _ := artifactpath.NewStorageOwner(c.Root, "scope", "work")
	os.MkdirAll(o.Directory(), 0700)
	rawPath := filepath.Join(c.Root, "live.raw")
	eventsPath := filepath.Join(c.Root, "live.events")
	os.WriteFile(rawPath, []byte("terminal"), 0600)
	os.WriteFile(eventsPath, nil, 0600)
	rawID, _ := CaptureIdentity(rawPath)
	eventID, _ := CaptureIdentity(eventsPath)
	w := artifactpath.CaptureWitness{Type: "capture_witness", Version: version, Generation: "generation", ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 18, Nonce: "nonce", RawFile: artifactpath.CaptureFileID{Device: rawID.Device, Inode: rawID.Inode}, EventsFile: artifactpath.CaptureFileID{Device: eventID.Device, Inode: eventID.Inode}}
	if version == 2 {
		w.OpenedAt = time.Date(2026, 9, 13, 11, 0, 0, 0, time.UTC)
	}
	data, _ := json.Marshal(w)
	data = append(data, '\n')
	os.WriteFile(eventsPath, data, 0600)
	got, e := ReadCaptureWitness(context.Background(), rawPath, eventsPath)
	if e != nil || got == nil || *got != w {
		t.Fatalf("%+v %v", got, e)
	}
	p, _ := artifactpath.ResolveScoped(o.Directory(), "work")
	set, _ := p.ParkedScrollbackArtifacts("20260913T120000")
	cap, _ := artifactpath.ParseParkedCapture(artifactpath.ArtifactMember{Owner: o, Path: set.Raw, Family: "parked-scrollback"}, time.UTC)
	os.WriteFile(cap.Raw, []byte("terminal"), 0600)
	os.WriteFile(cap.Events, data, 0600)
	conflict := w
	conflict.Generation = "other"
	bad, _ := json.Marshal(conflict)
	os.WriteFile(cap.Events, append(bad, '\n'), 0600)
	if e = c.PublishCaptureMetadata(context.Background(), o, cap, time.Now(), got); e == nil {
		t.Fatal("certified changed events generation")
	}
	os.WriteFile(cap.Events, data, 0600)
	c.BeforePersist = func() error { return errors.New("interrupted") }
	if e = c.PublishCaptureMetadata(context.Background(), o, cap, time.Now(), got); e == nil {
		t.Fatal("expected interrupted publication")
	}
	if _, e = os.Stat(cap.Metadata); !os.IsNotExist(e) {
		t.Fatal("published interrupted witness")
	}
	c.BeforePersist = nil
	if e = c.PublishCaptureMetadata(context.Background(), o, cap, time.Now(), got); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(cap.Metadata)
	meta, e := artifactpath.DecodeCaptureMetadata(b, cap)
	if e != nil || meta.Version != 2 || meta.Witness == nil || *meta.Witness != w {
		t.Fatalf("%+v %v", meta, e)
	}
	os.Rename(rawPath, rawPath+".old")
	os.WriteFile(rawPath, []byte("replacement"), 0600)
	if _, e = ReadCaptureWitness(context.Background(), rawPath, eventsPath); e == nil {
		t.Fatal("accepted replacement raw")
	}
	os.WriteFile(eventsPath, []byte("{\"type\":\"resize\"}\n"), 0600)
	if legacy, e := ReadCaptureWitness(context.Background(), rawPath, eventsPath); e != nil || legacy != nil {
		t.Fatal("legacy not explicit", e)
	}
}

func TestReadCaptureWitnessRejectsFIFOWithoutWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := ReadCaptureWitness(context.Background(), "unused.raw", path); done <- err }()
	select {
	case err := <-done:
		var invalid *osfs.InvalidMetadataError
		if !errors.As(err, &invalid) {
			t.Fatalf("expected unsafe file error, got %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		// Release the old blocking reader and join both goroutines before failing.
		released := make(chan error, 1)
		go func() {
			writer, err := os.OpenFile(path, os.O_WRONLY, 0)
			if err == nil {
				err = writer.Close()
			}
			released <- err
		}()
		if err := <-released; err != nil {
			t.Fatal(err)
		}
		<-done
		t.Fatal("capture witness read blocked waiting for FIFO writer")
	}
}

func TestReadCaptureWitnessRejectsOversizeBeforeDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(artifactpath.MaxCaptureWitnessEventsBytes + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = ReadCaptureWitness(context.Background(), "unused.raw", path)
	var invalid *osfs.InvalidMetadataError
	if !errors.As(err, &invalid) {
		t.Fatalf("expected size-bound rejection before decoding, got %v", err)
	}
}

func TestReadCaptureWitnessHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReadCaptureWitness(ctx, "unused.raw", filepath.Join(t.TempDir(), "missing.events")); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
