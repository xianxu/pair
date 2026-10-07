package couchtty

import (
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/terminalcapture"
)

// captureBadge renders only fixed labels, never filesystem or writer error text.
func captureBadge(status terminalcapture.Status) string {
	switch status.Phase {
	case terminalcapture.Recording:
		percent := int64(0)
		if status.LimitBytes > 0 {
			percent = status.WrittenBytes * 100 / status.LimitBytes
		}
		return fmt.Sprintf("REC %d%%", percent)
	case terminalcapture.Draining:
		return "REC draining"
	case terminalcapture.Closed:
		return "REC closed"
	case terminalcapture.Failed:
		switch {
		case errors.Is(status.Err, terminalcapture.ErrQueueFull):
			return "REC STOP:queue"
		case errors.Is(status.Err, terminalcapture.ErrFileLimit):
			return "REC STOP:full"
		default:
			return "REC STOP:IO"
		}
	default:
		return ""
	}
}

// SetCapture is composition-time injection, before any children are started.
func (c *Console) SetCapture(recorder *terminalcapture.Recorder) { c.capture = recorder }

// CloseCapture also covers launch failures before Run owns terminal teardown.
// The command owner calls this after Run has restored terminal modes, or on
// early launch failure. The command owner also reports the returned error.
func (c *Console) CloseCapture() error { return c.capture.Close() }

// traceCaptureTransition brackets a requested presentation change; it does not
// claim that a host has physically painted the resulting bytes.
func (c *Console) traceCaptureTransition(kind, id string, err error) {
	if c.capture == nil {
		return
	}
	event := terminalcapture.Record{Kind: kind, EndpointID: id}
	if err != nil {
		event.Error = err.Error()
	}
	c.capture.Record(event)
}
