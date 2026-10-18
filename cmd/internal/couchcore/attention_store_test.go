package couchcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttentionStoreRoundTripAndRefusals(t *testing.T) {
	store := AttentionStore{Dir: filepath.Join(t.TempDir(), "attention")}
	if items, err := store.Load(); err != nil || items != nil {
		t.Fatalf("no file: %v %v", items, err)
	}
	l := AttentionList{}.Notify(notifyItem("n1", attentionT0)).
		Conditions(AttentionQuestion, map[ThreadAddress]AttentionCondition{attentionW1: {Label: "pair:1", Title: "which one?"}}, attentionT0)
	if err := store.Save(l.Saved(attentionT0)); err != nil {
		t.Fatal(err)
	}
	items, err := store.Load()
	if err != nil || len(items) != 2 || items[0].Since.IsZero() || items[1].Title == "" && items[0].Title == "" {
		t.Fatalf("round trip: %+v %v", items, err)
	}
	path := filepath.Join(store.Dir, "list.json")
	for name, raw := range map[string]string{
		"version":  `{"version":2,"items":[]}`,
		"unknown":  `{"version":1,"items":[],"extra":1}`,
		"oversize": `{"version":1,"items":[],"pad":"` + strings.Repeat("x", AttentionStoreLimit) + `"}`,
	} {
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Load(); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(store.Dir, "elsewhere"), path); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil {
		t.Error("a symlinked list was followed")
	}
	if err := store.Save(nil); err == nil {
		t.Error("saved over a symlink")
	}
}
