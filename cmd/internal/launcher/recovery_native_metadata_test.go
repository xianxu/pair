package launcher

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

func TestRecoveryNativeOwnerLedgerExceedsSmallMetadataBound(t *testing.T) {
	w, check := nativeOwnerFixture(t, "codex", "other")
	owner, _ := artifactpath.NewStorageOwner(check.DataDir, check.RepoScope, "other")
	paths, _ := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
	if err := os.MkdirAll(owner.Directory(), 0700); err != nil {
		t.Fatal(err)
	}
	// Valid JSON whitespace makes a large ledger without excessive fixture
	// structure. Ledger writers have no 8 MiB file cap (#508).
	raw := append(bytes.TrimSuffix(w.raw, []byte{'\n'}), bytes.Repeat([]byte{' '}, 9<<20)...)
	raw = append(raw, '\n')
	if err := os.WriteFile(paths.Ledger(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := readiness.Encode(w.ready)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(AgentReadyPath(owner.Directory(), owner.Tag, "codex"), []byte(r), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkRecoveryNativeOwnersWithDiscovery(context.Background(), check, w.Observe, readNativeOwnerMetadata, noOrphanOwners); err != nil {
		t.Fatal(err)
	}
	// Special-file or cancelled reads remain unknown; no boundary relaxation.
	link := filepath.Join(owner.Directory(), "link.jsonl")
	if err := os.Symlink(paths.Ledger(), link); err != nil {
		t.Fatal(err)
	}
	if _, err := readNativeOwnerMetadata(context.Background(), link, -1); err == nil {
		t.Fatal("ledger symlink accepted")
	}
}
