package runtimebundlegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/runtimebundle/manifestmodel"
)

// explicitAssetPaths lists the individual files bundled into the extracted
// runtime. Since #104 M3 it carries NO helper binaries — every former helper is
// a `pair <subcommand>` reached via the single `pair` on the session PATH (the
// launcher fronts pair's own dir, cf. launcher/pathenv.go). Only the two shell
// shims (invoked by bare name inside a session) and the doctor assets remain;
// `pair` itself is never bundled (no self-embed).
var explicitAssetPaths = []string{
	"bin/pair-help",
	"bin/pair-notify",
	"doctor/README.md",
	"doctor/SKILL.md",
	"doctor/doctor.sh",
	"doctor/perf.sh",
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
	Compiler TerminfoCompiler
}

// TerminfoCompiler writes compiled entries beneath a caller-owned temporary
// directory. Generate validates and normalizes that tree before publication.
type TerminfoCompiler interface {
	Compile(source, destination string) error
}

type TicCompiler struct{}

func (TicCompiler) Compile(source, destination string) error {
	compiler := exec.Command("tic", "-x", "-o", destination, source)
	if output, err := compiler.CombinedOutput(); err != nil {
		return fmt.Errorf("tic: %w: %s", err, output)
	}
	return nil
}

func Generate(opts GenerateOptions) (manifestmodel.RuntimeManifest, error) {
	if opts.RepoRoot == "" {
		opts.RepoRoot = "."
	}
	if opts.OutRoot == "" {
		return manifestmodel.RuntimeManifest{}, fmt.Errorf("output root is required")
	}
	repoRoot, err := filepath.Abs(opts.RepoRoot)
	if err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	outRoot, err := filepath.Abs(opts.OutRoot)
	if err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	outParent := filepath.Dir(outRoot)
	outBase := filepath.Base(outRoot)
	if err := os.MkdirAll(outParent, 0o755); err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	stageRoot, err := os.MkdirTemp(outParent, "."+outBase+"-tmp-")
	if err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(stageRoot)
		}
	}()
	filesRoot := filepath.Join(stageRoot, "files")

	compiledRoot, err := os.MkdirTemp(outParent, ".pair-terminfo-")
	if err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	defer os.RemoveAll(compiledRoot)
	compiler := opts.Compiler
	if compiler == nil {
		compiler = TicCompiler{}
	}
	if err := compiler.Compile(filepath.Join(repoRoot, "terminfo", "pair-vt-256color.ti"), compiledRoot); err != nil {
		return manifestmodel.RuntimeManifest{}, fmt.Errorf("compile terminal profile: %w", err)
	}
	generated := map[string]string{}
	if err := filepath.WalkDir(compiledRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(compiledRoot, path)
		if err != nil {
			return err
		}
		// ncurses accepts hex directory names on every supported host; normalize
		// tic's platform-specific p/ versus70/ output for reproducible manifests.
		if rel != filepath.Join("p", "pair-vt-256color") && rel != filepath.Join("70", "pair-vt-256color") {
			return fmt.Errorf("unexpected compiled terminal entry %s", rel)
		}
		if !d.Type().IsRegular() || len(generated) != 0 {
			return fmt.Errorf("invalid or duplicate compiled terminal entry %s", rel)
		}
		generated["terminfo/70/pair-vt-256color"] = path
		return nil
	}); err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	if len(generated) != 1 {
		return manifestmodel.RuntimeManifest{}, fmt.Errorf("compiler did not produce pair-vt-256color")
	}
	paths := map[string]bool{}
	for logical := range generated {
		paths[logical] = true
	}
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
			return manifestmodel.RuntimeManifest{}, err
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

	manifest := manifestmodel.RuntimeManifest{Assets: make([]manifestmodel.RuntimeAsset, 0, len(ordered))}
	for _, logical := range ordered {
		src := filepath.Join(repoRoot, filepath.FromSlash(logical))
		if compiled, ok := generated[logical]; ok {
			src = compiled
		}
		info, err := os.Stat(src)
		if err != nil {
			return manifestmodel.RuntimeManifest{}, fmt.Errorf("asset %s: %w", logical, err)
		}
		if info.IsDir() {
			continue
		}
		digest, err := copyAsset(src, filepath.Join(filesRoot, filepath.FromSlash(logical)), info.Mode().Perm())
		if err != nil {
			return manifestmodel.RuntimeManifest{}, err
		}
		manifest.Assets = append(manifest.Assets, manifestmodel.RuntimeAsset{
			Path:   logical,
			Mode:   uint32(info.Mode().Perm()),
			Size:   info.Size(),
			Digest: "sha256:" + digest,
		})
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(stageRoot, "manifest.json"), encoded, 0o644); err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	unlock, err := acquirePublishLock(outRoot + ".lock")
	if err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	defer unlock()
	if err := os.RemoveAll(outRoot); err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	if err := os.Rename(stageRoot, outRoot); err != nil {
		return manifestmodel.RuntimeManifest{}, err
	}
	committed = true
	return manifest, nil
}

func acquirePublishLock(path string) (func(), error) {
	const attempts = 1000
	for i := 0; i < attempts; i++ {
		err := os.Mkdir(path, 0o755)
		if err == nil {
			return func() { _ = os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, fmt.Errorf("timed out waiting for runtime bundle publish lock %s", path)
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
