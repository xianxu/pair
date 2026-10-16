package couchcmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func resourcePersistenceDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
func TestResourceHistoryPersistenceFailureKeepsMemoryAndRecovers(t *testing.T) {
	dir := resourcePersistenceDir(t)
	p := newResourceHistoryPersistence(dir)
	now := time.Now()
	if err := os.MkdirAll(p.store.Dir, 0700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(p.store.Dir, ".history-stage")
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	cost, err := p.persist(context.Background(), couchcore.ResourceView{SampledAt: now})
	if err == nil || cost.StorageWrites != 0 || len(p.history.Frames) != 1 {
		t.Fatal("write failure stopped memory ring", cost, err, len(p.history.Frames))
	}
	if err := os.Remove(stage); err != nil {
		t.Fatal(err)
	}
	cost, err = p.persist(context.Background(), couchcore.ResourceView{SampledAt: now.Add(time.Minute)})
	if err != nil || cost.StorageWrites != 1 || len(p.history.Frames) != 2 {
		t.Fatal("write recovery lost memory frames", cost, err)
	}
	restored := newResourceHistoryPersistence(dir)
	if len(restored.history.Frames) != 2 {
		t.Fatal("history was not restored")
	}
	entries, err := os.ReadDir(p.store.Dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "history.json" {
		t.Fatal("non-history persistence artifact", entries, err)
	}
}
func TestResourceHistoryPersistenceCorruptReadStartsFresh(t *testing.T) {
	dir := resourcePersistenceDir(t)
	path := filepath.Join(dir, "resources")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "history.json"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	p := newResourceHistoryPersistence(dir)
	if p.loadError == nil || len(p.history.Frames) != 0 {
		t.Fatal("corrupt history not isolated")
	}
	cost, err := p.persist(context.Background(), couchcore.ResourceView{SampledAt: time.Now()})
	if cost.StorageWrites != 1 || err == nil || !strings.Contains(err.Error(), "continuity") {
		t.Fatal("load failure permanently disabled writes or hid continuity loss", cost, err)
	}
	raw, err := os.ReadFile(filepath.Join(path, "history.json"))
	if err != nil || couchcore.ValidateResourceHistory(raw) != nil || !strings.Contains(string(raw), "continuity") {
		t.Fatal("fresh history missing diagnostic", err)
	}
}
func TestResourceHistoryPersistenceUnsafeReadIsBestEffort(t *testing.T) {
	dir := resourcePersistenceDir(t)
	path := filepath.Join(dir, "resources")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dir, "outside")
	original := []byte(`{"version":1,"frames":[]}`)
	if err := os.WriteFile(outside, original, 0600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(path, "history.json")
	if err := os.Symlink(outside, file); err != nil {
		t.Fatal(err)
	}
	p := newResourceHistoryPersistence(dir)
	now := time.Now()
	cost, err := p.persist(context.Background(), couchcore.ResourceView{SampledAt: now})
	if err == nil || cost.StorageWrites != 0 || len(p.history.Frames) != 1 {
		t.Fatal("unsafe read blocked memory or allowed write", cost, err)
	}
	got, err := os.ReadFile(outside)
	if err != nil || string(got) != string(original) {
		t.Fatal("outside history changed")
	}
	if err = os.Remove(file); err != nil {
		t.Fatal(err)
	}
	cost, err = p.persist(context.Background(), couchcore.ResourceView{SampledAt: now.Add(time.Minute)})
	if cost.StorageWrites != 1 || len(p.history.Frames) != 2 || err == nil || !strings.Contains(err.Error(), "continuity") {
		t.Fatal("safe publication did not resume", cost, err)
	}
	var disk couchcore.ResourceHistory
	raw, readErr := os.ReadFile(file)
	if readErr != nil || json.Unmarshal(raw, &disk) != nil || len(disk.Frames) != 2 {
		t.Fatal("memory ring not published after repair")
	}
}
func TestResourceHistoryPersistencePrunesOnStartupAndLoadsOnce(t *testing.T) {
	dir := resourcePersistenceDir(t)
	p := newResourceHistoryPersistence(dir)
	now := time.Now()
	history := couchcore.ResourceHistory{Version: 1, Frames: []couchcore.ResourceFrame{{At: now.Add(-25 * time.Hour)}, {At: now.Add(-time.Hour)}}}
	raw, _ := json.Marshal(history)
	if err := p.store.Save(raw); err != nil {
		t.Fatal(err)
	}
	p = newResourceHistoryPersistence(dir)
	if len(p.history.Frames) != 1 {
		t.Fatal("expired startup history retained")
	}
	if err := os.WriteFile(filepath.Join(p.store.Dir, "history.json"), []byte("externally corrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.persist(context.Background(), couchcore.ResourceView{SampledAt: now}); err != nil {
		t.Fatal("worker reread aggregate history", err)
	}
	if len(p.history.Frames) != 2 {
		t.Fatal("loaded ring discarded")
	}
}
