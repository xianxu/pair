package sessionsync

import (
	"fmt"
	"strings"
)

// proposeMaxLines is the largest hunk written as per-line markers (pair#661,
// the operator's first guess, to tune). A larger one is a side-doc proposal.
const proposeMaxLines = 3

// proposeRemedy is the one fallback for every change that can't be a
// one-line marker.
const proposeRemedy = "write this change as a side-doc proposal instead, linked in place by `→ [proposal](…)`"

// ProposeMarkers keeps baseline's text and writes each changed line of the
// diff to after as a review marker, one per line: a changed pair becomes
// `🤖~old~{new}` around the changed bytes, an unpaired old line `🤖~old~`, an
// unpaired new line its own `🤖{new}` line. A hunk past proposeMaxLines
// refuses with propose-too-large; a line the marker grammar can't carry
// (a `~` in old text, a bracket right after a marker, a newline change)
// refuses with propose-unencodable.
func ProposeMarkers(baseline, after string) (string, error) {
	a, b := splitLines(baseline), splitLines(after)
	var out strings.Builder
	at := 0
	for _, h := range LineDiff(a, b) {
		for _, l := range a[at:h.OldLo] {
			out.WriteString(l)
		}
		at = h.OldHi
		k, m := h.OldHi-h.OldLo, h.NewHi-h.NewLo
		if max(k, m) > proposeMaxLines {
			return "", &Refusal{Exit: ExitRefused, Code: "propose-too-large",
				Reason: fmt.Sprintf("the change at %s spans %d lines, over the %d-line marker limit; %s", lineRange(h), max(k, m), proposeMaxLines, proposeRemedy)}
		}
		for i := 0; i < max(k, m); i++ {
			var line string
			var err error
			switch {
			case i < k && i < m:
				line, err = markLine(a[h.OldLo+i], b[h.NewLo+i])
			case i < k:
				line, err = markLine(a[h.OldLo+i], "")
			default:
				line, err = markLine("", b[h.NewLo+i])
			}
			if err != nil {
				return "", &Refusal{Exit: ExitRefused, Code: "propose-unencodable",
					Reason: fmt.Sprintf("the change at %s can't be a marker: %v; %s", lineRange(h), err, proposeRemedy)}
			}
			out.WriteString(line)
		}
	}
	for _, l := range a[at:] {
		out.WriteString(l)
	}
	return out.String(), nil
}

// markLine writes one changed line. old or new is "" for an unpaired line.
func markLine(old, new string) (string, error) {
	oldBody, oldNL := strings.CutSuffix(old, "\n")
	newBody, newNL := strings.CutSuffix(new, "\n")
	nl := ""
	switch {
	case old == "":
		if newNL {
			nl = "\n"
		}
		return marker("", newBody) + nl, checkMarker("", "")
	case new == "":
		if oldNL {
			nl = "\n"
		}
		return marker(oldBody, "") + nl, checkMarker(oldBody, "")
	}
	if oldNL != newNL {
		return "", fmt.Errorf("the line's final newline changes")
	}
	if oldNL {
		nl = "\n"
	}
	p, s := narrow(oldBody, newBody)
	before, after := oldBody[:p], oldBody[len(oldBody)-s:]
	gone, came := oldBody[p:len(oldBody)-s], newBody[p:len(newBody)-s]
	if err := checkMarker(gone, after); err != nil {
		return "", err
	}
	return before + marker(gone, came) + after + nl, nil
}

// marker is `🤖~old~{new}`, `🤖~old~` or `🤖{new}`, delimiter text escaped
// as nvim/marker_codec.lua's esc_quote does.
func marker(old, new string) string {
	m := "🤖"
	if old != "" {
		m += "~" + escQuote(old) + "~"
	}
	if new != "" {
		m += "{" + escQuote(new) + "}"
	}
	return m
}

// checkMarker refuses what the Lua grammar would misread: a strike closes at
// the first `~` whatever its escaping, and a `[`/`{` right after a marker
// would be parsed as one more of its sections.
func checkMarker(old, follow string) error {
	if strings.Contains(old, "~") {
		return fmt.Errorf("the removed text contains `~`")
	}
	if strings.HasPrefix(follow, "[") || strings.HasPrefix(follow, "{") {
		return fmt.Errorf("the text after the change starts with a bracket")
	}
	return nil
}

func escQuote(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\', '<', '>', '[', ']', '{', '}':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// lineRange names a hunk's lines, 1-based, in the file as it stands.
func lineRange(h Hunk) string {
	if h.OldHi-h.OldLo <= 1 {
		return fmt.Sprintf("line %d", h.OldLo+1)
	}
	return fmt.Sprintf("lines %d-%d", h.OldLo+1, h.OldHi)
}
