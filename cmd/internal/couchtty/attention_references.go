package couchtty

import (
	"regexp"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// Path/URL alternatives precede prose tokens and retain their internal brackets.
var referenceTokens = regexp.MustCompile(`[^\s()\[\]{}<>,;!?"'\x60/\\]*[/\\][^\s<>,;"'\x60]+|[^\s()\[\]{}<>,;!?"'\x60]+`)

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
	for _, span := range referenceTokens.FindAllStringIndex(text, -1) {
		token := strings.TrimRight(text[span[0]:span[1]], ".:")
		if strings.ContainsAny(token, `/\\@`) {
			continue
		}
		if match(token) {
			for i := span[0]; i < span[0]+len(token); i++ {
				accents[i] = true
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
