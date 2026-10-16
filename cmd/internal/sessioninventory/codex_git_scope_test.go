package sessioninventory

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func codexTestWorktrees(t *testing.T) (string, string, string) {
	t.Helper()
	base := t.TempDir()
	base, _ = filepath.EvalSymlinks(base)
	primary := filepath.Join(base, "primary")
	linked := filepath.Join(base, "linked")
	admin := filepath.Join(primary, ".git", "worktrees", "linked")
	for _, p := range []string{admin, filepath.Join(primary, "sub"), filepath.Join(linked, "sub")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for p, v := range map[string]string{filepath.Join(primary, ".git", "HEAD"): "ref: refs/heads/main\n", filepath.Join(linked, ".git"): "gitdir: " + admin + "\n", filepath.Join(admin, "gitdir"): filepath.Join(linked, ".git") + "\n", filepath.Join(admin, "commondir"): "../..\n"} {
		if err := os.WriteFile(p, []byte(v), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return primary, linked, admin
}
func TestCodexGitWorktreeScopeValidatesBacklinksAndCorrespondingDirectories(t *testing.T) {
	primary, linked, admin := codexTestWorktrees(t)
	id, ok := codexRepositoryIdentity(filepath.Join(linked, "sub"))
	if !ok || id.primary != primary || id.relative != "sub" {
		t.Fatalf("%+v %v", id, ok)
	}
	cwds, err := codexWorktreeCWDs(filepath.Join(linked, "sub"))
	if err != nil || len(cwds) != 2 || cwds[0] != filepath.Join(linked, "sub") || cwds[1] != filepath.Join(primary, "sub") {
		t.Fatalf("%v %v", cwds, err)
	}
	if err := os.WriteFile(filepath.Join(admin, "gitdir"), []byte(filepath.Join(primary, ".git")), 0600); err != nil {
		t.Fatal(err)
	}
	cwds, err = codexWorktreeCWDs(filepath.Join(primary, "sub"))
	if err != nil || len(cwds) != 1 {
		t.Fatalf("forged backlink widened scope: %v %v", cwds, err)
	}
}
func TestCodexGitProjectTrustFallsBackToValidatedPrimary(t *testing.T) {
	primary, linked, _ := codexTestWorktrees(t)
	r, q, ch, _ := codexTestConfig(t, "forced_login_method='api'\n")
	q.CWD = filepath.Join(linked, "sub")
	q.Overrides = []string{`projects={"` + primary + `"={trust_level='trusted'}}`}
	folder := filepath.Join(linked, ".codex")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "config.toml"), []byte("sqlite_home='primary-trusted-db'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigComplete || p.Scope.SQLiteHome != filepath.Join(folder, "primary-trusted-db") || len(p.Scope.CWDs) != 2 {
		t.Fatalf("%+v %v home %s", p, err, ch)
	}
	q.Disable = []string{"worktrees"}
	p, err = readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigComplete || len(p.Scope.CWDs) != 1 {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestCodexGitWorktreeDirectoryCannotBlockOnFIFO(t *testing.T) {
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, ".git", "HEAD"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(d, ".git", "worktrees")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := codexWorktreeCWDs(d); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("non-directory accepted")
		}
	case <-time.After(time.Second):
		f, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0600)
		if err == nil {
			f.Close()
		}
		t.Fatal("worktree enumeration blocked on FIFO")
	}
}

func TestCodexGitAliasedAdminCannotInheritPrimaryTrust(t *testing.T) {
	primary, linked, admin := codexTestWorktrees(t)
	alias := admin + "-alias"
	if err := os.Symlink(admin, alias); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: "+alias+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, q, _, _ := codexTestConfig(t, "")
	q.CWD = linked
	q.Overrides = []string{`projects={"` + primary + `"={trust_level='trusted'}}`}
	folder := filepath.Join(linked, ".codex")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "config.toml"), []byte("forced_login_method='api'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status == CodexConfigComplete {
		t.Fatalf("aliased admin inherited native-rejected trust: %+v", p)
	}
}

func TestCodexGitUnprovenDiscoveryNeverShrinksCompleteScope(t *testing.T) {
	for _, kind := range []string{"current-admin-alias", "linked-admin-alias", "linked-backlink-alias", "linked-common-alias", "primary-git-alias", "linked-nonregular-git", "linked-incomplete-metadata", "linked-permission-error"} {
		t.Run(kind, func(t *testing.T) {
			primary, linked, admin := codexTestWorktrees(t)
			r, q, _, _ := codexTestConfig(t, "forced_login_method='api'\n")
			q.CWD = filepath.Join(primary, "sub")
			write := func(p, s string) {
				t.Helper()
				if err := os.WriteFile(p, []byte(s), 0600); err != nil {
					t.Fatal(err)
				}
			}
			symlink := func(target, p string) {
				t.Helper()
				if err := os.Symlink(target, p); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "current-admin-alias", "linked-admin-alias":
				alias := admin + "-alias"
				symlink(admin, alias)
				write(filepath.Join(linked, ".git"), "gitdir: "+alias+"\n")
				if kind == "current-admin-alias" {
					q.CWD = filepath.Join(linked, "sub")
				}
			case "linked-backlink-alias":
				alias := linked + "-alias"
				symlink(linked, alias)
				write(filepath.Join(admin, "gitdir"), filepath.Join(alias, ".git")+"\n")
			case "linked-common-alias":
				alias := filepath.Join(primary, "common-alias")
				symlink(filepath.Join(primary, ".git"), alias)
				write(filepath.Join(admin, "commondir"), alias+"\n")
			case "primary-git-alias":
				original := filepath.Join(primary, ".git")
				backup := filepath.Join(primary, "admin-backup")
				if err := os.Rename(original, backup); err != nil {
					t.Fatal(err)
				}
				symlink(backup, original)
				q.CWD = filepath.Join(linked, "sub")
			case "linked-nonregular-git":
				p := filepath.Join(linked, ".git")
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(p, 0600); err != nil {
					t.Fatal(err)
				}
			case "linked-incomplete-metadata":
				if err := os.Remove(filepath.Join(admin, "commondir")); err != nil {
					t.Fatal(err)
				}
			case "linked-permission-error":
				if os.Geteuid() == 0 {
					t.Skip("root can bypass fixture permissions")
				}
				if err := os.Chmod(linked, 0); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(linked, 0700); err != nil {
						t.Error(err)
					}
				})
			}
			p, err := readCodexSelectionConfig(context.Background(), q, r)
			if err != nil {
				t.Fatal(err)
			}
			if p.Status != CodexConfigCompatibility {
				t.Fatalf("unproven %s silently produced authoritative scope: %+v", kind, p)
			}
		})
	}
}
func TestCodexGitNativeProvenExclusionsAndNonGitRemainComplete(t *testing.T) {
	for _, kind := range []string{"plain-non-git", "missing-corresponding-directory", "non-directory-corresponding-path", "non-directory-registration"} {
		t.Run(kind, func(t *testing.T) {
			primary, linked, _ := codexTestWorktrees(t)
			r, q, _, _ := codexTestConfig(t, "forced_login_method='api'\n")
			want := 1
			if kind != "plain-non-git" {
				q.CWD = filepath.Join(primary, "sub")
			}
			switch kind {
			case "missing-corresponding-directory", "non-directory-corresponding-path":
				p := filepath.Join(linked, "sub")
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if kind == "non-directory-corresponding-path" {
					if err := os.WriteFile(p, nil, 0600); err != nil {
						t.Fatal(err)
					}
				}
			case "non-directory-registration":
				want = 2
				if err := os.WriteFile(filepath.Join(primary, ".git", "worktrees", "notes"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			p, err := readCodexSelectionConfig(context.Background(), q, r)
			if err != nil || p.Status != CodexConfigComplete || len(p.Scope.CWDs) != want {
				t.Fatalf("%s: %+v %v want %d directories", kind, p, err, want)
			}
		})
	}
}
