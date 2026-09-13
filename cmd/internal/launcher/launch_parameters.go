package launcher

import (
	"fmt"
	"strings"
	"unicode"
)

// ParseLaunchParameters parses quoting only: it never expands variables, globs,
// commands, or escapes into control characters. Callers own input-size policy.
func ParseLaunchParameters(text string) ([]string, error) {
	args := []string{}
	var word strings.Builder
	var quote rune
	active, escape := false, false
	for _, r := range text {
		if r == 0 {
			return nil, fmt.Errorf("launch parameters contain NUL")
		}
		if escape {
			word.WriteRune(r)
			active = true
			escape = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escape = true
			active = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			active = true
			continue
		}
		if unicode.IsSpace(r) {
			if active {
				args = append(args, word.String())
				word.Reset()
				active = false
			}
			continue
		}
		word.WriteRune(r)
		active = true
	}
	if escape || quote != 0 {
		return nil, fmt.Errorf("unfinished quote or escape in launch parameters")
	}
	if active {
		args = append(args, word.String())
	}
	return args, nil
}

// FormatLaunchParameters produces a lossless, editable quoting representation.
func FormatLaunchParameters(args []string) string {
	out := make([]string, len(args))
	for i, arg := range args {
		if arg != "" && !strings.ContainsAny(arg, "'\"\\") && strings.IndexFunc(arg, unicode.IsSpace) < 0 {
			out[i] = arg
			continue
		}
		out[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	return strings.Join(out, " ")
}
