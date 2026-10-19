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
func TestDiagnosticGenerationsStayUntilBackup(t *testing.T) {
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
	names, _, done, err := generationPage(path, "", 100, false, opts)
	if err != nil || !done || len(names) != 6 {
		t.Fatal(names, done, err)
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
func TestGenerationPolicyAndRetentionSetting(t *testing.T) {
	now := time.Now()
	for _, value := range []string{"0", "-1", "366", "bad", "999999999999999999999999", " 7"} {
		if _, err := ParseRetentionDays(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	for _, days := range []int{1, 7, 365} {
		period, err := ParseRetentionDays(fmt.Sprint(days))
		if err != nil {
			t.Fatal(err)
		}
		for _, age := range []time.Duration{-time.Hour, 0, period - 1, period} {
			for _, size := range []int64{10, 11} {
				want := age >= 0 && (age >= period || size > 10)
				if got := DecideGeneration(now, now.Add(-age), size, 10, period); got != want {
					t.Fatalf("days=%d age=%s size=%d got=%t", days, age, size, got)
				}
			}
		}
		if DecideGeneration(now, time.Time{}, 11, 10, period) {
			t.Fatal("unknown age granted deletion")
		}
	}
}
func TestDiagnosticQuarantinePreservedBeyondCountLimit(t *testing.T) {
	path, opts := killedPartialForQuarantine(t)
	opts.MaxBytes = 6
	opts.MaxGenerations = 1
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	assertPreservedQuarantine(t, path, opts)
	for _, record := range []string{"next12", "fresh3"} {
		if _, err = w.Write([]byte(record)); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := Preview(path, opts, 100)
	if err != nil {
		t.Fatal(err)
	}
	marked := false
	for _, row := range rows {
		if row.Eligible {
			t.Fatal("backup fence bypassed", row)
		}
		if row.Unverified {
			marked = true
			body, err := os.ReadFile(row.Path)
			if err != nil || string(body) != "baseta" {
				t.Fatal(string(body), err)
			}
		}
	}
	if !marked {
		t.Fatal("quarantine evicted")
	}
}

func TestDiagnosticPolicyDoesNotStartDeletionBeforeBackup(t *testing.T) {
	for _, step := range []string{"delete-intent", "delete-payload", "delete-metadata", "delete-retire-intent"} {
		t.Run(step, func(t *testing.T) {
			path, _, opts := fixture(t)
			opts.MaxBytes = 4
			opts.MaxGenerations = 1
			hit := false
			opts.Fault = func(s string) error {
				if s == step {
					hit = true
					return errors.New(step)
				}
				return nil
			}
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			for _, record := range []string{"0000", "1111", "2222"} {
				if _, err = w.Write([]byte(record)); err != nil {
					t.Fatal(err)
				}
			}
			if hit {
				t.Fatal("eviction began before backup")
			}
			names, _, done, err := generationPage(path, "", 100, false, opts)
			if err != nil || !done || len(names) != 2 {
				t.Fatal(names, done, err)
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
func TestDiagnosticLargeInventoryDoesNotBlockRotation(t *testing.T) {
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
	for i := 0; i < 65; i++ {
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
	if n, err := w.Write([]byte("next")); n != 4 || err != nil {
		t.Fatal("backup-fenced inventory blocked rotation", n, err)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "next" {
		t.Fatal(string(body), err)
	}
	for i := 0; i < 65; i++ {
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

func TestDiagnosticOversizedLegacyGenerationEvictedWithoutPayloadCopy(t *testing.T) {
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
			if row.Bytes != 4<<30 || !row.Oversized || !row.Eligible || row.Reason != "pre-existing diagnostic generation exceeds byte limit" {
				t.Fatalf("legacy excess hidden: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("missing legacy generation")
	}
	if rows, err := Collect(path, opts, 100); err != nil || len(rows) != 1 || rows[0].Bytes != 4<<30 {
		t.Fatal(rows, err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("oversized segment remains", err)
	}
	if _, err := w.Write([]byte("new!")); err != nil {
		t.Fatal("fresh generation unusable", err)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "new!" {
		t.Fatal(string(b), err)
	}
	_ = now

}

func TestDiagnosticCountDebtDormantUntilBackup(t *testing.T) {
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
	tightened.Proof = func(context.Context, string, []Registration) error { return ErrUnknownWriters }
	narrow, err := Open(path, tightened)
	if err != nil {
		t.Fatal(err)
	}
	defer narrow.Close()
	for _, writer := range []*Writer{owner, peer, narrow} {
		if n, err := writer.Write([]byte("a")); n != 1 || err != nil {
			t.Fatal("dormant count debt blocked growth", n, err)
		}
	}
	if err = Maintain(path, opts); err != nil {
		t.Fatal(err)
	}
	names, _, done, err := generationPage(path, "", 100, false, opts)
	if err != nil || !done || len(names) != 4 {
		t.Fatal("count eviction before backup", names, done, err)
	}
}
