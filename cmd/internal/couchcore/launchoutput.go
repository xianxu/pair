package couchcore

import (
	"errors"
	"io"
	"sync"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

const launchOutputBytes = 16 * 1024

// LaunchOutputHandle is an optional runner capability. Output is an independent
// byte tail, not necessarily UTF-8. An empty stream means output is unavailable.
type LaunchOutputHandle interface {
	LaunchOutput() (stream string, output []byte)
}

// launchOutputTee captures before forwarding, so a failed terminal write does
// not destroy the evidence. The lock protects snapshots against os/exec's pump.
type launchOutputTee struct {
	mu      sync.Mutex
	ring    *ptychild.Ring
	forward io.Writer
}

func newLaunchOutputTee(forward io.Writer) *launchOutputTee {
	return &launchOutputTee{ring: ptychild.NewRing(launchOutputBytes), forward: forward}
}
func (w *launchOutputTee) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.ring.Append(p)
	w.mu.Unlock()
	return w.forward.Write(p)
}
func (w *launchOutputTee) snapshot() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ring.Snapshot()
}
func terminalLaunchOutput(child *ptychild.Child) (string, []byte) {
	output := child.Snapshot()
	if len(output) > launchOutputBytes {
		output = output[len(output)-launchOutputBytes:]
	}
	return "pty stdout/stderr", output
}

// blockedStartError preserves setup failure evidence when StartBlocked cannot
// return its already-started handle. Its snapshots are detached from that handle
// and fixed before returning; consumers must treat them as immutable.
type blockedStartError struct {
	cause         error
	HelperStarted bool
	BeforeExit    *int
	AfterExit     *int
	Stream        string
	Output        []byte
	CleanupError  string
}

func (e *blockedStartError) Error() string { return e.cause.Error() }
func (e *blockedStartError) Unwrap() error { return e.cause }

// failedBlockedStart samples without waiting on a running child. The callback
// retains the caller's existing cleanup policy (including whether it waits).
func failedBlockedStart(h Handle, cause error, cleanup func() error) error {
	e := &blockedStartError{HelperStarted: true, BeforeExit: observedLaunchExit(h), Stream: "unavailable"}
	snapshot := func() {
		if output, ok := h.(LaunchOutputHandle); ok {
			stream, raw := output.LaunchOutput()
			if stream != "" {
				e.Stream = stream
			}
			if len(raw) > launchOutputBytes {
				raw = raw[len(raw)-launchOutputBytes:]
			}
			e.Output = append([]byte(nil), raw...)
		}
	}
	snapshot()
	cleanupErr := cleanup()
	e.AfterExit = observedLaunchExit(h)
	snapshot()
	if cleanupErr != nil {
		e.CleanupError = boundLaunchFailureError(cleanupErr.Error())
	}
	e.cause = errors.Join(cause, cleanupErr)
	return e
}
