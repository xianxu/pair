package editorial

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Change replaces the half-open base byte interval with New.
type Change struct {
	Start int
	End   int
	New   string
}

// EditPolicy classifies one input change using the base snapshot's authority.
type EditPolicy struct {
	Change
	Effective    Effective
	Presentation string
}

// PolicyError is a located, actionable refusal with a stable machine code.
type PolicyError struct {
	Code   string
	Reason string
	Range  Range
}

func (e *PolicyError) Error() string {
	return fmt.Sprintf("%s at bytes %d:%d: %s", e.Code, e.Range.Start, e.Range.End, e.Reason)
}

type validationStats struct{ EditVisits, RegionComparisons, SourceComparisons, MarkerVisits, ChatComparisons int }

// ValidateEdits validates a complete sorted batch atomically, returning one
// classification per input change in the same order. It never obtains authority
// from candidate-written declarations.
func ValidateEdits(base []byte, policy *DocumentPolicy, changes []Change, catalog *Catalog) ([]EditPolicy, error) {
	out, _, e := validateEdits(base, policy, changes, catalog)
	return out, e
}
func validateEdits(base []byte, p *DocumentPolicy, changes []Change, c *Catalog) ([]EditPolicy, validationStats, error) {
	stats := validationStats{}
	fail := func(code, reason string, r Range) ([]EditPolicy, validationStats, error) {
		return nil, stats, &PolicyError{Code: code, Reason: reason, Range: r}
	}
	if p == nil || c == nil {
		return fail("editorial-invalid", "compiled base policy and catalog are required", Range{})
	}
	hash := sha256.Sum256(base)
	if hex.EncodeToString(hash[:]) != p.SourceHash {
		return fail("editorial-invalid", "policy does not belong to the supplied base snapshot", Range{})
	}
	previous := 0
	var candidate bytes.Buffer
	segments := []survivingSegment{}
	offset := 0
	for i, ch := range changes {
		stats.EditVisits++
		if ch.Start < previous || ch.End < ch.Start || ch.End > len(base) || (i > 0 && ch.Start == changes[i-1].Start) {
			return fail("editorial-invalid", "changes must be sorted, nonoverlapping base byte ranges", Range{ch.Start, ch.End})
		}
		if previous < ch.Start {
			segments = append(segments, survivingSegment{Range{previous, ch.Start}, previous + offset})
		}
		candidate.Write(base[previous:ch.Start])
		candidate.WriteString(ch.New)
		offset += len(ch.New) - (ch.End - ch.Start)
		previous = ch.End
	}
	if previous < len(base) {
		segments = append(segments, survivingSegment{Range{previous, len(base)}, previous + offset})
	}
	candidate.Write(base[previous:])
	// Classification and immutable-source intersection are monotone scans.
	result := make([]EditPolicy, 0, len(changes))
	regionIndex, ownedIndex := 0, 0
	for _, ch := range changes {
		r := Range{ch.Start, ch.End}
		for regionIndex+1 < len(p.Regions) && p.Regions[regionIndex].Range.End <= ch.Start {
			regionIndex++
		}
		if len(p.Regions) == 0 {
			return fail("editorial-invalid", "base policy has no regions", r)
		}
		effective := p.Regions[regionIndex].Effective
		for j := regionIndex + 1; j < len(p.Regions) && p.Regions[j].Range.Start < ch.End; j++ {
			stats.RegionComparisons++
			if !sameEffective(effective, p.Regions[j].Effective) {
				return fail("editorial-boundary", "one indivisible change crosses editorial policies; submit separate edits", r)
			}
		}
		for ownedIndex < len(p.Owned) && p.Owned[ownedIndex].Range.End <= ch.Start {
			ownedIndex++
			stats.SourceComparisons++
		}
		if ownedIndex < len(p.Owned) {
			stats.SourceComparisons++
			if overlapsChange(ch, p.Owned[ownedIndex].Range) {
				return fail("operator-owned", "editorial declarations are operator-owned", r)
			}
		}
		result = append(result, EditPolicy{Change: ch, Effective: effective, Presentation: effective.Presentation})
	}
	after, err := Compile(candidate.Bytes(), c)
	if err != nil {
		return fail("editorial-invalid", err.Error(), Range{})
	}
	if len(after.Owned) != len(p.Owned) {
		return fail("operator-owned", "agent changes may not insert, delete or detach editorial declarations", Range{})
	}
	afterOwned := map[int]OwnedSource{}
	for _, o := range after.Owned {
		afterOwned[o.Range.Start] = o
	}
	ownedMapper := positionMapper{changes: changes}
	for _, o := range p.Owned {
		stats.SourceComparisons++
		start, ok := ownedMapper.mapStart(o.Range.Start)
		if !ok {
			return fail("operator-owned", "editorial declaration was removed", o.Range)
		}
		got, exists := afterOwned[start]
		if !exists || got.Key != o.Key || !bytes.Equal(got.Raw, o.Raw) {
			return fail("operator-owned", "editorial declaration or its attachment changed", o.Range)
		}
	}
	// Heading identity and direct-parent identity protect declaration associations,
	// even when the candidate happens to retain the same scalar values.
	headingByStart := map[int]int{}
	for i, h := range after.Headings {
		headingByStart[h.Range.Start] = i
	}
	headingMapper := positionMapper{changes: changes}
	mappedHeadings := make([]int, len(p.Headings))
	for i, h := range p.Headings {
		mappedHeadings[i] = -1
		if start, ok := headingMapper.mapStart(h.Range.Start); ok {
			if index, exists := headingByStart[start]; exists {
				mappedHeadings[i] = index
			}
		}
	}
	for i, h := range p.Headings {
		if h.Directive == nil {
			continue
		}
		j := mappedHeadings[i]
		if j < 0 {
			return fail("operator-owned", "a heading owning an editorial declaration was removed or moved", h.Range)
		}
		target := after.Headings[j]
		wantParent := -1
		if h.Parent >= 0 {
			wantParent = mappedHeadings[h.Parent]
			if wantParent < 0 {
				return fail("operator-owned", "a declaration owning ancestor was removed", h.Range)
			}
		}
		if target.Name != h.Name || target.Level != h.Level || target.Parent != wantParent {
			return fail("operator-owned", "a declaration's heading may not be renamed, moved or reparented", h.Range)
		}
	}
	// Every surviving byte retains base authority. This catches structural heading
	// edits that move old text under a different override without touching settings.
	bi, ai := 0, 0
	for _, seg := range segments {
		pos := seg.base.Start
		for pos < seg.base.End {
			for bi+1 < len(p.Regions) && p.Regions[bi].Range.End <= pos {
				bi++
			}
			mapped := seg.afterStart + pos - seg.base.Start
			for ai+1 < len(after.Regions) && after.Regions[ai].Range.End <= mapped {
				ai++
			}
			stats.RegionComparisons++
			if !sameEffective(p.Regions[bi].Effective, after.Regions[ai].Effective) {
				return fail("editorial-boundary", "heading changes may not rescope surviving base content", Range{pos, seg.base.End})
			}
			end := min(seg.base.End, p.Regions[bi].Range.End)
			end = min(end, pos+after.Regions[ai].Range.End-mapped)
			if end <= pos {
				return fail("editorial-invalid", "invalid compiled interval table", Range{pos, pos})
			}
			pos = end
		}
	}
	markers := ScanMarkers(string(base))
	visible := markers[:0]
	codeIndex := 0
	for _, m := range markers {
		stats.MarkerVisits++
		for codeIndex < len(p.CodeRanges) && p.CodeRanges[codeIndex].End <= m.Range.Start {
			codeIndex++
		}
		if codeIndex < len(p.CodeRanges) && p.CodeRanges[codeIndex].Start <= m.Range.Start {
			continue
		}
		visible = append(visible, m)
	}
	markers = visible
	blocks, chatItems := indexBlocks(base, p)
	refs := []Range{}
	readyStarts := []int{}
	pending := []Range{}
	blockIndex := 0
	for _, m := range markers {
		if m.Pending {
			pending = append(pending, m.Range)
			continue
		}
		for blockIndex+1 < len(blocks) && blocks[blockIndex].End <= m.Range.Start {
			blockIndex++
		}
		start := m.Range.Start
		if !m.HasReference && len(blocks) > 0 {
			block := blocks[blockIndex]
			if block.Start <= m.Range.Start {
				start = block.Start
				if len(bytes.TrimSpace(base[start:m.Range.Start])) == 0 && blockIndex > 0 {
					start = blocks[blockIndex-1].Start
				}
			}
		}
		if m.HasReference {
			start = m.Range.Start
		}
		refs = append(refs, Range{start, m.Range.End})
		readyStarts = append(readyStarts, start)
	}
	// Ready referrals can overlap; merge once so all membership scans stay linear.
	refs = mergeRanges(refs)
	domain := processingDomains(p)
	frontiers := map[int]int{}
	markerRegion := 0
	readyIndex := 0
	for _, m := range markers {
		if !m.Ready {
			continue
		}
		for markerRegion+1 < len(p.Regions) && p.Regions[markerRegion].Range.End <= m.Range.Start {
			markerRegion++
		}
		id := domain[markerRegion]
		start := readyStarts[readyIndex]
		readyIndex++
		if old, ok := frontiers[id]; !ok || start < old {
			frontiers[id] = start
		}
	}
	pendingIndex, refIndex, chatIndex, editRegion := 0, 0, 0, 0
	for i, ch := range changes {
		r := Range{ch.Start, ch.End}
		for pendingIndex < len(pending) && pending[pendingIndex].End <= ch.Start {
			pendingIndex++
		}
		if pendingIndex < len(pending) && overlapsChange(ch, pending[pendingIndex]) {
			return fail("editorial-pending", "pending agent markers must remain unchanged", r)
		}
		for refIndex < len(refs) && refs[refIndex].End < ch.Start {
			refIndex++
		}
		inReference := refIndex < len(refs) && containsChange(refs[refIndex], ch)
		for chatIndex < len(chatItems) && chatItems[chatIndex].Range.End <= ch.Start {
			chatIndex++
		}
		chatAnswer := false
		if chatIndex < len(chatItems) {
			item := chatItems[chatIndex]
			stats.ChatComparisons++
			if overlapsChange(ch, item.Range) || ch.Start == item.AppendAt || ch.Start == item.Range.Start {
				if !inReference {
					if ch.Start != ch.End || ch.Start != item.AppendAt || item.Linked || !answerAppend.MatchString(ch.New) {
						return fail("editorial-chat", "preserve each chat ask; only append its answer link", r)
					}
					chatAnswer = true
				}
			}
		}
		for editRegion+1 < len(p.Regions) && p.Regions[editRegion].Range.End <= ch.Start {
			editRegion++
		}
		effective := result[i].Effective
		if !chatAnswer && !inReference {
			if effective.Scope == MarkersOnly {
				return fail("editorial-scope", "this processing mode permits changes only to ready marker referrals", r)
			}
			if effective.Frontier {
				if frontier, ok := frontiers[domain[editRegion]]; ok && ch.Start < frontier {
					return fail("editorial-scope", "text above the first ready marker's referral is settled", r)
				}
			}
		}
	}
	return result, stats, nil
}

type survivingSegment struct {
	base       Range
	afterStart int
}
type positionMapper struct {
	changes      []Change
	index, delta int
}

func (m *positionMapper) mapStart(pos int) (int, bool) {
	for m.index < len(m.changes) && m.changes[m.index].End <= pos {
		ch := m.changes[m.index]
		m.delta += len(ch.New) - (ch.End - ch.Start)
		m.index++
	}
	if m.index < len(m.changes) {
		ch := m.changes[m.index]
		if ch.Start <= pos && pos < ch.End {
			return 0, false
		}
	}
	return pos + m.delta, true
}
func sameEffective(a, b Effective) bool {
	return a.Collaboration == b.Collaboration && a.Presentation == b.Presentation && a.Processing == b.Processing && a.Instruction == b.Instruction && a.Scope == b.Scope && a.Frontier == b.Frontier
}
func overlapsChange(c Change, r Range) bool {
	if c.Start == c.End {
		return c.Start > r.Start && c.Start < r.End
	}
	return c.Start < r.End && c.End > r.Start
}
func containsChange(r Range, c Change) bool { return c.Start >= r.Start && c.End <= r.End }
func mergeRanges(in []Range) []Range {
	out := []Range{}
	for _, r := range in {
		if len(out) > 0 && r.Start <= out[len(out)-1].End {
			if r.Start < out[len(out)-1].Start {
				out[len(out)-1].Start = r.Start
			}
			out[len(out)-1].End = max(out[len(out)-1].End, r.End)
		} else {
			out = append(out, r)
		}
	}
	return out
}
func processingDomains(p *DocumentPolicy) []int {
	ids := make([]int, len(p.Regions))
	for i, r := range p.Regions {
		ids[i] = -1
		if r.Heading < 0 {
			continue
		}
		h := p.Headings[r.Heading]
		if h.Parent >= 0 {
			ids[i] = ids[h.Parent+1]
		}
		if _, ok := h.Overrides["processing-mode"]; ok {
			ids[i] = r.Heading
		}
	}
	return ids
}

var answerAppend = regexp.MustCompile(`^ → \[answer\]\([^\s()]+\)$`)

type chatItem struct {
	Range    Range
	AppendAt int
	Linked   bool
}

func indexBlocks(base []byte, p *DocumentPolicy) ([]Range, []chatItem) {
	items := []chatItem{}
	region := 0
	for _, r := range p.ListItems {
		for region+1 < len(p.Regions) && p.Regions[region].Range.End <= r.Start {
			region++
		}
		if p.Regions[region].Effective.Collaboration != "chat" {
			continue
		}
		end := r.End
		for end > r.Start && (base[end-1] == '\n' || base[end-1] == '\r') {
			end--
		}
		items = append(items, chatItem{Range: r, AppendAt: end, Linked: strings.Contains(string(base[r.Start:r.End]), "→ [answer](") || strings.Contains(string(base[r.Start:r.End]), "→ done:")})
	}
	return p.Blocks, items
}
