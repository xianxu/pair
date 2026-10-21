package couchschedule

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	MaxManualArgs       = 128
	MaxManualArgsBytes  = 16 << 10
	MaxManualOutputPage = 16 << 10
	MaxManualOutput     = 256 << 10
)

// ManualIntent is allocated once by the client, before the sole submit attempt.
// Its ID is occurrence identity; Digest identifies the immutable requested work.
type ManualIntent struct {
	ID   string   `json:"id"`
	Hub  string   `json:"hub"`
	Task string   `json:"task"`
	Args []string `json:"args,omitempty"`
}

func (m ManualIntent) Validate() error {
	if !ValidTaskName(m.ID) || len(m.ID) > 128 || !ValidTaskName(m.Task) || !filepath.IsAbs(m.Hub) || filepath.Clean(m.Hub) != m.Hub || len(m.Hub) > 4096 || !utf8.ValidString(m.Hub) || strings.ContainsRune(m.Hub, 0) {
		return errors.New("manual run requires a bounded ID, canonical absolute hub and task")
	}
	if len(m.Args) > MaxManualArgs {
		return errors.New("too many manual arguments")
	}
	n := 0
	for _, a := range m.Args {
		n += len(a) + 1
		if !utf8.ValidString(a) || strings.ContainsRune(a, 0) || n > MaxManualArgsBytes {
			return errors.New("manual arguments exceed text or byte limits")
		}
	}
	return nil
}
func (m ManualIntent) Digest() string {
	// Normalize nil/empty argv, preserving argument boundaries and order.
	args := append([]string{}, m.Args...)
	raw, _ := json.Marshal(struct {
		Hub, Task string
		Args      []string
	}{m.Hub, m.Task, args})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

type ManualRequest struct {
	Intent ManualIntent `json:"intent"`
	Offset int          `json:"offset,omitempty"`
}

// ManualStatus is an inspection-only page of one exact occurrence. Output uses
// JSON's base64 byte representation so arbitrary task bytes stay lossless.
type ManualStatus struct {
	ID      string `json:"id"`
	Hub     string `json:"hub"`
	Task    string `json:"task"`
	Stage   string `json:"stage"`
	Outcome string `json:"outcome,omitempty"`
	Exit    *int   `json:"exit,omitempty"`
	Output  []byte `json:"output,omitempty"`
	Offset  int    `json:"offset,omitempty"`
	Total   int    `json:"total,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s ManualStatus) Terminal() bool {
	return s.Stage == "done" || s.Stage == "unknown" || s.Stage == "cancelled" || s.Stage == "failed"
}
func (s ManualStatus) Validate() error {
	if err := (ManualIntent{ID: s.ID, Hub: s.Hub, Task: s.Task}).Validate(); err != nil {
		return err
	}
	if s.Stage == "" || len(s.Stage) > 128 || len(s.Outcome) > 128 || len(s.Error) > 4096 || s.Offset < 0 || s.Total < 0 || s.Total > MaxManualOutput || s.Offset > s.Total || len(s.Output) > MaxManualOutputPage || len(s.Output) > s.Total-s.Offset {
		return errors.New("invalid manual status or output page")
	}
	if s.Exit != nil && (*s.Exit < -1 || *s.Exit > 255) {
		return errors.New("invalid manual exit")
	}
	return nil
}
