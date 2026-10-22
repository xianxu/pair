package editorendpoint

import (
	"bytes"
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
