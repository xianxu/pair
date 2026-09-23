package couchcore

import (
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestParseProvisionRequest(t *testing.T) {
	for _, slot := range []string{"1", "20"} {
		r, err := ParseProvisionRequest("/fleet/repo with spaces", slot, "upstream")
		if err != nil || r.Slot <= 0 || r.Remote != "upstream" {
			t.Fatalf("%+v %v", r, err)
		}
	}
	for _, slot := range []string{"", "0", "01", "+1", "-1", " 1", "1\n", "99999999999999999999999999"} {
		if _, err := ParseProvisionRequest("/repo", slot, ""); err == nil {
			t.Errorf("accepted slot %q", slot)
		}
	}
	for _, remote := range []string{"-bad", "bad remote", "bad\n", "bad\x00", "bad\u007f"} {
		if _, err := ParseProvisionRequest("/repo", "1", remote); err == nil {
			t.Errorf("accepted remote %q", remote)
		}
	}
	for _, path := range []string{"", "  ", "/bad\x00"} {
		if _, err := ParseProvisionRequest(path, "1", ""); err == nil {
			t.Errorf("accepted path %q", path)
		}
	}
}

func FuzzProvisionRequest(f *testing.F) {
	for _, seed := range [][3]string{{"/fleet/repo", "1", ""}, {"/fleet/repo with spaces", "24", "upstream"}, {".", "2", "origin"}, {"", "1", "origin"}, {"/repo", "01", "origin"}, {"/repo", "+1", "origin"}, {"/repo", "1", "bad\nremote"}, {"/repo", "999999999999999999999999999", "origin"}} {
		f.Add(seed[0], seed[1], seed[2])
	}
	f.Fuzz(func(t *testing.T, path, slot, remote string) {
		got, err := ParseProvisionRequest(path, slot, remote)
		if err != nil {
			return
		}
		if got.Slot <= 0 || strconv.Itoa(got.Slot) != slot || got.Path != path || got.Remote != remote || got.Progress != nil {
			t.Fatalf("parser changed accepted request: %#v", got)
		}
		if strings.TrimSpace(got.Path) == "" || strings.ContainsRune(got.Path, 0) || !utf8.ValidString(got.Path) {
			t.Fatalf("accepted invalid path %q", got.Path)
		}
		if strings.HasPrefix(got.Remote, "-") || !utf8.ValidString(got.Remote) {
			t.Fatalf("accepted invalid remote %q", got.Remote)
		}
		for _, r := range got.Remote {
			if unicode.IsControl(r) || unicode.IsSpace(r) {
				t.Fatalf("accepted remote control/whitespace %q", got.Remote)
			}
		}
		again, err := ParseProvisionRequest(got.Path, strconv.Itoa(got.Slot), got.Remote)
		if err != nil || again != got {
			t.Fatalf("request round-trip changed: %#v -> %#v, %v", got, again, err)
		}
	})
}
