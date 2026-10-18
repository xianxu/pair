package testgit

import (
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunWithEmptyGitTemplate(t *testing.T) {
	ambient := t.TempDir()
	if err := os.Mkdir(filepath.Join(ambient, "hooks"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ambient, "hooks", "ambient.sample"), []byte("ambient"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_TEMPLATE_DIR", ambient)
	for _, status := range []int{0, 7} {
		var owned string
		got := RunWithEmptyGitTemplate(func() int {
			owned = os.Getenv("GIT_TEMPLATE_DIR")
			if owned == ambient || owned == "" {
				t.Fatalf("template=%q", owned)
			}
			entries, err := os.ReadDir(owned)
			if err != nil || len(entries) != 0 {
				t.Fatalf("empty template: %v %v", entries, err)
			}
			repo := filepath.Join(t.TempDir(), "repo")
			cmd := exec.Command("git", "init", "-q", repo)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git init: %v %s", err, out)
			}
			if _, err := os.Stat(filepath.Join(repo, ".git", "hooks")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("hooks materialized: %v", err)
			}
			return status
		})
		if got != status || os.Getenv("GIT_TEMPLATE_DIR") != ambient {
			t.Fatalf("status=%d template=%q", got, os.Getenv("GIT_TEMPLATE_DIR"))
		}
		if _, err := os.Stat(owned); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("template remains: %v", err)
		}
		if _, err := os.Stat(filepath.Join(ambient, "hooks", "ambient.sample")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunWithEmptyGitTemplatePanicAndUnset(t *testing.T) {
	old, present := os.LookupEnv("GIT_TEMPLATE_DIR")
	_ = os.Unsetenv("GIT_TEMPLATE_DIR")
	defer func() {
		if present {
			_ = os.Setenv("GIT_TEMPLATE_DIR", old)
		} else {
			_ = os.Unsetenv("GIT_TEMPLATE_DIR")
		}
	}()
	var owned string
	func() {
		defer func() {
			if recover() != "boom" {
				t.Error("panic not preserved")
			}
		}()
		RunWithEmptyGitTemplate(func() int { owned = os.Getenv("GIT_TEMPLATE_DIR"); panic("boom") })
	}()
	if _, ok := os.LookupEnv("GIT_TEMPLATE_DIR"); ok {
		t.Error("environment not restored to unset")
	}
	if _, err := os.Stat(owned); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("panic template remains: %v", err)
	}
}

func TestRunWithEmptyGitTemplateSetupFailure(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	called := false
	if got := RunWithEmptyGitTemplate(func() int { called = true; return 0 }); got == 0 || called {
		t.Fatalf("got=%d callback=%v", got, called)
	}
}

func TestFixtureTreeBudget(t *testing.T) {
	cap := FixtureTreeBudget{Files: 2, Dirs: 3, LogicalBytes: 5, SampleHooks: 1}
	equal := FixtureTreeStats{Files: 2, Dirs: 3, LogicalBytes: 5, SampleHooks: 1}
	if err := cap.Verdict(equal); err != nil {
		t.Fatal(err)
	}
	for _, got := range []FixtureTreeStats{{Files: 3}, {Dirs: 4}, {LogicalBytes: 6}, {SampleHooks: 2}} {
		if cap.Verdict(got) == nil {
			t.Errorf("accepted %+v", got)
		}
	}
	if err := (FixtureTreeBudget{}).Verdict(FixtureTreeStats{}); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]FixtureTreeStats{
		{{Files: math.MaxUint64}, {Files: 1}}, {{Dirs: math.MaxUint64}, {Dirs: 1}}, {{LogicalBytes: math.MaxUint64}, {LogicalBytes: 1}}, {{SampleHooks: math.MaxUint64}, {SampleHooks: 1}},
	} {
		if _, err := pair[0].Add(pair[1]); err == nil {
			t.Errorf("overflow accepted: %+v", pair)
		}
	}
}

func TestAuditFixtureTree(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "hooks"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hooks", "x.sample"), []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "empty"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := AuditFixtureTree(root)
	want := FixtureTreeStats{Files: 2, Dirs: 2, LogicalBytes: 5, SampleHooks: 1}
	if err != nil || got != want {
		t.Fatalf("got=%+v err=%v want=%+v", got, err, want)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := AuditFixtureTree(root); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := AuditFixtureTree(filepath.Join(root, "alias")); err == nil {
		t.Fatal("root symlink accepted")
	}
	if raw, err := os.ReadFile(sentinel); err != nil || string(raw) != "outside" {
		t.Fatalf("outside changed: %q %v", raw, err)
	}
	if _, err := AuditFixtureTree(filepath.Join(root, "empty")); err == nil {
		t.Fatal("regular file root accepted")
	}
	if _, err := AuditFixtureTree(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing root accepted")
	}
}

func TestFixtureTreeStatsAdd(t *testing.T) {
	a := FixtureTreeStats{Files: 2, Dirs: 3, LogicalBytes: 5, SampleHooks: 7}
	b := FixtureTreeStats{Files: 11, Dirs: 13, LogicalBytes: 17, SampleHooks: 19}
	want := FixtureTreeStats{Files: 13, Dirs: 16, LogicalBytes: 22, SampleHooks: 26}
	if got, err := a.Add(b); err != nil || got != want {
		t.Fatalf("sum=%+v err=%v want=%+v", got, err, want)
	}
	if got, err := a.Add(FixtureTreeStats{}); err != nil || got != a {
		t.Fatalf("identity=%+v err=%v", got, err)
	}
}
