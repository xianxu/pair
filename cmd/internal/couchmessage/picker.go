package couchmessage

import (
	"regexp"
	"strings"
)

// Picker is what a tail shows of an input-holding picker (pair#465): the
// reader's judgement over the rendered screen, never the wrapper's. Kind is
// "permission prompt" or "question" only when the screen's wording says so,
// else "picker"; a zero Picker means the tail shows none.
type Picker struct {
	Kind    string
	Title   string
	Detail  string // a permission prompt's header and command, e.g. "Bash command: git push origin HEAD"
	Options []string
}

const (
	PickerPermission = "permission prompt"
	PickerQuestion   = "question"
	PickerUnknown    = "picker"
)

var (
	tailMarkup = regexp.MustCompile(`‹(/?dim|/?rev|cursor)›`)
	// A picker's options are numbered; its cursor glyph marks the selected one.
	selectedOption = regexp.MustCompile(`^[❯›>]\s*\d+\.\s`)
	optionLine     = regexp.MustCompile(`^[❯›>]?\s*\d+\.\s`)
	// TailRule is a full-width rule: a separator, a dashed rule, or a box's
	// top or bottom border.
	TailRule = regexp.MustCompile(`^([─━\-═╌]+|[╭╰][─━]+[╮╯])$`)
	dashRule = regexp.MustCompile(`^╌+$`)
	// permissionTitle is Claude Code's wording for a tool permission prompt.
	permissionTitle = regexp.MustCompile(`^(Do you want to|Would you like to)\b`)
)

// StripTailMarkup renders one tail line (tail.go's grammar) as plain text,
// without a dialog box's side borders.
func StripTailMarkup(line string) string {
	const escaped = "\x00"
	line = strings.ReplaceAll(line, "‹‹", escaped)
	line = tailMarkup.ReplaceAllString(line, "")
	return strings.Trim(strings.ReplaceAll(line, escaped, "‹"), " \t│┃")
}

// ReadPicker reads the screen's last picker: the block around its selected
// option, scanned from the bottom so a numbered list in the agent's prose is
// never taken for one.
func ReadPicker(tail []string) Picker {
	lines := make([]string, len(tail))
	for i, l := range tail {
		lines[i] = StripTailMarkup(l)
	}
	sel := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if selectedOption.MatchString(lines[i]) {
			sel = i
			break
		}
	}
	if sel < 0 {
		return Picker{}
	}
	boundary := func(t string) bool { return t == "" || TailRule.MatchString(t) }
	// The options block: up to its first option (descriptions may sit between
	// options), down past a rule when an option follows it.
	first := sel
	for i := sel - 1; i >= 0 && !boundary(lines[i]); i-- {
		if optionLine.MatchString(lines[i]) {
			first = i
		}
	}
	var p Picker
	for i := first; i < len(lines); i++ {
		t := lines[i]
		if TailRule.MatchString(t) && i+1 < len(lines) && optionLine.MatchString(lines[i+1]) {
			continue
		}
		if boundary(t) {
			break
		}
		if optionLine.MatchString(t) {
			p.Options = append(p.Options, strings.TrimSpace(strings.TrimLeft(t, "❯›>")))
		}
	}
	// The title: the line right above the first option, or the paragraph above
	// a blank line when the options stand alone.
	titleEnd := first - 1
	if titleEnd >= 0 && !boundary(lines[titleEnd]) {
		p.Title = lines[titleEnd]
	} else {
		for titleEnd >= 0 && lines[titleEnd] == "" {
			titleEnd--
		}
		start := titleEnd
		for start-1 >= 0 && !boundary(lines[start-1]) {
			start--
		}
		if titleEnd >= 0 && !boundary(lines[titleEnd]) {
			p.Title = strings.Join(lines[start:titleEnd+1], " ")
		}
	}
	footer := ""
	for i := sel + 1; i < len(lines); i++ {
		footer += " " + lines[i]
	}
	switch {
	case permissionTitle.MatchString(p.Title):
		p.Kind = PickerPermission
		p.Detail = permissionDetail(lines, titleEnd)
	case strings.Contains(footer, "Enter to select"):
		p.Kind = PickerQuestion
	default:
		p.Kind = PickerUnknown
	}
	return p
}

// permissionDetail is the prompt's header and command: the first line after
// the rule that opens the prompt, then the lines between its first pair of
// dashed rules, or else the line after the header.
func permissionDetail(lines []string, title int) string {
	open := -1
	for i := title - 1; i >= 0; i-- {
		if TailRule.MatchString(lines[i]) && !dashRule.MatchString(lines[i]) {
			open = i
			break
		}
	}
	if open < 0 {
		return ""
	}
	var header string
	var command []string
	dashes := 0
	for i := open + 1; i < title && dashes < 2; i++ {
		t := lines[i]
		switch {
		case dashRule.MatchString(t):
			dashes++
		case t == "":
		case header == "":
			header = t
		case dashes == 1:
			command = append(command, t)
		}
	}
	if len(command) == 0 {
		// No dashed rules: the line after the header is the command.
		for i := open + 1; i < title; i++ {
			if t := lines[i]; t != "" && t != header && !TailRule.MatchString(t) {
				command = []string{t}
				break
			}
		}
	}
	if header == "" {
		return ""
	}
	if len(command) == 0 {
		return header
	}
	return header + ": " + strings.Join(command, " ")
}
