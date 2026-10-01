package couchcore

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestResolveRepositoryName(t *testing.T) {
	repos := []RepositoryName{
		{Key: "/w/pair", Dir: "pair"},
		{Key: "/w/pair-tools", Dir: "pair-tools"},
		{Key: "/w/parley.nvim", Dir: "parley.nvim"},
		{Key: "/w/xianxu.dev", Dir: "xianxu.dev", Alias: "blog"},
		{Key: "/w/brain", Dir: "brain"},
		{Key: "/w/brainstorm", Dir: "brainstorm"},
	}
	for _, tc := range []struct {
		raw, key string
		match    RepositoryMatch
	}{
		{"pair", "/w/pair", RepositoryExact},
		{"pair-tools", "/w/pair-tools", RepositoryExact},
		{"parley", "/w/parley.nvim", RepositoryPrefix},
		{"parley.nvim", "/w/parley.nvim", RepositoryExact},
		{"blog", "/w/xianxu.dev", RepositoryExact},
		{"bl", "/w/xianxu.dev", RepositoryPrefix},
		{"xianxu", "/w/xianxu.dev", RepositoryPrefix},
		{"brain", "/w/brain", RepositoryExact},
	} {
		got, match, err := ResolveRepositoryName(tc.raw, repos)
		if err != nil || got.Key != tc.key || match != tc.match {
			t.Fatalf("%q: got %+v %v %v, want %s %v", tc.raw, got, match, err, tc.key, tc.match)
		}
	}
	_, match, err := ResolveRepositoryName("bra", repos)
	if !errors.Is(err, ErrRepositoryAmbiguous) || match != RepositoryNone || !strings.Contains(err.Error(), "brain") || !strings.Contains(err.Error(), "brainstorm") {
		t.Fatalf("ambiguous prefix: %v %v", match, err)
	}
	_, _, err = ResolveRepositoryName("zzz", repos)
	if !errors.Is(err, ErrRepositoryNotFound) || !strings.Contains(err.Error(), "xianxu.dev (blog)") {
		t.Fatalf("miss: %v", err)
	}
	for _, raw := range []string{"", "pair:1", "a/b", " pair"} {
		if _, _, err := ResolveRepositoryName(raw, repos); !errors.Is(err, ErrRepositoryNotFound) {
			t.Fatalf("invalid %q: %v", raw, err)
		}
	}
}

func TestResolveRepositoryNameExactAmbiguity(t *testing.T) {
	// Two checkouts named parley in different fleet roots are never silently picked.
	_, _, err := ResolveRepositoryName("parley", []RepositoryName{{Key: "/a/parley", Dir: "parley"}, {Key: "/b/parley", Dir: "parley"}})
	if !errors.Is(err, ErrRepositoryAmbiguous) {
		t.Fatalf("shared dir: %v", err)
	}
	// An alias equal to another repository's directory is ambiguous, not first-wins.
	_, _, err = ResolveRepositoryName("blog", []RepositoryName{{Key: "/w/blog", Dir: "blog"}, {Key: "/w/xianxu.dev", Dir: "xianxu.dev", Alias: "blog"}})
	if !errors.Is(err, ErrRepositoryAmbiguous) {
		t.Fatalf("alias vs dir: %v", err)
	}
	// Dir and alias of the SAME repository both matching a prefix count once.
	got, match, err := ResolveRepositoryName("x", []RepositoryName{{Key: "/w/xianxu.dev", Dir: "xianxu.dev", Alias: "xd"}})
	if err != nil || got.Key != "/w/xianxu.dev" || match != RepositoryPrefix {
		t.Fatalf("same-key prefix: %+v %v %v", got, match, err)
	}
	// Duplicate rows of one key (two live slots of a family) resolve exactly.
	got, _, err = ResolveRepositoryName("pair", []RepositoryName{{Key: "pair", Dir: "pair"}, {Key: "pair", Dir: "pair"}})
	if err != nil || got.Key != "pair" {
		t.Fatalf("duplicate key rows: %+v %v", got, err)
	}
}

func TestFormatRepositoryCandidatesBounded(t *testing.T) {
	var repos []RepositoryName
	for i := 0; i < 128; i++ {
		dir := fmt.Sprintf("%03d-%s", i, strings.Repeat("x", 200))
		repos = append(repos, RepositoryName{Key: "/w/" + dir, Dir: dir, Alias: strings.Repeat("a", 200)})
	}
	out := FormatRepositoryCandidates(repos)
	if len(out) > MaxRepositoryCandidateBytes || !strings.Contains(out, "more") || !strings.HasPrefix(out, "000-") {
		t.Fatalf("unbounded or unordered (%d bytes): %.80q", len(out), out)
	}
	if got := FormatRepositoryCandidates(nil); got != "none" {
		t.Fatalf("empty: %q", got)
	}
	small := FormatRepositoryCandidates([]RepositoryName{{Key: "b", Dir: "b"}, {Key: "a", Dir: "a", Alias: "al"}, {Key: "a", Dir: "a", Alias: "al"}})
	if small != "a (al), b" {
		t.Fatalf("small: %q", small)
	}
}

func TestValidateRepositoryAlias(t *testing.T) {
	repos := []RepositoryName{
		{Key: "/w/pair", Dir: "pair"},
		{Key: "/w/xianxu.dev", Dir: "xianxu.dev", Alias: "blog"},
		{Key: "/w/notes", Dir: "notes", Alias: "n"},
	}
	for _, tc := range []struct {
		alias, self string
		ok          bool
	}{
		{"blog", "/w/xianxu.dev", true}, // re-setting its own alias
		{"web", "/w/xianxu.dev", true},
		{"", "/w/xianxu.dev", true}, // clear
		{"pair", "/w/xianxu.dev", false},
		{"n", "/w/xianxu.dev", false},
		{"blog", "/w/notes", false},
		{"xianxu.dev", "/w/xianxu.dev", false},
		{"a:b", "/w/notes", false},
		{"a/b", "/w/notes", false},
		{"..", "/w/notes", false},
		{"my blog", "/w/notes", false},
		{"web", "/w/missing", false},
	} {
		err := ValidateRepositoryAlias(tc.alias, tc.self, repos)
		if (err == nil) != tc.ok {
			t.Fatalf("%q for %s: %v", tc.alias, tc.self, err)
		}
	}
}
