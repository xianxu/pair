package couchcore

import (
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
