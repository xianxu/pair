//go:build darwin

package launcher

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestLayoutLifetimeChild(t *testing.T) {
	pairlifecycletest.WrapLifetimeChild(t, "PAIR_769_LAYOUT", "TestLayoutLifetimeChild")
}

func TestLayoutCallerLifetimeDarwin(t *testing.T) {
	_ = testpair.Binary(t)
	helper, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, layout := range []string{"main-2.kdl", "main-3.kdl"} {
		raw, err := os.ReadFile(filepath.Join("../../../zellij/layouts", layout))
		if err != nil {
			t.Fatal(err)
		}
		var stanza string
		re := regexp.MustCompile(`args "-c" (".*")`)
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.Contains(line, "exec pair wrap") {
				m := re.FindStringSubmatch(line)
				if len(m) != 2 {
					t.Fatal(line)
				}
				stanza, err = strconv.Unquote(m[1])
				if err != nil {
					t.Fatal(err)
				}
			}
		}
		if stanza == "" {
			t.Fatal("missing production agent stanza")
		}
		for _, mode := range []string{"normal", "group", "daemon"} {
			t.Run(layout+"/"+mode, func(t *testing.T) {
				root := t.TempDir()
				rt := newFakeRuntime()
				args := []string{"-test.run=^TestLayoutLifetimeChild$"}
				code, err := run(t, baseOpts(LaunchArgs{Agent: "native769", AgentExplicit: true, ForcedTag: "work", AgentArgs: args, AgentArgsExplicit: true}), rt)
				if err != nil || code != 0 {
					t.Fatalf("production create=%d %v", code, err)
				}
				testpair.At(t, filepath.Join(root, "pair"))
				if err := os.Symlink(helper, filepath.Join(root, "native769")); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "zellij"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, "/bin/sh", "-c", stanza)
				capture := filepath.Join(root, "scroll.raw")
				cmd.Env = testpair.Environment(t, root, "PAIR_SCROLLBACK_EVENTS_PATH="+capture+".events", "PATH="+root+":"+os.Getenv("PATH"), AgentCommandEnv+"="+rt.env[AgentCommandEnv], "PAIR_AGENT_PANE_PATH="+filepath.Join(root, "pane.json"), "PAIR_SCROLLBACK_RAW_PATH="+filepath.Join(root, "scroll.raw"))
				pairlifecycletest.AssertWrapLifetime(t, cmd, mode, capture, "PAIR_769_LAYOUT")
			})
		}
	}
}
