package testcouch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func upgradeFixture(t *testing.T) (*Fleet, Binaries) {
	t.Helper()
	b, roots, lookup := runtimeFilesFixture(t)
	if err := installRuntimeWithTools(b, roots, lookup); err != nil {
		t.Fatal(err)
	}
	candidate, _, _ := runtimeFilesFixture(t)
	for _, path := range []string{candidate.Pair, candidate.Couch, candidate.Helper} {
		if err := os.WriteFile(path, []byte("new "+filepath.Base(path)), 0700); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	idle, exited := make(chan struct{}), make(chan struct{})
	close(idle)
	close(exited)
	f := &Fleet{Roots: roots, Binaries: b, RunID: "old-owner", Environment: []string{"PAIR_HOME=" + b.Source, "COUCH_ISOLATED_ROOT=" + roots.Root, "PATH=" + roots.Mutable["bin"]}, ctx: ctx, cancel: cancel, idle: idle, t: t, console: &terminalProcess{done: exited}, model: OwnedRun{Phase: "ready"}}
	return f, candidate
}

func TestReplaceBuildChangesPhysicalTripletAndPreservesLifetime(t *testing.T) {
	f, candidate := upgradeFixture(t)
	old := f.Binaries
	beforeEnv := append([]string(nil), f.Environment...)
	beforeRoots := f.Roots
	beforeFixture, err := os.ReadFile(filepath.Join(f.Roots.Mutable["bin"], "claude"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.ReplaceBuild(t.Context(), candidate); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{"pair": candidate.Pair, "couch": candidate.Couch, "pair-launch-helper": candidate.Helper} {
		path := filepath.Join(f.Roots.Mutable["bin"], name)
		got, err := os.ReadFile(path)
		want, sourceErr := os.ReadFile(source)
		installed, statErr := os.Lstat(path)
		original, originalErr := os.Stat(source)
		if err != nil || sourceErr != nil || statErr != nil || originalErr != nil || string(got) != string(want) || !installed.Mode().IsRegular() || os.SameFile(installed, original) {
			t.Fatalf("%s was not physically replaced: %q %v %v %v %v", name, got, err, sourceErr, statErr, originalErr)
		}
	}
	wantEnv := append([]string(nil), beforeEnv...)
	wantEnv[0] = "PAIR_HOME=" + candidate.Source
	if !reflect.DeepEqual(f.Environment, wantEnv) || !reflect.DeepEqual(f.Roots, beforeRoots) || f.RunID != "old-owner" {
		t.Fatalf("upgrade changed isolation authority: env=%v roots=%+v run=%s", f.Environment, f.Roots, f.RunID)
	}
	afterFixture, err := os.ReadFile(filepath.Join(f.Roots.Mutable["bin"], "claude"))
	if err != nil || string(afterFixture) != string(beforeFixture) || f.Binaries.Fixture != old.Fixture || f.Binaries.Zellij != old.Zellij {
		t.Fatalf("old fixture/native tools changed: %v", err)
	}
	entries, err := os.ReadDir(f.Roots.Mutable["bin"])
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".upgrade-") {
			t.Fatalf("upgrade staging retained: %s", entry.Name())
		}
	}
}

func TestReplaceBuildRefusesBeforePublication(t *testing.T) {
	for _, scenario := range []string{"live-console", "no-console", "other-operation", "starting", "canceled", "symlink-source", "missing-helper", "nonexec", "source-directory", "bin-escape"} {
		t.Run(scenario, func(t *testing.T) {
			f, candidate := upgradeFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch scenario {
			case "live-console":
				f.console.done = make(chan struct{})
			case "no-console":
				f.console = nil
			case "other-operation":
				_, done, err := f.begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer done()
			case "starting":
				f.starting = true
			case "canceled":
				cancel()
			case "symlink-source":
				alias := filepath.Join(t.TempDir(), "alias")
				if err := os.Symlink(candidate.Pair, alias); err != nil {
					t.Fatal(err)
				}
				candidate.Pair = alias
			case "missing-helper":
				candidate.Helper = filepath.Join(t.TempDir(), "missing")
			case "nonexec":
				if err := os.Chmod(candidate.Couch, 0600); err != nil {
					t.Fatal(err)
				}
			case "source-directory":
				candidate.Source = filepath.Join(t.TempDir(), "missing")
			case "bin-escape":
				f.Roots.Mutable["bin"] = t.TempDir()
			}
			if err := f.ReplaceBuild(ctx, candidate); err == nil {
				t.Fatal("unsafe replacement accepted")
			}
			if f.Binaries.Source == candidate.Source || strings.Contains(strings.Join(f.Environment, "\n"), "PAIR_HOME="+candidate.Source) {
				t.Fatal("refusal changed selected build")
			}
		})
	}
}

func TestReplaceBuildPartialPublicationClosesAdmission(t *testing.T) {
	for _, scenario := range []string{"cancel-after-pair", "rename-couch-refused"} {
		t.Run(scenario, func(t *testing.T) {
			f, candidate := upgradeFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			oldCouch, err := os.ReadFile(filepath.Join(f.Roots.Mutable["bin"], "couch"))
			if err != nil {
				t.Fatal(err)
			}
			rename := func(source, destination string) error {
				if scenario == "rename-couch-refused" && filepath.Base(destination) == "couch" {
					return os.ErrPermission
				}
				if err := os.Rename(source, destination); err != nil {
					return err
				}
				if scenario == "cancel-after-pair" {
					cancel()
				}
				return nil
			}
			if err := f.replaceBuild(ctx, candidate, rename); err == nil {
				t.Fatal("partial publication succeeded")
			} else if scenario == "cancel-after-pair" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation cause lost: %v", err)
			}
			if _, done, err := f.begin(t.Context()); err == nil {
				done()
				t.Fatal("partial executable set admitted a new command")
			}
			pair, err := os.ReadFile(filepath.Join(f.Roots.Mutable["bin"], "pair"))
			if err != nil || string(pair) != "new candidate-pair" {
				t.Fatalf("partial pair: %q %v", pair, err)
			}
			couch, err := os.ReadFile(filepath.Join(f.Roots.Mutable["bin"], "couch"))
			if err != nil || string(couch) != string(oldCouch) {
				t.Fatalf("old couch changed: %q %v", couch, err)
			}
			if f.Binaries.Source == candidate.Source {
				t.Fatal("partial set became selected build")
			}
			entries, err := os.ReadDir(f.Roots.Mutable["bin"])
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".upgrade-") {
					t.Fatalf("staging retained: %s", entry.Name())
				}
			}
		})
	}
}
