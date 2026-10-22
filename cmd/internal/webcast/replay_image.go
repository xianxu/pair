// Shared private replay/browser lifetime used by fidelity and sketch images.
package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// serveReplay serves the recordings under root on loopback under a fresh
// token, returning the server's base URL and how to stop it.
func serveReplay(root string) (string, func(), error) {
	token, err := broadcast.NewToken()
	if err != nil {
		return "", nil, err
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	srv := &http.Server{Handler: NewReplayServer(token, root), ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(l)
	return fmt.Sprintf("http://%s/%s", l.Addr(), token), func() { srv.Close() }, nil
}

// headless is a headless Chrome on its debugging pipe with one page, on a
// throwaway profile.
type headless struct {
	cmd        *exec.Cmd
	conn       *cdp.Conn
	session    string
	profile    string
	vp         image.Point // the page's current device metrics
	output     *os.File
	commands   *os.File
	eventsDone chan struct{}
	closeOnce  sync.Once
	closeErr   error
}

func launchHeadless(ctx context.Context, bin string) (*headless, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	profile, err := os.MkdirTemp("", "couch-fidelity-")
	if err != nil {
		return nil, err
	}
	cmdR, cmdW, err := os.Pipe()
	if err != nil {
		os.RemoveAll(profile)
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		cmdR.Close()
		cmdW.Close()
		os.RemoveAll(profile)
		return nil, err
	}
	cmd := exec.Command(bin, "--remote-debugging-pipe", "--headless=new", "--user-data-dir="+profile,
		"--no-first-run", "--no-default-browser-check", "--hide-scrollbars", "--force-device-scale-factor=1", "about:blank")
	cmd.ExtraFiles = []*os.File{cmdR, outW}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Start()
	cmdR.Close()
	outW.Close()
	if err != nil {
		cmdW.Close()
		outR.Close()
		os.RemoveAll(profile)
		return nil, err
	}
	h := &headless{cmd: cmd, conn: cdp.New(outR, cmdW), profile: profile, output: outR, commands: cmdW, eventsDone: make(chan struct{})}
	go func() {
		defer close(h.eventsDone)
		for range h.conn.Events() { // nothing listened for; keep the queue empty
		}
	}()
	var target struct {
		TargetID string `json:"targetId"`
	}
	var att struct {
		SessionID string `json:"sessionId"`
	}
	if err := h.call(ctx, "", "Target.createTarget", map[string]any{"url": "about:blank"}, &target); err != nil {
		return nil, errors.Join(err, h.closeJoined())
	}
	if err := h.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": target.TargetID, "flatten": true}, &att); err != nil {
		return nil, errors.Join(err, h.closeJoined())
	}
	h.session = att.SessionID
	return h, nil
}

func (h *headless) call(ctx context.Context, session, method string, params, result any) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return h.conn.Call(ctx, session, method, params, result)
}

// eval runs a script in the page, awaiting a promise, into result.
func (h *headless) eval(ctx context.Context, expr string, result any) error {
	var res struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := h.call(ctx, h.session, "Runtime.evaluate", map[string]any{"expression": expr, "awaitPromise": true, "returnByValue": true}, &res); err != nil {
		return err
	}
	if e := res.ExceptionDetails; e != nil {
		return fmt.Errorf("%s %s", e.Text, e.Exception.Description)
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(res.Result.Value, result)
}

// open loads the fidelity page and waits until it has the records.
func (h *headless) open(ctx context.Context, url string) error {
	return h.openReady(ctx, url, "fidelity")
}

func (h *headless) openReady(ctx context.Context, url, api string) error {
	if err := h.call(ctx, h.session, "Page.navigate", map[string]any{"url": url}, nil); err != nil {
		return err
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		var ok bool
		err := h.eval(ctx, fmt.Sprintf("window.%s ? window.%s.ready.then(() => true) : false", api, api), &ok)
		if err == nil && ok {
			return nil
		}
		if time.Now().After(deadline) {
			if err == nil {
				err = errors.New("the page never loaded")
			}
			return fmt.Errorf("%s page: %w", api, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.conn.Done(): // Chrome is gone: no point waiting
			return fmt.Errorf("%s page: Chrome: %v", api, h.conn.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (h *headless) close() { _ = h.closeJoined() }

const replayBrowserCleanupBudget = 5 * time.Second

// closeJoined retains signal authority only until the unreaped leader is
// joined. Afterwards a numeric process group is observation, never authority.
func (h *headless) closeJoined() error {
	ctx, cancel := context.WithTimeout(context.Background(), replayBrowserCleanupBudget)
	defer cancel()
	return h.closeJoinedContext(ctx)
}

// All stages share one absolute deadline; a failed stage cannot renew it.
func (h *headless) closeJoinedContext(ctx context.Context) error {
	h.closeOnce.Do(func() {
		cleanupCertain := true
		graceCtx, graceCancel := context.WithTimeout(ctx, time.Second)
		canGrace := ctx.Err() == nil
		if canGrace && h.commands != nil {
			deadline, _ := graceCtx.Deadline()
			// Bound the pipe write itself: CDP Call's context bounds reply
			// waiting, while an unresponsive peer could fill its input pipe.
			canGrace = h.commands.SetWriteDeadline(deadline) == nil
		}
		if canGrace {
			_ = h.call(graceCtx, "", "Browser.close", nil, nil)
		}
		graceCancel()
		if h.cmd.Process != nil {
			err := syscall.Kill(-h.cmd.Process.Pid, syscall.SIGKILL)
			if err != nil && !errors.Is(err, syscall.ESRCH) {
				h.closeErr = errors.Join(h.closeErr, err)
			}
			waitDone := make(chan error, 1)
			go func() { waitDone <- h.cmd.Wait() }()
			select {
			case waitErr := <-waitDone:
				var exit *exec.ExitError
				if waitErr != nil && !errors.As(waitErr, &exit) {
					h.closeErr = errors.Join(h.closeErr, waitErr)
				}
			case <-ctx.Done():
				cleanupCertain = false
				h.closeErr = errors.Join(h.closeErr, errors.New("replay browser cleanup uncertain: leader wait did not finish"))
			}
		}
		_ = h.conn.Close()
		if h.output != nil {
			_ = h.output.Close()
		}
		if e := h.conn.Wait(ctx); e != nil {
			cleanupCertain = false
			h.closeErr = errors.Join(h.closeErr, fmt.Errorf("replay browser cleanup uncertain: protocol join: %w", e))
		}
		if h.eventsDone != nil {
			select {
			case <-h.eventsDone:
			case <-ctx.Done():
				cleanupCertain = false
				h.closeErr = errors.Join(h.closeErr, ctx.Err())
			}
		}
		// Child reaping can lag the leader. Observe only; never signal after Wait.
		for h.cmd.Process != nil {
			e := syscall.Kill(-h.cmd.Process.Pid, 0)
			gone := errors.Is(e, syscall.ESRCH)
			if errors.Is(e, syscall.EPERM) {
				members, inspectErr := procutil.ProcessGroupMembers(h.cmd.Process.Pid)
				gone = inspectErr == nil && !members
			}
			if gone {
				break
			}
			if ctx.Err() != nil {
				cleanupCertain = false
				h.closeErr = errors.Join(h.closeErr, fmt.Errorf("replay browser cleanup uncertain: process group %d remains (%v)", h.cmd.Process.Pid, e))
				break
			}
			select {
			case <-ctx.Done():
			case <-time.After(10 * time.Millisecond):
			}
		}
		if cleanupCertain {
			h.closeErr = errors.Join(h.closeErr, os.RemoveAll(h.profile))
		} else {
			h.closeErr = errors.Join(h.closeErr, fmt.Errorf("replay browser profile retained for reconciliation: %s", h.profile))
		}
	})
	return h.closeErr
}
