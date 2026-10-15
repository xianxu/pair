package sessionsync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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

// Size the externally encoded handoff representation, including its newline.
func handoffSizedState(t *testing.T, since time.Time, size int, paths []PathInterval) State {
	t.Helper()
	st := State{Since: since, Folded: []string{"kept", ""}, SubmitPaths: paths}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if size < len(raw)+1 {
		t.Fatal("invalid fixture budget")
	}
	st.Folded[1] = strings.Repeat("x", size-len(raw)-1)
	st.SubmitPaths = nil
	return st
}

func TestRegisterReservesHandoffStateEnvelopeAndRoundTrips(t *testing.T) {
	for _, spare := range []int{64, 4096} {
		t.Run(fmt.Sprint(spare), func(t *testing.T) {
			r := newRig(t)
			st := handoffSizedState(t, r.now, registeredStateMaxBytes-spare, nil)
			if err := saveState(r.p, st); err != nil {
				t.Fatal(err)
			}
			originalState := r.read(r.p.State)
			_, err := r.sync.Register(r.p.File, "ops:0")
			if spare == 64 {
				wantRefusal(t, err, 1, "registry-limit")
				if r.read(r.p.State) != originalState {
					t.Fatal("failed admission changed legacy state")
				}
				found, lookupErr := r.sync.Lookup(r.p.File)
				if lookupErr != nil || found != nil {
					t.Fatalf("admitted overflowing handoff: %+v %v", found, lookupErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.sync.Forget(r.p.File); err != nil {
				t.Fatal(err)
			}
			exported := r.read(r.p.State)
			if len(exported) > registeredStateMaxBytes {
				t.Fatal("forget exported unreadable state")
			}
			if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
				t.Fatalf("handoff cannot be reimported: %v", err)
			}
			p := registeredPaths(t, r.sync, r.p.File)
			adopted, err := LoadState(p)
			if err != nil || !reflect.DeepEqual(adopted.Folded, st.Folded) || !adopted.Since.Equal(st.Since) {
				t.Fatalf("handoff lost folds or horizon: %v", err)
			}
		})
	}
}

func TestRegisteredGrowthReservesHandoffBeforePublication(t *testing.T) {
	for _, operation := range []string{"register-alias", "rename", "write"} {
		t.Run(operation, func(t *testing.T) {
			r := newRig(t)
			if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
				t.Fatal(err)
			}
			d, err := withRegistry(r.sync, func(tx *registryTxn) (DocumentRegistration, error) {
				canonical, err := canonicalDocument(r.p.File)
				if err != nil {
					return DocumentRegistration{}, err
				}
				return *tx.find(canonical), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			st := handoffSizedState(t, r.now, registeredStateMaxBytes-16, d.Paths)
			p := registeredPaths(t, r.sync, r.p.File)
			if err := saveState(p, st); err != nil {
				t.Fatal(err)
			}
			// Keep folds live so writing cannot reclaim the reserved state accidentally.
			ledger, err := r.sync.Ledger.path("ops:0", ".jsonl")
			if err != nil {
				t.Fatal(err)
			}
			var rows strings.Builder
			for _, id := range st.Folded {
				raw, err := json.Marshal(Entry{ID: id, From: "ops:0", At: r.now, Head: "delivery"})
				if err != nil {
					t.Fatal(err)
				}
				rows.Write(raw)
				rows.WriteByte('\n')
			}
			r.put(ledger, rows.String())
			registry := filepath.Join(r.sync.registryStore().Dir, "registry.json")
			beforeRegistry, beforeDoc, beforeState, beforeShadow := r.read(registry), r.read(r.p.File), r.read(p.State), r.read(p.Shadow)
			switch operation {
			case "register-alias":
				alias := filepath.Join(filepath.Dir(r.p.File), "new-alias.md")
				if err := os.Symlink(r.p.File, alias); err != nil {
					t.Fatal(err)
				}
				_, err = r.sync.Register(alias, "ops:0")
			case "rename":
				next := filepath.Join(filepath.Dir(r.p.File), "new-name.md")
				r.put(next, r.read(r.p.File))
				r.now = r.now.Add(time.Minute)
				_, err = r.sync.Rename(r.p.File, next)
			case "write":
				_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "new\n", IfHash: r.hash("Trains")})
			}
			wantRefusal(t, err, 1, "registry-limit")
			if r.read(registry) != beforeRegistry || r.read(r.p.File) != beforeDoc || r.read(p.State) != beforeState || r.read(p.Shadow) != beforeShadow {
				t.Fatal("handoff overflow changed authority, document or metadata")
			}
			if _, err := r.sync.Forget(r.p.File); err != nil {
				t.Fatalf("refused growth stranded valid handoff: %v", err)
			}
			if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
				t.Fatalf("valid handoff cannot round trip: %v", err)
			}
		})
	}
}

func TestRegisteredPreparedWriteRejectsAmbiguousResultBeforeEffects(t *testing.T) {
	r := newRig(t)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	p := registeredPaths(t, r.sync, r.p.File)
	beforeDoc, beforeShadow, beforeState := r.read(r.p.File), r.read(p.Shadow), r.read(p.State)
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "## Trains\n\nduplicate\n", IfHash: r.hash("Trains")})
	wantRefusal(t, err, 1, "refused")
	if r.read(r.p.File) != beforeDoc || r.read(p.Shadow) != beforeShadow || r.read(p.State) != beforeState {
		t.Fatal("ambiguous prepared result changed document or metadata before refusal")
	}
}
