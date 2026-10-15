package wrapcmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestNativeAdmissionUnsupportedSelectorsNeverConsumeOrSpawn(t *testing.T) {
	for _, args := range [][]string{{"--teleport", "session"}, {"--from-pr=42"}, {"--cloud", "task"}, {"attach", "session"}, {"--resume", "id", "--teleport", "session"}, {"--fork-session", "--from-pr", "42"}, {"--session-id", "new", "--cloud", "task"}} {
		for _, grant := range []string{"", "valid-grant"} {
			world := &nativeAdmissionWorld{state: "reserved", agent: "claude"}
			spawns := 0
			_, err := startNativePTY(context.Background(), world, grant, "claude", exec.Command("/native/claude", args...), io.Discard, func(*exec.Cmd) (*os.File, error) { spawns++; return nil, nil })
			if err == nil || spawns != 0 || len(world.events) != 0 {
				t.Errorf("unsupported %q grant=%t: spawns=%d events=%v err=%v", args, grant != "", spawns, world.events, err)
			}
		}
	}
}

func TestNativeAdmissionWrapperEntryFormsRespectUnsupportedContextBoundary(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		allow bool
	}{
		{"teleport", []string{"--teleport", "session"}, false},
		{"from-pr", []string{"--from-pr", "42"}, false},
		{"from-pr-inline", []string{"--from-pr=42"}, false},
		{"cloud", []string{"--cloud", "task"}, false},
		{"attach", []string{"attach", "session"}, false},
		{"respawn", []string{"respawn", "session"}, false},
		{"new-does-not-hide", []string{"--session-id", "new", "--teleport", "session"}, false},
		{"fork-does-not-hide", []string{"--fork-session", "--from-pr", "42"}, false},
		{"option-value", []string{"--system-prompt", "--teleport"}, true},
		{"inline-option-value", []string{"--system-prompt=--from-pr"}, true},
		{"command-as-value", []string{"--model", "attach"}, true},
		{"prompt-after-delimiter", []string{"--", "--teleport", "--from-pr"}, true},
	}
	for _, tc := range cases {
		for _, fromEnv := range []bool{false, true} {
			form := "argv"
			if fromEnv {
				form = "from-launch-env"
			}
			t.Run(tc.name+"/"+form, func(t *testing.T) {
				isolateNotificationSockets(t)
				t.Setenv("HOME", t.TempDir())
				t.Setenv("PAIR_DATA_DIR", "")
				t.Setenv("PAIR_TAG", "")
				t.Setenv("PAIR_LAUNCH_ORDINAL", "")
				t.Setenv("PAIR_RETENTION_START_ID", "")
				t.Setenv(launcher.NativeAdmissionEnv, "")
				t.Setenv("PAIR_WRAP_REMAP_RETURN", "0")
				marker := filepath.Join(t.TempDir(), "spawned")
				t.Setenv("BR6_SPAWN_MARKER", marker)
				executable := filepath.Join(t.TempDir(), "claude")
				if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf spawned > \"$BR6_SPAWN_MARKER\"\n"), 0700); err != nil {
					t.Fatal(err)
				}
				argv := append([]string{executable}, tc.args...)
				if fromEnv {
					command, err := launcher.EncodeAgentCommand(launcher.AgentCommand{Executable: executable, Argv: tc.args})
					if err != nil {
						t.Fatal(err)
					}
					t.Setenv(launcher.AgentCommandEnv, command)
					argv = []string{"--from-launch-env"}
				}
				var stderr bytes.Buffer
				code := Run(argv, bytes.NewReader(nil), io.Discard, &stderr)
				_, err := os.Stat(marker)
				if tc.allow {
					if code != 0 || err != nil {
						t.Fatalf("literal value/prompt refused: code=%d marker=%v stderr=%s", code, err, &stderr)
					}
				} else {
					if code == 0 || !os.IsNotExist(err) {
						t.Fatalf("existing selector spawned without admission: code=%d marker=%v stderr=%s", code, err, &stderr)
					}
					if !strings.Contains(stderr.String(), "native resume") {
						t.Fatalf("refused before testing native selection: %s", &stderr)
					}
				}
			})
		}
	}
}

func TestNativeAdmissionUnsupportedWordsInsideValuesStillUseValidGrant(t *testing.T) {
	for _, args := range [][]string{{"--resume", "id", "--system-prompt", "--teleport"}, {"--resume", "id", "--", "--from-pr"}} {
		cmd := exec.Command("/native/claude", args...)
		digest, err := launcher.NativeAdmissionCommandDigest("claude", cmd.Args)
		if err != nil {
			t.Fatal(err)
		}
		world := &nativeAdmissionWorld{state: "reserved", agent: "claude", digest: digest}
		_, err = startNativePTY(context.Background(), world, "valid-grant", "claude", cmd, io.Discard, func(cmd *exec.Cmd) (*os.File, error) {
			world.events = append(world.events, "spawn")
			world.childAlive = true
			cmd.Process = &os.Process{Pid: 123}
			return nil, nil
		})
		if err != nil || world.state != "running" || strings.Join(world.events, ",") != "claim,spawn,publish" {
			t.Fatalf("literal value/prompt broke valid admission: args=%q state=%s events=%v err=%v", args, world.state, world.events, err)
		}
	}
}
