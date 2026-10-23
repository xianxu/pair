package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func legacyRepoFixture(t *testing.T) (*Couch, *ProvisionFixture, string) {
	t.Helper()
	f := newProvisionFixture(t)
	s, _ := repoPreferenceStoreFixture(t)
	if err := s.withLock(func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	c := &Couch{Threads: s, Git: ExecGit{}, Path: OSPathOps{}}
	identity, err := c.resolveRepoIdentity(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	return c, f, identity
}
func legacyRepoWrite(t *testing.T, s *ThreadStore, identity, path string, argv []string) (string, []byte) {
	t.Helper()
	p := PathLaunchPreference{SchemaVersion: PathLaunchPreferenceSchemaVersion, RepoIdentity: identity, PhysicalPath: path, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": argv}, Revision: 1}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(append([]byte(" \n"), raw...), '\n')
	name := s.pathLaunchPreferencePath(identity, path)
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return name, raw
}
func TestRepoPreferenceMigrationPreviewReadOnly(t *testing.T) {
	c, f, identity := legacyRepoFixture(t)
	name, before := legacyRepoWrite(t, c.Threads, identity, f.Primary, []string{"--no-alt-screen"})
	doc, sources, err := c.previewRepositoryPreferences(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveRepoArguments(repoRecordPointer(doc, identity), "codex", nil)
	if err != nil || !got.Found || !reflect.DeepEqual(got.Argv, []string{"--no-alt-screen"}) || len(sources) != 1 {
		t.Fatalf("resolution %+v sources%d err%v", got, len(sources), err)
	}
	after, err := os.ReadFile(name)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("legacy source changed", err)
	}
	for _, name := range []string{c.Threads.repoPreferencesPath(), c.Threads.manifestPath()} {
		if _, err := os.Lstat(name); !os.IsNotExist(err) {
			t.Fatalf("preview created %s: %v", name, err)
		}
	}
}
func repoRecordPointer(doc RepoPreferences, identity string) *RepoLaunchPreference {
	p, ok := doc.Repositories[identity]
	if !ok {
		return nil
	}
	return &p
}
func TestRepoPreferenceMigrationForgedIdentityAndFilename(t *testing.T) {
	for _, kind := range []string{"identity", "filename"} {
		t.Run(kind, func(t *testing.T) {
			c, f, identity := legacyRepoFixture(t)
			if kind == "identity" {
				identity += "-forged"
			}
			name, _ := legacyRepoWrite(t, c.Threads, identity, f.Primary, []string{})
			if kind == "filename" {
				if err := os.Rename(name, filepath.Join(filepath.Dir(name), "wrong.json")); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := c.previewRepositoryPreferences(context.Background()); err == nil {
				t.Fatal("unverified source accepted")
			}
		})
	}
}

func TestRepoPreferenceMigrationEnrolledSlotsShareVerifiedIdentity(t *testing.T) {
	c, f, identity := legacyRepoFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	repository, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Threads.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(c.Threads.namespace, repository.Slots[0].Identity)
	name, before := legacyRepoWrite(t, local, identity, f.host(1), []string{"--no-alt-screen"})
	doc, sources, err := c.previewRepositoryPreferences(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveRepoArguments(repoRecordPointer(doc, identity), "codex", nil)
	if err != nil || !got.Found || len(doc.Repositories) != 1 || len(sources) != 1 {
		t.Fatalf("got %+v doc%+v sources%d err%v", got, doc, len(sources), err)
	}
	after, err := os.ReadFile(name)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("local legacy source changed", err)
	}
}
func TestRepoPreferenceMigrationLocalSourceCannotClaimOtherCheckout(t *testing.T) {
	c, f, identity := legacyRepoFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	repository, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Threads.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	local := newSlotThreadStore(c.Threads.namespace, repository.Slots[0].Identity)
	legacyRepoWrite(t, local, identity, f.Primary, []string{})
	if _, _, err := c.previewRepositoryPreferences(context.Background()); err == nil {
		t.Fatal("local source outside slot accepted")
	}
}
func TestRepoPreferenceMigrationImportedSourceNeverReadAgain(t *testing.T) {
	c, f, identity := legacyRepoFixture(t)
	name, before := legacyRepoWrite(t, c.Threads, identity, f.Primary, []string{"--no-alt-screen"})
	source, err := DecodeLegacyRepoPreference(name, before)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Threads.rememberRepoConfiguration(identity, LaunchProfile{Agent: "codex", Argv: []string{"--accepted"}}, []LegacyRepoPreference{source}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("{corrupt compatibility history"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, sources, err := c.previewRepositoryPreferences(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveRepoArguments(repoRecordPointer(doc, identity), "codex", nil)
	if err != nil || len(sources) != 0 || !reflect.DeepEqual(got.Argv, []string{"--accepted"}) {
		t.Fatalf("got %+v sources%d err%v", got, len(sources), err)
	}
}
func TestRepoPreferenceMigrationDiscoveryBound(t *testing.T) {
	c, _, _ := legacyRepoFixture(t)
	dir := filepath.Join(c.Threads.root, "path-preferences")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for n := 0; n <= MaxRepoPreferenceSources; n++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%03d.json", n)), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := c.previewRepositoryPreferences(context.Background()); err == nil || !strings.Contains(err.Error(), "inspect/archive") {
		t.Fatalf("capacity error: %v", err)
	}
}

type countedLegacyGit struct {
	ExecGit
	probes int
}

func (g *countedLegacyGit) RunContext(ctx context.Context, dir string, args ...string) (string, error) {
	g.probes++
	return g.ExecGit.RunContext(ctx, dir, args...)
}
func TestRepoPreferenceMigrationCountsSourcesOnceAndStopsAfterImport(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			c, f, identity := legacyRepoFixture(t)
			for n := 0; n < count; n++ {
				path := filepath.Join(f.Primary, fmt.Sprintf("sub%d", n))
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
				legacyRepoWrite(t, c.Threads, identity, path, []string{"--no-alt-screen"})
			}
			counted := &countedLegacyGit{}
			c.Git = counted
			_, sources, err := c.previewRepositoryPreferences(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(sources) != count || counted.probes != count {
				t.Fatalf("sources=%d Git probes=%d want%d", len(sources), counted.probes, count)
			}
			if err := c.Threads.rememberRepoConfiguration(identity, LaunchProfile{Agent: "codex", Argv: []string{"--no-alt-screen"}}, sources); err != nil {
				t.Fatal(err)
			}
			counted.probes = 0
			_, sources, err = c.previewRepositoryPreferences(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(sources) != 0 || counted.probes != 0 {
				t.Fatalf("imported sources reread: sources=%d probes=%d", len(sources), counted.probes)
			}
		})
	}
}
func TestRepoPreferenceMigrationEmptyNamespaceAndCancellationDoNotInitialize(t *testing.T) {
	store, ns := repoPreferenceStoreFixture(t)
	c := &Couch{Threads: store, Git: ExecGit{}, Path: OSPathOps{}}
	doc, sources, err := c.previewRepositoryPreferences(context.Background())
	if err != nil || len(doc.Repositories) != 0 || len(sources) != 0 {
		t.Fatalf("doc%+v sources%d err%v", doc, len(sources), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := c.previewRepositoryPreferences(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(ns.Dir())
	if err != nil || len(entries) != 0 {
		t.Fatalf("created files%v err%v", entries, err)
	}
}
func TestRepoPreferenceMigrationConflictingGlobalAndLocalSources(t *testing.T) {
	c, f, identity := legacyRepoFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	repository, err := NewOSSlotCatalog(f).Discover(context.Background(), f.Primary)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Threads.EnrollSlotRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	legacyRepoWrite(t, c.Threads, identity, f.Primary, []string{"--model", "a"})
	local := newSlotThreadStore(c.Threads.namespace, repository.Slots[0].Identity)
	legacyRepoWrite(t, local, identity, f.host(1), []string{"--model", "b"})
	doc, sources, err := c.previewRepositoryPreferences(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveRepoArguments(repoRecordPointer(doc, identity), "codex", nil)
	if err != nil || got.Found || !got.Conflict || len(sources) != 2 {
		t.Fatalf("got%+v sources%d err%v", got, len(sources), err)
	}
}
func TestRepoPreferenceMigrationDirectoryFIFORefusesWithoutWaiting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := readRepoPreferenceDirectory(context.Background(), path); result <- err }()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("FIFO accepted as directory")
		}
	case <-time.After(200 * time.Millisecond):
		fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer unix.Close(fd)
		<-result
		t.Fatal("discovery blocked on a FIFO")
	}
}

func TestRepoPreferenceMigrationBoundsAggregateReadBytes(t *testing.T) {
	c, f, identity := legacyRepoFixture(t)
	for n := 0; n < 3; n++ {
		path := filepath.Join(f.Primary, fmt.Sprintf("large%d", n))
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
		name, raw := legacyRepoWrite(t, c.Threads, identity, path, []string{})
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		fields["future"] = json.RawMessage(`"` + strings.Repeat("x", MaxRepoPreferenceBytes/2+1) + `"`)
		raw, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	counted := &countedLegacyGit{}
	c.Git = counted
	if _, _, err := c.previewRepositoryPreferences(context.Background()); err == nil {
		t.Fatal("aggregate oversized sources accepted")
	}
	if counted.probes > 1 {
		t.Fatalf("read beyond aggregate byte budget: %d fully decoded sources", counted.probes)
	}
}
