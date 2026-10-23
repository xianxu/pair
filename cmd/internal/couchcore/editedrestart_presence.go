package couchcore

import "context"

// This owner-local capability completes only ID-only edited restart evidence.
// Ordinary BindTranscript and resume retain their existing policy and cost.
type editedRestartMetadataObserver interface {
	ObserveEditedRestartNativeMetadata(context.Context, ThreadRecord, string) (editedRestartNativeObservation, error)
}

func (c ScopedThreadArtifactCollisionChecker) ObserveEditedRestartNativeMetadata(ctx context.Context, thread ThreadRecord, nativeID string) (editedRestartNativeObservation, error) {
	runtime, err := c.nativeTranscriptRuntime(thread.Address.RepoScope, string(thread.Address.Tag))
	if err != nil {
		return editedRestartNativeObservation{}, err
	}
	return observeEditedRestartNativeMetadata(ctx, runtime, thread, nativeID)
}

var _ editedRestartMetadataObserver = ScopedThreadArtifactCollisionChecker{}
