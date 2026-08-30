package launcher

import "testing"

func TestOSRuntimeImplementsContextualCleanupBoundary(t *testing.T) {
	var _ contextualCleanupRuntime = OSRuntime{}
}
