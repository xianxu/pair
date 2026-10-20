package couchscratch

import "testing"

func TestScratchAllowed(t *testing.T) {
	const root = "/Volumes/TestScratch"
	tests := []struct {
		name string
		root string
		p    SandboxPolicy
		want bool
	}{
		{"unknown", root, SandboxPolicy{}, false},
		{"unknown disabled with grant", root, SandboxPolicy{Allowed: []string{root}}, false},
		{"disabled", root, SandboxPolicy{Known: true}, true},
		{"disabled ignores deny", root, SandboxPolicy{Known: true, Denied: []string{root}}, true},
		{"enabled no grants", root, SandboxPolicy{Known: true, Enabled: true}, false},
		{"exact grant", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{root}}, true},
		{"parent grant", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{"/Volumes"}}, true},
		{"filesystem grant", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{"/"}}, true},
		{"sibling prefix grant", root + "-other", SandboxPolicy{Known: true, Enabled: true, Allowed: []string{root}}, false},
		{"longer sibling grant", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{root + "-other"}}, false},
		{"descendant grant", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{root + "/tmp"}}, false},
		{"exact deny wins", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{root}, Denied: []string{root}}, false},
		{"parent deny wins", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{root}, Denied: []string{"/Volumes"}}, false},
		{"descendant deny wins", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{root}, Denied: []string{root + "/private"}}, false},
		{"sibling deny does not overlap", root, SandboxPolicy{Known: true, Enabled: true, Allowed: []string{root}, Denied: []string{root + "-other"}}, true},
		{"trailing separators", root + "/", SandboxPolicy{Known: true, Enabled: true, Allowed: []string{"/Volumes/"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScratchAllowed(tt.root, tt.p); got != tt.want {
				t.Fatalf("ScratchAllowed(%q, %+v) = %v, want %v", tt.root, tt.p, got, tt.want)
			}
		})
	}
}

func TestScratchAllowedRefusesMalformedPaths(t *testing.T) {
	const root = "/Volumes/TestScratch"
	for _, path := range []string{"", "relative", "~/scratch", "/Volumes/../TestScratch", "/Volumes/./TestScratch", "/Volumes//TestScratch", "/Volumes/*", "/Volumes/Te?tScratch", "/Volumes/[T]estScratch", "/Volumes/Test\x00Scratch"} {
		t.Run(path, func(t *testing.T) {
			if ScratchAllowed(path, SandboxPolicy{Known: true}) {
				t.Errorf("disabled policy allowed malformed root %q", path)
			}
			for _, enabled := range []bool{false, true} {
				if ScratchAllowed(root, SandboxPolicy{Known: true, Enabled: enabled, Allowed: []string{root, path}}) {
					t.Errorf("enabled=%v: malformed grant %q accepted", enabled, path)
				}
				if ScratchAllowed(root, SandboxPolicy{Known: true, Enabled: enabled, Allowed: []string{root}, Denied: []string{path}}) {
					t.Errorf("enabled=%v: malformed deny %q accepted", enabled, path)
				}
			}
		})
	}
}
