package titlepoller

import (
	"os"
	"path/filepath"
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
	if err := NewOSRuntime().Err(); err != nil {
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
			if err := (OSRuntime{}).WriteFile(path, "successor generation\n"); err != nil {
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
