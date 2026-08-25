package couchcore

import "testing"

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
