//go:build darwin

package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

const managedLifetimeMarker = "managed-native-final"

type managedLifetimeOwner struct {
	PID   int
	Start string
}

// The native target and its detached descendant have independent barriers. No
// terminal output, wrapper timer or readiness callback releases either one.
func TestManagedLifetimeNativeHelper(t *testing.T) {
	role := os.Getenv("PAIR_MANAGED_NATIVE_ROLE")
	if role == "" {
		t.Skip("subprocess helper")
	}
	root := os.Getenv("PAIR_MANAGED_ROOT")
	owner := managedLifetimeOwner{os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid()))}
	if owner.Start == "" {
		t.Fatal("native process identity unavailable")
	}
	raw, err := json.Marshal(owner)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, role+".json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if role == "target" && os.Getenv("PAIR_MANAGED_DAEMON") == "1" {
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		child := exec.Command(exe, "-test.run=^TestManagedLifetimeNativeHelper$")
		child.Env = setEnv(os.Environ(), "PAIR_MANAGED_NATIVE_ROLE", "descendant")
		child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		// It deliberately outlives the original process and keeps its slave.
		// The parent test owns its release and checks that Run did not kill it.
	}
	if role == "target" && os.Getenv("PAIR_MANAGED_CLOSED") == "1" {
		if err := errors.Join(os.Stdin.Close(), os.Stdout.Close(), os.Stderr.Close()); err != nil {
			os.Exit(91)
		}
		if err := os.WriteFile(filepath.Join(root, "stdio.closed"), []byte("closed"), 0600); err != nil {
			os.Exit(92)
		}
	}
	gate, err := os.Open(filepath.Join(root, role+".gate"))
	if err != nil {
		t.Fatal(err)
	}
	var release [1]byte
	_, err = io.ReadFull(gate, release[:])
	closeErr := gate.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("native barrier: %v; close: %v", err, closeErr)
	}
	if role == "target" {
		if _, err := io.WriteString(os.Stdout, managedLifetimeMarker); err != nil {
			t.Fatal(err)
		}
		os.Exit(7)
	}
	os.Exit(0)
}

func TestManagedLifetimeRunHelper(t *testing.T) {
	if os.Getenv("PAIR_MANAGED_WRAPPER") != "1" {
		t.Skip("subprocess helper")
	}
	// TestMain deliberately clears inherited Couch identity. This subprocess
	// opts into only the private registry owned by this test.
	t.Setenv("COUCH_THREAD_SCOPE", "managed")
	t.Setenv("COUCH_THREAD_TAG", "managed")
	root := os.Getenv("PAIR_MANAGED_ROOT")
	paths, err := artifactpath.ResolveScoped(filepath.Join(root, "data"), "managed")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "codex")
	if os.Getenv("PAIR_MANAGED_MISSING") == "1" {
		target = filepath.Join(root, "missing", "codex")
	}
	if os.Getenv("PAIR_MANAGED_RESTART") == "1" {
		execProcess = func(_ string, argv, _ []string) error {
			raw, err := json.Marshal(argv)
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(root, "restart.json"), raw, 0600); err != nil {
				return err
			}
			return errors.New("managed fixture records replacement without executing it")
		}
	}
	os.Exit(Run([]string{"--scrollback-log", paths.ScrollbackRaw("codex"), "--", target}, os.Stdin, os.Stdout, os.Stderr))
}

func TestManagedLifetimeReadyAndNativeExit(t *testing.T) {
	for _, scenario := range []string{"normal", "inherited-slave", "missing", "closed-stop", "closed-restart"} {
		t.Run(scenario, func(t *testing.T) {
			root := isolateNotificationSockets(t)
			data := filepath.Join(root, "data")
			if err := os.MkdirAll(data, 0700); err != nil {
				t.Fatal(err)
			}
			primary := filepath.Join(root, "pair")
			if err := os.MkdirAll(primary, 0700); err != nil {
				t.Fatal(err)
			}
			paths, err := artifactpath.ResolveScoped(data, "managed")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			hellos := make(chan couchmessage.SessionHello, 4)
			nativeExits := make(chan couchmessage.Request, 4)
			namespace := filepath.Join(root, "messages")
			registryPath, err := couchmessage.SocketPath(namespace, "registry")
			if err != nil {
				t.Fatal(err)
			}
			registry, err := couchmessage.StartSessionServer(ctx, registryPath, couchmessage.SessionHandler{
				Open: func(_ couchmessage.SessionToken, hello couchmessage.SessionHello) error {
					hellos <- hello
					return nil
				},
				Frame: func(couchmessage.SessionToken, couchmessage.SessionFrame) {}, Closed: func(couchmessage.SessionToken) {},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer registry.Close()
			brokerPath, err := couchmessage.SocketPath(namespace, "broker")
			if err != nil {
				t.Fatal(err)
			}
			broker, err := couchmessage.StartServer(ctx, brokerPath, func(_ context.Context, raw []byte) ([]byte, error) {
				var request couchmessage.Request
				if err := json.Unmarshal(raw, &request); err != nil {
					return nil, err
				}
				if request.Op == "native-exit" {
					nativeExits <- request
				}
				return json.Marshal(couchmessage.Response{Code: "ok"})
			})
			if err != nil {
				t.Fatal(err)
			}
			defer broker.Close()
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			identity := map[string]any{"schema_version": 2, "repo": "pair", "repo_identity": filepath.Join(primary, ".git"), "primary_root": primary, "fleet_root": root, "environment_root": root, "worktree_root": primary, "kind": "primary", "address": "pair:0", "slot": 0, "branch": nil, "head": nil, "resting_branch": "main"}
			raw, err := json.Marshal(identity)
			if err != nil {
				t.Fatal(err)
			}
			managedLifetimeWrite(t, filepath.Join(root, "sdlc"), "#!/bin/sh\ncat <<'IDENTITY'\n"+string(raw)+"\nIDENTITY\n")
			managedLifetimeWrite(t, filepath.Join(root, "codex"), "#!/bin/sh\nif [ \"$1\" = --version ]; then printf 'managed-test-version\\n'; exit 0; fi\nexec '"+strings.ReplaceAll(exe, "'", "'\\''")+"' -test.run='^TestManagedLifetimeNativeHelper$'\n")
			gates := map[string]*os.File{}
			for _, role := range []string{"target", "descendant"} {
				name := filepath.Join(root, role+".gate")
				if err := syscall.Mkfifo(name, 0600); err != nil {
					t.Fatal(err)
				}
				gate, err := os.OpenFile(name, os.O_RDWR, 0600)
				if err != nil {
					t.Fatal(err)
				}
				gates[role] = gate
			}
			defer func() {
				for role, gate := range gates {
					_, _ = gate.Write([]byte{1})
					_ = gate.Close()
					raw, err := os.ReadFile(filepath.Join(root, role+".json"))
					var owner managedLifetimeOwner
					if err == nil && json.Unmarshal(raw, &owner) == nil && owner.Start != "" && procutil.Identity(strconv.Itoa(owner.PID)) == owner.Start {
						_ = syscall.Kill(owner.PID, syscall.SIGKILL)
					}
				}
			}()
			wrapper := exec.CommandContext(ctx, exe, "-test.run=^TestManagedLifetimeRunHelper$")
			wrapper.Dir = primary
			wrapper.Env = testpair.Environment(t, root,
				"PATH="+root+string(os.PathListSeparator)+os.Getenv("PATH"), "PAIR_MANAGED_WRAPPER=1", "PAIR_MANAGED_ROOT="+root,
				"PAIR_MANAGED_NATIVE_ROLE=target", "PAIR_DATA_DIR="+data, "PAIR_TAG=managed",
				"PAIR_SCOPE_KEY=managed", "PAIR_AGENT=codex", "PAIR_SCROLLBACK_EVENTS_PATH="+paths.ScrollbackEvents("codex"),
				"PAIR_SESSION_NAME=managed-session", "PAIR_LAUNCH_NONCE=managed-nonce", "PAIR_LAUNCH_ORDINAL=5",
				"COUCH_STORE_DIR="+namespace, "PAIR_NOTIFY_SOCKET_DIR="+root, "PAIR_WRAP_LOG="+filepath.Join(root, "wrap.log"))
			// The helpers use the same closed 0/1 switches as production env.
			wrapper.Env = setEnv(wrapper.Env, "PAIR_MANAGED_DAEMON", map[bool]string{true: "1", false: "0"}[scenario == "inherited-slave"])
			wrapper.Env = setEnv(wrapper.Env, "PAIR_MANAGED_MISSING", map[bool]string{true: "1", false: "0"}[scenario == "missing"])
			wrapper.Env = setEnv(wrapper.Env, "PAIR_MANAGED_CLOSED", map[bool]string{true: "1", false: "0"}[strings.HasPrefix(scenario, "closed-")])
			wrapper.Env = setEnv(wrapper.Env, "PAIR_MANAGED_RESTART", map[bool]string{true: "1", false: "0"}[scenario == "closed-restart"])
			var output, diagnostic bytes.Buffer
			wrapper.Stdout, wrapper.Stderr = &output, &diagnostic
			if err := wrapper.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			joined := make(chan struct{})
			go func() { defer close(joined); done <- wrapper.Wait() }()
			defer func() { cancel(); _ = wrapper.Process.Kill(); <-joined }()
			if scenario == "missing" {
				managedLifetimeWait(t, done, &diagnostic, 1)
				if _, err := os.Stat(paths.AgentReady("codex")); !os.IsNotExist(err) {
					t.Fatalf("missing target published readiness: %v", err)
				}
				if len(nativeExits) != 0 || len(hellos) != 0 {
					t.Fatal("missing target published a managed session or native exit")
				}
				return
			}
			var ready readiness.ReadyRecord
			managedLifetimePoll(t, done, func() bool {
				raw, err := os.ReadFile(paths.AgentReady("codex"))
				if err != nil {
					return false
				}
				ready, err = readiness.Decode(string(raw))
				return err == nil
			})
			var target managedLifetimeOwner
			managedLifetimePoll(t, done, func() bool {
				raw, err := os.ReadFile(filepath.Join(root, "target.json"))
				return err == nil && json.Unmarshal(raw, &target) == nil
			})
			if ready.PID != target.PID || ready.AgentIdentity != target.Start || ready.AgentIdentity != procutil.Identity(strconv.Itoa(target.PID)) || ready.LaunchOrdinal != 5 || ready.Tag != "managed" || ready.Session != "managed-session" || ready.Nonce != "managed-nonce" || ready.Agent != "codex" {
				t.Fatalf("readiness does not name the native target: ready=%+v target=%+v", ready, target)
			}
			var hello couchmessage.SessionHello
			select {
			case hello = <-hellos:
			case err := <-done:
				t.Fatalf("wrapper exited before registry hello: %v; %s", err, diagnostic.String())
			case <-testwait.After(t):
				t.Fatal("no managed registry hello")
			}
			if hello.Build == nil || hello.Binding.PID != wrapper.Process.Pid || hello.Binding.Scope != "managed" || hello.Binding.Tag != "managed" || hello.Binding.Agent != "codex" {
				t.Fatalf("managed hello lacks the wrapper binding/build: %+v", hello)
			}
			if strings.HasPrefix(scenario, "closed-") {
				managedLifetimePoll(t, done, func() bool {
					raw, err := os.ReadFile(filepath.Join(root, "stdio.closed"))
					return err == nil && string(raw) == "closed"
				})
				sig, status := syscall.SIGTERM, 255
				if scenario == "closed-restart" {
					sig, status = syscall.SIGUSR2, 1
				}
				if err := wrapper.Process.Signal(sig); err != nil {
					t.Fatal(err)
				}
				managedLifetimeWait(t, done, &diagnostic, status)
				if procutil.Identity(strconv.Itoa(target.PID)) == target.Start {
					t.Fatal("controlled stop/restart returned without reaping the original native target")
				}
				if len(nativeExits) != 0 {
					t.Fatal("controlled stop/restart reported a native self exit")
				}
				if scenario == "closed-restart" {
					raw, err := os.ReadFile(filepath.Join(root, "restart.json"))
					var argv []string
					if err != nil || json.Unmarshal(raw, &argv) != nil || len(argv) < 2 || argv[0] != exe || argv[1] != "wrap" || argv[len(argv)-1] != filepath.Join(root, "codex") {
						t.Fatalf("replacement did not preserve actual Run launch: argv=%q err=%v", argv, err)
					}
				}
				return
			}
			var descendant managedLifetimeOwner
			if scenario == "inherited-slave" {
				managedLifetimePoll(t, done, func() bool {
					raw, err := os.ReadFile(filepath.Join(root, "descendant.json"))
					return err == nil && json.Unmarshal(raw, &descendant) == nil
				})
			}
			if _, err := gates["target"].Write([]byte{1}); err != nil {
				t.Fatal(err)
			}
			managedLifetimeWait(t, done, &diagnostic, 7)
			capture, err := os.ReadFile(paths.ScrollbackRaw("codex"))
			if err != nil || strings.Count(string(capture), managedLifetimeMarker) != 1 || strings.Count(output.String(), managedLifetimeMarker) != 1 {
				t.Fatalf("native final output missing/duplicated: capture=%q output=%q err=%v diagnostics=%s", capture, output.String(), err, diagnostic.String())
			}
			if scenario == "inherited-slave" {
				rows, err := (procutil.Source{}).Table()
				if err != nil {
					t.Fatal(err)
				}
				live := false
				for _, row := range rows {
					if row.PID == descendant.PID && !row.Zombie && procutil.Identity(strconv.Itoa(row.PID)) == descendant.Start {
						live = true
					}
				}
				if !live {
					t.Fatal("wrapper stopped the detached descendant that retained its slave")
				}
			}
			if len(nativeExits) != 1 {
				t.Fatalf("native-exit count=%d, want exactly one", len(nativeExits))
			}
			request := <-nativeExits
			report, err := couchmessage.ParseNativeExit(request.Body)
			want := couchmessage.NativeExit{LaunchOrdinal: 5, AgentIdentity: target.Start, ExitCode: 7}
			if err != nil || couchmessage.ValidateRequest(request) != nil || report != want || request.ID != couchmessage.NativeExitID(hello.Binding, 5) || request.Scope != hello.Binding.Scope || request.Tag != hello.Binding.Tag || request.Nonce != hello.Binding.Nonce || request.Session != hello.Binding.Session {
				t.Fatalf("native exit does not match original launch: request=%+v report=%+v want=%+v err=%v", request, report, want, err)
			}
		})
	}
}

func managedLifetimeWrite(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
}

func managedLifetimePoll(t *testing.T, done <-chan error, observed func() bool) {
	t.Helper()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	deadline := testwait.After(t)
	for !observed() {
		select {
		case err := <-done:
			t.Fatalf("wrapper exited before native launch evidence: %v", err)
		case <-deadline:
			t.Fatal("native launch evidence not observed")
		case <-ticker.C:
		}
	}
}

func managedLifetimeWait(t *testing.T, done <-chan error, diagnostic *bytes.Buffer, want int) {
	t.Helper()
	select {
	case err := <-done:
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != want {
			t.Fatalf("Run status=%d want=%d; %s", code, want, diagnostic.String())
		}
	case <-testwait.After(t):
		t.Fatalf("Run did not finish after native target release (want status %d)", want)
	}
}
