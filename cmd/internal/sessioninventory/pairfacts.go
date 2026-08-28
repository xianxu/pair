package sessioninventory

import (
	"bytes"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// PairLogFact is one ordered authored markdown-log entry after normalization.
// Position is its durable byte offset in the scoped Pair log.
// pair:155-concept pure new M2
type PairLogFact struct {
	ScopeKey string
	Tag      string
	Agent    Agent
	Position uint64
	Text     string
}

// PairLedgerFact is the inventory boundary's normalized durable ledger fact.
// pair:155-concept pure new M2
type PairLedgerFact = sessionledger.Record

type PairLogParseResult struct {
	Facts            []PairLogFact
	MalformedOffsets []uint64
}

// NormalizePairText is the canonical identity projection for operator-authored
// Pair text. It removes Pair's sticky comment framing and presentation-only
// whitespace while preserving case and meaningful internal spacing.
// pair:155-concept pure new M2
func NormalizePairText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "===") {
			continue
		}
		out = append(out, line)
	}
	for len(out) > 0 && pairBlankLine(out[0]) {
		out = out[1:]
	}
	for len(out) > 0 && pairBlankLine(out[len(out)-1]) {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func pairBlankLine(line string) bool { return strings.Trim(line, " \t\r\v\f") == "" }

var pairLogSeparator = []byte("\n\n---\n\n")

// ParsePairLog parses only a launch-delimited suffix of the existing markdown
// log. Any malformed framing fails the whole suffix closed so a truncated body
// can never become correlation evidence.
func ParsePairLog(raw []byte, offset uint64) PairLogParseResult {
	if offset > uint64(len(raw)) || (offset != 0 && offset != uint64(len(raw)) && !bytes.HasSuffix(raw[:offset], pairLogSeparator)) {
		return PairLogParseResult{MalformedOffsets: []uint64{offset}}
	}
	if offset == uint64(len(raw)) {
		return PairLogParseResult{}
	}
	var facts []PairLogFact
	cursor := int(offset)
	for cursor < len(raw) {
		separator := bytes.Index(raw[cursor:], pairLogSeparator)
		if separator < 0 {
			return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
		}
		separator += cursor
		entry := raw[cursor:separator]
		headerEnd := bytes.Index(entry, []byte("\n\n"))
		if headerEnd < 0 || !validPairLogHeader(entry[:headerEnd]) {
			return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
		}
		body := entry[headerEnd+2:]
		if !utf8.Valid(body) {
			return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
		}
		if text := NormalizePairText(string(body)); text != "" {
			facts = append(facts, PairLogFact{Position: uint64(cursor), Text: text})
		}
		cursor = separator + len(pairLogSeparator)
	}
	return PairLogParseResult{Facts: facts}
}

func validPairLogHeader(raw []byte) bool {
	const prefix = "## "
	if !bytes.HasPrefix(raw, []byte(prefix)) {
		return false
	}
	_, err := time.Parse("2006-01-02 15:04:05", string(raw[len(prefix):]))
	return err == nil
}
