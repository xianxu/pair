package wrapcmd

import (
	"bufio"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/xianxu/pair/cmd/internal/nativelock"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

// One serialized owner applies every lifecycle transition and lock IO effect.
// Its caller closes it only AFTER joining owned-child group cleanup.
type wrapperNativeLocks struct {
	mu                  sync.Mutex
	state               nativelock.State
	home, agent, holder string
	handles             []*nativelock.Handle
}

func newWrapperNativeLocks(home, agent, holder string) *wrapperNativeLocks {
	return &wrapperNativeLocks{home: home, agent: agent, holder: holder}
}
func (o *wrapperNativeLocks) Discover(id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	key, err := nativelock.NewKey(o.agent, id)
	if err != nil {
		return err
	}
	next, effects, err := nativelock.Advance(o.state, nativelock.Event{Kind: nativelock.Discovered, Key: key})
	if err != nil {
		return err
	}
	o.state = next
	if effects.Acquire == nil {
		return nil
	}
	h, err := nativelock.Acquire(o.home, key, o.holder)
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
