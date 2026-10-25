package broadcast

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"testing/synctest"
)

// No listener is needed: the real Host close owner shuts down an idle HTTP
// server, then the existing fake handle holds completion at its Close boundary.
func heldHost(t *testing.T, h *Host) (*Mount, *fakeHandle, func()) {
	t.Helper()
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	t.Cleanup(unblock)
	f := &FakeTunnel{CloseBlock: release}
	handle := &fakeHandle{localHandle: newLocalHandle("http://fixture.invalid"), f: f}
	m := &Mount{host: h, exited: make(chan struct{})}
	h.mu.Lock()
	h.state, h.srv, h.handle = hostOpen, &http.Server{}, handle
	h.gen++
	h.mounts[m] = struct{}{}
	h.mu.Unlock()
	return m, handle, unblock
}

func TestMountUnmountJoinsOwnedShutdown(t *testing.T) {
	for _, cause := range []string{"tunnel loss", "server loss", "last unmount"} {
		t.Run(cause, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				h := NewHost(HostConfig{})
				m, handle, release := heldHost(t, h)
				defer release()
				if cause != "last unmount" {
					served := make(chan error, 1)
					go h.watch(h.gen, served, handle)
					if cause == "tunnel loss" {
						close(handle.exited)
					} else {
						served <- errors.New("listener failed")
					}
					synctest.Wait()
					if m.Err() == nil {
						t.Fatal("loss was not published")
					}
				}
				returned := make(chan struct{}, 3)
				for range 3 {
					go func() { m.Unmount(); returned <- struct{}{} }()
				}
				synctest.Wait()
				if n := len(returned); n != 0 {
					t.Errorf("%d Unmount calls returned before owned handle closure", n)
				}
				release()
				synctest.Wait()
				if n := len(returned); n != 3 || handle.f.Stats().Closes != 1 {
					t.Fatalf("joined calls=%d, closes=%d", n, handle.f.Stats().Closes)
				}

				// A new generation is now closing. The old mount must join its
				// completed receipt, rather than the Host's current attempt.
				next, _, releaseNext := heldHost(t, h)
				defer releaseNext()
				nextDone := make(chan struct{})
				go func() { next.Unmount(); close(nextDone) }()
				synctest.Wait()
				oldDone := make(chan struct{})
				go func() { m.Unmount(); close(oldDone) }()
				synctest.Wait()
				select {
				case <-oldDone:
				default:
					t.Error("old mount joined a later generation's shutdown")
				}
				select {
				case <-nextDone:
					t.Error("new mount returned before its own closure")
				default:
				}
				releaseNext()
				synctest.Wait()
			})
		})
	}
}
