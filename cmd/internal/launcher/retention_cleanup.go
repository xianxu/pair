package launcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

// RemoveOwnerSessionBindings shares root coordination with all binding
// publishers. A scoped owner also removes matching compatibility-root rows.
func RemoveOwnerSessionBindings(held *storagegc.Locked, owner artifactpath.StorageOwner) error {
	if !held.Writable(owner.DataDir) {
		return errors.New("binding cleanup requires writable root coordination")
	}
	dirs := []string{owner.DataDir}
	if owner.Directory() != owner.DataDir {
		dirs = append(dirs, owner.Directory())
	}
	for _, dir := range dirs {
		scope, err := artifactpath.ResolveSelectedScope(dir)
		if err != nil {
			return err
		}
		path := scope.SessionBindings()
		st, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !st.Mode().IsRegular() || st.Size() > 16<<20 {
			return errors.New("unsafe session binding index")
		}
		// The directory was inventoried before detachment. Reject any intervening
		// alias before following the shared metadata path.
		canonical, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return err
		}
		if canonical != dir {
			return errors.New("aliased binding directory")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		index, err := DecodeSessionNameIndex(string(raw))
		if err != nil {
			return err
		}
		var lines []string
		changed := false
		for _, entry := range index.Entries {
			if entry.Tag == owner.Tag && (owner.RepoScope == "" || entry.ScopeKey == owner.RepoScope) {
				changed = true
				continue
			}
			line, err := BuildSessionNameIndexLine(entry)
			if err != nil {
				return err
			}
			lines = append(lines, line)
		}
		if !changed {
			continue
		}
		content := ""
		if len(lines) > 0 {
			content = strings.Join(lines, "\n") + "\n"
		}
		if err := (OSRuntime{}).WriteAtomic(path, content); err != nil {
			return err
		}
		payload, err := os.Open(path)
		if err != nil {
			return err
		}
		if err := errors.Join(payload.Sync(), payload.Close()); err != nil {
			return err
		}
		f, err := os.Open(dir)
		if err != nil {
			return err
		}
		err = errors.Join(f.Sync(), f.Close())
		if err != nil {
			return err
		}
	}
	return nil
}
