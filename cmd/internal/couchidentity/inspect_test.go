package couchidentity

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectPreservesAllocationAndRefusesLostAuthority(t *testing.T) {
	s := fixture(t)
	alloc(t, s)
	hp := filepath.Join(s.HostDir, "couch-identities.json")
	lp := filepath.Join(s.StoreDir, "identities.json")
	before, _ := os.ReadFile(hp)
	local, _ := os.ReadFile(lp)
	got, e := s.Inspect()
	if e != nil || got.Local.LastN != 1 || len(got.Registrations) != 1 {
		t.Fatalf("%+v %v", got, e)
	}
	after, _ := os.ReadFile(hp)
	if !bytes.Equal(before, after) {
		t.Fatal("inspection allocated")
	}
	if e = os.Remove(lp); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Inspect(); e == nil {
		t.Fatal("accepted missing consumed state")
	}
	os.WriteFile(lp, local, 0600)
	os.Remove(hp)
	if _, e = s.Inspect(); e == nil {
		t.Fatal("accepted missing authority")
	}
}
func TestInspectAbsentDoesNotCreate(t *testing.T) {
	s := fixture(t)
	if _, e := s.Inspect(); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(s.HostDir); !os.IsNotExist(e) {
		t.Fatal(e)
	}
}
func TestInspectRejectsRegressedAndMalformedAuthority(t *testing.T) {
	for _, bad := range []string{`{"schema_version":1,"next_c":1,"stores":[]}`, `{"schema_version":1`, strings.Repeat(" ", 4<<20+1)} {
		s := fixture(t)
		alloc(t, s)
		p := filepath.Join(s.HostDir, "couch-identities.json")
		os.WriteFile(p, []byte(bad), 0600)
		if _, e := s.Inspect(); e == nil {
			t.Fatal("bad authority accepted")
		}
		after, _ := os.ReadFile(p)
		if string(after) != bad {
			t.Fatal("authority changed")
		}
	}
}
