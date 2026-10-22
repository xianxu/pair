package editorendpoint

import (
	"errors"
	"os"
	"syscall"
)

// EndpointProcessSource supplies native process birth evidence without a host
// process-table scan. procutil.Source implements it.
type EndpointProcessSource interface {
	ReadIdentity(int) (string, error)
}

// EndpointProcessState describes evidence about a previously identified owner.
type EndpointProcessState string

const (
	EndpointProcessAlive    EndpointProcessState = "alive"
	EndpointProcessReplaced EndpointProcessState = "replaced"
	EndpointProcessDead     EndpointProcessState = "dead"
	EndpointProcessUnknown  EndpointProcessState = "unknown"
)

// ClassifyEndpointProcess compares a saved birth stamp with one native read.
// Establishing an initial stamp is a separate operation: an empty expected
// stamp cannot establish that an observed process owns an endpoint.
func ClassifyEndpointProcess(expected, observed string, err error) EndpointProcessState {
	if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrNotExist) {
		return EndpointProcessDead
	}
	if err != nil || expected == "" || observed == "" {
		return EndpointProcessUnknown
	}
	if expected == observed {
		return EndpointProcessAlive
	}
	return EndpointProcessReplaced
}
