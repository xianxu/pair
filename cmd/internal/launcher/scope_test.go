package launcher

import (
	"strings"
	"testing"
)

func TestRepoScopeUsesAbsolutePathForHiddenKey(t *testing.T) {
	a, err := ResolveRepoScope("/Users/alice/work/pair")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}
	b, err := ResolveRepoScope("/tmp/other/pair")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}

	if a.DisplayName != "pair" || b.DisplayName != "pair" {
		t.Fatalf("display names = %q, %q; want pair, pair", a.DisplayName, b.DisplayName)
	}
	if a.Key == "" || b.Key == "" {
		t.Fatalf("scope keys must be non-empty: %+v %+v", a, b)
	}
	if a.Key == b.Key {
		t.Fatalf("same basename at different paths got same key %q", a.Key)
	}
}

func TestRepoScopeCleansPathBeforeHashing(t *testing.T) {
	a, err := ResolveRepoScope("/Users/alice/work/pair")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}
	b, err := ResolveRepoScope("/Users/alice/work/../work/pair/.")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}

	if a.Root != b.Root {
		t.Fatalf("cleaned roots differ: %q vs %q", a.Root, b.Root)
	}
	if a.Key != b.Key {
		t.Fatalf("same cleaned path got different keys: %q vs %q", a.Key, b.Key)
	}
}

func TestNormalizeDisplayComponent(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{in: "pair", want: "pair"},
		{in: "parley.nvim", want: "parley_nvim"},
		{in: "hello world", want: "hello_world"},
		{in: "!!!", want: "___"},
		{in: "", want: "pair"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			if got := NormalizeDisplayComponent(tc.in); got != tc.want {
				t.Fatalf("NormalizeDisplayComponent(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRepoScopeDoesNotExposeHiddenKeyInDisplayFields(t *testing.T) {
	scope, err := ResolveRepoScope("/Users/alice/work/parley.nvim")
	if err != nil {
		t.Fatalf("ResolveRepoScope returned error: %v", err)
	}

	if strings.Contains(scope.DisplayName, scope.Key) {
		t.Fatalf("display name %q exposed hidden key %q", scope.DisplayName, scope.Key)
	}
	if strings.Contains(NormalizeDisplayComponent(scope.DisplayName), scope.Key) {
		t.Fatalf("normalized display component exposed hidden key %q", scope.Key)
	}
}
