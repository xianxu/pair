package couchcore

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func repoLegacy(t *testing.T, id, path, value string) LegacyRepoPreference {
	t.Helper()
	raw, err := json.Marshal(PathLaunchPreference{SchemaVersion: 1, RepoIdentity: "/repo/.git", PhysicalPath: path, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {value}}, Revision: 1})
	if err != nil {
		t.Fatal(err)
	}
	s, err := DecodeLegacyRepoPreference(id, raw)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRepoArgumentsExplicitEmptyAndIsolation(t *testing.T) {
	p, err := RememberRepoArguments(nil, "/repo/.git", LaunchProfile{Agent: "codex", Argv: []string{"--model", "one"}})
	if err != nil {
		t.Fatal(err)
	}
	p.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{ "x" : 1 }`)}
	empty := []string{}
	r, err := ResolveRepoArguments(&p, "codex", &empty)
	if err != nil || !r.Found || r.Argv == nil || len(r.Argv) != 0 {
		t.Fatalf("explicit empty: %+v %v", r, err)
	}
	r, err = ResolveRepoArguments(&p, "claude", nil)
	if err != nil || r.Found {
		t.Fatalf("cross agent: %+v %v", r, err)
	}
	r, err = ResolveRepoArguments(&p, "codex", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Argv[0] = "changed"
	if p.Agents["codex"].Argv[0] != "--model" {
		t.Fatal("resolution aliases input")
	}
	next, err := RememberRepoArguments(&p, "/repo/.git", LaunchProfile{Agent: "claude", Argv: empty})
	if err != nil {
		t.Fatal(err)
	}
	next.Unknown["future"][0] = '['
	if string(p.Unknown["future"]) != `{ "x" : 1 }` || len(p.Agents) != 1 {
		t.Fatal("remember mutated input")
	}
	if _, err := RememberRepoArguments(&p, "/other/.git", LaunchProfile{Agent: "codex", Argv: empty}); err == nil {
		t.Fatal("accepted wrong repository")
	}
}

func TestRepoArgumentsFoldOrderConflictAndConfirmed(t *testing.T) {
	a, b := repoLegacy(t, "a", "/repo/sub", "one"), repoLegacy(t, "b", "/repo/sub", "two")
	var outcomes []RepoLaunchPreference
	for _, order := range [][]LegacyRepoPreference{{a, b}, {b, a}} {
		p, err := FoldPathPreferences(nil, "/repo/.git", order[:1])
		if err != nil {
			t.Fatal(err)
		}
		p, err = FoldPathPreferences(&p, "/repo/.git", order[1:])
		if err != nil {
			t.Fatal(err)
		}
		r, err := ResolveRepoArguments(&p, "codex", nil)
		if err != nil || !r.Conflict || r.Found {
			t.Fatalf("conflict: %+v %v", r, err)
		}
		outcomes = append(outcomes, p)
	}
	if !reflect.DeepEqual(outcomes[0], outcomes[1]) {
		t.Fatalf("discovery order changed result: %+v %+v", outcomes[0], outcomes[1])
	}
	p, err := RememberRepoArguments(&outcomes[0], "/repo/.git", LaunchProfile{Agent: "codex", Argv: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	changed := repoLegacy(t, "a", "/repo/sub", "new")
	next, err := FoldPathPreferences(&p, "/repo/.git", []LegacyRepoPreference{changed})
	if err != nil {
		t.Fatal(err)
	}
	if !next.Agents["codex"].Confirmed || next.Agents["codex"].Conflict || len(next.Agents["codex"].Argv) != 0 || next.LegacySources["a"].Digest == a.Digest {
		t.Fatalf("confirmed configuration lost or source ignored: %+v", next)
	}
	if p.LegacySources["a"].Digest != a.Digest {
		t.Fatal("fold mutated original source")
	}
	if _, err := FoldPathPreferences(nil, "/repo/.git", []LegacyRepoPreference{a, changed}); err == nil {
		t.Fatal("accepted ambiguous same-source batch")
	}
}

func TestRepoArgumentsChangedSourceReevaluatesProvisional(t *testing.T) {
	a, b := repoLegacy(t, "a", "/repo/a", "one"), repoLegacy(t, "b", "/repo/b", "one")
	p, err := FoldPathPreferences(nil, "/repo/.git", []LegacyRepoPreference{a, b})
	if err != nil {
		t.Fatal(err)
	}
	rev := p.Revision
	p, err = FoldPathPreferences(&p, "/repo/.git", []LegacyRepoPreference{a, b})
	if err != nil || p.Revision != rev {
		t.Fatalf("unchanged fold: %+v %v", p, err)
	}
	b = repoLegacy(t, "b", "/repo/b", "two")
	p, err = FoldPathPreferences(&p, "/repo/.git", []LegacyRepoPreference{b})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Agents["codex"].Conflict {
		t.Fatal("changed source not reevaluated")
	}
	b = repoLegacy(t, "b", "/repo/b", "one")
	p, err = FoldPathPreferences(&p, "/repo/.git", []LegacyRepoPreference{b})
	if err != nil {
		t.Fatal(err)
	}
	if p.Agents["codex"].Conflict || p.Agents["codex"].Confirmed || !reflect.DeepEqual(p.Agents["codex"].Argv, []string{"one"}) {
		t.Fatalf("converged sources: %+v", p)
	}
}

func TestRepoArgumentsBorrowRequiresConfirmedAndPreservesEmpty(t *testing.T) {
	records := map[string]RepoLaunchPreference{}
	for _, key := range []string{"z", "b", "target"} {
		p, err := RememberRepoArguments(nil, key, LaunchProfile{Agent: "codex", Argv: []string{}})
		if err != nil {
			t.Fatal(err)
		}
		records[key] = p
	}
	provisional, err := FoldPathPreferences(nil, "/repo/.git", []LegacyRepoPreference{repoLegacy(t, "a", "/repo/a", "one")})
	if err != nil {
		t.Fatal(err)
	}
	records["/repo/.git"] = provisional
	got, ok, err := ChooseBorrowedArguments(records, "target", "codex")
	if err != nil || !ok || got.RepoIdentity != "b" || got.Argv == nil || len(got.Argv) != 0 {
		t.Fatalf("borrow: %+v %v %v", got, ok, err)
	}
	if _, ok, err := ChooseBorrowedArguments(records, "target", "claude"); err != nil || ok {
		t.Fatalf("borrowed other harness: %v %v", ok, err)
	}
}

func TestRepoArgumentsBoundsAndValidation(t *testing.T) {
	for _, profile := range []LaunchProfile{{Agent: "bogus", Argv: []string{}}, {Agent: "codex"}, {Agent: "codex", Argv: []string{strings.Repeat("x", MaxRepoArgumentBytes+1)}}} {
		if _, err := RememberRepoArguments(nil, "repo", profile); err == nil {
			t.Fatalf("accepted %+v", profile)
		}
	}
	p, err := RememberRepoArguments(nil, "repo", LaunchProfile{Agent: "codex", Argv: []string{strings.Repeat("x", MaxRepoArgumentBytes)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Agents["codex"].Argv[0]) != MaxRepoArgumentBytes {
		t.Fatal("lost exact bound")
	}
	sources := make([]LegacyRepoPreference, MaxRepoPreferenceSources+1)
	for i := range sources {
		sources[i] = repoLegacy(t, strings.Repeat("a", i+1), "/repo/a", "one")
	}
	full, err := FoldPathPreferences(nil, "/repo/.git", sources[:MaxRepoPreferenceSources])
	if err != nil || len(full.LegacySources) != MaxRepoPreferenceSources {
		t.Fatalf("exact source bound: %d %v", len(full.LegacySources), err)
	}
	if _, err := FoldPathPreferences(&full, "/repo/.git", sources[MaxRepoPreferenceSources:]); err == nil {
		t.Fatal("accepted cumulative source overflow")
	}
	if _, err := FoldPathPreferences(nil, "/repo/.git", sources); err == nil {
		t.Fatal("accepted too many sources")
	}
}

func TestRepoArgumentsGeneratedFoldOrdersPreserveProvenance(t *testing.T) {
	sources := []LegacyRepoPreference{repoLegacy(t, "a", "/repo/a", "one"), repoLegacy(t, "b", "/repo/b", "two"), repoLegacy(t, "c", "/repo/c", "one")}
	sources[0].Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{ "value": 1 }`)}
	var expected RepoLaunchPreference
	orders := [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	for n, order := range orders {
		var current *RepoLaunchPreference
		for _, i := range order {
			next, err := FoldPathPreferences(current, "/repo/.git", []LegacyRepoPreference{sources[i]})
			if err != nil {
				t.Fatal(err)
			}
			current = &next
		}
		if n == 0 {
			expected = *current
		} else if !reflect.DeepEqual(expected, *current) {
			t.Fatalf("order %v changed fold", order)
		}
	}
	replacement := repoLegacy(t, "a", "/repo/a", "three")
	next, err := FoldPathPreferences(&expected, "/repo/.git", []LegacyRepoPreference{replacement})
	if err != nil {
		t.Fatal(err)
	}
	if string(next.LegacySources["a"].Unknown["future"]) != `{ "value": 1 }` {
		t.Fatal("late source replacement erased provenance")
	}
	next.LegacySources["a"].Raw[0] = '['
	if expected.LegacySources["a"].Raw[0] != '{' {
		t.Fatal("fold aliases source bytes")
	}
}

func TestRepoArgumentsBorrowingCountsOneVisitPerRepository(t *testing.T) {
	for _, n := range []int{8, 16} {
		records := map[string]RepoLaunchPreference{}
		for i := 0; i < n; i++ {
			key := fmt.Sprintf("repo-%02d", i)
			p, err := RememberRepoArguments(nil, key, LaunchProfile{Agent: "codex", Argv: []string{}})
			if err != nil {
				t.Fatal(err)
			}
			records[key] = p
		}
		visits := map[string]int{}
		got, found, err := chooseBorrowedArguments(records, "target", "codex", func(key string) { visits[key]++ })
		if err != nil || !found || got.RepoIdentity != "repo-00" {
			t.Fatal(got, found, err)
		}
		if len(visits) != n {
			t.Fatalf("N=%d visited %d records", n, len(visits))
		}
		for key, count := range visits {
			if count != 1 {
				t.Fatalf("N=%d scanned %s %d times", n, key, count)
			}
		}
	}
}
