package wrapcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

// Startup records contain only effective identity values and fixed error facts.
// In particular, startup errors can contain argv or prompts, so their messages
// never enter this diagnostic channel. Existing stderr behavior is unchanged.
func startupIdentity(getenv func(string) string, pid int) map[string]any {
	fields := map[string]any{"actualPID": pid}
	for _, key := range []string{"PAIR_SCOPE_KEY", "PAIR_TAG", "PAIR_SESSION_NAME", "PAIR_LAUNCH_NONCE"} {
		value := strings.ToValidUTF8(getenv(key), "�")
		if len(value) > 128 {
			value = strings.ToValidUTF8(value[:115], "") + "<truncated>"
		}
		fields[key] = value
	}
	return fields
}

func (p *proxy) startupRecord(label string, fields map[string]any) error {
	if p.debugLogPath == "" {
		return nil
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	p.debugMu.Lock()
	defer p.debugMu.Unlock()
	if p.debugClosed {
		return io.ErrClosedPipe
	}
	if p.debugWriter == nil {
		p.debugWriter, err = diagnosticlog.OpenBuffered(p.debugLogPath, p.diagnosticLogOptions())
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(p.debugWriter, "[%s] %s: %q\n", time.Now().Format("15:04:05"), label, raw)
	return err
}

func (p *proxy) startupIdentity(stderr io.Writer) {
	if p.debugLogPath == "" {
		return
	}
	if err := p.startupRecord("STARTUP-identity", startupIdentity(os.Getenv, os.Getpid())); err != nil {
		startupDiagnosticUnavailable(stderr)
	}
}

func (p *proxy) startupError(stderr io.Writer, stage string, err error) {
	if p.debugLogPath == "" {
		return
	}
	var errno syscall.Errno
	errors.As(err, &errno)
	fields := map[string]any{"stage": stage, "error_type": fmt.Sprintf("%T", err), "errno": uint64(errno)}
	if e := p.startupRecord("STARTUP-error", fields); e != nil {
		startupDiagnosticUnavailable(stderr)
	}
}

func startupDiagnosticUnavailable(stderr io.Writer) {
	fmt.Fprintln(stderr, "pair-wrap: startup diagnostics unavailable")
}

func (p *proxy) closeStartupDiagnostics(stderr io.Writer) {
	if err := p.closeDebug(); err != nil {
		startupDiagnosticUnavailable(stderr)
	}
}

// Run can refuse storage protection before constructing the runtime proxy.
// Only that failure owns this short-lived writer; normal execution uses p's
// existing diagnostic writer and never opens a second owner.
func logEarlyStartupFailure(stderr io.Writer, err error) {
	p := &proxy{debugLogPath: os.Getenv("PAIR_WRAP_LOG")}
	defer p.closeStartupDiagnostics(stderr)
	p.startupIdentity(stderr)
	p.startupError(stderr, "storage-lease", err)
}
