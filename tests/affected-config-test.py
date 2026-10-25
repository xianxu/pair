#!/usr/bin/env python3
"""Qualify Pair's declaration against ariadne's actual affected implementation.

No planner is vendored or reproduced. The installed module or a sibling ariadne
checkout supplies its unmodified source in a disposable internal-package module.
For a not-yet-landed provider, explicitly set ARIADNE_AFFECTED_SOURCE and
ARIADNE_AFFECTED_REF (a local Git ref); the provider checkout is never modified.
"""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock


ROOT = Path(__file__).resolve().parents[1]
PACKAGES = ("affected", "jsonstrict")
PREFIX = "cmd/sdlc/internal/"
# Same-provider closure of current affected (golist.go's process group, and the
# shared strict decoder jsonstrict wraps). Optional: older planners lack them.
SUPPORTING_PACKAGES = (PREFIX + "processgroup", "pkg/procgroup", "cmd/internal/jsonstrict")


def provider_sources():
    explicit = os.environ.get("ARIADNE_AFFECTED_SOURCE")
    ref = os.environ.get("ARIADNE_AFFECTED_REF")
    if ref and not explicit:
        raise RuntimeError("ARIADNE_AFFECTED_REF requires ARIADNE_AFFECTED_SOURCE")
    candidates = [Path(explicit)] if explicit else [ROOT.parent / "ariadne"]
    if not explicit:
        installed = subprocess.run(
            ["go", "list", "-m", "-f", "{{.Dir}}", "github.com/xianxu/ariadne"],
            cwd=ROOT, text=True, capture_output=True, check=False,
            env={**os.environ, "GOWORK": "off", "GOPROXY": "off", "GOSUMDB": "off"},
        )
        if installed.returncode == 0 and installed.stdout.strip():
            candidates.append(Path(installed.stdout.strip()))
    for candidate in candidates:
        result = {}
        for directory in tuple(PREFIX + package for package in PACKAGES) + SUPPORTING_PACKAGES:
            if ref:
                listing = subprocess.run(
                    ["git", "-C", str(candidate), "ls-tree", "-r", "--name-only", ref, directory],
                    text=True, capture_output=True, check=True,
                ).stdout.splitlines()
                for name in listing:
                    if name.endswith(".go") and not name.endswith("_test.go"):
                        result[name] = subprocess.check_output(
                            ["git", "-C", str(candidate), "show", f"{ref}:{name}"]
                        )
            else:
                for path in (candidate / directory).glob("*.go"):
                    if not path.name.endswith("_test.go"):
                        result[path.relative_to(candidate).as_posix()] = path.read_bytes()
        if all(any(name.startswith(PREFIX + p + "/") for name in result) for p in PACKAGES):
            # This consumer requires ariadne#598 (sidecars, full excepts), not
            # merely an older planner. A stale sibling/module is unavailable
            # qualification evidence, not a reason to compile against an API
            # whose strict parse refuses pair's declaration.
            if b"SidecarFile" not in result.get(PREFIX + "affected/mapping.go", b""):
                continue
            # Keep module identity and dependency versions byte-for-byte from
            # this provider/ref, including processgroup's Darwin x/sys import.
            for name in ("go.mod", "go.sum"):
                if ref:
                    metadata = subprocess.run(
                        ["git", "-C", str(candidate), "show", f"{ref}:{name}"],
                        stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
                    )
                    if metadata.returncode == 0:
                        result[name] = metadata.stdout
                    elif name == "go.mod":
                        raise RuntimeError(f"provider {candidate} @ {ref} has no go.mod")
                elif (candidate / name).is_file():
                    result[name] = (candidate / name).read_bytes()
                elif name == "go.mod":
                    raise RuntimeError(f"provider {candidate} has no go.mod")
            return candidate, ref, result
    message = ("upstream affected planner and dependency sources unavailable; install them or set "
               "ARIADNE_AFFECTED_SOURCE (and optional ARIADNE_AFFECTED_REF) to qualify its candidate")
    if explicit:
        raise RuntimeError(message)
    raise unittest.SkipTest(message)


GO_TEST = r'''
package fixture

import (
    "bytes"
    "context"
    "encoding/json"
    "os"
    "os/exec"
    "path/filepath"
    "reflect"
    "sort"
    "strings"
    "sync"
    "syscall"
    "testing"
    "time"

    "github.com/xianxu/ariadne/cmd/sdlc/internal/affected"
)

func config(t *testing.T) (affected.Config, []byte) {
    t.Helper()
    data, err := os.ReadFile(filepath.Join(os.Getenv("PAIR_AFFECTED_ROOT"), ".sdlc", "affected.json"))
    if err != nil { t.Fatal(err) }
    cfg, err := affected.ParseConfig(data)
    if err != nil { t.Fatal(err) }
    names := []string{}
    for _, group := range cfg.Groups { names = append(names, group.Name) }
    sort.Strings(names)
    if !reflect.DeepEqual(names, []string{"build", "doctor", "lua", "review", "shell"}) {
        t.Fatalf("group contract: %v", names)
    }
    return cfg, data
}

// loadMapping compiles the real certificates under main's sidecar convention,
// as sdlc does from the trusted base.
func loadMapping(cfg affected.Config) (affected.Mapping, error) {
    root := os.Getenv("PAIR_AFFECTED_ROOT")
    var files []affected.SidecarFile
    for _, decl := range cfg.Sidecars {
        err := filepath.WalkDir(filepath.Join(root, decl.Prefix), func(path string, entry os.DirEntry, err error) error {
            if err != nil || entry.IsDir() || !strings.HasSuffix(path, decl.Suffix) { return err }
            content, err := os.ReadFile(path)
            if err != nil { return err }
            relative, err := filepath.Rel(root, path)
            files = append(files, affected.SidecarFile{Path: filepath.ToSlash(relative), Content: content})
            return err
        })
        if err != nil { return affected.Mapping{}, err }
    }
    return affected.Compile(cfg, files)
}

// The mapping is compiled once per test binary: every caller reads the same
// on-disk declaration, and the coverage tests plan hundreds of paths.
var loadOnce = sync.OnceValues(func() (affected.Mapping, error) {
    data, err := os.ReadFile(filepath.Join(os.Getenv("PAIR_AFFECTED_ROOT"), ".sdlc", "affected.json"))
    if err != nil { return affected.Mapping{}, err }
    cfg, err := affected.ParseConfig(data)
    if err != nil { return affected.Mapping{}, err }
    return loadMapping(cfg)
})

func plan(t *testing.T, paths []string, g affected.Graph, cfg affected.Config) affected.Selection {
    t.Helper()
    m, err := loadOnce()
    if err != nil { t.Fatal(err) }
    return affected.Plan(paths, g, cfg, m, nil)
}

func write(t *testing.T, root, name, text string) {
    t.Helper()
    path := filepath.Join(root, name)
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { t.Fatal(err) }
    if err := os.WriteFile(path, []byte(text), 0644); err != nil { t.Fatal(err) }
}

func graph(t *testing.T) affected.Graph {
    t.Helper()
    root := t.TempDir()
    write(t, root, "go.mod", "module example.com/fixture\n\ngo 1.26.3\n")
    write(t, root, "leaf/leaf.go", "package leaf\nconst Value = 1\n")
    write(t, root, "mid/mid.go", "package mid\nimport \"example.com/fixture/leaf\"\nconst Value = leaf.Value\n")
    write(t, root, "top/top.go", "package top\nimport \"example.com/fixture/mid\"\nconst Value = mid.Value\n")
    write(t, root, "testonly/only.go", "package testonly\n")
    write(t, root, "testonly/only_test.go", "package testonly\nimport (\"testing\"; \"example.com/fixture/leaf\")\nfunc TestValue(t *testing.T) { if leaf.Value != 1 { t.Fatal(leaf.Value) } }\n")
    write(t, root, "other/other.go", "package other\n")
    write(t, root, "cmd/internal/runtimebundle/embed.go", "package runtimebundle\nimport _ \"embed\"\n//go:embed assets/runtime/files/nvim/workbench_route.lua\nvar Lua string\n")
    write(t, root, "cmd/internal/runtimebundle/assets/runtime/files/nvim/workbench_route.lua", "return {}\n")
    cfg, _ := config(t)
    // Materialize each declared root, so the real planner checks every root.
    for _, group := range cfg.Groups {
        for _, pkg := range group.Go {
            dir := filepath.Clean(pkg)
            if dir == "cmd/internal/runtimebundle" { continue }
            write(t, root, filepath.Join(dir, "fixture.go"), "package fixture\n")
        }
    }
    g, err := affected.LoadGoGraph(context.Background(), root)
    if err != nil { t.Fatal(err) }
    return g
}

func exact(t *testing.T, cfg affected.Config, g affected.Graph, paths, packages, groups []string) {
    t.Helper()
    selected := plan(t, paths, g, cfg)
    if selected.Full { t.Fatalf("%v unexpectedly full: %s", paths, selected.Reason) }
    if len(selected.GoPkgs) != len(packages) || (len(packages) > 0 && !reflect.DeepEqual(selected.GoPkgs, packages)) {
        t.Fatalf("%v packages = %v, want %v", paths, selected.GoPkgs, packages)
    }
    if len(selected.Groups) != len(groups) || (len(groups) > 0 && !reflect.DeepEqual(selected.Groups, groups)) {
        t.Fatalf("%v groups = %v, want %v", paths, selected.Groups, groups)
    }
}

// Commands are owned by this fixture's bounded scope, including descendants.
func graphCommand(t *testing.T, ctx context.Context, dir string, argv ...string) {
    t.Helper()
    cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
    cmd.Dir = dir
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
    cmd.WaitDelay = time.Second
    out, err := cmd.CombinedOutput()
    if cmd.Process != nil { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
    if err != nil { t.Fatalf("%v: %v\n%s", argv, err, out) }
}

func TestDeclaredPreparationQualifiesCleanPairGraph(t *testing.T) {
    cfg, _ := config(t)
    if cfg.GraphPrepare == nil || !reflect.DeepEqual(cfg.GraphPrepare.Command, []string{"make", "runtimebundle-generate"}) || cfg.GraphPrepare.TimeoutMinutes != 2 {
        t.Fatalf("missing bounded runtime preparation: %+v", cfg.GraphPrepare)
    }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    t.Setenv("GOPROXY", "off")
    scratch := t.TempDir()
    clean := filepath.Join(scratch, "checkout")
    if err := os.Mkdir(clean, 0755); err != nil { t.Fatal(err) }
    archive := filepath.Join(scratch, "source.tar")
    graphCommand(t, ctx, os.Getenv("PAIR_AFFECTED_ROOT"), "git", "archive", "--format=tar", "--output="+archive, "HEAD")
    graphCommand(t, ctx, clean, "tar", "-xf", archive)
    assets := filepath.Join(clean, "cmd/internal/runtimebundle/assets/runtime/files")
    if _, err := os.Stat(assets); !os.IsNotExist(err) { t.Fatalf("archive borrowed generated assets: %v", err) }
    if _, err := affected.LoadGoGraph(ctx, clean); err == nil || !strings.Contains(err.Error(), "no matching files found") {
        t.Fatalf("cold graph must fail on missing generated embeds: %v", err)
    }
    graphCommand(t, ctx, clean, cfg.GraphPrepare.Command...)
    g, err := affected.LoadGoGraph(ctx, clean)
    if err != nil { t.Fatalf("declared generator did not establish graph: %v", err) }
    exact(t, cfg, g, []string{"cmd/internal/slugcmd/slug.go"}, []string{
        "github.com/xianxu/pair/cmd/internal/dispatcher",
        "github.com/xianxu/pair/cmd/internal/slugcmd",
        "github.com/xianxu/pair/cmd/pair-go",
    }, []string{"build"})
    write(t, clean, "cmd/internal/slugcmd/broken_graph.go", "package slugcmd\nimport _ \"github.com/xianxu/pair/graph_prepare_missing\"\n")
    if _, err := affected.LoadGoGraph(ctx, clean); err == nil || !strings.Contains(err.Error(), "graph_prepare_missing") {
        t.Fatalf("preparation concealed a genuine broken graph: %v", err)
    }
}

func TestStrictConfig(t *testing.T) {
    _, data := config(t)
    cases := map[string][]byte{
        "unknown": append([]byte(`{"unexpected":true,`), bytes.TrimSpace(data)[1:]...),
        "duplicate": append([]byte(`{"version":1,`), bytes.TrimSpace(data)[1:]...),
        "nested unknown": bytes.Replace(data, []byte(`"name"`), []byte(`"unexpected":true,"name"`), 1),
        "nested duplicate": bytes.Replace(data, []byte(`"name"`), []byte(`"name":"duplicate","name"`), 1),
    }
    for name, input := range cases {
        t.Run(name, func(t *testing.T) {
            if bytes.Equal(input, data) { t.Fatal("mutation did not apply") }
            if _, err := affected.ParseConfig(input); err == nil { t.Fatal("strict parser accepted mutation") }
        })
    }
}

func TestPackageUnion(t *testing.T) {
    cfg, _ := config(t)
    g := graph(t)
    expected := []string{"example.com/fixture/leaf", "example.com/fixture/mid", "example.com/fixture/testonly", "example.com/fixture/top"}
    exact(t, cfg, g, []string{"leaf/leaf.go"}, expected, nil)
    expected = append(expected[:2], append([]string{"example.com/fixture/other"}, expected[2:]...)...)
    exact(t, cfg, g, []string{"leaf/leaf.go", "other/other.go"}, expected, nil)
}

func TestPathsAndFallbacks(t *testing.T) {
    cfg, _ := config(t)
    g := graph(t)
    exact(t, cfg, g, []string{"cmd/internal/runtimebundle/assets/runtime/files/nvim/workbench_route.lua"},
        []string{"example.com/fixture/cmd/internal/runtimebundle"}, []string{"build", "lua"})
    for _, path := range []string{"Makefile", "Makefile.local", "tests/lib/helper.sh", "go.mod", "go.sum", ".sdlc/affected.json", ".github/workflows/ci.yml", "tests/webcast/unclaimed.fixture", "tests/broadcast-viewer/unclaimed.fixture", "tests/terminal-oracle/unclaimed.fixture", "tests/new-unowned.fixture", "cmd/internal/absent_platform_package/deleted.go", "probes/absent_platform_package/deleted.go"} {
        t.Run(path, func(t *testing.T) {
            selected := plan(t, []string{path}, g, cfg)
            if !selected.Full { t.Fatalf("%s must force full; got %+v", path, selected) }
        })
    }
}

// Non-Go source inputs have Go test consumers too. group.go is only a
// trigger: it does not add those consumers to GoPkgs, so these populations
// must retain full coverage until the provider can model that direction.
func TestActualNonGoInputsForceFull(t *testing.T) {
    cfg, _ := config(t)
    g, err := affected.LoadGoGraph(context.Background(), os.Getenv("PAIR_AFFECTED_ROOT"))
    if err != nil { t.Fatal(err) }
    paths := []string{
        "scripts/supervision-labels/labels.py",
        "scripts/supervision-labels/check.py",
        "scripts/build-go-atomic.sh",
        "nvim/init.lua", "nvim/workbench_actions.lua", "nvim/scrollback.lua",
        "nvim/review.lua", "nvim/review/comment.lua", "nvim/review/comment_float.lua",
        "zellij/config.kdl", "zellij/layouts/main-2.kdl", "zellij/layouts/main-3.kdl",
        "doctor/doctor.sh", "doctor/perf.sh", "doctor/fixtures/perf_capture.txt",
        "bin/pair-help", "bin/lib/dev-rebuild.sh",
        "tests/adapt-golden.expected", "tests/fixtures/zellij-replies.json",
        "tests/copy-on-select-test.sh", "tests/with-isolated-pair.sh",
        "tests/without-ancestors.py", "tests/lib/zellij_fixture_replies.py",
        "tests/terminal-oracle/driver.cjs", "tests/webcast/model.test.mjs",
        "tests/broadcast-viewer/pinspots.json",
        "testdata/edit-records/propose/fixture.json",
        "cmd/internal/sessioninventory/testdata/normalization/v1.json",
        "cmd/internal/wrapcmd/testdata/drive/codex/0.162.1/session-status.screen",
        "cmd/internal/webcast/notes/testdata/anchor-candidates.json",
        "terminfo/pair-vt-256color.ti",
    }
    for _, path := range paths {
        t.Run(path, func(t *testing.T) {
            selected := plan(t, []string{path}, g, cfg)
            if !selected.Full { t.Errorf("non-Go input %s omitted its Go consumers: %+v", path, selected) }
        })
    }
}

// pair#853: a reader certificate selects what its test selects, and a couch
// test file selects its package alone; the policy, selector, inventory tool,
// shared harness and couch production code stay full.
func TestCertificatesAndCouchTestsNarrow(t *testing.T) {
    cfg, _ := config(t)
    g, err := affected.LoadGoGraph(context.Background(), os.Getenv("PAIR_AFFECTED_ROOT"))
    if err != nil { t.Fatal(err) }
    same := func(certificate, test string) {
        t.Helper()
        got, want := plan(t, []string{certificate}, g, cfg), plan(t, []string{test}, g, cfg)
        if got.Full || want.Full || len(got.GoPkgs) == 0 || !reflect.DeepEqual(got, want) {
            t.Errorf("%s selected %+v, want its test's %+v", certificate, got, want)
        }
    }
    readers := "tests/affected-readers/"
    // #825's certificate; an added (or deleted) certificate needs no base copy.
    same(readers+"cmd/internal/testcouch/agentfixture/request_lifetime_test.go.json", "cmd/internal/testcouch/agentfixture/request_lifetime_test.go")
    same(readers+"cmd/internal/slugcmd/added_test.go.json", "cmd/internal/slugcmd/added_test.go")
    couch := []string{"github.com/xianxu/pair/cmd/couch"}
    for _, path := range []string{"cmd/couch/main_test.go", "cmd/couch/editorial_acceptance_test.go",
        readers + "cmd/couch/main_test.go.json", readers + "cmd/couch/editorial_acceptance_test.go.json"} {
        exact(t, cfg, g, []string{path}, couch, nil)
    }
    // A rename is a deletion plus an addition: the union of both tests' selections.
    renamed := plan(t, []string{readers + "cmd/couch/main_test.go.json", readers + "cmd/internal/slugcmd/renamed_test.go.json"}, g, cfg)
    union := plan(t, []string{"cmd/couch/main_test.go", "cmd/internal/slugcmd/renamed_test.go"}, g, cfg)
    if renamed.Full || !reflect.DeepEqual(renamed, union) { t.Errorf("rename selected %+v, want %+v", renamed, union) }
    for _, path := range []string{
        readers + "cmd/internal/absent_platform_package/gone_test.go.json", // no package owns its test
        readers + "notes.json", readers + "README.md",
        ".sdlc/affected.json", "scripts/test-affected.py", "tests/test-affected-test.py",
        "tests/affected-go-readers-test.py", "tests/affected-config-test.py", "tests/lib/helper.sh",
        "tests/autopair-test.sh", "cmd/couch/main.go", "cmd/couch/sub/nested_test.go", "cmd/couch/testdata/x.json",
    } {
        if selected := plan(t, []string{path}, g, cfg); !selected.Full { t.Errorf("%s must stay full: %+v", path, selected) }
    }
    if selected := plan(t, []string{readers + "cmd/couch/main_test.go.json", "scripts/test-affected.py"}, g, cfg); !selected.Full {
        t.Errorf("a certificate beside the selector must stay full: %+v", selected)
    }
}

// A certificate's declared reads add its test's package to an input's
// selection; they never remove the input's own full rule, and a malformed
// declaration refuses rather than narrowing.
func TestCertificateReads(t *testing.T) {
    cfg, _ := config(t)
    g, err := affected.LoadGoGraph(context.Background(), os.Getenv("PAIR_AFFECTED_ROOT"))
    if err != nil { t.Fatal(err) }
    certificate := "tests/affected-readers/cmd/internal/slugcmd/slug_test.go"
    sidecar := func(content string) (affected.Mapping, error) {
        return affected.Compile(cfg, []affected.SidecarFile{{Path: certificate + ".json", Content: []byte(content)}})
    }
    m, err := sidecar(`{"version": 1, "disposition": "local_or_literal_fixture", "sha256": "00", "reads": ["atlas/fixture.dat", "nvim/x.lua"], "future": {"kept": true}}`)
    if err != nil { t.Fatal(err) }
    if selected := affected.Plan([]string{"atlas/fixture.dat"}, g, cfg, affected.Mapping{}, nil); !selected.Full {
        t.Fatalf("undeclared input must stay unclassified: %+v", selected)
    }
    got := affected.Plan([]string{"atlas/fixture.dat"}, g, cfg, m, nil)
    want := affected.Plan([]string{"cmd/internal/slugcmd/slug_test.go"}, g, cfg, m, nil)
    if got.Full || !reflect.DeepEqual(got, want) { t.Errorf("declared read selected %+v, want %+v", got, want) }
    if selected := affected.Plan([]string{"nvim/x.lua"}, g, cfg, m, nil); !selected.Full {
        t.Errorf("a declared read narrowed a full input: %+v", selected)
    }
    for _, bad := range []string{`{"reads": "atlas/x"}`, `{"reads": ["../x"]}`, `{"reads": ["/abs"]}`, `{`} {
        if _, err := sidecar(bad); err == nil { t.Errorf("malformed reads %s compiled", bad) }
    }
}

func TestActualPairRoots(t *testing.T) {
    cfg, _ := config(t)
    g, err := affected.LoadGoGraph(context.Background(), os.Getenv("PAIR_AFFECTED_ROOT"))
    if err != nil { t.Fatal(err) }
    dirs := map[string]bool{}
    for _, dir := range g.Dir { dirs[dir] = true }
    for _, group := range cfg.Groups {
        for _, root := range group.Go {
            if !dirs[filepath.Clean(root)] { t.Errorf("%s has stale root %s", group.Name, root) }
            if root == "./cmd/pair-go" { t.Errorf("%s uses cross-cutting dispatcher root", group.Name) }
        }
    }
    // Non-Go fallbacks must not widen a legitimate Go-only selection.
    exact(t, cfg, g, []string{"cmd/internal/slugcmd/slug.go"}, []string{
        "github.com/xianxu/pair/cmd/internal/dispatcher",
        "github.com/xianxu/pair/cmd/internal/slugcmd",
        "github.com/xianxu/pair/cmd/pair-go",
    }, []string{"build"})
}

// This independent suite-to-command inventory is audited from real recipes,
// not generated from the declaration whose dependency completeness it tests.
func TestActualGroupConsumerCoverage(t *testing.T) {
    cfg, _ := config(t)
    root := os.Getenv("PAIR_AFFECTED_ROOT")
    g, err := affected.LoadGoGraph(context.Background(), root)
    if err != nil { t.Fatal(err) }
    data, err := os.ReadFile(filepath.Join(root, "tests", "affected-consumers.json"))
    if err != nil { t.Fatal(err) }
    var consumers map[string]map[string]struct { Go []string `json:"go"`; Files []string `json:"files"` }
    if err := json.Unmarshal(data, &consumers); err != nil { t.Fatal(err) }
    for group, targets := range consumers {
        for target, consumer := range targets {
            for _, pkg := range consumer.Go {
                name := "github.com/xianxu/pair/" + pkg
                dir, exists := g.Dir[name]
                if !exists { t.Fatalf("%s/%s: unknown command root %s", group, target, pkg) }
                // Every in-module dependency can alter this command too. Go
                // graph ownership is directory-based, including new Go files.
                dependencies := map[string]bool{name:true}
                for dep := range g.Deps[name] { dependencies[dep] = true }
                for dep := range dependencies {
                    changed := filepath.ToSlash(filepath.Join(g.Dir[dep], "selection_probe.go"))
                    selected := plan(t, []string{changed}, g, cfg)
                    covered := selected.Full // explicit full is conservative coverage
                    for _, got := range selected.Groups { if got == group { covered = true } }
                    if !covered {
                        t.Errorf("%s consumer %s dependency %s omits %s; groups=%v", target, dir, dep, group, selected.Groups)
                    }
                }
            }
        }
    }
}

func TestActualFileConsumerCoverage(t *testing.T) {
    cfg, _ := config(t)
    root := os.Getenv("PAIR_AFFECTED_ROOT")
    g, err := affected.LoadGoGraph(context.Background(), root)
    if err != nil { t.Fatal(err) }
    data, err := os.ReadFile(filepath.Join(root, "tests", "affected-consumers.json"))
    if err != nil { t.Fatal(err) }
    var consumers map[string]map[string]struct { Go []string `json:"go"`; Files []string `json:"files"` }
    if err := json.Unmarshal(data, &consumers); err != nil { t.Fatal(err) }
    checked := 0
    seen := map[string]bool{}
    for group, targets := range consumers {
        for target, consumer := range targets {
            // Runner regressions enforce the unconditional text-inventory guard.
            if target == "test-make-env" { continue }
            for _, pattern := range consumer.Files {
                var files []string
                if strings.HasSuffix(pattern, "/**") {
                    err := filepath.WalkDir(filepath.Join(root, strings.TrimSuffix(pattern, "/**")),
                        func(path string, entry os.DirEntry, err error) error {
                            if err != nil { return err }
                            if !entry.IsDir() { files = append(files, path) }
                            return nil
                        })
                    if err != nil { t.Fatal(err) }
                } else {
                    files, err = filepath.Glob(filepath.Join(root, pattern))
                    if err != nil { t.Fatal(err) }
                }
                if len(files) == 0 { t.Fatalf("%s/%s stale input pattern %s", group, target, pattern) }
                for _, file := range files {
                    relative, err := filepath.Rel(root, file)
                    if err != nil { t.Fatal(err) }
                    relative = filepath.ToSlash(relative)
                    if filepath.Ext(relative) == ".md" { continue } // upstream docs-only exception
                    key := group+"/"+relative
                    if seen[key] { continue }
                    seen[key] = true
                    checked++
                    selected := plan(t, []string{relative}, g, cfg)
                    // These audited non-Go inputs can also alter Go test
                    // behavior. Selecting their shell/Lua group alone omits
                    // those Go consumers; the governing fallback is full.
                    if !selected.Full { t.Errorf("%s/%s input %s must force full for Go consumers; selection=%+v", group, target, relative, selected) }
                }
            }
        }
    }
    t.Logf("checked %d distinct group/file inputs", checked)
}
'''


class ProviderSourceTest(unittest.TestCase):
    def provider(self, root, supporting=True):
        sources = {
            "go.mod": b"module github.com/xianxu/ariadne\n\ngo 1.26.3\n",
            "go.sum": b"// source-acquisition metadata fixture\n",
            PREFIX + "affected/config.go": b"package affected\n",
            PREFIX + "affected/mapping.go": b"package affected\n// SidecarFile source-acquisition fixture\n",
            PREFIX + "jsonstrict/jsonstrict.go": b"package jsonstrict\n",
        }
        if supporting:
            sources.update({
                PREFIX + "processgroup/process_unix.go": b"package processgroup\n// pinned support fixture\n",
                "pkg/procgroup/procgroup.go": b"package procgroup\n// pinned support fixture\n",
            })
        for name, data in sources.items():
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
        return sources

    def acquire(self, root, ref=None):
        with mock.patch.dict(os.environ, {"ARIADNE_AFFECTED_SOURCE": str(root)}):
            if ref is None:
                os.environ.pop("ARIADNE_AFFECTED_REF", None)
            else:
                os.environ["ARIADNE_AFFECTED_REF"] = ref
            return provider_sources()

    def test_checkout_copies_module_and_supporting_sources(self):
        with tempfile.TemporaryDirectory(prefix="pair-provider-source-") as scratch:
            root = Path(scratch)
            expected = self.provider(root)
            ignored = root / (PREFIX + "processgroup/process_test.go")
            ignored.write_bytes(b"package processgroup\n")
            candidate, ref, sources = self.acquire(root)
            self.assertEqual((candidate, ref, sources), (root, None, expected))

    def test_pinned_ref_never_mixes_checkout_metadata_or_support(self):
        with tempfile.TemporaryDirectory(prefix="pair-provider-ref-") as scratch:
            root = Path(scratch)
            expected = self.provider(root)
            subprocess.run(["git", "-c", "init.templateDir=", "init", "-q", str(root)], check=True)
            subprocess.run(["git", "-C", str(root), "add", "."], check=True)
            subprocess.run(["git", "-C", str(root), "-c", "user.name=Fixture", "-c",
                            "user.email=fixture@example.test", "-c", "core.hooksPath=/dev/null",
                            "commit", "-qm", "pinned provider"], check=True)
            pin = subprocess.check_output(["git", "-C", str(root), "rev-parse", "HEAD"], text=True).strip()
            for name in expected:
                (root / name).write_bytes(b"worktree changes must not enter pinned source\n")
            (root / "pkg/procgroup/checkout_only.go").write_bytes(b"package procgroup\n")
            candidate, ref, sources = self.acquire(root, pin)
            self.assertEqual((candidate, ref, sources), (root, pin, expected))

    def test_older_supported_source_needs_no_optional_package(self):
        with tempfile.TemporaryDirectory(prefix="pair-provider-older-") as scratch:
            root = Path(scratch)
            expected = self.provider(root, supporting=False)
            # A dependency-free older module may also have no checksum file.
            expected.pop("go.sum")
            (root / "go.sum").unlink()
            candidate, ref, sources = self.acquire(root)
            self.assertEqual((candidate, ref, sources), (root, None, expected))


class AffectedConfigTest(unittest.TestCase):
    def test_check_config(self):
        self.assertTrue((ROOT / ".sdlc/affected.json").is_file(), "missing .sdlc/affected.json")
        candidate, ref, sources = provider_sources()
        print(f"Qualifying upstream affected source: {candidate}" + (f" @ {ref}" if ref else ""), flush=True)
        with tempfile.TemporaryDirectory(prefix="pair-affected-config-") as scratch:
            root = Path(scratch)
            for name, data in sources.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(data)
            fixture = root / "cmd/sdlc/fixture"
            fixture.mkdir(parents=True)
            (fixture / "config_test.go").write_text(GO_TEST)
            result = subprocess.run(
                ["go", "test", "./cmd/sdlc/fixture", "-count=1", "-v", "-timeout", "3m"],
                cwd=root, env={**os.environ, "GOWORK": "off", "PAIR_AFFECTED_ROOT": str(ROOT)},
                text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            )
            self.assertEqual(result.returncode, 0, result.stdout)
            print(result.stdout, end="")


if __name__ == "__main__":
    unittest.main(verbosity=2)
