package webcast

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// ShareState is where a replay share is.
type ShareState string

const (
	ShareOpening ShareState = "opening"
	ShareReady   ShareState = "ready"
	ShareFailed  ShareState = "failed"
	ShareLost    ShareState = "lost"
	ShareClosed  ShareState = "closed"
)

// ShareStatus is a share as its requester sees it: Base (the public
// <base>/<token>, for ReplayLink) once ready, and Local when the Host has no
// tunnel (the link works on this machine only); Err once failed or lost.
type ShareStatus struct {
	State ShareState
	Base  string
	Local bool
	Err   error
}

// ReplayShares are a running Couch's replay shares (#477): each recording
// asked for is mounted on the shared Host under its own token, served by a
// ReplayServer scoped to it, so a link grants the one recording it names.
//
//	(none)        --Share-->         opening
//	opening       --mounted-->       ready
//	opening       --Mount failed-->  failed
//	ready         --mount lost-->    lost
//	failed | lost --Share-->         opening (a new token)
//	any           --Close-->         closed
//
// Shares live in memory until Close (Couch's teardown); the last unmount
// closes the Host's tunnel. One per recording asked for in this run.
type ReplayShares struct {
	host *broadcast.Host
	root string
	// ctx is every share's mount: Close cancels it, abandoning an open.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	// onReady tells the operator a recording is now public (set before the
	// first Share); mounted is a test seam between a mount completing and
	// its share recording it.
	onReady func(id string)
	mounted func()

	mu     sync.Mutex
	closed bool
	shares map[string]*share
}

type share struct {
	status ShareStatus
	mount  *broadcast.Mount
}

// NewReplayShares shares the recordings under root on host.
func NewReplayShares(host *broadcast.Host, root string) *ReplayShares {
	ctx, cancel := context.WithCancel(context.Background())
	return &ReplayShares{host: host, root: root, ctx: ctx, cancel: cancel, shares: map[string]*share{}}
}

// OnReady makes f be called, outside the lock, each time a share becomes
// ready: the operator sees what the tunnel serves. Call before Share.
func (s *ReplayShares) OnReady(f func(id string)) { s.onReady = f }

// Shared is the recordings whose shares are ready, sorted.
func (s *ReplayShares) Shared() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []string
	for id, sh := range s.shares {
		if sh.status.State == ShareReady {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

var (
	// ErrNoRecording refuses a Share for an id with no recording under root.
	ErrNoRecording = errors.New("no such recording")
	// ErrNotShared answers Status for a recording never asked for.
	ErrNotShared = errors.New("not shared")
)

// Share asks for id to be shared and answers where its share is. It never
// waits on the mount: it starts one unless the share is opening or ready,
// so a failed or lost share starts again, under a new token. A requester
// polls Status, which reads, until the share leaves opening.
func (s *ReplayShares) Share(id string) ShareStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ShareStatus{State: ShareClosed}
	}
	if sh := s.shares[id]; sh != nil && (sh.status.State == ShareOpening || sh.status.State == ShareReady) {
		return sh.status
	}
	if !recordingExists(s.root, id) {
		return ShareStatus{State: ShareFailed, Err: ErrNoRecording}
	}
	token, err := broadcast.NewToken()
	if err != nil {
		return ShareStatus{State: ShareFailed, Err: err}
	}
	sh := &share{status: ShareStatus{State: ShareOpening}}
	s.shares[id] = sh
	s.wg.Add(1)
	go s.mount(sh, id, token)
	return sh.status
}

// Status is where id's share is, without starting one: failed with
// ErrNotShared if it was never asked for.
func (s *ReplayShares) Status(id string) ShareStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ShareStatus{State: ShareClosed}
	}
	if sh := s.shares[id]; sh != nil {
		return sh.status
	}
	return ShareStatus{State: ShareFailed, Err: ErrNotShared}
}

// mount opens sh's mount and watches it until it is lost or the shares
// close.
func (s *ReplayShares) mount(sh *share, id, token string) {
	defer s.wg.Done()
	// Ready once its viewer page answers through the link.
	m, err := s.host.Mount(s.ctx, NewRecordingServer(token, s.root, id), token+"/r/"+id+"/")
	if s.mounted != nil {
		s.mounted()
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		if m != nil {
			m.Unmount() // completed after Close: nobody serves it
		}
		return
	}
	if err != nil {
		sh.status = ShareStatus{State: ShareFailed, Err: err}
		s.mu.Unlock()
		return
	}
	sh.status, sh.mount = ShareStatus{State: ShareReady, Base: m.Base() + "/" + token, Local: m.Mode() == broadcast.ModeLocalOnly}, m
	s.mu.Unlock()
	if s.onReady != nil {
		s.onReady(id)
	}
	select {
	case <-m.Exited():
		s.mu.Lock()
		if sh.status.State == ShareReady {
			sh.status, sh.mount = ShareStatus{State: ShareLost, Err: m.Err()}, nil
		}
		s.mu.Unlock()
	case <-s.ctx.Done():
	}
}

// Close ends every share and returns once nothing is mounted: an opening
// mount is abandoned, a ready one unmounted. Share answers closed after.
func (s *ReplayShares) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.wg.Wait()
		return
	}
	s.closed = true
	var mounts []*broadcast.Mount
	for _, sh := range s.shares {
		if sh.mount != nil {
			mounts = append(mounts, sh.mount)
		}
		sh.status, sh.mount = ShareStatus{State: ShareClosed}, nil
	}
	s.mu.Unlock()
	s.cancel()
	for _, m := range mounts {
		m.Unmount()
	}
	s.wg.Wait()
}
