package scrollbackcmd

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
)

const (
	maxRawBytes     = 64 << 20
	maxEventsBytes  = 8 << 20
	maxRenderedRows = 100000
	maxDimension    = 1000
	maxHistoryCells = 1 << 20
	maxHistoryBytes = 128 << 20
	maxScreenCells  = 262144
)

// Keep every retention bound explicit: increasing only rows leaves VT's smaller
// live-view cell/byte caps active and can silently return a truncated tail.
func boundedReplayLimits(rowLimit int) vt.Limits {
	limits := vt.DefaultLimits()
	limits.MaxCells = maxScreenCells
	limits.HistoryLines = rowLimit + 1
	limits.HistoryCells = maxHistoryCells
	limits.HistoryBytes = maxHistoryBytes
	return limits
}

// RenderBoundedLines replays a recording under the CLI safety envelope. Full
// reads (maxLines <= 0) either return the complete retained history projection
// or an error; terminal commands that erase history retain their usual meaning.
// Like RenderLines, this is for short-lived processes; the caller owns the lease.
func RenderBoundedLines(rawPath, eventsPath string, maxLines int) ([]string, error) {
	return renderBoundedLines(rawPath, eventsPath, maxLines, maxRenderedRows)
}

func renderBoundedLines(rawPath, eventsPath string, maxLines, rowLimit int) ([]string, error) {
	raw, err := readBoundedRecording(rawPath, maxRawBytes)
	if err != nil {
		return nil, err
	}
	data, err := readBoundedRecording(eventsPath, maxEventsBytes)
	if err != nil {
		return nil, err
	}
	events, err := decodeEventData(data, true)
	if err != nil {
		return nil, fmt.Errorf("recording %s: %w", eventsPath, err)
	}
	var previousOffset int64
	for _, e := range events {
		if e.Offset < previousOffset || e.Offset > int64(len(raw)) {
			return nil, fmt.Errorf("recording %s: %s offset %d outside ordered raw range %d..%d", eventsPath, e.Type, e.Offset, previousOffset, len(raw))
		}
		previousOffset = e.Offset
		if e.Type == "resize" && (e.Cols <= 0 || e.Rows <= 0 || e.Cols > maxDimension || e.Rows > maxDimension) {
			return nil, fmt.Errorf("recording %s: resize %dx%d outside 1..%d", eventsPath, e.Cols, e.Rows, maxDimension)
		}
		if e.Type == "resize" && e.Cols > maxScreenCells/e.Rows {
			return nil, fmt.Errorf("recording %s: resize %dx%d exceeds %d screen cells", eventsPath, e.Cols, e.Rows, maxScreenCells)
		}
	}
	r, err := replayData(raw, events, true, rowLimit+1, rowLimit)
	if err != nil {
		return nil, fmt.Errorf("recording raw %s events %s: %w", rawPath, eventsPath, err)
	}
	// Preserve RenderLines' history cap, followed by the visible buffer.
	if maxLines > 0 && r.viewportTop-1 > maxLines {
		r.lines = r.lines[min(len(r.lines), r.viewportTop-1-maxLines):]
	}
	return r.lines, nil
}

// Read a finite snapshot of a regular file. NOFOLLOW and NONBLOCK close the
// lstat/open substitution race for links and FIFOs; fstat validates the actual
// descriptor. LimitReader prevents a growing producer from extending allocation.
func readBoundedRecording(path string, limit int64) ([]byte, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("recording %s: %w", path, err)
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("recording %s: %w", path, err)
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("recording %s: requires a regular file", path)
	}
	if st.Size() > limit {
		return nil, fmt.Errorf("recording %s: exceeds %d bytes", path, limit)
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, fmt.Errorf("recording %s: %w", path, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("recording %s: exceeds %d bytes", path, limit)
	}
	return data, nil
}

// replayOwnedCapture is the replay IO seam for deterministic collection ordering.
var replayOwnedCapture = RenderBoundedLines

// RenderOwnedCaptureLines selects the current capture for an empty token, or
// the exact parked token, and holds the existing retention lease through replay.
func RenderOwnedCaptureLines(dataDir, scope, tag, agent, token string, maxLines int) ([]string, error) {
	scoped, err := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: scope, Tag: tag})
	if err != nil {
		return nil, err
	}
	owner, err := storagegc.SelectedOwner(scoped.ScopeDir(), scope, tag)
	if err != nil {
		return nil, err
	}
	paths, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	if err != nil {
		return nil, err
	}
	var raw, events string
	if token == "" {
		capture, err := paths.ScrollbackArtifacts(agent)
		if err != nil {
			return nil, err
		}
		raw, events = capture.Raw, capture.Events
	} else {
		capture, err := paths.ParkedScrollbackArtifacts(token)
		if err != nil {
			return nil, err
		}
		raw, events = capture.Raw, capture.Events
	}
	env := map[string]string{"PAIR_DATA_DIR": scoped.ScopeDir(), "PAIR_SCOPE_KEY": scope, "PAIR_TAG": tag, "PAIR_AGENT": agent}
	lease, err := acquireRenderLease(func(key string) string { return env[key] }, raw, events)
	if err != nil {
		return nil, fmt.Errorf("retention raw %s events %s: %w", raw, events, err)
	}
	defer lease.Close()
	return replayOwnedCapture(raw, events, maxLines)
}
