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
	var start Operation
	for _, op := range Operations() {
		if op.Name == "start" {
			start = op
			break
		}
	}
	if start.Invoke == nil {
		t.Fatal("start operation not found")
	}

	result, err := start.Invoke(env.Couch, map[string]string{})
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

func operationNamed(t *testing.T, name string) Operation {
	t.Helper()
	for _, op := range Operations() {
		if op.Name == name {
			return op
		}
	}
	t.Fatalf("operation %q not found", name)
	return Operation{}
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

	result, err := operationNamed(t, "name").Invoke(env.Couch, map[string]string{
		"ref": string(created.Address.Tag), "name": "compiler", "repo-scope": created.Address.RepoScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	named, ok := result.(ThreadRecord)
	if !ok || named.Name != "compiler" {
		t.Fatalf("name result = %#v", result)
	}

	result, err = operationNamed(t, "describe").Invoke(env.Couch, map[string]string{
		"ref": string(created.Address.Tag), "description": "operator context", "repo-scope": created.Address.RepoScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	described := result.(ThreadRecord)
	if described.Name != "compiler" || described.Description != "operator context" || described.PublishedSummary != "agent summary" {
		t.Fatalf("describe crossed metadata fields: %+v", described)
	}
}

func TestPublishDescriptionUsesExactCompositeContextAndDistinctField(t *testing.T) {
	env := newTestEnv(t, "/repo")
	created := createOperationThread(t, env.Couch)

	result, err := operationNamed(t, "publish-description").Invoke(env.Couch, map[string]string{
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
