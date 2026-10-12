package webcast

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
)

// peer is a scripted Chrome on the other end of the pipe: it records every
// command, answers each (or fails or ignores chosen methods), sends the
// events a test asks for, and exits like Chrome does, when its command pipe
// closes or Browser.close arrives.
type peer struct {
	t      *testing.T
	calls  chan cdp.Message
	wmu    sync.Mutex
	out    *io.PipeWriter
	failOn map[string]bool
	ignore map[string]bool
	// exitOnClose: Browser.close makes the peer exit, as Chrome does.
	exitOnClose bool
	// in is Couch's read end; closeReader makes an exit close it first, as
	// Launch's reaper does, so the reader fails with a closed pipe rather
	// than EOF.
	in          *io.PipeReader
	closeReader bool

	mu       sync.Mutex
	code     int
	kills    int
	exitOnce sync.Once
	exited   chan struct{}
}

func (p *peer) Kill() {
	p.mu.Lock()
	p.kills++
	p.mu.Unlock()
	p.exit(-1)
}
func (p *peer) Exited() <-chan struct{} { return p.exited }
func (p *peer) ExitCode() int           { p.mu.Lock(); defer p.mu.Unlock(); return p.code }

func (p *peer) exit(code int) {
	p.exitOnce.Do(func() {
		p.mu.Lock()
		p.code = code
		p.mu.Unlock()
		if p.closeReader {
			p.in.Close()
		}
		p.out.Close()
		close(p.exited)
	})
}

func (p *peer) send(raw string) {
	p.wmu.Lock()
	defer p.wmu.Unlock()
	p.out.Write(append([]byte(raw), 0))
}

func (p *peer) serve(in io.Reader) {
	r := bufio.NewReader(in)
	for {
		frame, err := r.ReadBytes(0)
		if err != nil {
			p.exit(0) // Chrome exits when its command pipe closes
			return
		}
		var m cdp.Message
		json.Unmarshal(frame[:len(frame)-1], &m)
		p.calls <- m
		switch {
		case p.ignore[m.Method]:
		case p.failOn[m.Method]:
			p.send(fmt.Sprintf(`{"id":%d,"error":{"code":-32000,"message":"no"}}`, m.ID))
		default:
			p.send(fmt.Sprintf(`{"id":%d,"result":{}}`, m.ID))
		}
		if m.Method == "Browser.close" && p.exitOnClose {
			p.exit(0)
		}
	}
}

// expect waits for the next command with method on session, skipping
// others, and returns it.
func (p *peer) expect(session, method string) cdp.Message {
	p.t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case m := <-p.calls:
			if m.Method == method && m.SessionID == session {
				return m
			}
		case <-deadline:
			p.t.Fatalf("no %s on %q", method, session)
		}
	}
}

// never fails if method arrives within d.
func (p *peer) never(method string, d time.Duration) {
	p.t.Helper()
	deadline := time.After(d)
	for {
		select {
		case m := <-p.calls:
			if m.Method == method {
				p.t.Fatalf("unexpected %s", method)
			}
		case <-deadline:
			return
		}
	}
}

func startPeer(t *testing.T, tweak func(*peer)) (*Chrome, *peer) {
	t.Helper()
	cmdR, cmdW := io.Pipe()
	outR, outW := io.Pipe()
	p := &peer{t: t, calls: make(chan cdp.Message, 256), out: outW, in: outR, exitOnClose: true,
		failOn: map[string]bool{}, ignore: map[string]bool{}, exited: make(chan struct{})}
	if tweak != nil {
		tweak(p)
	}
	go p.serve(cmdR)
	c := start(cdp.New(outR, cmdW), p, &tail{max: stderrTail}, LaunchOptions{})
	t.Cleanup(func() {
		c.Close()
		<-c.Done()
	})
	p.expect("", "Target.setAutoAttach")
	return c, p
}

// openPage announces page target with session, attached and held, as
// auto-attach does.
func (p *peer) openPage(target, session string) {
	p.send(`{"method":"Target.targetCreated","params":{"targetInfo":{"targetId":"` + target + `","type":"page","url":"about:blank","title":""}}}`)
	p.send(`{"method":"Target.attachedToTarget","params":{"sessionId":"` + session + `","targetInfo":{"targetId":"` + target + `","type":"page","url":"about:blank"},"waitingForDebugger":true}}`)
}

func waitDone(t *testing.T, c *Chrome) {
	t.Helper()
	select {
	case <-c.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Chrome did not end")
	}
}

func TestChromeSetsUpEachPageThenReleasesIt(t *testing.T) {
	c, p := startPeer(t, nil)
	p.openPage("T1", "S1")
	for _, m := range []string{"Runtime.addBinding", "Runtime.enable", "Page.enable", "Page.addScriptToEvaluateOnNewDocument", "Runtime.runIfWaitingForDebugger"} {
		p.expect("S1", m)
	}
	select {
	case <-c.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("not ready after the first page's setup")
	}
	waitState(t, c, "first page", func(s State) bool { return s.HasTab && s.Tab.ID == "T1" && s.Phase == PhaseIdle })
}

func TestChromeReleasesAPageWhoseSetupFailed(t *testing.T) {
	_, p := startPeer(t, func(p *peer) { p.failOn["Runtime.addBinding"] = true })
	p.openPage("T1", "S1")
	p.expect("S1", "Runtime.runIfWaitingForDebugger")
}

func TestChromePageControlToggles(t *testing.T) {
	c, p := startPeer(t, nil)
	p.openPage("T1", "S1")
	p.expect("S1", "Runtime.runIfWaitingForDebugger")
	p.send(`{"method":"Runtime.executionContextCreated","sessionId":"S1","params":{"context":{"id":7,"name":"couchcast","auxData":{"frameId":"T1"}}}}`)
	p.send(`{"method":"Runtime.bindingCalled","sessionId":"S1","params":{"name":"__couchcast","payload":"{\"t\":\"toggle\"}","executionContextId":7}}`)
	waitState(t, c, "live from the page", func(s State) bool { return s.Phase == PhaseLive })
	m := p.expect("S1", "Runtime.evaluate")
	if !strings.Contains(string(m.Params), `"contextId":7`) || !strings.Contains(string(m.Params), "(true)") {
		t.Fatalf("control not told it's live: %s", m.Params)
	}
	if err := c.SetLive(false); err != nil {
		t.Fatal(err)
	}
	if m := p.expect("S1", "Runtime.evaluate"); !strings.Contains(string(m.Params), "(false)") {
		t.Fatalf("control not told it's paused: %s", m.Params)
	}
}

func TestChromeEndsWhenTheLastPageStaysClosed(t *testing.T) {
	defer func(g time.Duration) { noPagesGrace = g }(noPagesGrace)
	noPagesGrace = 20 * time.Millisecond
	c, p := startPeer(t, nil)
	p.openPage("T1", "S1")
	p.expect("S1", "Runtime.runIfWaitingForDebugger")
	p.send(`{"method":"Target.targetDestroyed","params":{"targetId":"T1"}}`)
	p.expect("", "Browser.close")
	waitDone(t, c)
	if !errors.Is(c.Err(), ErrClosedByOperator) {
		t.Fatalf("err %v", c.Err())
	}
}

func TestChromeKeepsRunningWhenAPageReplacesTheLast(t *testing.T) {
	defer func(g time.Duration) { noPagesGrace = g }(noPagesGrace)
	noPagesGrace = 150 * time.Millisecond
	c, p := startPeer(t, nil)
	p.openPage("T1", "S1")
	p.expect("S1", "Runtime.runIfWaitingForDebugger")
	p.send(`{"method":"Target.targetDestroyed","params":{"targetId":"T1"}}`)
	p.openPage("T2", "S2")
	p.never("Browser.close", 3*noPagesGrace)
	if c.Err() != nil || c.State().Tab.ID != "T2" {
		t.Fatalf("err %v state %+v", c.Err(), c.State())
	}
}

// The first end cause wins: the last window closing, then Close, is still
// the operator's doing. Chrome ignoring Browser.close is killed after
// closeWait, and SetLive meanwhile refuses.
func TestChromeEndCausePrecedenceAndKill(t *testing.T) {
	defer func(g, w time.Duration) { noPagesGrace, closeWait = g, w }(noPagesGrace, closeWait)
	noPagesGrace, closeWait = 10*time.Millisecond, 300*time.Millisecond
	c, p := startPeer(t, func(p *peer) { p.exitOnClose = false })
	p.openPage("T1", "S1")
	p.expect("S1", "Runtime.runIfWaitingForDebugger")
	p.send(`{"method":"Target.targetDestroyed","params":{"targetId":"T1"}}`)
	p.expect("", "Browser.close")
	if err := c.SetLive(true); !errors.Is(err, ErrEnded) {
		t.Fatalf("SetLive while ending: %v", err)
	}
	c.Close()
	if !errors.Is(c.Err(), ErrClosedByOperator) {
		t.Fatalf("err %v, want the first cause", c.Err())
	}
	p.mu.Lock()
	kills := p.kills
	p.mu.Unlock()
	if kills != 1 {
		t.Fatalf("kills %d", kills)
	}
}

func TestChromeCloseIsNotAnError(t *testing.T) {
	c, p := startPeer(t, nil)
	p.openPage("T1", "S1")
	c.Close()
	if c.Err() != nil || c.State().Phase != PhaseEnded {
		t.Fatalf("err %v state %+v", c.Err(), c.State())
	}
	if err := c.SetLive(true); !errors.Is(err, ErrEnded) {
		t.Fatalf("SetLive after end: %v", err)
	}
}

func TestChromeExitReasons(t *testing.T) {
	t.Run("quit by hand", func(t *testing.T) {
		c, p := startPeer(t, nil)
		p.exit(0)
		waitDone(t, c)
		if !errors.Is(c.Err(), ErrClosedByOperator) {
			t.Fatalf("err %v", c.Err())
		}
	})
	// The reaper closing Couch's read end before the reader saw EOF is still
	// a quit, not a lost connection.
	t.Run("quit, reader closed first", func(t *testing.T) {
		c, p := startPeer(t, func(p *peer) { p.closeReader = true })
		p.exit(0)
		waitDone(t, c)
		if !errors.Is(c.Err(), ErrClosedByOperator) {
			t.Fatalf("err %v", c.Err())
		}
	})
	t.Run("crash", func(t *testing.T) {
		c, p := startPeer(t, nil)
		p.exit(11)
		waitDone(t, c)
		if c.Err() == nil || !strings.Contains(c.Err().Error(), "status 11") {
			t.Fatalf("err %v", c.Err())
		}
	})
	t.Run("lost connection", func(t *testing.T) {
		c, p := startPeer(t, nil)
		p.send(`{not json`)
		waitDone(t, c)
		if c.Err() == nil || errors.Is(c.Err(), ErrClosedByOperator) || !strings.Contains(c.Err().Error(), "lost Chrome") {
			t.Fatalf("err %v", c.Err())
		}
	})
}

func TestEndError(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		cause   endCause
		connErr error
		code    int
		want    error  // checked with errors.Is when set
		has     string // else the message contains this
	}{
		{endRequested, boom, 3, nil, ""},
		{endLastWindow, boom, 0, ErrClosedByOperator, ""},
		{notEnding, fmt.Errorf("cdp: read: %w", io.EOF), 0, ErrClosedByOperator, ""},
		{notEnding, cdp.ErrClosed, 0, ErrClosedByOperator, ""},
		{notEnding, cdp.ErrBacklog, 0, cdp.ErrBacklog, ""},
		{notEnding, fmt.Errorf("cdp: read: %w", io.ErrClosedPipe), 0, ErrClosedByOperator, ""},
		{notEnding, fmt.Errorf("cdp: read: %w", os.ErrClosed), 0, ErrClosedByOperator, ""},
		{notEnding, errors.New("cdp: write: broken pipe"), 0, ErrClosedByOperator, ""},
		{notEnding, nil, 9, nil, "status 9"},
		{notEnding, nil, 0, ErrClosedByOperator, ""},
	}
	for i, tc := range cases {
		got := endError(tc.cause, tc.connErr, tc.code, "")
		switch {
		case tc.has != "":
			if got == nil || !strings.Contains(got.Error(), tc.has) {
				t.Errorf("%d: %v, want %q", i, got, tc.has)
			}
		case tc.want == nil:
			if got != nil {
				t.Errorf("%d: %v, want nil", i, got)
			}
		case !errors.Is(got, tc.want):
			t.Errorf("%d: %v, want %v", i, got, tc.want)
		}
	}
}

func TestProfileInUse(t *testing.T) {
	inUse := "ERROR:process_singleton_posix.cc:347] Failed to create /p/SingletonLock: File exists (17)"
	other := "ERROR:process_singleton_posix.cc] Failed to create socket directory: Operation not permitted (1)"
	if !profileInUse(inUse) || !profileInUse("Opening in existing browser session.") || profileInUse(other) {
		t.Fatal("profileInUse misreads Chrome's output")
	}
}

// A process that exits while a child it started still holds its output pipe
// is reaped within processWaitDelay, so Close can't hang on Chrome's helpers.
func TestStartProcessReapsDespiteInheritedOutput(t *testing.T) {
	defer func(d time.Duration) { processWaitDelay = d }(processWaitDelay)
	processWaitDelay = 200 * time.Millisecond
	cmd := exec.Command("sh", "-c", "sleep 30 & exit 0")
	out := &tail{max: 1024}
	cmd.Stdout, cmd.Stderr = out, out
	reaped := false
	p, err := startProcess(cmd, func() { reaped = true })
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.Exited():
	case <-time.After(5 * time.Second):
		t.Fatal("not reaped while a child holds the output pipe")
	}
	if !reaped || p.ExitCode() != 0 {
		t.Fatalf("reaped %v code %d", reaped, p.ExitCode())
	}
}
