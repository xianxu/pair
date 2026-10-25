package testcouch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReplaceBuild changes only the private executable triplet and PAIR_HOME after
// Detach has joined the console. Old actors, fixtures and cleanup stay owned by
// this Fleet. The caller must keep both CandidateOwners alive until Close.
func (f *Fleet) ReplaceBuild(ctx context.Context, candidate Binaries) error {
	return f.replaceBuild(ctx, candidate, os.Rename)
}

func (f *Fleet) replaceBuild(ctx context.Context, candidate Binaries, rename func(string, string) error) (result error) {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	f.mu.Lock()
	published := false
	defer func() {
		failure := RunEvent{Kind: "unknown", Error: fmt.Sprintf("partial build replacement: %v", result)}
		if published && result != nil {
			// Close admission before releasing the same mutex held by begin.
			f.model = ReduceRun(f.model, failure)
		}
		f.mu.Unlock()
		if published && result != nil {
			f.record(failure)
		}
	}()
	if f.active != 1 || f.starting || f.console == nil {
		return fmt.Errorf("build replacement requires a joined console and no other operation")
	}
	select {
	case <-f.console.done:
	default:
		return fmt.Errorf("detach and join the console before replacing its build")
	}
	if _, err := physicalDirectory(candidate.Source); err != nil {
		return err
	}
	roots, err := physicalRoots(f.Roots)
	if err != nil {
		return err
	}
	bin := roots.Mutable["bin"]
	if bin == "" {
		return fmt.Errorf("owned runtime bin required")
	}
	files := []runtimeFile{{name: "pair", source: candidate.Pair}, {name: "couch", source: candidate.Couch}, {name: "pair-launch-helper", source: candidate.Helper}}
	for i := range files {
		file := &files[i]
		file.info, err = os.Lstat(file.source)
		if err != nil {
			return err
		}
		if err := validateToolFacts(file.source, file.info.Mode()); err != nil {
			return err
		}
		old, err := os.Lstat(filepath.Join(bin, file.name))
		if err != nil || !old.Mode().IsRegular() {
			return fmt.Errorf("existing runtime %s is not a regular file: %v", file.name, err)
		}
	}
	stage, err := os.MkdirTemp(bin, ".upgrade-")
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, os.RemoveAll(stage)) }()
	for _, file := range files {
		if err := copyRuntimeExecutableContext(ctx, file.source, filepath.Join(stage, file.name), file.info); err != nil {
			return err
		}
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := rename(filepath.Join(stage, file.name), filepath.Join(bin, file.name)); err != nil {
			return err
		}
		published = true
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f.Binaries.Source, f.Binaries.Pair, f.Binaries.Couch, f.Binaries.Helper = candidate.Source, candidate.Pair, candidate.Couch, candidate.Helper
	for i, entry := range f.Environment {
		if strings.HasPrefix(entry, "PAIR_HOME=") {
			f.Environment[i] = "PAIR_HOME=" + candidate.Source
		}
	}
	return nil
}
