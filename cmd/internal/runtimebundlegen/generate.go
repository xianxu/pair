package runtimebundlegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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

var explicitAssetPaths = []string{
	"bin/pair-shell",
	"bin/pair-help",
	"bin/pair-notify",
	"bin/pair-quit.sh",
	"bin/pair-restart.sh",
	"bin/pair-scrollback-open",
	"bin/pair-changelog-open",
	"bin/pair-review-open",
	"bin/pair-review-readiness",
	"bin/pair-review-target",
	"bin/copy-on-select.sh",
	"bin/clipboard-to-pane.sh",
	"bin/flash-pane.sh",
	"bin/pair-title.sh",
	"bin/pair-session-watch.sh",
	"bin/pair-wrap",
	"bin/pair-slug",
	"bin/pair-context",
	"bin/pair-scrollback-render",
	"bin/pair-changelog",
	"bin/pair-continuation",
	"bin/pair-session-watch",
	"doctor/README.md",
	"doctor/SKILL.md",
	"doctor/doctor.sh",
	"doctor/emitter-health.sh",
}

var assetDirs = []string{
	"bin/lib",
	"nvim",
	"zellij",
}

type GenerateOptions struct {
	RepoRoot string
	OutRoot  string
}

func Generate(opts GenerateOptions) (RuntimeManifest, error) {
	if opts.RepoRoot == "" {
		opts.RepoRoot = "."
	}
	if opts.OutRoot == "" {
		return RuntimeManifest{}, fmt.Errorf("output root is required")
	}
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return RuntimeManifest{}, err
	}
	outRoot, err := filepath.Abs(opts.OutRoot)
	if err != nil {
		return RuntimeManifest{}, err
	}
	outParent := filepath.Dir(outRoot)
	outBase := filepath.Base(outRoot)
	if err := os.MkdirAll(outParent, 0o755); err != nil {
		return RuntimeManifest{}, err
	}
	stageRoot, err := os.MkdirTemp(outParent, "."+outBase+"-tmp-")
	if err != nil {
		return RuntimeManifest{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(stageRoot)
		}
	}()
	filesRoot := filepath.Join(stageRoot, "files")

	paths := map[string]bool{}
	for _, p := range explicitAssetPaths {
		paths[p] = true
	}
	for _, dir := range assetDirs {
		root := filepath.Join(repoRoot, filepath.FromSlash(dir))
		if err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(repoRoot, p)
			if err != nil {
				return err
			}
			logical := filepath.ToSlash(rel)
			if shouldExclude(logical) {
				return nil
			}
			paths[logical] = true
			return nil
		}); err != nil {
			return RuntimeManifest{}, err
		}
	}

	ordered := make([]string, 0, len(paths))
	for p := range paths {
		if shouldExclude(p) {
			continue
		}
		ordered = append(ordered, p)
	}
	sort.Strings(ordered)

	manifest := RuntimeManifest{Assets: make([]RuntimeAsset, 0, len(ordered))}
	for _, logical := range ordered {
		src := filepath.Join(repoRoot, filepath.FromSlash(logical))
		info, err := os.Stat(src)
		if err != nil {
			return RuntimeManifest{}, fmt.Errorf("asset %s: %w", logical, err)
		}
		if info.IsDir() {
			continue
		}
		digest, err := copyAsset(src, filepath.Join(filesRoot, filepath.FromSlash(logical)), info.Mode().Perm())
		if err != nil {
			return RuntimeManifest{}, err
		}
		manifest.Assets = append(manifest.Assets, RuntimeAsset{
			Path:   logical,
			Mode:   uint32(info.Mode().Perm()),
			Size:   info.Size(),
			Digest: "sha256:" + digest,
		})
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return RuntimeManifest{}, err
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(stageRoot, "manifest.json"), encoded, 0o644); err != nil {
		return RuntimeManifest{}, err
	}
	if err := os.RemoveAll(outRoot); err != nil {
		return RuntimeManifest{}, err
	}
	if err := os.Rename(stageRoot, outRoot); err != nil {
		return RuntimeManifest{}, err
	}
	committed = true
	return manifest, nil
}

func shouldExclude(logical string) bool {
	base := filepath.Base(logical)
	if base == ".DS_Store" || strings.Contains(logical, "__pycache__/") {
		return true
	}
	if strings.HasSuffix(logical, "_test.lua") {
		return true
	}
	switch logical {
	case "bin/pair", "bin/pair-go", "bin/pair-dev":
		return true
	}
	return false
}

func copyAsset(src, dst string, mode os.FileMode) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(out, h), in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return "", closeErr
	}
	if err := os.Chmod(tmp, mode); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
