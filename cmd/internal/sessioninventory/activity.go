package sessioninventory

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// SessionActivity is the inventory-owned projection of an authorized root's
// creation and last-activity timestamps.
// pair:155-concept pure new final
type SessionActivity struct {
	CreatedAt              time.Time  `json:"created_at"`
	CreatedTimeSource      TimeSource `json:"created_time_source"`
	LastActivityAt         time.Time  `json:"last_activity_at"`
	LastActivityTimeSource TimeSource `json:"last_activity_time_source"`
}

// ActivityForSession stats only the established scanner-authorized root
// transcript. Other binding states remain explicit absence.
func ActivityForSession(runtime Runtime, query SessionQuery) (SessionActivity, bool, error) {
	if query.Status != BindingEstablished || query.Root == nil {
		return SessionActivity{}, false, nil
	}
	transcript, err := RootTranscript(*query.Root)
	if err != nil {
		return SessionActivity{}, false, err
	}
	known := false
	for _, candidate := range runtime.NativeRoots(query.Root.Agent) {
		if candidate.Name == transcript.StorageRoot {
			known = true
			break
		}
	}
	if !known {
		return SessionActivity{}, false, fmt.Errorf("%w: %s", ErrRootUnknown, transcript.StorageRoot)
	}
	// One stat of the bound transcript, not a walk of its store (pair#420):
	// title pollers and Couch's idle fading ask this of every thread.
	file, err := runtime.StatFile(Artifact{StorageRoot: transcript.StorageRoot, RelativePath: transcript.RelativePath})
	if errors.Is(err, os.ErrNotExist) {
		return SessionActivity{}, false, nil
	}
	if err != nil {
		return SessionActivity{}, false, err
	}
	if file.ModTime == nil || file.ModTime.IsZero() {
		return SessionActivity{}, false, nil
	}
	activity := SessionActivity{LastActivityAt: file.ModTime.UTC(), LastActivityTimeSource: TimeSourceMTime}
	if query.Root.Time != nil {
		activity.CreatedAt = query.Root.Time.Value.UTC()
		activity.CreatedTimeSource = query.Root.Time.Source
	}
	return activity, true, nil
}
