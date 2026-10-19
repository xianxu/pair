package diagnosticlog

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestRecentGenerationsRotateBeyondFormerCountAndInventoryBounds(t *testing.T) {
	path, now, opts := fixture(t)
	opts.MaxBytes = 1
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	var earliest string
	var firstPayload, firstMetadata []byte
	const total = 70
	for i := 0; i < total; i++ {
		*now = now.Add(time.Second)
		if n, err := w.Write([]byte{byte(i)}); n != 1 || err != nil {
			t.Fatal(i, n, err)
		}
		if i == 1 {
			names, _, _, err := generationPage(path, "", 100, false, opts)
			if err != nil || len(names) != 1 {
				t.Fatal(names, err)
			}
			earliest = segmentPath(path, names[0])
			firstPayload, err = os.ReadFile(earliest)
			if err != nil {
				t.Fatal(err)
			}
			firstMetadata, err = os.ReadFile(earliest + ".json")
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	names, _, complete, err := generationPage(path, "", 100, false, opts)
	if err != nil || !complete || len(names) != total-1 {
		t.Fatalf("backup fence lost generations: got=%d complete=%t err=%v", len(names), complete, err)
	}
	seen := map[byte]bool{}
	for _, name := range names {
		body, err := os.ReadFile(segmentPath(path, name))
		if err != nil || len(body) != 1 {
			t.Fatal(name, body, err)
		}
		var metadata generation
		p := segmentPath(path, name)
		if err := readJSON(p+".json", &metadata); err != nil {
			t.Fatal(name, err)
		}
		if metadata.Name != name || metadata.Size != 1 {
			t.Fatal("invalid retained metadata", metadata)
		}
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := matches(st, metadata); err != nil {
			t.Fatal("retained identity changed", err)
		}
		seen[body[0]] = true
	}
	for i := 0; i < total-1; i++ {
		if !seen[byte(i)] {
			t.Fatal("generation evicted", i)
		}
	}
	payload, err := os.ReadFile(earliest)
	if err != nil || !bytes.Equal(payload, firstPayload) {
		t.Fatal("earliest payload changed", err)
	}
	metadata, err := os.ReadFile(earliest + ".json")
	if err != nil || !bytes.Equal(metadata, firstMetadata) {
		t.Fatal("earliest metadata changed", err)
	}

}
func TestExpiredManagedAndLegacyDiagnosticsNeedNoBackup(t *testing.T) {
	path, now, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("old"))
	*now = now.Add(25 * time.Hour)
	w.Write([]byte("new"))
	w.Close()
	*now = now.Add(RetentionPeriod + time.Hour)
	if rows, err := Collect(path, opts, 100); err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expired current remains", err)
	}
	legacy := path + ".legacy"
	if err := os.WriteFile(legacy, []byte("legacy"), 0600); err != nil {
		t.Fatal(err)
	}
	stamp := now.Add(-RetentionPeriod - time.Hour)
	os.Chtimes(legacy, stamp, stamp)
	if rows, err := CollectLegacy(legacy, opts); err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatal("expired legacy remains", err)
	}
}
func TestPreviouslyPersistedDeletionIntentFinishesWithoutBackup(t *testing.T) {
	path, p, opts := deletionIntentFixture(t)
	if _, err := Collect(path, opts, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("pending deletion remains", err)
	}
	state, err := load(path)
	if err != nil || state.Deleting != nil {
		t.Fatal("intent not completed", err)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "new" {
		t.Fatal(string(b), err)
	}
}
