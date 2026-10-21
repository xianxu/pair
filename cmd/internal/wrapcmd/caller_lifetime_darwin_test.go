//go:build darwin

package wrapcmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestWrapLifetimeChild(t *testing.T) {
	pairlifecycletest.WrapLifetimeChild(t, "PAIR_769_WRAP", "TestWrapLifetimeChild")
}

func TestWrapCallerLifetimeDarwin(t *testing.T) {
	image := testpair.Binary(t)
	helper, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, name := range []string{"pair", "pair-go", "pair-wrap"} {
		testpair.At(t, filepath.Join(root, name))
	}
	for _, route := range []string{"pair", "pair-go", "pair-wrap", "fresh"} {
		for _, mode := range []string{"normal", "group", "daemon"} {
			t.Run(route+"/"+mode, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
				defer cancel()
				capture := filepath.Join(t.TempDir(), "scroll.raw")
				argv := []string{filepath.Join(root, route), "wrap", "--scrollback-log", capture, helper, "-test.run=^TestWrapLifetimeChild$"}
				env := testpair.Environment(t, t.TempDir(), "PAIR_SCROLLBACK_EVENTS_PATH="+capture+".events")
				if route == "pair-wrap" {
					argv = append([]string{filepath.Join(root, route)}, argv[2:]...)
				}
				if route == "fresh" {
					req, err := freshAgentInvocation(image, capture, []string{helper, "-test.run=^TestWrapLifetimeChild$"}, env)
					if err != nil {
						t.Fatal(err)
					}
					argv, env = req.argv, req.env
				}
				cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
				cmd.Env = env
				pairlifecycletest.AssertWrapLifetime(t, cmd, mode, capture, "PAIR_769_WRAP")
			})
		}
	}
}
