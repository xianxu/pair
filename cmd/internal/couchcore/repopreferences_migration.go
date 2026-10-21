package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/osfs"
	"golang.org/x/sys/unix"
)

// previewRepositoryPreferences imports only Couch's compatibility records into
// an in-memory view. Persisting their provenance belongs to final adoption.
func (c *Couch) previewRepositoryPreferences(ctx context.Context) (RepoPreferences, []LegacyRepoPreference, error) {
	if err := ctx.Err(); err != nil {
		return RepoPreferences{}, nil, err
	}
	if c == nil || c.Threads == nil {
		return RepoPreferences{}, nil, errors.New("repository preferences: thread store unavailable")
	}
	global, err := c.Threads.repoPreferencesBackend()
	if err != nil {
		return RepoPreferences{}, nil, err
	}
	global.readOnly = true
	doc := RepoPreferences{SchemaVersion: RepoPreferencesSchemaVersion, Repositories: map[string]RepoLaunchPreference{}}
	var manifest threadManifest
	err = global.withPreviewLockContext(ctx, func() error {
		var err error
		doc, err = global.readRepoPreferencesLocked()
		if err != nil {
			return err
		}
		manifest, _, _, err = global.loadManifestLockedContext(ctx)
		return err
	})
	if err != nil {
		return RepoPreferences{}, nil, err
	}
	imported := map[string]bool{}
	for _, repo := range doc.Repositories {
		for source := range repo.LegacySources {
			imported[source] = true
		}
	}
	paths, err := readRepoPreferenceDirectory(ctx, filepath.Join(global.root, "path-preferences"))
	if err != nil {
		return RepoPreferences{}, nil, err
	}
	var sources []LegacyRepoPreference
	remainingBytes := int64(MaxRepoPreferenceBytes)
	for _, entry := range paths {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := filepath.Join(global.root, "path-preferences", entry.Name())
		if imported[name] {
			continue
		}
		source, exists, err := c.readLegacyRepoPreference(ctx, name, nil, remainingBytes)
		if err != nil {
			return RepoPreferences{}, nil, err
		}
		if !exists {
			continue
		}
		if name != global.pathLaunchPreferencePath(source.Preference.RepoIdentity, source.Preference.PhysicalPath) {
			return RepoPreferences{}, nil, fmt.Errorf("legacy preference filename does not match its identity: %s", name)
		}
		sources = append(sources, source)
		remainingBytes -= int64(len(source.Raw))
	}
	if len(manifest.SlotRepositories) > MaxRepoPreferenceRepos {
		return RepoPreferences{}, nil, errors.New("repository preference enrollment exceeds 256 repositories; inspect/archive settings")
	}
	// Inventory remains a location hint. Only an independently observed common
	// Git directory plus retained checkout containment authorizes each source.
	directories := map[string][]os.DirEntry{}
	for _, root := range manifest.SlotRepositories {
		if err := ctx.Err(); err != nil {
			return RepoPreferences{}, nil, err
		}
		if err := checkSlotDirectory(root); err != nil {
			return RepoPreferences{}, nil, err
		}
		directory := WorktreesRoot(filepath.Dir(root))
		entries, ok := directories[directory]
		if !ok {
			entries, err = readRepoPreferenceDirectory(ctx, directory)
			if err != nil {
				return RepoPreferences{}, nil, err
			}
			directories[directory] = entries
		}
		for _, entry := range entries {
			number, matches := ParseEnvName(filepath.Base(root), entry.Name())
			if !matches {
				continue
			}
			slot := conventionalSlot(root, number)
			slot.RepoIdentity = retainedSlotCommonGit(manifest, root)
			if _, err := inspectSlot(slot); err != nil {
				return RepoPreferences{}, nil, err
			}
			name := newSlotThreadStore(global.namespace, slot).pathLaunchPreferencePath("", "")
			if imported[name] {
				continue
			}
			source, exists, err := c.readLegacyRepoPreference(ctx, name, &slot, remainingBytes)
			if err != nil {
				return RepoPreferences{}, nil, err
			}
			if !exists {
				continue
			}
			if len(sources) >= MaxRepoPreferenceSources {
				return RepoPreferences{}, nil, errors.New("repository preference discovery exceeds 256 sources; inspect/archive settings")
			}
			sources = append(sources, source)
			remainingBytes -= int64(len(source.Raw))
		}
	}
	folded, err := foldRepoPreferenceSources(doc, sources)
	if err != nil {
		return RepoPreferences{}, nil, err
	}
	return folded, sources, nil
}

// Bound allocation and traversal together: oversized directories require manual
// inspection/archive rather than an unbounded scan on each form preview.
func readRepoPreferenceDirectory(ctx context.Context, path string) ([]os.DirEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := provisionSafePath(path); err != nil {
		return nil, err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	entries, readErr := f.ReadDir(MaxRepoPreferenceSources + 1)
	closeErr := f.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(entries) > MaxRepoPreferenceSources {
		return nil, fmt.Errorf("repository preference discovery exceeds %d entries in %s; inspect/archive settings", MaxRepoPreferenceSources, path)
	}
	return entries, ctx.Err()
}
func (c *Couch) readLegacyRepoPreference(ctx context.Context, path string, slot *SlotIdentity, remainingBytes int64) (LegacyRepoPreference, bool, error) {
	if err := provisionSafePath(path); err != nil {
		return LegacyRepoPreference{}, false, err
	}
	if remainingBytes <= 0 {
		return LegacyRepoPreference{}, false, errors.New("repository preference discovery exceeds 4 MiB; inspect/archive settings")
	}
	raw, err := osfs.ReadRegularMetadata(ctx, path, remainingBytes)
	if errors.Is(err, os.ErrNotExist) {
		return LegacyRepoPreference{}, false, nil
	}
	if err != nil {
		return LegacyRepoPreference{}, false, fmt.Errorf("repository preference discovery cannot read %s within 4 MiB; inspect/archive settings: %w", path, err)
	}
	source, err := DecodeLegacyRepoPreference(path, raw)
	if err != nil {
		return source, false, fmt.Errorf("legacy preference %s: %w", path, err)
	}
	p := source.Preference
	observed, err := c.resolveRepoIdentity(ctx, p.PhysicalPath)
	if err != nil {
		return source, false, err
	}
	if observed != p.RepoIdentity {
		return source, false, fmt.Errorf("legacy preference repository identity is unverified: %s", path)
	}
	if slot != nil {
		if err := retainedPathWithinCheckout(slot.WorktreeRoot, p.PhysicalPath); err != nil {
			return source, false, err
		}
		_, belongs, err := CheckoutMembership(slot.RepoIdentity, slot.WorktreeRoot, p.PhysicalPath, "", observed)
		if err != nil {
			return source, false, err
		}
		if !belongs {
			return source, false, fmt.Errorf("legacy preference is outside retained slot authority: %s", path)
		}
	}
	return source, true, nil
}
