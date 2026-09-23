package couchcore

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func identityJSON(t *testing.T, slot bool) map[string]any {
	t.Helper()
	m := map[string]any{"schema_version": 2, "repo": "my-repo", "repo_identity": "/fleet/my-repo/.git", "primary_root": "/fleet/my-repo", "fleet_root": "/fleet", "environment_root": "/fleet", "worktree_root": "/fleet/my-repo", "kind": "primary", "address": "my-repo:0", "slot": 0, "branch": "main", "head": strings.Repeat("a", 40), "resting_branch": "main"}
	if slot {
		m["kind"] = "slot"
		m["address"] = "my-repo:2"
		m["slot"] = 2
		m["branch"] = "feature/test"
		m["resting_branch"] = "main-slot2"
		m["environment_root"] = "/fleet/worktree/my-repo-slot2"
		m["worktree_root"] = "/fleet/worktree/my-repo-slot2/my-repo"
		m["environment_host"] = map[string]any{"repo": "my-repo", "slot": 2, "repo_identity": "/fleet/my-repo/.git", "primary_root": "/fleet/my-repo", "worktree_root": m["worktree_root"]}
	}
	return m
}
func parseIdentityMap(t *testing.T, m map[string]any) error {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ParseWorkspaceIdentity(b)
	return err
}
func TestParseWorkspaceIdentity(t *testing.T) {
	for _, slot := range []bool{false, true} {
		if err := parseIdentityMap(t, identityJSON(t, slot)); err != nil {
			t.Fatal(err)
		}
	}
	m := identityJSON(t, false)
	m["head"] = nil
	if err := parseIdentityMap(t, m); err != nil {
		t.Fatal(err)
	}
	m = identityJSON(t, true)
	m["branch"] = nil
	if err := parseIdentityMap(t, m); err != nil {
		t.Fatal(err)
	}
	m = identityJSON(t, false)
	m["kind"] = "worktree"
	m["worktree_root"] = "/other/worktree"
	m["address"] = nil
	m["slot"] = nil
	m["resting_branch"] = nil
	if err := parseIdentityMap(t, m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "repo", "repo_identity", "primary_root", "fleet_root", "environment_root", "worktree_root", "kind", "address", "slot", "branch", "head", "resting_branch"} {
		m := identityJSON(t, false)
		delete(m, key)
		if err := parseIdentityMap(t, m); err == nil {
			t.Errorf("accepted missing %s", key)
		}
	}
	for _, change := range []struct {
		k string
		v any
	}{{"schema_version", 1}, {"kind", "bogus"}, {"repo", "other"}, {"repo_identity", "relative"}, {"primary_root", "/else/my-repo"}, {"slot", 1}, {"address", "my-repo:1"}, {"resting_branch", "other"}, {"head", strings.Repeat("0", 40)}, {"head", strings.Repeat("A", 40)}, {"branch", ""}} {
		m := identityJSON(t, false)
		m[change.k] = change.v
		if err := parseIdentityMap(t, m); err == nil {
			t.Errorf("accepted %s=%v", change.k, change.v)
		}
	}
	for _, change := range []struct {
		k string
		v any
	}{{"slot", 0}, {"head", nil}, {"environment_host", nil}, {"environment_root", "/other"}, {"branch", "main"}, {"branch", "main-slot3"}} {
		m := identityJSON(t, true)
		m[change.k] = change.v
		if err := parseIdentityMap(t, m); err == nil {
			t.Errorf("accepted slot %s=%v", change.k, change.v)
		}
	}
	b, _ := json.Marshal(identityJSON(t, false))
	for _, raw := range [][]byte{nil, []byte("null"), b[:len(b)-1], append(append([]byte{}, b...), []byte(" {}")...), []byte(strings.Replace(string(b), `"schema_version":2`, `"schema_version":2,"schema_version":2`, 1)), []byte(strings.Repeat(" ", 1<<20) + string(b))} {
		if _, err := ParseWorkspaceIdentity(raw); err == nil {
			t.Error("accepted malformed JSON")
		}
	}
}
func TestValidWorkspaceOID(t *testing.T) {
	for _, n := range []int{40, 64} {
		if !validWorkspaceOID(strings.Repeat("a", n)) {
			t.Fatal(n)
		}
	}
	for _, s := range []string{"", strings.Repeat("a", 39), strings.Repeat("0", 40), strings.Repeat("G", 64)} {
		if validWorkspaceOID(s) {
			t.Errorf("accepted %q", s)
		}
	}
}

func TestParseWorkspaceIdentityDependencyAndFeatureWorktree(t *testing.T) {
	dependency := func() map[string]any {
		m := identityJSON(t, true)
		m["kind"] = "dependency"
		m["repo"] = "base"
		m["primary_root"] = "/fleet/worktree/my-repo-slot2/base"
		m["worktree_root"] = m["primary_root"]
		m["repo_identity"] = "/fleet/worktree/my-repo-slot2/base/.git"
		m["address"] = nil
		m["slot"] = nil
		m["resting_branch"] = nil
		return m
	}
	m := dependency()
	if err := parseIdentityMap(t, m); err != nil {
		t.Fatal(err)
	}
	m = dependency()
	m["kind"] = "worktree"
	m["worktree_root"] = "/elsewhere/base-feature"
	if err := parseIdentityMap(t, m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"repo", "slot", "repo_identity", "primary_root", "worktree_root"} {
		m := dependency()
		delete(m["environment_host"].(map[string]any), key)
		if err := parseIdentityMap(t, m); err == nil {
			t.Errorf("accepted missing host %s", key)
		}
	}
	for _, change := range []struct {
		k string
		v any
	}{{"repo_identity", "/fleet/my-repo/.git"}, {"primary_root", "/elsewhere/base"}, {"environment_host", nil}, {"slot", 1}} {
		m := dependency()
		m[change.k] = change.v
		if err := parseIdentityMap(t, m); err == nil {
			t.Errorf("accepted dependency %s=%v", change.k, change.v)
		}
	}
	m = dependency()
	m["kind"] = "worktree"
	m["worktree_root"] = "/elsewhere/base-feature"
	m["primary_root"] = "/unrelated/base"
	if err := parseIdentityMap(t, m); err == nil {
		t.Fatal("accepted dependency feature worktree with unrelated primary")
	}
}

// FuzzWorkspaceIdentity exercises the closed JSON transport, including nullable
// fields. Once accepted, encoding the typed value must preserve every fact and
// remain an accepted observation.
func FuzzWorkspaceIdentity(f *testing.F) {
	primary := `{"schema_version":2,"repo":"repo","repo_identity":"/fleet/repo/.git","primary_root":"/fleet/repo","fleet_root":"/fleet","environment_root":"/fleet","worktree_root":"/fleet/repo","kind":"primary","address":"repo:0","slot":0,"branch":"main","head":null,"resting_branch":"main"}`
	slot := `{"schema_version":2,"repo":"repo","repo_identity":"/fleet/repo/.git","primary_root":"/fleet/repo","fleet_root":"/fleet","environment_root":"/fleet/worktree/repo-slot1","environment_host":{"repo":"repo","slot":1,"repo_identity":"/fleet/repo/.git","primary_root":"/fleet/repo","worktree_root":"/fleet/worktree/repo-slot1/repo"},"worktree_root":"/fleet/worktree/repo-slot1/repo","kind":"slot","address":"repo:1","slot":1,"branch":null,"head":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","resting_branch":"main-slot1"}`
	for _, seed := range []string{primary, slot, strings.Replace(primary, `"branch":"main"`, `"branch":null`, 1), `null`, `{}`, primary + ` {}`, strings.Replace(primary, `"slot":0`, `"slot":0,"slot":1`, 1)} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		got, err := ParseWorkspaceIdentity(raw)
		if err != nil {
			return
		}
		encoded, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		again, err := ParseWorkspaceIdentity(encoded)
		if err != nil {
			t.Fatalf("accepted identity cannot round-trip: %v; encoded=%s", err, encoded)
		}
		if !reflect.DeepEqual(got, again) {
			t.Fatalf("round-trip changed identity: %#v -> %#v", got, again)
		}
		if got.SchemaVersion != 2 || !filepath.IsAbs(got.WorktreeRoot) || !filepath.IsAbs(got.RepoIdentity) {
			t.Fatalf("accepted invalid identity facts: %#v", got)
		}
		if got.Head != nil {
			if n := len(*got.Head); n != 40 && n != 64 {
				t.Fatalf("accepted partial OID %q", *got.Head)
			}
		}
	})
}
