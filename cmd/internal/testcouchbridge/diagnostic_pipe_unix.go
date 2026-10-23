//go:build unix

package testcouchbridge

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// DiagnosticPipe is a best-effort stderr sink: one nonblocking syscall, no
// poller, queue, retry or goroutine. The caller must not use or duplicate the
// writer afterward; O_NONBLOCK affects all copies of its open-file description.
// The descriptor remains owned by the executable, not the fleet's resource set.
type DiagnosticPipe struct{ file *os.File }

func NewDiagnosticPipe(file *os.File) (*DiagnosticPipe, error) {
	if file == nil {
		return nil, fmt.Errorf("stderr must be a pipe")
	}
	if _, err := configurePipe(file, true); err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}
	return &DiagnosticPipe{file: file}, nil
}

func (p *DiagnosticPipe) Write(data []byte) (int, error) {
	conn, err := p.file.SyscallConn()
	if err != nil {
		return 0, err
	}
	var count int
	var writeErr error
	err = conn.Control(func(fd uintptr) { count, writeErr = unix.Write(int(fd), data) })
	if err != nil {
		return 0, err
	}
	if count < 0 {
		count = 0
	}
	if writeErr == nil && count != len(data) {
		writeErr = io.ErrShortWrite
	}
	return count, writeErr
}
