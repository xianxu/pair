package couchtty

import (
	"context"
	"strings"
	"sync"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/remotesession"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

type remoteSessionControl interface {
	On() error
	Off()
	Snapshot() remotesession.RemoteStatus
	ConsoleLink() string
	Changes() <-chan struct{}
	Join(context.Context) error
}

// remoteConsoleView is a leaf-lock snapshot for the operator-only layer.
// Neither the chrome nor a notice ever receives this capability link.
type remoteConsoleView struct {
	mu                                sync.Mutex
	link                              string
	cacheLink                         string
	cacheCols, cacheRows, cacheHeight int
	cacheCells                        []terminal.Cell
}

// SetRemoteSession is startup wiring from Couch's leased composition root.
// Reset authority is retained privately and invoked only by a decoded host key.
func (c *Console) SetRemoteSession(s *remotesession.Supervisor, consoleReset func() error, configError string) {
	c.mu.Lock()
	if s != nil {
		c.remoteSession = s
	}
	c.remoteConsoleReset = consoleReset
	c.remoteConfigError = configError
	c.mu.Unlock()
	c.refreshRemoteSession()
	if s == nil {
		return
	}
	// Startup wiring precedes Run/teardown, so this worker is registered before
	// the console can begin its join (GoTracked runs inline before Run).
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		for {
			select {
			case _, ok := <-s.Changes():
				if !ok {
					return
				}
				c.refreshRemoteSession()
			case <-c.lifetime.Done():
				return
			}
		}
	}()
}

func (c *Console) refreshRemoteSession() {
	c.mu.Lock()
	s, configError := c.remoteSession, c.remoteConfigError
	c.mu.Unlock()
	status := remotesession.RemoteStatus{}
	link := ""
	if s != nil {
		status, link = s.Snapshot(), s.ConsoleLink()
	} else if configError != "" {
		status = remotesession.RemoteStatus{Status: "blocked", Detail: configError}
	}
	c.mu.Lock()
	c.remoteStatus = status
	c.mu.Unlock()
	c.remoteView.mu.Lock()
	c.remoteView.link = link
	c.remoteView.mu.Unlock()
	c.repaint()
}

func (c *Console) toggleRemoteSession() {
	c.mu.Lock()
	s, status := c.remoteSession, c.remoteStatus
	c.mu.Unlock()
	if s == nil {
		c.setNotice("Remote unavailable: configure protected named ingress and Cloudflare Access")
		return
	}
	if status.Status == "off" {
		if err := s.On(); err != nil {
			c.setNotice("Remote: " + err.Error())
		}
	} else if status.Status == "disabled" {
		c.setNotice("Remote disabled: wait for teardown, then press Ctrl+Alt+R at this machine to reset; click again to enable")
	} else if status.Status == "blocked" {
		c.setNotice("Remote blocked: repair its configuration or state and restart Couch")
	} else {
		s.Off()
	}
	c.refreshRemoteSession()
}

// routeRemoteSessionKey is reachable only from routeInputEvent's decoded
// KeyPress branch, after terminal replies are rejected. Repeats do not reset.
// Ctrl+Alt+R clears a settled kill latch; it does not enable remote mode.
func (c *Console) routeRemoteSessionKey(key uv.KeyPressEvent) bool {
	k := uv.Key(key)
	if k.Code != 'r' || k.IsRepeat || k.Mod&^(uv.ModCapsLock|uv.ModNumLock|uv.ModScrollLock) != uv.ModCtrl|uv.ModAlt {
		return false
	}
	c.mu.Lock()
	canReset := c.remoteStatus.Status == "disabled" && c.remoteStatus.CanReset
	reset := c.remoteConsoleReset
	c.mu.Unlock()
	if !canReset || reset == nil {
		return false
	}
	if err := reset(); err != nil {
		c.setNotice("Remote reset: " + err.Error())
	}
	c.refreshRemoteSession()
	return true
}

func (c *Console) endRemoteSessionForShutdown() {
	c.mu.Lock()
	s := c.remoteSession
	c.mu.Unlock()
	if s != nil {
		s.Off()
		if err := s.Join(context.Background()); err != nil {
			c.terminalError(err)
		}
	}
}

// remoteSessionLayer is composed with the shared Codes layer, exclusively on
// Presenter's operator path. No listener, capture or broadcast sees its cells.
func (c *Console) remoteSessionLayer(f terminal.Frame) terminal.Frame {
	c.remoteView.mu.Lock()
	link := c.remoteView.link
	cols, rows := f.Geometry.Cols, f.Geometry.Rows
	cached := c.remoteView.cacheLink == link && c.remoteView.cacheCols == cols && c.remoteView.cacheRows == rows
	cells, height := c.remoteView.cacheCells, c.remoteView.cacheHeight
	c.remoteView.mu.Unlock()
	if link == "" || len(link) > 1024 || cols < 1 || rows < 3 || len(f.Cells) != cols*rows {
		return f
	}
	if !cached {
		cells, height = remoteSessionCells(link, cols, rows)
		c.remoteView.mu.Lock()
		if c.remoteView.link != link {
			c.remoteView.mu.Unlock()
			return f
		}
		c.remoteView.cacheLink, c.remoteView.cacheCols, c.remoteView.cacheRows = link, cols, rows
		c.remoteView.cacheCells, c.remoteView.cacheHeight = cells, height
		c.remoteView.mu.Unlock()
	}
	if len(cells) == 0 {
		return f
	}
	out := f.Clone()
	start := rows - 1 - height
	copy(out.Cells[start*cols:], cells)
	for y := start; y < rows-1 && y < len(out.Rows); y++ {
		out.Rows[y].UsedColumns = cols
	}
	return out
}

func remoteSessionCells(link string, cols, rows int) ([]terminal.Cell, int) {
	original := link
	link = rowtext.Sanitize(link)
	// The text remains complete even when a narrow terminal cannot fit the QR.
	lines := []string{"Remote link (console only)"}
	for len(link) > 0 {
		n := min(len(link), cols)
		lines = append(lines, link[:n])
		link = link[n:]
	}
	if len(lines) > rows-1 {
		return nil, 0
	}
	qr := remoteLinkQR(original, cols, rows-1-len(lines))
	lines = append(lines, qr...)
	cells, err := terminal.StyledRows(strings.Join(lines, "\n"), cols, len(lines))
	if err != nil {
		return nil, 0
	}
	return cells, len(lines)
}
