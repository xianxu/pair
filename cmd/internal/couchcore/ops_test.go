package couchcore

import (
	"context"
	"testing"
	"time"
)

func TestStartOperationDefaultsEmptyPathToDot(t *testing.T) {
	cwd := NormalizePath(".")
	env := newTestEnv(t, cwd)
	preparedValue, err := dispatchTestOperation(env.Couch, "prepare-start", map[string]string{})
	if err != nil {
		t.Fatalf("prepare-start empty path: %v", err)
	}
	prepared := preparedValue.(PreparedStart)
	result, err := dispatchTestOperation(env.Couch, "start", prepared.Resolution.CommitArgs())
	if err != nil {
		t.Fatalf("start empty path: %v", err)
	}
	got, ok := result.(StartResult)
	if !ok {
		t.Fatalf("result = %T, want StartResult", result)
	}
	if got.Record.Args.Cwd != cwd {
		t.Fatalf("record cwd = %q, want %q", got.Record.Args.Cwd, cwd)
	}
	if launch := env.Runner.Child(got.Handle.ID()); launch.Dir != cwd {
		t.Fatalf("launch dir = %q, want %q", launch.Dir, cwd)
	}
}

func dispatchTestOperation(c *Couch, name string, args map[string]string) (any, error) {
	return DispatchOperation(OperationExecutors{
		DirectStore: DirectStoreExecutor(c),
		LiveOwner:   CouchLiveOwnerExecutor(c),
	}, OperationCall{Name: name, Args: args, Implicit: true, Context: context.Background()})
}

func createOperationThread(t *testing.T, c *Couch) ThreadRecord {
	t.Helper()
	record := metadataThread("816fc349d3faebf8", "couch-0102030405060708", "/repo/task", "")
	record.CreatedAt = time.Unix(2, 0).UTC()
	created, err := c.Threads.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

// The name and describe operations left the switcher and the CLI (pair#363);
// the stored fields stay, written through ApplyThreadMetadata, and a patch of
// one field must not cross into another.
func TestThreadMetadataPatchesDoNotCrossFields(t *testing.T) {
	env := newTestEnv(t, "/repo")
	created := createOperationThread(t, env.Couch)
	name, description := "compiler", "operator context"
	named, err := env.Couch.ApplyThreadMetadata(created.Address, ThreadMetadataPatch{Name: &name})
	if err != nil || named.Name != "compiler" {
		t.Fatalf("name patch = %+v, %v", named, err)
	}
	described, err := env.Couch.ApplyThreadMetadata(created.Address, ThreadMetadataPatch{Description: &description})
	if err != nil {
		t.Fatal(err)
	}
	if described.Name != "compiler" || described.Description != "operator context" || described.PublishedSummary != "agent summary" {
		t.Fatalf("describe crossed metadata fields: %+v", described)
	}
}

func TestCompositeReferenceOperationsRefuseEmptyRepositoryScope(t *testing.T) {
	env := newTestEnv(t, "/repo")
	created := createOperationThread(t, env.Couch)
	for _, call := range []struct {
		name string
		args map[string]string
	}{
		{"show", map[string]string{"ref": string(created.Address.Tag), "repo-scope": ""}},
		{"name", map[string]string{"ref": string(created.Address.Tag), "name": "x", "repo-scope": ""}},
		{"describe", map[string]string{"ref": string(created.Address.Tag), "repo-scope": ""}},
	} {
		if _, err := dispatchTestOperation(env.Couch, call.name, call.args); err == nil {
			t.Errorf("%s accepted an empty collision domain", call.name)
		}
	}
}

func TestPublishDescriptionUsesExactCompositeContextAndDistinctField(t *testing.T) {
	env := newTestEnv(t, "/repo")
	created := createOperationThread(t, env.Couch)

	result, err := dispatchTestOperation(env.Couch, "publish-description", map[string]string{
		"description": "agent is fixing parser",
		"repo-scope":  created.Address.RepoScope,
		"tag":         string(created.Address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	published := result.(ThreadRecord)
	if published.PublishedSummary != "agent is fixing parser" || published.Description != "operator description" {
		t.Fatalf("published metadata = %+v", published)
	}

	result, err = dispatchTestOperation(env.Couch, "publish-description", map[string]string{
		"description": "",
		"repo-scope":  created.Address.RepoScope,
		"tag":         string(created.Address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	published = result.(ThreadRecord)
	if published.PublishedSummary != "" || published.Description != "operator description" {
		t.Fatalf("explicit empty published summary did not clear only that field: %+v", published)
	}
}
