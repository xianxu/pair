package couchtty

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func presentationOrdinary(root, tag string) couchcore.ActionableThreadSummary {
	scope, _ := launcher.ResolveRepoScope(root)
	return couchcore.ActionableThreadSummary{Address: couchcore.ThreadAddress{RepoScope: scope.Key, Tag: couchcore.ThreadTag(tag)}, StartingPath: root, WorkingPath: root}
}
func presentationSlot(root string, n int) couchcore.ActionableThreadSummary {
	env := filepath.Join(filepath.Dir(root), "worktree", fmt.Sprintf("%s-slot%d", filepath.Base(root), n))
	slot := couchcore.SlotIdentity{Repo: filepath.Base(root), RepoIdentity: filepath.Join(root, ".git"), PrimaryRoot: root, EnvironmentRoot: env, WorktreeRoot: filepath.Join(env, filepath.Base(root)), Number: n}
	target := couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: slot}
	key, err := target.RowKey()
	if err != nil {
		panic(err)
	}
	return couchcore.ActionableThreadSummary{Target: target, RowKey: key}
}
func TestPresentThreadsGroupingAndPermutation(t *testing.T) {
	primary := presentationOrdinary("/src/pair", "one")
	primary.StartingPath = "/src/pair/cmd/internal"
	primary.WorkingPath = "/elsewhere"
	primary.Name = "renamed"
	rows := []couchcore.ActionableThreadSummary{presentationSlot("/src/pair", 10), presentationOrdinary("/src/zebra", "z"), presentationSlot("/src/pair", 2), primary}
	saved := append([]couchcore.ActionableThreadSummary(nil), rows...)
	got := PresentThreads(rows)
	if !reflect.DeepEqual(saved, rows) {
		t.Fatal("mutated input")
	}
	if len(got) != 4 || got[0].Label != "pair" || got[0].Path != "/src/pair" || got[1].Label != "pair:2" || got[2].Label != "pair:10" || got[1].Indent != 2 || got[1].Path != rows[2].Target.Slot.WorktreeRoot || got[1].SlotNumber != 2 {
		t.Fatalf("projection: %+v", got)
	}
	for i := 0; i < 30; i++ {
		rand.New(rand.NewSource(int64(i))).Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
		if !reflect.DeepEqual(got, PresentThreads(rows)) {
			t.Fatal("input order affected projection")
		}
	}
	rows[0].Name = "arbitrary rename"
	if p := PresentThreads(rows); p[0].GroupKey != got[0].GroupKey {
		t.Fatal("rename affected order")
	}
}
func TestPresentThreadsRootRecoveryAndFallback(t *testing.T) {
	known := presentationOrdinary("/src/repo", "known")
	known.StartingPath = "/src/repo/sub/dir"
	known.WorkingPath = "/moved"
	known.Name = "custom"
	unknown := presentationOrdinary("/unknown", "unknown")
	unknown.StartingPath = "relative"
	unknown.WorkingPath = "/fallback"
	missing := presentationOrdinary("/missing", "missing")
	missing.StartingPath = ""
	missing.WorkingPath = ""
	invalid := presentationOrdinary("/invalid", "invalid")
	invalid.Target = presentationSlot("/invalid", 2).Target
	invalid.Target.Slot.Number = 0
	for _, tc := range []struct {
		row         couchcore.ActionableThreadSummary
		path, label string
	}{{known, "/src/repo", "custom"}, {unknown, "/fallback", "fallback"}, {missing, "(path unavailable)", "missing"}, {invalid, "/invalid", invalid.Label()}} {
		got := PresentThreads([]couchcore.ActionableThreadSummary{tc.row})[0]
		if got.Path != tc.path || got.Label != tc.label || got.SlotNumber != 0 {
			t.Fatalf("got %+v; want %s %s", got, tc.path, tc.label)
		}
	}
}
func TestPresentThreadsCollisions(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{presentationOrdinary("/a/repo", "native-11111111"), presentationOrdinary("/a/repo", "native-22222222"), presentationSlot("/a/repo", 2), presentationSlot("/b/repo", 2)}
	got := PresentThreads(rows)
	labels := map[string]bool{}
	for _, p := range got {
		labels[p.Label] = true
	}
	for _, want := range []string{"repo [/a/repo]·11111111", "repo [/a/repo]·22222222", "repo:2 [/a/repo]", "repo:2 [/b/repo]"} {
		if !labels[want] {
			t.Fatalf("missing %q: %+v", want, got)
		}
	}
	if got[0].GroupKey == got[len(got)-1].GroupKey {
		t.Fatal("same-name roots merged")
	}
	ordinary := []couchcore.ActionableThreadSummary{presentationOrdinary("/a/repo", "one"), presentationOrdinary("/b/repo", "two")}
	for i := range ordinary {
		ordinary[i].Name = "same"
	}
	for _, p := range PresentThreads(ordinary) {
		if !strings.Contains(p.Label, "·") {
			t.Fatalf("ordinary collision lost tag: %+v", p)
		}
	}
}
func TestPresentThreadsAddresslessSlotsAndUnknownScopes(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{presentationSlot("/a/repo", 10), presentationSlot("/a/repo", 2)}
	got := PresentThreads(rows)
	if len(got) != 2 || got[0].SlotNumber != 2 || got[1].SlotNumber != 10 {
		t.Fatalf("addressless rows lost: %+v", got)
	}
	a := presentationOrdinary("/a", "a")
	b := presentationOrdinary("/b", "b")
	a.Address.RepoScope = ""
	b.Address.RepoScope = ""
	got = PresentThreads([]couchcore.ActionableThreadSummary{a, b})
	if got[0].GroupKey == got[1].GroupKey {
		t.Fatal("empty scopes merged")
	}
}
func BenchmarkPresentThreads1000(b *testing.B) {
	rows := make([]couchcore.ActionableThreadSummary, 1000)
	for i := range rows {
		rows[i] = presentationSlot(fmt.Sprintf("/src/repo%d", i/10), i%10+1)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PresentThreads(rows)
	}
}

func TestPresentThreadsStableGroupOrderAndKnownRoot(t *testing.T) {
	known := presentationOrdinary("/src/alpha", "a")
	unknown := known
	unknown.Address.Tag = "b"
	unknown.StartingPath = "/unrelated"
	unknown.WorkingPath = "/misleading/zebra"
	other := presentationOrdinary("/src/beta", "c")
	rows := []couchcore.ActionableThreadSummary{unknown, other, known}
	before := PresentThreads(rows)
	for i := range rows {
		rows[i].Name = fmt.Sprintf("renamed-%d", i)
		rows[i].WorkingPath = fmt.Sprintf("/elsewhere/%d", i)
	}
	after := PresentThreads(rows)
	for i := range before {
		if before[i].Row.Address != after[i].Row.Address {
			t.Fatalf("mutable fields reordered rows: %+v", after)
		}
	}
	if before[0].Path != "/src/alpha" || before[1].Path != "/src/alpha" {
		t.Fatalf("known root did not propagate: %+v", before)
	}
	for i := 0; i < 20; i++ {
		rand.New(rand.NewSource(int64(i))).Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
		if !reflect.DeepEqual(after, PresentThreads(rows)) {
			t.Fatal("mixed known/unknown root depends on arrival order")
		}
	}
}

func TestPresentThreadsSlotCustomNameKeepsCanonicalLabel(t *testing.T) {
	row := presentationSlot("/src/repo", 2)
	row.Name = "task nickname"
	got := PresentThreads([]couchcore.ActionableThreadSummary{row})
	if got[0].Label != "repo:2" || got[0].Row.Name != "task nickname" {
		t.Fatalf("custom slot name displaced canonical label: %+v", got)
	}
}
