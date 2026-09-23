package couchcore

import (
	"strconv"
	"testing"
)

func TestParseWorkspaceReference(t *testing.T) {
	for _, raw := range []string{":0", ":1", ":29", "pair:0", "my-repo:32", "my repo:2", "项目:1"} {
		ref, matched, err := ParseWorkspaceReference(raw)
		if err != nil || !matched || ref.String() != raw {
			t.Fatalf("%q: %+v %v %v", raw, ref, matched, err)
		}
	}
	for _, raw := range []string{"pair", "opaque-tag", "/tmp/repo:1", "./repo:1", "../repo", "https://host/repo", ""} {
		_, matched, err := ParseWorkspaceReference(raw)
		if matched || err != nil {
			t.Fatalf("ordinary %q: matched %v, %v", raw, matched, err)
		}
	}
	for _, raw := range []string{":", ":01", ":-1", ":+1", ": 1", " :1", ":1 ", "pair:01", "pair:", "pair:x", "pair:1:2", ":99999999999999999999999999", ":1/x", "repo :1", "..:1"} {
		_, matched, err := ParseWorkspaceReference(raw)
		if !matched || err == nil {
			t.Fatalf("invalid %q: matched %v, %v", raw, matched, err)
		}
	}
	raw := ":" + strconv.Itoa(int(^uint(0)>>1))
	if _, matched, err := ParseWorkspaceReference(raw); !matched || err != nil {
		t.Fatalf("maximum: %v %v", matched, err)
	}
}

func FuzzWorkspaceReference(f *testing.F) {
	for _, raw := range []string{":0", "pair:1", ":01", ":999999999999999999999999", "../repo", ":1/x"} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		ref, matched, err := ParseWorkspaceReference(raw)
		if err == nil && matched {
			if ref.Number < 0 || ref.String() != raw {
				t.Fatalf("accepted noncanonical %q: %+v", raw, ref)
			}
			again, recognized, e := ParseWorkspaceReference(ref.String())
			if e != nil || !recognized || again != ref {
				t.Fatalf("roundtrip failed: %+v", ref)
			}
		}
	})
}
