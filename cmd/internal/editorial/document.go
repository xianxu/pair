package editorial

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/xianxu/ariadne/pkg/sidedoc"

	"github.com/yuin/goldmark/ast"
	"go.yaml.in/yaml/v3"
)

// Effective is the single collaborative policy plus the chat item grammar.
type Effective struct {
	Collaboration string `json:"collaboration"`
	Legacy        bool   `json:"legacy,omitempty"`
	Presentation  string `json:"presentation"`
	Processing    string `json:"processing"`
	Instruction   string `json:"instruction,omitempty"`
	Modes         []Mode `json:"modes"`
	Scope         Scope  `json:"scope"`
	Frontier      bool   `json:"frontier"`
}

// Heading identifies an ATX heading and its nearest parent in source coordinates.
type Heading struct {
	Range     Range             `json:"range"`
	Level     int               `json:"level"`
	Parent    int               `json:"parent"`
	Name      string            `json:"name"`
	Directive *OwnedSource      `json:"directive,omitempty"`
	Overrides map[string]string `json:"overrides,omitempty"`
	Tag       *OwnedSource      `json:"tag,omitempty"`
	Protected bool              `json:"-"`
}

// Region is a contiguous source interval with one effective editorial policy.
type Region struct {
	Range     Range     `json:"range"`
	Heading   int       `json:"heading"`
	Effective Effective `json:"effective"`
	chat      int
}

// CompileStats counts this package's source/heading/map work, excluding Goldmark internals.
type CompileStats struct {
	LineAdvances      int
	HeadingVisits     int
	LegacyAssignments int
	ListSourceEvents  int
	SharedScans       int
	SectionEvents     int
	ItemSourceBytes   int
	ItemJoins         int
}

// RenderableBlock retains source placement and the actual content tail of a
// paragraph, text block or heading. Syntax outside that tail is not appendable.
type RenderableBlock struct {
	Range      Range
	ContentEnd int
	Kind       string
	SafeTail   bool
}

// AnswerLink records a real parsed hyperlink label outside code and image alt text.
type AnswerLink struct {
	Label       Range
	Destination string
}

// DocumentPolicy is an immutable compilation of one document snapshot.
type DocumentPolicy struct {
	Stats       CompileStats      `json:"-"`
	Defaults    Effective         `json:"defaults"`
	Regions     []Region          `json:"regions"`
	Headings    []Heading         `json:"headings"`
	Owned       []OwnedSource     `json:"owned"`
	Diagnostics []Diagnostic      `json:"diagnostics,omitempty"`
	Renderable  []RenderableBlock `json:"-"`
	AnswerLinks []AnswerLink      `json:"-"`
	Blocks      []Range           `json:"-"`
	ListItems   []Range           `json:"-"`
	CodeRanges  []Range           `json:"-"`
	SourceHash  string            `json:"sourceHash"`
	legacy      map[string]string
	legacyField *sourceField
	parsed      parsedSource
	multiple    bool
	scan        sidedoc.ChatScan
	items       []chatItem
}

func defaultEffective(c *Catalog) Effective {
	return Effective{Collaboration: "collaboration", Presentation: "highlight", Scope: WholeDocument}
}

// NormalizeHeading is retained only for the explicit legacy migration.
func NormalizeHeading(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	if strings.HasSuffix(s, ")") {
		if i := strings.LastIndex(s, " ("); i >= 0 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
}

// Compile adapts shared chat facts to one immutable original-byte snapshot.
// Retired values never select processing or presentation and never refuse.
func Compile(src []byte, c *Catalog) (*DocumentPolicy, error) {
	p, err := parseOwnedSource(src)
	if err != nil {
		return nil, err
	}
	scan := sidedoc.ChatSections(string(src))
	d := &DocumentPolicy{Defaults: defaultEffective(c), Owned: p.owned, parsed: p, legacy: map[string]string{}, scan: scan}
	d.Stats.SharedScans = 1
	hash := sha256.Sum256(src)
	d.SourceHash = hex.EncodeToString(hash[:])
	for _, notice := range scan.Notices {
		d.Diagnostics = append(d.Diagnostics, Diagnostic{Message: notice})
	}
	for _, line := range scan.Misplaced {
		d.Diagnostics = append(d.Diagnostics, Diagnostic{Message: fmt.Sprintf("@@chat@@ at line %d is not directly above a heading", line)})
	}
	if f, ok := p.fields["collaboration-mode"]; ok && f.value.Kind == yaml.MappingNode {
		d.legacyField = &f
	}
	// Goldmark owns Markdown structure, including code masking and heading recognition.
	body := src[p.body:]
	lines := sourceLines(src)
	bodyLine := 0
	for bodyLine < len(lines) && lines[bodyLine].Start < p.body {
		bodyLine++
	}
	tree, listExtents, listEvents := parseSourceMarkdown(body, src, lines[bodyLine:])
	d.Stats.ListSourceEvents = listEvents
	lineCursor := 0
	var rawBlocks, rawItems []Range
	var rawRenderable []RenderableBlock
	imageDepth := 0
	var activeItem ast.Node
	itemRange := Range{-1, -1}
	ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			if _, ok := n.(*ast.Image); ok {
				imageDepth--
			}
			if n == activeItem {
				if itemRange.Start >= 0 {
					rawItems = append(rawItems, itemRange)
				}
				activeItem = nil
			}
			return ast.WalkContinue, nil
		}
		if _, ok := n.(*ast.Image); ok {
			imageDepth++
		}
		if link, ok := n.(*ast.Link); ok && imageDepth == 0 {
			if label, ok := link.FirstChild().(*ast.Text); ok && label.NextSibling() == nil && string(label.Segment.Value(body)) == "answer" {
				d.AnswerLinks = append(d.AnswerLinks, AnswerLink{Label: Range{p.body + label.Segment.Start, p.body + label.Segment.Stop}, Destination: string(link.Destination)})
			}
		}
		if _, ok := n.(*ast.ListItem); ok && activeItem == nil {
			activeItem = n
			itemRange = Range{-1, -1}
			if extent, ok := listExtents[n]; ok {
				itemRange = extent
			}
		}
		if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
			first, last := n.Lines().At(0), n.Lines().At(n.Lines().Len()-1)
			r := Range{p.body + first.Start, p.body + last.Stop}
			if activeItem != nil {
				if itemRange.Start < 0 || r.Start < itemRange.Start {
					itemRange.Start = r.Start
				}
				itemRange.End = max(itemRange.End, r.End)
			}
			switch n.(type) {
			case *ast.Paragraph, *ast.Heading, *ast.TextBlock:
				rawBlocks = append(rawBlocks, r)
				contentEnd := r.End
				for contentEnd > r.Start && (src[contentEnd-1] == '\n' || src[contentEnd-1] == '\r') {
					contentEnd--
				}
				kind := n.Kind().String()
				if heading, ok := n.(*ast.Heading); ok {
					kind = fmt.Sprintf("Heading%d", heading.Level)
				}
				safe := true
				if tail, ok := n.LastChild().(*ast.Text); ok && tail.HardLineBreak() && !tail.IsRaw() {
					safe = false
				}
				rawRenderable = append(rawRenderable, RenderableBlock{Range: r, ContentEnd: contentEnd, Kind: kind, SafeTail: safe})
			}
		}

		if span, ok := n.(*ast.CodeSpan); ok {
			for child := span.FirstChild(); child != nil; child = child.NextSibling() {
				if t, ok := child.(*ast.Text); ok {
					d.CodeRanges = append(d.CodeRanges, Range{p.body + t.Segment.Start, p.body + t.Segment.Stop})
				}
			}
			return ast.WalkSkipChildren, nil
		}
		if fence, ok := n.(*ast.FencedCodeBlock); ok && fence.Info != nil {
			d.CodeRanges = append(d.CodeRanges, Range{p.body + fence.Info.Segment.Start, p.body + fence.Info.Segment.Stop})
		}
		switch n.(type) {
		case *ast.CodeBlock, *ast.FencedCodeBlock:
			for j := 0; j < n.Lines().Len(); j++ {
				seg := n.Lines().At(j)
				pos := p.body + seg.Start
				for lineCursor < len(lines) && lines[lineCursor].End <= pos {
					lineCursor++
					d.Stats.LineAdvances++
				}
				if lineCursor < len(lines) {
					d.CodeRanges = append(d.CodeRanges, lines[lineCursor])
				}
			}
			return ast.WalkContinue, nil
		}
		return ast.WalkContinue, nil
	})
	d.Blocks = expandToLines(rawBlocks, lines)
	for i, fact := range rawRenderable {
		fact.Range = d.Blocks[i]
		d.Renderable = append(d.Renderable, fact)
	}
	d.ListItems = expandToLines(rawItems, lines)

	// Heading identity comes from the shared parser, including setext headings.
	headingsByLine := make(map[int]int, len(scan.Headings))
	stack := []int{}
	for _, fact := range scan.Headings {
		if fact.Line < 1 || fact.Line > len(lines) {
			continue
		}
		h := Heading{Range: lines[fact.Line-1], Level: fact.Level, Parent: -1, Name: fact.Name}
		for len(stack) > 0 && d.Headings[stack[len(stack)-1]].Level >= h.Level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			h.Parent = stack[len(stack)-1]
		}
		index := len(d.Headings)
		d.Headings = append(d.Headings, h)
		headingsByLine[fact.Line] = index
		stack = append(stack, index)
		d.Stats.HeadingVisits++
	}
	tags := make(map[int]bool, len(scan.Tags))
	for _, line := range scan.Tags {
		tags[line] = true
	}
	// Source ownership is independent of effective routing. In particular a
	// whole-document legacy alias must not hide the tags inside it.
	for line, index := range headingsByLine {
		h := &d.Headings[index]
		if line > 1 && tags[line-1] {
			r := lines[line-2]
			o := OwnedSource{Range: r, Key: "chat-tag", Raw: bytes.Clone(src[r.Start:r.End]), Heading: index}
			h.Tag = &o
			h.Protected = true
		}
		if line < len(lines) && legacyDirective(string(src[lines[line].Start:lines[line].End])) {
			r := lines[line]
			o := OwnedSource{Range: r, Key: "heading-directive", Raw: bytes.Clone(src[r.Start:r.End]), Heading: index}
			h.Directive = &o
			h.Protected = true
		}
	}
	// Append declarations in source order for the validator's monotone joins.
	for i := range d.Headings {
		h := &d.Headings[i]
		if h.Tag != nil {
			d.Owned = append(d.Owned, *h.Tag)
		}
		if h.Directive != nil {
			d.Owned = append(d.Owned, *h.Directive)
		}
	}
	// Shared sections are laminar and source ordered. Sweep each source line
	// once; do not call the linear InChat/Innermost lookup for every item.
	active := []int{}
	next, heading := 0, -1
	for i, r := range lines {
		line := i + 1
		for len(active) > 0 && scan.Sections[active[len(active)-1]].End < line {
			active = active[:len(active)-1]
			d.Stats.SectionEvents++
		}
		for next < len(scan.Sections) && scan.Sections[next].Heading < line {
			if scan.Sections[next].End >= line {
				active = append(active, next)
			}
			next++
			d.Stats.SectionEvents++
		}
		if n, ok := headingsByLine[line]; ok {
			heading = n
		}
		e := d.Defaults
		chat := -1
		if len(active) > 0 {
			chat = active[len(active)-1]
			e.Collaboration = "chat"
			e.Legacy = scan.Sections[chat].Legacy
		}
		if len(d.Regions) > 0 {
			last := &d.Regions[len(d.Regions)-1]
			if last.Heading == heading && last.chat == chat {
				last.Range.End = r.End
				continue
			}
		}
		d.Regions = append(d.Regions, Region{Range: r, Heading: heading, Effective: e, chat: chat})
	}
	if len(d.Regions) == 0 {
		d.Regions = []Region{{Range: Range{0, len(src)}, Heading: -1, Effective: d.Defaults, chat: -1}}
	}
	d.items = compileChatItems(src, d)
	return d, nil
}

// legacyDirective only locates operator-owned raw bytes. Shared ChatSections
// alone decides what the values mean (including malformed/ignored values).
func legacyDirective(raw string) bool {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "<!--") || !strings.HasSuffix(raw, "-->") {
		return false
	}
	for _, part := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(raw, "<!--"), "-->"), ";") {
		key, _, ok := strings.Cut(part, ":")
		if ok && ownedKey(strings.TrimSpace(key)) {
			return true
		}
	}
	return false
}

// expandToLines widens sorted disjoint block facts using one monotone source walk.
func expandToLines(raw, lines []Range) []Range {
	out := make([]Range, 0, len(raw))
	cursor := 0
	for _, r := range raw {
		for cursor < len(lines) && lines[cursor].End <= r.Start {
			cursor++
		}
		if cursor == len(lines) {
			break
		}
		start := lines[cursor].Start
		for cursor+1 < len(lines) && lines[cursor].End < r.End {
			cursor++
		}
		out = append(out, Range{start, lines[cursor].End})
	}
	return out
}
