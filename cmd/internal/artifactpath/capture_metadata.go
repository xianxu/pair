package artifactpath

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
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
	Witness        *CaptureWitness      `json:"witness,omitempty"`
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
	if err := strictjson.Decode(data, &m); err != nil {
		return m, err
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	if _, present := fields["witness"]; m.Version == 1 && present {
		return m, errors.New("legacy capture metadata cannot contain witness")
	}
	validIdentity := func(i CaptureFileIdentity) bool { return i.Size >= 0 && !i.ModifiedAt.IsZero() && i.Inode != 0 }
	_, offset := m.CapturedAt.Zone()
	if (m.Version != 1 && m.Version != 2) || m.Raw != filepath.Base(capture.Raw) || m.Events != filepath.Base(capture.Events) || m.CapturedAt.IsZero() || offset != 0 || !validIdentity(m.RawIdentity) || (m.EventsIdentity != nil && !validIdentity(*m.EventsIdentity)) {
		return m, errors.New("invalid capture metadata identity or timestamp")
	}
	if (m.Version == 1 && m.Witness != nil) || (m.Version == 2 && (m.Witness == nil || m.EventsIdentity == nil)) {
		return m, errors.New("invalid capture witness version")
	}
	if m.Witness != nil {
		if err := ValidateCaptureWitness(*m.Witness); err != nil {
			return m, err
		}
	}
	return m, nil
}
