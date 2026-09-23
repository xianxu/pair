package couchcore

import "context"

// WorkspaceReadiness prepares a host without allocating or launching a thread.
// Callers inject this seam; ordinary startup never implicitly invokes it.
type WorkspaceReadiness interface {
	Ensure(context.Context, ProvisionRequest) (ProvisionResult, error)
}
