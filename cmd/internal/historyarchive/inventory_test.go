package historyarchive

import (
	"encoding/json"
	"testing"
	"time"
)

func TestInventoryDefaultsAndRouting(t *testing.T) {
	i, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if len(i.Sources) < 15 {
		t.Fatal("missing sources")
	}
	for _, s := range i.Sources {
		if _, err := i.Resolve(s.Kind); err != nil {
			t.Fatal(err)
		}
		for _, age := range []time.Duration{0, time.Hour, 365 * 24 * time.Hour} {
			if sink, err := i.Route(s.Kind, age); err != nil || sink != "local" {
				t.Fatalf("route %q %v", sink, err)
			}
		}
	}
	if _, err := i.Resolve("unknown"); err == nil {
		t.Fatal("unknown accepted")
	}
	if _, err := i.Route("ops-documents", -1); err == nil {
		t.Fatal("negative age accepted")
	}
	i.Sources = append(i.Sources, i.Sources[0])
	b, _ := json.Marshal(i)
	if _, err := DecodeInventory(b); err == nil {
		t.Fatal("duplicate source accepted")
	}
}
func TestInventoryRejectAmbiguousRules(t *testing.T) {
	i, _ := Defaults()
	i.Sinks = append(i.Sinks, i.Sinks[0])
	b, _ := json.Marshal(i)
	if _, err := DecodeInventory(b); err == nil {
		t.Fatal("ambiguous sinks accepted")
	}
}
func TestInventoryAgeBoundariesAndRoots(t *testing.T) {
	i, _ := Defaults()
	i.Sinks = []SinkRule{{Kind: "*", MinAgeSeconds: 0, MaxAgeSeconds: 60, Sink: "warm"}, {Kind: "*", MinAgeSeconds: 60, Sink: "cold"}}
	b, _ := json.Marshal(i)
	i, err := DecodeInventory(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		age  time.Duration
		want string
	}{{0, "warm"}, {60*time.Second - time.Nanosecond, "warm"}, {60 * time.Second, "cold"}} {
		got, err := i.Route("ops-documents", tc.age)
		if err != nil || got != tc.want {
			t.Fatalf("%v: %q %v", tc.age, got, err)
		}
	}
	for _, tc := range []struct {
		kind, override, home string
		env                  map[string]string
		want                 string
	}{{"codex-transcripts", "", "/home/me", nil, "/home/me/.codex"}, {"codex-transcripts", "", "", map[string]string{"CODEX_HOME": "/codex"}, "/codex"}, {"pair-scrollback", "", "/home/me", nil, "/home/me/.local/share/pair"}, {"pair-scrollback", "", "", map[string]string{"XDG_DATA_HOME": "/data"}, "/data/pair"}, {"ops-documents", "/ops", "", nil, "/ops"}} {
		got, err := i.ResolveRoot(tc.kind, tc.override, tc.home, tc.env)
		if err != nil || got != tc.want {
			t.Fatalf("root %q %v", got, err)
		}
	}
	if _, err := i.ResolveRoot("ops-documents", "", "/home/me", nil); err == nil {
		t.Fatal("missing override accepted")
	}
}

func TestInventoryCouchBackendRoots(t *testing.T) {
	inv, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"couch-threads", "webcast-recordings"} {
		for _, tc := range []struct {
			env  map[string]string
			want string
		}{{nil, "/home/me/.local/share/pair/couch"}, {map[string]string{"COUCH_STORE_DIR": "/custom/store"}, "/custom/store"}, {map[string]string{"XDG_DATA_HOME": "/data"}, "/data/pair/couch"}} {
			root, err := inv.ResolveRoot(kind, "", "/home/me", tc.env)
			if err != nil || root != tc.want {
				t.Errorf("%s root %q err %v want %q", kind, root, err, tc.want)
			}
		}
	}
	if _, err := inv.ResolveRoot("couch-slot-threads", "", "/home/me", map[string]string{"COUCH_STORE_DIR": "/global"}); err == nil {
		t.Fatal("slot root guessed from global store")
	}
	root, err := inv.ResolveRoot("couch-slot-threads", "/fleet/worktree/repo-slot1/.couch", "", nil)
	if err != nil || root != "/fleet/worktree/repo-slot1/.couch" {
		t.Fatalf("slot root %q %v", root, err)
	}
}
