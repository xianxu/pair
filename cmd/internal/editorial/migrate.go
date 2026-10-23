package editorial

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FillCanonical adds missing effective components without rewriting any existing setting.
// A refusal always returns nil candidate bytes.
func FillCanonical(src []byte, c *Catalog) ([]byte, error) {
	d, e := Compile(src, c)
	if e != nil {
		return nil, e
	}
	if d.multiple {
		return nil, fmt.Errorf("multiple legacy processing modes: choose one canonical processing-mode before fill")
	}
	p := d.parsed
	nl := p.newline
	var add strings.Builder
	for _, kv := range [][2]string{{"collaboration-mode", d.Defaults.Collaboration}, {"presentation-mode", d.Defaults.Presentation}, {"processing-mode", d.Defaults.Processing}} {
		if _, ok := p.fields[kv[0]]; !ok {
			add.WriteString(kv[0] + ": " + kv[1] + nl)
		}
	}
	if _, ok := p.fields["processing-instruction"]; !ok && d.Defaults.Instruction != "" {
		var n yaml.Node
		n.Kind = yaml.ScalarNode
		n.Tag = "!!str"
		n.Value = d.Defaults.Instruction
		v, err := yaml.Marshal(&n)
		if err != nil {
			return nil, err
		}
		add.WriteString("processing-instruction: " + strings.ReplaceAll(strings.TrimRight(string(v), "\n"), "\n", nl) + nl)
	}
	if add.Len() == 0 {
		return bytes.Clone(src), nil
	}
	if p.front.End == 0 {
		return []byte("---" + nl + add.String() + "---" + nl + nl + string(src)), nil
	}
	closeStart := p.closeStart
	if p.indent > 0 {
		text := strings.ReplaceAll(add.String(), nl, nl+strings.Repeat(" ", p.indent))
		text = strings.TrimSuffix(text, strings.Repeat(" ", p.indent))
		add.Reset()
		add.WriteString(strings.Repeat(" ", p.indent) + text)
	}
	out := make([]byte, 0, len(src)+add.Len())
	out = append(out, src[:closeStart]...)
	out = append(out, add.String()...)
	out = append(out, src[closeStart:]...)
	return out, nil
}

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

// PlanMigration is the sole explicit legacy-map transform. It preserves routing
// at every heading occurrence and validates the resulting canonical document.
func PlanMigration(src []byte, c *Catalog) (*Migration, error) {
	d, err := Compile(src, c)
	if err != nil {
		return nil, err
	}
	if d.multiple {
		return nil, fmt.Errorf("multiple legacy processing modes prevent migration")
	}
	candidate := bytes.Clone(src)
	if d.legacyField != nil {
		names := map[string]bool{}
		for _, h := range d.Headings {
			names[NormalizeHeading(h.Name)] = true
		}
		for name := range d.legacy {
			if !names[name] {
				return nil, fmt.Errorf("legacy map name %q matches no heading", name)
			}
		}
		reps := []sourceReplacement{{d.legacyField.Range, []byte(strings.Repeat(" ", d.parsed.indent) + "collaboration-mode: " + d.Defaults.Collaboration + d.parsed.newline)}}
		for headingIndex, h := range d.Headings {
			mode, ok := d.legacy[NormalizeHeading(h.Name)]
			if !ok {
				continue
			}
			if d.Regions[headingIndex+1].Effective.Collaboration != mode {
				return nil, fmt.Errorf("heading %q: map conflicts with applicable heading collaboration-mode", h.Name)
			}
			if h.Directive != nil {
				if v, ok := h.Overrides["collaboration-mode"]; ok {
					if v != mode {
						return nil, fmt.Errorf("heading %q: map conflicts with heading collaboration-mode", h.Name)
					}
					continue
				}
				raw := h.Directive.Raw
				line := strings.TrimRight(string(raw), "\r\n")
				line = strings.TrimSuffix(line, " -->") + "; collaboration-mode: " + mode + " -->"
				suffix := raw[len(strings.TrimRight(string(raw), "\r\n")):]
				reps = append(reps, sourceReplacement{h.Directive.Range, append([]byte(line), suffix...)})
			} else {
				prefix := ""
				if h.Range.End > 0 && src[h.Range.End-1] != '\n' {
					prefix = d.parsed.newline
				}
				reps = append(reps, sourceReplacement{Range{h.Range.End, h.Range.End}, []byte(prefix + "<!-- collaboration-mode: " + mode + " -->" + d.parsed.newline)})
			}
		}
		sort.Slice(reps, func(i, j int) bool { return reps[i].r.Start < reps[j].r.Start })
		var out bytes.Buffer
		pos := 0
		for _, r := range reps {
			out.Write(src[pos:r.r.Start])
			out.Write(r.after)
			pos = r.r.End
		}
		out.Write(src[pos:])
		candidate = out.Bytes()
	}
	candidate, err = FillCanonical(candidate, c)
	if err != nil {
		return nil, err
	}
	after, err := Compile(candidate, c)
	if err != nil {
		return nil, err
	}
	// An independent stack projection validates routing rather than trusting the
	// compiler's effective-region comparison alone.
	beforeRoutes := routingProjection(d)
	afterRoutes := routingProjection(after)
	if len(beforeRoutes) != len(afterRoutes) {
		return nil, fmt.Errorf("migration changed heading structure")
	}
	for i := range beforeRoutes {
		if beforeRoutes[i] != afterRoutes[i] {
			return nil, fmt.Errorf("migration changed routing at heading %d", i)
		}
	}
	return &Migration{BeforeHash: d.SourceHash, After: candidate, Diagnostics: d.Diagnostics}, nil
}
func routingProjection(d *DocumentPolicy) []string {
	routes := []string{d.Defaults.Collaboration}
	type ancestor struct {
		level   int
		mode    string
		comment string
	}
	stack := []ancestor{}
	for _, h := range d.Headings {
		for len(stack) > 0 && stack[len(stack)-1].level >= h.Level {
			stack = stack[:len(stack)-1]
		}
		mode := d.Defaults.Collaboration
		comment := ""
		if len(stack) > 0 {
			comment = stack[len(stack)-1].comment
			mode = stack[len(stack)-1].mode
		}
		if mapped, ok := d.legacy[NormalizeHeading(h.Name)]; ok {
			mode = mapped
		}
		if override, ok := h.Overrides["collaboration-mode"]; ok {
			comment = override
		}
		if comment != "" {
			mode = comment
		}
		routes = append(routes, mode)
		stack = append(stack, ancestor{h.Level, mode, comment})
	}
	return routes
}
