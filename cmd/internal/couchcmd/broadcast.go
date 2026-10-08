package couchcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

var (
	tunnelName   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	hostnameRule = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)
)

// broadcastSettings reads the broadcast options (#395). Environment only:
// Couch has no global config file.
//
//	COUCH_BROADCAST_TUNNEL=<name>       a named cloudflared tunnel (one-time setup:
//	COUCH_BROADCAST_HOSTNAME=<host>     tunnel login/create/route dns), over a private socket
//	COUCH_BROADCAST_TUNNEL unset        an anonymous quick tunnel
//	COUCH_BROADCAST_TUNNEL=off          this machine only
//	COUCH_BROADCAST_SWITCHER=show       include the switcher (default: a placeholder)
//
// store is Couch's store directory, which holds the tunnels' run records;
// guard is the argv that runs the guard (empty runs cloudflared directly).
// An unrecognised value is a startup error naming the variable, so a typo
// can't silently pick a different exposure.
func broadcastSettings(getenv func(string) string, store string, guard []string) (broadcast.Config, error) {
	var cfg broadcast.Config
	switch v := getenv("COUCH_BROADCAST_SWITCHER"); v {
	case "":
	case "show":
		cfg.Hub.ShowSwitcher = true
	default:
		return cfg, fmt.Errorf("COUCH_BROADCAST_SWITCHER must be unset or \"show\", not %q", v)
	}
	tunnel, hostname := getenv("COUCH_BROADCAST_TUNNEL"), getenv("COUCH_BROADCAST_HOSTNAME")
	cloudflared := broadcast.Cloudflared{Guard: guard}
	if store != "" {
		cloudflared.Records = filepath.Join(store, "broadcast")
	}
	switch {
	case tunnel == "off":
		if hostname != "" {
			return cfg, fmt.Errorf("COUCH_BROADCAST_HOSTNAME is set but COUCH_BROADCAST_TUNNEL=off")
		}
		cfg.Tunnel = broadcast.LocalOnly{}
	case tunnel == "":
		if hostname != "" {
			return cfg, fmt.Errorf("COUCH_BROADCAST_HOSTNAME needs COUCH_BROADCAST_TUNNEL, the named tunnel that serves it")
		}
		cfg.Tunnel = cloudflared
	default:
		if !tunnelName.MatchString(tunnel) {
			return cfg, fmt.Errorf("COUCH_BROADCAST_TUNNEL %q is not a tunnel name (letters, digits, . _ -)", tunnel)
		}
		if !hostnameRule.MatchString(hostname) {
			return cfg, fmt.Errorf("COUCH_BROADCAST_TUNNEL=%s needs COUCH_BROADCAST_HOSTNAME, the hostname routed to it, not %q", tunnel, hostname)
		}
		cloudflared.Named = &broadcast.NamedTunnel{Name: tunnel, Hostname: hostname}
		cfg.Tunnel = cloudflared
	}
	return cfg, nil
}

// guardArgv is how this binary runs the broadcast guard.
func guardArgv() []string {
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	return []string{exe, broadcast.GuardSubcommand}
}
