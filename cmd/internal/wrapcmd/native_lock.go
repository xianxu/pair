package wrapcmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/nativelock"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

// One serialized owner applies every lifecycle transition and lock IO effect.
// Its caller closes it only AFTER joining owned-child group cleanup.
type wrapperNativeLocks struct {
	mu          sync.Mutex
	state       nativelock.State
	home, agent string
	// holderFn labels this wrapper; it runs once, on the first acquisition, so
	// a wrap that never locks a conversation never resolves it.
	holderFn func() string
	holder   string
	handles  []*nativelock.Handle
}

func newWrapperNativeLocks(home, agent string, holder func() string) *wrapperNativeLocks {
	return &wrapperNativeLocks{home: home, agent: agent, holderFn: holder}
}

func (o *wrapperNativeLocks) holderLabel() string {
	if o.holder == "" && o.holderFn != nil {
		o.holder = o.holderFn()
	}
	return o.holder
}

// errNoticeIgnored marks a binding notice that cannot change ownership: it
// arrived after stop began (a restart or exit is already under way), or its ID
// cannot name a lock. The agent keeps running; only a held lock refuses.
var errNoticeIgnored = errors.New("binding notice ignored")

func (o *wrapperNativeLocks) Discover(id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if phase := o.state.Phase(); phase == nativelock.Stopping || phase == nativelock.Exited {
		return fmt.Errorf("%w: wrapper is stopping", errNoticeIgnored)
	}
	key, err := nativelock.NewKey(o.agent, id)
	if err != nil {
		return fmt.Errorf("%w: %v", errNoticeIgnored, err)
	}
	next, effects, err := nativelock.Advance(o.state, nativelock.Event{Kind: nativelock.Discovered, Key: key})
	if err != nil {
		return err
	}
	o.state = next
	if effects.Acquire == nil {
		return nil
	}
	h, err := nativelock.Acquire(o.home, key, o.holderLabel())
	kind := nativelock.Acquired
	if err != nil {
		kind = nativelock.Failed
	}
	o.state, _, _ = nativelock.Advance(o.state, nativelock.Event{Kind: kind, Key: key})
	if h != nil {
		o.handles = append(o.handles, h)
	}
	return err
}
func (o *wrapperNativeLocks) Unbound() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.state.Phase() == nativelock.Unknown
}
func (o *wrapperNativeLocks) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.state, _, _ = nativelock.Advance(o.state, nativelock.Event{Kind: nativelock.Stop})
}
func (o *wrapperNativeLocks) Close() {
	o.Stop()
	o.mu.Lock()
	defer o.mu.Unlock()
	var effects nativelock.Effects
	o.state, effects, _ = nativelock.Advance(o.state, nativelock.Event{Kind: nativelock.CleanupComplete})
	if effects.Release {
		for _, h := range o.handles {
			_ = h.Close()
		}
		o.handles = nil
	}
}

type nativeBindingEvent struct {
	id  string
	err error
}
type bindingNoticeStream struct {
	events chan nativeBindingEvent
	reader *os.File
	stop   chan struct{}
	done   chan struct{}
	once   sync.Once
}

func readBindingNotices(reader *os.File, ordinal uint64) *bindingNoticeStream {
	stream := &bindingNoticeStream{events: make(chan nativeBindingEvent), reader: reader, stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(stream.done)
		defer close(stream.events)
		send := func(e nativeBindingEvent) bool {
			select {
			case stream.events <- e:
				return true
			case <-stream.stop:
				return false
			}
		}
		scan := bufio.NewScanner(reader)
		scan.Buffer(make([]byte, sessionwatch.MaxBindingNotice), sessionwatch.MaxBindingNotice)
		for scan.Scan() {
			id, err := sessionwatch.DecodeBindingNotice(scan.Bytes(), ordinal)
			if !send(nativeBindingEvent{id, err}) || err != nil {
				return
			}
		}
		err := scan.Err()
		if err == nil {
			err = io.EOF
		}
		send(nativeBindingEvent{err: err})
	}()
	return stream
}
func (s *bindingNoticeStream) Close() {
	if s == nil {
		return
	}
	s.once.Do(func() { close(s.stop); _ = s.reader.Close(); <-s.done })
}

// startNativePTY is shared by both wrapper command transports. Selectors retain
// their ordinary CLI meaning; exact IDs are locked before the native spawn.
func startNativePTY(locks *wrapperNativeLocks, cmd *exec.Cmd, stderr io.Writer, start func(*exec.Cmd) (*os.File, error)) (*os.File, error) {
	if !launcher.IsSupportedAgent(locks.agent) {
		return start(cmd)
	}
	selector, err := launcher.ParseNativeResumeSelector(locks.agent, cmd.Args[1:])
	if err != nil {
		return nil, fmt.Errorf("native resume selection: %w", err)
	}
	compat := launcher.CodexIndirectResumeCompatibility(selector)
	id := selector.NativeID
	if selector.Kind != launcher.NativeResumeExplicit || compat {
		id = ""
	}
	if selector.NewNativeID != "" {
		id = selector.NewNativeID
	}
	if id == "" && selector.Kind != launcher.NativeResumeFork && !compat {
		id = os.Getenv("PAIR_SESSION_ID")
	}
	if id != "" {
		if err := locks.Discover(id); err != nil {
			return nil, err
		}
	}
	if compat {
		fmt.Fprintln(stderr, launcher.CodexIndirectResumeCompatibilityNotice)
	}
	return start(cmd)
}

// nativeLockHolder labels this wrapper in its locks the way Couch names a slot
// ("pair:3"), so a refused resume says "in use by pair:3" from either observer
// (pair#595). Outside a slot it falls back to the Pair tag, then "standalone".
func nativeLockHolder() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if cwd, err := os.Getwd(); err == nil {
		if identity, err := couchcore.NewOSSlotCatalog(couchcore.OSProvisionIO{}).ResolveWorkspace(ctx, cwd); err == nil && identity.Address != nil {
			return *identity.Address
		}
	}
	if tag := os.Getenv("PAIR_TAG"); tag != "" {
		return tag
	}
	return "standalone"
}

// handleBindingNotice applies one watcher binding notice (pair#595). A held
// lock is a late collision: the owned group stops and the wrapper reports it.
// A notice that cannot change ownership is logged and dropped, so a restart
// already under way still re-execs and an unusable ID never kills the agent.
func (p *proxy) handleBindingNotice(event nativeBindingEvent) {
	if event.err != nil {
		p.nativeBindingNotices = nil
		if p.nativeLocks.Unbound() {
			p.emitNotification("Conversation lock discovery unavailable; relaunch to pin the conversation ID")
		}
		p.debug("LOCK-discovery-unavailable", event.err.Error())
		return
	}
	err := p.nativeLocks.Discover(event.id)
	switch {
	case err == nil:
	case errors.Is(err, errNoticeIgnored):
		p.debug("LOCK-notice-ignored", err.Error())
	default:
		p.nativeLockErr = err
		p.ownedGroup.StartStop()
	}
}

// stopForSignal is a controlled exit or agent restart: locks stop taking new
// conversations first, then the owned group stops; the locks themselves are
// released only after the group is reaped (pair#595). SIGUSR2 marks a restart,
// so the wrapper re-execs once the group is gone.
func (p *proxy) stopForSignal(s os.Signal, locks *wrapperNativeLocks, group *ownedGroup) {
	if s == syscall.SIGUSR2 {
		if !p.restartFresh.CompareAndSwap(false, true) {
			return
		}
		p.traceWrap("agent-restart-request", nil)
	}
	locks.Stop()
	group.StartStop()
}
