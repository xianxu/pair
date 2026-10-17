package couchcore

import (
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
		if err = (osfs.FS{}).WriteAtomic(os.Getenv("PAIR_AGENT_READY_PATH"), string(raw)); err != nil {
			return err
		}
		if err = (osfs.FS{}).WriteAtomic(os.Getenv("PAIR_TEST_ZELLIJ_READY"), "ready"); err != nil {
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
		defer socket.Close()
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
		pane, _ := json.Marshal([]map[string]any{{"id": 1, "pane_command": strings.Join(child.Args, " ")}})
		if err = (osfs.FS{}).WriteAtomic(os.Getenv("PAIR_TEST_ZELLIJ_PANES"), string(pane)); err != nil {
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
		fmt.Fprintln(os.Stdout, "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND")
	case len(args) > 0 && args[0] == "list-sessions":
		if _, err := os.Stat(os.Getenv("PAIR_TEST_ZELLIJ_READY")); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("No active zellij sessions found.")
			}
			return err
		}
		fmt.Fprintln(os.Stdout, os.Getenv("PAIR_SESSION_NAME"))
	}
	return nil
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
