package sessioninventory

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// PairLogFact is one ordered authored markdown-log entry after normalization.
// Position is its durable byte offset in the scoped Pair log.
// pair:155-concept pure new M2
type PairLogFact struct {
	ScopeKey     string
	Tag          string
	Agent        Agent
	Position     uint64
	Text         string
	AuthoredText string
	AppendID     string
}

// PairLedgerFact is the inventory boundary's normalized durable ledger fact.
// pair:155-concept pure new M2
type PairLedgerFact = sessionledger.Record

type PairLogParseResult struct {
	Facts            []PairLogFact
	Entries          []PairLogEntry
	MalformedOffsets []uint64
}

type PairLogEntry struct {
	Position     uint64
	AuthoredText string
	AppendID     string
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

// EncodePairLogEntry is the versioned, byte-counted framing shared by the
// durable writer and parser. The body may contain arbitrary Markdown,
// including the legacy visual separator and header-shaped text.
func EncodePairLogEntry(body []byte, now time.Time) []byte {
	return EncodePairLogEntryWithID(body, now, "")
}

func EncodePairLogEntryWithID(body []byte, now time.Time, appendID string) []byte {
	marker := fmt.Sprintf("<!-- pair-log-v1 bytes=%d", len(body))
	if appendID != "" {
		marker += " append_id=" + appendID
	}
	header := fmt.Sprintf("## %s\n%s -->\n\n", now.Format("2006-01-02 15:04:05"), marker)
	entry := make([]byte, 0, len(header)+len(body)+len(pairLogSeparator))
	entry = append(entry, header...)
	entry = append(entry, body...)
	entry = append(entry, pairLogSeparator...)
	return entry
}

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
	var entries []PairLogEntry
	cursor := int(offset)
	for cursor < len(raw) {
		headerEndRelative := bytes.Index(raw[cursor:], []byte("\n\n"))
		if headerEndRelative < 0 {
			return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
		}
		headerEnd := cursor + headerEndRelative
		bodyStart := headerEnd + 2
		bodyBytes, appendID, versioned, valid := pairLogHeader(raw[cursor:headerEnd])
		if !valid {
			return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
		}
		separator := -1
		if versioned {
			if bodyBytes > uint64(len(raw)-bodyStart) {
				return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
			}
			separator = bodyStart + int(bodyBytes)
			if !bytes.HasPrefix(raw[separator:], pairLogSeparator) {
				return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
			}
		} else {
			separatorRelative := bytes.Index(raw[bodyStart:], pairLogSeparator)
			if separatorRelative < 0 {
				return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
			}
			separator = bodyStart + separatorRelative
		}
		body := raw[bodyStart:separator]
		if !utf8.Valid(body) {
			return PairLogParseResult{MalformedOffsets: []uint64{uint64(cursor)}}
		}
		entries = append(entries, PairLogEntry{Position: uint64(cursor), AuthoredText: string(body), AppendID: appendID})
		if text := NormalizePairText(string(body)); text != "" {
			facts = append(facts, PairLogFact{Position: uint64(cursor), Text: text, AuthoredText: string(body), AppendID: appendID})
		}
		cursor = separator + len(pairLogSeparator)
	}
	return PairLogParseResult{Facts: facts, Entries: entries}
}

func pairLogHeader(raw []byte) (uint64, string, bool, bool) {
	const prefix = "## "
	lines := bytes.Split(raw, []byte{'\n'})
	if len(lines) == 0 || !bytes.HasPrefix(lines[0], []byte(prefix)) {
		return 0, "", false, false
	}
	if _, err := time.Parse("2006-01-02 15:04:05", string(lines[0][len(prefix):])); err != nil {
		return 0, "", false, false
	}
	if len(lines) == 1 {
		return 0, "", false, true
	}
	if len(lines) != 2 {
		return 0, "", false, false
	}
	const markerPrefix, markerSuffix = "<!-- pair-log-v1 bytes=", " -->"
	marker := string(lines[1])
	if !strings.HasPrefix(marker, markerPrefix) || !strings.HasSuffix(marker, markerSuffix) {
		return 0, "", false, false
	}
	fields := strings.Fields(strings.TrimSuffix(strings.TrimPrefix(marker, markerPrefix), markerSuffix))
	if len(fields) < 1 || len(fields) > 2 {
		return 0, "", false, false
	}
	count, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0, "", false, false
	}
	appendID := ""
	if len(fields) == 2 {
		if !strings.HasPrefix(fields[1], "append_id=") {
			return 0, "", false, false
		}
		appendID = strings.TrimPrefix(fields[1], "append_id=")
		if !ValidPairLogAppendID(appendID) {
			return 0, "", false, false
		}
	}
	return count, appendID, true, true
}

func ValidPairLogAppendID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
