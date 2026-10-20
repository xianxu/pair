package couchcmd

import (
	"time"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

// archiveWarningKey identifies a database by source/kind/path or a Drive root.
// pair:m5-concept pure
type archiveWarningKey struct {
	Source      historyarchive.SourceIdentity
	Kind, Path  string
	Destination string // canonical configured root, independent of physical folder ID
}

// archiveWarningGate reserves daily warnings for a single owner. Its zero value
// is ready for use. A new instance starts a new Couch lifetime; health changes
// and output failures do not release reservations.
// pair:m5-concept pure
type archiveWarningGate struct {
	day      time.Time
	keys     map[archiveWarningKey]struct{}
	overflow bool
}

// Reserve consumes admission before the caller attempts output. It retains only
// the latest UTC day's reservations, refusing earlier dates after clock rollback.
// overflow reports only the first new key beyond the daily bound.
func (g *archiveWarningGate) Reserve(now time.Time, key archiveWarningKey) (admit bool, overflow bool) {
	utc := now.UTC()
	day := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	if g.keys != nil && day.Before(g.day) {
		return false, false
	}
	if g.keys == nil || day.After(g.day) {
		g.day = day
		g.keys = make(map[archiveWarningKey]struct{})
		g.overflow = false
	}
	if _, exists := g.keys[key]; exists {
		return false, false
	}
	if len(g.keys) >= historyarchive.MaxFiles {
		if !g.overflow {
			g.overflow = true
			return false, true
		}
		return false, false
	}
	g.keys[key] = struct{}{}
	return true, false
}
