package testpair

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/internal/reporoot"
)

func TestEnvironmentStartsEmptyAndUsesDisposableHome(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(t.TempDir(), "events")
	if err := os.WriteFile(foreign, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_SCROLLBACK_EVENTS_PATH", foreign)
	t.Setenv("COUCH_TREE", "foreign-owner")
	t.Setenv("UNLISTED_TEST_SECRET", "never-inherit")
	env := Environment(t, root, "FIXTURE_VALUE=explicit", "PAIR_SCROLLBACK_EVENTS_PATH="+filepath.Join(root, "events"))
	values := map[string]string{}
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		values[key] = value
	}
	if values["HOME"] != root || values["PATH"] != os.Getenv("PATH") || values["FIXTURE_VALUE"] != "explicit" {
		t.Fatal("environment defaults/explicit values differ")
	}
	for _, key := range []string{"COUCH_TREE", "UNLISTED_TEST_SECRET"} {
		if _, ok := values[key]; ok {
			t.Errorf("inherited %s", key)
		}
	}
	cmd := exec.Command("/bin/sh", "-c", `printf local > "$PAIR_SCROLLBACK_EVENTS_PATH"`)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("disposable write: %v %s", err, out)
	}
	if got, _ := os.ReadFile(foreign); string(got) != "preserve" {
		t.Fatal("foreign capture changed")
	}
	if got, _ := os.ReadFile(filepath.Join(root, "events")); string(got) != "local" {
		t.Fatal("disposable capture not written")
	}
}

func TestEnvironmentShellEntryUsesSameEmptyAuthority(t *testing.T) {
	root := t.TempDir()
	repo, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("/bin/sh", filepath.Join(repo, "tests/lib/pair-test-env.sh"), root, "FIXTURE_VALUE=literal space $(never-run)", "--", "/bin/sh", "-c", `[ "$HOME" = "$1" ] && [ -z "${PAIR_SCROLLBACK_EVENTS_PATH+x}" ] && [ "$FIXTURE_VALUE" = 'literal space $(never-run)' ]`, "fixture", root)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/foreign", "PAIR_SCROLLBACK_EVENTS_PATH=/foreign/events"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shell empty environment/literal argv: %v %s", err, out)
	}
}

func TestApplyEnvironmentReplacesAndRestoresProcessEnvironment(t *testing.T) {
	t.Setenv("PAIR_TEST_FOREIGN_AUTHORITY", "foreign")
	before := os.Getenv("HOME")
	root := t.TempDir()
	t.Run("disposable", func(t *testing.T) {
		ApplyEnvironment(t, root, "FIXTURE_EXPLICIT=yes")
		if _, ok := os.LookupEnv("PAIR_TEST_FOREIGN_AUTHORITY"); ok {
			t.Fatal("ambient authority survived replacement")
		}
		if os.Getenv("HOME") != root || os.Getenv("FIXTURE_EXPLICIT") != "yes" {
			t.Fatal("disposable environment missing")
		}
	})
	if os.Getenv("PAIR_TEST_FOREIGN_AUTHORITY") != "foreign" || os.Getenv("HOME") != before {
		t.Fatal("process environment not restored")
	}
}
