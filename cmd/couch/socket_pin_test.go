package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestCouchSocketPinBoundary(t *testing.T) {
	if os.Getenv("PAIR_TEST_SOCKET_PIN_HELPER") == "1" {
		originalTemp := os.Getenv("TMPDIR")
		code := runWithPinnedSocketRoot(func() int {
			if os.Getenv("TMPDIR") != originalTemp {
				fmt.Fprintln(os.Stderr, "Couch TMPDIR changed")
				return 91
			}
			// A command owned by Couch inherits the pinned root even when its
			// temporary-directory environment is subsequently overlaid.
			cmd := exec.Command("/usr/bin/env")
			cmd.Env = append(os.Environ(), "TMPDIR="+filepath.Join(os.Getenv("HOME"), "ram-slot"))
			out, err := cmd.Output()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 92
			}
			_, _ = os.Stdout.Write(out)
			return 37
		}, os.Stderr)
		os.Exit(code)
	}
	for _, tc := range []struct {
		name, explicit, runtimeMode string
		invalid                     bool
	}{
		{name: "implicit"},
		{name: "implicit runtime absent", runtimeMode: "absent"},
		{name: "implicit runtime relative", runtimeMode: "relative"},
		{name: "custom", explicit: "custom"},
		{name: "relative rejected", explicit: "relative-root", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tmp := filepath.Join(root, "original-tmp")
			if err := os.MkdirAll(tmp, 0700); err != nil {
				t.Fatal(err)
			}
			explicit := tc.explicit
			runtimeDir := filepath.Join(root, "runtime")
			if tc.runtimeMode == "absent" {
				runtimeDir = ""
			}
			if tc.runtimeMode == "relative" {
				runtimeDir = "relative-runtime"
			}
			if explicit == "custom" {
				explicit = filepath.Join(root, "short-link", "sockets")
			}
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(exe, "-test.run=^TestCouchSocketPinBoundary$")
			cmd.Env = testpair.Environment(t, root,
				"PAIR_TEST_SOCKET_PIN_HELPER=1", "TMPDIR="+tmp,
				"ZELLIJ_SOCKET_DIR="+explicit,
				"XDG_CONFIG_HOME="+filepath.Join(root, "config"),
				"XDG_CACHE_HOME="+filepath.Join(root, "cache"),
				"XDG_DATA_HOME="+filepath.Join(root, "data"),
				"XDG_RUNTIME_DIR="+runtimeDir,
			)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("helper error=%v out=%q stderr=%q", err, out, stderr.String())
			}
			if tc.invalid {
				if exit.ExitCode() != 1 || len(out) != 0 || !strings.Contains(stderr.String(), "Zellij socket") {
					t.Fatalf("invalid root reached commands: exit=%d out=%q stderr=%q", exit.ExitCode(), out, stderr.String())
				}
				return
			}
			want := explicit
			if want == "" {
				want = filepath.Join(tmp, "zellij-"+strconv.Itoa(os.Getuid()))
				if runtime.GOOS == "linux" && filepath.IsAbs(runtimeDir) {
					want = filepath.Join(runtimeDir, "zellij")
				}
			}
			if exit.ExitCode() != 37 || stderr.Len() != 0 || !bytes.Contains(out, []byte("ZELLIJ_SOCKET_DIR="+want+"\n")) {
				t.Fatalf("namespace not pinned before commands: exit=%d want=%q out=%q stderr=%q", exit.ExitCode(), want, out, stderr.String())
			}
		})
	}
}
