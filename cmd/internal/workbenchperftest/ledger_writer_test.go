package workbenchperftest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A portable stateful stand-in for the clone operation. Native byte ceilings
// separately exercise the actual filesystem backend on Darwin.
func copyLedgerForTest(source, destination string) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, raw, 0600)
}

func TestFixturePaddingConstructionGrowth(t *testing.T) {
	const size = 64 << 10
	for _, owners := range []int{8, 16} {
		t.Run(fmt.Sprint(owners), func(t *testing.T) {
			f := newFixture(t, owners, size, copyLedgerForTest)
			var records uint64
			for _, owner := range f.Owners {
				raw, err := os.ReadFile(owner.Ledger)
				if err != nil {
					t.Fatal(err)
				}
				records += uint64(len(bytes.TrimLeft(raw, " ")))
			}
			if got := f.ledgerWrites; got.PaddingBytes != size || got.RecordBytes != records || got.Clones != uint64(owners) || got.Patches != uint64(owners) {
				t.Fatalf("construction counts %+v; want one %d-byte seed, %d record bytes and %d clones/patches", got, size, records, owners)
			}
		})
	}
}

func TestLedgerWriterExactBytesAndIndependentFiles(t *testing.T) {
	root := t.TempDir()
	w := ledgerWriter{directory: root, size: 128, clone: copyLedgerForTest, patch: patchLedgerFile, useClones: true}
	paths := []string{filepath.Join(root, "first"), filepath.Join(root, "second")}
	for i, records := range []string{"{\"owner\":1}\n", "{\"owner\":222}\n"} {
		if err := w.write(paths[i], records); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(paths[i])
		if err != nil {
			t.Fatal(err)
		}
		want := strings.Repeat(" ", 128-len(records)) + records
		if string(raw) != want {
			t.Fatalf("ledger = %q, want %q", raw, want)
		}
	}
	a, err := os.Stat(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.Stat(paths[1])
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(a, b) {
		t.Fatal("mutable ledger files share an inode")
	}
	if _, err := patchLedgerFile(paths[0], "changed", 0); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(paths[1])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(second, []byte("changed")) {
		t.Fatal("sibling changed with another ledger")
	}
	seed, err := os.ReadFile(w.template)
	if err != nil {
		t.Fatal(err)
	}
	if string(seed) != strings.Repeat(" ", 128) {
		t.Fatal("owner patch mutated the shared padding seed")
	}
}

func TestLedgerWriterSmallSizeRetainsAllRecords(t *testing.T) {
	for _, size := range []int{-int(^uint(0)>>1) - 1, -1, 0, 1, 5} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			root := t.TempDir()
			w := ledgerWriter{directory: root, size: size, useClones: true, clone: func(string, string) error {
				t.Fatal("small ledger should not clone padding")
				return nil
			}}
			path := filepath.Join(root, "ledger")
			if err := w.write(path, "record\n"); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil || string(raw) != "record\n" {
				t.Fatalf("raw=%q err=%v", raw, err)
			}
		})
	}
}

func TestLedgerWriterUnsupportedCloneStreamsAndStopsProbing(t *testing.T) {
	root := t.TempDir()
	clones := 0
	w := ledgerWriter{directory: root, size: 128, useClones: true, clone: func(string, string) error {
		clones++
		return fmt.Errorf("filesystem: %w", errLedgerCloneUnsupported)
	}}
	for _, name := range []string{"one", "two"} {
		path := filepath.Join(root, name)
		if err := w.write(path, "required\n"); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(path)
		if err != nil || string(raw) != strings.Repeat(" ", 119)+"required\n" {
			t.Fatalf("fallback raw=%q err=%v", raw, err)
		}
	}
	if clones != 1 {
		t.Fatalf("clone probes=%d, want 1", clones)
	}
}

func TestLedgerWriterFatalCloneAndPartialPatchLeaveNoLedger(t *testing.T) {
	for _, mode := range []string{"clone", "patch"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "ledger")
			failure := errors.New("owned storage failure")
			w := ledgerWriter{directory: root, size: 128, useClones: true, clone: copyLedgerForTest, patch: patchLedgerFile}
			if mode == "clone" {
				w.clone = func(_, destination string) error {
					if err := os.WriteFile(destination, []byte("partial"), 0600); err != nil {
						t.Fatal(err)
					}
					return failure
				}
			} else {
				w.patch = func(destination, records string, offset int64) (int, error) {
					n, err := patchLedgerFile(destination, records[:2], offset)
					if err != nil {
						t.Fatal(err)
					}
					return n, failure
				}
			}
			if err := w.write(path, "record\n"); !errors.Is(err, failure) {
				t.Fatalf("error=%v, want storage failure", err)
			}
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("partial ledger survived: %v", err)
			}
		})
	}
}

func TestLedgerWriterRefusesExistingDestination(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "existing")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	w := ledgerWriter{directory: root, size: 128, useClones: true, clone: func(_, destination string) error {
		return &os.PathError{Op: "clone", Path: destination, Err: os.ErrExist}
	}}
	if err := w.write(path, "record\n"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("error=%v, want existing-file refusal", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "keep" {
		t.Fatalf("existing ledger changed: %q %v", raw, err)
	}
}

func TestLedgerWriterShortPatchIsFailure(t *testing.T) {
	root := t.TempDir()
	w := ledgerWriter{directory: root, size: 128, useClones: true, clone: copyLedgerForTest, patch: func(string, string, int64) (int, error) { return 1, nil }}
	path := filepath.Join(root, "ledger")
	if err := w.write(path, "records\n"); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("error=%v, want short-write refusal", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("short-patched ledger remains: %v", err)
	}
}
