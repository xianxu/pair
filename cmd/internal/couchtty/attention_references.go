package couchtty

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

var referencePattern = regexp.MustCompile(`^(?:[A-Za-z0-9][A-Za-z0-9_.-]*[:#][0-9]+|#[0-9]+)$`)

func referenceToken(token string) bool {
	if !referencePattern.MatchString(token) {
		return false
	}
	if strings.Contains(token, ":") {
		ref, matched, err := couchcore.ParseWorkspaceReference(token)
		return matched && err == nil && ref.Repo != ""
	}
	return true
}

// referenceAccents marks source tokens before wrapping can split paths or URLs.
func referenceAccents(text string, match func(string) bool) []bool {
	accents := make([]bool, len(text))
	offset := 0
	for _, chunk := range strings.FieldsFunc(text, unicode.IsSpace) {
		start := offset + strings.Index(text[offset:], chunk)
		offset = start + len(chunk)
		if strings.ContainsAny(chunk, `/\\@`) {
			continue
		}
		pos := start
		for _, part := range strings.FieldsFunc(chunk, func(r rune) bool { return strings.ContainsRune("()[]{}<>,;!?\"'`“”‘’", r) }) {
			at := pos + strings.Index(text[pos:offset], part)
			pos = at + len(part)
			token := strings.TrimRight(part, ".:")
			if match(token) {
				for i := at; i < at+len(token); i++ {
					accents[i] = true
				}
			}
		}
	}
	return accents
}

// sanitizedNote preserves paragraph boundaries for the shared wrapper.
func sanitizedNote(text string) string {
	parts := strings.Split(text, "\n")
	for i := range parts {
		parts[i] = rowtext.Sanitize(strings.ReplaceAll(parts[i], "\t", " "))
	}
	return strings.Join(parts, "\n")
}

// projectReferenceAccents walks forward through the unchanged wrapper's lines.
// A clipped final ellipsis is synthetic and receives no source accent.
func projectReferenceAccents(source, line string, accents []bool, offset *int) []bool {
	if len(accents) == 0 {
		return nil
	}
	text := strings.TrimSuffix(line, "…")
	at := strings.Index(source[*offset:], text)
	if at < 0 {
		return nil
	}
	at += *offset
	*offset = at + len(text)
	result := make([]bool, len(line))
	copy(result, accents[at:*offset])
	return result
}
