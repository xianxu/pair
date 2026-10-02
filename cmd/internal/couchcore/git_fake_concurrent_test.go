package couchcore

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestFakeGitConcurrentRunContextRetainsEveryCall(t *testing.T) {
	const callers, callsPerCaller = 16, 8
	replies := make(map[GitCall]string, callers)
	for i := 0; i < callers; i++ {
		replies[GitCall{Dir: fmt.Sprintf("/repo/%d", i), Args: "rev-parse HEAD"}] = fmt.Sprintf("head-%d\n", i)
	}
	git := NewFakeGit(replies)
	start := make(chan struct{})
	var joined sync.WaitGroup
	for i := 0; i < callers; i++ {
		joined.Add(1)
		go func(i int) {
			defer joined.Done()
			<-start
			for j := 0; j < callsPerCaller; j++ {
				got, err := git.RunContext(context.Background(), fmt.Sprintf("/repo/%d", i), "rev-parse", "HEAD")
				if want := fmt.Sprintf("head-%d", i); err != nil || got != want {
					t.Errorf("caller %d: reply=%q, err=%v; want %q", i, got, err, want)
				}
			}
		}(i)
	}
	close(start)
	joined.Wait()

	// Ops remains a post-execution observation surface: join every caller
	// before reading it, just as single-threaded fake consumers do.
	if len(git.Ops) != callers*callsPerCaller {
		t.Fatalf("recorded %d calls, want %d", len(git.Ops), callers*callsPerCaller)
	}
	counts := make(map[string]int)
	for _, op := range git.Ops {
		counts[op]++
	}
	for i := 0; i < callers; i++ {
		if op := fmt.Sprintf("/repo/%d: rev-parse HEAD", i); counts[op] != callsPerCaller {
			t.Errorf("recorded %d calls for %q, want %d", counts[op], op, callsPerCaller)
		}
	}
}
