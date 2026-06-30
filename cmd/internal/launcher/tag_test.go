package launcher

import "testing"

func TestNormalizeTag(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{name: "bare", raw: "demo", want: "demo"},
		{name: "prefixed", raw: "pair-demo", want: "demo"},
		{name: "underscore", raw: "pair-demo_2", want: "demo_2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeTag(tc.raw)
			if err != nil {
				t.Fatalf("NormalizeTag returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeTag(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestNormalizeTagRejectsInvalid(t *testing.T) {
	for _, raw := range []string{"", "pair-", "bad/slug", "has space"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := NormalizeTag(raw); err == nil {
				t.Fatalf("NormalizeTag(%q) returned nil error", raw)
			}
		})
	}
}

func TestDefaultTag(t *testing.T) {
	for _, tc := range []struct {
		cwd  string
		want string
	}{
		{cwd: "/Users/xianxu/workspace/pair", want: "pair"},
		{cwd: "/tmp/hello world", want: "hello_world"},
		{cwd: "/tmp/!!!", want: "___"},
		{cwd: "", want: "pair"},
	} {
		t.Run(tc.cwd, func(t *testing.T) {
			if got := DefaultTag(tc.cwd); got != tc.want {
				t.Fatalf("DefaultTag(%q) = %q, want %q", tc.cwd, got, tc.want)
			}
		})
	}
}
