package runtimebundle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type StoreInput struct {
	StoreRoot string
	Manifest  RuntimeManifest
	ReadAsset func(string) ([]byte, error)
	Keep      int
}

type StoreResult struct {
	Digest   string
	PairHome string
}

type runtimeMarker struct {
	Digest     string `json:"digest"`
	AssetCount int    `json:"asset_count"`
	Generated  string `json:"generated,omitempty"`
}

func Extract(input StoreInput) (StoreResult, error) {
	if input.ReadAsset == nil {
		return StoreResult{}, fmt.Errorf("asset reader is required")
	}
	digest, err := input.Manifest.ManifestDigest()
	if err != nil {
		return StoreResult{}, err
	}
	pairHome := filepath.Join(input.StoreRoot, digest, "pair-home")
	existing, err := scanExisting(pairHome, input.Manifest)
	if err != nil {
		return StoreResult{}, err
	}
	plan, err := PlanExtraction(ExtractionInput{
		StoreRoot:   input.StoreRoot,
		RuntimeRoot: pairHome,
		Manifest:    input.Manifest,
		Existing:    existing,
	})
	if err != nil {
		return StoreResult{}, err
	}
	for _, asset := range plan.Writes {
		data, err := input.ReadAsset(asset.Path)
		if err != nil {
			return StoreResult{}, fmt.Errorf("read embedded asset %s: %w", asset.Path, err)
		}
		if digestFor(string(data)) != asset.Digest {
			// Binary assets may not be valid UTF-8 but string preserves bytes.
			return StoreResult{}, fmt.Errorf("embedded asset %s digest mismatch", asset.Path)
		}
		if int64(len(data)) != asset.Size {
			return StoreResult{}, fmt.Errorf("embedded asset %s size mismatch", asset.Path)
		}
		if err := writeFileAtomic(filepath.Join(pairHome, filepath.FromSlash(asset.Path)), data, os.FileMode(asset.Mode)); err != nil {
			return StoreResult{}, err
		}
	}
	if err := writeMarker(filepath.Join(input.StoreRoot, digest, "manifest.json"), digest, len(input.Manifest.Assets)); err != nil {
		return StoreResult{}, err
	}
	if err := applyCleanup(input.StoreRoot, digest, input.Keep); err != nil {
		return StoreResult{}, err
	}
	return StoreResult{Digest: digest, PairHome: pairHome}, nil
}

func scanExisting(root string, manifest RuntimeManifest) (map[string]ExistingAsset, error) {
	existing := map[string]ExistingAsset{}
	for _, asset := range manifest.Assets {
		p := filepath.Join(root, filepath.FromSlash(asset.Path))
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(data)
		existing[asset.Path] = ExistingAsset{
			Mode:   uint32(info.Mode().Perm()),
			Size:   info.Size(),
			Digest: "sha256:" + hex.EncodeToString(h[:]),
		}
	}
	return existing, nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	committed = true
	return nil
}

func writeMarker(path, digest string, assetCount int) error {
	data, err := json.MarshalIndent(runtimeMarker{
		Digest:     digest,
		AssetCount: assetCount,
		Generated:  time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}

func applyCleanup(storeRoot, selectedDigest string, keep int) error {
	entries, err := os.ReadDir(storeRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	gens := make([]RuntimeGeneration, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		info, err := entry.Info()
		if err != nil {
			return err
		}
		gens = append(gens, RuntimeGeneration{
			Digest:    name,
			HasMarker: markerValid(filepath.Join(storeRoot, name, "manifest.json"), name),
			ModUnix:   info.ModTime().Unix(),
		})
	}
	plan, err := PlanCleanup(CleanupInput{SelectedDigest: selectedDigest, Keep: keep, Generations: gens})
	if err != nil {
		return err
	}
	sort.Strings(plan.DeleteDigests)
	for _, digest := range plan.DeleteDigests {
		if err := os.RemoveAll(filepath.Join(storeRoot, digest)); err != nil {
			return err
		}
	}
	return nil
}

func markerValid(path, digest string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var marker runtimeMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return false
	}
	return marker.Digest == digest
}
