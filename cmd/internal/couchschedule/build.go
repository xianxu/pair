package couchschedule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"golang.org/x/sys/unix"
)

type Artifact struct {
	Path, Key string
	lease     *artifactLease
}

// Close releases the cache lifetime lease after execution and child cleanup.
func (a Artifact) Close() error {
	if a.lease == nil {
		return nil
	}
	return a.lease.Close()
}

type Builder struct {
	CacheDir, GoBinary string
	Run                func(context.Context, string, []string, []string) ([]byte, error)
	AfterCapture       func()
	CacheBytes         int64
	cacheNow           func() time.Time
	cacheFault         func(string) error
}

func (b Builder) Build(ctx context.Context, hubRoot, task string) (Artifact, error) {
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	if b.Run == nil {
		goBinary := b.GoBinary
		if goBinary == "" {
			goBinary = "go"
		}
		resolved, err := exec.LookPath(goBinary)
		if err != nil {
			return Artifact{}, err
		}
		b.GoBinary, err = filepath.Abs(resolved)
		if err != nil {
			return Artifact{}, err
		}
	}
	if !ValidTaskName(task) {
		return Artifact{}, errors.New("invalid scheduled task name")
	}
	root, err := filepath.Abs(hubRoot)
	if err != nil {
		return Artifact{}, err
	}
	if err = buildSafePath(root, root); err != nil {
		return Artifact{}, err
	}
	for _, unsupported := range []string{"vendor", "go.work"} {
		if _, e := os.Lstat(filepath.Join(root, unsupported)); e == nil {
			return Artifact{}, fmt.Errorf("scheduled builds refuse %s", unsupported)
		} else if !os.IsNotExist(e) {
			return Artifact{}, e
		}
	}
	mod, err := osfs.ReadRegularMetadata(ctx, filepath.Join(root, "go.mod"), 2<<20)
	if err != nil {
		return Artifact{}, err
	}
	stage, err := os.MkdirTemp("", "couch-build-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(stage)
	modfile := filepath.Join(stage, "capture.mod")
	if err = os.WriteFile(modfile, mod, 0600); err != nil {
		return Artifact{}, err
	}
	if sum, e := osfs.ReadRegularMetadata(ctx, filepath.Join(root, "go.sum"), 2<<20); e == nil {
		if err = os.WriteFile(filepath.Join(stage, "capture.sum"), sum, 0600); err != nil {
			return Artifact{}, err
		}
	} else if !os.IsNotExist(e) {
		return Artifact{}, e
	}
	env := buildEnvironment(false)
	toolchain, err := b.command(ctx, root, env, "env", "-json", "GOVERSION", "GOOS", "GOARCH", "GOROOT", "GOTOOLDIR")
	if err != nil {
		return Artifact{}, err
	}
	compiler, err := b.command(ctx, root, env, "tool", "compile", "-V=full")
	if err != nil {
		return Artifact{}, err
	}
	tool := string(toolchain) + string(compiler) + "\nrecipe=1;CGO_ENABLED=0;GOWORK=off;-trimpath;-buildvcs=false;target=" + runtime.GOOS + "/" + runtime.GOARCH + ";task=" + task
	moduleRaw, err := b.command(ctx, root, env, "list", "-m", "-json", "-mod=readonly", "-modfile="+modfile, "all")
	if err != nil {
		return Artifact{}, err
	}
	var modules []buildModule
	if err = decodeBuildStream(moduleRaw, func(d *json.Decoder) error {
		var m buildModule
		if e := d.Decode(&m); e != nil {
			return e
		}
		modules = append(modules, m)
		return nil
	}); err != nil {
		return Artifact{}, err
	}
	packageRaw, err := b.command(ctx, root, env, "list", "-deps", "-json", "-mod=readonly", "-modfile="+modfile, "./scripts/scheduled/"+task)
	if err != nil {
		return Artifact{}, err
	}
	var packages []buildPackage
	if err = decodeBuildStream(packageRaw, func(d *json.Decoder) error {
		var p buildPackage
		if e := d.Decode(&p); e != nil {
			return e
		}
		packages = append(packages, p)
		return nil
	}); err != nil {
		return Artifact{}, err
	}
	inputs, mapped, mainPath, err := captureBuild(ctx, root, modules, packages, mod)
	if err != nil {
		return Artifact{}, err
	}
	manifest := manifestFor(tool, inputs)
	if b.AfterCapture != nil {
		b.AfterCapture()
	}
	if err = ctx.Err(); err != nil {
		return Artifact{}, err
	}
	if b.CacheDir == "" {
		return Artifact{}, errors.New("build cache directory required")
	}
	cache, err := filepath.Abs(b.CacheDir)
	if err != nil {
		return Artifact{}, err
	}
	if err = durablefile.EnsureDirectory(cache); err != nil {
		return Artifact{}, err
	}
	if err = buildSafePath(cache, cache); err != nil {
		return Artifact{}, err
	}
	if _, err = cacheNames(cache); err != nil {
		return Artifact{}, err
	}
	unlock, err := buildLock(ctx, filepath.Join(cache, manifest.Key+".lock"))
	if err != nil {
		return Artifact{}, err
	}
	defer unlock()
	dir := filepath.Join(cache, manifest.Key)
	if _, err := os.Lstat(dir); err == nil {
		if err = buildSafePath(cache, dir); err != nil {
			return Artifact{}, err
		}
	} else if !os.IsNotExist(err) {
		return Artifact{}, err
	}
	if err = validateCacheShape(cache, dir); err != nil && !os.IsNotExist(err) {
		return Artifact{}, err
	}
	if a, ok := cachedBuild(ctx, dir, manifest); ok {
		lease, e := cacheLease(cache, manifest.Key, unix.LOCK_SH)
		if e != nil {
			return Artifact{}, e
		}
		a.lease = lease
		if e = b.maintainCache(ctx, cache, manifest, nil); e != nil {
			_ = a.Close()
			return Artifact{}, e
		}
		return a, nil
	}
	lease, err := cacheLease(cache, manifest.Key, unix.LOCK_EX)
	if err != nil {
		return Artifact{}, err
	}
	keepLease := false
	defer func() {
		if !keepLease {
			_ = lease.Close()
		}
	}()

	view := filepath.Join(stage, "view")
	for _, in := range inputs {
		p := filepath.Join(view, mapped[in.Module], filepath.FromSlash(in.Path))
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return Artifact{}, err
		}
		if err = os.WriteFile(p, in.Data, 0600); err != nil {
			return Artifact{}, err
		}
	}
	mainDir := filepath.Join(view, mapped[mainPath])
	// Remove all original replacements, including version-specific ones, then
	// map every graph module to its captured source directory.
	edits := []string{"mod", "edit"}
	var modInfo struct {
		Replace []struct {
			Old struct{ Path, Version string }
		}
	}
	info, err := b.command(ctx, mainDir, buildEnvironment(true), "mod", "edit", "-json")
	if err != nil {
		return Artifact{}, err
	}
	if err = json.Unmarshal(info, &modInfo); err != nil {
		return Artifact{}, err
	}
	for _, r := range modInfo.Replace {
		old := r.Old.Path
		if r.Old.Version != "" {
			old += "@" + r.Old.Version
		}
		edits = append(edits, "-dropreplace="+old)
	}
	for _, m := range modules {
		if m.Path == mainPath {
			continue
		}
		edits = append(edits, "-replace="+m.Path+"=../"+mapped[m.Path])
	}
	if len(edits) > 2 {
		if _, err = b.command(ctx, mainDir, buildEnvironment(true), edits...); err != nil {
			return Artifact{}, err
		}
	}
	binary := filepath.Join(stage, "task")
	if _, err = b.command(ctx, mainDir, buildEnvironment(true), "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-o", binary, "./scripts/scheduled/"+task); err != nil {
		return Artifact{}, err
	}
	raw, err := osfs.ReadRegularMetadata(ctx, binary, 256<<20)
	if err != nil {
		return Artifact{}, err
	}
	manifest.BinarySHA256 = digestBuild(raw)
	if err = b.maintainCache(ctx, cache, manifest, raw); err != nil {
		return Artifact{}, err
	}
	if err = unix.Flock(int(lease.file.Fd()), unix.LOCK_SH); err != nil {
		return Artifact{}, err
	}
	keepLease = true
	return Artifact{Path: filepath.Join(dir, "task"), Key: manifest.Key, lease: lease}, nil
}

type buildModule struct {
	Path, Version, Dir, GoMod string
	Main                      bool
	Replace                   *buildModule
	Error                     *struct{ Err string }
}
type buildPackage struct {
	Dir, Name                                                                      string
	Standard                                                                       bool
	Module                                                                         *buildModule
	GoFiles, SFiles, HFiles, EmbedFiles                                            []string
	CgoFiles, CFiles, CXXFiles, MFiles, FFiles, SwigFiles, SwigCXXFiles, SysoFiles []string
	Error                                                                          *struct{ Err string }
	Incomplete                                                                     bool
}

func decodeBuildStream(raw []byte, fn func(*json.Decoder) error) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	for {
		err := fn(d)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func captureBuild(ctx context.Context, root string, modules []buildModule, packages []buildPackage, rootMod []byte) ([]buildInput, map[string]string, string, error) {
	sort.Slice(modules, func(i, j int) bool { return modules[i].Path < modules[j].Path })
	mapped := map[string]string{}
	dirs := map[string]string{}
	seen := map[string]bool{}
	var inputs []buildInput
	total := 0
	mainPath := ""
	add := func(module, rel, path string, data []byte) error {
		key := module + "\x00" + rel
		if seen[key] {
			return nil
		}
		if data == nil {
			var err error
			data, err = osfs.ReadRegularMetadata(ctx, path, maxBuildInputBytes)
			if err != nil {
				return err
			}
		}
		if len(inputs) >= maxBuildInputFiles || len(data) > maxBuildInputBytes-total {
			return errors.New("build input capture exceeds bound")
		}
		total += len(data)
		seen[key] = true
		inputs = append(inputs, buildInput{module, filepath.ToSlash(rel), append([]byte(nil), data...)})
		return nil
	}
	for i, m := range modules {
		if m.Error != nil {
			return nil, nil, "", errors.New(m.Error.Err)
		}
		actual := m
		// A graph-only module may have metadata but no downloaded sources.
		// Packages resolve used modules; retain unused manifests in the view so
		// graph evaluation never needs the live module cache during compilation.
		if actual.Dir == "" {
			for _, p := range packages {
				if p.Module != nil && p.Module.Path == m.Path {
					actual = *p.Module
					break
				}
			}
		}
		if actual.Replace != nil {
			actual = *actual.Replace
			if actual.Version == "" && (filepath.IsAbs(actual.Path) || !buildWithin(root, actual.Dir)) {
				return nil, nil, "", errors.New("local module replacement escapes hub")
			}
		}
		if actual.GoMod == "" {
			return nil, nil, "", fmt.Errorf("module %s has no captured manifest", m.Path)
		}
		if actual.Dir != "" {
			if err := buildSafePath(actual.Dir, actual.Dir); err != nil {
				return nil, nil, "", err
			}
		}
		mapped[m.Path] = fmt.Sprintf("m%04d", i)
		dirs[m.Path] = actual.Dir
		var data []byte
		if m.Main {
			mainPath = m.Path
			data = rootMod
			dirs[m.Path] = root
		}
		moduleFile := actual.GoMod
		if actual.Dir != "" {
			moduleFile = filepath.Join(actual.Dir, "go.mod")
			if err := buildSafePath(actual.Dir, moduleFile); err != nil {
				return nil, nil, "", err
			}
		}
		if err := add(m.Path, "go.mod", moduleFile, data); err != nil {
			return nil, nil, "", err
		}
		if dirs[m.Path] == "" {
			continue
		}
		if sum, e := osfs.ReadRegularMetadata(ctx, filepath.Join(dirs[m.Path], "go.sum"), 2<<20); e == nil {
			if err := add(m.Path, "go.sum", "", sum); err != nil {
				return nil, nil, "", err
			}
		} else if !os.IsNotExist(e) {
			return nil, nil, "", e
		}
	}
	if mainPath == "" {
		return nil, nil, "", errors.New("root main module not found")
	}
	mainFound := false
	for _, p := range packages {
		if p.Standard {
			continue
		}
		if p.Error != nil || p.Incomplete {
			return nil, nil, "", fmt.Errorf("incomplete package graph: %s", p.Dir)
		}
		if p.Name == "main" {
			mainFound = true
		}
		if len(p.CgoFiles)+len(p.CFiles)+len(p.CXXFiles)+len(p.MFiles)+len(p.FFiles)+len(p.SwigFiles)+len(p.SwigCXXFiles)+len(p.SysoFiles) > 0 {
			return nil, nil, "", fmt.Errorf("unsupported foreign build inputs: %s", p.Dir)
		}
		if p.Module == nil || dirs[p.Module.Path] == "" {
			return nil, nil, "", errors.New("package has no captured module")
		}
		base := dirs[p.Module.Path]
		for _, group := range [][]string{p.GoFiles, p.SFiles, p.HFiles, p.EmbedFiles} {
			for _, name := range group {
				path := filepath.Join(p.Dir, filepath.FromSlash(name))
				if err := buildSafePath(base, path); err != nil {
					return nil, nil, "", err
				}
				rel, err := filepath.Rel(base, path)
				if err != nil {
					return nil, nil, "", err
				}
				if err = add(p.Module.Path, rel, path, nil); err != nil {
					return nil, nil, "", err
				}
			}
		}
	}
	if !mainFound {
		return nil, nil, "", errors.New("scheduled task is not a main package")
	}
	return inputs, mapped, mainPath, nil
}

func buildWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
func buildSafePath(root, path string) error {
	if !buildWithin(root, path) {
		return errors.New("build input escapes module")
	}
	physical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if physical != path {
		return fmt.Errorf("symlinked build input: %s", path)
	}
	return nil
}

func buildEnvironment(offline bool) []string {
	overrides := map[string]string{"GOWORK": "off", "GOTOOLCHAIN": "local", "GOFLAGS": "", "CGO_ENABLED": "0", "GOOS": runtime.GOOS, "GOARCH": runtime.GOARCH, "GOENV": "off"}
	if offline {
		overrides["GOPROXY"] = "off"
		overrides["GOSUMDB"] = "off"
	}
	var env []string
	for _, v := range os.Environ() {
		k, _, _ := strings.Cut(v, "=")
		if _, ok := overrides[k]; !ok {
			env = append(env, v)
		}
	}
	for k, v := range overrides {
		env = append(env, k+"="+v)
	}
	return env
}

type buildCapture struct {
	buffer   bytes.Buffer
	overflow bool
}

func (c *buildCapture) Len() int       { return c.buffer.Len() }
func (c *buildCapture) Bytes() []byte  { return c.buffer.Bytes() }
func (c *buildCapture) String() string { return c.buffer.String() }

func (c *buildCapture) Write(p []byte) (int, error) {
	n := len(p)
	left := maxBuildCommandBytes - c.Len()
	if len(p) > left {
		p = p[:left]
		c.overflow = true
	}
	_, _ = c.buffer.Write(p)
	return n, nil
}
func (b Builder) command(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if b.Run != nil {
		return b.Run(ctx, dir, env, args)
	}
	goBin := b.GoBinary
	if goBin == "" {
		var err error
		goBin, err = exec.LookPath("go")
		if err != nil {
			return nil, err
		}
	}
	cmd := exec.CommandContext(ctx, goBin, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.WaitDelay = time.Second
	var out, stderr buildCapture
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if out.overflow || stderr.overflow {
		return nil, errors.New("go command output exceeds bound")
	}
	if err != nil {
		return nil, fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return out.Bytes(), nil
}

func buildLock(ctx context.Context, path string) (func(), error) {
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		f.Close()
		return nil, errors.New("unsafe build lock file")
	}
	for {
		err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = f.Close() }, nil
		}
		if err != unix.EWOULDBLOCK && err != unix.EAGAIN {
			f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func cachedBuild(ctx context.Context, dir string, want BuildManifest) (Artifact, bool) {
	raw, err := osfs.ReadRegularMetadata(ctx, filepath.Join(dir, "manifest.json"), 2<<20)
	if err != nil {
		return Artifact{}, false
	}
	var got BuildManifest
	if strictjson.Decode(raw, &got) != nil || got.Version != 1 || got.Key != want.Key || got.BinarySHA256 == "" {
		return Artifact{}, false
	}
	got.BinarySHA256 = ""
	got.Key = ""
	got.LastUsed = 0
	encoded, _ := json.Marshal(got)
	if digestBuild(encoded) != want.Key {
		return Artifact{}, false
	}
	var committed BuildManifest
	_ = json.Unmarshal(raw, &committed)
	binary := filepath.Join(dir, "task")
	data, err := osfs.ReadRegularMetadata(ctx, binary, 256<<20)
	if err != nil || digestBuild(data) != committed.BinarySHA256 {
		return Artifact{}, false
	}
	info, err := os.Lstat(binary)
	if err != nil || info.Mode().Perm() != 0700 {
		return Artifact{}, false
	}
	return Artifact{Path: binary, Key: want.Key}, true
}
