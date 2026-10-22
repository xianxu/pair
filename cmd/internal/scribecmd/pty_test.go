package scribecmd

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type countedCloser struct {
	calls atomic.Int64
	err   error
}

func (c *countedCloser) Close() error { c.calls.Add(1); return c.err }

func TestSlaveLeaseReleasesOnce(t *testing.T) {
	for _, n := range []int{8, 16} {
		for i := 0; i < n; i++ {
			want := errors.New("close failure")
			c := &countedCloser{err: want}
			p := &startedPTY{slave: c}
			var wg sync.WaitGroup
			for j := 0; j < 8; j++ {
				wg.Go(func() {
					if err := p.releaseSlave(); !errors.Is(err, want) {
						t.Errorf("release = %v", err)
					}
				})
			}
			wg.Wait()
			if c.calls.Load() != 1 {
				t.Fatalf("lease closed %d times", c.calls.Load())
			}
		}
	}
	p := &startedPTY{}
	if err := p.releaseSlave(); err != nil {
		t.Fatal(err)
	}
}
