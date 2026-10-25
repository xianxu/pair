package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
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
		// Model the real layout's first-pane command before entering pair wrap.
		// AgentReady alone cannot release the creator's namespace lease (#866).
		paneEnv := append(os.Environ(), "ZELLIJ_SESSION_NAME="+os.Getenv("PAIR_SESSION_NAME"), "ZELLIJ_PANE_ID=1")
		if err := launcher.PublishPaneReady(context.Background(), paneEnv); err != nil {
			return err
		}
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
		child.Env, err = registrationFixtureEnvironment()
		if err != nil {
			return err
		}
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err = child.Start(); err != nil {
			return err
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
	deadline := time.Now().Add(30 * time.Second)
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
