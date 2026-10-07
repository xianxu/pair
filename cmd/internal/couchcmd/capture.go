package couchcmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	term "github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/terminalcapture"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

// capturePath validates an explicit per-process opt-in. Regular Couch accepts
// an absolute destination; an isolated runtime retains its confinement rule.
func capturePath(getenv func(string) string) (string, error) {
	path := getenv("COUCH_CAPTURE_DIR")
	if path == "" {
		return "", nil
	}
	root := getenv("COUCH_ISOLATED_ROOT")
	if root == "" {
		canonical, err := canonicalFuture(path)
		if err != nil {
			return "", fmt.Errorf("COUCH_CAPTURE_DIR: %w", err)
		}
		return canonical, nil
	}
	root, err := canonicalFuture(root)
	if err != nil {
		return "", fmt.Errorf("COUCH_CAPTURE_DIR: %w", err)
	}
	return confinedDirectory(root, path, "COUCH_CAPTURE_DIR")
}

// captureSettings keeps validation identical at singleton admission and console
// composition. A limit alone never enables capture.
func captureSettings(getenv func(string) string) (string, terminalcapture.Config, error) {
	path, err := capturePath(getenv)
	config := terminalcapture.Config{}
	if err != nil || path == "" {
		return path, config, err
	}
	raw := getenv("COUCH_CAPTURE_MAX_MIB")
	if raw == "" {
		return path, config, nil
	}
	valid := true
	for _, c := range raw {
		if c < '0' || c > '9' {
			valid = false
			break
		}
	}
	n, parseErr := strconv.ParseUint(raw, 10, 64)
	if !valid || parseErr != nil || n < 1 || n > 1<<20 {
		return "", config, fmt.Errorf("COUCH_CAPTURE_MAX_MIB must be an integer from 1 to 1048576")
	}
	config.MaxBytes = int64(n) << 20
	return path, config, nil
}

// captureWriter observes the transport's receipt, not just the requested bytes.
// The recorder copies borrowed bytes before returning and never performs disk IO
// on this path. Its failures do not change the terminal's write result.
type captureWriter struct {
	writer   ttyio.Writer
	recorder *terminalcapture.Recorder
}

func (w captureWriter) WriteContext(ctx context.Context, p []byte) (int, error) {
	start := time.Now()
	n, err := w.writer.WriteContext(ctx, p)
	event := terminalcapture.Record{Kind: "host-write", Data: p, Requested: len(p), Accepted: n, DurationNS: time.Since(start).Nanoseconds()}
	if err != nil {
		event.Error = err.Error()
	}
	w.recorder.Record(event)
	return n, err
}

// Embedding the concrete host preserves ReadContext and Terminated capabilities.
// Close only closes the host. Recorder drain happens after terminal restoration.
type captureHost struct {
	*hostty.OSHost
	recorder *terminalcapture.Recorder
}

func (h *captureHost) WriteContext(ctx context.Context, p []byte) (int, error) {
	return (captureWriter{h.OSHost, h.recorder}).WriteContext(ctx, p)
}
func (h *captureHost) Write(p []byte) (int, error) { return h.WriteContext(context.Background(), p) }
func (h *captureHost) Size() (ptychild.Size, error) {
	size, err := h.OSHost.Size()
	event := terminalcapture.Record{Kind: "host-geometry", Rows: int(size.Rows), Cols: int(size.Cols)}
	if err != nil {
		event.Error = err.Error()
	}
	h.recorder.Record(event)
	return size, err
}
func captureObserver(rec *terminalcapture.Recorder) term.Observer {
	if rec == nil {
		return nil
	}
	return func(o term.Observation) {
		rec.Record(terminalcapture.Record{Kind: o.Kind, EndpointID: o.EndpointID, Rows: o.Geometry.Rows, Cols: o.Geometry.Cols, Data: o.Data})
	}
}
