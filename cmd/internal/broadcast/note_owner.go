package broadcast

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

// NoteOwnerHeader projects ownership for the UI. Its value is public, not a
// credential: handlers accept authority only from the lock gate's context.
const NoteOwnerHeader = "X-Couch-Note-Owner"

// OperatorNoteOwner is reserved for the trusted isolated Chrome context.
const OperatorNoteOwner = "operator"

type noteOwnerKey struct{}

// The lock closes revoked synchronously when its session generation ends.
// Keep that proof with the owner so delayed requests cannot use stale authority
// while the gate's cancellation goroutine is still waiting to run.
type noteOwnerProof struct {
	owner   string
	revoked <-chan struct{}
}

func sessionNoteOwner(session string) string {
	sum := sha256.Sum256([]byte("couch-note-owner\x00" + session))
	return hex.EncodeToString(sum[:])
}

// NoteOwnerFromContext returns the owner supplied after cookie validation.
// Requests outside the gate have no owner; body/header claims cannot set it.
func NoteOwnerFromContext(ctx context.Context) string {
	proof, _ := ctx.Value(noteOwnerKey{}).(noteOwnerProof)
	select {
	case <-proof.revoked:
		return ""
	default:
		return proof.owner
	}
}
