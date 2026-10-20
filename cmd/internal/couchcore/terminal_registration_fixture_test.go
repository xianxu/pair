package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"golang.org/x/term"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

// runRegistrationTerminalFixture models only the terminal process boundary:
// an exact server command, a real socket, and an owner-bearing live child.
// Production admission still reads exact identities and observes the live session.
func runRegistrationTerminalFixture() int {
	if err := registrationTerminalFixture(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func registrationTerminalFixture() error {
	args := os.Args[1:]
	if os.Getenv("PAIR_TEST_COUCH_REAL_WRAPPER") == "1" {
		root, controls := os.Getenv("COUCH_ISOLATED_ROOT"), os.Getenv("PAIR_TEST_COUCH_ACCEPTANCE_DIR")
		physical, e := filepath.EvalSymlinks(root)
		physicalControls, controlErr := filepath.EvalSymlinks(controls)
		physicalHome, homeErr := filepath.EvalSymlinks(os.Getenv("HOME"))
		relative, relErr := filepath.Rel(root, controls)
		if root == "" || !filepath.IsAbs(root) || e != nil || physical != root || controls == "" || controlErr != nil || physicalControls != controls || homeErr != nil || physicalHome != os.Getenv("HOME") || relErr != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("real-wrapper fixture requires disposable COUCH_ISOLATED_ROOT and subordinate control directory")
		}
		if !strings.HasPrefix(os.Getenv("HOME"), root+string(filepath.Separator)) {
			return errors.New("real-wrapper fixture HOME escapes isolation")
		}
		session := os.Getenv("PAIR_SESSION_NAME")
		for i, arg := range args {
			if (arg == "--session" || arg == "-s") && i+1 < len(args) {
				session = args[i+1]
			}
			if arg == "--server" && i+1 < len(args) {
				session = filepath.Base(args[i+1])
			}
		}
		if session == "" && filepath.Base(os.Args[0]) == "zellij" {
			session = "_query"
		}
		if session == "." || session == ".." || filepath.Base(session) != session {
			return errors.New("real-wrapper fixture invalid session path")
		}
		dir := filepath.Join(controls, session)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		for key, name := range map[string]string{"PAIR_TEST_ZELLIJ_READY": "ready", "PAIR_TEST_ZELLIJ_PANES": "panes", "PAIR_TEST_ZELLIJ_LOG": "zellij.log", "PAIR_WRAP_LOG": "wrapper.debug"} {
			_ = os.Setenv(key, filepath.Join(dir, name))
		}
		if filepath.Base(os.Args[0]) == "claude" {
			if len(args) > 0 && args[0] == "--version" {
				_ = (osfs.FS{}).WriteAtomic(filepath.Join(dir, "version-called"), "actual --version subprocess")
				fmt.Fprintln(os.Stdout, "2.1.0 (Claude Code)")
				return nil
			}
			return runRegistrationRealAgent(dir)
		}
	}
	var err error
	if filepath.Base(os.Args[0]) == "claude" {
		if len(args) == 1 && args[0] == "--fixture-agent" {
			if report := os.Getenv("PAIR_TEST_ZELLIJ_ENV_REPORT"); report != "" {
				raw, err := json.Marshal(os.Environ())
				if err != nil {
					return err
				}
				if err := os.WriteFile(report+".tmp", raw, 0600); err != nil {
					return err
				}
				if err := os.Rename(report+".tmp", report); err != nil {
					return err
				}
			}
			return awaitRegistrationFixtureRelease()
		}
		return nil
	}
	if filepath.Base(os.Args[0]) == "pair-wrap" {
		agent := exec.Command(filepath.Join(filepath.Dir(os.Args[0]), "claude"), "--fixture-agent")
		agent.Env, err = registrationFixtureEnvironment()
		if err != nil {
			return err
		}
		agent.Stdout, agent.Stderr = os.Stdout, os.Stderr
		if err := agent.Start(); err != nil {
			return err
		}
		defer func() { _ = agent.Process.Kill(); _ = agent.Wait() }()

		raw, err := readiness.Encode(readiness.ReadyRecord{Tag: os.Getenv("PAIR_TAG"), Agent: os.Getenv("PAIR_AGENT"), Session: os.Getenv("PAIR_SESSION_NAME"), Nonce: os.Getenv("PAIR_LAUNCH_NONCE"), PID: agent.Process.Pid, AgentIdentity: procutil.Identity(strconv.Itoa(agent.Process.Pid))})
		if err != nil {
			return err
		}
		if err = publishRegistrationFixtureReadiness(osfs.FS{}, os.Getenv("PAIR_TEST_ZELLIJ_READY"), os.Getenv("PAIR_AGENT_READY_PATH"), string(raw)); err != nil {
			return err
		}
		if err := awaitRegistrationFixtureRelease(); err != nil {
			return err
		}
		return agent.Wait()
	}
	if f, err := os.OpenFile(os.Getenv("PAIR_TEST_ZELLIJ_LOG"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
		fmt.Fprintln(f, strings.Join(args, " "))
		f.Close()
	}
	if len(args) == 2 && args[0] == "--server" {
		socket, err := net.Listen("unix", args[1])
		if err != nil {
			return err
		}
		acceptDone := make(chan struct{})
		go func() {
			defer close(acceptDone)
			for {
				conn, err := socket.Accept()
				if err != nil {
					return
				}
				_ = conn.Close()
			}
		}()
		defer func() { _ = socket.Close(); <-acceptDone }()
		state := registrationFixtureServer{Session: filepath.Base(args[1]), Socket: args[1], PID: os.Getpid(), Identity: procutil.Identity(strconv.Itoa(os.Getpid()))}
		raw, err := json.Marshal(state)
		if err != nil {
			return err
		}
		statePath := os.Getenv("PAIR_TEST_ZELLIJ_READY") + ".server"
		if err := (osfs.FS{}).WriteAtomic(statePath, string(raw)); err != nil {
			return err
		}
		defer os.Remove(statePath)
		child := exec.Command(filepath.Join(filepath.Dir(os.Args[0]), "pair-wrap"), "--scrollback-log", os.Getenv("PAIR_SCROLLBACK_RAW_PATH"))
		realWrapper := os.Getenv("PAIR_TEST_COUCH_REAL_WRAPPER") == "1"
		if realWrapper {
			child.Args = append(child.Args, filepath.Join(filepath.Dir(os.Args[0]), "claude"))
		}
		child.Env, err = registrationFixtureEnvironment()
		if err != nil {
			return err
		}
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if realWrapper {
			child.Stdin, child.Stdout, child.Stderr = nil, nil, nil
			raw, err := json.Marshal(child.Env)
			if err != nil {
				return err
			}
			if err := (osfs.FS{}).WriteAtomic(filepath.Join(filepath.Dir(os.Getenv("PAIR_TEST_ZELLIJ_READY")), "wrapper-environment.json"), string(raw)); err != nil {
				return err
			}
			terminal, err := pty.StartWithSize(child, &pty.Winsize{Rows: 24, Cols: 80})
			if err != nil {
				return err
			}
			defer terminal.Close()
			go func() {
				buffer := make([]byte, 8192)
				for {
					n, err := terminal.Read(buffer)
					if n > 0 {
						_, _ = os.Stdout.Write(buffer[:n])
					}
					if err != nil {
						return
					}
				}
			}()
			if err := (osfs.FS{}).WriteAtomic(os.Getenv("PAIR_TEST_ZELLIJ_READY"), "ready"); err != nil {
				return err
			}
		} else if err = child.Start(); err != nil {
			return err
		}
		if realWrapper {
			raw, _ := json.Marshal(struct {
				PID      int
				Identity string
			}{child.Process.Pid, procutil.Identity(strconv.Itoa(child.Process.Pid))})
			if err := (osfs.FS{}).WriteAtomic(filepath.Join(filepath.Dir(os.Getenv("PAIR_TEST_ZELLIJ_READY")), "owned-wrapper.json"), string(raw)); err != nil {
				return err
			}
		}
		defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
		pane, renderErr := registrationFixtureReply("panes", []map[string]any{{"id": 1, "pane_command": strings.Join(child.Args, " ")}})
		if renderErr != nil {
			return renderErr
		}
		if err = (osfs.FS{}).WriteAtomic(os.Getenv("PAIR_TEST_ZELLIJ_PANES"), pane.Stdout); err != nil {
			return err
		}
		if err := awaitRegistrationFixtureRelease(); err != nil {
			return err
		}
		if realWrapper {
			// The disposable server owns this wrapper. Release is fixture teardown,
			// independent of the production detach path, which leaves it running.
			_ = child.Process.Signal(os.Interrupt)
			deadline := time.AfterFunc(2*time.Second, func() { _ = child.Process.Kill() })
			defer deadline.Stop()
		}
		return child.Wait()
	}
	joined := " " + strings.Join(args, " ") + " "
	switch {
	case strings.Contains(joined, " --new-session-with-layout "):
		socket := filepath.Join(os.Getenv("ZELLIJ_SOCKET_DIR"), "contract_version_1", os.Getenv("PAIR_SESSION_NAME"))
		if request := os.Getenv("PAIR_TEST_ZELLIJ_SERVER_REQUEST"); request != "" {
			if err := (osfs.FS{}).WriteAtomic(request, socket); err != nil {
				return err
			}
			return awaitRegistrationFixtureFile(os.Getenv("PAIR_TEST_ZELLIJ_CLIENT_RELEASE"))
		}

		server := exec.Command(os.Args[0], "--server", socket)
		server.Env, err = registrationFixtureEnvironment()
		if err != nil {
			return err
		}
		server.Stdout, server.Stderr = os.Stdout, os.Stderr
		if os.Getenv("PAIR_TEST_COUCH_REAL_WRAPPER") == "1" {
			server.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		}
		return server.Run()
	case strings.Contains(joined, " attach "):
		if err := (osfs.FS{}).WriteAtomic(os.Getenv("PAIR_TEST_ZELLIJ_ATTACH_READY"), "attach"); err != nil {
			return err
		}
		return awaitRegistrationFixtureFile(os.Getenv("PAIR_TEST_ZELLIJ_ATTACH_RELEASE"))
	case strings.Contains(joined, " list-panes "):
		raw, err := os.ReadFile(os.Getenv("PAIR_TEST_ZELLIJ_PANES"))
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(raw)
		return err
	case strings.Contains(joined, " list-clients "):
		reply, err := registrationFixtureCatalog("couch.registration.clients")
		if err != nil {
			return err
		}
		return registrationFixtureEmit(reply)
	case len(args) > 0 && args[0] == "list-sessions":
		if os.Getenv("PAIR_TEST_COUCH_REAL_WRAPPER") == "1" {
			entries, err := os.ReadDir(os.Getenv("PAIR_TEST_COUCH_ACCEPTANCE_DIR"))
			if err != nil {
				return err
			}
			var sessions []string
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				session, err := registrationFixtureLiveSession(filepath.Join(os.Getenv("PAIR_TEST_COUCH_ACCEPTANCE_DIR"), entry.Name(), "ready"))
				if err != nil {
					return err
				}
				if session != "" {
					sessions = append(sessions, session)
				}
			}
			if len(sessions) == 0 {
				reply, err := registrationFixtureCatalog("couch.registration.no-sessions")
				if err != nil {
					return err
				}
				return registrationFixtureEmit(reply)
			}
			kind := "sessions-full"
			if len(args) == 2 && args[1] == "--short" {
				kind = "sessions-short"
			}
			reply, err := registrationFixtureReply(kind, sessions)
			if err != nil {
				return err
			}
			return registrationFixtureEmit(reply)
		}
		session, err := registrationFixtureLiveSession(os.Getenv("PAIR_TEST_ZELLIJ_READY"))
		if err != nil {
			return err
		}
		if session == "" {
			reply, err := registrationFixtureCatalog("couch.registration.no-sessions")
			if err != nil {
				return err
			}
			return registrationFixtureEmit(reply)
		}
		if len(args) == 2 && args[1] == "--short" {
			reply, err := registrationFixtureReply("sessions-short", []string{session})
			if err != nil {
				return err
			}
			return registrationFixtureEmit(reply)
		} else if len(args) == 2 && args[1] == "--no-formatting" {
			reply, err := registrationFixtureReply("sessions-full", []string{session})
			if err != nil {
				return err
			}
			return registrationFixtureEmit(reply)
		} else {
			return fmt.Errorf("unsupported fixture session listing: %q", args)
		}
	}
	return nil
}

func publishRegistrationFixtureReadiness(fs interface{ WriteAtomic(string, string) error }, listingPath, agentPath, record string) error {
	// Agent readiness permits production admission to query the listing
	// immediately. Publish its prerequisite before announcing readiness (#698).
	if err := fs.WriteAtomic(listingPath, "ready"); err != nil {
		return err
	}
	return fs.WriteAtomic(agentPath, record)
}

// Nested fixture processes run outside testing.M and may have a disposable
// working directory. Carry only the named fixture snapshot through the same
// canonical empty-environment owner as their parent test.
func registrationFixtureEnvironment() ([]string, error) {
	var explicit []string
	for _, key := range []string{
		"PAIR_TEST_COUCH_TERMINAL_FIXTURE", "PAIR_TEST_ZELLIJ_READY",
		"PAIR_TEST_COUCH_REAL_WRAPPER", "PAIR_TEST_COUCH_ACCEPTANCE_DIR",
		"COUCH_STORE_DIR", "COUCH_IDENTITY_DIR", "COUCH_ISOLATED_ROOT", "COUCH_PAIR_DATA_DIR", "COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG",
		"PAIR_HOME", "PATH", "TERM", "PAIR_WRAP_LOG",
		"PAIR_TEST_ZELLIJ_RELEASE", "PAIR_TEST_ZELLIJ_LOG", "PAIR_TEST_ZELLIJ_PANES",
		"PAIR_TEST_ZELLIJ_SERVER_REQUEST", "PAIR_TEST_ZELLIJ_CLIENT_RELEASE",
		"PAIR_TEST_ZELLIJ_ATTACH_READY", "PAIR_TEST_ZELLIJ_ATTACH_RELEASE",
		"PAIR_TEST_ZELLIJ_ENV_REPORT", "ZELLIJ_SOCKET_DIR",
		"PAIR_TAG", "PAIR_AGENT", "PAIR_SESSION_NAME", "PAIR_LAUNCH_NONCE",
		"PAIR_AGENT_READY_PATH", "PAIR_SCROLLBACK_RAW_PATH", "PAIR_DATA_DIR", "PAIR_SCOPE_KEY",
	} {
		if value, ok := os.LookupEnv(key); ok {
			explicit = append(explicit, key+"="+value)
		}
	}
	return testpair.EnvironmentFrom(os.Getenv("PAIR_TEST_ENV_SCRIPT"), os.Getenv("HOME"), explicit...)
}

func awaitRegistrationFixtureRelease() error {
	return awaitRegistrationFixtureFile(os.Getenv("PAIR_TEST_ZELLIJ_RELEASE"))
}

func awaitRegistrationFixtureFile(path string) error {
	budget := 30 * time.Second
	if os.Getenv("PAIR_TEST_COUCH_REAL_WRAPPER") == "1" {
		budget = 95 * time.Second
	}
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("registration terminal fixture was not released")
}

// Dynamic process identities cannot be pre-rendered by the parent. Resolve only
// the explicitly passed canonical test helper's sibling generator; all query
// output still comes from that single fixture serializer.
func registrationFixtureReply(kind string, payload any) (zellijtest.ReplyEnvelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return zellijtest.ReplyEnvelope{}, err
	}
	return registrationFixtureRender("--kind", kind, "--data", string(raw))
}
func registrationFixtureCatalog(name string) (zellijtest.ReplyEnvelope, error) {
	return registrationFixtureRender("--case", name)
}
func registrationFixtureRender(args ...string) (zellijtest.ReplyEnvelope, error) {
	command, err := zellijtest.ReplyCommandFrom(os.Getenv("PAIR_TEST_ENV_SCRIPT"))
	if err != nil {
		return zellijtest.ReplyEnvelope{}, err
	}
	// Arguments remain positional data, never shell interpolation.
	cmd := exec.Command("/bin/sh", append([]string{"-c", "exec " + command + " --envelope \"$@\"", "fixture"}, args...)...)
	raw, err := cmd.Output()
	if err != nil {
		return zellijtest.ReplyEnvelope{}, fmt.Errorf("render dynamic fixture: %w", err)
	}
	var reply zellijtest.ReplyEnvelope
	if err := json.Unmarshal(raw, &reply); err != nil {
		return reply, err
	}
	return reply, nil
}
func registrationFixtureEmit(reply zellijtest.ReplyEnvelope) error {
	if _, err := fmt.Fprint(os.Stdout, reply.Stdout); err != nil {
		return err
	}
	if reply.Code != 0 {
		return fmt.Errorf("%s", strings.TrimSuffix(reply.Stderr, "\n"))
	}
	_, err := fmt.Fprint(os.Stderr, reply.Stderr)
	return err
}

// Server state is published by the owned server before its wrapper can announce
// readiness. The old readiness marker alone never grants session lifetime.
type registrationFixtureServer struct {
	Session  string `json:"session"`
	Socket   string `json:"socket"`
	PID      int    `json:"pid"`
	Identity string `json:"identity"`
}

func registrationFixtureLiveSession(ready string) (string, error) {
	if _, err := os.Stat(ready); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	raw, err := os.ReadFile(ready + ".server")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var server registrationFixtureServer
	if err := json.Unmarshal(raw, &server); err != nil {
		return "", fmt.Errorf("invalid fixture server state: %w", err)
	}
	if server.Session == "" || server.Socket == "" || filepath.Base(server.Socket) != server.Session || server.PID <= 0 || server.Identity == "" {
		return "", fmt.Errorf("incomplete fixture server state: %s", raw)
	}
	pid := strconv.Itoa(server.PID)
	if !procutil.Alive(pid) {
		return "", nil
	}
	identity := procutil.Identity(pid)
	if identity == "" {
		return "", fmt.Errorf("fixture server identity unavailable for %s", pid)
	}
	if identity != server.Identity {
		return "", nil
	}
	conn, err := net.DialTimeout("unix", server.Socket, time.Second)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED) {
			return "", nil
		}
		return "", fmt.Errorf("fixture server endpoint: %w", err)
	}
	_ = conn.Close()
	return server.Session, nil
}

// runRegistrationRealAgent exercises the real wrapper's composer and public
// subprocess caller environment. No identity or message endpoint is fabricated.
func runRegistrationRealAgent(dir string) error {
	owned, _ := json.Marshal(struct {
		PID      int
		Identity string
	}{os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid()))})
	if err := (osfs.FS{}).WriteAtomic(filepath.Join(dir, "owned-agent.json"), string(owned)); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := (osfs.FS{}).WriteAtomic(filepath.Join(dir, "cwd"), cwd); err != nil {
		return err
	}
	raw, err := json.Marshal(os.Environ())
	if err != nil {
		return err
	}
	if err := (osfs.FS{}).WriteAtomic(filepath.Join(dir, "environment.json"), string(raw)); err != nil {
		return err
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	paint := func(body string) {
		if body != "" {
			body = "received message"
		}
		rule := "\x1b[38;2;136;136;136m" + strings.Repeat("─", 40) + "\x1b[39m"
		fmt.Fprint(os.Stdout, "\x1b[2J\x1b[?2004h\x1b[20;1H"+rule+"\x1b[21;1H❯ "+body+"\x1b[22;1H"+rule+"\x1b[?25h\x1b[21;3H")
	}
	input := make(chan []byte)
	go func() {
		defer close(input)
		b := make([]byte, 8192)
		for {
			n, e := os.Stdin.Read(b)
			if n > 0 {
				input <- append([]byte(nil), b[:n]...)
			}
			if e != nil {
				return
			}
		}
	}()
	paint("")
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var pending []byte
	receivedNumber := 0
	for {
		select {
		case b, ok := <-input:
			if !ok {
				return nil
			}
			pending = append(pending, b...)
			if i := bytes.Index(pending, []byte("\x1b[201~")); i >= 0 {
				body := bytes.TrimPrefix(pending[:i], []byte("\x1b[200~"))
				paint(string(body))
				pending = pending[i+6:]
				receivedNumber++
				if err := (osfs.FS{}).WriteAtomic(filepath.Join(dir, fmt.Sprintf("received-%03d", receivedNumber)), string(body)); err != nil {
					return err
				}
				if err := (osfs.FS{}).WriteAtomic(filepath.Join(dir, "received"), string(body)); err != nil {
					return err
				}
			}
			if bytes.Contains(pending, []byte("\x1b\r")) || bytes.Contains(pending, []byte("\r")) {
				pending = nil
				paint("")
			}
		case <-ticker.C:
			if _, err := os.Stat(os.Getenv("PAIR_TEST_ZELLIJ_RELEASE")); err == nil {
				return nil
			}
			path := filepath.Join(dir, "command.json")
			raw, err := os.ReadFile(path)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			_ = os.Remove(path)
			var argv []string
			if err := json.Unmarshal(raw, &argv); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			cmd := exec.CommandContext(ctx, filepath.Join(filepath.Dir(os.Args[0]), "couch"), argv...)
			cmd.Env = os.Environ()
			out, runErr := cmd.CombinedOutput()
			cancel()
			result := struct {
				Output string
				Error  string
			}{Output: string(out)}
			if runErr != nil {
				result.Error = runErr.Error()
			}
			data, err := json.Marshal(result)
			if err != nil {
				return err
			}
			if err := (osfs.FS{}).WriteAtomic(filepath.Join(dir, "result.json"), string(data)); err != nil {
				return err
			}
		}
	}
}

func TestRealWrapperFixtureRequiresPhysicalDisposableRoots(t *testing.T) {
	for _, kind := range []string{"missing-root", "external-controls", "symlink-controls", "symlink-home", "session-traversal"} {
		t.Run(kind, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			controls, home := filepath.Join(root, "controls"), filepath.Join(root, "home")
			if err := os.MkdirAll(controls, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(home, 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PAIR_TEST_COUCH_REAL_WRAPPER", "1")
			t.Setenv("COUCH_ISOLATED_ROOT", root)
			t.Setenv("PAIR_TEST_COUCH_ACCEPTANCE_DIR", controls)
			t.Setenv("HOME", home)
			t.Setenv("PAIR_SESSION_NAME", "safe-session")
			switch kind {
			case "missing-root":
				t.Setenv("COUCH_ISOLATED_ROOT", "")
			case "external-controls":
				t.Setenv("PAIR_TEST_COUCH_ACCEPTANCE_DIR", t.TempDir())
			case "symlink-controls":
				link := filepath.Join(root, "control-link")
				if err := os.Symlink(controls, link); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PAIR_TEST_COUCH_ACCEPTANCE_DIR", link)
			case "symlink-home":
				link := filepath.Join(root, "home-link")
				if err := os.Symlink(home, link); err != nil {
					t.Fatal(err)
				}
				t.Setenv("HOME", link)
			case "session-traversal":
				t.Setenv("PAIR_SESSION_NAME", "../outside")
			}
			if err := registrationTerminalFixture(); err == nil {
				t.Fatal("unsafe real-wrapper fixture mode admitted")
			}
		})
	}
}
