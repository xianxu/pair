package launcher

import "testing"

// The 📁 session-name scheme (#130). These cover the pure core: the ownership
// predicate that keeps pair off foreign sessions, and the composer implementing
// spec rules 1-4.

func TestIsPairSessionName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"pair-x", true},             // legacy scheme
		{"pair-pair-pair", true},     // legacy, fully spelled
		{"📁x", true},                 // new scheme
		{"📁parley-nvim", true},       // new scheme with residual
		{"fabulous-aardvark", false}, // a real foreign session seen in the global list
		{"", false},
		{"repair-shop", false}, // must not match `pair-` mid-word
		{"folder📁", false},     // prefix only, not substring
	}
	for _, c := range cases {
		if got := isPairSessionName(c.name); got != c.want {
			t.Errorf("isPairSessionName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

// A foreign session must never be classified as pair's. This is the one
// predicate mistake that destroys someone else's state rather than confusing
// pair: SessionBlocksReuse force-deletes an EXITED session it believes is ours.
func TestIsPairSessionNameRejectsForeign(t *testing.T) {
	foreign := []string{
		"fabulous-aardvark",
		"nimble-narwhal",
		"main",
		"work",
		"scratch",
	}
	for _, name := range foreign {
		if isPairSessionName(name) {
			t.Errorf("isPairSessionName(%q) = true; a foreign session must never be a pair session", name)
		}
	}
}

func TestComposeSessionName(t *testing.T) {
	cases := []struct {
		what string
		repo string
		tag  string
		want string
	}{
		// The four worked examples from the spec.
		{"tag equals repo", "pair", "pair", "📁pair"},
		{"tag prefixed by repo", "pair", "pair-1", "📁pair-1"},
		{"dotted repo, underscored tag", "parley.nvim", "parley_nvim", "📁parley-nvim"},
		{"tag unrelated to repo", "parley.nvim", "work", "📁parley-work"},

		// Where the rules interact.
		{"repo token appears non-leading in tag", "pair", "my-pair", "📁pair-my-pair"},
		{"case differs, so no redundancy is dropped", "Pair", "pair", "📁Pair-pair"},
		{"empty tag falls back to the normalizer's default", "pair", "", "📁pair"},
		{"repo with no alphanumerics", "...", "work", "📁pair-work"},
		{"multi-token residual is fully preserved", "pair", "pair-feature-two", "📁pair-feature-two"},
		{"hyphenated repo keeps only its first token", "claude-code", "work", "📁claude-work"},
	}
	for _, c := range cases {
		scope := RepoScope{DisplayName: c.repo}
		if got := ComposeSessionName(scope, c.tag); got != c.want {
			t.Errorf("%s: ComposeSessionName(%q, %q) = %q, want %q", c.what, c.repo, c.tag, got, c.want)
		}
	}
}

// The headline outcomes the issue exists to produce, asserted directly so a
// regression names itself.
func TestComposeSessionNameRetiresTheRedundantSpellings(t *testing.T) {
	if got := ComposeSessionName(RepoScope{DisplayName: "pair"}, "pair"); got != "📁pair" {
		t.Errorf("pair-pair-pair should become 📁pair, got %q", got)
	}
	got := ComposeSessionName(RepoScope{DisplayName: "parley.nvim"}, "parley_nvim")
	if got != "📁parley-nvim" {
		t.Errorf("pair-parley_nvim-parley_nvim should become 📁parley-nvim untruncated, got %q", got)
	}
	// And it must fit: the whole point is that this no longer needs shortening.
	if len(got) > 24 {
		t.Errorf("%q is %d bytes; zellij's budget on this machine is 24", got, len(got))
	}
}

func TestSessionNameParts(t *testing.T) {
	repo, residual := sessionNameParts(RepoScope{DisplayName: "parley.nvim"}, "parley_nvim")
	if repo != "parley" {
		t.Errorf("repo = %q, want %q", repo, "parley")
	}
	if len(residual) != 1 || residual[0] != "nvim" {
		t.Errorf("residual = %v, want [nvim]", residual)
	}

	// Rule 4 drops only the LEADING matching run.
	_, residual = sessionNameParts(RepoScope{DisplayName: "pair"}, "pair-pair-x")
	if len(residual) != 2 || residual[0] != "pair" || residual[1] != "x" {
		t.Errorf("residual = %v, want [pair x] — only the leading run is dropped", residual)
	}
}
