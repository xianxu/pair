package sessionsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func modelEvent(kind, file string, n int) RegistryEvent {
	return RegistryEvent{Kind: kind, File: file, ID: fmt.Sprintf("%032x", n), Owner: "pair:5", At: time.Date(2026, 1, 1, 0, n, 0, 0, time.UTC)}
}
func modelCode(t *testing.T, err error, code string) {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) || r.Code != code || r.Exit != 1 {
		t.Fatalf("error=%v, want %s", err, code)
	}
}
func modelApply(t *testing.T, r RegistryState, e RegistryEvent) (RegistryState, RegistryChange) {
	t.Helper()
	next, c, err := r.transition(e)
	if err != nil {
		t.Fatal(err)
	}
	return next, c
}
func TestRegistryModelLifecycle(t *testing.T) {
	r := RegistryState{Version: 1}
	e := modelEvent("register", "/a", 1)
	r, c := modelApply(t, r, e)
	if !c.Changed || c.Registration.ID != e.ID {
		t.Fatalf("%+v", c)
	}
	before, _ := json.Marshal(r)
	_, c = modelApply(t, r, e)
	if c.Changed {
		t.Fatal("register retry changed")
	}
	bad := e
	bad.Owner = "pair:6"
	_, _, err := r.transition(bad)
	modelCode(t, err, "owner-conflict")
	rename := modelEvent("rename", "/b", 2)
	rename.Old = "/a"
	next, c := modelApply(t, r, rename)
	if !c.Changed || c.Registration.ID != e.ID || len(c.Registration.Paths) != 2 {
		t.Fatalf("%+v", c)
	}
	after, _ := json.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("input mutated")
	}
	_, c = modelApply(t, next, rename)
	if c.Changed {
		t.Fatal("rename retry changed")
	}
	wrong := rename
	wrong.Old = "/unknown"
	_, _, err = next.transition(wrong)
	modelCode(t, err, "not-registered")
	// Reusing the old name creates an independent identity; it must not replay or take over /b.
	next, _ = modelApply(t, next, modelEvent("register", "/a", 3))
	_, _, err = next.transition(rename)
	modelCode(t, err, "target-conflict")
	next, c = modelApply(t, next, modelEvent("forget", "/a", 4))
	if !c.Changed || len(next.Documents) != 1 || next.Documents[0].File != "/b" {
		t.Fatal("forget removed wrong identity")
	}
	_, c = modelApply(t, next, modelEvent("forget", "/a", 5))
	if c.Changed {
		t.Fatal("forget retry changed")
	}
}
func TestRegistryModelDetachedResults(t *testing.T) {
	r, _ := modelApply(t, RegistryState{Version: 1}, modelEvent("register", "/a", 1))
	next, c := modelApply(t, r, modelEvent("register", "/a", 1))
	next.Documents[0].Paths[0].File = "/changed"
	if r.Documents[0].Paths[0].File != "/a" {
		t.Fatal("state aliases input")
	}
	c.Registration.Paths[0].File = "/other"
	if r.Documents[0].Paths[0].File != "/a" {
		t.Fatal("change aliases input")
	}
}
func TestRegistryModelBoundsAndNoEviction(t *testing.T) {
	r := RegistryState{Version: 1}
	// Construct valid independent documents without quadratic transitions.
	for i := 1; i <= registryMaxDocuments; i++ {
		e := modelEvent("register", fmt.Sprintf("/d%d", i), 1)
		e.ID = fmt.Sprintf("%032x", i)
		r.Documents = append(r.Documents, DocumentRegistration{ID: e.ID, File: e.File, RegisteredOwner: e.Owner, RegisteredAt: e.At, Paths: []PathInterval{{File: e.File, From: e.At}}})
	}
	raw, _ := json.Marshal(r)
	_, _, err := r.transition(modelEvent("register", "/extra", 1))
	modelCode(t, err, "registry-limit")
	if !strings.Contains(err.Error(), "couch --session-sync forget FILE") {
		t.Fatal(err)
	}
	after, _ := json.Marshal(r)
	if string(raw) != string(after) {
		t.Fatal("capacity failure mutated input")
	}
	r = RegistryState{Version: 1}
	r, _ = modelApply(t, r, modelEvent("register", "/a", 0))
	for i := 1; i < registryMaxPaths; i++ {
		e := modelEvent("rename", fmt.Sprintf("/alias%d", i), i)
		e.Old = r.Documents[0].File
		r, _ = modelApply(t, r, e)
	}
	e := modelEvent("rename", "/overflow", 300)
	e.Old = r.Documents[0].File
	_, _, err = r.transition(e)
	modelCode(t, err, "registry-limit")
	if len(r.Documents[0].Paths) != registryMaxPaths {
		t.Fatal("history silently pruned")
	}
}
func TestDecodeRegistryValidation(t *testing.T) {
	r, _ := modelApply(t, RegistryState{Version: 1}, modelEvent("register", "/a", 1))
	valid, _ := json.Marshal(r)
	cases := map[string][]byte{"empty": nil, "truncated": valid[:len(valid)-1], "null": []byte("null"), "missing version": []byte(`{"Documents":[]}`), "unknown": []byte(`{"Version":2}`), "unknown field": []byte(`{"Version":1,"extra":1}`), "trailing": append(append([]byte{}, valid...), []byte(" {}")...), "oversize": []byte(strings.Repeat(" ", registryMaxBytes+1))}
	edits := map[string]func(*RegistryState){
		"id":             func(r *RegistryState) { r.Documents[0].ID = strings.Repeat("A", 32) },
		"relative":       func(r *RegistryState) { r.Documents[0].File = "a" },
		"unclean":        func(r *RegistryState) { r.Documents[0].Paths[0].Spelling = "/a/../a" },
		"newline":        func(r *RegistryState) { r.Documents[0].File = "/a\nb" },
		"nul":            func(r *RegistryState) { r.Documents[0].File = "/a\x00b" },
		"owner":          func(r *RegistryState) { r.Documents[0].RegisteredOwner = "pair" },
		"zero time":      func(r *RegistryState) { r.Documents[0].RegisteredAt = time.Time{} },
		"no paths":       func(r *RegistryState) { r.Documents[0].Paths = nil },
		"wrong current":  func(r *RegistryState) { r.Documents[0].Paths[0].File = "/other" },
		"closed current": func(r *RegistryState) { v := r.Documents[0].RegisteredAt; r.Documents[0].Paths[0].Until = &v },
		"duplicate":      func(r *RegistryState) { r.Documents = append(r.Documents, r.Documents[0]) },
	}
	for name, edit := range edits {
		var v RegistryState
		json.Unmarshal(valid, &v)
		edit(&v)
		cases[name], _ = json.Marshal(v)
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) { _, err := decodeRegistry(raw); modelCode(t, err, "registry-invalid") })
	}
	decoded, err := decodeRegistry(valid)
	if err != nil || !reflect.DeepEqual(decoded, r) {
		t.Fatalf("roundtrip: %+v %v", decoded, err)
	}
}
func TestRegistryModelImportedHistoryAndChronology(t *testing.T) {
	e := modelEvent("register", "/b", 5)
	end := e.At.Add(-time.Minute)
	e.InitialPaths = []PathInterval{{File: "/a", From: e.At.Add(-2 * time.Hour), Until: &end}}
	r, c := modelApply(t, RegistryState{Version: 1}, e)
	if len(c.Registration.Paths) != 2 || !c.Registration.Paths[0].From.Equal(e.InitialPaths[0].From) {
		t.Fatal("lost imported history")
	}
	raw, _ := json.Marshal(r)
	if _, err := decodeRegistry(raw); err != nil {
		t.Fatal(err)
	}
	rename := modelEvent("rename", "/c", 1)
	rename.Old = "/b"
	_, _, err := r.transition(rename)
	modelCode(t, err, "registry-invalid")
	// Overlap and an open historical interval are rejected.
	for _, open := range []bool{false, true} {
		copy := r
		copy.Documents = append([]DocumentRegistration{}, r.Documents...)
		copy.Documents[0].Paths = append([]PathInterval{}, r.Documents[0].Paths...)
		if open {
			copy.Documents[0].Paths[0].Until = nil
		} else {
			v := e.At.Add(time.Hour)
			copy.Documents[0].Paths[0].Until = &v
		}
		b, _ := json.Marshal(copy)
		_, err = decodeRegistry(b)
		modelCode(t, err, "registry-invalid")
	}
}
func FuzzDecodeRegistry(f *testing.F) {
	f.Add([]byte(`{"Version":1,"Documents":[]}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"Version":1`))
	f.Fuzz(func(t *testing.T, b []byte) {
		r, err := decodeRegistry(b)
		if err != nil {
			return
		}
		encoded, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		next, err := decodeRegistry(encoded)
		if err != nil || !reflect.DeepEqual(r, next) {
			t.Fatalf("roundtrip %v", err)
		}
	})
}

func TestRegistryModelSequencePreservesIndependentDocuments(t *testing.T) {
	r := RegistryState{Version: 1}
	r, _ = modelApply(t, r, modelEvent("register", "/independent", 1))
	r, _ = modelApply(t, r, modelEvent("register", "/moving", 2))
	independent := r.Documents[0]
	for i := 3; i < 60; i++ {
		old := r.Documents[1].File
		e := modelEvent("rename", fmt.Sprintf("/move%d", i), i)
		e.Old = old
		next, c := modelApply(t, r, e)
		if !reflect.DeepEqual(next.Documents[0], independent) || c.Registration.ID != r.Documents[1].ID || len(next.Documents[1].Paths) != len(r.Documents[1].Paths)+1 {
			t.Fatal("lost identity or independent registration")
		}
		retry, c := modelApply(t, next, e)
		if c.Changed || !reflect.DeepEqual(retry, next) {
			t.Fatal("non-idempotent rename")
		}
		raw, _ := json.Marshal(next)
		decoded, err := decodeRegistry(raw)
		if err != nil || !reflect.DeepEqual(decoded, next) {
			t.Fatalf("roundtrip: %v", err)
		}
		r = next
	}
	// Historical spelling is not a forgetting authority.
	next, c := modelApply(t, r, modelEvent("forget", "/moving", 61))
	if c.Changed || !reflect.DeepEqual(next, r) {
		t.Fatal("historical forget removed live document")
	}
}

func TestRegistryModelEncodedCapacity(t *testing.T) {
	r := RegistryState{Version: 1}
	for i := 0; i < 15; i++ {
		e := modelEvent("register", fmt.Sprintf("/%d%s", i, strings.Repeat("x", 500000)), i+1)
		r.Documents = append(r.Documents, DocumentRegistration{ID: e.ID, File: e.File, RegisteredOwner: e.Owner, RegisteredAt: e.At, Paths: []PathInterval{{File: e.File, From: e.At}}})
	}
	e := modelEvent("register", "/"+strings.Repeat("x", 1000000), 50)
	_, _, err := r.transition(e)
	modelCode(t, err, "registry-limit")
	if len(r.Documents) != 15 {
		t.Fatal("byte limit evicted registration")
	}
}

func TestEncodeRegistryValidation(t *testing.T) {
	r, _ := modelApply(t, RegistryState{Version: 1}, modelEvent("register", "/a", 1))
	raw, err := encodeRegistry(r)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeRegistry(raw)
	if err != nil || !reflect.DeepEqual(decoded, r) {
		t.Fatalf("roundtrip %v", err)
	}
	r.Version = 2
	_, err = encodeRegistry(r)
	modelCode(t, err, "registry-invalid")
}

func TestRegistryModelLegacyUnboundedFirstInterval(t *testing.T) {
	e := modelEvent("register", "/legacy", 5)
	e.InitialPaths = []PathInterval{{File: e.File, From: time.Time{}}}
	r, _, err := (RegistryState{Version: 1}).transition(e)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Documents[0].Paths[0].From.IsZero() {
		t.Fatal("legacy horizon changed")
	}
	rename := modelEvent("rename", "/renamed", 6)
	rename.Old = e.File
	r, _ = modelApply(t, r, rename)
	raw, err := encodeRegistry(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decodeRegistry(raw); err != nil {
		t.Fatal(err)
	}
	r.Documents[0].Paths[1].From = time.Time{}
	_, err = encodeRegistry(r)
	modelCode(t, err, "registry-invalid")
}

func TestRegistryModelAdditionalSpellingsAreBoundedAndDetached(t *testing.T) {
	e := modelEvent("register", "/a", 1)
	r, initial := modelApply(t, RegistryState{Version: 1}, e)
	e.Spelling = "/alias"
	next, c := modelApply(t, r, e)
	raw, _ := json.Marshal(next)
	if !c.Changed || !strings.Contains(string(raw), `"Spellings":["/alias"]`) {
		t.Fatalf("new alias was not retained: %s %+v", raw, c)
	}
	if c.Registration.ID != initial.Registration.ID || !c.Registration.RegisteredAt.Equal(initial.Registration.RegisteredAt) || !c.Registration.Paths[0].From.Equal(initial.Registration.Paths[0].From) || len(c.Registration.Paths) != len(initial.Registration.Paths) {
		t.Fatal("alias changed identity or owed horizon")
	}
	_, retry := modelApply(t, next, e)
	if retry.Changed {
		t.Fatal("same alias retry changed registry")
	}
	e.Owner = "ops:0"
	_, _, err := next.transition(e)
	modelCode(t, err, "owner-conflict")
	// Fill the additional-spelling bound without consuming rename history.
	e.Owner = "pair:5"
	for i := 1; i < registryMaxPaths; i++ {
		e.Spelling = fmt.Sprintf("/alias%d", i)
		next, _ = modelApply(t, next, e)
	}
	before, _ := json.Marshal(next)
	e.Spelling = "/overflow"
	_, _, err = next.transition(e)
	modelCode(t, err, "registry-limit")
	after, _ := json.Marshal(next)
	if string(before) != string(after) {
		t.Fatal("overflow evicted admitted spelling")
	}
	// A returned change must not alias either input or returned state.
	c.Registration.Paths[0].Spellings[0] = "/mutated"
	if next.Documents[0].Paths[0].Spellings[0] != "/alias" || len(r.Documents[0].Paths[0].Spellings) != 0 {
		t.Fatal("spelling slices alias input or change")
	}
}

func TestRegistryModelAdditionalSpellingSurvivesRenameAndCountsTowardTotalBound(t *testing.T) {
	r, _ := modelApply(t, RegistryState{Version: 1}, modelEvent("register", "/a", 1))
	alias := modelEvent("register", "/a", 2)
	alias.Spelling = "/alias"
	r, _ = modelApply(t, r, alias)
	rename := modelEvent("rename", "/b", 3)
	rename.Old = "/a"
	r, _ = modelApply(t, r, rename)
	if r.Documents[0].Paths[0].Until == nil || !reflect.DeepEqual(r.Documents[0].Paths[0].Spellings, []string{"/alias"}) {
		t.Fatal("rename lost closed interval alias")
	}
	for i := 1; i < registryMaxPaths; i++ {
		alias.File = "/b"
		alias.Spelling = fmt.Sprintf("/alias%d", i)
		r, _ = modelApply(t, r, alias)
	}
	alias.Spelling = "/overflow"
	_, _, err := r.transition(alias)
	modelCode(t, err, "registry-limit")
	// The decoder must enforce the same total even when each interval is small.
	r.Documents[0].Paths[1].Spellings = append(r.Documents[0].Paths[1].Spellings, "/overflow")
	raw, _ := json.Marshal(r)
	_, err = decodeRegistry(raw)
	modelCode(t, err, "registry-invalid")
}

func TestRegistryModelImportedHistoryAdmitsNewCallerAlias(t *testing.T) {
	e := modelEvent("register", "/a", 2)
	e.Spelling = "/newalias"
	e.InitialPaths = []PathInterval{{File: "/a", Spelling: "/oldalias", From: e.At.Add(-time.Hour)}}
	_, c := modelApply(t, RegistryState{Version: 1}, e)
	if !reflect.DeepEqual(c.Registration.Paths[0].Spellings, []string{"/newalias"}) {
		t.Fatalf("import dropped caller alias: %+v", c.Registration.Paths)
	}
}

func TestRegistryModelRenameRetryRecognizesAdditionalPreviousAlias(t *testing.T) {
	r, _ := modelApply(t, RegistryState{Version: 1}, modelEvent("register", "/canonical", 1))
	alias := modelEvent("register", "/canonical", 2)
	alias.Spelling = "/alias"
	r, _ = modelApply(t, r, alias)
	rename := modelEvent("rename", "/canonical", 3)
	rename.Old = "/canonical"
	rename.Spelling = "/movedalias"
	r, _ = modelApply(t, r, rename)
	rename.Old = "/alias"
	_, c, err := r.transition(rename)
	if err != nil || c.Changed || c.Registration == nil || c.Registration.ID != r.Documents[0].ID {
		t.Fatalf("alias retry: %+v %v", c, err)
	}
}
