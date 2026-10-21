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

func TestResolveZellijSocketRootPreservesEveryNativeRootSpelling(t *testing.T) {
	for _, tc := range []struct {
		name, explicit, temp, runtimeDir, platform, want string
		uid                                              int
		bad                                              bool
	}{
		{name: "explicit parent traversal", explicit: "/short/alias/../socket///", temp: "/tmp", runtimeDir: "/run/user/501", platform: "linux", uid: 501, want: "/short/alias/../socket///"},
		{name: "explicit double root", explicit: "//", temp: "/tmp", platform: "darwin", uid: 501, want: "//"},
		{name: "explicit bypasses unused fallback validation", explicit: "/socket", temp: "relative", platform: "darwin", uid: -1, want: "/socket"},
		{name: "runtime parent traversal", temp: "/tmp", runtimeDir: "/short/alias/../runtime///", platform: "linux", uid: 501, want: "/short/alias/../runtime///zellij"},
		{name: "runtime slash", temp: "/tmp", runtimeDir: "/", platform: "linux", uid: 501, want: "/zellij"},
		{name: "runtime double root", temp: "/tmp", runtimeDir: "//", platform: "linux", uid: 501, want: "//zellij"},
		{name: "runtime bypasses unused fallback validation", temp: "relative", runtimeDir: "/run/user/501/", platform: "linux", uid: -1, want: "/run/user/501/zellij"},
		{name: "temp parent traversal Darwin", temp: "/short/alias/../temp///", platform: "darwin", uid: 501, want: "/short/alias/../temp///zellij-501"},
		{name: "temp parent traversal Linux", temp: "/short/alias/../temp///", runtimeDir: "relative", platform: "linux", uid: 501, want: "/short/alias/../temp///zellij-501"},
		{name: "temp double root Darwin", temp: "//", platform: "darwin", uid: 501, want: "//zellij-501"},
		{name: "temp double root Linux", temp: "//", platform: "linux", uid: 501, want: "//zellij-501"},
		{name: "temp root", temp: "/", platform: "darwin", uid: 501, want: "/zellij-501"},
		{name: "invalid explicit", explicit: "relative", temp: "/tmp", platform: "linux", uid: 501, bad: true},
		{name: "invalid runtime", temp: "/tmp", runtimeDir: "/run/\x00user", platform: "linux", uid: 501, bad: true},
		{name: "invalid temp", temp: "/short/\x00temp", platform: "darwin", uid: 501, bad: true},
		{name: "invalid uid", temp: "/tmp", platform: "darwin", uid: -1, bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveZellijSocketRoot(tc.explicit, tc.temp, tc.runtimeDir, tc.platform, tc.uid)
			if (err != nil) != tc.bad || got != tc.want {
				t.Fatalf("got=%q err=%v want=%q bad=%v", got, err, tc.want, tc.bad)
			}
		})
	}
}

func TestAppendNativeSocketPathPreservesRootBytes(t *testing.T) {
	for _, tc := range []struct{ root, suffix, want string }{
		{"/socket", "contract_version_1/a", "/socket/contract_version_1/a"},
		{"/socket/", "contract_version_1/a", "/socket/contract_version_1/a"},
		{"/socket///", "contract_version_1/a", "/socket///contract_version_1/a"},
		{"/alias/../socket//", "zellij-501", "/alias/../socket//zellij-501"},
		{"/", "zellij", "/zellij"},
		{"//", "zellij", "//zellij"},
	} {
		if got := appendNativeSocketPath(tc.root, tc.suffix); got != tc.want {
			t.Fatalf("append(%q,%q)=%q want=%q", tc.root, tc.suffix, got, tc.want)
		}
	}
}
