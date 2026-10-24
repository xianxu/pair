package sessionsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/editorial"
)

const migrationDocument = "---\ntype: session-sync\nowner: ops:0\nopaque: {keep: yes}\ncollaboration-mode:\n  default: discussion\n  chat: [Notes]\n---\n# Notes\nhello\n"

func TestEditorialMigrationPreviewAndConfirmedApply(t *testing.T) {
	for _, registered := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "registered"}[registered], func(t *testing.T) {
			r := newRig(t)
			r.put(r.p.File, migrationDocument)
			paths := r.p
			if registered {
				if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
					t.Fatal(err)
				}
				_, err := withDocument(r.sync, r.p.File, func(f *file) (bool, error) { paths = f.paths; return true, nil })
				if err != nil {
					t.Fatal(err)
				}
			}
			r.put(paths.Round, "old round")
			beforeShadow := r.read(paths.Shadow)
			beforeState := r.read(paths.State)
			preview, err := r.sync.MigrateEditorial(EditorialMigrationRequest{File: r.p.File})
			if err != nil {
				t.Fatal(err)
			}
			want, err := editorial.PlanMigration([]byte(migrationDocument))
			if err != nil {
				t.Fatal(err)
			}
			if preview.Candidate != string(want.After) || preview.BeforeHash != want.BeforeHash || preview.Applied {
				t.Fatalf("preview %+v", preview)
			}
			if r.read(r.p.File) != migrationDocument || r.read(paths.Round) != "old round" || r.read(paths.Shadow) != beforeShadow || r.read(paths.State) != beforeState {
				t.Fatal("preview mutated document metadata")
			}
			applied, err := r.sync.MigrateEditorial(EditorialMigrationRequest{File: r.p.File, Confirm: true, IfHash: preview.BeforeHash})
			if err != nil {
				t.Fatal(err)
			}
			if !applied.Applied || r.read(r.p.File) != preview.Candidate || applied.Hash == preview.BeforeHash {
				t.Fatalf("applied %+v", applied)
			}
			if _, err := os.Stat(paths.Round); !os.IsNotExist(err) {
				t.Fatalf("old round retained: %v", err)
			}
			if r.read(paths.Shadow) != beforeShadow || r.read(paths.State) != beforeState {
				t.Fatal("operator migration acknowledged unseen edits")
			}
		})
	}
}

func TestEditorialMigrationPreflightsRegisteredGrowth(t *testing.T) {
	r := newRig(t)
	prefix := "---\ncollaboration-mode: {chat: [Notes]}\n---\n" + strings.Repeat("# Notes\nbody\n\n", 8)
	base := prefix + strings.Repeat("x", documentMaxBytes-len(prefix))
	r.put(r.p.File, base)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	paths := registeredPaths(t, r.sync, r.p.File)
	r.put(paths.Round, "previous round")
	beforeShadow, beforeState := r.read(paths.Shadow), r.read(paths.State)
	_, err := r.sync.MigrateEditorial(EditorialMigrationRequest{File: r.p.File, Confirm: true, IfHash: migrationHash(base)})
	wantRefusal(t, err, ExitRefused, "registry-limit")
	if r.read(r.p.File) != base || r.read(paths.Round) != "previous round" || r.read(paths.Shadow) != beforeShadow || r.read(paths.State) != beforeState {
		t.Fatal("oversized migration changed document or metadata")
	}
}

func TestEditorialMigrationRefusalsDoNotPublish(t *testing.T) {
	for _, name := range []string{"stale", "operator-lock", "alias-lock", "lossy-ask", "dangling", "missing-hash", "short-hash", "preview-hash"} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			r.put(r.p.File, migrationDocument)
			request := EditorialMigrationRequest{File: r.p.File}
			preview, err := r.sync.MigrateEditorial(request)
			if err != nil {
				t.Fatal(err)
			}
			request.Confirm = true
			request.IfHash = preview.BeforeHash
			switch name {
			case "stale":
				r.put(r.p.File, migrationDocument+"changed\n")
			case "operator-lock":
				r.put(r.p.Lock, "operator")
			case "alias-lock":
				if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
					t.Fatal(err)
				}
				alias := filepath.Join(filepath.Dir(r.p.File), "alias.md")
				if err := os.Symlink(r.p.File, alias); err != nil {
					t.Fatal(err)
				}
				r.put(alias+".lock", "operator")
				request.File = alias
			case "lossy-ask":
				r.put(r.p.File, strings.Replace(migrationDocument, "hello", "- legacy bullet ask", 1))
				request.Confirm = false
				request.IfHash = ""
			case "dangling":
				r.put(r.p.File, strings.Replace(migrationDocument, "[Notes]", "[Absent]", 1))
				request.Confirm = false
				request.IfHash = ""
			case "missing-hash":
				request.IfHash = ""
			case "short-hash":
				request.IfHash = preview.BeforeHash[:12]
			case "preview-hash":
				request.Confirm = false
			}
			source := r.read(r.p.File)
			r.put(r.p.Round, "old round")
			if _, err := r.sync.MigrateEditorial(request); err == nil {
				t.Fatal("unsafe migration accepted")
			}
			if r.read(r.p.File) != source || r.read(r.p.Round) != "old round" {
				t.Fatal("refusal mutated files")
			}
		})
	}
}

func TestEditorialMigrationNoopKeepsRoundAndRegisteredIdentity(t *testing.T) {
	r := newRig(t)
	source := "---\nowner: product-team\nopaque: {keep: yes}\ncollaboration-mode: discussion\npresentation-mode: highlight\nprocessing-mode: free-form\n---\n# Notes\nhello\n"
	r.put(r.p.File, source)
	if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
		t.Fatal(err)
	}
	var paths Paths
	if _, err := withDocument(r.sync, r.p.File, func(f *file) (bool, error) { paths = f.paths; return true, nil }); err != nil {
		t.Fatal(err)
	}
	r.put(paths.Round, "keep me")
	p, err := r.sync.MigrateEditorial(EditorialMigrationRequest{File: r.p.File})
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.sync.MigrateEditorial(EditorialMigrationRequest{File: r.p.File, Confirm: true, IfHash: p.BeforeHash})
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied || result.Candidate != source || r.read(paths.Round) != "keep me" {
		t.Fatalf("noop changed document: %+v", result)
	}
	registration, err := r.sync.Lookup(r.p.File)
	if err != nil {
		t.Fatal(err)
	}
	if registration.Owner != "ops:0" || frontmatter(r.read(r.p.File), "owner") != "product-team" || IsSessionSync(r.read(r.p.File)) {
		t.Fatal("registered identity changed")
	}
}

func TestEditorialMigrationRefusesPreviewUnderOperatorLock(t *testing.T) {
	r := newRig(t)
	r.put(r.p.File, migrationDocument)
	r.put(r.p.Lock, "operator")
	_, err := r.sync.MigrateEditorial(EditorialMigrationRequest{File: r.p.File})
	wantRefusal(t, err, ExitOperatorLock, "operator-lock")
	if r.read(r.p.File) != migrationDocument {
		t.Fatal("preview under unsaved operator buffer mutated source")
	}
}

func TestEditorialMigrationRoundClearFailurePreservesDocument(t *testing.T) {
	r := newRig(t)
	r.put(r.p.File, migrationDocument)
	p, err := r.sync.MigrateEditorial(EditorialMigrationRequest{File: r.p.File})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(r.p.Round, 0700); err != nil {
		t.Fatal(err)
	}
	r.put(filepath.Join(r.p.Round, "unremovable"), "content")
	_, err = r.sync.MigrateEditorial(EditorialMigrationRequest{File: r.p.File, Confirm: true, IfHash: p.BeforeHash})
	if err == nil {
		t.Fatal("migration published with uncleared previous round")
	}
	if r.read(r.p.File) != migrationDocument {
		t.Fatal("failed round clearing changed source")
	}
}
