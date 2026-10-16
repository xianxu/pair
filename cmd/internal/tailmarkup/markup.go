// Package tailmarkup renders the styled terminal-tail grammar shared by readers.
package tailmarkup

import (
	"regexp"
	"strings"
)

var markup = regexp.MustCompile(`‹(/?dim|/?rev|cursor)›`)

// Strip renders plain text without dialog side borders. Doubled opening marks
// escape literal text, so literal tag spellings are preserved.
func Strip(line string) string {
	const escaped = "\x00"
	line = strings.ReplaceAll(line, "‹‹", escaped)
	line = markup.ReplaceAllString(line, "")
	return strings.Trim(strings.ReplaceAll(line, escaped, "‹"), " \t│┃")
}
