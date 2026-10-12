package assets

import (
	"net/url"
	"strings"
)

// RewriteCSS passes every URL a stylesheet would load through to and
// returns the sheet with each one replaced: url(…) (bare or quoted), an
// @import string, and a plain string directly inside image-set() or
// -webkit-image-set(). Each is resolved against base first (an http or https
// URL; empty, data: and #fragment references are kept, and so is anything
// that doesn't resolve). A URL is rewritten only when to changes it, so a
// sheet whose URLs all come back unchanged keeps its bytes. Comments are
// copied as they are.
//
// The store absolutizes with it at Put (to returns its argument); the proxy
// points a served sheet at assets it holds (to returns a hash). The viewer's
// css.js finds the same spans: tests/webcast/css_urls.json holds both.
func RewriteCSS(text, base string, to func(abs string) string) string {
	s := cssScan{src: text, to: to}
	if base != "" {
		s.base, _ = url.Parse(base)
	}
	s.run()
	return s.out.String()
}

type cssScan struct {
	src  string
	base *url.URL
	to   func(string) string
	out  strings.Builder
	i    int
	// sets holds, for each image-set( we are inside, the paren depth of its
	// arguments; a string at that depth is a URL.
	depth int
	sets  []int
}

func (s *cssScan) run() {
	for s.i < len(s.src) {
		c := s.src[s.i]
		switch {
		case c == '/' && strings.HasPrefix(s.src[s.i:], "/*"):
			end := strings.Index(s.src[s.i+2:], "*/")
			if end < 0 {
				s.copyTo(len(s.src))
			} else {
				s.copyTo(s.i + 2 + end + 2)
			}
		case c == '"' || c == '\'':
			start := s.i
			raw, end, ok := readString(s.src, s.i)
			if ok && len(s.sets) > 0 && s.sets[len(s.sets)-1] == s.depth {
				s.out.WriteString(s.replace(s.src[start:end], raw, quote))
				s.i = end
			} else {
				s.copyTo(end)
			}
		case c == '(':
			s.depth++
			s.copyTo(s.i + 1)
		case c == ')':
			if len(s.sets) > 0 && s.sets[len(s.sets)-1] == s.depth {
				s.sets = s.sets[:len(s.sets)-1]
			}
			s.depth--
			s.copyTo(s.i + 1)
		case c == '@' && hasFold(s.src[s.i:], "@import"):
			s.copyTo(s.i + len("@import"))
			j := skipSpace(s.src, s.i)
			if j < len(s.src) && (s.src[j] == '"' || s.src[j] == '\'') {
				s.copyTo(j)
				raw, end, ok := readString(s.src, j)
				if ok {
					s.out.WriteString(s.replace(s.src[j:end], raw, quote))
					s.i = end
				}
			}
		case identStart(s.src, s.i) && hasFold(s.src[s.i:], "url("):
			s.url()
		case identStart(s.src, s.i) && (hasFold(s.src[s.i:], "image-set(") || hasFold(s.src[s.i:], "-webkit-image-set(")):
			open := strings.IndexByte(s.src[s.i:], '(')
			s.copyTo(s.i + open + 1)
			s.depth++
			s.sets = append(s.sets, s.depth)
		default:
			s.copyTo(s.i + 1)
		}
	}
}

// url handles url( at s.i: a quoted or bare reference up to its ')'.
func (s *cssScan) url() {
	start := s.i
	j := skipSpace(s.src, s.i+len("url("))
	var raw string
	end := -1
	if j < len(s.src) && (s.src[j] == '"' || s.src[j] == '\'') {
		r, k, ok := readString(s.src, j)
		if ok {
			if k = skipSpace(s.src, k); k < len(s.src) && s.src[k] == ')' {
				raw, end = r, k+1
			}
		}
	} else if k := strings.IndexByte(s.src[j:], ')'); k >= 0 {
		raw, end = strings.TrimSpace(s.src[j:j+k]), j+k+1
	}
	if end < 0 {
		s.copyTo(len(s.src)) // unterminated: nothing after it is CSS
		return
	}
	s.out.WriteString(s.replace(s.src[start:end], raw, func(v string) string { return "url(" + quote(v) + ")" }))
	s.i = end
}

// replace is the text for one reference: the original when it isn't a URL
// to rewrite or to leaves it as it was, else form(new).
func (s *cssScan) replace(orig, raw string, form func(string) string) string {
	abs := s.resolve(raw)
	if abs == "" {
		return orig
	}
	to := s.to(abs)
	if to == raw {
		return orig
	}
	return form(to)
}

func (s *cssScan) resolve(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] == '#' || hasFold(raw, "data:") {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if s.base != nil {
		u = s.base.ResolveReference(u)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.String()
}

func (s *cssScan) copyTo(end int) {
	s.out.WriteString(s.src[s.i:end])
	s.i = end
}

// readString reads the CSS string starting at quote src[i]: its value (with
// backslash escapes of single characters undone), and the index after its
// closing quote. ok is false when it runs to the end, or to a newline.
func readString(src string, i int) (value string, end int, ok bool) {
	q := src[i]
	var b strings.Builder
	for j := i + 1; j < len(src); j++ {
		switch c := src[j]; {
		case c == q:
			return b.String(), j + 1, true
		case c == '\n':
			return "", j, false
		case c == '\\' && j+1 < len(src):
			j++
			b.WriteByte(src[j])
		default:
			b.WriteByte(c)
		}
	}
	return "", len(src), false
}

// quote is a CSS double-quoted string.
func quote(v string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\a `).Replace(v) + `"`
}

func skipSpace(src string, i int) int {
	for i < len(src) && strings.IndexByte(" \t\n\r\f", src[i]) >= 0 {
		i++
	}
	return i
}

func hasFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// identStart: src[i] begins a name (the previous byte can't continue one),
// so "myurl(" is not url(.
func identStart(src string, i int) bool {
	if i == 0 {
		return true
	}
	c := src[i-1]
	return !(c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80)
}
