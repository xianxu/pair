// Package zellijprobe is the harness every zellij-driving probe shares: start a
// session under a pty, read what it renders, tear it down.
//
// It exists because the copies diverged in a way that mattered. `cursorsaveslots`
// and `zellijscrollregion` held 196 IDENTICAL lines, and the copy in
// `cursorsaveslots` carried a defect the other's successor had already fixed: it
// discovered its session by DIFFING `zellij list-sessions` before and after, took
// an arbitrary new name (`for name := range after { break }` over a map), and
// force-deleted it. Any session that appeared in that window -- an operator
// launching a workbench, a parallel probe -- was a candidate for deletion, from
// `make test-smoke`, a routine target (pair#199 BR-51/BR-52).
//
// So the safety property is structural here rather than remembered: Start NAMES
// the session it creates, and Close deletes that name. A probe can only ever
// destroy a session it made.
package zellijprobe

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// Scrub returns the environment with every variable matching one of prefixes
// removed, plus extra appended.
//
// ZELLIJ* is not optional for a probe that starts zellij: zellij refuses to
// NEST, and with those set it prints a session list and exits -- which the first
// version of two different probes then read as a failing measurement.
func Scrub(prefixes []string, extra ...string) []string {
	env := make([]string, 0, len(os.Environ())+len(extra))
outer:
	for _, kv := range os.Environ() {
		for _, p := range prefixes {
			if strings.HasPrefix(kv, p) {
				continue outer
			}
		}
		env = append(env, kv)
	}
	return append(env, extra...)
}

// SyncBuffer is an append-only buffer safe across the reader goroutine and the
// verdict. A mutex rather than a channel because the reader never stops and
// nothing needs to observe individual writes -- but it must be synchronised:
// `go run -race` flags the unguarded version, and a torn read would corrupt the
// one artifact the verdict is computed from.
type SyncBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *SyncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *SyncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func (b *SyncBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buf)
}

// TailOf is the last n bytes, and it exists because slicing by hand panicked:
// `frame[len(frame)-3000:]` on a session that produced under 3000 bytes -- which
// is exactly the failed-session path a probe has to survive to report.
func TailOf(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// WriteLayout copies dir/layout.kdl to a temp file with each placeholder
// replaced. zellij resolves a relative command against ITS cwd, not ours, so
// commands in a probe layout have to be absolute and therefore substituted at
// run time.
func WriteLayout(dir string, replacements map[string]string) (string, error) {
	src, err := os.ReadFile(filepath.Join(dir, "layout.kdl"))
	if err != nil {
		return "", err
	}
	body := string(src)
	for placeholder, value := range replacements {
		body = strings.ReplaceAll(body, placeholder, value)
	}
	f, err := os.CreateTemp("", "zellijprobe-*.kdl")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		return "", err
	}
	return f.Name(), f.Close()
}

// Options describes the session to start.
type Options struct {
	// ConfigFile is the zellij config. Pass the repo's own: it sets
	// show_startup_tips false, without which zellij opens a tips plugin pane
	// that TAKES FOCUS -- and a dump-screen then dumps that pane instead, which
	// one probe mistook for its own output.
	ConfigFile string
	Layout     string
	// NamePrefix seeds the session name; the pid is appended so concurrent runs
	// cannot collide.
	NamePrefix string
	Rows, Cols uint16
	Env        []string
}

// Session is a zellij session THIS PROCESS CREATED, and therefore may delete.
type Session struct {
	Name string
	PTY  *os.File
	Seen *SyncBuffer
	cmd  *exec.Cmd
}

// Start creates a new named session running layout, under a pty of the given
// size, and begins collecting everything it renders.
//
// `--new-session-with-layout`, NOT `--layout` plus `--session`: with a layout,
// `--session` means ATTACH ("will be added to the session as a new tab"), so
// naming it up front made zellij try to attach to a session that did not exist.
// The diff-the-list workaround that fact produced is what BR-51 was about; this
// flag is the answer, and it is what lets the name be deterministic.
func Start(opts Options) (*Session, error) {
	if opts.Rows == 0 {
		opts.Rows = 24
	}
	if opts.Cols == 0 {
		opts.Cols = 80
	}
	name := fmt.Sprintf("%s-%d", opts.NamePrefix, os.Getpid())
	cmd := exec.Command("zellij",
		"--config", opts.ConfigFile,
		"--new-session-with-layout", opts.Layout,
		"--session", name)
	cmd.Env = opts.Env
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: opts.Rows, Cols: opts.Cols})
	if err != nil {
		return nil, err
	}
	s := &Session{Name: name, PTY: f, Seen: &SyncBuffer{}, cmd: cmd}
	go func() {
		buf := make([]byte, 8192)
		for {
			n, rerr := f.Read(buf)
			if n > 0 {
				_, _ = s.Seen.Write(buf[:n])
			}
			if rerr != nil {
				return
			}
		}
	}()
	return s, nil
}

// WaitUntilListed reports whether the session this probe created shows up in
// zellij's own list within limit.
//
// This is a PRECONDITION check, not a result: a probe whose session never came
// up has measured nothing, and reporting a verdict from it is the defect class
// pair#208 spent fourteen rounds on.
func (s *Session) WaitUntilListed(limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		out, _ := exec.Command("zellij", "list-sessions", "--short").Output()
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == s.Name {
				return true
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

// Action runs `zellij --session <ours> action …` against this session only.
func (s *Session) Action(env []string, args ...string) ([]byte, error) {
	full := append([]string{"--session", s.Name, "action"}, args...)
	cmd := exec.Command("zellij", full...)
	cmd.Env = env
	return cmd.CombinedOutput()
}

// Close tears down the pty and deletes THE SESSION THIS PROBE NAMED. Deleting by
// a name we chose is the whole point: there is no path here that can force-delete
// a session someone else created.
func (s *Session) Close() {
	if s == nil {
		return
	}
	if s.PTY != nil {
		_ = s.PTY.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	_ = exec.Command("zellij", "delete-session", s.Name, "--force").Run()
}
