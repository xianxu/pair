package couchcore

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

// These helpers construct and inspect legacy/private preference fixtures.
// Production resolves repository arguments through the bounded migration seam;
// it neither writes legacy path defaults nor exposes a second preference reader.

func RecordSuccessfulLaunch(current *PathLaunchPreference, repoIdentity, physicalPath string, profile LaunchProfile) (PathLaunchPreference, error) {
	if repoIdentity == "" || !filepath.IsAbs(physicalPath) {
		return PathLaunchPreference{}, fmt.Errorf("successful launch requires repository identity and absolute path")
	}
	if profile.Agent == "" || profile.Argv == nil {
		return PathLaunchPreference{}, fmt.Errorf("successful launch has incomplete profile")
	}
	var next PathLaunchPreference
	if current == nil {
		next = PathLaunchPreference{
			SchemaVersion: PathLaunchPreferenceSchemaVersion,
			RepoIdentity:  repoIdentity, PhysicalPath: physicalPath,
			ArgvByAgent: map[string][]string{}, Revision: 1,
		}
	} else {
		if err := validatePathLaunchPreference(*current); err != nil {
			return PathLaunchPreference{}, err
		}
		if current.RepoIdentity != repoIdentity || current.PhysicalPath != physicalPath {
			return PathLaunchPreference{}, fmt.Errorf("path launch preference address mismatch")
		}
		next = clonePathLaunchPreference(*current)
		next.Revision++
	}
	next.LastAgent = profile.Agent
	next.ArgvByAgent[profile.Agent] = cloneArgv(profile.Argv)
	if err := validatePathLaunchPreference(next); err != nil {
		return PathLaunchPreference{}, err
	}
	return next, nil
}

func (s *ThreadStore) GetPathLaunchPreference(repoIdentity, physicalPath string) (PathLaunchPreference, bool, error) {
	return s.getPathLaunchPreference(repoIdentity, physicalPath, "")
}

// getPathLaunchPreference accepts checkout scope already proved by an operation's
// catalog, so a read-only preview can inspect legacy enrollment before backfill.
func (s *ThreadStore) getPathLaunchPreference(repoIdentity, physicalPath, scope string) (PathLaunchPreference, bool, error) {
	if repoIdentity == "" {
		return PathLaunchPreference{}, false, errors.New("path launch preference has no repository identity")
	}
	if !filepath.IsAbs(physicalPath) {
		return PathLaunchPreference{}, false, errors.New("path launch preference path must be absolute")
	}
	backend, routeErr := s.storeForPath(physicalPath, scope, repoIdentity)
	if routeErr != nil {
		return PathLaunchPreference{}, false, routeErr
	}
	if backend != s {
		return backend.getPathLaunchPreference(repoIdentity, physicalPath, scope)
	}
	var result PathLaunchPreference
	var found bool
	err := s.withLock(func() error {
		raw, exists, err := s.readOptionalPayload(s.pathLaunchPreferencePath(repoIdentity, physicalPath))
		if err != nil || !exists {
			return err
		}
		if err := strictThreadStoreJSON(raw, &result); err != nil {
			return err
		}
		if err := validatePathLaunchPreference(result); err != nil {
			return err
		}
		if result.RepoIdentity != repoIdentity || result.PhysicalPath != physicalPath {
			return errors.New("path launch preference path/address mismatch")
		}
		result = clonePathLaunchPreference(result)
		found = true
		return nil
	})
	return result, found, err
}

func (s *ThreadStore) PreviewPathLaunchPreference(repoIdentity, path, scope string) (PathLaunchPreference, bool, error) {
	if s == nil {
		return PathLaunchPreference{}, false, errors.New("thread store unavailable")
	}
	view := *s
	view.readOnly = true
	return view.getPathLaunchPreference(repoIdentity, path, scope)
}

// A preview cannot initialize a root/lock, enroll retention, or replay journals.
func (s *ThreadStore) previewRepoPreferences() (RepoPreferences, error) {
	global, err := s.repoPreferencesBackend()
	if err != nil {
		return RepoPreferences{}, err
	}
	global.readOnly = true
	doc := RepoPreferences{SchemaVersion: RepoPreferencesSchemaVersion, Repositories: map[string]RepoLaunchPreference{}}
	err = global.withPreviewLock(func() error { var readErr error; doc, readErr = global.readRepoPreferencesLocked(); return readErr })
	return doc, err
}

// Test fixtures accept configuration directly; production publishes only through
// final adoption with its shared migration/publication deadline.
func (s *ThreadStore) rememberRepoConfiguration(repoIdentity string, profile LaunchProfile, sources []LegacyRepoPreference) error {
	return s.rememberRepoConfigurationContext(context.Background(), repoIdentity, profile, sources)
}
