package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/osfs"
)

// Repository arguments belong to the global Couch namespace, even when a thread
// is routed to a numbered slot's local backend. Never route this document by path.
func (s *ThreadStore) repoPreferencesBackend() (*ThreadStore, error) {
	if s == nil || s.namespace.Dir() == "" {
		return nil, errors.New("repository preferences have no Couch namespace")
	}
	global := NewThreadStore(s.namespace)
	global.coordinator = s.coordinator
	global.hooks = s.hooks
	global.inspection = s.inspection
	global.readOnly = s.readOnly
	return global, nil
}

func (s *ThreadStore) repoPreferencesPath() string {
	return filepath.Join(s.namespace.Dir(), "threadstore", "repo-preferences.json")
}

func (s *ThreadStore) readRepoPreferencesLocked() (RepoPreferences, error) {
	empty := RepoPreferences{SchemaVersion: RepoPreferencesSchemaVersion, Repositories: map[string]RepoLaunchPreference{}}
	raw, err := osfs.ReadRegularMetadata(context.Background(), s.repoPreferencesPath(), MaxRepoPreferenceBytes)
	if errors.Is(err, os.ErrNotExist) {
		return empty, nil
	}
	if err != nil {
		return RepoPreferences{}, fmt.Errorf("read repository arguments %s: %w", s.repoPreferencesPath(), err)
	}
	doc, err := DecodeRepoPreferences(raw)
	if err != nil {
		return RepoPreferences{}, fmt.Errorf("decode repository arguments %s: %w", s.repoPreferencesPath(), err)
	}
	return doc, nil
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

// Final adoption calls this after releasing thread/registry locks. A publication
// error is a warning on the adopted child; it never enters launch rollback.
func (s *ThreadStore) rememberRepoConfiguration(repoIdentity string, profile LaunchProfile, sources []LegacyRepoPreference) error {
	if s == nil || s.readOnly {
		return errors.New("cannot publish repository arguments through a preview store")
	}
	global, err := s.repoPreferencesBackend()
	if err != nil {
		return err
	}
	return global.withLock(func() error {
		doc, err := global.readRepoPreferencesLocked()
		if err != nil {
			return err
		}
		doc, err = foldRepoPreferenceSources(doc, sources)
		if err != nil {
			return err
		}
		var current *RepoLaunchPreference
		if found, ok := doc.Repositories[repoIdentity]; ok {
			current = &found
		}
		next, err := RememberRepoArguments(current, repoIdentity, profile)
		if err != nil {
			return err
		}
		doc.Repositories[repoIdentity] = next
		raw, err := EncodeRepoPreferences(doc)
		if err != nil {
			return fmt.Errorf("remember repository arguments %s (inspect/archive settings): %w", global.repoPreferencesPath(), err)
		}
		return global.writeStoreAtomicLocked(global.repoPreferencesPath(), raw)
	})
}

// One namespace inventory may contain sources from several repositories. Group
// it once so accepting a target records all verified fold markers without
// treating another repository's legacy values as confirmed configuration.
func foldRepoPreferenceSources(doc RepoPreferences, sources []LegacyRepoPreference) (RepoPreferences, error) {
	if len(sources) > MaxRepoPreferenceSources {
		return RepoPreferences{}, errors.New("repository preferences: too many legacy sources (inspect/archive settings)")
	}
	groups := make(map[string][]LegacyRepoPreference)
	for _, source := range sources {
		id := source.Preference.RepoIdentity
		groups[id] = append(groups[id], source)
	}
	next := RepoPreferences{SchemaVersion: doc.SchemaVersion, Unknown: doc.Unknown, Repositories: make(map[string]RepoLaunchPreference, len(doc.Repositories))}
	for id, record := range doc.Repositories {
		next.Repositories[id] = record
	}
	for id, group := range groups {
		var current *RepoLaunchPreference
		if record, ok := next.Repositories[id]; ok {
			current = &record
		}
		folded, err := FoldPathPreferences(current, id, group)
		if err != nil {
			return RepoPreferences{}, err
		}
		next.Repositories[id] = folded
	}
	if _, err := EncodeRepoPreferences(next); err != nil {
		return RepoPreferences{}, err
	}
	return next, nil
}
