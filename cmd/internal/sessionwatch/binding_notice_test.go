package sessionwatch

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestBindingNoticeStrictCurrentLaunch(t *testing.T) {
	for _, raw := range []string{`{"ordinal":7,"native_id":"id"}`, `{"ordinal":6,"native_id":"id"}`, `{"ordinal":7,"native_id":""}`, `{"ordinal":7,"native_id":"id","other":1}`, `{"ordinal":7,"native_id":"id"} {}`, `{"ordinal":7,`} {
		id, err := DecodeBindingNotice([]byte(raw), 7)
		if raw == `{"ordinal":7,"native_id":"id"}` {
			if err != nil || id != "id" {
				t.Fatalf("valid notice: %q %v", id, err)
			}
		} else if err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	var b bytes.Buffer
	if err := NotifyBinding(&b, 7, "id"); err != nil {
		t.Fatal(err)
	}
	if id, err := DecodeBindingNotice(bytes.TrimSpace(b.Bytes()), 7); err != nil || id != "id" {
		t.Fatalf("roundtrip %q %v", id, err)
	}
}

func TestObserveNotifiesOnlyCommittedBinding(t *testing.T) {
	inventory := sessioninventory.Inventory{Forests: []sessioninventory.Forest{{Agent: sessioninventory.AgentCodex, Roots: []sessioninventory.Node{{StableID: "root", NativeID: "id", Role: sessioninventory.RoleRoot}}}}}
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	round := sessioninventory.RoundObservation{ScopeKey: "scope", Tag: "work", Agent: sessioninventory.AgentCodex, RootNodeID: "root", PairPositions: []uint64{10}, NativePositions: []uint64{2}}
	for _, fail := range []bool{false, true} {
		store := &fakeLifecycleStore{}
		if fail {
			store.bindingErr = errors.New("uncommitted")
		}
		called := false
		_, err := ObserveAndPersist(ObserveInput{Owner: owner, LaunchOrdinal: 7, Inventory: inventory, LiveRounds: []sessioninventory.RoundObservation{round}, NotifyBinding: func(ordinal uint64, id string) error {
			called = true
			if len(store.records) != 1 || ordinal != 7 || id != "id" {
				t.Fatalf("notification before durable binding: %+v", store.records)
			}
			return nil
		}}, store, nil)
		if fail {
			if err == nil || called {
				t.Fatalf("failed commit notified: %v %v", called, err)
			}
		} else if err != nil || !called {
			t.Fatalf("missing notification: %v %v", called, err)
		}
	}
}

func TestBindingNotificationFailurePreservesConfig(t *testing.T) {
	inventory := sessioninventory.Inventory{Forests: []sessioninventory.Forest{{Agent: sessioninventory.AgentCodex, Roots: []sessioninventory.Node{{StableID: "root", NativeID: "id", Role: sessioninventory.RoleRoot}}}}}
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	round := sessioninventory.RoundObservation{ScopeKey: "scope", Tag: "work", Agent: sessioninventory.AgentCodex, RootNodeID: "root", PairPositions: []uint64{10}, NativePositions: []uint64{2}}
	cached := false
	resolved, err := ObserveAndPersist(ObserveInput{Owner: owner, LaunchOrdinal: 7, Inventory: inventory, LiveRounds: []sessioninventory.RoundObservation{round}, NotifyBinding: func(uint64, string) error { return errors.New("broken pipe") }}, &fakeLifecycleStore{}, func(ConfigPayload) error { cached = true; return nil })
	if err != nil || !cached || len(resolved.Diagnostics) == 0 {
		t.Fatalf("best effort notification lost persistence: err=%v cached=%v diagnostics=%v", err, cached, resolved.Diagnostics)
	}
}

// pair#614: a binding notice accepts exactly the conversation IDs a lock key
// accepts, so the wrapper never receives an ID it cannot lock.
func TestBindingNoticeIDRuleMatchesLockKey(t *testing.T) {
	for _, id := range []string{"019d1111-1111-7111-8111-111111111111", "has space", "zero​width", "a/b", "..", "\x01", strings.Repeat("x", 513), strings.Repeat("x", 512)} {
		var buf bytes.Buffer
		notifyErr := NotifyBinding(&buf, 7, id)
		_, keyErr := artifactpath.NativeLockFilename("codex", id)
		if (notifyErr == nil) != (keyErr == nil) {
			t.Errorf("%q: notice err=%v, lock key err=%v", id, notifyErr, keyErr)
		}
		if notifyErr == nil {
			if got, err := DecodeBindingNotice(bytes.TrimSpace(buf.Bytes()), 7); err != nil || got != id {
				t.Errorf("%q: round trip %q %v", id, got, err)
			}
		}
	}
}
