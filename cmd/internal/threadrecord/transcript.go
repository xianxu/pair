package threadrecord

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// Where a stored transcript binding came from (pair#695 P2).
const (
	TranscriptSourceLedger       = "ledger"        // copied from an established ledger binding
	TranscriptSourceLegacyLedger = "legacy-ledger" // a record from before the field, bound once
)

// TranscriptBinding is the agent's exact native conversation, stored on the
// thread record so a resume opens it by one stat instead of a search. The
// file is named by the ledger's authorization proof: a native root NAME plus a
// path under it. The root's location comes from the runtime, never the record.
type TranscriptBinding struct {
	Agent        string    `json:"agent"`
	NativeID     string    `json:"native_id"`
	StorageRoot  string    `json:"storage_root,omitempty"` // empty while only the ID is known
	RelativePath string    `json:"relative_path,omitempty"`
	BoundAt      time.Time `json:"bound_at"`
	Source       string    `json:"source"`
	// Unknown carries members a newer build added inside the binding, so they
	// neither make the record unreadable nor vanish on rewrite (pair#695 §2a).
	// Their tokens survive exactly; the record's indenting encoder may change
	// only their whitespace.
	Unknown map[string]json.RawMessage `json:"-"`
}

type transcriptWire TranscriptBinding

var transcriptKnownFields = strictjson.KnownFields(reflect.TypeOf(transcriptWire{}))

// UnmarshalJSON decodes the owned members strictly and keeps the rest.
func (b *TranscriptBinding) UnmarshalJSON(raw []byte) error {
	var wire transcriptWire
	unknown, err := strictjson.DecodeKnown(raw, transcriptKnownFields, &wire)
	if err != nil {
		return err
	}
	*b = TranscriptBinding(wire)
	if len(unknown) > 0 {
		b.Unknown = unknown
	}
	return nil
}

// MarshalJSON writes the owned members, then the preserved ones verbatim.
func (b TranscriptBinding) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(transcriptWire(b))
	if err != nil {
		return nil, err
	}
	return strictjson.AppendMembers(raw, b.Unknown, transcriptKnownFields)
}

// Established reports whether the binding names a file.
func (b TranscriptBinding) Established() bool { return b.StorageRoot != "" }

// Validate is the structural acceptance rule, pure: the root must be one of
// the agent's native root names, and the path a root transcript naming
// exactly this native ID, read by the scanners' own parser.
func (b TranscriptBinding) Validate() error {
	if b.BoundAt.IsZero() {
		return errors.New("transcript binding has no bound time")
	}
	return b.ValidateLocation()
}

// ValidateLocation is Validate without the bound time, for a binding chosen
// but not yet stamped by the write that stores it.
func (b TranscriptBinding) ValidateLocation() error {
	if b.Agent == "" {
		return errors.New("transcript binding has no agent")
	}
	if b.NativeID == "" || strings.ContainsAny(b.NativeID, `/\`) || strings.ContainsRune(b.NativeID, 0) {
		return fmt.Errorf("transcript binding has an invalid native id %q", b.NativeID)
	}
	switch b.Source {
	case TranscriptSourceLedger, TranscriptSourceLegacyLedger:
	default:
		return fmt.Errorf("transcript binding has an unknown source %q", b.Source)
	}
	if (b.StorageRoot == "") != (b.RelativePath == "") {
		return errors.New("transcript binding names a root without a path, or a path without a root")
	}
	if !b.Established() {
		return nil
	}
	agent := sessioninventory.Agent(b.Agent)
	if !slices.Contains(sessioninventory.NativeRootNames(agent), b.StorageRoot) {
		return fmt.Errorf("transcript root %q is not a %s root", b.StorageRoot, b.Agent)
	}
	if !filepath.IsLocal(b.RelativePath) || filepath.Clean(b.RelativePath) != b.RelativePath {
		return fmt.Errorf("transcript path %q is not a clean path under its root", b.RelativePath)
	}
	if id, ok := sessioninventory.RootTranscriptID(agent, b.RelativePath); !ok || id != b.NativeID {
		return fmt.Errorf("transcript path %q is not the root transcript of native id %s", b.RelativePath, b.NativeID)
	}
	return nil
}

// CloneTranscriptBinding copies a binding pointer.
func CloneTranscriptBinding(b *TranscriptBinding) *TranscriptBinding {
	if b == nil {
		return nil
	}
	out := *b
	out.Unknown = strictjson.CloneRaw(b.Unknown)
	return &out
}
