package wrapcmd

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

type wrapPTYCountedCloser struct {
	calls atomic.Int32
	err   error
}

func (c *wrapPTYCountedCloser) Close() error { c.calls.Add(1); return c.err }

func TestPTYReleaseSlaveStableAndConcurrent(t *testing.T) {
	for _, failure := range []error{nil, errors.New("slave close failed")} {
		closer := &wrapPTYCountedCloser{err: failure}
		p := &startedPTY{slave: closer}
		var callers sync.WaitGroup
		for range 32 {
			callers.Go(func() {
				if got := p.releaseSlave(); got != failure {
					t.Errorf("release error %v, want original %v", got, failure)
				}
			})
		}
		callers.Wait()
		if closer.calls.Load() != 1 {
			t.Errorf("close calls=%d, want1", closer.calls.Load())
		}
		if got := p.releaseSlave(); got != failure {
			t.Errorf("repeated release=%v, want%v", got, failure)
		}
	}
	if err := (&startedPTY{}).releaseSlave(); err != nil {
		t.Errorf("nil lease release: %v", err)
	}
}

func TestPTYReleaseSlaveClosesActualDescriptorOnce(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	p := &startedPTY{slave: w}
	for range 2 {
		if err := p.releaseSlave(); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("slave descriptor survived release: %v", err)
	}
}
