package editorial

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"go.yaml.in/yaml/v3"
)

// Effective contains the three independently resolved axes and processing brief metadata.
type Effective struct {
	Collaboration string `json:"collaboration"`
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
}

// Region is a contiguous source interval with one effective editorial policy.
type Region struct {
	Range     Range     `json:"range"`
	Heading   int       `json:"heading"`
	Effective Effective `json:"effective"`
}

// CompileStats counts this package's source/heading/map work, excluding Goldmark internals.
type CompileStats struct {
	LineAdvances      int
	HeadingVisits     int
	LegacyAssignments int
	ListSourceEvents  int
}

// DocumentPolicy is an immutable compilation of one document snapshot.
type DocumentPolicy struct {
	Stats       CompileStats  `json:"-"`
	Defaults    Effective     `json:"defaults"`
	Regions     []Region      `json:"regions"`
	Headings    []Heading     `json:"headings"`
	Owned       []OwnedSource `json:"owned"`
	Diagnostics []Diagnostic  `json:"diagnostics,omitempty"`
	Blocks      []Range       `json:"-"`
	ListItems   []Range       `json:"-"`
	CodeRanges  []Range       `json:"-"`
	SourceHash  string        `json:"sourceHash"`
	legacy      map[string]string
	legacyField *sourceField
	parsed      parsedSource
	multiple    bool
}

func defaultEffective(c *Catalog) Effective {
	m, _ := c.ResolveProcessing("free-form")
	return Effective{Collaboration: "discussion", Presentation: "highlight", Processing: m.Name, Modes: []Mode{m}, Scope: m.Scope, Frontier: m.Frontier}
}
func setProcessing(e *Effective, modes []Mode) {
	if len(modes) == 0 {
		return
	}
	e.Modes = append([]Mode(nil), modes...)
	e.Processing = modes[0].Name
	e.Scope = modes[0].Scope
	e.Frontier = modes[0].Frontier
}

// NormalizeHeading applies the pinned legacy map name normalization.
func NormalizeHeading(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	if strings.HasSuffix(s, ")") {
		if i := strings.LastIndex(s, " ("); i >= 0 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
}

// Compile resolves a snapshot without changing source bytes.
func Compile(src []byte, c *Catalog) (*DocumentPolicy, error) {
	if c == nil {
		return nil, fmt.Errorf("editorial catalog is required")
	}
	p, err := parseOwnedSource(src)
	if err != nil {
		return nil, err
	}
	d := &DocumentPolicy{Defaults: defaultEffective(c), Owned: p.owned, parsed: p, legacy: map[string]string{}}
	hash := sha256.Sum256(src)
	d.SourceHash = hex.EncodeToString(hash[:])
	warn := func(f sourceField, msg string) { d.Diagnostics = append(d.Diagnostics, Diagnostic{f.Range, msg}) }
	if f, ok := p.fields["collaboration-mode"]; ok {
		if f.value.Kind == yaml.MappingNode {
			d.legacyField = &f
			warn(f, "legacy collaboration-mode map is deprecated")
			if err := readLegacyMap(f.value, d); err != nil {
				return nil, err
			}
		} else {
			v, e := scalar(f)
			if e != nil {
				return nil, e
			}
			v, e = ResolveCollaboration(v)
			if e != nil {
				return nil, e
			}
			d.Defaults.Collaboration = v
		}
	}
	if f, ok := p.fields["presentation"]; ok {
		warn(f, "presentation is deprecated; use presentation-mode")
	}
	f, ok := p.fields["presentation-mode"]
	if !ok {
		f, ok = p.fields["presentation"]
	}
	if ok {
		v, e := scalar(f)
		if e != nil {
			return nil, e
		}
		if v != "" {
			resolved, e := ResolvePresentation(v)
			if e != nil {
				warn(f, e.Error()+"; valid choices: highlight, direct, propose; using highlight")
			} else {
				d.Defaults.Presentation = resolved
			}
		}
	}
	var legacyModes []Mode
	var legacyInstruction string
	if f, ok := p.fields["process-instruction"]; ok {
		warn(f, "process-instruction is deprecated")
		_, hasMode := p.fields["processing-mode"]
		_, hasInstruction := p.fields["processing-instruction"]
		v, e := scalar(f)
		if e != nil {
			if !hasMode || !hasInstruction {
				return nil, e
			}
		} else {
			selectors, instruction, lexErr := splitLegacyProcessing(v)
			d.multiple = len(selectors) > 1
			if !hasInstruction {
				legacyInstruction = instruction
			}
			if !hasMode {
				if lexErr != nil {
					return nil, lexErr
				}
				for _, selector := range selectors {
					m, e := c.ResolveProcessing(selector)
					if e != nil {
						return nil, e
					}
					legacyModes = append(legacyModes, m)
				}
			}
		}
	}
	if f, ok := p.fields["processing-mode"]; ok {
		v, e := scalar(f)
		if e != nil {
			return nil, e
		}
		m, e := c.ResolveProcessing(v)
		if e != nil {
			return nil, e
		}
		setProcessing(&d.Defaults, []Mode{m})
	} else if len(legacyModes) > 0 {
		setProcessing(&d.Defaults, legacyModes)
	}
	if f, ok := p.fields["processing-instruction"]; ok {
		v, e := scalar(f)
		if e != nil {
			return nil, e
		}
		d.Defaults.Instruction = v
	} else {
		d.Defaults.Instruction = legacyInstruction
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
	var activeItem ast.Node
	itemRange := Range{-1, -1}
	ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			if n == activeItem {
				if itemRange.Start >= 0 {
					rawItems = append(rawItems, itemRange)
				}
				activeItem = nil
			}
			return ast.WalkContinue, nil
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
		h, ok := n.(*ast.Heading)
		if !ok || h.Lines().Len() == 0 {
			return ast.WalkContinue, nil
		}
		seg := h.Lines().At(0)
		pos := p.body + seg.Start
		for lineCursor < len(lines) && lines[lineCursor].End <= pos {
			lineCursor++
			d.Stats.LineAdvances++
		}
		line := lineCursor
		if line == len(lines) {
			return ast.WalkContinue, nil
		}
		raw := bytes.TrimLeft(src[lines[line].Start:lines[line].End], " \t")
		if len(raw) == 0 || raw[0] != '#' {
			return ast.WalkContinue, nil
		}
		name := strings.TrimSpace(string(raw))
		name = strings.TrimLeft(name, "#")
		name = strings.TrimSpace(name)
		if end := len(name) - len(strings.TrimRight(name, "#")); end > 0 && len(name) > end && (name[len(name)-end-1] == ' ' || name[len(name)-end-1] == '\t') {
			name = strings.TrimSpace(name[:len(name)-end])
		}
		d.Headings = append(d.Headings, Heading{Range: lines[line], Level: h.Level, Parent: -1, Name: name})
		return ast.WalkContinue, nil
	})
	d.Blocks = expandToLines(rawBlocks, lines)
	d.ListItems = expandToLines(rawItems, lines)
	// Parent stack and directive association follow source order.
	stack := []int{}
	codeIndex := 0
	commentCollaboration := make([]string, len(d.Headings))
	matched := map[string]bool{}
	d.Regions = append(d.Regions, Region{Range: Range{0, len(src)}, Heading: -1, Effective: d.Defaults})
	for i := range d.Headings {
		h := &d.Headings[i]
		d.Stats.HeadingVisits++
		for len(stack) > 0 && d.Headings[stack[len(stack)-1]].Level >= h.Level {
			stack = stack[:len(stack)-1]
		}
		e := d.Defaults
		if len(stack) > 0 {
			h.Parent = stack[len(stack)-1]
			e = d.Regions[h.Parent+1].Effective
		}
		stack = append(stack, i)
		if h.Parent >= 0 {
			commentCollaboration[i] = commentCollaboration[h.Parent]
		}
		if v, ok := d.legacy[NormalizeHeading(h.Name)]; ok {
			e.Collaboration = v
			matched[NormalizeHeading(h.Name)] = true
		}
		if commentCollaboration[i] != "" {
			e.Collaboration = commentCollaboration[i]
		}
		lineEnd := h.Range.End
		for codeIndex < len(d.CodeRanges) && d.CodeRanges[codeIndex].End <= lineEnd {
			codeIndex++
		}
		inCode := codeIndex < len(d.CodeRanges) && d.CodeRanges[codeIndex].Start <= lineEnd
		if lineEnd < len(src) && !inCode {
			end := bytes.IndexByte(src[lineEnd:], '\n')
			if end < 0 {
				end = len(src)
			} else {
				end += lineEnd + 1
			}
			r := Range{lineEnd, end}
			raw := strings.TrimRight(string(src[r.Start:r.End]), "\r\n")
			over, recognized, e2 := parseDirective(raw, c)
			if e2 != nil {
				return nil, fmt.Errorf("byte %d heading %q: %w", r.Start, h.Name, e2)
			}
			if recognized {
				o := OwnedSource{Range: r, Key: "heading-directive", Raw: bytes.Clone(src[r.Start:r.End]), Heading: i}
				h.Directive = &o
				h.Overrides = over
				d.Owned = append(d.Owned, o)
				if v := over["collaboration-mode"]; v != "" {
					e.Collaboration = v
					commentCollaboration[i] = v
				}
				if v := over["presentation-mode"]; v != "" {
					e.Presentation = v
				}
				if v := over["processing-mode"]; v != "" {
					m, _ := c.ResolveProcessing(v)
					setProcessing(&e, []Mode{m})
				}
				if end < len(src) {
					nextEnd := bytes.IndexByte(src[end:], '\n')
					if nextEnd < 0 {
						nextEnd = len(src)
					} else {
						nextEnd += end
					}
					_, again, err := parseDirective(strings.TrimRight(string(src[end:nextEnd]), "\r\n"), c)
					if again || err != nil {
						return nil, fmt.Errorf("byte %d: multiple adjacent mode comments", end)
					}
				}
			}
		}
		d.Regions[len(d.Regions)-1].Range.End = h.Range.Start
		d.Regions = append(d.Regions, Region{Range: Range{h.Range.Start, len(src)}, Heading: i, Effective: e})
	}
	for name := range d.legacy {
		if !matched[name] {
			d.Diagnostics = append(d.Diagnostics, Diagnostic{d.legacyField.Range, fmt.Sprintf("legacy collaboration map names %q, which matches no heading", name)})
		}
	}
	return d, nil
}
func splitLegacyProcessing(v string) ([]string, string, error) {
	v = strings.TrimSpace(v)
	var selectors []string
	for strings.HasPrefix(v, "[") {
		end := strings.IndexByte(v, ']')
		if end < 0 {
			return selectors, v, fmt.Errorf("unclosed legacy processing selector")
		}
		selectors = append(selectors, v[1:end])
		v = strings.TrimSpace(v[end+1:])
	}
	return selectors, v, nil
}
func parseLegacyProcessing(v string, c *Catalog) ([]Mode, string, error) {
	selectors, instruction, e := splitLegacyProcessing(v)
	if e != nil {
		return nil, "", e
	}
	var modes []Mode
	for _, selector := range selectors {
		m, e := c.ResolveProcessing(selector)
		if e != nil {
			return nil, "", e
		}
		modes = append(modes, m)
	}
	return modes, instruction, nil
}

func parseDirective(raw string, c *Catalog) (map[string]string, bool, error) {
	recognized := false
	innerStart := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "<!--"))
	for _, part := range strings.Split(innerStart, ";") {
		part = strings.TrimSpace(part)
		for _, key := range []string{"collaboration-mode", "presentation-mode", "processing-mode"} {
			if strings.HasPrefix(part, key) {
				rest := part[len(key):]
				if rest == "" || rest[0] == ':' || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '\r' || rest[0] == '\n' {
					recognized = true
				}
			}
		}
	}

	if !strings.HasPrefix(strings.TrimSpace(raw), "<!--") || !recognized {
		return nil, false, nil
	}
	fail := func(msg string) (map[string]string, bool, error) {
		return nil, true, fmt.Errorf("%s; valid collaboration: discussion, collaboration, chat; presentation: highlight, direct, propose; processing: catalog names or unique prefixes", msg)
	}
	if !strings.HasPrefix(raw, "<!-- ") || !strings.HasSuffix(raw, " -->") {
		return fail("malformed mode declaration")
	}
	inner := raw[5 : len(raw)-4]
	out := map[string]string{}
	for _, part := range strings.Split(inner, "; ") {
		key, value, ok := strings.Cut(part, ": ")
		if !ok || value == "" || strings.TrimSpace(value) != value {
			return fail("malformed mode declaration")
		}
		if _, ok := out[key]; ok {
			return fail("duplicate " + key)
		}
		var err error
		switch key {
		case "collaboration-mode":
			_, err = ResolveCollaboration(value)
		case "presentation-mode":
			_, err = ResolvePresentation(value)
		case "processing-mode":
			var m Mode
			m, err = c.ResolveProcessing(value)
			value = m.Name
		default:
			return fail("unknown declaration key " + key)
		}
		if err != nil {
			return fail(err.Error())
		}
		out[key] = value
	}
	return out, true, nil
}
func readLegacyMap(n *yaml.Node, d *DocumentPolicy) error {
	seen := map[string]bool{}
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if seen[k.Value] {
			return fmt.Errorf("duplicate legacy map key %q", k.Value)
		}
		seen[k.Value] = true
		if k.Value == "default" {
			if v.Kind != yaml.ScalarNode {
				return fmt.Errorf("legacy default must be scalar")
			}
			mode, e := ResolveCollaboration(v.Value)
			if e != nil {
				return e
			}
			d.Defaults.Collaboration = mode
			continue
		}
		mode, e := ResolveCollaboration(k.Value)
		if e != nil {
			return e
		}
		var names []*yaml.Node
		if v.Kind == yaml.SequenceNode {
			names = v.Content
		} else if v.Kind == yaml.ScalarNode {
			names = []*yaml.Node{v}
		} else {
			return fmt.Errorf("legacy names must be scalar or list")
		}
		for _, name := range names {
			if name.Kind != yaml.ScalarNode {
				return fmt.Errorf("legacy name must be scalar")
			}
			d.Stats.LegacyAssignments++
			normalized := NormalizeHeading(name.Value)
			if prev, ok := d.legacy[normalized]; ok && prev != mode {
				return fmt.Errorf("heading %q assigned both %s and %s", name.Value, prev, mode)
			}
			d.legacy[normalized] = mode
		}
	}
	return nil
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
