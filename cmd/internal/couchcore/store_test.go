package couchcore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTripsRegistryAndNaming(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	w := Worktree("/Users/x/KBench")
	reg, _ := NewRegistry().Register(ActorRecord{ID: "couch-a", Args: StartArgs{Worktree: w}, PID: 42, Identity: "tok"})
	names := NewNamingTable().SetName(w, "kaggle").SetDescription(w, "arc-agi-3")

	if err := s.Save(reg, names); err != nil {
		t.Fatalf("Save: %v", err)
	}
	gotReg, gotNames, _, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	recs := gotReg.Get(w)
	if len(recs) != 1 || recs[0].PID != 42 || recs[0].Identity != "tok" {
		t.Fatalf("registry lost detail: %+v", recs)
	}
	if recs[0].Args.Worktree != w {
		t.Fatalf("worktree case lost: %q", recs[0].Args.Worktree)
	}
	if got := gotNames.Lookup("kaggle"); len(got) != 1 || got[0] != w {
		t.Fatalf("naming lost: %v -- names must survive a couch restart", got)
	}
}

func TestLoadOnMissingStoreIsEmptyNotError(t *testing.T) {
	reg, names, pol, err := NewStore(t.TempDir()).Load()
	if err != nil {
		t.Fatalf("first run must not error: %v", err)
	}
	if len(reg.Records()) != 0 || len(names.All()) != 0 {
		t.Fatal("expected empty state")
	}
	if got := pol.Mode("anything"); got != InPlaceSerial {
		t.Fatalf("default mode = %q", got)
	}
}

func TestLoadReadsRecordedPolicy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "policy.json"),
		[]byte(`{"kbench":"heavy-local-state","xianxu.dev":"worktree-parallel"}`), 0o644); err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	_, _, pol, err := NewStore(dir).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := pol.Mode("kbench"); got != HeavyLocalState {
		t.Fatalf("kbench = %q; policy must be READ, not inferred", got)
	}
}

func TestSaveIsAtomicallyReplaceable(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Save(NewRegistry(), NewNamingTable()); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	reg, _ := NewRegistry().Register(ActorRecord{ID: "couch-a", Args: StartArgs{Worktree: "/repo"}})
	if err := s.Save(reg, NewNamingTable()); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	got, _, _, _ := s.Load()
	if len(got.Records()) != 1 {
		t.Fatalf("second Save did not replace the snapshot: %+v", got.Records())
	}
}
