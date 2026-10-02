package couchsingleton

import (
	"reflect"
	"testing"
)

func TestDecideAdoptionPermutations(t *testing.T) {
	d := Roots{"/pair/couch", "/pair", "/identity"}
	other := d
	other.StoreDir = "/other"
	for _, tc := range []struct {
		name, state, owner, problem string
		exclude, fail               bool
	}{
		{"empty", "empty", "", "", false, false}, {"sole", "populated", "", "", false, false}, {"unknown", "unknown", "", "", false, true}, {"excluded unknown", "unknown", "", "", true, true}, {"excluded live", "populated", "live", "", true, true}, {"excluded missing", "missing", "", "", true, false}, {"missing", "missing", "", "", false, true}, {"unreadable", "populated", "", "corrupt", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := Request{}
			if tc.exclude {
				q.Exclude = []string{other.StoreDir}
			}
			a := Candidate{Roots: d, State: "empty"}
			b := Candidate{Roots: other, State: tc.state, Owner: tc.owner, Problem: tc.problem, Registered: true}
			var first Selection
			for i, cs := range [][]Candidate{{a, b}, {b, a}} {
				got, e := DecideAdoption(q, d, cs)
				if (e != nil) != tc.fail {
					t.Fatal(got, e)
				}
				if i == 0 {
					first = got
				} else if !reflect.DeepEqual(first, got) {
					t.Fatal("order changed decision")
				}
			}
		})
	}
	a := Candidate{Roots: d, State: "populated"}
	b := Candidate{Roots: other, State: "populated"}
	if _, e := DecideAdoption(Request{}, d, []Candidate{a, b}); e == nil {
		t.Fatal("ambiguous admitted")
	}
	if _, e := DecideAdoption(Request{}, d, []Candidate{a, a}); e == nil {
		t.Fatal("duplicate admitted")
	}
}
func TestDecideAdoptionRejectsInvalidCandidateConfiguration(t *testing.T) {
	d := Roots{"/pair/couch", "/pair", "/identity"}
	bad := d
	bad.StoreDir = d.IdentityDir
	for _, c := range []Candidate{{Roots: bad, State: "populated"}, {Roots: d, State: "unrecognized"}} {
		if _, e := DecideAdoption(Request{}, d, []Candidate{c}); e == nil {
			t.Fatal("invalid candidate accepted", c)
		}
	}
}
