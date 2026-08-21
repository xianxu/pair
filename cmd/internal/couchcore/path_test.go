package couchcore

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePathCollapsesSpellings(t *testing.T) {
	base := "/Users/x/workspace/pair"
	want := NormalizePath(base)
	for _, s := range []string{
		base,
		base + "/",
		base + "/.",
		base + "/cmd/..",
		"/Users/x/workspace//pair",
		"/Users/x/workspace/./pair",
	} {
		if got := NormalizePath(s); got != want {
			t.Errorf("NormalizePath(%q) = %q, want %q", s, got, want)
		}
	}
}

func TestNormalizePathPreservesCase(t *testing.T) {
	// Case is preserved on purpose: pair feeds ResolveRepoScope a raw path, so
	// a folded path would derive a different scope key for the same tree.
	// Folding belongs to the lookup key only.
	if got := NormalizePath("/Users/x/KBench"); !strings.HasSuffix(got, "KBench") {
		t.Fatalf("NormalizePath folded case: %q", got)
	}
}

func TestNormalizePathIsAbsolute(t *testing.T) {
	if got := NormalizePath("relative/path"); !filepath.IsAbs(got) {
		t.Fatalf("non-absolute %q", got)
	}
}

// foldWith is tested directly so both directions are asserted on every
// platform. FoldKey itself can only exercise the host's branch.
func TestFoldWithBothDirections(t *testing.T) {
	const upper, lower = "/Users/x/Pair", "/users/x/pair"
	if got := foldWith(upper, true); got != lower {
		t.Errorf("foldWith(insensitive) = %q, want %q", got, lower)
	}
	if got := foldWith(upper, false); got != upper {
		t.Errorf("foldWith(sensitive) = %q, want it unchanged", got)
	}
}

func TestFoldKeyUsesPlatformSensitivity(t *testing.T) {
	got := FoldKey("/Users/x/Pair")
	want := foldWith("/Users/x/Pair", caseInsensitiveFS())
	if got != want {
		t.Fatalf("FoldKey = %q, want %q", got, want)
	}
}
