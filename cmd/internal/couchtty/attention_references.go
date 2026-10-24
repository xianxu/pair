package couchtty

import (
	"regexp"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
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

// referenceAccents recognizes only complete standalone display-row words.
// Wrapped row edge words may be fragments, so they conservatively stay plain.
func referenceAccents(text string, skip int, wrapped bool, match func(string) bool) []bool {
	accents := make([]bool, len(text))
	words := strings.Fields(text)
	offset := 0
	for i, word := range words {
		at := offset + strings.Index(text[offset:], word)
		offset = at + len(word)
		if at < skip || wrapped && (i == 0 || i == len(words)-1) {
			continue
		}
		left := strings.TrimLeft(word, ",;:!?\"'`")
		token := strings.TrimRight(left, ".,;:!?\"'`")
		if match(token) {
			start := at + len(word) - len(left)
			for j := start; j < start+len(token); j++ {
				accents[j] = true
			}
		}
	}
	return accents
}
