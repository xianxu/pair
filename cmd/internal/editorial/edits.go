package editorial

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/xianxu/ariadne/pkg/sidedoc"
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

type validationStats struct {
	TailJoinVisits, ItemJoinVisits                                                  int
	EditVisits, RegionComparisons, SourceComparisons, MarkerVisits, ChatComparisons int
}

// ValidateEdits validates a complete sorted batch atomically, returning one
// classification per input change in the same order. It never obtains authority
// from candidate-written declarations.
func ValidateEdits(base []byte, policy *DocumentPolicy, changes []Change) ([]EditPolicy, error) {
	out, _, e := validateEdits(base, policy, changes)
	return out, e
}
func validateEdits(base []byte, p *DocumentPolicy, changes []Change) ([]EditPolicy, validationStats, error) {
	stats := validationStats{}
	fail := func(code, reason string, r Range) ([]EditPolicy, validationStats, error) {
		return nil, stats, &PolicyError{Code: code, Reason: reason, Range: r}
	}
	if p == nil {
		return fail("editorial-invalid", "compiled base policy is required", Range{})
	}
	hash := sha256.Sum256(base)
	if hex.EncodeToString(hash[:]) != p.SourceHash {
		return fail("editorial-invalid", "policy does not belong to the supplied base snapshot", Range{})
	}
	previous := 0
	var candidate bytes.Buffer
	segments := []survivingSegment{}
	offset := 0
	candidateChanges := make([]Range, 0, len(changes))
	for i, ch := range changes {
		stats.EditVisits++
		if ch.Start < previous || ch.End < ch.Start || ch.End > len(base) || (i > 0 && ch.Start == changes[i-1].Start) {
			return fail("editorial-invalid", "changes must be sorted, nonoverlapping base byte ranges", Range{ch.Start, ch.End})
		}
		if previous < ch.Start {
			segments = append(segments, survivingSegment{Range{previous, ch.Start}, previous + offset})
		}
		candidate.Write(base[previous:ch.Start])
		newStart := candidate.Len()
		candidate.WriteString(ch.New)
		candidateChanges = append(candidateChanges, Range{newStart, candidate.Len()})
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
		regionIndex = advanceChangeRegion(p.Regions, regionIndex, ch)
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
	after, err := Compile(candidate.Bytes())
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
		if !h.Protected {
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
	markers := visibleMarkers(string(base), p.CodeRanges)
	candidateSource := candidate.String()
	candidateMarkers := visibleMarkers(candidateSource, after.CodeRanges)
	stats.MarkerVisits += len(markers) + len(candidateMarkers)
	markerAt := map[int]Marker{}
	for _, m := range candidateMarkers {
		markerAt[m.Range.Start] = m
	}
	readyIdentity := map[int]string{}
	markerMapper := positionMapper{changes: changes}
	for _, m := range markers {
		mapped, ok := markerMapper.mapStart(m.Range.Start)
		if !ok {
			if m.Pending {
				return fail("editorial-pending", "pending agent markers must remain unchanged", m.Range)
			}
			continue
		}
		raw := string(base[m.Range.Start:m.Range.End])
		if m.Ready {
			readyIdentity[mapped] = raw
		}
		if m.Pending {
			got, exists := markerAt[mapped]
			if !exists || got.Range.End-got.Range.Start != len(raw) || candidateSource[got.Range.Start:got.Range.End] != raw {
				return fail("editorial-pending", "pending marker identity and turn extent must remain unchanged", m.Range)
			}
		}
	}
	for _, m := range candidateMarkers {
		if m.Ready {
			raw := candidateSource[m.Range.Start:m.Range.End]
			if old, ok := readyIdentity[m.Range.Start]; !ok || old != raw {
				return fail("editorial-pending", "agents may not create or expose human-ready marker turns", m.Range)
			}
		}
	}

	blocks, chatItems := indexBlocks(base, p, markers)
	safeAnswers, tailWork := safeAnswerAppends(base, p, after, changes, candidateChanges)
	stats.TailJoinVisits = tailWork
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
	candidateItems := selectChatItems(candidate.Bytes(), after)
	if reason, where, work := preserveChatItems(chatItems, candidateItems, changes, refs); reason != "" {
		stats.ItemJoinVisits = work
		return fail("editorial-chat", reason, where)
	} else {
		stats.ItemJoinVisits = work
	}

	pendingIndex, refIndex, chatIndex := 0, 0, 0
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
		if chatIndex < len(chatItems) {
			item := chatItems[chatIndex]
			stats.ChatComparisons++
			if overlapsChange(ch, item.Range) || ch.Start == item.AppendAt || ch.Start == item.Range.Start {
				if !inReference {
					if ch.Start != ch.End || ch.Start != item.AppendAt || item.Linked || !validAnswerAppend(ch.New) || !safeAnswers[i] {
						return fail("editorial-chat", "preserve each chat ask; only append its answer link", r)
					}
				}
			}
		}

	}
	return result, stats, nil
}

// An insertion immediately before a heading belongs to the preceding region;
// a replacement starting on the heading belongs to that heading's region.
func advanceChangeRegion(regions []Region, index int, ch Change) int {
	for index+1 < len(regions) {
		end := regions[index].Range.End
		if end > ch.Start || (end == ch.Start && ch.Start == ch.End) {
			break
		}
		index++
	}
	return index
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

// mapEnd excludes an insertion exactly after the protected interval.
func (m *positionMapper) mapEnd(pos int) (int, bool) {
	for m.index < len(m.changes) {
		ch := m.changes[m.index]
		if ch.End > pos || (ch.End == pos && ch.Start == ch.End) {
			break
		}
		m.delta += len(ch.New) - (ch.End - ch.Start)
		m.index++
	}
	if m.index < len(m.changes) {
		ch := m.changes[m.index]
		if ch.Start < pos && pos < ch.End {
			return 0, false
		}
	}
	return pos + m.delta, true
}
func sameEffective(a, b Effective) bool {
	return a.Collaboration == b.Collaboration && a.Legacy == b.Legacy
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

// Pending historical markers escape their link punctuation. After decoding,
// their destinations obey the same shared rule as ordinary chat answers.
var existingAnswerLine = regexp.MustCompile(`(?m)(?:^|[ \t])→ (?:\[answer\]\(([^()]*)\)|(done: [^\r\n]+))[ \t]*\r?$`)

func hasExistingAnswer(text string) bool {
	for _, match := range existingAnswerLine.FindAllStringSubmatch(text, -1) {
		if match[2] != "" || sidedoc.ValidAnswerTarget(match[1]) {
			return true
		}
	}
	return false
}

func validAnswerAppend(text string) bool {
	const prefix = " → [answer]("
	if !strings.HasPrefix(text, prefix) || !strings.HasSuffix(text, ")") {
		return false
	}
	return sidedoc.ValidAnswerTarget(text[len(prefix) : len(text)-1])
}

type chatItem struct {
	Range    Range
	AppendAt int
	Linked   bool
}

func selectChatItems(base []byte, p *DocumentPolicy) []chatItem {
	return append([]chatItem(nil), p.items...)
}

// compileChatItems visits disjoint governing ranges, with one global minimum
// indent per original section. No overlapping parent range is reparsed.
func compileChatItems(base []byte, p *DocumentPolicy) []chatItem {
	type fact struct {
		item          chatItem
		owner, indent int
	}
	var facts []fact
	minimum := make(map[int]int)
	for i := 0; i < len(p.Regions); {
		first := p.Regions[i]
		end := first.Range.End
		j := i + 1
		for j < len(p.Regions) && p.Regions[j].chat == first.chat {
			end = p.Regions[j].Range.End
			j++
		}
		i = j
		if first.chat < 0 {
			continue
		}
		begin := max(first.Range.Start, p.parsed.body)
		if begin >= end {
			continue
		}
		chunk := base[begin:end]
		p.Stats.ItemSourceBytes += len(chunk)
		lines := sourceLines(chunk)
		for _, shared := range sidedoc.ChatItems(string(chunk), first.Effective.Legacy) {
			p.Stats.ItemJoins++
			if shared.Start < 0 || shared.Start >= len(lines) || len(shared.Lines) == 0 {
				continue
			}
			last := shared.Start + len(shared.Lines) - 1
			if last >= len(lines) {
				continue
			}
			r := Range{begin + lines[shared.Start].Start, begin + lines[last].End}
			appendAt := r.End
			for appendAt > r.Start && (base[appendAt-1] == '\n' || base[appendAt-1] == '\r') {
				appendAt--
			}
			indent := sidedoc.IndentOf(shared.Lines[0])
			if old, ok := minimum[first.chat]; !ok || indent < old {
				minimum[first.chat] = indent
			}
			facts = append(facts, fact{chatItem{Range: r, AppendAt: appendAt, Linked: shared.Answered}, first.chat, indent})
		}
	}
	var out []chatItem
	for _, f := range facts {
		p.Stats.ItemJoins++
		if f.indent == minimum[f.owner] {
			out = append(out, f.item)
		}
	}
	return out
}

func indexBlocks(base []byte, p *DocumentPolicy, markers []Marker) ([]Range, []chatItem) {
	items := selectChatItems(base, p)
	// Direct answer status already comes from shared ChatItems. Only historical
	// escaped pending-marker links need the additional decoded observation below.
	lines := sourceLines(base)
	lineEnd := make([]int, len(lines))
	for i, r := range lines {
		end := r.End
		for end > r.Start && (base[end-1] == '\n' || base[end-1] == '\r' || base[end-1] == ' ' || base[end-1] == '\t') {
			end--
		}
		lineEnd[i] = end
	}
	itemCursor, lineCursor := 0, 0
	for _, m := range markers {
		if !m.Pending || !m.LastTurnAgent {
			continue
		}
		for itemCursor < len(items) && items[itemCursor].Range.End <= m.Range.Start {
			itemCursor++
		}
		if itemCursor == len(items) {
			break
		}
		if m.Range.Start < items[itemCursor].Range.Start || m.Range.End > items[itemCursor].Range.End {
			continue
		}
		for lineCursor < len(lines) && lines[lineCursor].End < m.Range.End {
			lineCursor++
		}
		if lineCursor == len(lines) || m.Range.End != lineEnd[lineCursor] {
			continue
		}
		decoded := UnescapeMarkerText(string(base[m.LastTurn.Start:m.LastTurn.End]))
		if hasExistingAnswer(decoded) {
			items[itemCursor].Linked = true
		}
	}
	return p.Blocks, items
}

func visibleMarkers(source string, code []Range) []Marker {
	markers := ScanMarkers(source)
	out := markers[:0]
	cursor := 0
	for _, m := range markers {
		for cursor < len(code) && code[cursor].End <= m.Range.Start {
			cursor++
		}
		if cursor < len(code) && code[cursor].Start <= m.Range.Start {
			continue
		}
		out = append(out, m)
	}
	return out
}

// safeAnswerAppends joins precompiled facts in linear passes. Candidate facts
// verify visibility and preserved block kind; authority remains in base facts.
func safeAnswerAppends(source []byte, base, after *DocumentPolicy, changes []Change, insertions []Range) ([]bool, int) {
	work := 0
	type projectedTail struct {
		block RenderableBlock
		start int
	}
	tails := map[int]projectedTail{}
	mapper := positionMapper{changes: changes}
	for _, block := range base.Renderable {
		work++
		mapped, ok := mapper.mapStart(block.Range.Start)
		end := block.Range.End
		for end > block.Range.Start && (source[end-1] == '\r' || source[end-1] == '\n') {
			work++
			end--
		}
		suffixSafe := block.ContentEnd <= end
		for pos := block.ContentEnd; pos < end; pos++ {
			work++
			if source[pos] != ' ' && source[pos] != '\t' {
				suffixSafe = false
				break
			}
		}
		if ok && block.SafeTail && suffixSafe {
			tails[end] = projectedTail{block, mapped}
		}
	}
	afterBlocks := map[int]RenderableBlock{}
	for _, block := range after.Renderable {
		work++
		afterBlocks[block.Range.Start] = block
	}
	links := map[int]AnswerLink{}
	for _, link := range after.AnswerLinks {
		work++
		links[link.Label.Start] = link
	}
	safe := make([]bool, len(changes))
	for i, ch := range changes {
		work++
		if ch.Start != ch.End || !validAnswerAppend(ch.New) {
			continue
		}
		tail, ok := tails[ch.Start]
		if !ok {
			continue
		}
		candidateBlock, ok := afterBlocks[tail.start]
		if !ok || candidateBlock.Kind != tail.block.Kind || candidateBlock.ContentEnd != insertions[i].End {
			continue
		}
		label := insertions[i].Start + len(" → [")
		link, ok := links[label]
		if !ok || link.Label.End != label+len("answer") {
			continue
		}
		destination := ch.New[len(" → [answer](") : len(ch.New)-1]
		if link.Destination != destination {
			continue
		}
		safe[i] = true
	}
	return safe, work
}

// preserveChatItems checks the selected ask identity/extent globally, so edits
// just outside an item cannot hide it in code, add continuation, or demote it.
// A batch made solely of actual ready-referral edits retains the existing
// operator authority to reshape the item those referrals address.
func preserveChatItems(base, after []chatItem, changes []Change, refs []Range) (string, Range, int) {
	work := 0
	onlyReferrals := len(changes) > 0
	authorized := map[int]bool{}
	ri, ii := 0, 0
	for _, ch := range changes {
		work++
		for ri < len(refs) && refs[ri].End < ch.Start {
			ri++
			work++
		}
		inRef := ri < len(refs) && containsChange(refs[ri], ch)
		if !inRef {
			onlyReferrals = false
		}
		for ii < len(base) && base[ii].Range.End <= ch.Start {
			ii++
			work++
		}
		if inRef && ii < len(base) && overlapsChange(ch, base[ii].Range) {
			authorized[ii] = true
		}
	}
	candidate := map[int]chatItem{}
	for _, item := range after {
		work++
		candidate[item.Range.Start] = item
	}
	starts, ends := positionMapper{changes: changes}, positionMapper{changes: changes}
	for i, item := range base {
		work++
		start, startOK := starts.mapStart(item.Range.Start)
		end, endOK := ends.mapEnd(item.Range.End)
		got, found := candidate[start]
		if startOK && endOK && found && got.Range.End == end {
			continue
		}
		if onlyReferrals && authorized[i] {
			continue
		}
		return "preserve the complete rendered chat item and its source membership", item.Range, work
	}
	return "", Range{}, work
}
