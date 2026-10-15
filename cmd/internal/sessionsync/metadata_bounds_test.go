package sessionsync

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func registeredPaths(t *testing.T, s *Sync, path string) Paths {
	t.Helper()
	p, err := withRegistry(s, func(tx *registryTxn) (Paths, error) {
		canonical, err := canonicalDocument(path)
		if err != nil {
			return Paths{}, err
		}
		return tx.metadata(*tx.find(canonical)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRegistrationRefusesOversizedImportedMetadata(t *testing.T) {
	for _, kind := range []string{"shadow", "state"} {
		t.Run(kind, func(t *testing.T) {
			r := newRig(t)
			if kind == "shadow" {
				r.put(r.p.Shadow, strings.Repeat("x", documentMaxBytes+1))
			} else {
				raw, err := json.Marshal(State{Folded: []string{strings.Repeat("x", (1<<20)+1)}})
				if err != nil {
					t.Fatal(err)
				}
				r.put(r.p.State, string(raw))
			}
			beforeDoc, beforeState, beforeShadow := r.read(r.p.File), r.read(r.p.State), r.read(r.p.Shadow)
			_, err := r.sync.Register(r.p.File, "ops:0")
			wantRefusal(t, err, 1, "registry-limit")
			got, err := r.sync.Lookup(r.p.File)
			if err != nil || got != nil {
				t.Fatalf("admitted oversized import: %+v %v", got, err)
			}
			if r.read(r.p.File) != beforeDoc || r.read(r.p.State) != beforeState || r.read(r.p.Shadow) != beforeShadow {
				t.Fatal("refused import changed legacy authority")
			}
		})
	}
}

func TestRegisteredSectionWritePreflightsPreservedShadow(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(filepath.Dir(r.p.File), "plain.md")
	r.put(path, "## Work\n\nold\n\n## Unseen\n\n"+strings.Repeat("u", 2<<20))
	if _, err := r.sync.Register(path, "ops:0"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Read(path, ""); err != nil {
		t.Fatal(err)
	}
	// The operator deletes a different section; its old bytes must remain unseen.
	r.put(path, "## Work\n\nold\n")
	read, err := r.sync.Read(path, "Work")
	if err != nil {
		t.Fatal(err)
	}
	p := registeredPaths(t, r.sync, path)
	beforeDoc, beforeState, beforeShadow := r.read(path), r.read(p.State), r.read(p.Shadow)
	_, err = r.sync.Write(WriteRequest{File: path, Section: "Work", Body: strings.Repeat("w", (2<<20)+1024), IfHash: read.Hash})
	wantRefusal(t, err, 1, "registry-limit")
	if r.read(path) != beforeDoc || r.read(p.State) != beforeState || r.read(p.Shadow) != beforeShadow {
		t.Fatal("oversized prospective shadow changed document or metadata")
	}
	if _, err := r.sync.Read(path, "Work"); err != nil {
		t.Fatalf("refusal stranded registration: %v", err)
	}
}

func TestRegisteredWritePreflightsRestampedDocument(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(filepath.Dir(r.p.File), "plain.md")
	r.put(path, "## Work\n\nold\n")
	if _, err := r.sync.Register(path, "ops:0"); err != nil {
		t.Fatal(err)
	}
	read, err := r.sync.Read(path, "")
	if err != nil {
		t.Fatal(err)
	}
	p := registeredPaths(t, r.sync, path)
	beforeDoc, beforeState, beforeShadow := r.read(path), r.read(p.State), r.read(p.Shadow)
	r.now = r.now.AddDate(8000, 0, 0) // force a five-digit year after registration
	prefix := "_Updated 2026-10-10 12:00._\n\n"
	body := prefix + strings.Repeat("w", documentMaxBytes-len(prefix))
	if len(Restamp(body, r.now)) <= documentMaxBytes {
		t.Fatal("fixture does not exceed bound after restamp")
	}
	_, err = r.sync.Write(WriteRequest{File: path, Body: body, IfHash: read.Hash})
	wantRefusal(t, err, 1, "registry-limit")
	if r.read(path) != beforeDoc || r.read(p.State) != beforeState || r.read(p.Shadow) != beforeShadow {
		t.Fatal("restamp growth changed document or metadata")
	}
}

func TestRegisteredWritePreflightsFoldStateSize(t *testing.T) {
	r := newRig(t)
	path := filepath.Join(filepath.Dir(r.p.File), "plain.md")
	r.put(path, "## Work\n\nold\n")
	if _, err := r.sync.Register(path, "ops:0"); err != nil {
		t.Fatal(err)
	}
	read, err := r.sync.Read(path, "")
	if err != nil {
		t.Fatal(err)
	}
	p := registeredPaths(t, r.sync, path)
	beforeDoc, beforeState, beforeShadow := r.read(path), r.read(p.State), r.read(p.Shadow)
	var rows strings.Builder
	var ids []string
	for i := 0; i < 260; i++ {
		id := fmt.Sprintf("%d-%s", i, strings.Repeat("x", 4096))
		ids = append(ids, id)
		raw, err := json.Marshal(Entry{ID: id, From: "ops:0", At: r.now, Head: "delivery"})
		if err != nil {
			t.Fatal(err)
		}
		rows.Write(raw)
		rows.WriteByte('\n')
	}
	ledger, err := r.sync.Ledger.path("ops:0", ".jsonl")
	if err != nil {
		t.Fatal(err)
	}
	r.put(ledger, rows.String())
	_, err = r.sync.Write(WriteRequest{File: path, Body: "## Work\n\nnew\n", IfHash: read.Hash, Fold: ids})
	wantRefusal(t, err, 1, "registry-limit")
	if r.read(path) != beforeDoc || r.read(p.State) != beforeState || r.read(p.Shadow) != beforeShadow {
		t.Fatal("oversized fold state changed document or metadata")
	}
}
