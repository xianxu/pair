package terminalcapture

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sys/unix"
)

func TestCaptureStorageReservesAcrossLaunchesAndPreservesEvidence(t *testing.T) {
	root := filepath.Join(t.TempDir(), "captures")
	first, f, err := openCaptureFileWithBudget(root, 100, 200, 64)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("saved evidence"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	second, f, err := openCaptureFileWithBudget(root, 100, 200, 64)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openCaptureFileWithBudget(root, 1, 200, 64); err == nil {
		t.Fatal("third reservation exceeded aggregate budget")
	}
	data, err := os.ReadFile(filepath.Join(first, "events.jsonl"))
	if err != nil || string(data) != "saved evidence" {
		t.Fatalf("evidence changed: %q, %v", data, err)
	}
	// Moving a saved session out of this root explicitly frees its reservation.
	if err := os.Rename(second, filepath.Join(t.TempDir(), "retained")); err != nil {
		t.Fatal(err)
	}
	_, f, err = openCaptureFileWithBudget(root, 100, 200, 64)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestCaptureStorageSessionCountAndPrivateFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "captures")
	dir, f, err := openCaptureFileWithBudget(root, 1, 100, 1)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	for path, mode := range map[string]os.FileMode{root: 0700, dir: 0700, filepath.Join(root, ".capture.lock"): 0600, filepath.Join(dir, "budget.json"): 0600, filepath.Join(dir, "events.jsonl"): 0600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != mode {
			t.Errorf("%s mode = %o, want %o", path, info.Mode().Perm(), mode)
		}
	}
	if _, _, err := openCaptureFileWithBudget(root, 1, 100, 1); err == nil {
		t.Fatal("session count exceeded")
	}
}

func TestCaptureStorageConcurrentAdmissionAndBusyLock(t *testing.T) {
	root := t.TempDir()
	lock, err := os.OpenFile(filepath.Join(root, ".capture.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	if _, _, err := openCaptureFileWithBudget(root, 100, 100, 64); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("busy lock: %v", err)
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	admitted := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, f, err := openCaptureFileWithBudget(root, 100, 100, 64)
			if err == nil {
				f.Close()
				mu.Lock()
				admitted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if admitted != 1 {
		t.Fatalf("admitted = %d, want exactly one", admitted)
	}
	if _, err := os.Stat(filepath.Join(root, ".capture.lock")); err != nil {
		t.Fatalf("persistent lock removed: %v", err)
	}
}

func TestCaptureStorageRejectsUnknownOrUnsafeSessions(t *testing.T) {
	for _, tc := range []struct {
		name, budget, events string
		symlink              string
	}{
		{name: "unfinished legacy", events: "{\"version\":1,\"kind\":\"capture-start\"}\n"},
		{name: "empty crashed directory"},
		{name: "corrupt metadata", budget: "{"},
		{name: "duplicate reservation key", budget: `{"schema_version":1,"max_bytes":999,"max_bytes":100}`},
		{name: "duplicate legacy status", events: "{\"version\":1,\"kind\":\"capture-end\",\"status\":\"unknown\",\"status\":\"complete\"}\n"},
		{name: "unknown schema", budget: `{"schema_version":2,"max_bytes":100}`},
		{name: "unknown metadata field", budget: `{"schema_version":1,"max_bytes":100,"extra":true}`},
		{name: "trailing metadata", budget: `{"schema_version":1,"max_bytes":100}{}`},
		{name: "oversized metadata", budget: strings.Repeat(" ", 4097)},
		{name: "negative reservation", budget: `{"schema_version":1,"max_bytes":-1}`},
		{name: "bytes exceed reservation", budget: `{"schema_version":1,"max_bytes":1}`, events: "too much"},
		{name: "truncated legacy end", events: `{"version":1,"kind":"capture-end","status":"complete"}`},
		{name: "unknown legacy schema", events: "{\"version\":2,\"kind\":\"capture-end\",\"status\":\"complete\"}\n"},
		{name: "unknown legacy status", events: "{\"version\":1,\"kind\":\"capture-end\",\"status\":\"unknown\"}\n"},
		{name: "symlink session", symlink: "session"},
		{name: "symlink budget", symlink: "budget"},
		{name: "symlink events", symlink: "events"},
		{name: "symlink lock", symlink: "lock"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "session-saved")
			target := t.TempDir()
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if tc.budget != "" {
				if err := os.WriteFile(filepath.Join(dir, "budget.json"), []byte(tc.budget), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name != "empty crashed directory" && tc.symlink != "session" && tc.symlink != "events" {
				if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte(tc.events), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var link string
			switch tc.symlink {
			case "session":
				os.Remove(dir)
				link = dir
			case "budget":
				link = filepath.Join(dir, "budget.json")
			case "events":
				link = filepath.Join(dir, "events.jsonl")
			case "lock":
				link = filepath.Join(root, ".capture.lock")
			}
			if link != "" {
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := openCaptureFileWithBudget(root, 10, 1000, 64); err == nil {
				t.Fatal("unsafe/unknown session accepted")
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.Name() != "session-saved" && entry.Name() != ".capture.lock" {
					t.Errorf("failed admission created %s", entry.Name())
				}
			}
		})
	}
}

func TestCaptureStorageChargesFinishedLegacyActualBytes(t *testing.T) {
	for _, status := range []string{"complete", "incomplete"} {
		t.Run(status, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "session-legacy")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			data := []byte(strings.Repeat("x", 5000) + "\n" + fmt.Sprintf("{\"version\":1,\"kind\":\"capture-end\",\"status\":%q}\n", status))
			if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := openCaptureFileWithBudget(root, 10, int64(len(data))+9, 64); err == nil {
				t.Fatal("legacy bytes not charged")
			}
			_, f, err := openCaptureFileWithBudget(root, 10, int64(len(data))+10, 64)
			if err != nil {
				t.Fatal(err)
			}
			f.Close()
			got, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
			if err != nil || string(got) != string(data) {
				t.Fatalf("legacy changed: %v", err)
			}
		})
	}
}

func TestCaptureStorageOpenKeepsCompletedReservation(t *testing.T) {
	root := t.TempDir()
	r, err := Open(root, Config{MaxBytes: MaxSessionBytes})
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(r.Dir(), "events.jsonl")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	extra, err := Open(root, Config{MaxBytes: 1 << 20})
	if err == nil {
		_ = extra.Close()
		t.Fatal("Open ignored completed session reservation")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("admission changed prior evidence")
	}
}
