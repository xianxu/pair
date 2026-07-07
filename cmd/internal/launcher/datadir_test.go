package launcher

import (
	"path/filepath"
	"testing"
)

func TestResolveDataDir(t *testing.T) {
	for _, tc := range []struct {
		name string
		home string
		xdg  string
		want string
	}{
		{name: "xdg", home: "/home/me", xdg: "/tmp/xdg", want: "/tmp/xdg/pair"},
		{name: "home", home: "/home/me", want: "/home/me/.local/share/pair"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveDataDir(tc.home, tc.xdg); got != tc.want {
				t.Fatalf("ResolveDataDir(%q, %q) = %q, want %q", tc.home, tc.xdg, got, tc.want)
			}
		})
	}
}

func TestScopedLaunchDataDir(t *testing.T) {
	global := "/home/me/.local/share/pair"
	got := ScopedLaunchDataDir(global, "/work/pair")
	scope := mustScope(t, "/work/pair")
	want := filepath.Join(global, "repos", scope.Key)
	if got != want {
		t.Fatalf("ScopedLaunchDataDir = %q, want %q", got, want)
	}
}
