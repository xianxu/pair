package runtimebundle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

type RuntimeAsset struct {
	Path   string
	Mode   uint32
	Size   int64
	Digest string
}

type RuntimeManifest struct {
	Assets []RuntimeAsset
}

func (m RuntimeManifest) Validate() error {
	seen := map[string]bool{}
	for _, asset := range m.Assets {
		if err := validateAsset(asset); err != nil {
			return err
		}
		if seen[asset.Path] {
			return fmt.Errorf("duplicate asset path %q", asset.Path)
		}
		seen[asset.Path] = true
	}
	return nil
}

func (m RuntimeManifest) ManifestDigest() (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	assets := append([]RuntimeAsset(nil), m.Assets...)
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Path < assets[j].Path
	})
	h := sha256.New()
	for _, asset := range assets {
		_, _ = fmt.Fprintf(h, "%s\x00%o\x00%d\x00%s\x00", asset.Path, asset.Mode, asset.Size, asset.Digest)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validateAsset(asset RuntimeAsset) error {
	if asset.Path == "" {
		return errors.New("asset path is empty")
	}
	if strings.HasPrefix(asset.Path, "/") {
		return fmt.Errorf("asset path %q is absolute", asset.Path)
	}
	clean := path.Clean(asset.Path)
	if clean != asset.Path || clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return fmt.Errorf("asset path %q is not clean relative path", asset.Path)
	}
	if asset.Digest == "" {
		return fmt.Errorf("asset %q digest is empty", asset.Path)
	}
	return nil
}

func digestFor(s string) string {
	h := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(h[:])
}
