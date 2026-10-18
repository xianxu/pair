package repolint

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type repositoryCommit struct {
	parents []string
	tree    map[string][]byte
}

// fakeRepository owns actual repository facts. Commits copy index bytes and
// never change afterward; reads resolve the current refs and ancestor graph.
// It intentionally models only the clean checkout histories used by this test.
type fakeRepository struct {
	working, index map[string][]byte
	refs           map[string]string
	commits        map[string]repositoryCommit
	head, attached string
	tracking       mainTracking
	sequence       int
	indexError     error
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{working: map[string][]byte{}, index: map[string][]byte{}, refs: map[string]string{}, commits: map[string]repositoryCommit{}, attached: "refs/heads/main", tracking: mainTracking{true, ".", "refs/heads/main"}}
}
func cloneTree(tree map[string][]byte) map[string][]byte {
	copy := make(map[string][]byte, len(tree))
	for name, data := range tree {
		copy[name] = bytes.Clone(data)
	}
	return copy
}
func (f *fakeRepository) resolve(revision string) (string, error) {
	if revision == "HEAD" {
		revision = f.head
		if f.attached != "" {
			if id, ok := f.refs[f.attached]; ok {
				revision = id
			}
		}
	}
	if id, ok := f.refs[revision]; ok {
		revision = id
	}
	if _, ok := f.commits[revision]; !ok {
		return "", fmt.Errorf("unknown repository revision %q", revision)
	}
	return revision, nil
}
func (f *fakeRepository) TrackedPaths() ([]string, error) {
	if f.indexError != nil {
		return nil, f.indexError
	}
	if f.index == nil {
		return nil, fmt.Errorf("repository index unavailable")
	}
	paths := make([]string, 0, len(f.index))
	for name := range f.index {
		paths = append(paths, name)
	}
	slices.Sort(paths)
	return paths, nil
}
func (f *fakeRepository) MainTracking() (mainTracking, error) { return f.tracking, nil }
func (f *fakeRepository) RemoteMainRefs() ([]string, error) {
	refs := []string{}
	for ref := range f.refs {
		if strings.HasPrefix(ref, "refs/remotes/") && strings.HasSuffix(ref, "/main") {
			refs = append(refs, ref)
		}
	}
	slices.Sort(refs)
	return refs, nil
}
func (f *fakeRepository) PathsAt(revision string) ([]string, error) {
	id, err := f.resolve(revision)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(f.commits[id].tree))
	for name := range f.commits[id].tree {
		paths = append(paths, name)
	}
	slices.Sort(paths)
	return paths, nil
}
func (f *fakeRepository) ReadBlobs(revision string, paths []string) (map[string][]byte, error) {
	id, err := f.resolve(revision)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, name := range paths {
		source, ok := f.commits[id].tree[name]
		if !ok {
			return nil, fmt.Errorf("missing blob %s:%q", revision, name)
		}
		out[name] = bytes.Clone(source)
	}
	return out, nil
}
func (f *fakeRepository) ancestors(revision string) (map[string]bool, error) {
	id, err := f.resolve(revision)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	pending := []string{id}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[current] {
			continue
		}
		commit, ok := f.commits[current]
		if !ok {
			return nil, fmt.Errorf("unknown parent commit %q", current)
		}
		seen[current] = true
		pending = append(pending, commit.parents...)
	}
	return seen, nil
}
func (f *fakeRepository) IsAncestor(older, newer string) (bool, error) {
	id, err := f.resolve(older)
	if err != nil {
		return false, err
	}
	ancestors, err := f.ancestors(newer)
	if err != nil {
		return false, err
	}
	return ancestors[id], nil
}
func (f *fakeRepository) MergeBase(left, right string) (string, error) {
	a, err := f.ancestors(left)
	if err != nil {
		return "", err
	}
	b, err := f.ancestors(right)
	if err != nil {
		return "", err
	}
	var bases []string
	for candidate := range a {
		if !b[candidate] {
			continue
		}
		best := true
		for other := range a {
			if other == candidate || !b[other] {
				continue
			}
			ancestor, err := f.IsAncestor(candidate, other)
			if err != nil {
				return "", err
			}
			if ancestor {
				best = false
				break
			}
		}
		if best {
			bases = append(bases, candidate)
		}
	}
	if len(bases) != 1 {
		return "", fmt.Errorf("need one unambiguous merge base, found %d", len(bases))
	}
	return bases[0], nil
}
func (f *fakeRepository) PathExisted(revision, path string) (bool, error) {
	ancestors, err := f.ancestors(revision)
	if err != nil {
		return false, err
	}
	for id := range ancestors {
		if _, ok := f.commits[id].tree[path]; ok {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeRepository) Write(name string, data []byte) error {
	f.working[name] = bytes.Clone(data)
	return nil
}
func (f *fakeRepository) Stage(name string) error {
	source, ok := f.working[name]
	if ok {
		f.index[name] = bytes.Clone(source)
	} else if _, tracked := f.index[name]; tracked {
		delete(f.index, name)
	} else {
		return fmt.Errorf("cannot stage unknown path %q", name)
	}
	return nil
}
func (f *fakeRepository) Remove(name string, stage bool) error {
	if _, ok := f.working[name]; !ok {
		return os.ErrNotExist
	}
	delete(f.working, name)
	if stage {
		return f.Stage(name)
	}
	return nil
}
func (f *fakeRepository) Commit() (string, error) {
	var parents []string
	if f.head != "" {
		id, err := f.resolve("HEAD")
		if err != nil {
			return "", err
		}
		parents = []string{id}
	}
	return f.commitWithParents(parents)
}
func (f *fakeRepository) commitWithParents(parents []string) (string, error) {
	resolved := make([]string, 0, len(parents))
	for _, parent := range parents {
		id, err := f.resolve(parent)
		if err != nil {
			return "", err
		}
		resolved = append(resolved, id)
	}
	f.sequence++
	id := fmt.Sprintf("commit-%d", f.sequence)
	f.commits[id] = repositoryCommit{resolved, cloneTree(f.index)}
	f.head = id
	if f.attached != "" {
		f.refs[f.attached] = id
	}
	return id, nil
}
func (f *fakeRepository) Checkout(ref string) error {
	id, err := f.resolve(ref)
	if err != nil {
		return err
	}
	if current, err := f.resolve("HEAD"); err == nil && !maps.EqualFunc(f.index, f.commits[current].tree, bytes.Equal) {
		return fmt.Errorf("fake checkout requires clean index")
	}
	for name := range f.commits[id].tree {
		if _, untracked := f.working[name]; untracked {
			if _, tracked := f.index[name]; !tracked {
				return fmt.Errorf("fake checkout refuses untracked collision")
			}
		}
	}
	// Unsupported dirty-checkout cases refuse instead of pretending to model
	// Git's merge/collision behavior. The shared histories commit before moving.
	for name, source := range f.index {
		if !bytes.Equal(f.working[name], source) {
			return fmt.Errorf("fake checkout requires clean tracked worktree")
		}
	}
	for name := range f.index {
		delete(f.working, name)
	}
	f.index = cloneTree(f.commits[id].tree)
	for name, source := range f.index {
		f.working[name] = bytes.Clone(source)
	}
	f.head = id
	f.attached = ""
	if strings.HasPrefix(ref, "refs/heads/") {
		f.attached = ref
	}
	return nil
}
func (f *fakeRepository) MoveRef(ref, revision string) error {
	id, err := f.resolve(revision)
	if err != nil {
		return err
	}
	f.refs[ref] = id
	return nil
}
func (f *fakeRepository) ConfigureMain(config mainTracking) error { f.tracking = config; return nil }
func (f *fakeRepository) readWorking(root, name string) ([]byte, error) {
	relative, err := filepath.Rel(root, name)
	if err != nil {
		return nil, err
	}
	source, ok := f.working[filepath.ToSlash(relative)]
	if !ok {
		return nil, os.ErrNotExist
	}
	return bytes.Clone(source), nil
}

func (f *fakeRepository) CommitParents(parents []string) (string, error) {
	return f.commitWithParents(parents)
}
