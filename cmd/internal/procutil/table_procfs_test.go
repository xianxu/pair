package procutil

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type procfsTableReader struct {
	root string
}

func (r procfsTableReader) Table() ([]Process, error) { return readLinuxTable(r.root) }
func (r procfsTableReader) Command(int) (CommandInfo, error) {
	panic("unknown table rows must not be enriched")
}
func (r procfsTableReader) Identity(int) (string, error) {
	panic("unknown table rows must not be enriched")
}

func TestLinuxTablePreservesUnreadableListedPIDs(t *testing.T) {
	for _, failure := range []string{"stat read", "stat parse"} {
		t.Run(failure, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "42")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			stat := filepath.Join(dir, "stat")
			if failure == "stat read" {
				// EISDIR is deterministic even when the test runs as root.
				if err := os.Mkdir(stat, 0700); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(stat, []byte("malformed stat"), 0600); err != nil {
				t.Fatal(err)
			}
			source := NewSource(procfsTableReader{root: root})
			rows, err := source.Table()
			if err != nil || !reflect.DeepEqual(rows, []Process{{PID: 42}}) {
				t.Errorf("unreadable PID was lost or acquired false metadata: %+v, %v", rows, err)
			}
			observed, err := source.SnapshotNamed(context.Background(), "zellij")
			if err != nil || len(observed) != 1 || observed[0].PID != 42 || observed[0].Err == nil {
				t.Fatalf("named capture lost uncertainty: %+v, %v", observed, err)
			}
		})
	}
}

func TestLinuxTableDistinguishesDisappearanceAndTerminalRows(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"gone", "0", "41", "42"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	// 41 vanished between listing and reading; 42 is an unreaped terminal PID.
	if err := os.WriteFile(filepath.Join(root, "42", "stat"), []byte("42 (zellij) Z 3 0 0 0 0 0 0 0 0 0 0 0 0 0 0 7 0 0 123"), 0600); err != nil {
		t.Fatal(err)
	}
	rows, err := readLinuxTable(root)
	want := []Process{{PID: 42, PPID: 3, Name: "zellij", Identity: "123", Nice: 7, Zombie: true}}
	if err != nil || !reflect.DeepEqual(rows, want) {
		t.Fatalf("table = %+v, %v; want %+v", rows, err, want)
	}
	if _, err := readLinuxTable(filepath.Join(root, "absent")); !os.IsNotExist(err) {
		t.Fatalf("unavailable census = %v", err)
	}
}
