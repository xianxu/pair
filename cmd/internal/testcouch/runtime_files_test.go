package testcouch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runtimeFilesFixture(t *testing.T) (Binaries, RootsPlan, func(string) (string, error)) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := RootsPlan{Root: root, Mutable: map[string]string{"bin": filepath.Join(root, "bin"), "config": filepath.Join(root, "config")}}
	for _, dir := range p.Mutable {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	source := t.TempDir()
	executable := func(name string) string {
		path := filepath.Join(source, name)
		if err := os.WriteFile(path, []byte("executable "+name), 0700); err != nil {
			t.Fatal(err)
		}
		return path
	}
	b := Binaries{Source: source, Pair: executable("candidate-pair"), Couch: executable("candidate-couch"), Helper: executable("candidate-helper"), Fixture: executable("candidate-fixture"), Git: executable("git"), SDLC: executable("sdlc"), Weave: executable("weave"), Zellij: executable("zellij"), Nvim: executable("nvim")}
	utility := executable("utility")
	return b, p, func(string) (string, error) { return utility, nil }
}
func TestInstallRuntimeCopiesPhysicalAgentsAndPinsTools(t *testing.T) {
	b, p, lookup := runtimeFilesFixture(t)
	if err := installRuntimeWithTools(b, p, lookup); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{"pair": b.Pair, "couch": b.Couch, "pair-launch-helper": b.Helper, "claude": b.Fixture, "codex": b.Fixture} {
		path := filepath.Join(p.Mutable["bin"], name)
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		before, err := os.Stat(source)
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || os.SameFile(info, before) {
			t.Fatalf("%s is not an independent executable copy: %v", name, info.Mode())
		}
		got, _ := os.ReadFile(path)
		want, _ := os.ReadFile(source)
		if string(got) != string(want) {
			t.Fatalf("%s copy changed bytes", name)
		}
	}
	for name, source := range map[string]string{"git": b.Git, "sdlc": b.SDLC, "weave": b.Weave, "zellij": b.Zellij, "nvim": b.Nvim} {
		got, err := os.Readlink(filepath.Join(p.Mutable["bin"], name))
		if err != nil || got != source {
			t.Fatalf("%s target = %q %v", name, got, err)
		}
	}
	for _, name := range runtimeUtilities {
		if _, err := os.Readlink(filepath.Join(p.Mutable["bin"], name)); err != nil {
			t.Fatalf("utility %s: %v", name, err)
		}
	}
	config, err := os.ReadFile(filepath.Join(p.Mutable["config"], "config.kdl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, setting := range []string{"pane_frames false", "show_startup_tips false", "show_release_notes false"} {
		if !strings.Contains(string(config), setting) {
			t.Fatalf("missing native config setting %q", setting)
		}
	}
	original, _ := os.ReadFile(b.Fixture)
	if string(original) != "executable candidate-fixture" {
		t.Fatal("candidate source mutated")
	}
}
func TestInstallRuntimeRefusesUnsafeFactsBeforeEffects(t *testing.T) {
	for _, scenario := range []string{"source-missing", "source-symlink", "source-nonexec", "source-oversized", "tool-relative", "utility-missing", "bin-escape", "existing-bin-entry", "existing-config"} {
		t.Run(scenario, func(t *testing.T) {
			b, p, lookup := runtimeFilesFixture(t)
			outside := t.TempDir()
			switch scenario {
			case "source-missing":
				b.Fixture = filepath.Join(outside, "missing")
			case "source-symlink":
				alias := filepath.Join(outside, "alias")
				if err := os.Symlink(b.Fixture, alias); err != nil {
					t.Fatal(err)
				}
				b.Fixture = alias
			case "source-nonexec":
				if err := os.Chmod(b.Fixture, 0600); err != nil {
					t.Fatal(err)
				}
			case "source-oversized":
				if err := os.Truncate(b.Fixture, maxRuntimeFileBytes+1); err != nil {
					t.Fatal(err)
				}
			case "tool-relative":
				b.Git = "git"
			case "utility-missing":
				lookup = func(string) (string, error) { return "", errors.New("missing utility") }
			case "bin-escape":
				if err := os.Remove(p.Mutable["bin"]); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, p.Mutable["bin"]); err != nil {
					t.Fatal(err)
				}
			case "existing-bin-entry":
				if err := os.WriteFile(filepath.Join(p.Mutable["bin"], "keep"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "existing-config":
				if err := os.WriteFile(filepath.Join(p.Mutable["config"], "config.kdl"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadDir(p.Mutable["bin"])
			configBefore, _ := os.ReadDir(p.Mutable["config"])
			if err := installRuntimeWithTools(b, p, lookup); err == nil {
				t.Fatal("unsafe runtime installation accepted")
			}
			after, _ := os.ReadDir(p.Mutable["bin"])
			configAfter, _ := os.ReadDir(p.Mutable["config"])
			if len(after) != len(before) || len(configAfter) != len(configBefore) {
				t.Fatalf("refusal wrote runtime files: bin %v -> %v; config %v -> %v", before, after, configBefore, configAfter)
			}
		})
	}
}

func TestRuntimeCopyRefusesChangedSourceOrExistingDestination(t *testing.T) {
	for _, scenario := range []string{"replacement-source", "existing-destination"} {
		t.Run(scenario, func(t *testing.T) {
			b, p, _ := runtimeFilesFixture(t)
			expected, err := os.Stat(b.Pair)
			if err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(p.Mutable["bin"], "pair")
			if scenario == "replacement-source" {
				if err := os.Rename(b.Pair, b.Pair+"-old"); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(b.Pair, []byte("replacement"), 0700); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(dest, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := copyRuntimeExecutable(b.Pair, dest, expected); err == nil {
				t.Fatal("unsafe copy accepted")
			}
			if scenario == "replacement-source" {
				if _, err := os.Stat(dest); !os.IsNotExist(err) {
					t.Fatalf("changed source created destination: %v", err)
				}
			} else {
				raw, err := os.ReadFile(dest)
				if err != nil || string(raw) != "keep" {
					t.Fatalf("existing destination changed: %q %v", raw, err)
				}
			}
		})
	}
}
