package editorial

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Migration contains the verified candidate bytes and the base snapshot digest.
type Migration struct {
	BeforeHash  string       `json:"beforeHash"`
	After       []byte       `json:"after"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}
type sourceReplacement struct {
	r     Range
	after []byte
}

// PlanMigration replaces only a legacy collaboration map with explicit tags.
// Shared scan facts select headings; the compiled eligible-item projection must
// survive exactly. Refusals return no candidate and never renumber asks.
func PlanMigration(src []byte) (*Migration, error) {
	d, err := Compile(src)
	if err != nil {
		return nil, err
	}
	result := &Migration{BeforeHash: d.SourceHash, After: bytes.Clone(src), Diagnostics: d.Diagnostics}
	field, ok := d.parsed.fields["collaboration-mode"]
	if !ok || field.value.Kind != yaml.MappingNode {
		return result, nil
	}
	for _, notice := range d.scan.Notices {
		if strings.HasPrefix(notice, "collaboration-mode map name ") && strings.HasSuffix(notice, " matches no heading") {
			return nil, fmt.Errorf("migration refused: %s", notice)
		}
	}
	reps := []sourceReplacement{{r: field.Range}}
	selected := make(map[int]bool)
	whole := false
	for _, section := range d.scan.Sections {
		if section.Heading == 0 {
			whole = true
			continue
		}
		if section.Source == "legacy-map" {
			selected[section.Heading] = true
		}
	}
	// A whole-document legacy default can be expressed by tags on root
	// headings only; the item proof below rejects unrepresentable preamble asks.
	if whole {
		for i, h := range d.Headings {
			if h.Parent < 0 {
				selected[d.scan.Headings[i].Line] = true
			}
		}
	}
	tags := make(map[int]bool, len(d.scan.Tags))
	for _, line := range d.scan.Tags {
		tags[line+1] = true
	}
	lines := sourceLines(src)
	for i, h := range d.scan.Headings {
		if !selected[h.Line] || tags[h.Line] {
			continue
		}
		at := d.Headings[i].Range.Start
		reps = append(reps, sourceReplacement{r: Range{at, at}, after: []byte("@@chat@@" + d.parsed.newline)})
	}
	// The frontmatter removal precedes source-ordered heading insertions.
	var out bytes.Buffer
	pos := 0
	for _, r := range reps {
		out.Write(src[pos:r.r.Start])
		out.Write(r.after)
		pos = r.r.End
	}
	out.Write(src[pos:])
	candidate := out.Bytes()
	after, err := Compile(candidate)
	if err != nil {
		return nil, err
	}
	// Each proof walks source offsets monotonically. A fresh cursor per
	// projection avoids rescanning every insertion for every source line.
	translator := func() func(int, bool) int {
		cursor, delta := 0, 0
		return func(offset int, end bool) int {
			for cursor < len(reps) {
				r := reps[cursor]
				if offset < r.r.End || (end && r.r.Start == r.r.End && offset == r.r.Start) {
					break
				}
				delta += len(r.after) - (r.r.End - r.r.Start)
				cursor++
			}
			return offset + delta
		}
	}
	sectionName := func(offset int) string {
		name := "preamble"
		for _, h := range d.Headings {
			if h.Range.Start > offset {
				break
			}
			name = strings.TrimSpace(h.Name)
		}
		return name
	}
	refuse := func(offset int, reason string) (*Migration, error) {
		return nil, fmt.Errorf("migration refused in section %q: %s", sectionName(offset), reason)
	}
	// Compare real heading identity and source placement rather than parsing a
	// second set of legacy routing rules. Setext-to-tag attempts fail this proof.
	if len(d.scan.Headings) != len(after.scan.Headings) {
		return nil, fmt.Errorf("migration refused: heading structure changed")
	}
	translate := translator()
	for i, h := range d.Headings {
		next := after.Headings[i]
		if h.Name != next.Name || h.Level != next.Level || translate(h.Range.Start, false) != next.Range.Start {
			return refuse(h.Range.Start, "tag cannot preserve this heading")
		}
	}
	// Preserve chat coverage on original body lines under real headings. The
	// preamble has no tag attachment; eligible asks there are checked below.
	region, nextRegion := 0, 0
	headingLines := make(map[int]bool, len(d.scan.Headings))
	for _, h := range d.scan.Headings {
		headingLines[h.Line] = true
	}
	tagLines := make(map[int]bool, len(d.scan.Tags))
	for _, line := range d.scan.Tags {
		tagLines[line] = true
	}
	translate = translator()
	for i, line := range lines {
		if line.Start < d.parsed.body || len(d.Headings) == 0 || line.Start < d.Headings[0].Range.Start || headingLines[i+1] || tagLines[i+1] {
			continue
		}
		for region+1 < len(d.Regions) && d.Regions[region].Range.End <= line.Start {
			region++
		}
		at := translate(line.Start, false)
		for nextRegion+1 < len(after.Regions) && after.Regions[nextRegion].Range.End <= at {
			nextRegion++
		}
		if (d.Regions[region].Effective.Collaboration == "chat") != (after.Regions[nextRegion].Effective.Collaboration == "chat") {
			return refuse(line.Start, "chat section extent would change")
		}
	}
	translate = translator()
	beforeItems, afterItems := d.items, after.items
	for i, item := range beforeItems {
		if i >= len(afterItems) {
			return refuse(item.Range.Start, "eligible chat ask would be lost; keep legacy settings or intentionally edit the ask")
		}
		next := afterItems[i]
		if translate(item.Range.Start, false) != next.Range.Start || translate(item.AppendAt, false) != next.AppendAt || translate(item.Range.End, true) != next.Range.End || item.Linked != next.Linked || !bytes.Equal(src[item.Range.Start:item.Range.End], candidate[next.Range.Start:next.Range.End]) {
			return refuse(item.Range.Start, "eligible chat asks would change; keep legacy settings or intentionally edit the asks")
		}
	}
	if len(afterItems) > len(beforeItems) {
		return nil, fmt.Errorf("migration refused: eligible chat asks would be added")
	}
	result.After = bytes.Clone(candidate)
	return result, nil
}
