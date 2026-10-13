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
	c, _ := NewCoordinator(t.TempDir())
	o, _ := artifactpath.NewStorageOwner(c.Root, "scope", "work")
	os.MkdirAll(o.Directory(), 0700)
	rawPath := filepath.Join(c.Root, "live.raw")
	eventsPath := filepath.Join(c.Root, "live.events")
	os.WriteFile(rawPath, []byte("terminal"), 0600)
	os.WriteFile(eventsPath, nil, 0600)
	rawID, _ := CaptureIdentity(rawPath)
	eventID, _ := CaptureIdentity(eventsPath)
	w := artifactpath.CaptureWitness{Type: "capture_witness", Version: 1, Generation: "generation", ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 18, Nonce: "nonce", RawFile: artifactpath.CaptureFileID{Device: rawID.Device, Inode: rawID.Inode}, EventsFile: artifactpath.CaptureFileID{Device: eventID.Device, Inode: eventID.Inode}}
	data, _ := json.Marshal(w)
	data = append(data, '\n')
	os.WriteFile(eventsPath, data, 0600)
	got, e := ReadCaptureWitness(rawPath, eventsPath)
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
	if _, e = ReadCaptureWitness(rawPath, eventsPath); e == nil {
		t.Fatal("accepted replacement raw")
	}
	os.WriteFile(eventsPath, []byte("{\"type\":\"resize\"}\n"), 0600)
	if legacy, e := ReadCaptureWitness(rawPath, eventsPath); e != nil || legacy != nil {
		t.Fatal("legacy not explicit", e)
	}
}
