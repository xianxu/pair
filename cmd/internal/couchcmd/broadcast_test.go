package couchcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func TestBroadcastSettings(t *testing.T) {
	cases := []struct {
		env        map[string]string
		wantErr    string
		wantShow   bool
		wantTunnel string
	}{
		{env: nil, wantTunnel: "local"},
		{env: map[string]string{"COUCH_BROADCAST_TUNNEL": "off"}, wantTunnel: "local"},
		{env: map[string]string{"COUCH_BROADCAST_SWITCHER": "show"}, wantShow: true, wantTunnel: "local"},
		{env: map[string]string{"COUCH_BROADCAST_SWITCHER": "yes"}, wantErr: "COUCH_BROADCAST_SWITCHER"},
		{env: map[string]string{"COUCH_BROADCAST_TUNNEL": "cloudflared"}, wantErr: "COUCH_BROADCAST_TUNNEL"},
	}
	for _, c := range cases {
		cfg, err := broadcastSettings(func(k string) string { return c.env[k] })
		if c.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("%v: err = %v, want one naming %s", c.env, err, c.wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%v: %v", c.env, err)
		}
		if cfg.Hub.ShowSwitcher != c.wantShow {
			t.Errorf("%v: ShowSwitcher = %v", c.env, cfg.Hub.ShowSwitcher)
		}
		if _, ok := cfg.Tunnel.(broadcast.LocalOnly); !ok || c.wantTunnel != "local" {
			t.Errorf("%v: tunnel %T", c.env, cfg.Tunnel)
		}
	}
}
