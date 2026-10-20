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


ROOT = Path(__file__).resolve().parents[1]
PACKAGES = ("affected", "jsonstrict")
PREFIX = "cmd/sdlc/internal/"


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
        for package in PACKAGES:
            directory = PREFIX + package
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
            return candidate, ref, result
    message = ("upstream affected/jsonstrict source unavailable; install ariadne#473 or set "
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
    "path/filepath"
    "reflect"
    "sort"
    "strings"
    "testing"

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
    selected := affected.Plan(paths, g, cfg, nil)
    if selected.Full { t.Fatalf("%v unexpectedly full: %s", paths, selected.Reason) }
    if len(selected.GoPkgs) != len(packages) || (len(packages) > 0 && !reflect.DeepEqual(selected.GoPkgs, packages)) {
        t.Fatalf("%v packages = %v, want %v", paths, selected.GoPkgs, packages)
    }
    if len(selected.Groups) != len(groups) || (len(groups) > 0 && !reflect.DeepEqual(selected.Groups, groups)) {
        t.Fatalf("%v groups = %v, want %v", paths, selected.Groups, groups)
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
    exact(t, cfg, g, []string{"scripts/supervision-labels/check.py"}, nil, []string{"build"})
    exact(t, cfg, g, []string{"nvim/workbench_route.lua"}, nil, []string{"build", "lua", "review", "shell"})
    exact(t, cfg, g, []string{"cmd/internal/runtimebundle/assets/runtime/files/nvim/workbench_route.lua"},
        []string{"example.com/fixture/cmd/internal/runtimebundle"}, []string{"build", "lua"})
    for _, path := range []string{"Makefile", "Makefile.local", "tests/lib/helper.sh", "go.mod", "go.sum", ".sdlc/affected.json", ".github/workflows/ci.yml", "tests/webcast/unclaimed.fixture", "tests/broadcast-viewer/unclaimed.fixture", "tests/terminal-oracle/unclaimed.fixture", "tests/new-unowned.fixture", "cmd/internal/absent_platform_package/deleted.go", "probes/absent_platform_package/deleted.go"} {
        t.Run(path, func(t *testing.T) {
            selected := affected.Plan([]string{path}, g, cfg, nil)
            if !selected.Full { t.Fatalf("%s must force full; got %+v", path, selected) }
        })
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
                    selected := affected.Plan([]string{changed}, g, cfg, nil)
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
                    selected := affected.Plan([]string{relative}, g, cfg, nil)
                    covered := selected.Full
                    for _, got := range selected.Groups { if got == group { covered = true } }
                    if !covered { t.Errorf("%s/%s input %s omits consumer; groups=%v", group, target, relative, selected.Groups) }
                }
            }
        }
    }
    t.Logf("checked %d distinct group/file inputs", checked)
}
'''


class AffectedConfigTest(unittest.TestCase):
    def test_check_config(self):
        self.assertTrue((ROOT / ".sdlc/affected.json").is_file(), "missing .sdlc/affected.json")
        candidate, ref, sources = provider_sources()
        print(f"Qualifying upstream affected source: {candidate}" + (f" @ {ref}" if ref else ""), flush=True)
        with tempfile.TemporaryDirectory(prefix="pair-affected-config-") as scratch:
            root = Path(scratch)
            (root / "go.mod").write_text("module github.com/xianxu/ariadne/cmd/sdlc\n\ngo 1.26.3\n")
            for name, data in sources.items():
                path = root / name.removeprefix("cmd/sdlc/")
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(data)
            fixture = root / "fixture"
            fixture.mkdir()
            (fixture / "config_test.go").write_text(GO_TEST)
            result = subprocess.run(
                ["go", "test", "./fixture", "-count=1", "-v", "-timeout", "3m"],
                cwd=root, env={**os.environ, "GOWORK": "off", "PAIR_AFFECTED_ROOT": str(ROOT)},
                text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            )
            self.assertEqual(result.returncode, 0, result.stdout)
            print(result.stdout, end="")


if __name__ == "__main__":
    unittest.main(verbosity=2)
