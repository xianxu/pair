package titlepoller

import (
	"github.com/xianxu/pair/cmd/internal/testpair"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTitleRuntimeRequiresExactLaunchOrdinal(t *testing.T) {
	for _, ordinal := range []string{"", "0", "01", "+1", "-1", "18446744073709551616", "1 extra"} {
		t.Run(ordinal, func(t *testing.T) {
			t.Setenv("PAIR_LAUNCH_ORDINAL", ordinal)
			if NewOSRuntime().Err() == nil {
				t.Fatal("unproven launch ordinal admitted")
			}
		})
	}
	t.Setenv("PAIR_LAUNCH_ORDINAL", "1")
	if err := titlePublicationRuntime(t, t.TempDir()).Err(); err != nil {
		t.Fatalf("exact launch rejected: %v", err)
	}
}

func TestTitlePublicationsReplaceWholeFile(t *testing.T) {
	for _, name := range []string{"title-pid-fixture", "cmux-owner-fixture"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path, prior := filepath.Join(root, name), filepath.Join(root, "prior-inode")
			if err := os.WriteFile(path, []byte("prior generation\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Link(path, prior); err != nil {
				t.Fatal(err)
			}
			if err := titlePublicationRuntime(t, root).WriteFile(path, "successor generation\n"); err != nil {
				t.Fatal(err)
			}
			if raw, err := os.ReadFile(prior); err != nil || string(raw) != "prior generation\n" {
				t.Fatalf("publication modified previously visible inode: %q, %v", raw, err)
			}
			if raw, err := os.ReadFile(path); err != nil || string(raw) != "successor generation\n" {
				t.Fatalf("publication missing complete successor: %q, %v", raw, err)
			}
		})
	}
}

func titlePublicationRuntime(t *testing.T, root string) OSRuntime {
	t.Helper()
	testpair.ApplyEnvironment(t, t.TempDir(), "PAIR_DATA_DIR="+root, "PAIR_TAG=fixture", "PAIR_LAUNCH_ORDINAL=1", "CMUX_WORKSPACE_ID=fixture")
	rt := NewOSRuntime()
	if err := rt.Err(); err != nil {
		t.Fatal(err)
	}
	return rt
}

func TestTitlePublicationRenameFailureLeavesNoUnclassifiedSibling(t *testing.T) {
	for _, name := range []string{"title-pid-fixture", "cmux-owner-fixture"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			rt := titlePublicationRuntime(t, root)
			path := filepath.Join(root, name)
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			if err := rt.WriteFile(path, "unpublished successor\n"); err == nil {
				t.Fatal("directory target accepted")
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), name+".") {
					t.Fatalf("unrecoverable publication residue: %s", entry.Name())
				}
			}
		})
	}
}

func TestTitlePublicationRefusesOtherDestinations(t *testing.T) {
	root := t.TempDir()
	rt := titlePublicationRuntime(t, root)
	for _, path := range []string{filepath.Join(root, "title-pid-other"), filepath.Join(root, "cmux-owner-other"), filepath.Join(t.TempDir(), "title-pid-fixture")} {
		if err := rt.WriteFile(path, "wrong owner\n"); err == nil {
			t.Errorf("published outside configured title paths: %s", path)
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("rejected path received bytes: %s: %v", path, err)
		}
	}
}
