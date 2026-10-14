package couchcore

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// Layout is which pair layout couch launches its threads in. It is chosen once
// per couch process (Couch.Layout) and recorded per thread
// (ThreadRecord.Layout) as a witness of what that thread's session actually is.
//
// It is an ALIAS for launcher.LayoutMode, not a second type: launcher already
// owns this vocabulary, its parse, and the argv spellings that `pair` accepts.
// couch's whole feature rests on pair parsing the flag couch emits, so the two
// must not be able to drift -- see TestCouchLayoutFlagsAreWhatPairParses.
type Layout = launcher.LayoutMode

const (
	Layout2 = launcher.Layout2
	Layout3 = launcher.Layout3
	// DefaultLayout is the layout for a new Couch process, not legacy records.
	DefaultLayout = Layout3
	// LayoutUnknown is a persisted value this binary does not recognise. It is
	// never chosen and never formatted: it exists so a row can carry "we cannot
	// prove this session's layout" instead of fabricating layout2.
	LayoutUnknown Layout = "unknown"
)

// ParseLayout turns an untrusted string -- a CLI flag or a persisted field --
// into a Layout. Empty means a record written before layout was recorded, and
// those are layout2 with certainty: couch pinned layout2 from 2026-08-22 until
// #198. An unrecognised value is refused rather than defaulted.
func ParseLayout(raw string) (Layout, error) {
	if strings.TrimSpace(raw) == "" {
		return Layout2, nil
	}
	layout, ok := launcher.ParseLayoutMode(raw)
	if !ok {
		return "", fmt.Errorf("unknown layout %q", raw)
	}
	return layout, nil
}

// NormalizeLayout is ParseLayout for the one caller that cannot return an
// error: ProjectActionableThreads. An unrecognised value becomes LayoutUnknown
// rather than a default, preserving the uncertainty in the projection.
func NormalizeLayout(raw string) Layout {
	layout, err := ParseLayout(raw)
	if err != nil {
		return LayoutUnknown
	}
	return layout
}
