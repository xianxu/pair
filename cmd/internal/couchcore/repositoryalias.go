package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// RepositoryAliasResult reports one repository's alias after a read or write.
type RepositoryAliasResult struct {
	Repository string
	Alias      string
}

// RepositoryAlias reads (alias nil) or sets the alias of the repository that
// ref names; "" clears it. ref is anything a slot reference accepts.
func (c *Couch) RepositoryAlias(ctx context.Context, ref string, alias *string) (RepositoryAliasResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.Threads == nil || c.Slots == nil {
		return RepositoryAliasResult{}, errors.New("repository aliases need the couch store")
	}
	target, identity, err := c.resolveSlotInput(ctx, ref)
	if err != nil {
		return RepositoryAliasResult{}, err
	}
	primary := identity.PrimaryRoot
	if target != nil {
		primary = target.Slot.PrimaryRoot
	}
	if primary == "" {
		return RepositoryAliasResult{}, fmt.Errorf("%s is not inside a Git repository", ref)
	}
	if alias != nil {
		// An existing directory of that name beside the repository would win
		// over the alias in every reference, so the alias could never work.
		if *alias != "" && workspaceRepoName(*alias) {
			if _, err := os.Lstat(filepath.Join(filepath.Dir(primary), *alias)); !errors.Is(err, fs.ErrNotExist) {
				return RepositoryAliasResult{}, fmt.Errorf("alias %q would be shadowed by %s", *alias, filepath.Join(filepath.Dir(primary), *alias))
			}
		}
		if err := c.Threads.SetRepositoryAlias(primary, *alias); err != nil {
			return RepositoryAliasResult{}, err
		}
	}
	names, err := c.Threads.RepositoryNames()
	if err != nil {
		return RepositoryAliasResult{}, err
	}
	for _, name := range names {
		if name.Key == primary {
			return RepositoryAliasResult{Repository: name.Dir, Alias: name.Alias}, nil
		}
	}
	return RepositoryAliasResult{}, fmt.Errorf("%w: %s is not enrolled; open it in couch first", ErrRepositoryNotFound, primary)
}
