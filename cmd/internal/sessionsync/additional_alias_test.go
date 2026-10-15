package sessionsync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAdditionalRegisteredLeafAliasKeepsOperatorLockAuthority(t *testing.T) {
	r := newRig(t)
	first, err := r.sync.Register(r.p.File, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias.md")
	if err := os.Symlink(r.p.File, alias); err != nil {
		t.Fatal(err)
	}
	second, err := r.sync.Register(alias, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	if first.Registration.ID != second.Registration.ID || !first.Registration.RegisteredAt.Equal(second.Registration.RegisteredAt) {
		t.Fatal("alias registration reset identity")
	}
	h := r.hash("Trains")
	r.put(alias+".lock", "holder: operator\n")
	_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "must not land", IfHash: h})
	wantRefusal(t, err, 3, "operator-lock")
}

func TestForgetExportsMetadataToAdditionalRegisteredLeafAlias(t *testing.T) {
	r := newRig(t)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias.md")
	if err := os.Symlink(r.p.File, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Register(alias, "ops:0"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "latest\n", IfHash: r.hash("Trains")}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Forget(r.p.File); err != nil {
		t.Fatal(err)
	}
	if _, diff, err := r.sync.DiffText(alias); err != nil || diff != "" {
		t.Fatalf("alias handoff lost latest shadow: %q %v", diff, err)
	}
}

func TestAdditionalAliasRenameKeepsLateSubmitAcrossReuse(t *testing.T) {
	r := newRig(t)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(filepath.Dir(r.p.File), "additional.md")
	moved := filepath.Join(filepath.Dir(r.p.File), "moved.md")
	if err := os.Symlink(r.p.File, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Register(alias, "ops:0"); err != nil {
		t.Fatal(err)
	}
	addressed := r.now
	r.now = r.now.Add(time.Minute)
	if err := os.Rename(alias, moved); err != nil {
		t.Fatal(err)
	}
	renamed, err := r.sync.Rename(alias, moved)
	if err != nil || !renamed.Changed {
		t.Fatalf("rename: %+v %v", renamed, err)
	}
	retried, err := r.sync.Rename(alias, moved)
	if err != nil || retried.Changed || retried.Registration.ID != renamed.Registration.ID {
		t.Fatalf("retry: %+v %v", retried, err)
	}
	// The former alias now belongs to a different, later-registered document.
	r.put(alias, "## Work\n\nreplacement\n")
	if _, err := r.sync.Register(alias, "ops:0"); err != nil {
		t.Fatal(err)
	}
	if err := r.sync.Ledger.Append("ops:0", Entry{ID: "late-alias", From: "ops:0", At: r.now.Add(time.Minute), AddressedAt: addressed, Head: SubmitPrefix + alias}); err != nil {
		t.Fatal(err)
	}
	original, err := r.sync.Status(r.p.File)
	if err != nil || len(original.Owed) != 1 || original.Owed[0].ID != "late-alias" {
		t.Fatalf("lost old alias submit: %+v %v", original, err)
	}
	replacement, err := r.sync.Status(alias)
	if err != nil || len(replacement.Owed) != 0 {
		t.Fatalf("replacement inherited submit: %+v %v", replacement, err)
	}
}
