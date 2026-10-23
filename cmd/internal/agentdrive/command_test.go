package agentdrive

import (
	"regexp"
	"testing"
	"unicode"
	"unicode/utf8"
)

var slashCommandOracle = regexp.MustCompile(`^/[A-Za-z][A-Za-z0-9_-]*([ \t].*)?$`)

func TestSlashCommand(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "bare command", text: "/help", want: true},
		{name: "command with underscore and hyphen", text: "/do_this-now", want: true},
		{name: "arguments after space", text: "/search query here", want: true},
		{name: "arguments after tab", text: "/search\tquery", want: true},
		{name: "unicode arguments", text: "/search café 猫", want: true},
		{name: "empty", text: "", want: false},
		{name: "leading whitespace", text: " /help", want: false},
		{name: "slash only", text: "/", want: false},
		{name: "space without command", text: "/ help", want: false},
		{name: "path in command token", text: "/foo/bar", want: false},
		{name: "non ascii command head", text: "/écho", want: false},
		{name: "punctuation command head", text: "/!help", want: false},
		{name: "at command head", text: "/@help", want: false},
		{name: "newline", text: "/help\nnext", want: false},
		{name: "carriage return", text: "/help\rnext", want: false},
		{name: "vertical tab in arguments", text: "/help\varg", want: false},
		{name: "nul in arguments", text: "/help\x00arg", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SlashCommand(tt.text); got != tt.want {
				t.Errorf("SlashCommand(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func FuzzSlashCommand(f *testing.F) {
	for _, seed := range []string{
		"/help", "/search hello", "/search\t猫", "/", "/ help", " /help",
		"/foo/bar", "/écho", "/help\nnext", "/help\rnext", "/help\x00arg",
		"/ok\vno", "/ok\x7fno",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		want := utf8.ValidString(text) && slashCommandOracle.MatchString(text)
		argsAt := len(text)
		if want {
			for i := 1; i < len(text); i++ {
				if text[i] == ' ' || text[i] == '\t' {
					argsAt = i + 1
					break
				}
			}
		}
		for _, r := range text[argsAt:] {
			if unicode.IsControl(r) {
				want = false
				break
			}
		}
		if got := SlashCommand(text); got != want {
			t.Fatalf("SlashCommand(%q) = %v, oracle wants %v", text, got, want)
		}
	})
}
