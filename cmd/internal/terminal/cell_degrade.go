package terminal

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// backendCellFault retains only the first cell replaced in one owned row.
// It never escapes the publication capture except as bounded JSON evidence.
type backendCellFault struct {
	column int
	cell   Cell
	reason string
}

// repairBackendRow returns a repaired row without changing its input. A valid
// row is returned unchanged; a faulty row is copied once before repair.
// Walking backwards means an unsafe continuation is repaired before its wide
// head is checked. Thus a broken pair needs no rescan or retained fault set.
// inspected records the linear cell-work bound, independent of fault count.
func repairBackendRow(row []Cell) (repaired []Cell, first *backendCellFault, inspected int) {
	repaired = row
	for x := len(row) - 1; x >= 0; x-- {
		inspected++
		if len(repaired[x].Content) > MaxClusterBytes || validateCell(repaired, x) != nil {
			c := repaired[x]
			if first == nil {
				repaired = append([]Cell(nil), row...)
				first = &backendCellFault{}
			}
			*first = backendCellFault{column: x, cell: c, reason: cellFaultReason(c, repaired, x)}
			repaired[x] = Cell{Content: "�", Width: 1, Style: c.Style}
		}
	}
	return repaired, first, inspected
}

func cellFaultReason(c Cell, row []Cell, x int) string {
	switch {
	case len(c.Content) > MaxClusterBytes:
		return "cluster-limit"
	case !utf8.ValidString(c.Content):
		return "invalid-utf8"
	case !plainText(c.Content):
		return "control-content"
	case !validLink(c):
		return "unsafe-link"
	case c.Width < 0 || c.Width > 2:
		return "unsupported-width"
	case c.Width == 2 && (x+1 == len(row) || row[x+1].Width != 0 || row[x+1].Content != ""):
		return "broken-wide-cell"
	default:
		return "content-width-mismatch"
	}
}

// describeCellFault encodes a bounded, printable description. Content samples
// contain at most 64 codepoints; controls use literal Unicode escape notation.
// At most 256 codepoints are scanned (1024 bytes). If content_scan_clipped is
// true, codepoint and invalid-byte counts are lower bounds. Original byte
// lengths remain exact. Hyperlinks contribute lengths only, never raw data.
func describeCellFault(source string, row int, historyID uint64, fault backendCellFault) []byte {
	c := fault.cell
	var prefix strings.Builder
	var invalid [32]byte
	codepoints, invalidCount, pos := 0, 0, 0
	values := make([]string, 0, 64)
	for pos < len(c.Content) && codepoints < 256 {
		r, n := utf8.DecodeRuneInString(c.Content[pos:])
		if r == utf8.RuneError && n == 1 {
			if invalidCount < len(invalid) {
				invalid[invalidCount] = c.Content[pos]
			}
			invalidCount++
		}
		if codepoints < 64 {
			values = append(values, fmt.Sprintf("U+%04X", r))
			if unicode.IsControl(r) {
				fmt.Fprintf(&prefix, "\\u%04x", r)
			} else {
				prefix.WriteRune(r)
			}
		}
		codepoints++
		pos += n
	}
	evidence := struct {
		Source                 string   `json:"source"`
		Row                    int      `json:"row"`
		Column                 int      `json:"column"`
		HistoryRowID           uint64   `json:"history_row_id,omitempty"`
		Reason                 string   `json:"reason"`
		Width                  int      `json:"width"`
		ContentBytes           int      `json:"content_bytes"`
		ContentCodepoints      int      `json:"content_codepoints"`
		ContentPrefix          string   `json:"content_prefix"`
		ContentCodepointValues []string `json:"content_codepoint_values"`
		ContentScannedBytes    int      `json:"content_scanned_bytes"`
		ContentScanClipped     bool     `json:"content_scan_clipped"`
		ContentClipped         bool     `json:"content_clipped"`
		InvalidBytesHex        string   `json:"invalid_bytes_hex"`
		InvalidBytes           int      `json:"invalid_bytes"`
		InvalidBytesClipped    bool     `json:"invalid_bytes_clipped"`
		LinkURLBytes           int      `json:"link_url_bytes"`
		LinkParamsBytes        int      `json:"link_params_bytes"`
	}{source, row, fault.column, historyID, fault.reason, c.Width, len(c.Content), codepoints, prefix.String(), values, pos, pos < len(c.Content), codepoints > 64, hex.EncodeToString(invalid[:min(invalidCount, len(invalid))]), invalidCount, invalidCount > len(invalid) || pos < len(c.Content), len(c.Link.URL), len(c.Link.Params)}
	data, _ := json.Marshal(evidence) // All fields are finite integers or strings.
	return data
}

// repairCapturedRow observes at most one fault for the endpoint lifetime.
// The observer contract admits only a nonblocking enqueue under the mutex.
func (e *Endpoint) repairCapturedRow(row []Cell, source string, y int, historyID uint64) {
	repaired, fault, _ := repairBackendRow(row)
	if fault != nil {
		copy(row, repaired)
	}
	if fault != nil && !e.cellDegraded {
		e.cellDegraded = true
		if e.observer != nil {
			e.observe("cell-degraded", describeCellFault(source, y, historyID, *fault))
		}
	}
}
