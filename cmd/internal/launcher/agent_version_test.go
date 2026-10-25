package launcher

import "testing"

func TestParseAgentVersion(t *testing.T) {
	for in, want := range map[string]string{
		"codex-cli 0.164.0\n":         "0.164.0",
		"0.164.0":                     "0.164.0",
		"codex-cli 0.164.0-alpha.1\n": "0.164.0-alpha.1",
		"":                            "",
		"codex-cli":                   "",
		"codex-cli 0.164\n":           "",
	} {
		if got := parseAgentVersion(in); got != want {
			t.Fatalf("parse(%q) = %q want %q", in, got, want)
		}
	}
}

func TestVersionAtLeast(t *testing.T) {
	min := [3]int{0, 164, 0}
	for version, want := range map[string]bool{
		"0.164.0": true, "0.164.1": true, "0.165.0": true, "1.0.0": true,
		"0.163.9": false, "0.163.0": false, "": false, "junk": false, "0.164.0-alpha.1": true,
	} {
		if got := versionAtLeast(version, min); got != want {
			t.Fatalf("versionAtLeast(%q) = %t want %t", version, got, want)
		}
	}
}
