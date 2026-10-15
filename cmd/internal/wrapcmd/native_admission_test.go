package wrapcmd

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
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

func TestNativeResumeWithoutAdmissionNeverSpawns(t *testing.T) {
	for _, fromEnvironment := range []bool{false, true} {
		name := "argv"
		if fromEnvironment {
			name = "from-launch-env"
		}
		t.Run(name, func(t *testing.T) {
			isolateNotificationSockets(t)
			t.Setenv("PAIR_RETENTION_START_ID", "")
			t.Setenv("PAIR_DATA_DIR", t.TempDir())
			t.Setenv("PAIR_SCOPE_KEY", "scope")
			t.Setenv("PAIR_TAG", "admission-test")
			t.Setenv("PAIR_SESSION_NAME", "pair-admission-test")
			t.Setenv("PAIR_LAUNCH_ORDINAL", "1")
			t.Setenv("PAIR_LAUNCH_NONCE", "launch-attempt")
			t.Setenv("PAIR_NATIVE_ADMISSION", "")
			marker := filepath.Join(t.TempDir(), "spawned")
			executable := filepath.Join(t.TempDir(), "codex")
			if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf spawned > '"+marker+"'\n"), 0700); err != nil {
				t.Fatal(err)
			}
			argv := []string{executable, "resume", "019f0102-0304-7506-8708-090a0b0c0d0e"}
			if fromEnvironment {
				command, err := launcher.EncodeAgentCommand(launcher.AgentCommand{Executable: executable, Argv: argv[1:]})
				if err != nil {
					t.Fatal(err)
				}
				t.Setenv(launcher.AgentCommandEnv, command)
				argv = []string{"--from-launch-env"}
			}
			var stderr bytes.Buffer
			code := Run(argv, bytes.NewReader(nil), &bytes.Buffer{}, &stderr)
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("unadmitted native resume spawned: code=%d stderr=%s markererr=%v", code, &stderr, err)
			}
			if code == 0 {
				t.Fatalf("unadmitted native resume succeeded: %s", &stderr)
			}
		})
	}
}

type nativeAdmissionWorld struct {
	state, agent, digest string
	events               []string
	childAlive           bool
	publishErr           error
}

func (w *nativeAdmissionWorld) Claim(_ context.Context, grant, agent, digest string) (launcher.NativeAdmissionClaim, error) {
	w.events = append(w.events, "claim")
	if grant != "valid-grant" || w.state != "reserved" || agent != w.agent || digest != w.digest {
		return launcher.NativeAdmissionClaim{}, errors.New("admission does not match or was consumed")
	}
	w.state = "spawning"
	return launcher.NativeAdmissionClaim{}, nil
}

func (w *nativeAdmissionWorld) PublishChild(_ context.Context, _ launcher.NativeAdmissionClaim, pid int) error {
	w.events = append(w.events, "publish")
	if w.publishErr != nil {
		return w.publishErr
	}
	if w.state != "spawning" || pid != 123 || !w.childAlive {
		return errors.New("child generation unavailable")
	}
	w.state = "running"
	return nil
}

func nativeAdmissionTestCommand(t *testing.T) (*exec.Cmd, *nativeAdmissionWorld) {
	t.Helper()
	cmd := exec.Command("/native/codex", "resume", "019f0102-0304-7506-8708-090a0b0c0d0e", "--", "positional prompt")
	digest, err := launcher.NativeAdmissionCommandDigest("codex", cmd.Args)
	if err != nil {
		t.Fatal(err)
	}
	return cmd, &nativeAdmissionWorld{state: "reserved", agent: "codex", digest: digest}
}

func TestNativeAdmissionConsumptionAndChildPublicationEncloseSpawn(t *testing.T) {
	cmd, world := nativeAdmissionTestCommand(t)
	spawn := func(command *exec.Cmd) (*os.File, error) {
		world.events = append(world.events, "spawn")
		if world.state != "spawning" {
			t.Fatal("native spawn preceded durable consumption")
		}
		world.childAlive = true
		command.Process = &os.Process{Pid: 123}
		return nil, nil
	}
	if _, err := startNativePTY(context.Background(), world, "valid-grant", "codex", cmd, io.Discard, spawn); err != nil {
		t.Fatal(err)
	}
	// Returning success is the caller's permission to configure/publish ready.
	world.events = append(world.events, "ready")
	if !reflect.DeepEqual(world.events, []string{"claim", "spawn", "publish", "ready"}) || world.state != "running" {
		t.Fatalf("native ownership/readiness order: %v state=%s", world.events, world.state)
	}
	before := len(world.events)
	if _, err := startNativePTY(context.Background(), world, "valid-grant", "codex", cmd, io.Discard, spawn); err == nil {
		t.Fatal("duplicate grant spawned another native writer")
	}
	if !reflect.DeepEqual(world.events[before:], []string{"claim"}) || !world.childAlive || world.state != "running" {
		t.Fatalf("replayed grant changed existing native owner: %+v", world)
	}
}

func TestNativeAdmissionFailureDoesNotInventReleaseOrReadiness(t *testing.T) {
	for _, failure := range []string{"claim", "spawn", "publish", "missing-child"} {
		t.Run(failure, func(t *testing.T) {
			cmd, world := nativeAdmissionTestCommand(t)
			grant := "valid-grant"
			if failure == "claim" {
				grant = "forged"
			}
			if failure == "publish" {
				world.publishErr = errors.New("publication outcome uncertain")
			}
			spawns := 0
			_, err := startNativePTY(context.Background(), world, grant, "codex", cmd, io.Discard, func(command *exec.Cmd) (*os.File, error) {
				spawns++
				world.events = append(world.events, "spawn")
				if failure == "spawn" {
					return nil, errors.New("spawn outcome unavailable")
				}
				world.childAlive = true
				if failure != "missing-child" {
					command.Process = &os.Process{Pid: 123}
				}
				return nil, nil
			})
			if err == nil {
				t.Fatal("failed admission/spawn/publication permitted readiness")
			}
			if failure == "claim" {
				if spawns != 0 || world.state != "reserved" {
					t.Fatalf("invalid grant affected native launch: %+v", world)
				}
			} else if spawns != 1 || world.state != "spawning" {
				t.Fatalf("uncertain launch lost ownership: %+v", world)
			}
			if failure == "publish" && !world.childAlive {
				t.Fatal("publication error pretended the child did not exist")
			}
		})
	}
}

func TestNativeAdmissionSelectorsAndCommandMustMatchTheGrant(t *testing.T) {
	for _, tc := range []struct {
		name, agent, grant string
		args               []string
		allowed            bool
	}{
		{name: "missing", agent: "codex", args: []string{"resume", "019f0102-0304-7506-8708-090a0b0c0d0e"}},
		{name: "latest-unpinned", agent: "codex", args: []string{"resume", "--last"}, grant: "valid-grant"},
		{name: "picker-unpinned", agent: "codex", args: []string{"resume"}, grant: "valid-grant"},
		{name: "changed-id", agent: "codex", args: []string{"resume", "other-id"}, grant: "valid-grant"},
		{name: "changed-prompt", agent: "codex", args: []string{"resume", "019f0102-0304-7506-8708-090a0b0c0d0e", "--", "changed"}, grant: "valid-grant"},
		{name: "stale-on-fresh", agent: "codex", args: []string{"--model", "m"}, grant: "valid-grant"},
		{name: "codex-name", agent: "codex", args: []string{"resume", "saved-name"}, allowed: true},
		{name: "codex-prefix", agent: "codex", args: []string{"resume", "019f0102"}, allowed: true},
		{name: "codex-latest", agent: "codex", args: []string{"resume", "--last"}, allowed: true},
		{name: "codex-picker", agent: "codex", args: []string{"resume"}, allowed: true},
		{name: "codex-name-invalid-grant", agent: "codex", args: []string{"resume", "saved-name"}, grant: "forged"},
		{name: "codex-name-mismatched-grant", agent: "codex", args: []string{"resume", "saved-name"}, grant: "valid-grant"},
		{name: "codex-latest-invalid-grant", agent: "codex", args: []string{"resume", "--last"}, grant: "forged"},
		{name: "muse-missing", agent: "muse", args: []string{"resume", "native-id"}},
		{name: "agy-missing", agent: "agy", args: []string{"--resume", "native-id"}},
		{name: "fresh", agent: "codex", args: []string{"--model", "m"}, allowed: true},
		{name: "fork", agent: "codex", args: []string{"fork", "native-id"}, allowed: true},
		{name: "shell", agent: "sh", args: []string{"-c", "printf resume"}, allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, world := nativeAdmissionTestCommand(t)
			cmd := exec.Command("/native/"+tc.agent, tc.args...)
			spawns := 0
			_, err := startNativePTY(context.Background(), world, tc.grant, tc.agent, cmd, io.Discard, func(*exec.Cmd) (*os.File, error) {
				spawns++
				return nil, nil
			})
			if tc.allowed {
				if err != nil || spawns != 1 || len(world.events) != 0 {
					t.Fatalf("ordinary Fresh/fork/shell acquired native admission: err=%v spawns=%d events=%v", err, spawns, world.events)
				}
			} else if err == nil || spawns != 0 {
				t.Fatalf("mismatched native admission spawned: err=%v spawns=%d", err, spawns)
			}
		})
	}
}

func TestNativeAdmissionCapabilityIsNotInheritedOrCarriedIntoFreshExec(t *testing.T) {
	t.Setenv("PAIR_NATIVE_ADMISSION", "old-capability")
	grant, err := consumeNativeAdmissionEnvironment(os.Getenv, os.Unsetenv)
	if err != nil || grant != "old-capability" {
		t.Fatalf("consume capability: %q %v", grant, err)
	}
	if _, present := os.LookupEnv("PAIR_NATIVE_ADMISSION"); present {
		t.Fatal("wrapper helpers and watcher can still inherit the capability")
	}
	request, err := freshAgentInvocation("/pair", "", []string{"codex", "resume", "019f0102-0304-7506-8708-090a0b0c0d0e"}, []string{"PAIR_NATIVE_ADMISSION=stale", "PAIR_NATIVE_ADMISSION=duplicate", "UNRELATED=preserved"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range request.env {
		if strings.HasPrefix(item, "PAIR_NATIVE_ADMISSION=") {
			t.Fatal("same-PID Fresh exec retained native admission capability")
		}
	}
	if envValue(request.env, "UNRELATED") != "preserved" {
		t.Fatal("Fresh environment scrub dropped unrelated state")
	}
}

func TestNativeAdmissionRealWrapperPublishesBeforeWatcherAndScrubsCapability(t *testing.T) {
	for _, fromEnvironment := range []bool{false, true} {
		t.Run(fmt.Sprint("from-launch-env=", fromEnvironment), func(t *testing.T) {
			isolateNotificationSockets(t)
			home, data := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("PAIR_RETENTION_START_ID", "")
			t.Setenv("PAIR_DATA_DIR", data)
			t.Setenv("PAIR_SCOPE_KEY", "scope")
			t.Setenv("PAIR_TAG", "admission-live")
			t.Setenv("PAIR_SESSION_NAME", "pair-admission-live")
			t.Setenv("PAIR_LAUNCH_NONCE", "launch-attempt")
			t.Setenv("PAIR_LAUNCH_ORDINAL", "1")
			t.Setenv("PAIR_WRAP_REMAP_RETURN", "0")
			stop := filepath.Join(t.TempDir(), "stop")
			if err := syscall.Mkfifo(stop, 0600); err != nil {
				t.Fatal(err)
			}
			childEnvPath := filepath.Join(t.TempDir(), "native-env")
			t.Setenv("NATIVE_ENV", childEnvPath)
			t.Setenv("STOP_PIPE", stop)
			executable := filepath.Join(t.TempDir(), "codex")
			if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf '%s' \"${PAIR_NATIVE_ADMISSION-unset}\" > \"$NATIVE_ENV\"\nread line < \"$STOP_PIPE\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			argv := []string{executable, "resume", "019f0102-0304-7506-8708-090a0b0c0d0e"}
			digest, err := launcher.NativeAdmissionCommandDigest("codex", argv)
			if err != nil {
				t.Fatal(err)
			}
			store := launcher.NativeAdmissionStore{Home: home}
			grant, err := store.Reserve(t.Context(), launcher.NativeAdmissionRequest{Agent: "codex", NativeID: "019f0102-0304-7506-8708-090a0b0c0d0e", Attempt: "wrapper-test", CommandDigest: digest}, func(context.Context, launcher.NativeAdmissionOwners) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv(launcher.NativeAdmissionEnv, grant)
			if fromEnvironment {
				command, err := launcher.EncodeAgentCommand(launcher.AgentCommand{Executable: executable, Argv: argv[1:]})
				if err != nil {
					t.Fatal(err)
				}
				t.Setenv(launcher.AgentCommandEnv, command)
				argv = []string{"--from-launch-env"}
			}
			oldWatcher := startWatcherProcess
			defer func() { startWatcherProcess = oldWatcher }()
			watchers := 0
			var watcherErr error
			startWatcherProcess = func(_ []string, env []string) (observedErr error) {
				defer func() { watcherErr = observedErr }()
				watchers++
				defer func() {
					if err := os.WriteFile(stop, []byte("stop\n"), 0600); err != nil {
						t.Error(err)
					}
				}()
				for _, item := range env {
					if strings.HasPrefix(item, launcher.NativeAdmissionEnv+"=") {
						t.Error("watcher inherited admission capability")
					}
				}
				cache, err := artifactpath.ResolvePairCache(home)
				if err != nil {
					return err
				}
				path, err := cache.NativeAdmissions()
				if err != nil {
					return err
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				var admission struct {
					Entries []launcher.NativeAdmissionEntry
				}
				if err := json.Unmarshal(raw, &admission); err != nil {
					return err
				}
				if len(admission.Entries) != 1 || admission.Entries[0].Phase != launcher.NativeAdmissionRunning {
					return fmt.Errorf("watcher started before child ownership publication: %s", raw)
				}
				readyRaw, err := os.ReadFile(filepath.Join(data, "agent-ready-admission-live-codex.json"))
				if err != nil {
					return err
				}
				ready, err := readiness.Decode(string(readyRaw))
				if err != nil || ready.PID != admission.Entries[0].Child.PID || ready.AgentIdentity != procutil.Identity(strconv.Itoa(ready.PID)) || admission.Entries[0].Child.Birth != procutil.StrictIdentity(strconv.Itoa(ready.PID)) {
					return fmt.Errorf("readiness differs from admitted child: ready=%+v entry=%+v err=%v", ready, admission.Entries[0], err)
				}
				return nil
			}
			var stderr bytes.Buffer
			if code := Run(argv, bytes.NewReader(nil), &bytes.Buffer{}, &stderr); code != 0 || watchers != 1 || watcherErr != nil {
				t.Fatalf("admitted native wrapper: code=%d watchers=%d watchererr=%v stderr=%s", code, watchers, watcherErr, &stderr)
			}
			if raw, err := os.ReadFile(childEnvPath); err != nil || string(raw) != "unset" {
				t.Fatalf("native child inherited grant: %q %v", raw, err)
			}
			if _, present := os.LookupEnv(launcher.NativeAdmissionEnv); present {
				t.Fatal("wrapper retained capability after handoff")
			}
			t.Setenv(launcher.NativeAdmissionEnv, grant)
			stderr.Reset()
			if code := Run(argv, bytes.NewReader(nil), &bytes.Buffer{}, &stderr); code == 0 || watchers != 1 {
				t.Fatalf("replayed grant spawned another native writer: code=%d watchers=%d stderr=%s", code, watchers, &stderr)
			}
		})
	}
}

func TestCodexIndirectCompatibilityPreservesActualWrapperCommand(t *testing.T) {
	for _, args := range [][]string{{"resume", "--last"}, {"resume"}, {"resume", "saved-name", "positional prompt"}, {"resume", "019f0102"}} {
		for _, fromEnvironment := range []bool{false, true} {
			t.Run(fmt.Sprint(args, "/from-launch-env=", fromEnvironment), func(t *testing.T) {
				isolateNotificationSockets(t)
				t.Setenv("PAIR_RETENTION_START_ID", "")
				t.Setenv("PAIR_TAG", "compatibility-test")
				t.Setenv("PAIR_SCOPE_KEY", "scope")
				t.Setenv("PAIR_SESSION_NAME", "pair-compatibility-test")
				oldWatcher := startWatcherProcess
				startWatcherProcess = func([]string, []string) error { return nil }
				t.Cleanup(func() { startWatcherProcess = oldWatcher })
				t.Setenv("PAIR_DATA_DIR", t.TempDir())
				t.Setenv(launcher.NativeAdmissionEnv, "")
				record := filepath.Join(t.TempDir(), "args")
				t.Setenv("RECORDED_ARGS", record)
				executable := filepath.Join(t.TempDir(), "codex")
				if err := os.WriteFile(executable, []byte("#!/bin/sh\n[ \"$1\" = resume ] || exit 0\nprintf '%s\\0' \"$@\" > \"$RECORDED_ARGS\"\n"), 0700); err != nil {
					t.Fatal(err)
				}
				argv := append([]string{executable}, args...)
				if fromEnvironment {
					encoded, err := launcher.EncodeAgentCommand(launcher.AgentCommand{Executable: executable, Argv: args})
					if err != nil {
						t.Fatal(err)
					}
					t.Setenv(launcher.AgentCommandEnv, encoded)
					argv = []string{"--from-launch-env"}
				}
				var stderr bytes.Buffer
				if code := Run(argv, bytes.NewReader(nil), &bytes.Buffer{}, &stderr); code != 0 {
					t.Fatalf("compatibility command refused: code=%d stderr=%s", code, &stderr)
				}
				raw, err := os.ReadFile(record)
				if err != nil || string(raw) != strings.Join(args, "\x00")+"\x00" {
					t.Fatalf("native argv changed: raw=%q args=%q err=%v", raw, args, err)
				}
				if stderr.String() != launcher.CodexIndirectResumeCompatibilityNotice+"\n" {
					t.Fatalf("expected exactly one compatibility notice: %q", stderr.String())
				}
			})
		}
	}
}

func TestCodexIndirectCompatibilityCannotBypassConsumedGrant(t *testing.T) {
	cmd := exec.Command("/native/codex", "resume", "saved-name")
	digest, err := launcher.NativeAdmissionCommandDigest("codex", cmd.Args)
	if err != nil {
		t.Fatal(err)
	}
	world := &nativeAdmissionWorld{state: "running", agent: "codex", digest: digest, childAlive: true}
	var notice bytes.Buffer
	spawns := 0
	_, err = startNativePTY(t.Context(), world, "valid-grant", "codex", cmd, &notice, func(*exec.Cmd) (*os.File, error) { spawns++; return nil, nil })
	if err == nil || spawns != 0 || notice.Len() != 0 || !reflect.DeepEqual(world.events, []string{"claim"}) || world.state != "running" || !world.childAlive {
		t.Fatalf("compatibility bypassed consumed grant: err=%v spawns=%d notice=%q world=%+v", err, spawns, notice.String(), world)
	}
}
