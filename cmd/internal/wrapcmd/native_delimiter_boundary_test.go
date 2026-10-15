package wrapcmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestNativeAdmissionDelimiterCanonicalIDsNeverBypassEitherWrapperForm(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	cases := []struct {
		agent string
		args  []string
		allow bool
	}{{"codex", []string{"resume", "--", id}, false}, {"codex", []string{"exec", "resume", "--", id}, false}, {"muse", []string{"resume", "--", id}, false}}
	for _, agent := range launcher.AgentInventory() {
		option := "--model"
		if agent == "muse" {
			option = "--provider"
		}
		selector := []string{"--resume", id}
		if agent == "codex" || agent == "muse" {
			selector = []string{"resume", id}
		}
		if agent == "agy" {
			selector = []string{"--conversation", id}
		}
		cases = append(cases, struct {
			agent string
			args  []string
			allow bool
		}{agent, append([]string{option, "--"}, selector...), false}, struct {
			agent string
			args  []string
			allow bool
		}{agent, []string{"--", "--resume", id}, true})
	}
	for _, tc := range cases {
		for _, fromEnv := range []bool{false, true} {
			form := "argv"
			if fromEnv {
				form = "from-launch-env"
			}
			t.Run(tc.agent+"/"+strings.Join(tc.args, " ")+"/"+form, func(t *testing.T) {
				isolateNotificationSockets(t)
				t.Setenv("HOME", t.TempDir())
				t.Setenv("PAIR_DATA_DIR", "")
				t.Setenv("PAIR_TAG", "")
				t.Setenv("PAIR_LAUNCH_ORDINAL", "")
				t.Setenv("PAIR_RETENTION_START_ID", "")
				t.Setenv(launcher.NativeAdmissionEnv, "")
				t.Setenv("PAIR_WRAP_REMAP_RETURN", "0")
				marker := filepath.Join(t.TempDir(), "spawned")
				t.Setenv("DELIMITER_SPAWN_MARKER", marker)
				executable := filepath.Join(t.TempDir(), tc.agent)
				if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf spawned > \"$DELIMITER_SPAWN_MARKER\"\n"), 0700); err != nil {
					t.Fatal(err)
				}
				args := append([]string{executable}, tc.args...)
				if fromEnv {
					encoded, err := launcher.EncodeAgentCommand(launcher.AgentCommand{Executable: executable, Argv: tc.args})
					if err != nil {
						t.Fatal(err)
					}
					t.Setenv(launcher.AgentCommandEnv, encoded)
					args = []string{"--from-launch-env"}
				}
				var stderr bytes.Buffer
				code := Run(args, bytes.NewReader(nil), io.Discard, &stderr)
				_, err := os.Stat(marker)
				if tc.allow {
					if code != 0 || err != nil {
						t.Fatalf("literal prompt refused: code=%d marker=%v stderr=%s", code, err, &stderr)
					}
				} else if code == 0 || !os.IsNotExist(err) {
					t.Fatalf("canonical ID bypassed admission: code=%d marker=%v stderr=%s", code, err, &stderr)
				}
			})
		}
	}
}
