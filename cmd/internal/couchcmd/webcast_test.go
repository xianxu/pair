package couchcmd

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast"
)

func TestWebcastSettings(t *testing.T) {
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: broadcast.LocalOnly{}})
	noPath := func(string) (string, error) { return "", exec.ErrNotFound }
	none := func(string) bool { return false }
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

	t.Run("a typo in COUCH_WEBCAST_CHROME refuses the start", func(t *testing.T) {
		_, err := webcastSettings(env(map[string]string{webcast.ChromeEnv: "/nope"}), "/store", host, noPath, none)
		if err == nil || !strings.Contains(err.Error(), webcast.ChromeEnv) {
			t.Fatalf("err %v", err)
		}
	})
	t.Run("no Chrome installed is a cast-time refusal, not a startup one", func(t *testing.T) {
		cfg, err := webcastSettings(env(nil), "/store", host, noPath, none)
		if err != nil || cfg.Host != host {
			t.Fatalf("cfg %+v err %v", cfg, err)
		}
		if _, err := cfg.Launch(context.Background(), nil); !errors.Is(err, webcast.ErrNoChrome) {
			t.Fatalf("launch: %v", err)
		}
	})
	t.Run("no store", func(t *testing.T) {
		cfg, _ := webcastSettings(env(nil), "", host, noPath, none)
		if _, err := cfg.Launch(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "store") {
			t.Fatalf("launch: %v", err)
		}
	})
}
