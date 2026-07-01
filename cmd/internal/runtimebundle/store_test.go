package runtimebundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreExtractsAssetsAndWritesMarker(t *testing.T) {
	dir := t.TempDir()
	shellContent := "pair shell\n"
	initContent := "init\n"
	manifest := RuntimeManifest{Assets: []RuntimeAsset{
		{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(shellContent)), Digest: digestFor(shellContent)},
		{Path: "nvim/init.lua", Mode: 0o644, Size: int64(len(initContent)), Digest: digestFor(initContent)},
	}}

	res, err := Extract(StoreInput{
		StoreRoot: dir,
		Manifest:  manifest,
		ReadAsset: fakeAssetReader(map[string]string{
			"bin/pair-shell": shellContent,
			"nvim/init.lua":  initContent,
		}),
		Keep: 1,
	})
	if err != nil {
		t.Fatalf("Extract error = %v", err)
	}
	shell := filepath.Join(res.PairHome, "bin", "pair-shell")
	got, err := os.ReadFile(shell)
	if err != nil {
		t.Fatalf("ReadFile(pair-shell) error = %v", err)
	}
	if string(got) != "pair shell\n" {
		t.Fatalf("pair-shell content = %q", got)
	}
	info, err := os.Stat(shell)
	if err != nil {
		t.Fatalf("Stat(pair-shell) error = %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("pair-shell mode = %o, want 755", info.Mode().Perm())
	}
	marker, err := os.ReadFile(filepath.Join(filepath.Dir(res.PairHome), "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile(marker) error = %v", err)
	}
	if len(marker) == 0 {
		t.Fatal("marker is empty")
	}
}

func TestStoreSecondExtractIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	content := "pair shell\n"
	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(content)), Digest: digestFor(content)}}}
	input := StoreInput{
		StoreRoot: dir,
		Manifest:  manifest,
		ReadAsset: fakeAssetReader(map[string]string{"bin/pair-shell": content}),
		Keep:      1,
	}
	first, err := Extract(input)
	if err != nil {
		t.Fatalf("first Extract error = %v", err)
	}
	second, err := Extract(input)
	if err != nil {
		t.Fatalf("second Extract error = %v", err)
	}
	if first.PairHome != second.PairHome {
		t.Fatalf("PairHome changed: %q != %q", first.PairHome, second.PairHome)
	}
}

func TestStoreCleanupPreservesSelectedRuntime(t *testing.T) {
	dir := t.TempDir()
	oldDigest := strings.Repeat("a", 64)
	old := filepath.Join(dir, oldDigest, "pair-home")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatalf("MkdirAll(old) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, oldDigest, "manifest.json"), []byte(`{"digest":"`+oldDigest+`","asset_count":0}`), 0o644); err != nil {
		t.Fatalf("WriteFile(old marker) error = %v", err)
	}
	content := "pair shell\n"
	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(content)), Digest: digestFor(content)}}}

	res, err := Extract(StoreInput{
		StoreRoot: dir,
		Manifest:  manifest,
		ReadAsset: fakeAssetReader(map[string]string{"bin/pair-shell": content}),
		Keep:      0,
	})
	if err != nil {
		t.Fatalf("Extract error = %v", err)
	}
	if _, err := os.Stat(res.PairHome); err != nil {
		t.Fatalf("selected runtime was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, oldDigest)); !os.IsNotExist(err) {
		t.Fatalf("old runtime still exists or stat failed unexpectedly: %v", err)
	}
}

func TestStoreCleanupIgnoresMarkerDigestMismatch(t *testing.T) {
	dir := t.TempDir()
	oldDigest := strings.Repeat("b", 64)
	if err := os.MkdirAll(filepath.Join(dir, oldDigest, "pair-home"), 0o755); err != nil {
		t.Fatalf("MkdirAll(old) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, oldDigest, "manifest.json"), []byte(`{"digest":"`+strings.Repeat("c", 64)+`","asset_count":0}`), 0o644); err != nil {
		t.Fatalf("WriteFile(old marker) error = %v", err)
	}
	content := "pair shell\n"
	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-shell", Mode: 0o755, Size: int64(len(content)), Digest: digestFor(content)}}}

	if _, err := Extract(StoreInput{
		StoreRoot: dir,
		Manifest:  manifest,
		ReadAsset: fakeAssetReader(map[string]string{"bin/pair-shell": content}),
		Keep:      0,
	}); err != nil {
		t.Fatalf("Extract error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, oldDigest)); err != nil {
		t.Fatalf("mismatched-marker runtime should be ignored, not deleted: %v", err)
	}
}

func fakeAssetReader(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		return []byte(files[path]), nil
	}
}
