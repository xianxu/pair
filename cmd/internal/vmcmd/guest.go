package vmcmd

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
)

// GuestScript is staged into the owned input share by the host runtime.
//
//go:embed guest-run.sh
var GuestScript []byte

type GuestPaths struct{ Input, Results, WorkRoot string }
type GuestInvocation struct{ Args, Env []string }

// BuildGuestInvocation constructs literal Tart exec argv without consulting the host environment.
func BuildGuestInvocation(r RunRequest, p GuestPaths) (GuestInvocation, error) {
	paths := []string{p.Input, p.Results, p.WorkRoot}
	for i, a := range paths {
		if err := validateRunPath(a); err != nil || a == "/" {
			return GuestInvocation{}, fmt.Errorf("invalid guest root %q", a)
		}
		for _, b := range paths[:i] {
			if a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/") {
				return GuestInvocation{}, fmt.Errorf("guest roots overlap")
			}
		}
	}
	if len(r.Args) == 0 || r.Args[0] == "" {
		return GuestInvocation{}, fmt.Errorf("guest command required")
	}
	for _, a := range r.Args {
		if strings.ContainsRune(a, 0) {
			return GuestInvocation{}, fmt.Errorf("guest argv contains NUL")
		}
	}
	env := []string{"PATH=/opt/pair/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin", "LANG=C", "TERM=xterm-256color"}
	for _, entry := range [][2]string{{"HOME", "home"}, {"TMPDIR", "tmp"}, {"XDG_CONFIG_HOME", "config"}, {"XDG_DATA_HOME", "data"}, {"XDG_CACHE_HOME", "cache"}, {"XDG_STATE_HOME", "state"}, {"ZELLIJ_SOCKET_DIR", "sockets"}, {"PAIR_DATA_DIR", "pair"}, {"COUCH_ISOLATED_ROOT", "couch"}, {"WF_SDLC_QUEUE_ROOT", "queue"}} {
		env = append(env, entry[0]+"="+filepath.Join(p.WorkRoot, entry[1]))
	}
	args := append([]string{"/usr/bin/env", "-i"}, env...)
	args = append(args, "/bin/bash", "--noprofile", "--norc", filepath.Join(p.Input, "guest-run.sh"), filepath.Join(p.Input, "checkout"), p.WorkRoot, p.Results, "--")
	args = append(args, r.Args...)
	return GuestInvocation{Args: args, Env: env}, nil
}
