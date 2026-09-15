package storagegc

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// CollectionTransaction freezes deletion authority before the first rename.
// Source paths are root-relative; destination paths are relative to its private
// quarantine directory. Only Phase changes after publication.
type CollectionTransaction struct {
	Version     int                         `json:"version"`
	ID          string                      `json:"id"`
	Owner       artifactpath.StorageOwner   `json:"owner"`
	Incarnation string                      `json:"incarnation"`
	Bucket      artifactpath.RetentionClass `json:"bucket"`
	Entries     []CollectionEntry           `json:"entries"`
	Archives    []ArchiveReference          `json:"archives,omitempty"`
	Phase       string                      `json:"phase"`
}

// CollectionEvent records completion of one externally proved effect group.
// Reducer input is a value; filesystem adapters own obtaining that evidence.
type CollectionEvent string

const (
	CollectionDetachmentProved CollectionEvent = "detachment-proved"
	CollectionOwnerRetired     CollectionEvent = "owner-retired"
)

func validCollectionPhase(phase string) bool {
	return phase == "prepared" || phase == "detached" || phase == "finalized"
}

// ReduceTransaction enforces the persisted lifecycle. Frozen authority never
// changes. Prepared may still inspect/move exact sources; detached must never
// revisit source names; finalized may only dispose of receipts/quarantine.
// Successful-step retries are idempotent. A detachment event after finalization
// is refused, so receipt cleanup cannot accidentally authorize a new detach.
func ReduceTransaction(current CollectionTransaction, event CollectionEvent) (CollectionTransaction, error) {
	next := current
	switch {
	case event == CollectionDetachmentProved && (current.Phase == "prepared" || current.Phase == "detached"):
		next.Phase = "detached"
	case event == CollectionOwnerRetired && (current.Phase == "detached" || current.Phase == "finalized"):
		next.Phase = "finalized"
	default:
		return current, fmt.Errorf("collection phase %q rejects event %q", current.Phase, event)
	}
	return next, nil
}
