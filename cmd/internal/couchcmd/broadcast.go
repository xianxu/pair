package couchcmd

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

// broadcastSettings reads the broadcast options (#395):
//
//	COUCH_BROADCAST_SWITCHER=show  include the switcher in broadcasts (default: a placeholder)
//	COUCH_BROADCAST_TUNNEL=off     serve on loopback only, no tunnel
//
// An unrecognised value is a startup error naming the variable, so a typo
// can't silently broadcast the switcher or pick an unexpected exposure.
func broadcastSettings(getenv func(string) string) (broadcast.Config, error) {
	var cfg broadcast.Config
	switch v := getenv("COUCH_BROADCAST_SWITCHER"); v {
	case "":
	case "show":
		cfg.Hub.ShowSwitcher = true
	default:
		return cfg, fmt.Errorf("COUCH_BROADCAST_SWITCHER must be unset or \"show\", not %q", v)
	}
	switch v := getenv("COUCH_BROADCAST_TUNNEL"); v {
	case "", "off":
		// Until the cloudflared tunnel lands (#395 M4), every broadcast is
		// local-only.
		cfg.Tunnel = broadcast.LocalOnly{}
	default:
		return cfg, fmt.Errorf("COUCH_BROADCAST_TUNNEL must be unset or \"off\", not %q", v)
	}
	return cfg, nil
}
