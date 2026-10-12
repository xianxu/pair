package webcast

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestFindChrome(t *testing.T) {
	const appChrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	const userChromium = "/home/op/Applications/Chromium.app/Contents/MacOS/Chromium"
	cases := []struct {
		name   string
		env    map[string]string
		files  []string
		onPath map[string]string
		want   string
		err    string
	}{
		{name: "env wins", env: map[string]string{ChromeEnv: "/opt/c"}, files: []string{"/opt/c", appChrome}, want: "/opt/c"},
		{name: "env typo refuses", env: map[string]string{ChromeEnv: "/opt/typo"}, files: []string{appChrome}, err: ChromeEnv},
		{name: "system bundle", files: []string{appChrome, userChromium}, want: appChrome},
		{name: "user bundle", env: map[string]string{"HOME": "/home/op"}, files: []string{userChromium}, want: userChromium},
		{name: "path", onPath: map[string]string{"chromium": "/usr/bin/chromium"}, want: "/usr/bin/chromium"},
		{name: "none", err: ChromeEnv},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(k string) string { return tc.env[k] }
			exists := func(p string) bool {
				for _, f := range tc.files {
					if f == p {
						return true
					}
				}
				return false
			}
			lookPath := func(n string) (string, error) {
				if p, ok := tc.onPath[n]; ok {
					return p, nil
				}
				return "", exec.ErrNotFound
			}
			got, err := FindChrome(getenv, lookPath, exists)
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("got %q, %v; want error naming %s", got, err, tc.err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
	if _, err := FindChrome(func(string) string { return "" }, func(string) (string, error) { return "", exec.ErrNotFound }, func(string) bool { return false }); !errors.Is(err, ErrNoChrome) {
		t.Fatalf("none: %v", err)
	}
}
