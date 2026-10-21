package testcouch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

type terminalProcess struct {
	done       chan struct{}
	cmd        *exec.Cmd
	file       *os.File
	endpoint   *terminal.Endpoint
	generation string
	output     buildOutput
	mu         sync.Mutex
	err        error
}

func startTerminal(cmd *exec.Cmd) (*terminalProcess, error) {
	if cmd.Env == nil {
		return nil, fmt.Errorf("native terminal requires an explicit environment")
	}
	file, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 32, Cols: 120})
	if err != nil {
		return nil, err
	}
	stream, err := ttyio.NewFile(file, file, true)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = file.Close()
		_ = cmd.Wait()
		return nil, err
	}
	ep, err := terminal.NewEndpoint("disposable-couch", terminal.Geometry{Rows: 32, Cols: 120}, stream)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = stream.Close()
		_ = cmd.Wait()
		return nil, err
	}
	p := &terminalProcess{done: make(chan struct{}), cmd: cmd, file: file, endpoint: ep, generation: procutil.Identity(fmt.Sprint(cmd.Process.Pid))}
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buffer := make([]byte, 8192)
		for {
			n, e := stream.Read(buffer)
			if n > 0 {
				_, _ = p.output.Write(buffer[:n])
				if _, feedErr := ep.Feed(buffer[:n], time.Now()); feedErr != nil {
					return
				}
			}
			if e != nil {
				return
			}
		}
	}()
	go func() {
		err := cmd.Wait()
		select {
		case <-readDone:
		case <-time.After(500 * time.Millisecond):
			_ = stream.Close()
			<-readDone
		}
		ep.EndInput()
		ep.Close()
		_ = stream.Close()
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
		close(p.done)
	}()
	return p, nil
}
func (p *terminalProcess) Close(ctx context.Context) error {
	select {
	case <-p.done:
		return nil
	default:
	}
	if p.generation == "" {
		return fmt.Errorf("direct child generation unavailable")
	}
	err := (nativeProcesses{}).Signal(p.cmd.Process.Pid, p.generation, syscall.SIGTERM)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (p *terminalProcess) Input(data []byte) error { _, err := p.endpoint.WriteInput(data); return err }
func (p *terminalProcess) Resize(rows, cols uint16) error {
	return p.endpoint.Resize(terminal.Geometry{Rows: int(rows), Cols: int(cols)}, func(terminal.Geometry) error { return pty.Setsize(p.file, &pty.Winsize{Rows: rows, Cols: cols}) })
}
func (p *terminalProcess) Screen() string {
	frame, err := p.endpoint.Snapshot(time.Now())
	if err != nil {
		return ""
	}
	var text strings.Builder
	for y := 0; y < frame.Geometry.Rows; y++ {
		for x := 0; x < frame.Geometry.Cols; x++ {
			c := frame.Cells[y*frame.Geometry.Cols+x]
			if c.Content == "" {
				text.WriteByte(' ')
			} else {
				text.WriteString(c.Content)
			}
		}
		text.WriteByte('\n')
	}
	return text.String()
}
func (p *terminalProcess) Output() string { return p.output.String() }
func waitUntil(ctx context.Context, f func() (bool, error)) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		ok, err := f()
		if err != nil || ok {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
