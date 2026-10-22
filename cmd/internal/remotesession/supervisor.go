package remotesession

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
)

// SupervisorConfig is supplied by Couch's leased namespace owner. Construction
// reads local authority only; default off starts no listener or network lookup.
type SupervisorConfig struct {
	Namespace         string
	Origin            string
	Host              *broadcast.Host
	CaptureConfigured bool
	Documents         DocumentBackend
	Verifier          *AccessVerifier
	Origins           *OriginStore // shared with local submit/forget; one Couch writer
}

// RemoteStatus deliberately contains no capability link. CLI replies may be
// recorded in a Pair child transcript; only the host console asks ConsoleLink.
type RemoteStatus struct {
	Status   string
	Detail   string
	CanReset bool
}

type remoteEpoch struct {
	ctx            context.Context
	cancel         context.CancelFunc
	sessions       *Sessions
	capability     string
	generation     uint64
	started        chan struct{}
	settled        chan struct{}
	mount          *broadcast.Mount // published before started closes
	openCleanupErr error            // independent of whether opening returned a mount
	cleanup        bool
	cleaned        bool
	cleanupErr     error
	operations     int
	requests       sync.WaitGroup // Add only under the shared gate while ready
}

// Supervisor owns the reducer, persistent command order and every exposure.
// No disk/network operation runs while its admission mutex is held.
type Supervisor struct {
	gate               admissionGate
	store              *Store
	paths              artifactpath.RemoteStatePaths
	policy             remotePolicy
	host               *broadcast.Host
	backend            DocumentBackend
	origins            *OriginStore
	audit              *Audit
	documentOperations map[string]int
	captureConfigured  bool
	changed            chan struct{}
	epoch              *remoteEpoch
	persistTail        <-chan struct{}
	persistPending     int
	detail             string
	handler            http.Handler
}

// NewSupervisor returns reset authority separately for capture by Couch's owned
// host-key branch. There is intentionally no exported Reset method or opcode.
func NewSupervisor(c SupervisorConfig) (*Supervisor, func() error, error) {
	p, err := newRemotePolicy(c.Origin)
	if err != nil {
		return nil, nil, err
	}
	if c.Host == nil {
		return nil, nil, errors.New("remote requires protected transport")
	}
	paths, err := artifactpath.ResolveCouchRemote(c.Namespace)
	if err != nil {
		return nil, nil, err
	}
	store, err := OpenStore(paths.Latch())
	if err != nil {
		return nil, nil, err
	}
	origins := c.Origins
	if origins == nil {
		origins, err = OpenOriginStore(paths.Origins())
		if err != nil {
			return nil, nil, err
		}
	}
	audit, err := OpenAudit(paths.Audit(), paths.PreviousAudit(), time.Now)
	if err != nil {
		return nil, nil, err
	}
	done := make(chan struct{})
	close(done)
	s := &Supervisor{store: store, paths: paths, policy: p, host: c.Host, backend: c.Documents, origins: origins, audit: audit,
		captureConfigured: c.CaptureConfigured, changed: make(chan struct{}, 1), persistTail: done,
	}
	s.handler = newRemoteServer(s, c.Verifier)
	s.gate.state.joined = true
	phase, _ := store.Snapshot()
	if phase != latchOff {
		s.gate.state.phase = phaseKilled
	}
	return s, s.consoleReset, nil
}

func (s *Supervisor) Changes() <-chan struct{} { return s.changed }
func (s *Supervisor) notifyLocked() {
	select {
	case s.changed <- struct{}{}:
	default:
	}
}
func (s *Supervisor) Snapshot() RemoteStatus {
	s.gate.mu.Lock()
	defer s.gate.mu.Unlock()
	v := RemoteStatus{Detail: s.detail}
	switch s.gate.state.phase {
	case phaseOff:
		v.Status = "off"
	case phasePreparing, phaseOpening:
		v.Status = "starting"
	case phaseReady:
		v.Status = "on"
	case phaseStopping:
		v.Status = "stopping"
	case phaseResetting:
		v.Status = "resetting"
	case phaseKilled:
		v.Status = "disabled"
		v.CanReset = s.gate.state.joined && s.persistPending == 0
	default:
		v.Status = "blocked"
	}
	return v
}
func (s *Supervisor) ConsoleLink() string {
	s.gate.mu.Lock()
	defer s.gate.mu.Unlock()
	if s.gate.state.phase != phaseReady || s.epoch == nil {
		return ""
	}
	return s.policy.origin + "/remote/" + s.epoch.capability + "/"
}

func (s *Supervisor) On() error {
	s.gate.mu.Lock()
	if s.captureConfigured {
		s.gate.mu.Unlock()
		return errors.New("remote unavailable with capture: unset COUCH_CAPTURE_DIR and restart Couch, or keep capture and skip remote")
	}
	if s.gate.state.phase != phaseOff {
		s.gate.mu.Unlock()
		return errors.New("remote is not off; inspect status at the machine")
	}
	s.gate.mu.Unlock()
	capability, err := randomCredential()
	if err != nil {
		return errors.New("remote capability generation failed")
	}
	s.gate.mu.Lock()
	defer s.gate.mu.Unlock()
	// Entropy acquisition is outside the admission gate. A concurrent kill or
	// another enable wins normally; these bytes cannot resurrect an old state.
	if s.gate.state.phase != phaseOff {
		return errors.New("remote is not off; inspect status at the machine")
	}
	s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventOn})
	if s.gate.state.phase != phasePreparing {
		return errors.New("remote lifecycle exhausted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &remoteEpoch{ctx: ctx, cancel: cancel, sessions: newSessions(s.gate.state.epoch), capability: capability, started: make(chan struct{}), settled: make(chan struct{})}
	s.epoch = e
	s.detail = ""
	attempt := s.gate.state.attempt
	s.persistLocked(func() error {
		var err error
		e.generation, err = s.store.Arm()
		return err
	}, func(err error) {
		if err != nil {
			close(e.started)
			s.blockLocked("remote latch publication uncertain; repair storage and restart Couch")
			return
		}
		if s.epoch != e || s.gate.state.phase != phasePreparing {
			close(e.started)
			return
		}
		s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventArmed, attempt: attempt})
		go s.open(e, attempt)
	})
	s.notifyLocked()
	return nil
}

// persistLocked schedules writes in acceptance order, including reset then
// kill. A delayed reset can never become the final write after a later kill.
// At most one enable/stop/reset plus one kill is outstanding; duplicate actions
// produce no extra job. Jobs do IO outside gate, then settle under that gate.
func (s *Supervisor) persistLocked(write func() error, complete func(error)) {
	prior := s.persistTail
	done := make(chan struct{})
	s.persistTail = done
	s.persistPending++
	go func() {
		<-prior
		err := write()
		s.gate.mu.Lock()
		s.persistPending--
		complete(err)
		close(done)
		s.finishLocked()
		s.notifyLocked()
		s.gate.mu.Unlock()
	}()
}

func (s *Supervisor) open(e *remoteEpoch, attempt uint64) {
	mt, err := s.host.MountProtected(e.ctx, remoteMount{s.handler})
	if err == nil && mt.Base() != s.policy.origin {
		err = errors.New("remote transport origin mismatch")
	}
	e.mount = mt
	if errors.Is(err, broadcast.ErrProtectedCleanup) {
		e.openCleanupErr = err
	}
	s.gate.mu.Lock()
	close(e.started)
	if err != nil {
		if s.epoch == e && (s.gate.state.phase == phaseOpening || s.gate.state.phase == phasePreparing) {
			detail := "remote transport failed; reset at the machine after cleanup"
			if errors.Is(err, broadcast.ErrTunnelBusy) {
				detail = "remote transport busy: stop the existing casts, then retry; reset at the machine after cleanup"
			}
			// Classify safe recovery text; connector errors may contain secrets.
			s.killLocked(detail)
		}
	} else {
		s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventOpened, attempt: attempt})
		if s.gate.state.phase != phaseReady || s.epoch != e {
			s.cleanupLocked(e)
		}
	}
	s.notifyLocked()
	s.gate.mu.Unlock()
	if mt != nil {
		select {
		case <-e.ctx.Done():
		case <-mt.Exited():
			s.gate.mu.Lock()
			if s.epoch == e && s.gate.state.phase == phaseReady {
				s.killLocked("remote transport ended; reset at the machine after cleanup")
			}
			s.gate.mu.Unlock()
		}
	}
}

func (s *Supervisor) Off() {
	s.gate.mu.Lock()
	defer s.gate.mu.Unlock()
	next, _ := ReduceRemote(s.gate.state, remoteEvent{kind: eventOff})
	if next == s.gate.state {
		return
	}
	s.gate.state = next
	s.revokeLocked()
	s.cleanupLocked(s.epoch)
	s.notifyLocked()
}

// Kill is the immediate authority fence. Persistence and connector teardown do
// not delay revocation, and a disk failure cannot restore an in-memory grant.
func (s *Supervisor) Kill() { s.gate.mu.Lock(); defer s.gate.mu.Unlock(); s.killLocked("") }
func (s *Supervisor) killLocked(detail string) {
	if s.gate.state.phase == phaseKilled || s.gate.state.phase == phaseBlocked {
		return
	}
	s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventKill})
	s.detail = detail
	s.revokeLocked()
	s.persistLocked(s.store.Kill, func(err error) {
		if err != nil {
			s.blockLocked("remote kill publication uncertain; repair storage and restart Couch")
		}
	})
	s.cleanupLocked(s.epoch)
	s.notifyLocked()
}
func (s *Supervisor) revokeLocked() {
	if e := s.epoch; e != nil {
		e.capability = ""
		e.sessions.Revoke()
		e.cancel()
	}
}
func (s *Supervisor) blockLocked(reason string) {
	s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventPoisoned, attempt: s.gate.state.attempt})
	s.detail = reason
	s.revokeLocked()
	s.cleanupLocked(s.epoch)
}
func (s *Supervisor) cleanupLocked(e *remoteEpoch) {
	if e == nil {
		s.finishLocked()
		return
	}
	if e.cleanup {
		return
	}
	e.cleanup = true
	go func() {
		<-e.started
		err := e.openCleanupErr
		if e.mount != nil {
			err = errors.Join(err, e.mount.UnmountConfirmed())
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if s.backend != nil {
			err = errors.Join(err, s.backend.Join(ctx))
		}
		joined := make(chan struct{})
		go func() { e.requests.Wait(); close(joined) }()
		select {
		case <-joined:
		case <-ctx.Done():
			err = errors.Join(err, ctx.Err())
		}
		s.gate.mu.Lock()
		e.cleaned = true
		e.cleanupErr = err
		if err != nil {
			s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventPoisoned, attempt: s.gate.state.attempt})
			s.detail = "remote disabled; cleanup unconfirmed; restart Couch after local inspection"
		}
		s.finishLocked()
		s.notifyLocked()
		s.gate.mu.Unlock()
	}()
}
func (s *Supervisor) finishLocked() {
	e := s.epoch
	if s.persistPending != 0 || (e != nil && !e.cleaned) {
		return
	}
	if e != nil {
		select {
		case <-e.settled:
		default:
			close(e.settled)
		}
	}
	if e != nil && e.cleanupErr != nil {
		return
	}
	state := s.gate.state
	if state.phase == phaseStopping && !state.joined {
		s.gate.state, _ = ReduceRemote(state, remoteEvent{kind: eventJoined, attempt: state.attempt})
		s.persistLocked(func() error { return s.store.stop(e.generation) }, func(err error) {
			if err != nil {
				s.blockLocked("remote stop publication uncertain; repair storage and restart Couch")
				return
			}
			s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventOffStored, attempt: state.attempt})
		})
	} else if state.phase == phaseKilled || state.phase == phaseBlocked {
		s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventJoined, attempt: state.attempt})
	}
}
func (s *Supervisor) consoleReset() error {
	s.gate.mu.Lock()
	defer s.gate.mu.Unlock()
	if s.gate.state.phase != phaseKilled || !s.gate.state.joined || s.persistPending != 0 {
		return errors.New("remote reset requires confirmed cleanup at the machine")
	}
	s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventConsoleReset})
	attempt := s.gate.state.attempt
	s.persistLocked(s.store.consoleReset, func(err error) {
		if err != nil {
			s.blockLocked("remote reset publication uncertain; repair storage and restart Couch")
			return
		}
		s.gate.state, _ = ReduceRemote(s.gate.state, remoteEvent{kind: eventResetStored, attempt: attempt})
	})
	s.notifyLocked()
	return nil
}

// Join observes the current exposure and accepted persistent work. Call Off or
// Kill first. It never grants or implicitly resets remote authority.
func (s *Supervisor) Join(ctx context.Context) error {
	for {
		s.gate.mu.Lock()
		tail := s.persistTail
		e := s.epoch
		done := s.persistPending == 0 && (e == nil || e.cleaned)
		var err error
		if e != nil {
			err = e.cleanupErr
		}
		s.gate.mu.Unlock()
		if done {
			return err
		}
		var settled <-chan struct{}
		if e != nil {
			settled = e.settled
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tail:
		case <-settled:
		}
		// A settled persistence tail can precede transport cleanup. Wait for a
		// state notification instead of spinning on its already-closed channel.
		s.gate.mu.Lock()
		still := s.persistPending == 0 && e != nil && !e.cleaned
		s.gate.mu.Unlock()
		if still {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-settled:
			}
		}
	}
}

type remoteMount struct{ http.Handler }

func (remoteMount) Owns(segment string) bool      { return segment == "remote" }
func (remoteMount) LockFor(string) *castlock.Lock { return nil }
