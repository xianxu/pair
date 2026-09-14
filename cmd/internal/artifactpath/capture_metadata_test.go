package artifactpath

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestCaptureMetadataExactFamilyAndSchema(t *testing.T) {
	o, _ := NewStorageOwner("/pair", "repo", "work")
	p, _ := ResolveScoped(o.Directory(), o.Tag)
	set, _ := p.ParkedScrollbackArtifacts("20260913T120000")
	m, err := MatchArtifact(set.Metadata, []StorageOwner{o}, nil)
	if err != nil {
		t.Fatal(err)
	}
	cap, err := ParseParkedCapture(m, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	identity := CaptureFileIdentity{Size: 4, ModifiedAt: time.Now().UTC(), Device: 1, Inode: 2}
	meta := CaptureMetadata{Version: 1, Raw: filepath.Base(cap.Raw), Events: filepath.Base(cap.Events), CapturedAt: time.Now().UTC(), RawIdentity: identity}
	data, _ := json.Marshal(meta)
	if _, err := DecodeCaptureMetadata(data, cap); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*CaptureMetadata){func(m *CaptureMetadata) { m.Version = 2 }, func(m *CaptureMetadata) { m.Raw = "../" + m.Raw }, func(m *CaptureMetadata) { m.Events = "other.events.jsonl" }, func(m *CaptureMetadata) { m.CapturedAt = time.Time{} }, func(m *CaptureMetadata) { m.RawIdentity.Size = -1 }} {
		bad := meta
		mutate(&bad)
		data, _ = json.Marshal(bad)
		if _, err := DecodeCaptureMetadata(data, cap); err == nil {
			t.Fatal("accepted invalid metadata")
		}
	}
	data, _ = json.Marshal(meta)
	for _, bad := range [][]byte{append(append([]byte{}, data...), []byte(" {}")...), []byte(`{"version":1,"unknown":true}`)} {
		if _, err := DecodeCaptureMetadata(bad, cap); err == nil {
			t.Fatal("accepted schema violation")
		}
	}
}
