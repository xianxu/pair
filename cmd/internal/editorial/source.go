package editorial

import (
	"bytes"
	"fmt"
	"github.com/xianxu/ariadne/pkg/sidedoc"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Range uses half-open byte offsets in the original document.
type Range struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Diagnostic locates a nonfatal compatibility warning in the base snapshot.
type Diagnostic struct {
	Range   Range  `json:"range"`
	Message string `json:"message"`
}

// OwnedSource retains exact settings bytes and their heading association (-1 for document settings).
type OwnedSource struct {
	Range   Range  `json:"range"`
	Key     string `json:"key"`
	Raw     []byte `json:"raw"`
	Heading int    `json:"heading"`
}
type sourceField struct {
	OwnedSource
	value *yaml.Node
}
type parsedSource struct {
	fields     map[string]sourceField
	owned      []OwnedSource
	front      Range
	body       int
	newline    string
	closeStart int
	indent     int
}

func ownedKey(k string) bool {
	if k == "collaboration-mode" {
		return true
	}
	for _, retired := range sidedoc.RetiredKeys {
		if k == retired {
			return true
		}
	}
	return false
}
func parseOwnedSource(src []byte) (parsedSource, error) {
	p := parsedSource{fields: map[string]sourceField{}, newline: "\n"}
	if bytes.Contains(src, []byte("\r\n")) {
		p.newline = "\r\n"
	}
	lines := sourceLines(src)
	if len(lines) == 0 || strings.TrimRight(string(src[lines[0].Start:lines[0].End]), "\r\n") != "---" {
		return p, nil
	}
	close := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(string(src[lines[i].Start:lines[i].End]), "\r\n") == "---" {
			close = i
			break
		}
	}
	if close < 0 {
		return p, nil
	}
	p.closeStart = lines[close].Start
	if bytes.HasSuffix(src[:lines[0].End], []byte("\r\n")) {
		p.newline = "\r\n"
	} else {
		p.newline = "\n"
	}
	p.front = Range{0, lines[close].End}
	p.body = p.front.End
	baseIndent := -1
	for i := 1; i < close; i++ {
		line := strings.TrimRight(string(src[lines[i].Start:lines[i].End]), "\r\n")
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(trimmed)
		if baseIndent < 0 || indent < baseIndent {
			baseIndent = indent
		}
	}
	if baseIndent < 0 {
		baseIndent = 0
	}
	p.indent = baseIndent
	for i := 1; i < close; i++ {
		r := lines[i]
		line := strings.TrimRight(string(src[r.Start:r.End]), "\r\n")
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent != baseIndent {
			continue
		}
		line = line[indent:]
		if len(line) == 0 || line[0] == '\t' {
			continue
		}
		keyRaw, _, ok := strings.Cut(line, ":")
		key := keyRaw
		if ok && !ownedKey(key) {
			var keyNode yaml.Node
			if yaml.Unmarshal([]byte(keyRaw), &keyNode) == nil && len(keyNode.Content) > 0 && keyNode.Content[0].Kind == yaml.ScalarNode {
				key = keyNode.Content[0].Value
			}
		}
		if !ok || !ownedKey(key) {
			continue
		}

		j := i + 1
		for j < close {
			next := strings.TrimRight(string(src[lines[j].Start:lines[j].End]), "\r\n")
			nextIndent := len(next) - len(strings.TrimLeft(next, " "))
			if strings.TrimSpace(next) != "" && nextIndent <= baseIndent && next[0] != '\t' {
				break
			}
			j++
		}
		// Blank separators belong to the surrounding document, not the setting.
		end := j
		for end > i+1 && len(bytes.TrimSpace(src[lines[end-1].Start:lines[end-1].End])) == 0 {
			end--
		}
		r.End = lines[end-1].End
		raw := src[r.Start:r.End]
		valueRaw := raw[indent+len(keyRaw)+1:]
		var node yaml.Node
		err := yaml.Unmarshal(raw, &node)
		value := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
		if err == nil && len(node.Content) > 0 && node.Content[0].Kind == yaml.MappingNode && len(node.Content[0].Content) == 2 {
			value = node.Content[0].Content[1]
		} else if err == nil {
			err = fmt.Errorf("owned setting must be one mapping entry")
		}
		if err == nil && (value.Style == yaml.LiteralStyle || value.Style == yaml.FoldedStyle) && end < j {
			r.End = lines[j-1].End
			raw = src[r.Start:r.End]
			err = yaml.Unmarshal(raw, &node)
			if err == nil {
				value = node.Content[0].Content[1]
			}
		}
		if key == "process-instruction" && (err != nil || value.Kind != yaml.ScalarNode) {
			value = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.TrimSpace(string(valueRaw))}
			err = nil
		}

		f := sourceField{OwnedSource: OwnedSource{Range: r, Key: key, Raw: bytes.Clone(raw), Heading: -1}, value: value}
		p.fields[key] = f
		p.owned = append(p.owned, f.OwnedSource)
	}
	return p, nil
}
func sourceLines(src []byte) []Range {
	var out []Range
	for pos := 0; pos < len(src); {
		end := bytes.IndexByte(src[pos:], '\n')
		if end < 0 {
			end = len(src)
		} else {
			end += pos + 1
		}
		out = append(out, Range{pos, end})
		pos = end
	}
	return out
}
func scalar(f sourceField) (string, error) {
	if f.value.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("%s at byte %d must be a scalar", f.Key, f.Range.Start)
	}
	return f.value.Value, nil
}
