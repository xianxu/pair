// Package terminal owns the terminal connection shared by Pair's compositors.
package terminal

import (
	"fmt"
	"time"
)

const (
	MaxCells         = 262144
	MaxStringBytes   = 64 << 10
	MaxParams        = 32
	MaxClusterBytes  = 256
	MaxTitleBytes    = 4096
	MaxLinkBytes     = 2048
	MaxHistoryLines  = 1000
	MaxHistoryCells  = 65536
	MaxKeyboardStack = 16
	MaxInputPackets  = 128
	MaxInputBytes    = 1 << 20
	MaxPasteBytes    = 1 << 20
	MaxPendingEvents = 128
	WriteTimeout     = 2 * time.Second
	FrameInterval    = 16 * time.Millisecond
	SyncTimeout      = 150 * time.Millisecond
)

// Geometry contains virtual cells, excluding any compositor chrome.
type Geometry struct{ Cols, Rows int }

func (g Geometry) Validate() error {
	if g.Cols < 1 || g.Rows < 1 || g.Cols > MaxCells || g.Rows > MaxCells/g.Cols {
		return fmt.Errorf("terminal: invalid geometry %dx%d (maximum %d cells)", g.Cols, g.Rows, MaxCells)
	}
	return nil
}

// Profile is the explicit connection contract. Environment and query handlers
// must use the same profile; TERM is a terminfo entry, not a claim about the host.
type Profile struct {
	TERM          string
	ClipboardRead bool
	Graphics      bool
}
