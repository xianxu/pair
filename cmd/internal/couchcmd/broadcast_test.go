package couchcmd

import (
	"io"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func TestBroadcastSettings(t *testing.T) {
	guard := []string{"/bin/couch", broadcast.GuardSubcommand}
	cases := []struct {
		name    string
		env     map[string]string
		wantErr string
		check   func(t *testing.T, cfg broadcast.Config)
	}{
		{name: "unset is a quick tunnel", check: func(t *testing.T, cfg broadcast.Config) {
			c, ok := cfg.Tunnel.(broadcast.Cloudflared)
			if !ok || c.Named != nil || c.Records != "/store/broadcast" || len(c.Guard) != 2 {
				t.Fatalf("tunnel %+v", cfg.Tunnel)
			}
		}},
		{name: "named tunnel", env: map[string]string{"COUCH_BROADCAST_TUNNEL": "couch-broadcast", "COUCH_BROADCAST_HOSTNAME": "live.functeer.com"}, check: func(t *testing.T, cfg broadcast.Config) {
			c, ok := cfg.Tunnel.(broadcast.Cloudflared)
			if !ok || c.Named == nil || *c.Named != (broadcast.NamedTunnel{Name: "couch-broadcast", Hostname: "live.functeer.com"}) {
				t.Fatalf("tunnel %+v", cfg.Tunnel)
			}
		}},
		{name: "off", env: map[string]string{"COUCH_BROADCAST_TUNNEL": "off"}, check: func(t *testing.T, cfg broadcast.Config) {
			if _, ok := cfg.Tunnel.(broadcast.LocalOnly); !ok {
				t.Fatalf("tunnel %T", cfg.Tunnel)
			}
		}},
		{name: "switcher shown", env: map[string]string{"COUCH_BROADCAST_SWITCHER": "show"}, check: func(t *testing.T, cfg broadcast.Config) {
			if !cfg.Hub.ShowSwitcher {
				t.Fatal("ShowSwitcher off")
			}
		}},
		{name: "switcher typo", env: map[string]string{"COUCH_BROADCAST_SWITCHER": "yes"}, wantErr: "COUCH_BROADCAST_SWITCHER"},
		{name: "named without hostname", env: map[string]string{"COUCH_BROADCAST_TUNNEL": "couch-broadcast"}, wantErr: "COUCH_BROADCAST_HOSTNAME"},
		{name: "hostname without tunnel", env: map[string]string{"COUCH_BROADCAST_HOSTNAME": "live.functeer.com"}, wantErr: "COUCH_BROADCAST_TUNNEL"},
		{name: "hostname with off", env: map[string]string{"COUCH_BROADCAST_TUNNEL": "off", "COUCH_BROADCAST_HOSTNAME": "live.functeer.com"}, wantErr: "off"},
		{name: "bad tunnel name", env: map[string]string{"COUCH_BROADCAST_TUNNEL": "../x", "COUCH_BROADCAST_HOSTNAME": "live.functeer.com"}, wantErr: "not a tunnel name"},
		{name: "bad hostname", env: map[string]string{"COUCH_BROADCAST_TUNNEL": "t", "COUCH_BROADCAST_HOSTNAME": "live functeer"}, wantErr: "COUCH_BROADCAST_HOSTNAME"},
		{name: "hostname with a path", env: map[string]string{"COUCH_BROADCAST_TUNNEL": "t", "COUCH_BROADCAST_HOSTNAME": "live.functeer.com/x"}, wantErr: "COUCH_BROADCAST_HOSTNAME"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := broadcastSettings(func(k string) string { return c.env[k] }, "/store", guard)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err = %v, want one mentioning %s", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			c.check(t, cfg)
		})
	}
}

// The guard is reached through couch's own argv, before any CLI parsing.
func TestGuardSubcommandDispatch(t *testing.T) {
	var stderr strings.Builder
	// An open pipe: Couch alive. (EOF would mean Couch is gone, and the guard
	// would stop the child before it could exit on its own.)
	owner, couch := io.Pipe()
	defer couch.Close()
	code := Run([]string{broadcast.GuardSubcommand, "--", "/bin/sh", "-c", "exit 5"}, owner, nil, &stderr)
	if code != 5 || !strings.Contains(stderr.String(), "broadcast-guard: child ") {
		t.Fatalf("code %d stderr %q", code, stderr.String())
	}
}
