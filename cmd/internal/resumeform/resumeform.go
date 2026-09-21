// Package resumeform owns the per-agent spellings of a resume binding and
// their removal from argv. Extraction (launcher), persisted-config stripping
// (launcher and sessionwatch) and fresh-arg validation all read this one
// table, so a spelling accepted at one site cannot be lost at another
// (pair#300 BR-15).
package resumeform

import "strings"

// Form is one agent's resume spellings. Space forms take the id from the next
// token, and only when that token is not itself a flag — `--resume [id]` is
// optional-valued; inline forms carry it after `=`; glued forms attach it
// directly to a short flag (`-r<id>`). A cluster that merely contains the
// short selector is spelled identically in argv, so both normalize to the
// glued reading.
type Form struct {
	Space  []string
	Inline []string
	Glued  []string
}

var Forms = map[string]Form{
	"claude": {
		Space:  []string{"--resume", "-r"},
		Inline: []string{"--resume=", "-r="},
		Glued:  []string{"-r"},
	},
	"agy": {
		Space:  []string{"--resume", "--conversation"},
		Inline: []string{"--conversation="},
	},
	"qoder": {
		Space:  []string{"--resume", "-r"},
		Inline: []string{"--resume=", "-r="},
		Glued:  []string{"-r"},
	},
}

// Selector reports whether one argv token spells a resume binding for the
// agent: a space form (bare or with `=value`), an inline form, or a glued
// short form. Fresh launches refuse these.
func Selector(agent, tok string) bool {
	form, ok := Forms[agent]
	if !ok {
		return false
	}
	flag, _, _ := strings.Cut(tok, "=")
	if hasSpelling(form.Space, flag) {
		return true
	}
	if _, ok := hasPrefix(form.Inline, tok); ok {
		return true
	}
	return gluedValue(form.Glued, tok) != ""
}

// Extract returns the session id the agent's spellings pin on argv, or "".
// Only a non-empty value pins: a bare `flag=` or a space form followed by a
// flag keeps scanning.
func Extract(agent string, args []string) string {
	form, ok := Forms[agent]
	if !ok {
		return ""
	}
	prev := ""
	for _, tok := range args {
		if !strings.HasPrefix(tok, "-") && hasSpelling(form.Space, prev) {
			return tok
		}
		if v, ok := hasPrefix(form.Inline, tok); ok && v != "" {
			return v
		}
		if v := gluedValue(form.Glued, tok); v != "" {
			return v
		}
		prev = tok
	}
	return ""
}

// Strip removes every spelling in the table from args, preserving order. A
// space form followed by another flag was valueless: only the flag token is
// dropped, never the next flag. Agent-agnostic: a spelling the current agent
// never uses strips as a no-op, so both persist sites share this superset.
func Strip(args []string) []string {
	glued := gluedSpellings()
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case spaceForm(arg):
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
		case inlineForm(arg), gluedValue(glued, arg) != "":
		default:
			out = append(out, arg)
		}
	}
	return out
}

func spaceForm(tok string) bool {
	for _, form := range Forms {
		if hasSpelling(form.Space, tok) {
			return true
		}
	}
	return false
}

func inlineForm(tok string) bool {
	for _, form := range Forms {
		if _, ok := hasPrefix(form.Inline, tok); ok {
			return true
		}
	}
	return false
}

func gluedSpellings() []string {
	var all []string
	for _, form := range Forms {
		all = append(all, form.Glued...)
	}
	return all
}

func hasSpelling(spellings []string, tok string) bool {
	for _, spelling := range spellings {
		if tok == spelling {
			return true
		}
	}
	return false
}

func hasPrefix(prefixes []string, tok string) (string, bool) {
	for _, prefix := range prefixes {
		if v, ok := strings.CutPrefix(tok, prefix); ok {
			return v, true
		}
	}
	return "", false
}

// gluedValue returns the attached id when tok is a glued short form; a bare
// spelling or an inline `flag=` token yields "".
func gluedValue(spellings []string, tok string) string {
	for _, spelling := range spellings {
		if v, ok := strings.CutPrefix(tok, spelling); ok && v != "" && !strings.HasPrefix(v, "=") {
			return v
		}
	}
	return ""
}
