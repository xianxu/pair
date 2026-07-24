package launcher

import "strings"

// LayoutMode is the Pair workbench topology selected for one tag.
type LayoutMode string

const (
	Layout2 LayoutMode = "layout2"
	Layout3 LayoutMode = "layout3"
)

// LayoutRequest preserves whether the operator explicitly selected a topology.
// An implicit zero request is resolved from the tag record, then defaults to
// Layout2.
type LayoutRequest struct {
	Mode     LayoutMode
	Explicit bool
}

// LayoutResolution is the selected topology plus whether an explicit request
// disagrees with an existing topology.
type LayoutResolution struct {
	Mode     LayoutMode
	Conflict bool
}

// ParseLayoutMode decodes the durable workbench-layout record.
func ParseLayoutMode(raw string) (LayoutMode, bool) {
	switch LayoutMode(strings.TrimSpace(raw)) {
	case Layout2:
		return Layout2, true
	case Layout3:
		return Layout3, true
	default:
		return "", false
	}
}

// ResolveLayout applies explicit request > valid record > Layout2.
func ResolveLayout(request LayoutRequest, recorded LayoutMode, recordValid bool) LayoutResolution {
	if request.Explicit {
		return LayoutResolution{
			Mode:     request.Mode,
			Conflict: recordValid && recorded != request.Mode,
		}
	}
	if recordValid {
		return LayoutResolution{Mode: recorded}
	}
	return LayoutResolution{Mode: Layout2}
}

func LayoutAssetBasename(mode LayoutMode) string {
	if mode == Layout3 {
		return "main-3.kdl"
	}
	return "main-2.kdl"
}
