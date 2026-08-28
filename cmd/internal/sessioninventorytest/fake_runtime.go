// Package sessioninventorytest provides a stateful implementation of
// sessioninventory.Runtime for scanner and integration tests.
package sessioninventorytest

import (
	"fmt"
	"sort"
	"sync"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type Operation string

const (
	OperationListFiles Operation = "list_files"
	OperationReadFile  Operation = "read_file"
	OperationSQLite    Operation = "sqlite"
	OperationOpenFiles Operation = "open_files"
)

type storedFile struct {
	entry   sessioninventory.FileEntry
	content []byte
}

type sqliteKey struct {
	artifact string
	query    string
}

type processState struct {
	identity string
	children []string
	open     []string
}

type FakeRuntime struct {
	mu sync.RWMutex

	roots        map[sessioninventory.Agent][]sessioninventory.StorageRoot
	pairRoot     sessioninventory.StorageRoot
	files        map[string]storedFile
	listingOrder map[string][]string
	sqlite       map[sqliteKey]sessioninventory.SQLiteResult
	processes    map[string]processState
	errors       map[string]error
}

var _ sessioninventory.Runtime = (*FakeRuntime)(nil)

func NewFakeRuntime() *FakeRuntime {
	return &FakeRuntime{
		roots:        make(map[sessioninventory.Agent][]sessioninventory.StorageRoot),
		files:        make(map[string]storedFile),
		listingOrder: make(map[string][]string),
		sqlite:       make(map[sqliteKey]sessioninventory.SQLiteResult),
		processes:    make(map[string]processState),
		errors:       make(map[string]error),
	}
}

func (f *FakeRuntime) AddRoot(root sessioninventory.StorageRoot) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.roots[root.Agent] = append(f.roots[root.Agent], root)
}

func (f *FakeRuntime) SetPairDataRoot(root sessioninventory.StorageRoot) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pairRoot = root
}

func (f *FakeRuntime) PutFile(entry sessioninventory.FileEntry, content []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entry.Size = int64(len(content))
	f.files[artifactKey(entry.Artifact)] = storedFile{entry: cloneFileEntry(entry), content: append([]byte(nil), content...)}
}

func (f *FakeRuntime) SetListingOrder(storageRoot string, relativePaths []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listingOrder[storageRoot] = append([]string(nil), relativePaths...)
}

func (f *FakeRuntime) PutSQLite(artifact sessioninventory.Artifact, query string, result sessioninventory.SQLiteResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sqlite[sqliteKey{artifact: artifactKey(artifact), query: query}] = cloneSQLite(result)
}

func (f *FakeRuntime) SetProcess(pid, identity string, children, openFiles []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.processes[pid] = processState{
		identity: identity,
		children: append([]string(nil), children...),
		open:     append([]string(nil), openFiles...),
	}
}

func (f *FakeRuntime) SetError(operation Operation, key string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	mapKey := errorKey(operation, key)
	if err == nil {
		delete(f.errors, mapKey)
		return
	}
	f.errors[mapKey] = err
}

func (f *FakeRuntime) NativeRoots(agent sessioninventory.Agent) []sessioninventory.StorageRoot {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]sessioninventory.StorageRoot(nil), f.roots[agent]...)
}

func (f *FakeRuntime) PairDataRoot() sessioninventory.StorageRoot {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.pairRoot
}

func (f *FakeRuntime) ListFiles(root sessioninventory.StorageRoot) ([]sessioninventory.FileEntry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	listingErr := f.errors[errorKey(OperationListFiles, root.Name)]
	byPath := make(map[string]sessioninventory.FileEntry)
	for _, stored := range f.files {
		if stored.entry.Artifact.StorageRoot == root.Name {
			byPath[stored.entry.Artifact.RelativePath] = cloneFileEntry(stored.entry)
		}
	}
	paths := append([]string(nil), f.listingOrder[root.Name]...)
	seen := make(map[string]struct{}, len(paths))
	result := make([]sessioninventory.FileEntry, 0, len(byPath))
	for _, relativePath := range paths {
		entry, ok := byPath[relativePath]
		if !ok {
			continue
		}
		result = append(result, entry)
		seen[relativePath] = struct{}{}
	}
	var remainder []string
	for relativePath := range byPath {
		if _, ok := seen[relativePath]; !ok {
			remainder = append(remainder, relativePath)
		}
	}
	sort.Strings(remainder)
	for _, relativePath := range remainder {
		result = append(result, byPath[relativePath])
	}
	return result, listingErr
}

func (f *FakeRuntime) ReadFile(artifact sessioninventory.Artifact, limit int64) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := artifactKey(artifact)
	if err := f.errors[errorKey(OperationReadFile, key)]; err != nil {
		return nil, err
	}
	stored, ok := f.files[key]
	if !ok {
		return nil, fmt.Errorf("read %s: file not found", key)
	}
	if limit < 0 || int64(len(stored.content)) > limit {
		return nil, sessioninventory.ErrReadLimit
	}
	return append([]byte(nil), stored.content...), nil
}

func (f *FakeRuntime) ReadAt(artifact sessioninventory.Artifact, offset, limit int64) ([]byte, bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := artifactKey(artifact)
	if err := f.errors[errorKey(OperationReadFile, key)]; err != nil {
		return nil, false, err
	}
	stored, ok := f.files[key]
	if !ok {
		return nil, false, fmt.Errorf("read %s: file not found", key)
	}
	if offset < 0 || limit < 0 {
		return nil, false, sessioninventory.ErrReadLimit
	}
	if offset >= int64(len(stored.content)) {
		return nil, true, nil
	}
	end := offset + limit
	if end < offset || end > int64(len(stored.content)) {
		end = int64(len(stored.content))
	}
	return append([]byte(nil), stored.content[offset:end]...), end == int64(len(stored.content)), nil
}

func (f *FakeRuntime) QuerySQLite(artifact sessioninventory.Artifact, query string, limit int64) (sessioninventory.SQLiteResult, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	key := artifactKey(artifact)
	if err := f.errors[errorKey(OperationSQLite, key)]; err != nil {
		return sessioninventory.SQLiteResult{}, err
	}
	result, ok := f.sqlite[sqliteKey{artifact: key, query: query}]
	if !ok {
		return sessioninventory.SQLiteResult{}, fmt.Errorf("query %s: result not found", key)
	}
	cloned := cloneSQLite(result)
	var size int64
	for _, column := range cloned.Columns {
		size += int64(len(column))
	}
	for _, row := range cloned.Rows {
		for _, value := range row {
			size += int64(len(value))
		}
	}
	if limit < 0 || size > limit {
		return sessioninventory.SQLiteResult{}, sessioninventory.ErrReadLimit
	}
	return cloned, nil
}

func (f *FakeRuntime) ProcessChildren() map[string][]string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make(map[string][]string, len(f.processes))
	for pid, state := range f.processes {
		result[pid] = append([]string(nil), state.children...)
	}
	return result
}

func (f *FakeRuntime) ProcessIdentity(pid string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.processes[pid].identity
}

func (f *FakeRuntime) OpenFiles(pid string) []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.errors[errorKey(OperationOpenFiles, pid)] != nil {
		return nil
	}
	return append([]string(nil), f.processes[pid].open...)
}

func artifactKey(artifact sessioninventory.Artifact) string {
	return artifact.StorageRoot + ":" + artifact.RelativePath
}

func errorKey(operation Operation, key string) string {
	return string(operation) + ":" + key
}

func cloneFileEntry(entry sessioninventory.FileEntry) sessioninventory.FileEntry {
	cloned := entry
	if entry.BirthTime != nil {
		value := *entry.BirthTime
		cloned.BirthTime = &value
	}
	if entry.ModTime != nil {
		value := *entry.ModTime
		cloned.ModTime = &value
	}
	return cloned
}

func cloneSQLite(result sessioninventory.SQLiteResult) sessioninventory.SQLiteResult {
	cloned := sessioninventory.SQLiteResult{Columns: append([]string(nil), result.Columns...), Rows: make([][]string, len(result.Rows))}
	for i, row := range result.Rows {
		cloned.Rows[i] = append([]string(nil), row...)
	}
	return cloned
}
