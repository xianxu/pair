package titlefmt

import "testing"

// Folds in every case the two former implementations carried —
// launcher.TestTildeAbbrev and titlepoller.TestAbbrevCwd — so consolidating
// loses no coverage. The sibling-path cases are the interesting ones: a naive
// strings.HasPrefix(path, home) without the trailing slash mangles
// "/home/x-other" into "~-other".
func TestTildeAbbrev(t *testing.T) {
	cases := []struct{ path, home, want string }{
		{"/home/x", "/home/x", "~"},
		{"/home/x/repo", "/home/x", "~/repo"},
		{"/Users/x/a/b", "/Users/x", "~/a/b"},
		{"/home/x-other", "/home/x", "/home/x-other"}, // sibling not mangled
		{"/Users/xyz", "/Users/x", "/Users/xyz"},      // not under $HOME/
		{"/tmp/z", "/home/x", "/tmp/z"},
		{"/tmp/z", "", "/tmp/z"}, // no home → unchanged
		{"", "/home/x", ""},
	}
	for _, c := range cases {
		if got := TildeAbbrev(c.path, c.home); got != c.want {
			t.Errorf("TildeAbbrev(%q,%q) = %q, want %q", c.path, c.home, got, c.want)
		}
	}
}

// The adversarial input for a prefix builder is degenerate, not long: the risk
// is emitting a dangling " · " with nothing in front of it, or a stray bracket.
func TestPaneTitlePrefix(t *testing.T) {
	cases := []struct{ cwd, tag, want string }{
		{"~/workspace/pair", "work", "~/workspace/pair [work] · "},
		{"~/workspace/pair", "", "~/workspace/pair · "},
		{"", "work", "[work] · "},
		{"", "", ""}, // nothing to say ⇒ no bare separator
		{"~", "a", "~ [a] · "},
	}
	for _, c := range cases {
		if got := PaneTitlePrefix(c.cwd, c.tag); got != c.want {
			t.Errorf("PaneTitlePrefix(%q,%q) = %q, want %q", c.cwd, c.tag, got, c.want)
		}
	}
}

// EmojiTitle had no test at all before #129 (the package reported "no test
// files"), and #130 plans to change what feeds it — pin the current contract
// first so that change is a visible diff rather than a silent one.
func TestEmojiTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"pair", "pair"},                   // single word: untouched
		{"brain", "brain"},                 // same
		{"pair-brain-book", "♋-🧠-📗"},       // compound: every known token maps
		{"pair-work", "♋-work"},            // unknown token left alone
		{"📁pair", "📁pair"},                 // no hyphen ⇒ verbatim (#130's shape)
		{"📁pair-1", "📁pair-1"},             // "📁pair" is not the "pair" key
		{"", ""},
	}
	for _, c := range cases {
		if got := EmojiTitle(c.in); got != c.want {
			t.Errorf("EmojiTitle(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
