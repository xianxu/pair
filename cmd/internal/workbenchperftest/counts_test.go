package workbenchperftest

import (
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestCounterGeneratedArithmetic(t *testing.T) {
	var c Counter
	// The oracle stores events, independently sums each key, and never uses Delta.
	type event struct {
		key    Key
		n      int64
		decode bool
	}
	var events []event
	rng := rand.New(rand.NewSource(204))
	oracle := func(from int) Snapshot {
		s := Snapshot{Read: map[Key]int64{}, Decoded: map[Key]int64{}}
		for _, e := range events[from:] {
			if e.decode {
				s.Decoded[e.key] += e.n
			} else {
				s.Read[e.key] += e.n
			}
		}
		return s
	}
	for round := 0; round < 20; round++ {
		before, start := c.Snapshot(), len(events)
		for i := 0; i < 100; i++ {
			k := Key{Family: []string{"ledger", "native", "index"}[rng.Intn(3)], Owner: []string{"", "a", "b"}[rng.Intn(3)], Path: []string{"same", "other"}[rng.Intn(2)]}
			n := int64(rng.Intn(2001) - 1000)
			decode := rng.Intn(2) == 0
			events = append(events, event{k, n, decode})
			if decode {
				c.RecordDecode(k.Family, k.Owner, k.Path, n)
			} else {
				c.RecordRead(k.Family, k.Owner, k.Path, n)
			}
		}
		got, want := c.Snapshot(), oracle(0)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("round %d totals got=%+v want=%+v", round, got, want)
		}
		delta := got.Delta(before)
		expected := oracle(start)
		for k := range before.Read {
			if _, ok := expected.Read[k]; !ok {
				expected.Read[k] = 0
			}
		}
		for k := range before.Decoded {
			if _, ok := expected.Decoded[k]; !ok {
				expected.Decoded[k] = 0
			}
		}
		if !reflect.DeepEqual(delta, expected) {
			t.Fatalf("round %d delta got=%+v want=%+v", round, delta, expected)
		}
	}
}

func TestSnapshotsIndependentAndNegativeDelta(t *testing.T) {
	var c Counter
	c.RecordRead("ledger", "a", "p", 11)
	c.RecordDecode("native", "a", "t", 7)
	before := c.Snapshot()
	k := Key{"ledger", "a", "p"}
	before.Read[k] = 100
	before.Decoded[Key{"native", "a", "t"}] = 90
	before.Read[Key{"index", "gone", "g"}] = 3
	after := c.Snapshot()
	delta := after.Delta(before)
	if delta.Read[k] != -89 || delta.Read[Key{"index", "gone", "g"}] != -3 || delta.DecodeBytes() != -83 {
		t.Fatalf("lost negative delta: %+v", delta)
	}
	delta.Read[k] = 500
	if after.Read[k] != 11 || c.Snapshot().Read[k] != 11 {
		t.Fatal("snapshot/delta aliases counter")
	}
	if after.ReadBytes() != 11 || after.DecodeBytes() != 7 || after.ReadFamily("ledger") != 11 || after.DecodeFamily("native") != 7 {
		t.Fatalf("wrong totals: %+v", after)
	}
}

func TestCounterConcurrentJoinedAndObserver(t *testing.T) {
	var c Counter
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				c.RecordRead("index", "owner", "index", 3)
				c.Observer("owner").DecodeWork(sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "ledger-work.jsonl"}, 5)
				_ = c.Snapshot()
			}
		}()
	}
	wg.Wait()
	if got := c.Snapshot(); got.ReadBytes() != 24000 || got.DecodeBytes() != 40000 {
		t.Fatalf("lost concurrent updates: %+v", got)
	}
	c.Observer("native-owner").ReadWork(sessioninventory.Artifact{StorageRoot: "codex", RelativePath: "rollout.jsonl"}, 17)
	c.Observer("catalog-owner").ReadWork(sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "inventory.json"}, 19)
	got := c.Snapshot()
	if got.Read[Key{"native", "native-owner", "codex/rollout.jsonl"}] != 17 || got.Read[Key{"catalog", "catalog-owner", "pair-data/inventory.json"}] != 19 || got.Decoded[Key{"ledger", "owner", "pair-data/ledger-work.jsonl"}] != 40000 {
		t.Fatalf("observer identities lost: %+v", got)
	}
}
