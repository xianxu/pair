package artifactpath

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

var ErrCaptureWitnessAbsent = errors.New("capture has no producer witness")

const MaxCaptureWitnessEventsBytes = 8 << 20

// CaptureFileID binds a producer's open file, independently of its growing size.
type CaptureFileID struct {
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
}

// CaptureWitness is capture provenance, never native-conversation authorization.
type CaptureWitness struct {
	Type          string        `json:"type"`
	Version       int           `json:"version"`
	Offset        int64         `json:"offset"`
	Generation    string        `json:"generation"`
	ScopeKey      string        `json:"scope_key"`
	Tag           string        `json:"tag"`
	Agent         string        `json:"agent"`
	LaunchOrdinal uint64        `json:"launch_ordinal"`
	Nonce         string        `json:"nonce"`
	RawFile       CaptureFileID `json:"raw_file"`
	EventsFile    CaptureFileID `json:"events_file"`
}

func ValidateCaptureWitness(w CaptureWitness) error {
	if w.Type != "capture_witness" || w.Version != 1 || w.Offset != 0 || w.LaunchOrdinal == 0 || w.RawFile.Inode == 0 || w.EventsFile.Inode == 0 || w.RawFile == w.EventsFile {
		return errors.New("invalid capture witness generation")
	}
	for _, s := range []string{w.Generation, w.ScopeKey, w.Tag, w.Agent, w.Nonce} {
		if len(s) > 256 || !componentPattern.MatchString(s) {
			return errors.New("invalid capture witness identity")
		}
	}
	return nil
}

// DecodeCaptureWitness checks the entire bounded sidecar for mixed generations.
// Legacy sidecars return explicit absence. A witness must be the first record.
func DecodeCaptureWitness(data []byte) (CaptureWitness, error) {
	var w CaptureWitness
	if len(data) > MaxCaptureWitnessEventsBytes {
		return w, errors.New("capture events exceed witness bound")
	}
	found := false
	first := true
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if err := strictjson.RejectDuplicateKeys(line); err != nil {
			return CaptureWitness{}, err
		}
		var kind struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &kind); err != nil {
			return CaptureWitness{}, fmt.Errorf("invalid capture event: %w", err)
		}
		if kind.Type == "capture_witness" {
			if !first || found {
				return CaptureWitness{}, errors.New("mixed capture witness generations")
			}
			if err := strictjson.Decode(line, &w); err != nil {
				return CaptureWitness{}, err
			}
			if err := ValidateCaptureWitness(w); err != nil {
				return CaptureWitness{}, err
			}
			found = true
		}
		first = false
	}
	if !found {
		return CaptureWitness{}, ErrCaptureWitnessAbsent
	}
	return w, nil
}
