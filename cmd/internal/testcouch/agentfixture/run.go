package agentfixture

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/pathscope"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// Config names only disposable writable roots and the fixed read-only Couch
// executable. CLI children inherit this actor process's real os.Environ.
type Size struct{ Cols, Rows int }

type Config struct {
	Resizes                                                      <-chan Size
	Root, ControlDir, Home, CodexHome, Agent, CouchBinary, RunID string
	Args                                                         []string
	Width, Height                                                int
}
type Ready struct {
	RunID      string `json:"run_id"`
	Agent      string `json:"agent"`
	Version    string `json:"version"`
	PID        int    `json:"pid"`
	NativeID   string `json:"native_id"`
	Transcript string `json:"transcript"`
	CWD        string `json:"cwd"`
}
type AcceptedInput struct {
	PID      int       `json:"pid"`
	Sequence uint64    `json:"sequence"`
	NativeID string    `json:"native_id"`
	Body     string    `json:"body"`
	At       time.Time `json:"at"`
}

func ConfigFromEnvironment(agent string, args []string) (Config, error) {
	c := Config{Agent: agent, Args: args, Root: os.Getenv("PAIR_TESTCOUCH_ROOT"), RunID: os.Getenv("PAIR_TESTCOUCH_RUN_ID"), Home: os.Getenv("HOME"), CodexHome: os.Getenv("CODEX_HOME"), CouchBinary: os.Getenv("PAIR_TESTCOUCH_COUCH")}
	if c.CodexHome == "" {
		c.CodexHome = filepath.Join(c.Home, ".codex")
	}
	tree := os.Getenv("COUCH_TREE")
	cwd, err := os.Getwd()
	if err != nil {
		return c, err
	}
	if tree == "" || tree != cwd {
		return c, errors.New("fixture actual COUCH_TREE must equal its canonical cwd")
	}
	base := os.Getenv("PAIR_TESTCOUCH_CONTROL")
	if !filepath.IsAbs(base) {
		return c, errors.New("fixture control base must be absolute")
	}
	c.ControlDir = ActorDirectory(base, tree)
	return c, c.validate()
}
func (c Config) validate() error {
	if Version(c.Agent) == "" {
		return errors.New("fixture agent must be claude or codex")
	}
	root, err := pathscope.CanonicalFuture(c.Root)
	if err != nil || root != c.Root || root == "/" {
		return errors.New("fixture root must be a canonical disposable directory")
	}
	if c.RunID == "" || len(c.RunID) > 128 || c.Root != os.Getenv("COUCH_ISOLATED_ROOT") {
		return errors.New("fixture requires matching isolated root and run owner")
	}
	owner, err := readBounded(filepath.Join(c.Root, OwnershipFile), 256)
	if err != nil || strings.TrimSpace(string(owner)) != c.RunID {
		return errors.New("fixture root owner mismatch")
	}
	if c.Home != os.Getenv("HOME") {
		return errors.New("fixture HOME differs from actor environment")
	}
	if explicit := os.Getenv("CODEX_HOME"); explicit != "" && explicit != c.CodexHome {
		return errors.New("fixture CODEX_HOME differs from actor environment")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	paths := []string{c.ControlDir, c.Home, c.CodexHome, cwd}
	for _, key := range []string{"COUCH_TREE", "COUCH_STORE_DIR", "COUCH_IDENTITY_DIR", "PAIR_DATA_DIR", "COUCH_PAIR_DATA_DIR", "CLAUDE_CONFIG_DIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR", "ZELLIJ_SOCKET_DIR", "ZELLIJ_CONFIG_DIR"} {
		if p := os.Getenv(key); p != "" {
			paths = append(paths, p)
		}
	}
	for _, path := range paths {
		physical, err := pathscope.CanonicalFuture(path)
		if err != nil || physical != path || (!pathscope.Within(c.Root, physical) || physical == c.Root) {
			return fmt.Errorf("fixture writable path escapes owned root: %q", path)
		}
	}
	if !filepath.IsAbs(c.CouchBinary) {
		return errors.New("fixture Couch executable must be absolute")
	}
	info, err := os.Stat(c.CouchBinary)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return errors.New("fixture Couch executable unavailable")
	}
	return nil
}

// Run requires a closable owned input so cancellation can join its sole reader.
// os.Stdin and io.PipeReader satisfy the contract; no producer survives Run.
func Run(parent context.Context, c Config, stdin io.Reader, stdout io.Writer) (result error) {
	if err := c.validate(); err != nil {
		return err
	}
	input, ok := stdin.(io.ReadCloser)
	if !ok {
		return errors.New("fixture stdin must be closable for joined cancellation")
	}
	for _, p := range []string{c.ControlDir, c.Home, c.CodexHome, filepath.Join(c.ControlDir, "commands"), filepath.Join(c.ControlDir, "results")} {
		if err := privateDirectory(p); err != nil {
			return err
		}
	}
	cv, err := openConversation(c)
	if err != nil {
		return err
	}
	defer cv.Close()
	if c.Width == 0 {
		c.Width = 120
	}
	if c.Height == 0 {
		c.Height = 38
	}
	state := AgentState{Agent: c.Agent, Phase: Idle}
	capture, err := CapturedComposer(c.Agent)
	if err != nil {
		return err
	}
	if _, err = stdout.Write(append(capture, []byte(Paint(state, c.Width, c.Height))...)); err != nil {
		return err
	}
	ready := Ready{RunID: c.RunID, Agent: c.Agent, Version: Version(c.Agent), PID: os.Getpid(), NativeID: cv.id, Transcript: cv.path, CWD: cv.cwd}
	if err := atomicJSON(filepath.Join(c.ControlDir, "ready.json"), ready, false); err != nil {
		return err
	}
	if err := atomicJSON(filepath.Join(c.ControlDir, "state.json"), state, false); err != nil {
		return err
	}
	accepted, err := os.OpenFile(filepath.Join(c.ControlDir, "accepted.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer accepted.Close()
	ctx, cancel := context.WithCancel(parent)
	var workers sync.WaitGroup
	type inputEvent struct {
		bytes []byte
		err   error
	}
	chunks := make(chan inputEvent, 4)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf := make([]byte, 4096)
		for {
			n, e := input.Read(buf)
			if n > 0 {
				select {
				case chunks <- inputEvent{bytes: append([]byte(nil), buf[:n]...)}:
				case <-ctx.Done():
					return
				}
			}
			if e != nil {
				select {
				case chunks <- inputEvent{err: e}:
				case <-ctx.Done():
				}
				return
			}
		}
	}()
	defer func() { cancel(); _ = input.Close(); <-readDone; workers.Wait() }()
	// Only the bounded command mailbox is polled; fleet liveness is never polled.
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	finished := make(chan Result, MaxConcurrentCLI)
	active := 0
	processed := 0
	seen := map[string]bool{}
	publishState := func() error {
		if _, e := io.WriteString(stdout, Paint(state, c.Width, c.Height)); e != nil {
			return e
		}
		return atomicJSON(filepath.Join(c.ControlDir, "state.json"), state, false)
	}
	sizes := c.Resizes
	for {
		select {
		case size, open := <-sizes:
			if !open {
				sizes = nil
				continue
			}
			if size.Cols < 20 || size.Cols > 240 || size.Rows < 10 || size.Rows > 120 {
				return errors.New("fixture resize outside supported dimensions")
			}
			c.Width, c.Height = size.Cols, size.Rows
			if e := publishState(); e != nil {
				return e
			}
		case <-ctx.Done():
			return nil
		case event := <-chunks:
			if event.err != nil {
				if errors.Is(event.err, io.EOF) || errors.Is(event.err, os.ErrClosed) {
					return nil
				}
				return event.err
			}
			chunk := event.bytes
			previousAccepted := state.Accepted
			next, bodies, e := ReduceAgent(state, chunk)
			if e != nil {
				return e
			}
			state = next
			for index, body := range bodies {
				if e = cv.accept(body); e != nil {
					return e
				}
				raw, e := json.Marshal(AcceptedInput{PID: os.Getpid(), Sequence: previousAccepted + uint64(index) + 1, NativeID: cv.id, Body: body, At: time.Now().UTC()})
				if e != nil {
					return e
				}
				if _, e = accepted.Write(append(raw, '\n')); e != nil {
					return e
				}
				// Echo only after submission; the repaint below clears the composer,
				// which production peer delivery observes before publishing a receipt.
				safe := strings.Map(func(r rune) rune {
					if r == 0x1b || r == 0 {
						return -1
					}
					return r
				}, body)
				if _, e = io.WriteString(stdout, "\r\n"+safe+"\r\nFixture completed the submitted input.\r\n"); e != nil {
					return e
				}
			}
			if e = publishState(); e != nil {
				return e
			}
		case r := <-finished:
			active--
			if err := atomicJSON(filepath.Join(c.ControlDir, "results", r.ID+".json"), r, true); err != nil {
				return err
			}
		case <-ticker.C:
			entries, e := os.ReadDir(filepath.Join(c.ControlDir, "commands"))
			if e != nil {
				return e
			}
			if len(entries) > MaxRequests {
				return errors.New("fixture command mailbox exceeds limit")
			}
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasPrefix(name, ".publish-") {
					continue
				}
				if !strings.HasSuffix(name, ".json") || entry.IsDir() {
					return errors.New("invalid fixture mailbox entry")
				}
				id := strings.TrimSuffix(name, ".json")
				if !requestID.MatchString(id) {
					return errors.New("invalid fixture mailbox filename")
				}
				if seen[id] {
					return errors.New("duplicate fixture request ID")
				}
				path := filepath.Join(c.ControlDir, "commands", name)
				raw, e := readBounded(path, MaxCommandBytes)
				var command Command
				if e == nil {
					command, e = DecodeCommand(raw)
				}
				if e == nil && command.ID != id {
					e = errors.New("fixture filename/request correlation mismatch")
				}
				if e == nil && (command.PID != os.Getpid() || command.RunID != c.RunID) {
					e = errors.New("fixture actor incarnation mismatch")
				}
				if e == nil && command.Kind == Exec && active == MaxConcurrentCLI {
					continue
				}
				if processed >= MaxRequests {
					return errors.New("fixture lifetime request limit reached")
				}
				processed++
				seen[id] = true
				if err := os.Remove(path); err != nil {
					return err
				}
				if e != nil {
					if err := atomicJSON(filepath.Join(c.ControlDir, "results", id+".json"), Result{ID: id, ExitCode: -1, Error: e.Error()}, true); err != nil {
						return err
					}
					continue
				}
				switch command.Kind {
				case Exec:
					active++
					workers.Add(1)
					go func(command Command) {
						defer workers.Done()
						r := runCLI(ctx, c.CouchBinary, command)
						select {
						case finished <- r:
						case <-ctx.Done():
						}
					}(command)
				case SetState:
					state.Phase = command.State
					if e = publishState(); e != nil {
						return e
					}
					if e = atomicJSON(filepath.Join(c.ControlDir, "results", id+".json"), Result{ID: id}, true); e != nil {
						return e
					}
				case Stop:
					if e = atomicJSON(filepath.Join(c.ControlDir, "results", id+".json"), Result{ID: id}, true); e != nil {
						return e
					}
					return nil
				}
			}
		}
	}
}

type boundedOutput struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (w *boundedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := MaxOutputBytes - w.buffer.Len()
	if len(p) > remaining {
		p = p[:remaining]
		w.exceeded = true
	}
	_, _ = w.buffer.Write(p)
	return n, nil
}
func (w *boundedOutput) String() string { return w.buffer.String() }
func runCLI(parent context.Context, executable string, c Command) Result {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	out, errout := &boundedOutput{}, &boundedOutput{}
	cmd := exec.CommandContext(ctx, executable, c.Args...)
	cmd.Env = os.Environ()
	cmd.Stdout, cmd.Stderr = out, errout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = time.Second
	var ownerMu sync.Mutex
	identity := ""
	cmd.Cancel = func() error {
		ownerMu.Lock()
		defer ownerMu.Unlock()
		if cmd.Process == nil || identity == "" || procutil.StrictIdentity(strconv.Itoa(cmd.Process.Pid)) != identity {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	ownerMu.Lock()
	err := cmd.Start()
	if err == nil {
		identity = procutil.StrictIdentity(strconv.Itoa(cmd.Process.Pid))
	}
	ownerMu.Unlock()
	if err == nil {
		err = cmd.Wait()
	}
	r := Result{ID: c.ID, Stdout: out.String(), Stderr: errout.String(), Truncated: out.exceeded || errout.exceeded}
	if err != nil {
		r.ExitCode = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			r.ExitCode = exit.ExitCode()
		}
		r.Error = err.Error()
	}
	return r
}
