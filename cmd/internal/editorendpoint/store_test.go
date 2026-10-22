package editorendpoint

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEndpointStoreRetainsAuthorityAcrossInterruptedPublication(t *testing.T) {
	namespace := endpointTestNamespace(t)
	store := EndpointStore{Namespace: namespace, Path: filepath.Join(namespace, "editor-endpoints.json"), Stage: filepath.Join(namespace, "editor-endpoints.stage")}
	state, err := store.Load()
	if err != nil || state.Namespace != namespace || state.Version != 1 {
		t.Fatalf("empty store: %+v %v", state, err)
	}
	if err = store.Publish(state); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(store.Stage, []byte("interrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Lstat(store.Stage); !os.IsNotExist(err) {
		t.Fatalf("stage remains: %v", err)
	}
	got, err := os.ReadFile(store.Path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatalf("cleanup changed authority: %v", err)
	}
	if err = os.WriteFile(store.Path, []byte("truncated"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(store.Stage, []byte("retain evidence"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Load(); err == nil {
		t.Fatal("damaged committed authority silently reset")
	}
	if _, err = os.Stat(store.Stage); err != nil {
		t.Fatal("damaged authority lost its stage evidence", err)
	}
}

func TestEndpointStoreRefusesUnsafeFilesAndStages(t *testing.T) {
	for _, stage := range []bool{false, true} {
		namespace := endpointTestNamespace(t)
		store := EndpointStore{Namespace: namespace, Path: filepath.Join(namespace, "editor-endpoints.json"), Stage: filepath.Join(namespace, "editor-endpoints.stage")}
		target := filepath.Join(namespace, "outside")
		if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		path := store.Path
		if stage {
			path = store.Stage
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Load(); err == nil {
			t.Fatal("symlink accepted", path)
		}
		got, err := os.ReadFile(target)
		if err != nil || string(got) != "keep" {
			t.Fatal("outside target changed", err)
		}
	}
}

func endpointTestNamespace(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEndpointStoreOpaqueNumbersSurviveActualReopenRewrite(t *testing.T) {
	namespace := endpointTestNamespace(t)
	store := EndpointStore{Namespace: namespace, Path: filepath.Join(namespace, "state.json"), Stage: filepath.Join(namespace, "stage.json")}
	s := endpointOpaqueNumberState(t)
	s.Namespace = namespace
	for i := range s.Records {
		s.Records[i].Binding.Namespace = namespace
		if s.Records[i].Current != nil {
			s.Records[i].Current.Namespace = namespace
		}
	}
	if e := store.Publish(s); e != nil {
		t.Fatal(e)
	}
	original, e := os.ReadFile(store.Path)
	if e != nil {
		t.Fatal(e)
	}
	reopened := EndpointStore{Namespace: namespace, Path: store.Path, Stage: store.Stage}
	n, e := reopened.Load()
	if e != nil {
		t.Fatal(e)
	}
	if e = reopened.Publish(n); e != nil {
		t.Fatal(e)
	}
	again, e := os.ReadFile(store.Path)
	if e != nil || !bytes.Equal(original, again) {
		t.Fatalf("actual reopen rewrite changed opaque numbers: %v", e)
	}
	for level, unknown := range []map[string]json.RawMessage{n.Unknown, n.Records[0].Unknown, n.Records[0].Binding.Unknown, n.Records[0].Current.Unknown} {
		if !bytes.Equal(unknown["future"], s.Unknown["future"]) {
			t.Fatalf("level%d number bytes changed", level)
		}
	}
}
