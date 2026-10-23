package editorial

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// listSourceParser delegates all Markdown decisions to Goldmark. Its only
// addition is retaining source membership observed at successful list-container
// transitions; child Lines intentionally omit syntax such as fence delimiters.
// Each instance and node map belongs to one Compile call.
type listSourceParser struct {
	parser.BlockParser
	lines      []Range
	contentEnd []int
	extents    map[ast.Node]Range
	events     int
	offset     int
}

func (p *listSourceParser) Open(parent ast.Node, reader text.Reader, context parser.Context) (ast.Node, parser.State) {
	line, position := reader.Position()
	node, state := p.BlockParser.Open(parent, reader, context)
	if node != nil {
		p.record(node, line, position.Start)
	}
	return node, state
}
func (p *listSourceParser) Continue(node ast.Node, reader text.Reader, context parser.Context) parser.State {
	line, position := reader.Position()
	state := p.BlockParser.Continue(node, reader, context)
	if state&parser.Continue != 0 {
		p.record(node, line, position.Start)
	}
	return state
}
func (p *listSourceParser) record(node ast.Node, line int, start int) {
	p.events++
	if line < 0 || line >= len(p.lines) || start+p.offset >= p.contentEnd[line] {
		return
	}
	r := p.lines[line]
	if old, ok := p.extents[node]; ok {
		r.Start = min(r.Start, old.Start)
		r.End = max(r.End, old.End)
	}
	p.extents[node] = r
}

// parseSourceMarkdown uses Goldmark's complete default configuration, replacing
// only the list-item parser with a transparent source-membership observer.
// lines are original-document coordinates for the reader's body source lines.
func parseSourceMarkdown(body, source []byte, lines []Range) (ast.Node, map[ast.Node]Range, int) {
	observer := &listSourceParser{BlockParser: parser.NewListItemParser(), lines: lines, extents: map[ast.Node]Range{}, contentEnd: make([]int, len(lines))}
	if len(lines) > 0 {
		observer.offset = lines[0].Start
	}
	for i, r := range lines {
		end := r.End
		for end > r.Start {
			b := source[end-1]
			if b != ' ' && b != '\t' && b != '\r' && b != '\n' {
				break
			}
			end--
		}
		observer.contentEnd[i] = end
	}
	blocks := parser.DefaultBlockParsers()
	for i := range blocks {
		if blocks[i].Value == parser.NewListItemParser() {
			blocks[i].Value = observer
		}
	}
	markdown := parser.NewParser(parser.WithBlockParsers(blocks...), parser.WithInlineParsers(parser.DefaultInlineParsers()...), parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...))
	tree := markdown.Parse(text.NewReader(body))
	return tree, observer.extents, observer.events
}
