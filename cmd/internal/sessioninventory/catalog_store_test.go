package sessioninventory

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestCatalogStoreRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "session-inventory-catalog.json")
	store := CatalogStore{Runtime: CatalogOSRuntime{}}
	written, err := store.Update(path, func(current Catalog) (Catalog, error) {
		current.Entries = append(current.Entries, storeTestEntry("a.jsonl"))
		return current, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if written.Generation != 1 || CatalogOutcomeOf(err) != CatalogCommitted {
		t.Fatalf("written = %#v, outcome=%v", written, CatalogOutcomeOf(err))
	}
	read, err := store.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Generation != 1 || len(read.Entries) != 1 || read.Entries[0].Artifact.RelativePath != "a.jsonl" {
		t.Fatalf("read = %#v", read)
	}
}

func TestCatalogStoreRecomputesConcurrentUpdatesFromLockedGeneration(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "session-inventory-catalog.json")
	store := CatalogStore{Runtime: CatalogOSRuntime{}}
	var wait sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 2)
	for _, name := range []string{"a.jsonl", "b.jsonl"} {
		name := name
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			_, err := store.Update(path, func(current Catalog) (Catalog, error) {
				current.Entries = append(current.Entries, storeTestEntry(name))
				return current, nil
			})
			errs <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	read, err := store.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Generation != 2 || len(read.Entries) != 2 {
		t.Fatalf("read = %#v, want two serialized generations", read)
	}
}

func TestCatalogStorePublicationOutcomesMatchRecoveryAuthority(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		failure     catalogStoreFailure
		wantOutcome CatalogCommitOutcome
		wantCatalog bool
	}{
		{name: "write", failure: catalogFailWrite, wantOutcome: CatalogNotAuthoritative},
		{name: "file sync", failure: catalogFailFileSync, wantOutcome: CatalogNotAuthoritative},
		{name: "rename", failure: catalogFailRename, wantOutcome: CatalogNotAuthoritative},
		{name: "directory sync", failure: catalogFailDirectorySync, wantOutcome: CatalogIndeterminate, wantCatalog: true},
		{name: "unlock", failure: catalogFailUnlock, wantOutcome: CatalogCommitted, wantCatalog: true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "session-inventory-catalog.json")
			runtime := &catalogFailureRuntime{failure: test.failure}
			store := CatalogStore{Runtime: runtime}
			_, err := store.Update(path, func(current Catalog) (Catalog, error) {
				current.Entries = append(current.Entries, storeTestEntry("a.jsonl"))
				return current, nil
			})
			if err == nil {
				t.Fatal("Update succeeded at injected failure")
			}
			if got := CatalogOutcomeOf(err); got != test.wantOutcome {
				t.Fatalf("outcome = %v, want %v (err %v)", got, test.wantOutcome, err)
			}
			catalog, readErr := (CatalogStore{Runtime: CatalogOSRuntime{}}).Read(path)
			if test.wantCatalog {
				if readErr != nil || catalog.Generation != 1 || len(catalog.Entries) != 1 {
					t.Fatalf("published catalog = %#v, %v", catalog, readErr)
				}
			} else if !errors.Is(readErr, os.ErrNotExist) {
				t.Fatalf("pre-publication failure left authority: %#v, %v", catalog, readErr)
			}
		})
	}
}

type catalogStoreFailure string

const (
	catalogFailWrite         catalogStoreFailure = "write"
	catalogFailFileSync      catalogStoreFailure = "file-sync"
	catalogFailRename        catalogStoreFailure = "rename"
	catalogFailDirectorySync catalogStoreFailure = "directory-sync"
	catalogFailUnlock        catalogStoreFailure = "unlock"
)

type catalogFailureRuntime struct {
	CatalogOSRuntime
	failure catalogStoreFailure
}

func (r *catalogFailureRuntime) Lock(path string) (CatalogUnlocker, error) {
	lock, err := r.CatalogOSRuntime.Lock(path)
	if err != nil {
		return nil, err
	}
	return catalogFailureUnlocker{CatalogUnlocker: lock, fail: r.failure == catalogFailUnlock}, nil
}

func (r *catalogFailureRuntime) CreateTemp(dir, pattern string) (CatalogFile, error) {
	file, err := r.CatalogOSRuntime.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &catalogFailureFile{CatalogFile: file, failure: r.failure}, nil
}

func (r *catalogFailureRuntime) Rename(oldPath, newPath string) error {
	if r.failure == catalogFailRename {
		return errors.New("injected rename failure")
	}
	return r.CatalogOSRuntime.Rename(oldPath, newPath)
}

func (r *catalogFailureRuntime) SyncDirectory(path string) error {
	if r.failure == catalogFailDirectorySync {
		return errors.New("injected directory sync failure")
	}
	return r.CatalogOSRuntime.SyncDirectory(path)
}

type catalogFailureFile struct {
	CatalogFile
	failure catalogStoreFailure
}

func (f *catalogFailureFile) Write(raw []byte) (int, error) {
	if f.failure == catalogFailWrite {
		return 0, errors.New("injected write failure")
	}
	return f.CatalogFile.Write(raw)
}

func (f *catalogFailureFile) Sync() error {
	if f.failure == catalogFailFileSync {
		return errors.New("injected file sync failure")
	}
	return f.CatalogFile.Sync()
}

type catalogFailureUnlocker struct {
	CatalogUnlocker
	fail bool
}

func (u catalogFailureUnlocker) Close() error {
	err := u.CatalogUnlocker.Close()
	if u.fail {
		return errors.Join(err, io.ErrClosedPipe)
	}
	return err
}

func storeTestEntry(relativePath string) CatalogEntry {
	return CatalogEntry{
		Agent:            AgentClaude,
		Artifact:         Artifact{StorageRoot: "claude-projects", RelativePath: relativePath, Kind: ArtifactTranscript},
		Fingerprint:      ArtifactFingerprint{StableFileID: StableFileID(relativePath), GenerationToken: "gen:1", MutationToken: "ctime:1", Size: 0},
		Authorization:    AuthorizationCandidate,
		ScannerSchema:    "claude-v1",
		ProviderContract: ProviderClaudeJSONLV1,
	}
}
