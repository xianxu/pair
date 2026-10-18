package sessionsync

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// EditRecord is one precise edit in the review pane's format (pair#661):
// replace the Occurrence-th plain match of Old in the base text with New.
// Explain is the reason shown beside the change.
type EditRecord struct {
	Old        string
	Occurrence int
	New        string
	Explain    string
}

// ResolvedEdit is a record that resolved: its index in the batch (1-based),
// the byte range of its Old in the base text, and its replacement.
type ResolvedEdit struct {
	Index              int
	BaseStart, BaseEnd int
	New, Explain       string
}

// BadEdit is a record that did not resolve, and why: `empty old`,
// `not found` or `overlap`, the review pane's reasons.
type BadEdit struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

type wireEdit struct {
	Old        *string      `json:"old"`
	Occurrence *json.Number `json:"occurrence"`
	New        *string      `json:"new"`
	Explain    string       `json:"explain"`
}

// ParseEdits reads EDITS.json in either handoff form, `[records]` or
// `{"records": [records]}`. A record needs string old and new; occurrence,
// when present, is an integer and defaults to 1.
func ParseEdits(raw []byte) ([]EditRecord, error) {
	raw = bytes.TrimSpace(raw)
	var wire []wireEdit
	if len(raw) > 0 && raw[0] == '{' {
		var wrapped struct {
			Records *[]wireEdit `json:"records"`
		}
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return nil, fmt.Errorf("edits: %w", err)
		}
		if wrapped.Records == nil {
			return nil, errors.New(`edits: an object needs a "records" array`)
		}
		wire = *wrapped.Records
	} else if err := json.Unmarshal(raw, &wire); err != nil {
		return nil, fmt.Errorf("edits: %w", err)
	}
	records := make([]EditRecord, len(wire))
	for i, w := range wire {
		if w.Old == nil || w.New == nil {
			return nil, fmt.Errorf("edits: record %d needs string old and new", i+1)
		}
		occurrence := 1
		if w.Occurrence != nil {
			n, err := w.Occurrence.Int64()
			if err != nil {
				return nil, fmt.Errorf("edits: record %d occurrence must be an integer", i+1)
			}
			if n < 1 {
				return nil, fmt.Errorf("edits: record %d occurrence must be 1 or more", i+1)
			}
			occurrence = int(n)
		}
		records[i] = EditRecord{Old: *w.Old, Occurrence: occurrence, New: *w.New, Explain: w.Explain}
	}
	return records, nil
}

// nthOffset is the byte offset of the n-th (1-based) plain, non-overlapping
// match of needle in hay, or -1 (the Lua resolver's nth_offset).
func nthOffset(hay, needle string, n int) int {
	if n < 1 {
		return -1
	}
	from, found := 0, -1
	for i := 0; i < n; i++ {
		at := strings.Index(hay[from:], needle)
		if at < 0 {
			return -1
		}
		found = from + at
		from = found + len(needle)
	}
	return found
}

// ResolveEdits resolves every record against base, all or nothing. With no
// bad record it returns the new text and the resolved edits in base order;
// otherwise next is unused and bad lists every failing record by index. The
// fixtures in testdata/edit-records pin these rules for both resolvers.
func ResolveEdits(base string, records []EditRecord) (next string, resolved []ResolvedEdit, bad []BadEdit) {
	for i, r := range records {
		switch off := nthOffset(base, r.Old, r.Occurrence); {
		case r.Old == "":
			bad = append(bad, BadEdit{i + 1, "empty old"})
		case off < 0:
			bad = append(bad, BadEdit{i + 1, "not found"})
		default:
			resolved = append(resolved, ResolvedEdit{Index: i + 1, BaseStart: off, BaseEnd: off + len(r.Old), New: r.New, Explain: r.Explain})
		}
	}
	sort.SliceStable(resolved, func(a, b int) bool { return resolved[a].BaseStart < resolved[b].BaseStart })
	kept, prevEnd := resolved[:0], 0
	for _, e := range resolved {
		if e.BaseStart < prevEnd {
			bad = append(bad, BadEdit{e.Index, "overlap"})
			continue
		}
		kept, prevEnd = append(kept, e), e.BaseEnd
	}
	if len(bad) != 0 {
		sort.Slice(bad, func(a, b int) bool { return bad[a].Index < bad[b].Index })
		return "", nil, bad
	}
	return applyResolved(base, kept), kept, nil
}

// applyResolved writes resolved, non-overlapping edits (in base order) into base.
func applyResolved(base string, edits []ResolvedEdit) string {
	var b strings.Builder
	at := 0
	for _, e := range edits {
		b.WriteString(base[at:e.BaseStart])
		b.WriteString(e.New)
		at = e.BaseEnd
	}
	b.WriteString(base[at:])
	return b.String()
}
