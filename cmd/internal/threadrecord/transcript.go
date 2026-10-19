package threadrecord

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// Where a stored transcript binding came from (pair#695 P2).
const (
	TranscriptSourceMinted       = "minted"        // the ID Couch chose at launch; no file yet
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
	StorageRoot  string    `json:"storage_root,omitempty"`
	RelativePath string    `json:"relative_path,omitempty"`
	BoundAt      time.Time `json:"bound_at"`
	Source       string    `json:"source"`
}

// Established reports whether the binding names a file.
func (b TranscriptBinding) Established() bool { return b.StorageRoot != "" }

// Validate is the structural acceptance rule, pure: the root must be one of
// the agent's native root names, and the path a root transcript naming
// exactly this native ID, read by the scanners' own parser.
func (b TranscriptBinding) Validate() error {
	if b.Agent == "" {
		return errors.New("transcript binding has no agent")
	}
	if b.NativeID == "" || strings.ContainsAny(b.NativeID, `/\`) || strings.ContainsRune(b.NativeID, 0) {
		return fmt.Errorf("transcript binding has an invalid native id %q", b.NativeID)
	}
	if b.BoundAt.IsZero() {
		return errors.New("transcript binding has no bound time")
	}
	switch b.Source {
	case TranscriptSourceMinted, TranscriptSourceLedger, TranscriptSourceLegacyLedger:
	default:
		return fmt.Errorf("transcript binding has an unknown source %q", b.Source)
	}
	if (b.StorageRoot == "") != (b.RelativePath == "") {
		return errors.New("transcript binding names a root without a path, or a path without a root")
	}
	if !b.Established() {
		return nil
	}
	if b.Source == TranscriptSourceMinted {
		return errors.New("a minted transcript binding cannot name a file")
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
	return &out
}
