package couchcore

import (
	"fmt"
	"sync"
	"testing"
)

// Writers no longer share one goroutine (pair#205): queue workers insert
// actors while the console forgets them and an abort retires one off the
// console goroutine. Every write goes through mutateRegistry, so none is lost
// in memory or on disk.
func TestConcurrentRegistryWritersLoseNoUpdate(t *testing.T) {
	env := newTestEnv(t, "/repo")
	c := env.Couch
	const n = 16
	doomed := make([]ActorRecord, n)
	for i := range doomed {
		doomed[i] = ActorRecord{ID: ActorID(fmt.Sprintf("doomed-%d", i)), Args: StartArgs{Worktree: "/repo"}}
	}
	if err := c.mutateRegistry(func(reg Registry) (Registry, error) {
		for _, a := range doomed {
			reg = reg.Insert(a)
		}
		return reg, nil
	}); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			kept := ActorRecord{ID: ActorID(fmt.Sprintf("kept-%d", i)), Args: StartArgs{Worktree: "/repo"}}
			if err := c.mutateRegistry(func(reg Registry) (Registry, error) { return reg.Insert(kept), nil }); err != nil {
				t.Error(err)
			}
		}(i)
		go func(i int) {
			defer wg.Done()
			if err := c.Forget("/repo", doomed[i].ID); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()

	check := func(where string, reg Registry) {
		ids := map[ActorID]bool{}
		for _, a := range reg.Records() {
			ids[a.ID] = true
		}
		for i := 0; i < n; i++ {
			if !ids[ActorID(fmt.Sprintf("kept-%d", i))] {
				t.Errorf("%s lost kept-%d", where, i)
			}
			if ids[doomed[i].ID] {
				t.Errorf("%s still holds forgotten %s", where, doomed[i].ID)
			}
		}
	}
	check("memory", c.actorRegistry())
	saved, _, err := c.Store.Load()
	if err != nil {
		t.Fatal(err)
	}
	check("disk", saved)
}
