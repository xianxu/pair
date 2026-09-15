package storagegc

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

// RootInventory contains paths and filesystem metadata only. Unknown entries
// are explicit blockers, never payloads inspected for a guessed owner.
type RootInventory struct {
	Groups   []artifactpath.ArtifactGroup
	Unknown  []string
	Complete bool
	Entries  int
}

type inventoryNamespace struct {
	scope        string
	paths, names []string
	blockers     []string
}

func InventoryRoot(root string, known []artifactpath.StorageOwner, agents []string, limit int) (RootInventory, error) {
	return InventoryRootExcluding(root, known, agents, limit, nil)
}
func InventoryRootExcluding(root string, known []artifactpath.StorageOwner, agents []string, limit int, excluded []string) (RootInventory, error) {
	return inventoryRootContext(context.Background(), root, known, agents, limit, excluded, nil)
}

func inventoryRootContext(ctx context.Context, root string, known []artifactpath.StorageOwner, agents []string, limit int, excluded []string, pending func(string, string) error) (RootInventory, error) {
	result := RootInventory{Complete: true}
	if limit <= 0 {
		return result, errors.New("positive inventory budget required")
	}
	if err := checkDirectory(root, false); err != nil {
		return result, err
	}
	exclude := map[string]bool{}
	for _, path := range excluded {
		if path == root {
			return result, errors.New("external store overlaps entire Pair root")
		}
		exclude[path] = true
	}
	spaces := map[string]*inventoryNamespace{"": {}}
	var walk func(string, string) error
	walk = func(dir, scope string) error {
		f, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer f.Close()
		for {
			if result.Entries >= limit {
				result.Complete = false
				return nil
			}
			n := min(256, limit-result.Entries)
			names, err := f.Readdirnames(n)
			if err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			result.Entries += len(names)
			for _, name := range names {
				if err := ctx.Err(); err != nil {
					return err
				}
				path := filepath.Join(dir, name)
				if exclude[path] {
					continue
				}
				if dir == root && name == ".retention" {
					continue
				}
				st, e := os.Lstat(path)
				if e != nil {
					return e
				}
				if dir == root && name == "repos" {
					if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
						result.Unknown = append(result.Unknown, path)
						continue
					}
					// The repos container is a namespace boundary, not an owner directory.
					sf, e := os.Open(path)
					if e != nil {
						return e
					}
					for {
						if result.Entries >= limit {
							result.Complete = false
							break
						}
						scopes, re := sf.Readdirnames(min(256, limit-result.Entries))
						if re != nil && !errors.Is(re, io.EOF) {
							sf.Close()
							return re
						}
						result.Entries += len(scopes)
						for _, key := range scopes {
							if err := ctx.Err(); err != nil {
								sf.Close()
								return err
							}
							child := filepath.Join(path, key)
							if _, e := artifactpath.NewStorageOwner(root, key, "validation"); e != nil {
								result.Unknown = append(result.Unknown, child)
								continue
							}
							if e := checkDirectory(child, false); e != nil {
								result.Unknown = append(result.Unknown, child)
								continue
							}
							spaces[key] = &inventoryNamespace{scope: key}
							if e := walk(child, key); e != nil {
								sf.Close()
								return e
							}
						}
						if errors.Is(re, io.EOF) {
							break
						}
					}
					sf.Close()
					continue
				}
				ownerDirectory := root
				if scope != "" {
					ownerDirectory = filepath.Join(root, "repos", scope)
				}
				if dir == ownerDirectory && isPendingMetadata(name) {
					if !st.Mode().IsRegular() {
						return errors.New("unsafe unpublished metadata")
					}
					if pending != nil {
						if err := pending(dir, name); err != nil {
							return err
						}
					}
					continue
				}
				ns := spaces[scope]
				ns.paths = append(ns.paths, path)
				ownerDir := root
				if scope != "" {
					ownerDir = filepath.Join(root, "repos", scope)
				}
				if dir == ownerDir {
					ns.names = append(ns.names, name)
				}
				if st.Mode()&os.ModeSymlink != 0 || (!st.IsDir() && !st.Mode().IsRegular()) {
					ns.blockers = append(ns.blockers, "unsafe artifact type: "+path)
					continue
				}
				if st.IsDir() {
					if e := walk(path, scope); e != nil {
						return e
					}
				}
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
		}
	}
	if err := walk(root, ""); err != nil {
		return result, err
	}
	for scope, ns := range spaces {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		owners, err := artifactpath.DiscoverStorageOwners(root, scope, ns.names)
		if err != nil {
			return result, err
		}
		seen := map[artifactpath.StorageOwner]bool{}
		for _, o := range owners {
			seen[o] = true
		}
		for _, o := range known {
			if o.DataDir != root || o.RepoScope != scope {
				continue
			}
			if !seen[o] {
				owners = append(owners, o)
				seen[o] = true
			}
		}
		index, err := artifactpath.NewMatchIndex(owners, agents)
		if err != nil {
			return result, err
		}
		groups := map[artifactpath.StorageOwner]*artifactpath.ArtifactGroup{}
		for _, o := range owners {
			groups[o] = &artifactpath.ArtifactGroup{Owner: o, Blockers: append([]string(nil), ns.blockers...)}
		}
		probe, _ := artifactpath.NewStorageOwner(root, scope, "validation")
		for _, path := range ns.paths {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			if artifactpath.SharedStorageArtifact(probe, path, agents) {
				continue
			}
			if index.DiagnosticLock(path) {
				info, err := os.Lstat(path)
				if err != nil {
					return result, err
				}
				if info.Mode().IsRegular() && info.Size() == 0 {
					continue
				}
			}
			member, err := index.Match(path)
			if err != nil {
				result.Unknown = append(result.Unknown, path)
				for _, g := range groups {
					g.Blockers = append(g.Blockers, err.Error())
				}
				continue
			}
			groups[member.Owner].Members = append(groups[member.Owner].Members, member)
		}
		for _, g := range groups {
			sort.Slice(g.Members, func(i, j int) bool { return g.Members[i].Path < g.Members[j].Path })
			result.Groups = append(result.Groups, *g)
		}
	}
	if len(result.Unknown) > 0 {
		for i := range result.Groups {
			result.Groups[i].Blockers = append(result.Groups[i].Blockers, "unrecognized entries in Pair root")
		}
	}
	sort.Slice(result.Groups, func(i, j int) bool { return result.Groups[i].Owner.Key() < result.Groups[j].Owner.Key() })
	sort.Strings(result.Unknown)
	return result, nil
}
