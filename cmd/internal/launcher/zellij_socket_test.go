package launcher

import "testing"

func TestResolveZellijSocketRoot(t *testing.T) {
	for _, tc := range []struct {
		name, explicit, temp, want string
		uid                        int
		bad                        bool
	}{
		{name: "implicit", temp: "/private/tmp/original", uid: 501, want: "/private/tmp/original/zellij-501"},
		{name: "custom", explicit: "/short/custom", temp: "/ram/slot", uid: 501, want: "/short/custom"},
		{name: "preserve spelling", explicit: "/short/link/../socket/", temp: "/ram/slot", uid: 501, want: "/short/link/../socket/"},
		{name: "relative explicit", explicit: "socket", temp: "/tmp", uid: 501, bad: true},
		{name: "relative temp", temp: "tmp", uid: 501, bad: true},
		{name: "empty temp", uid: 501, bad: true},
		{name: "invalid explicit", explicit: "/tmp/socket\x00suffix", temp: "/tmp", uid: 501, bad: true},
		{name: "invalid temp", temp: "/tmp/\x00", uid: 501, bad: true},
		{name: "invalid uid", temp: "/tmp", uid: -1, bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveZellijSocketRoot(tc.explicit, tc.temp, "", "darwin", tc.uid)
			if (err != nil) != tc.bad || got != tc.want {
				t.Fatalf("ResolveZellijSocketRoot() = %q, %v; want %q, bad=%v", got, err, tc.want, tc.bad)
			}
		})
	}
}

func TestPinnedZellijSocketRootSurvivesSlotTempOverlay(t *testing.T) {
	for _, explicit := range []string{"", "/short/custom-sockets"} {
		root, err := ResolveZellijSocketRoot(explicit, "/original/tmp", "", "darwin", 501)
		if err != nil {
			t.Fatal(err)
		}
		childRoot, err := ResolveZellijSocketRoot(root, "/ram/slot/tmp", "", "darwin", 501)
		if err != nil || childRoot != root {
			t.Fatalf("pinned namespace moved under temp overlay: original=%q child=%q err=%v", root, childRoot, err)
		}
	}
}

func TestResolveZellijSocketRootMatchesNativeLinuxRuntimeDirectory(t *testing.T) {
	for _, tc := range []struct{ name, explicit, runtimeDir, platform, want string }{
		{"Linux XDG", "", "/run/user/501", "linux", "/run/user/501/zellij"},
		{"Linux absent XDG", "", "", "linux", "/original/tmp/zellij-501"},
		{"Linux relative XDG", "", "relative-runtime", "linux", "/original/tmp/zellij-501"},
		{"Linux short alias spelling", "", "/short/link/../runtime", "linux", "/short/link/../runtime/zellij"},
		{"explicit overrides Linux XDG", "/custom/link/", "/run/user/501", "linux", "/custom/link/"},
		{"Darwin ignores XDG", "", "/run/user/501", "darwin", "/original/tmp/zellij-501"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := ResolveZellijSocketRoot(tc.explicit, "/original/tmp", tc.runtimeDir, tc.platform, 501)
			if err != nil || root != tc.want {
				t.Fatalf("root=%q err=%v want=%q", root, err, tc.want)
			}
			child, err := ResolveZellijSocketRoot(root, "/ram/slot/tmp", "/changed/runtime", tc.platform, 501)
			if err != nil || child != root {
				t.Fatalf("pinned root moved: %q -> %q err=%v", root, child, err)
			}
		})
	}
}
