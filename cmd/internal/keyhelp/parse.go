package keyhelp

import (
	"strings"
)

// NvimKeymap is one `vim.keymap.set` call carrying a `pair:` desc.
type NvimKeymap struct {
	Key  string // the lhs, when it is a quoted literal ("<M-CR>")
	Raw  string // the lhs argument as written, for Dynamic/Unresolved rows
	Desc string // the desc with the "pair: " prefix stripped and trimmed
}

// KeymapScan is a THREE-WAY classification of the keymaps in a Lua source, not a
// flat slice.
//
// The lhs argument comes in three syntactic forms and conflating them is how a
// parser lies about its own coverage. Specifically: a "second quoted string is the
// lhs" rule reads init.lua:3872 (`vim.keymap.set('i', open, …, desc = 'pair:
// autopair ' .. open)`) as Key == "pair: autopair " — a plausible-looking garbage
// row — while a `count == number of desc lines` guard still passes, because the
// count is right and only the ASSIGNMENT is wrong. So the lhs is taken by argument
// POSITION, and anything that is not a quoted literal is reported rather than
// guessed.
type KeymapScan struct {
	Resolved   []NvimKeymap // lhs is a quoted literal — usable
	Dynamic    []NvimKeymap // lhs is an interpolated literal ('<M-' .. i .. '>')
	Unresolved []NvimKeymap // lhs is an unquoted expression (open, tostring(i))
}

const keymapCall = "vim.keymap.set("

// ParseNvimKeymaps extracts every `vim.keymap.set` call whose desc begins
// "pair: ". Pure: takes source text, returns a classification.
func ParseNvimKeymaps(src string) KeymapScan {
	var scan KeymapScan
	for idx := 0; ; {
		i := strings.Index(src[idx:], keymapCall)
		if i < 0 {
			return scan
		}
		start := idx + i + len(keymapCall)
		body, next := callBody(src, start)
		idx = next

		desc, ok := pairDesc(body)
		if !ok {
			continue
		}
		raw := strings.TrimSpace(argAt(body, 1))
		km := NvimKeymap{Raw: raw, Desc: desc}
		switch {
		case isQuotedLiteral(raw):
			km.Key = unquote(raw)
			scan.Resolved = append(scan.Resolved, km)
		case strings.Contains(raw, "..") && strings.ContainsAny(raw, `'"`):
			scan.Dynamic = append(scan.Dynamic, km)
		default:
			scan.Unresolved = append(scan.Unresolved, km)
		}
	}
}

// callBody returns the text of a call's arguments starting at start (just past the
// open paren) up to the matching close paren, and the index to resume scanning
// from. Quote-aware so a paren inside a string does not close the call early.
func callBody(src string, start int) (body string, next int) {
	depth := 1
	var quote byte
	for i := start; i < len(src); i++ {
		c := src[i]
		if quote != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
			if depth == 0 {
				return src[start:i], i + 1
			}
		}
	}
	return src[start:], len(src)
}

// argAt returns the n-th (0-based) top-level comma-separated argument of a call
// body. Top-level means commas inside nested braces/parens/strings do not split —
// which is what makes `{ 'n', 'i' }` count as ONE argument, so the lhs is reliably
// argument 1.
func argAt(body string, n int) string {
	depth := 0
	var quote byte
	arg := 0
	begin := 0
	for i := 0; i < len(body); i++ {
		c := body[i]
		if quote != 0 {
			if c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
		case ',':
			if depth == 0 {
				if arg == n {
					return body[begin:i]
				}
				arg++
				begin = i + 1
			}
		}
	}
	if arg == n {
		return body[begin:]
	}
	return ""
}

const descMarker = "desc = 'pair: "

// pairDesc pulls the `pair:` description out of a call body, stripping the prefix
// and trimming. Several real descs carry a trailing space ('pair: autopair ').
func pairDesc(body string) (string, bool) {
	i := strings.Index(body, descMarker)
	if i < 0 {
		return "", false
	}
	rest := body[i+len(descMarker):]
	end := strings.IndexByte(rest, '\'')
	if end < 0 {
		return "", false
	}
	return strings.TrimSpace(rest[:end]), true
}

// isQuotedLiteral reports whether an argument is a plain quoted string with no
// concatenation — the only form whose key is knowable statically.
func isQuotedLiteral(s string) bool {
	if len(s) < 2 || strings.Contains(s, "..") {
		return false
	}
	q := s[0]
	return (q == '\'' || q == '"') && s[len(s)-1] == q
}

func unquote(s string) string { return s[1 : len(s)-1] }

// isBarePrintableKey covers lhs values that are real keys but not <...> forms —
// `z=`, `ZZ`, `ZQ`, digits. Used by the parser's shape invariant test.
func isBarePrintableKey(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			return false
		}
	}
	return true
}

// ZellijBind is one `bind "<key>" { Run … }` from config.kdl.
type ZellijBind struct {
	Key string
}

// ParseZellijRunBinds extracts the zellij-level binds that Run a command.
//
// Only `Run` binds are user-facing in their own right; `WriteChars`/`Write <n>;`
// binds are plumbing that forwards an escape sequence to nvim, whose keymap desc
// already documents the behaviour. Attribution is per-bind-block (the body between
// this bind's braces), so a Write bind sitting immediately before a Run bind — the
// config.kdl:157→163 shape — cannot have the Run attributed to it.
func ParseZellijRunBinds(src string) []ZellijBind {
	var out []ZellijBind
	const marker = `bind "`
	for idx := 0; ; {
		i := strings.Index(src[idx:], marker)
		if i < 0 {
			return out
		}
		at := idx + i
		// Reject `unbind "…"`, which contains `bind "`.
		if at >= 2 && strings.HasSuffix(src[:at], "un") {
			idx = at + len(marker)
			continue
		}
		keyStart := at + len(marker)
		keyEnd := strings.IndexByte(src[keyStart:], '"')
		if keyEnd < 0 {
			return out
		}
		key := src[keyStart : keyStart+keyEnd]
		rest := src[keyStart+keyEnd:]
		brace := strings.IndexByte(rest, '{')
		if brace < 0 {
			return out
		}
		body, _ := callBody(rest, brace+1)
		if containsRunVerb(body) {
			out = append(out, ZellijBind{Key: key})
		}
		idx = keyStart + keyEnd
	}
}

// containsRunVerb reports whether a bind body invokes Run (as opposed to
// WriteChars / Write).
func containsRunVerb(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "Run ") || strings.HasPrefix(t, "Run\t") {
			return true
		}
	}
	return false
}
