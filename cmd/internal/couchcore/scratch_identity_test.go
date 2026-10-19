package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func scratchAuthority(t *testing.T, store *ThreadStore, roots map[string]string, families []RepositoryFamily) {
	t.Helper()
	manifest := threadManifest{SchemaVersion: 2, Threads: []ThreadAddress{}, SlotRepositoryIdentities: roots, RepositoryFamilies: families}
	for root := range roots {
		manifest.SlotRepositories = append(manifest.SlotRepositories, root)
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.manifestPath(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.root, "store.lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestScratchSlotRetainedNumberedAndPrimaryWithoutCheckout(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, 3} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			root := filepath.Join(t.TempDir(), "pair")
			identity := filepath.Join(root, ".git")
			scratchAuthority(t, store, map[string]string{root: identity}, nil)
			checkout := root
			if n != 0 {
				checkout = conventionalSlot(root, n).WorktreeRoot
			}
			record := recordAtCheckout(t, checkout, filepath.Join(checkout, "subdir"), "scratch")
			got, err := store.ScratchSlot(t.Context(), record)
			want := ScratchSlotIdentity{Repo: "pair", RepoIdentity: identity, Number: n}
			if err != nil || got != want {
				t.Fatalf("ScratchSlot = %+v, %v; want %+v", got, err, want)
			}
			if _, err := os.Stat(checkout); !os.IsNotExist(err) {
				t.Fatalf("fixture checkout should not exist: %v", err)
			}
		})
	}
}

func TestScratchSlotPrimaryFamilyAndSameBasenameStayDistinct(t *testing.T) {
	t.Parallel()
	store, _ := newTestThreadStore(t)
	base := t.TempDir()
	roots := []string{filepath.Join(base, "first", "pair"), filepath.Join(base, "second", "pair")}
	families := []RepositoryFamily{}
	for _, root := range roots {
		families = append(families, RepositoryFamily{PrimaryRoot: root, RepoIdentity: filepath.Join(root, ".git"), RelativeStart: "."})
	}
	scratchAuthority(t, store, nil, families)
	var previous ScratchSlotIdentity
	for _, root := range roots {
		record := recordAtCheckout(t, root, root, "scratch")
		got, err := store.ScratchSlot(t.Context(), record)
		if err != nil || got.Repo != "pair" || got.Number != 0 || got.RepoIdentity != filepath.Join(root, ".git") {
			t.Fatalf("ScratchSlot = %+v,%v", got, err)
		}
		if previous.RepoIdentity == got.RepoIdentity {
			t.Fatal("same-basename repository identities collided")
		}
		previous = got
	}
}

func TestScratchSlotRefusesUnknownOrContradictoryAuthority(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"unknown", "wrong-scope", "wrong-repository", "conflicting-manifest"} {
		t.Run(mode, func(t *testing.T) {
			store, _ := newTestThreadStore(t)
			root := filepath.Join(t.TempDir(), "pair")
			record := recordAtCheckout(t, root, root, "scratch")
			roots := map[string]string{root: filepath.Join(root, ".git")}
			var families []RepositoryFamily
			switch mode {
			case "unknown":
				roots = nil
			case "wrong-scope":
				record = recordAtCheckout(t, filepath.Dir(root), root, "scratch")
			case "wrong-repository":
				record.Incarnations = []ThreadIncarnation{{State: IncarnationUnknown, RepoIdentity: filepath.Join(root, "foreign.git")}}
			case "conflicting-manifest":
				families = []RepositoryFamily{{PrimaryRoot: root, RepoIdentity: filepath.Join(root, "foreign.git"), RelativeStart: "."}}
			}
			scratchAuthority(t, store, roots, families)
			if got, err := store.ScratchSlot(t.Context(), record); err == nil {
				t.Fatalf("unverified owner accepted: %+v", got)
			}
		})
	}
}

func TestScratchSlotHonorsCancelledObservation(t *testing.T) {
	store, _ := newTestThreadStore(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := store.ScratchSlot(ctx, validThreadRecord(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled lookup=%v", err)
	}
}
