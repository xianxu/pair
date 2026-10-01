package couchcore

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
)

// repositoryAliasFile is its own payload rather than a manifest field: the
// manifest decodes strictly, so a new field there would make the whole store
// unreadable to an older couch binary, and slots build their own.
//
// Lifecycle: SetRepositoryAlias creates and rewrites it; it holds at most one
// entry per enrolled repository; clearing removes the entry, and entries for
// roots no longer enrolled are ignored on read and dropped on the next write.
type repositoryAliasFile struct {
	SchemaVersion int               `json:"schema_version"`
	Aliases       map[string]string `json:"aliases"`
}

const repositoryAliasSchema = 1

func (s *ThreadStore) repositoryAliasPath() string {
	return filepath.Join(s.root, "repository-aliases.json")
}

// RepositoryNames lists every enrolled repository with its alias. It reads the
// root store whichever backend it is called on.
func (s *ThreadStore) RepositoryNames() ([]RepositoryName, error) {
	root := s.familyRootStore()
	view := *root
	view.readOnly = true
	var names []RepositoryName
	err := view.withPreviewLock(func() error {
		var err error
		names, _, err = view.repositoryNamesLocked()
		return err
	})
	return names, err
}

// SetRepositoryAlias sets (or, with "", clears) the alias of the enrolled
// repository at primaryRoot. Validation reads the same locked state it writes.
func (s *ThreadStore) SetRepositoryAlias(primaryRoot, alias string) error {
	root := s.familyRootStore()
	return root.withLock(func() error {
		names, aliases, err := root.repositoryNamesLocked()
		if err != nil {
			return err
		}
		if err := ValidateRepositoryAlias(alias, primaryRoot, names); err != nil {
			return err
		}
		next := map[string]string{}
		for _, name := range names {
			if a := aliases[name.Key]; a != "" && name.Key != primaryRoot {
				next[name.Key] = a
			}
		}
		if alias != "" {
			next[primaryRoot] = alias
		}
		raw, err := json.MarshalIndent(repositoryAliasFile{SchemaVersion: repositoryAliasSchema, Aliases: next}, "", "  ")
		if err != nil {
			return err
		}
		return root.writeStoreAtomicLocked(root.repositoryAliasPath(), append(raw, '\n'))
	})
}

func (s *ThreadStore) repositoryNamesLocked() ([]RepositoryName, map[string]string, error) {
	manifest, _, _, err := s.loadManifestLocked()
	if err != nil {
		return nil, nil, err
	}
	aliases := map[string]string{}
	raw, exists, err := s.readOptionalPayload(s.repositoryAliasPath())
	if err != nil {
		return nil, nil, err
	}
	if exists {
		var file repositoryAliasFile
		invalid := func(reason string) error {
			return fmt.Errorf("repository aliases in %s are invalid (%s); remove the file to reset aliases", s.repositoryAliasPath(), reason)
		}
		if err := strictThreadStoreJSON(raw, &file); err != nil {
			return nil, nil, invalid(err.Error())
		}
		if file.SchemaVersion != repositoryAliasSchema || file.Aliases == nil {
			return nil, nil, invalid("unsupported schema")
		}
		aliases = file.Aliases
	}
	roots := append([]string(nil), manifest.SlotRepositories...)
	sort.Strings(roots)
	names := make([]RepositoryName, 0, len(roots))
	for _, root := range roots {
		names = append(names, RepositoryName{Key: root, Dir: filepath.Base(root), Alias: aliases[root]})
	}
	// A stored alias that no longer validates (a later enrollment took the
	// name) is withheld rather than allowed to shadow a directory.
	for i := range names {
		if names[i].Alias == "" {
			continue
		}
		alias := names[i].Alias
		names[i].Alias = ""
		if ValidateRepositoryAlias(alias, names[i].Key, names) == nil {
			names[i].Alias = alias
		}
	}
	return names, aliases, nil
}
