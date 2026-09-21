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
// directly to a short flag (`-r<id>`). A short-flag cluster that contains the
// glued letter shares the glued reading only at the first position (`-r<id>`);
// elsewhere (`-pr`) the CLI would read the letters before it, so the cluster
// letter is refused outright — see ShortLetters.
type Form struct {
	Space  []string
	Inline []string
	Glued  []string
}

var forms = map[string]Form{
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

// Forms returns a copy of the whole table. Read-only consumers and tests use
// it; the table itself stays unexported so no package can write the single
// source of truth.
func Forms() map[string]Form {
	out := make(map[string]Form, len(forms))
	for agent, form := range forms {
		out[agent] = Form{
			Space:  append([]string(nil), form.Space...),
			Inline: append([]string(nil), form.Inline...),
			Glued:  append([]string(nil), form.Glued...),
		}
	}
	return out
}

// ShortLetters returns the agent's glued short-flag letters as a string
// (`-r` yields `r`). A fresh launch must refuse any short-flag cluster that
// contains one of them, in any position: the cluster form is a resume binding
// the moment the letter is present, and the validator cannot tell `-pr` apart
// from a legitimately clustered `-p` with a glued `r` value.
func ShortLetters(agent string) string {
	var letters string
	for _, spelling := range forms[agent].Glued {
		letter, ok := strings.CutPrefix(spelling, "-")
		if ok && len(letter) == 1 {
			letters += letter
		}
	}
	return letters
}

// Selector reports whether one argv token spells a resume binding for the
// agent: a space form (bare or with `=value`), an inline form, or a glued
// short form. Fresh launches refuse these.
func Selector(agent, tok string) bool {
	form, ok := forms[agent]
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
	form, ok := forms[agent]
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

// Strip removes the agent's own spellings from args, preserving order. A
// space form followed by another flag was valueless: only the flag token is
// dropped, never the next flag. Strictly per-agent: another agent's spelling
// (or a glued `-r<x>` for an agent with no glued form) is not a resume binding
// here and is preserved.
func Strip(agent string, args []string) []string {
	form := forms[agent]
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case hasSpelling(form.Space, arg):
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
		case inlineToken(form, arg), gluedValue(form.Glued, arg) != "":
		default:
			out = append(out, arg)
		}
	}
	return out
}

func inlineToken(form Form, tok string) bool {
	_, ok := hasPrefix(form.Inline, tok)
	return ok
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
