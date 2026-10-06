package couchcore

import (
	"os"
	"path/filepath"
	"testing"
)

// depsWorld builds a slot environment on disk: the host and each named
// checkout with an optional construct/deps and base.manifest.
type depsWorld struct {
	t   *testing.T
	env string
}

func newDepsWorld(t *testing.T) depsWorld {
	t.Helper()
	env, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return depsWorld{t: t, env: env}
}

func (w depsWorld) checkout(name, deps string, layer bool) string {
	w.t.Helper()
	dir := filepath.Join(w.env, name)
	if err := os.MkdirAll(filepath.Join(dir, "construct"), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if deps != "" {
		if err := os.WriteFile(filepath.Join(dir, "construct", "deps"), []byte(deps), 0o644); err != nil {
			w.t.Fatal(err)
		}
	}
	if layer {
		if err := os.WriteFile(filepath.Join(dir, "construct", "base.manifest"), []byte("# layer\n"), 0o644); err != nil {
			w.t.Fatal(err)
		}
	}
	return dir
}

func TestDeclaredDepsTransitiveWithAbsent(t *testing.T) {
	w := newDepsWorld(t)
	host := w.checkout("pair", "substrate ../a https://example.invalid/a.git\ndata https://example.invalid/d.git data/d\n", false)
	w.checkout("a", "substrate ../b\n", true)
	got, err := DeclaredDepsOf(w.env, host)
	if err != nil {
		t.Fatal(err)
	}
	if got.NotLayer != nil {
		t.Fatalf("unexpected not-layer %+v", got.NotLayer)
	}
	want := []DeclaredDep{
		{Rel: "a", Path: filepath.Join(w.env, "a"), Owner: host, Source: "https://example.invalid/a.git", Present: true},
		{Rel: "b", Path: filepath.Join(w.env, "b"), Owner: filepath.Join(w.env, "a"), Present: false},
	}
	if len(got.Deps) != len(want) {
		t.Fatalf("deps = %+v, want %+v", got.Deps, want)
	}
	for i := range want {
		if got.Deps[i] != want[i] {
			t.Errorf("dep %d = %+v, want %+v", i, got.Deps[i], want[i])
		}
	}
}

func TestDeclaredDepsNoDeclaration(t *testing.T) {
	w := newDepsWorld(t)
	host := w.checkout("pair", "", false)
	got, err := DeclaredDepsOf(w.env, host)
	if err != nil || len(got.Deps) != 0 || got.NotLayer != nil {
		t.Fatalf("got %+v, %v; want no dependencies", got, err)
	}
}

func TestDeclaredDepsMalformedIsAnError(t *testing.T) {
	w := newDepsWorld(t)
	host := w.checkout("pair", "substrate\n", false)
	if _, err := DeclaredDepsOf(w.env, host); err == nil {
		t.Fatal("a malformed row must be an error (deps unknown), never no dependencies")
	}
}

func TestDeclaredDepsPresentNonLayer(t *testing.T) {
	w := newDepsWorld(t)
	host := w.checkout("pair", "substrate ../ariadne\n", false)
	w.checkout("ariadne", "", false) // half-cloned: present, no base.manifest
	got, err := DeclaredDepsOf(w.env, host)
	if err != nil {
		t.Fatal(err)
	}
	if got.NotLayer == nil || got.NotLayer.Rel != "ariadne" || got.NotLayer.Owner != host || !got.NotLayer.Present {
		t.Fatalf("NotLayer = %+v, want the present non-layer ariadne declared by the host", got.NotLayer)
	}
}

func TestDeclaredDepsOutsideTheEnvironment(t *testing.T) {
	w := newDepsWorld(t)
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	host := w.checkout("pair", "substrate "+outside+"\n", false)
	// An outside non-layer stops the walk, and is marked Outside so reconcile
	// never sets it aside: it is not slot state.
	got, err := DeclaredDepsOf(w.env, host)
	if err != nil || got.NotLayer == nil || !got.NotLayer.Outside || got.NotLayer.Rel != "" {
		t.Fatalf("got %+v, %v; want an Outside not-layer", got, err)
	}
	if err := os.MkdirAll(filepath.Join(outside, "construct"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "construct", "base.manifest"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = DeclaredDepsOf(w.env, host)
	if err != nil || len(got.Deps) != 1 || !got.Deps[0].Outside || got.Deps[0].Rel != "" || !got.Deps[0].Present {
		t.Fatalf("got %+v, %v; want one present dependency marked Outside", got, err)
	}
}
