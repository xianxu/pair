package launcher

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxPromptBytes bounds a launch prompt: one line pointing at durable state,
// never the state itself.
const MaxPromptBytes = 4096

// ValidatePrompt admits a launch prompt: one bounded line of text, no control
// characters, and not something an agent's parser could read as a flag.
func ValidatePrompt(prompt string) error {
	if prompt == "" || len(prompt) > MaxPromptBytes || !utf8.ValidString(prompt) || strings.IndexFunc(prompt, unicode.IsControl) >= 0 || strings.HasPrefix(prompt, "-") {
		return errors.New("invalid launch prompt: one line of at most 4096 bytes, not starting with '-'")
	}
	return nil
}

// PromptArgs places a fresh conversation's first message (#522) in the
// agent's own form, from each CLI's measured grammar: agy takes it as
// `-i PROMPT`; the others take a positional prompt. It goes FIRST, before the
// saved argv: a trailing positional is swallowed by any saved flag that takes
// several values (`--add-dir a b`), and nothing precedes the first argument
// to swallow it. The prompt never starts with '-' (ValidatePrompt), so it is
// never read as a flag. An empty prompt leaves argv as it is. The result is a
// new slice.
func PromptArgs(agent string, argv []string, prompt string) []string {
	if prompt == "" {
		return append([]string{}, argv...)
	}
	head := []string{prompt}
	if filepath.Base(agent) == "agy" {
		head = []string{"-i", prompt}
	}
	return append(head, argv...)
}
