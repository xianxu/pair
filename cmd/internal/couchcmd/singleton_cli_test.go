package couchcmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAdoptionCLI(t *testing.T) {
	for i, args := range [][]string{
		{"--adopt-store", "/data/couch"},
		{"--adopt-store", "/data/couch", "--pair-data", "/data/pair", "--identity-dir", "/data/identity", "--legacy-store", "/old/one", "--legacy-store", "/old/two", "--exclude-store", "/retired/one", "--exclude-store", "/retired/two", "--apply", strings.Repeat("a1", 32)},
	} {
		got, err := ParseCLI(args, nil)
		if err != nil {
			t.Fatalf("ParseCLI(%q): %v", args, err)
		}
		want := adoptionArgs{StoreDir: "/data/couch"}
		if i == 1 {
			want.PairDataDir, want.IdentityDir, want.Expect = "/data/pair", "/data/identity", strings.Repeat("a1", 32)
			want.Stores, want.Exclude = []string{"/old/one", "/old/two"}, []string{"/retired/one", "/retired/two"}
		}
		if got.kind != cliAdopt || got.layout != "" || !reflect.DeepEqual(got.adoption, want) {
			t.Errorf("ParseCLI(%q) = %+v; want adoption %+v without layout", args, got, want)
		}
	}
}

func TestParseAdoptionCLIRejectsInvalidArguments(t *testing.T) {
	cases := [][]string{
		{"--adopt-store"}, {"--adopt-store", ""}, {"--adopt-store", "relative"},
		{"--adopt-store", "/store", "--layout2"}, {"--layout2", "--adopt-store", "/store"},
		{"--adopt-store", "/store", "--unknown", "/value"},
		{"--adopt-store", "/store", "extra"},
	}
	for _, flag := range []string{"--adopt-store", "--pair-data", "--identity-dir", "--legacy-store", "--exclude-store"} {
		for _, path := range []string{"", "relative", "/bad\x00path", "/bad\npath", "/bad\xffpath"} {
			cases = append(cases, []string{"--adopt-store", "/store", flag, path})
		}
		cases = append(cases, []string{"--adopt-store", "/store", flag})
	}
	for _, flag := range []string{"--adopt-store", "--pair-data", "--identity-dir"} {
		cases = append(cases, []string{"--adopt-store", "/store", flag, "/one", flag, "/two"})
	}
	for _, digest := range []string{"", "abc", strings.Repeat("A", 64), strings.Repeat("g", 64), strings.Repeat("a", 63), strings.Repeat("a", 65)} {
		cases = append(cases, []string{"--adopt-store", "/store", "--apply", digest})
	}
	cases = append(cases, []string{"--adopt-store", "/store", "--apply", strings.Repeat("a", 64), "--apply", strings.Repeat("a", 64)})
	for _, args := range cases {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Errorf("accepted invalid argv %q", args)
		}
	}
}
