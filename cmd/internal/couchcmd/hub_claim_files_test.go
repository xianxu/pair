package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/durablefile"
)

func TestHubClaimFilesReadAndValidate(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := ReadHubClaimFile(root)
	if err != nil || len(doc.Bytes()) != 0 {
		t.Fatalf("%v %v", doc, err)
	}
	path := filepath.Join(root, ".hub")
	if err := os.WriteFile(path, []byte("pair:7\n"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err = ReadHubClaimFile(root)
	if err != nil || len(doc.Claims()) != 1 {
		t.Fatalf("%v %v", doc, err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "private"), path); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadHubClaimFile(root); err == nil {
		t.Fatal("symlink accepted")
	}
}
func TestHubClaimFilesPartialWriteIsTruthfulAndRetryable(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	for _, p := range []string{a, b} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(b, ".hub"), []byte("pair:7\n"), 0600); err != nil {
		t.Fatal(err)
	}
	edits := []couchcore.HubClaimEdit{{Root: a, After: []byte("pair:7\n")}, {Root: b, Before: []byte("pair:7\n"), After: []byte{}}}
	files := hubClaimFiles{write: func(path string, raw []byte, mode os.FileMode) error {
		if filepath.Dir(path) == b {
			return errors.New("disk full")
		}
		return durablefile.WriteAtomicMode(path, raw, ".hub-", mode)
	}}
	err := files.Apply(context.Background(), edits)
	if err == nil || !strings.Contains(err.Error(), b) || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("%v", err)
	}
	got, _ := os.ReadFile(filepath.Join(a, ".hub"))
	if string(got) != "pair:7\n" {
		t.Fatal("lost target evidence")
	}
	if err := (hubClaimFiles{}).Apply(context.Background(), edits); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(b, ".hub"))
	if len(got) != 0 {
		t.Fatal("source not reconciled")
	}
}
func TestHubClaimFilesConflictPreflightPreventsWrites(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	for _, p := range []string{a, b} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(b, ".hub"), []byte("human\n"), 0600); err != nil {
		t.Fatal(err)
	}
	edits := []couchcore.HubClaimEdit{{Root: a, After: []byte("pair:7\n")}, {Root: b, Before: []byte("pair:7\n"), After: []byte{}}}
	if err := (hubClaimFiles{}).Apply(context.Background(), edits); err == nil {
		t.Fatal("overwrote human edit")
	}
	if _, err := os.Stat(filepath.Join(a, ".hub")); !os.IsNotExist(err) {
		t.Fatalf("target changed: %v", err)
	}
}
func TestHubClaimFilesLostWriteAcknowledgment(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	edit := couchcore.HubClaimEdit{Root: root, After: []byte("pair:7\n")}
	files := hubClaimFiles{write: func(path string, raw []byte, mode os.FileMode) error {
		if err := durablefile.WriteAtomicMode(path, raw, ".hub-", mode); err != nil {
			return err
		}
		return errors.New("reply lost")
	}}
	if err := files.Apply(context.Background(), []couchcore.HubClaimEdit{edit}); err == nil {
		t.Fatal("claimed confirmed write")
	}
	if err := (hubClaimFiles{}).Apply(context.Background(), []couchcore.HubClaimEdit{edit}); err != nil {
		t.Fatal(err)
	}
}
