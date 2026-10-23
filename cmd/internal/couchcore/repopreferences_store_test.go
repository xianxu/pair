package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestRepoPreferencePublicationFoldsNamespaceSourcesBeforeAcceptance(t *testing.T) {
	store, _ := repoPreferenceStoreFixture(t)
	var sources []LegacyRepoPreference
	for _, repo := range []string{"/repo/.git", "/other/.git"} {
		raw, err := json.Marshal(PathLaunchPreference{SchemaVersion: 1, RepoIdentity: repo, PhysicalPath: filepath.Dir(repo), LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {"legacy"}}, Revision: 1})
		if err != nil {
			t.Fatal(err)
		}
		source, err := DecodeLegacyRepoPreference(repo+"/source", raw)
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, source)
	}
	if err := store.rememberRepoConfiguration("/repo/.git", LaunchProfile{Agent: "codex", Argv: []string{"accepted"}}, sources); err != nil {
		t.Fatal(err)
	}
	doc, err := store.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	for _, repo := range []string{"/repo/.git", "/other/.git"} {
		p := doc.Repositories[repo]
		if len(p.LegacySources) != 1 {
			t.Fatalf("source lost for %s: %+v", repo, p)
		}
		if p.Agents["codex"].Confirmed != (repo == "/repo/.git") {
			t.Fatalf("acceptance leaked: %+v", p)
		}
	}
	if got := doc.Repositories["/repo/.git"].Agents["codex"].Argv; !reflect.DeepEqual(got, []string{"accepted"}) {
		t.Fatal(got)
	}
}

func repoPreferenceStoreFixture(t *testing.T) (*ThreadStore, CouchNamespace) {
	t.Helper()
	ns, err := ResolveCouchNamespace(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	return NewThreadStore(ns), ns
}

func TestRepoPreferencePreviewDoesNotInitializeOrWrite(t *testing.T) {
	store, ns := repoPreferenceStoreFixture(t)
	doc, err := store.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != RepoPreferencesSchemaVersion || len(doc.Repositories) != 0 {
		t.Fatal(doc)
	}
	files, err := os.ReadDir(ns.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("preview created files: %v", files)
	}
}

func TestRepoPreferenceStoreSharesGlobalDocumentWithLocalBackend(t *testing.T) {
	global, ns := repoPreferenceStoreFixture(t)
	profile := LaunchProfile{Agent: "codex", Argv: []string{"--model", "model-a"}}
	if err := global.rememberRepoConfiguration("/repo/.git", profile, nil); err != nil {
		t.Fatal(err)
	}
	local := &ThreadStore{namespace: ns, root: filepath.Join(t.TempDir(), "local-couch"), layout: StoreLayout{Local: true}}
	doc, err := local.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	got := doc.Repositories["/repo/.git"].Agents["codex"]
	if !got.Confirmed || !reflect.DeepEqual(got.Argv, profile.Argv) {
		t.Fatal(got)
	}
	if _, err := os.Lstat(local.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("local metadata touched: %v", err)
	}
	if err := local.rememberRepoConfiguration("/repo/.git", LaunchProfile{Agent: "claude", Argv: []string{}}, nil); err != nil {
		t.Fatal(err)
	}
	updated, err := global.previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if args, ok := updated.Repositories["/repo/.git"].Agents["claude"]; !ok || args.Argv == nil || len(args.Argv) != 0 {
		t.Fatal(updated)
	}
}

func TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents(t *testing.T) {
	_, ns := repoPreferenceStoreFixture(t)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for n := 0; n < 8; n++ {
		for _, agent := range []string{"claude", "codex"} {
			wg.Add(1)
			go func(n int, agent string) {
				defer wg.Done()
				errs <- NewThreadStore(ns).rememberRepoConfiguration(fmt.Sprintf("/repo-%d/.git", n), LaunchProfile{Agent: agent, Argv: []string{"--model", fmt.Sprint(n)}}, nil)
			}(n, agent)
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	doc, err := NewThreadStore(ns).previewRepoPreferences()
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Repositories) != 8 {
		t.Fatal(len(doc.Repositories))
	}
	for _, repo := range doc.Repositories {
		if len(repo.Agents) != 2 {
			t.Fatal(repo)
		}
	}
}

func TestRepoPreferencePublicationFailurePreservesPriorBytes(t *testing.T) {
	store, _ := repoPreferenceStoreFixture(t)
	if err := store.rememberRepoConfiguration("/repo/.git", LaunchProfile{Agent: "codex", Argv: []string{}}, nil); err != nil {
		t.Fatal(err)
	}
	prior, err := os.ReadFile(store.repoPreferencesPath())
	if err != nil {
		t.Fatal(err)
	}
	store.hooks.AfterPublicationWrite = func(string) error { return errors.New("private fixture write refusal") }
	if err := store.rememberRepoConfiguration("/repo/.git", LaunchProfile{Agent: "codex", Argv: []string{"--model", "new"}}, nil); err == nil {
		t.Fatal("publication succeeded")
	}
	after, err := os.ReadFile(store.repoPreferencesPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(prior) {
		t.Fatal("failed publication changed prior bytes")
	}
}

func TestRepoPreferenceReadOnlyStoreCannotPublish(t *testing.T) {
	store, _ := repoPreferenceStoreFixture(t)
	store.readOnly = true
	if err := store.rememberRepoConfiguration("/repo/.git", LaunchProfile{Agent: "codex", Argv: []string{}}, nil); err == nil {
		t.Fatal("preview published")
	}
	if _, err := os.Lstat(store.root); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("read-only publication initialized store", err)
	}
}
