package couchcore

import (
	"testing"
	"time"
)

func TestOperationsDeclareTheirExecutionOwner(t *testing.T) {
	want := map[string]OperationExecution{
		"start":               ExecuteLiveOwner,
		"list":                ExecuteDirectStore,
		"show":                ExecuteDirectStore,
		"stop":                ExecuteLiveOwner,
		"name":                ExecuteDirectStore,
		"describe":            ExecuteDirectStore,
		"publish-description": ExecuteDirectStore,
		"switch":              ExecuteLiveOwner,
		"attach":              ExecuteLiveOwner,
	}
	for _, op := range Operations() {
		if op.Execution == ExecuteUnknown {
			t.Errorf("%s has fail-open zero execution owner", op.Name)
		}
		if op.Execution != want[op.Name] {
			t.Errorf("%s execution = %v, want %v", op.Name, op.Execution, want[op.Name])
		}
	}
}

func TestStartOperationDefaultsEmptyPathToDot(t *testing.T) {
	cwd := NormalizePath(".")
	env := newTestEnv(t, cwd)
	result, err := dispatchTestOperation(env.Couch, "start", map[string]string{})
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
	}, OperationCall{Name: name, Args: args, Implicit: true})
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

func TestMetadataOperationsMutateCompositeThreadRecord(t *testing.T) {
	env := newTestEnv(t, "/repo")
	created := createOperationThread(t, env.Couch)

	result, err := dispatchTestOperation(env.Couch, "name", map[string]string{
		"ref": string(created.Address.Tag), "name": "compiler", "repo-scope": created.Address.RepoScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	named, ok := result.(ThreadRecord)
	if !ok || named.Name != "compiler" {
		t.Fatalf("name result = %#v", result)
	}

	result, err = dispatchTestOperation(env.Couch, "describe", map[string]string{
		"ref": string(created.Address.Tag), "description": "operator context", "repo-scope": created.Address.RepoScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	described := result.(ThreadRecord)
	if described.Name != "compiler" || described.Description != "operator context" || described.PublishedSummary != "agent summary" {
		t.Fatalf("describe crossed metadata fields: %+v", described)
	}

	result, err = dispatchTestOperation(env.Couch, "describe", map[string]string{
		"ref": string(created.Address.Tag), "description": "", "repo-scope": created.Address.RepoScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	cleared, ok := result.(ThreadRecord)
	if !ok || cleared.Description != "" || cleared.Name != "compiler" {
		t.Fatalf("explicit empty description did not clear only that field: %#v", result)
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
}
