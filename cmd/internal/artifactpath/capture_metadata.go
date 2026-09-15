package artifactpath

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"time"
)

// CaptureFileIdentity detects replacement or later modification without reading payloads.
// pair:m5-concept pure
type CaptureFileIdentity struct {
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Device     uint64    `json:"device"`
	Inode      uint64    `json:"inode"`
}

// CaptureMetadata records producer time and the exact completed raw/events pair.
// A nil EventsIdentity records the absence of an optional events file.
// pair:m5-concept pure
type CaptureMetadata struct {
	Version        int                  `json:"version"`
	Raw            string               `json:"raw"`
	Events         string               `json:"events"`
	CapturedAt     time.Time            `json:"captured_at"`
	RawIdentity    CaptureFileIdentity  `json:"raw_identity"`
	EventsIdentity *CaptureFileIdentity `json:"events_identity"`
}

func DecodeCaptureMetadata(data []byte, capture ParkedCapture) (CaptureMetadata, error) {
	var m CaptureMetadata
	if len(data) > 8192 {
		return m, errors.New("capture metadata too large")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&m); err != nil {
		return m, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return m, errors.New("trailing capture metadata")
	}
	validIdentity := func(i CaptureFileIdentity) bool { return i.Size >= 0 && !i.ModifiedAt.IsZero() && i.Inode != 0 }
	_, offset := m.CapturedAt.Zone()
	if m.Version != 1 || m.Raw != filepath.Base(capture.Raw) || m.Events != filepath.Base(capture.Events) || m.CapturedAt.IsZero() || offset != 0 || !validIdentity(m.RawIdentity) || (m.EventsIdentity != nil && !validIdentity(*m.EventsIdentity)) {
		return m, errors.New("invalid capture metadata identity or timestamp")
	}
	return m, nil
}
