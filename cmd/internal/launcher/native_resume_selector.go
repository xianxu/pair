package launcher

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/resumeform"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type NativeResumeKind string

const (
	NativeResumeNone     NativeResumeKind = "none"
	NativeResumeExplicit NativeResumeKind = "explicit"
	NativeResumeLatest   NativeResumeKind = "latest"
	NativeResumePicker   NativeResumeKind = "picker"
	NativeResumeFork     NativeResumeKind = "fork"
	NativeResumeNew      NativeResumeKind = "new"
)

// NativeResumeSelector retains the caller invocation. Only Explicit, Latest
// and Picker select an existing writer; Fork and New allocate a distinct one.
// NativeID is the resume/fork source; NewNativeID is an explicit destination.
type NativeResumeSelector struct {
	Agent                 string
	Kind                  NativeResumeKind
	NativeID              string
	NewNativeID           string
	Args                  []string
	IncludeNonInteractive bool
	All                   bool
	CWD                   string
	Remote                bool
	ScopeExpanded         bool
	remove                map[int]bool
	replace               map[int]string
	command               int
	positional            int
	delimiter             int
}

// ParseNativeResumeSelector reads native selectors only in option/command
// positions. A delimiter ends options, but a pending resume command still takes
// its first positional session ID; remaining positional text is the prompt.
func ParseNativeResumeSelector(agent string, args []string) (NativeResumeSelector, error) {
	s := NativeResumeSelector{Agent: agent, Kind: NativeResumeNone, Args: append([]string(nil), args...), remove: map[int]bool{}, replace: map[int]string{}, command: -1, positional: -1, delimiter: -1}
	if !IsSupportedAgent(agent) {
		return s, nil
	}
	flags := args
	form := resumeform.Forms()[agent]
	fork, newID, commandSeen, execSeen, last := false, false, false, false, false
	selected := false
	selectKind := func(kind NativeResumeKind, id string) error {
		if selected {
			return errors.New("multiple native conversation selectors")
		}
		selected = true
		s.Kind = kind
		s.NativeID = id
		return nil
	}
	// --last changes the first positional after resume into the optional prompt.
	// Inspect only actual option positions in the main walk before applying it.
	for i := 0; i < len(flags); i++ {
		a := flags[i]
		if strings.ContainsRune(a, 0) {
			return s, errors.New("native arguments contain NUL")
		}
		if s.delimiter >= 0 {
			if s.command >= 0 && s.positional < 0 && !last {
				s.positional = i
				s.NativeID = a
				if a != "" {
					s.Kind = NativeResumeExplicit
				}
				s.remove[i] = true
			}
			continue
		}
		if a == "--" {
			s.delimiter = i
			continue
		}
		name, value, inline := strings.Cut(a, "=")
		if (agent == "codex" && strings.HasPrefix(a, "-C") && len(a) > 2) || (agent == "qoder" && strings.HasPrefix(a, "-w") && len(a) > 2) {
			s.CWD = strings.TrimPrefix(a[2:], "=")
			continue
		}
		if name == "--remote" || name == "--remote-session" || name == "--teleport" || name == "--cloud" {
			s.Remote = true
		}
		if name == "--worktree" || (name == "-w" && agent != "qoder") || (agent == "qoder" && name == "--config-dir") {
			s.ScopeExpanded = true
		}
		if agent == "muse" && (name == "--worktree" || name == "-w") && !inline {
			if i+1 < len(flags) && (flags[i+1] == "off" || flags[i+1] == "create" || flags[i+1] == "existing") {
				i++
			}
			continue
		}
		if name == "--fork-session" {
			fork = true
			continue
		}
		if name == "--session-id" || (agent == "grok" && (name == "-s" || resumeform.HasSessionID(agent, []string{a}))) {
			newID = true
			if name == "--session-id" || name == "-s" {
				if inline {
					s.NewNativeID = value
				} else {
					if i+1 >= len(flags) || strings.HasPrefix(flags[i+1], "-") {
						return s, errors.New("missing new native session ID")
					}
					i++
					s.NewNativeID = flags[i]
				}
			} else {
				// Grok accepts the destination glued to its short -s option.
				s.NewNativeID = a[2:]
			}
			continue
		}
		if (agent == "codex" || agent == "muse") && name == "--last" {
			if inline {
				return s, errors.New("--last does not take a value")
			}
			last = true
			s.remove[i] = true
			continue
		}
		if (agent == "codex" || agent == "muse") && name == "--all" {
			s.All = true
			continue
		}
		if agent == "codex" && name == "--include-non-interactive" {
			s.IncludeNonInteractive = true
			continue
		}
		if nativeContinueSelector(agent, name) {
			if agent == "agy" && inline {
				if value == "false" {
					continue
				}
				if value == "true" {
					inline = false
				}
			}
			if inline {
				return s, errors.New("continue does not take a value")
			}
			if err := selectKind(NativeResumeLatest, ""); err != nil {
				return s, err
			}
			s.remove[i] = true
			continue
		}
		selectorToken := a
		if agent == "agy" && strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && resumeform.Selector(agent, "-"+a) {
			selectorToken = "-" + a
		}
		if resumeform.Selector(agent, selectorToken) {
			id := resumeform.Extract(agent, []string{selectorToken})
			selectorName, selectorValue, selectorInline := strings.Cut(selectorToken, "=")
			if id == "" && selectorInline && nativeSpaceSelector(form, selectorName) {
				id = selectorValue
			}
			s.remove[i] = true
			if id == "" && !inline && nativeSpaceSelector(form, selectorToken) && i+1 < len(flags) && !strings.HasPrefix(flags[i+1], "-") {
				id = flags[i+1]
				i++
				s.remove[i] = true
			}
			kind := NativeResumeExplicit
			if id == "" {
				kind = NativeResumePicker
				if agent == "grok" {
					kind = NativeResumeLatest
				}
			}
			if err := selectKind(kind, id); err != nil {
				return s, err
			}
			continue
		}
		// Supported resume/new/fork forms above have explicit semantics. Any
		// remaining context flag from the Fresh authority needs an existing-
		// session mapping we cannot currently prove, even beside a fork/new ID.
		if freshAgentSpecs[agent].forbidsFlag(name) {
			return s, fmt.Errorf("unsupported existing-conversation selector %q; use an exact native ID or Fresh", a)
		}
		// A clustered short selector cannot hide from the same option-letter
		// authority used by Fresh validation. Value-taking letters end the cluster.
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && len(a) > 2 && !inline {
			spec := freshAgentSpecs[agent]
			for j := 1; j < len(a); j++ {
				letter := a[j : j+1]
				if strings.Contains(spec.valueShort, letter) {
					break
				}
				if strings.Contains(resumeform.ShortLetters(agent), letter) || nativeContinueSelector(agent, "-"+letter) {
					id := ""
					kind := NativeResumeLatest
					end := j + 1
					if strings.Contains(resumeform.ShortLetters(agent), letter) {
						id = a[j+1:]
						end = len(a)
						if id == "" && i+1 < len(flags) && !strings.HasPrefix(flags[i+1], "-") {
							id = flags[i+1]
							i++
							s.remove[i] = true
						}
						kind = NativeResumeExplicit
						if id == "" {
							kind = NativeResumePicker
						}
					}
					index := i
					if s.remove[i] {
						index = i - 1
					}
					if kind == NativeResumeLatest {
						for k := j + 1; k < len(a); k++ {
							letter := a[k : k+1]
							if strings.Contains(spec.valueShort, letter) {
								break
							}
							if strings.Contains(resumeform.ShortLetters(agent), letter) || nativeContinueSelector(agent, "-"+letter) {
								return s, errors.New("multiple native selectors in short cluster")
							}
						}
					}
					rest := "-" + a[1:j] + a[end:]
					if rest == "-" {
						s.remove[index] = true
					} else {
						s.replace[index] = rest
					}
					if err := selectKind(kind, id); err != nil {
						return s, err
					}
					break
				}
			}
		}
		if strings.HasPrefix(a, "-") {
			if !inline && freshValueOption(agent, name) && i+1 < len(flags) {
				if freshAgentSpecs[agent].takesOptionalValue(name) && strings.HasPrefix(flags[i+1], "-") {
					continue
				}
				if name == "--cwd" || name == "--cd" || name == "-C" || name == "--workspace" || (agent == "qoder" && name == "-w") {
					s.CWD = flags[i+1]
				}
				i++
				if freshVariadicOption(agent, name) {
					for i+1 < len(flags) && !strings.HasPrefix(flags[i+1], "-") {
						i++
					}
				}
			} else if inline && (name == "--cwd" || name == "--cd" || name == "--workspace") {
				s.CWD = value
			}
			continue
		}
		if commandSeen && s.command >= 0 && s.positional < 0 && !last {
			s.positional = i
			s.NativeID = a
			s.Kind = NativeResumeExplicit
			s.remove[i] = true
			continue
		}
		if !commandSeen {
			if agent == "codex" && !execSeen && (a == "exec" || a == "e") {
				execSeen = true
				continue
			}
			commandSeen = true
			if (agent == "codex" || agent == "muse") && (a == "resume" || (agent == "codex" && a == "fork")) {
				s.command = i
				s.remove[i] = true
				if a == "fork" {
					fork = true
				}
				id := ""
				if i+1 < len(flags) && !strings.HasPrefix(flags[i+1], "-") {
					id = flags[i+1]
					i++
					s.positional = i
					s.remove[i] = true
				}
				kind := NativeResumePicker
				if id != "" {
					kind = NativeResumeExplicit
				}
				if err := selectKind(kind, id); err != nil {
					return s, err
				}
				continue
			}
			// Reuse the Fresh command authority rather than restating native
			// attach/respawn spellings in a second classifier.
			if err := ValidateFreshAgentArgs(agent, []string{a}); err != nil {
				return s, fmt.Errorf("unsupported existing-conversation command: %w", err)
			}
		}
	}
	if last {
		if s.command < 0 {
			return s, errors.New("--last requires resume or fork")
		}
		// Undo a positional classified as ID: with --last it is the native prompt.
		if s.positional >= 0 {
			delete(s.remove, s.positional)
			s.positional = -1
		}
		s.NativeID = ""
		s.Kind = NativeResumeLatest
	}
	if fork {
		s.Kind = NativeResumeFork
	} else if newID {
		if selected {
			return s, errors.New("new session ID with resume requires fork")
		}
		s.Kind = NativeResumeNew
	}
	return s, nil
}

func nativeSpaceSelector(form resumeform.Form, s string) bool {
	for _, v := range form.Space {
		if v == s {
			return true
		}
	}
	return false
}
func nativeContinueSelector(agent, s string) bool {
	form := resumeform.Forms()[agent]
	for _, v := range form.Continue {
		if v == s {
			return true
		}
	}
	spec := freshAgentSpecs[agent]
	return (s == "--continue" && spec.forbidsFlag(s)) || (s == "-c" && strings.Contains(spec.forbiddenShort, "c")) || (agent == "agy" && (s == "-c" || s == "-continue"))
}

// PinNativeResumeSelector changes only this invocation's parsed resume tokens.
// It neither selects a target nor edits persisted launch preferences.
func PinNativeResumeSelector(s NativeResumeSelector, nativeID string) ([]string, error) {
	if s.Kind != NativeResumeExplicit && s.Kind != NativeResumeLatest && s.Kind != NativeResumePicker {
		return nil, errors.New("native selector is not a resume")
	}
	if nativeID == "" || strings.HasPrefix(nativeID, "-") || strings.ContainsAny(nativeID, "\x00\r\n") {
		return nil, errors.New("invalid canonical native ID")
	}
	// Reparse to keep exported copies from changing the private rewrite plan.
	parsed, err := ParseNativeResumeSelector(s.Agent, s.Args)
	if err != nil {
		return nil, err
	}
	if parsed.Kind != s.Kind || parsed.NativeID != s.NativeID {
		return nil, fmt.Errorf("native selector changed before pinning")
	}
	var out []string
	// Preserve a command's positional ID after -- in place. Moving it would
	// change the native delimiter/prompt grammar and can duplicate the ID.
	postDelimiterID := parsed.delimiter >= 0 && parsed.positional > parsed.delimiter
	inserted := false
	for i, a := range s.Args {
		if parsed.command < 0 && i == parsed.delimiter {
			out = append(out, resumeToken(s.Agent, nativeID)...)
			inserted = true
		}
		if i == parsed.command {
			out = append(out, "resume")
			if !postDelimiterID {
				out = append(out, nativeID)
			}
		}
		if postDelimiterID && i == parsed.positional {
			out = append(out, nativeID)
			continue
		}
		if parsed.remove[i] {
			continue
		}
		if replacement, ok := parsed.replace[i]; ok {
			a = replacement
		}
		out = append(out, a)
	}
	if parsed.command < 0 && !inserted {
		out = append(out, resumeToken(s.Agent, nativeID)...)
	}
	return out, nil
}

// CodexIndirectResumeCompatibilityNotice discloses the operator-approved
// temporary unreserved path without changing the requested native invocation.
const CodexIndirectResumeCompatibilityNotice = "Codex resume selector passed through unchanged; concurrent native resume is not guarded for this invocation."

// CodexIndirectResumeCompatibility is the single shared policy used before
// Pair reservation and at the wrapper boundary. Full native UUIDs remain
// guarded even when no inventory artifact exists for that ID.
func CodexIndirectResumeCompatibility(s NativeResumeSelector) bool {
	if s.Agent != "codex" {
		return false
	}
	switch s.Kind {
	case NativeResumeLatest, NativeResumePicker:
		return true
	case NativeResumeExplicit:
		return !sessioninventory.IsCanonicalCodexSessionID(s.NativeID)
	default:
		return false
	}
}
