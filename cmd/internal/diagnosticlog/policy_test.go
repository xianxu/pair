package diagnosticlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiagnosticLimitsPersistAcrossConflictingWriters(t *testing.T) {
	path, _, opts := fixture(t)
	opts.MaxBytes = 5
	limited, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer limited.Close()
	loose := opts
	loose.MaxBytes = 1000
	loose.Proof = func(context.Context, string, []Registration) error { return ErrUnknownWriters }
	other, err := Open(path, loose)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if n, err := limited.Write([]byte("abc")); n != 3 || err != nil {
		t.Fatal(n, err)
	}
	if n, err := other.Write([]byte("def")); n != 0 || !errors.Is(err, ErrLimit) {
		t.Fatalf("handle widened persisted cap: n=%d err=%v", n, err)
	}
	state, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != 2 || state.Limits.MaxBytes != 5 || state.Limits.MaxGenerations != 3 {
		t.Fatalf("limits not authoritative: %+v", state)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "abc" {
		t.Fatal(string(body), err)
	}
}
func TestDiagnosticGenerationCountBound(t *testing.T) {
	path, _, opts := fixture(t)
	opts.MaxBytes = 4
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for _, record := range []string{"0000", "1111", "2222", "3333", "4444", "5555", "6666"} {
		if n, err := w.Write([]byte(record)); n != 4 || err != nil {
			t.Fatal(record, n, err)
		}
	}
	names, _, complete, err := generationPage(path, "", 65, false, opts)
	if err != nil || !complete || len(names) != 3 {
		t.Fatalf("retained=%d complete=%t err=%v", len(names), complete, err)
	}
}

func TestResolveDiagnosticLimitsTightensOnly(t *testing.T) {
	defaults := DiagnosticLimits{MaxGenerationBytes, MaxClosedGenerations}
	for _, tc := range []struct {
		name                       string
		persisted, requested, want DiagnosticLimits
		bad                        bool
	}{
		{name: "defaults", want: defaults},
		{name: "initial smaller", requested: DiagnosticLimits{10, 2}, want: DiagnosticLimits{10, 2}},
		{name: "never wider", persisted: DiagnosticLimits{10, 1}, requested: DiagnosticLimits{100, 3}, want: DiagnosticLimits{10, 1}},
		{name: "tighten", persisted: DiagnosticLimits{10, 3}, requested: DiagnosticLimits{5, 1}, want: DiagnosticLimits{5, 1}},
		{name: "negative request", requested: DiagnosticLimits{-1, 0}, bad: true},
		{name: "negative count", requested: DiagnosticLimits{0, -1}, bad: true},
		{name: "partial persisted", persisted: DiagnosticLimits{10, 0}, bad: true},
		{name: "oversized persisted", persisted: DiagnosticLimits{MaxGenerationBytes + 1, 3}, bad: true},
		{name: "overcount persisted", persisted: DiagnosticLimits{10, 4}, bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveDiagnosticLimits(tc.persisted, tc.requested)
			if (err != nil) != tc.bad || (!tc.bad && got != tc.want) {
				t.Fatal(got, err)
			}
		})
	}
}
func TestSelectExcessGenerationsRequiresCertifiedCompleteInventory(t *testing.T) {
	current, _ := quarantineReservation()
	now := current.LastWrite.Add(time.Hour)
	makeGeneration := func(n int) generation {
		g := current
		g.Name = fmt.Sprintf("segment-%032x.log", n)
		g.LastWrite = g.LastWrite.Add(time.Duration(n) * time.Second)
		return g
	}
	candidates := []generation{makeGeneration(3), makeGeneration(1), makeGeneration(2), makeGeneration(4)}
	limits := DiagnosticLimits{10, 3}
	selected, err := SelectExcessGenerations(now, candidates, limits, true)
	if err != nil || len(selected) != 1 || selected[0].Name != makeGeneration(1).Name {
		t.Fatal(selected, err)
	}
	if candidates[0].Name != makeGeneration(3).Name {
		t.Fatal("input mutated")
	}
	for _, tc := range []struct {
		name     string
		change   func([]generation) []generation
		complete bool
	}{
		{"incomplete", func(g []generation) []generation { return g }, false},
		{"current", func(g []generation) []generation { g[0].Name = ""; return g }, true},
		{"duplicate", func(g []generation) []generation { g[0] = g[1]; return g }, true},
		{"future", func(g []generation) []generation { g[0].LastWrite = now.Add(time.Second); return g }, true},
		{"missing identity", func(g []generation) []generation { g[0].Identity = identity{}; return g }, true},
		{"invalid uncertainty", func(g []generation) []generation {
			g[0].UnverifiedTail = &unverifiedTail{Offset: 0, Length: 1, Digest: "bad"}
			return g
		}, true},
		{"too many", func(g []generation) []generation { return make([]generation, maxPolicyCandidates+1) }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.change(append([]generation(nil), candidates...))
			if _, err := SelectExcessGenerations(now, g, limits, tc.complete); err == nil {
				t.Fatal("uncertified selection")
			}
		})
	}
	// Equal timestamps use a stable path tie break without trusting directory order.
	for i := range candidates {
		candidates[i].LastWrite = current.LastWrite
	}
	selected, err = SelectExcessGenerations(now, candidates, limits, true)
	if err != nil || len(selected) != 1 || selected[0].Name != makeGeneration(1).Name {
		t.Fatal(selected, err)
	}
}
func TestDiagnosticQuarantineCountsAndDeletesWithExactMetadata(t *testing.T) {
	path, opts := killedPartialForQuarantine(t)
	opts.MaxBytes = 6
	opts.MaxGenerations = 1
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	assertPreservedQuarantine(t, path, opts)
	rows, err := Preview(path, opts, 100)
	if err != nil {
		t.Fatal(err)
	}
	marked := false
	for _, row := range rows {
		if row.Unverified {
			marked = true
		}
	}
	if !marked {
		t.Fatal("preview hides quarantined uncertainty")
	}

	if _, err = w.Write([]byte("next12")); err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("fresh3")); err != nil {
		t.Fatal(err)
	}
	names, _, done, err := generationPage(path, "", 65, false, opts)
	if err != nil || !done || len(names) != 1 {
		t.Fatal(names, done, err)
	}
	var g generation
	if err = readJSON(segmentPath(path, names[0])+".json", &g); err != nil {
		t.Fatal(err)
	}
	if g.UnverifiedTail != nil {
		t.Fatal("old quarantine escaped generation count")
	}
	body, err := os.ReadFile(segmentPath(path, names[0]))
	if err != nil || string(body) != "next12" {
		t.Fatal(string(body), err)
	}
}
func TestDiagnosticPolicyDeletionReplay(t *testing.T) {
	for _, step := range []string{"delete-intent", "delete-payload", "delete-metadata", "delete-retire-intent"} {
		t.Run(step, func(t *testing.T) {
			path, _, opts := fixture(t)
			opts.MaxBytes = 4
			opts.MaxGenerations = 1
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			for _, record := range []string{"0000", "1111"} {
				if _, err = w.Write([]byte(record)); err != nil {
					t.Fatal(err)
				}
			}
			stop := errors.New(step)
			w.options.Fault = func(s string) error {
				if s == step {
					return stop
				}
				return nil
			}
			if _, err = w.Write([]byte("2222")); !errors.Is(err, stop) {
				t.Fatal(err)
			}
			w.Close()
			w, err = Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			if _, err = w.Write([]byte("2222")); err != nil {
				t.Fatal(err)
			}
			names, _, done, err := generationPage(path, "", 65, false, opts)
			if err != nil || !done || len(names) != 1 {
				t.Fatal(names, done, err)
			}
			body, err := os.ReadFile(path)
			if err != nil || string(body) != "2222" {
				t.Fatal(string(body), err)
			}
		})
	}
}

func TestDiagnosticLegacyIntentRecoveredBeforeLimitsMigration(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte("base")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	legacy, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	legacy.Version = 1
	legacy.Limits = DiagnosticLimits{}
	legacy.Appending = &appendIntent{Before: legacy.Current, Data: []byte("tail"), At: opts.Now()}
	if err = save(path, legacy, true, opts); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("ta")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	opts.MaxBytes = 5
	opts.Proof = func(context.Context, string, []Registration) error { return ErrUnknownWriters }
	w, err = Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	migrated, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.Version != 2 || migrated.Appending != nil || migrated.Limits.MaxBytes != 5 || migrated.Current.Size != 6 {
		t.Fatalf("migration lost old confirmed partial: %+v", migrated)
	}
	if n, err := w.Write([]byte("!")); n != 0 || !errors.Is(err, ErrLimit) {
		t.Fatal(n, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "baseta" {
		t.Fatal(string(body), err)
	}
}
func TestDiagnosticIncompleteInventoryRefusesGrowth(t *testing.T) {
	path, _, opts := fixture(t)
	opts.MaxBytes = 4
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err = w.Write([]byte("base")); err != nil {
		t.Fatal(err)
	}
	state, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxPolicyCandidates+1; i++ {
		name := fmt.Sprintf("segment-%04x%028x.log", i, i)
		p := segmentPath(path, name)
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte("kept"), 0600); err != nil {
			t.Fatal(err)
		}
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		g := state.Current
		g.Name = name
		g.Identity = fileIdentity(st)
		g.Size = st.Size()
		g.ModTime = st.ModTime()
		encoded, err := json.Marshal(g)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p+".json", encoded, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := w.Write([]byte("next")); n != 0 || !errors.Is(err, ErrMore) {
		t.Fatal("incomplete inventory authorized growth", n, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "base" {
		t.Fatal(string(body), err)
	}
	for i := 0; i < maxPolicyCandidates+1; i++ {
		p := segmentPath(path, fmt.Sprintf("segment-%04x%028x.log", i, i))
		body, err := os.ReadFile(p)
		if err != nil || string(body) != "kept" {
			t.Fatal(i, string(body), err)
		}
	}
}
func TestDiagnosticV2RefusesLegacyStrictReader(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	body, err := os.ReadFile(filepath.Join(directory(path), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Freeze the v1 decoder's exact field inventory, rather than letting the
	// current struct silently accept newly published policy authority.
	var legacy struct {
		Creating  *generation    `json:"creating,omitempty"`
		Appending *appendIntent  `json:"appending,omitempty"`
		Version   int            `json:"version"`
		Path      string         `json:"path"`
		Current   generation     `json:"current"`
		Writers   []Registration `json:"writers"`
		Pending   *generation    `json:"pending,omitempty"`
		Deleting  *generation    `json:"deleting,omitempty"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&legacy); err == nil {
		t.Fatal("legacy writer accepted authority it cannot enforce")
	}
}

func TestDiagnosticOversizedLegacyGenerationReportedUntilCountEviction(t *testing.T) {
	path, now, opts := fixture(t)
	opts.MaxBytes = 4
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	path = w.path
	state, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("segment-%032x.log", 1)
	p := segmentPath(path, name)
	if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	// Sparse legacy evidence models a multi-GiB old trace without writing GiB.
	if err = f.Truncate(4 << 30); err != nil {
		t.Fatal(err)
	}
	f.Close()
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	g := state.Current
	g.Name = name
	g.Identity = fileIdentity(st)
	g.Size = st.Size()
	g.ModTime = st.ModTime()
	body, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(p+".json", body, 0600); err != nil {
		t.Fatal(err)
	}
	rows, err := Preview(path, opts, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range rows {
		if row.Path == p {
			found = true
			if row.Bytes != 4<<30 || !row.Oversized || row.Eligible || row.Reason != "pre-existing diagnostic generation exceeds byte limit" {
				t.Fatalf("legacy excess hidden: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("missing legacy generation")
	}
	for _, record := range []string{"0000", "1111", "2222"} {
		*now = now.Add(time.Second)
		if _, err = w.Write([]byte(record)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = os.Stat(p); err != nil {
		t.Fatal("legacy data removed before ordinary count eviction", err)
	}
	*now = now.Add(time.Second)
	if _, err = w.Write([]byte("3333")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("old certified legacy segment escaped count eviction", err)
	}
}

func TestDiagnosticTightenedCountDebtBlocksEveryOpenWriter(t *testing.T) {
	path, now, opts := fixture(t)
	opts.MaxBytes = 4
	owner, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	for _, record := range []string{"0000", "1111", "2222", "z"} {
		*now = now.Add(time.Second)
		if _, err = owner.Write([]byte(record)); err != nil {
			t.Fatal(err)
		}
	}
	peer, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	tightened := opts
	tightened.MaxGenerations = 1
	unknown := func(context.Context, string, []Registration) error { return ErrUnknownWriters }
	tightened.Proof = unknown
	narrow, err := Open(path, tightened)
	if err != nil {
		t.Fatal(err)
	}
	defer narrow.Close()
	owner.options.Proof = unknown
	peer.options.Proof = unknown
	for _, writer := range []*Writer{owner, peer, narrow} {
		if n, err := writer.Write([]byte("a")); n != 0 || !errors.Is(err, ErrLimit) || !errors.Is(err, ErrUnknownWriters) {
			t.Fatal("count debt allowed growth", n, err)
		}
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "z" {
		t.Fatal(string(body), err)
	}
	names, _, done, err := generationPage(path, "", 65, false, opts)
	if err != nil || !done || len(names) != 3 {
		t.Fatal("unknown proof changed evidence", names, done, err)
	}
	if err = Maintain(path, opts); err != nil {
		t.Fatal(err)
	}
	if n, err := peer.Write([]byte("a")); n != 1 || err != nil {
		t.Fatal(n, err)
	}
	state, err := load(path)
	if err != nil || state.Limits.MaxGenerations != 1 {
		t.Fatal(state.Limits, err)
	}
	names, _, done, err = generationPage(path, "", 65, false, opts)
	if err != nil || !done || len(names) != 1 {
		t.Fatal(names, done, err)
	}
}
