package sessioninventory

import (
	"fmt"
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
	var storage StorageRoot
	for _, candidate := range runtime.NativeRoots(query.Root.Agent) {
		if candidate.Name == transcript.StorageRoot {
			storage = candidate
			break
		}
	}
	if storage.Name == "" {
		return SessionActivity{}, false, fmt.Errorf("%w: %s", ErrRootUnknown, transcript.StorageRoot)
	}
	files, listErr := runtime.ListFiles(storage)
	for _, file := range files {
		if file.Artifact.StorageRoot != transcript.StorageRoot || file.Artifact.RelativePath != transcript.RelativePath {
			continue
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
	if listErr != nil {
		return SessionActivity{}, false, listErr
	}
	return SessionActivity{}, false, nil
}
