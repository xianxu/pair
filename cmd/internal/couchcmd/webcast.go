package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast"
)

// webcastSettings configures the cast Chrome (#454) on the Host the screen
// broadcast shares. COUCH_WEBCAST_CHROME, when set, must name an existing
// file: checked now, so a typo refuses the start rather than surfacing at the
// first cast. Otherwise Chrome is looked up when a cast starts, so installing
// it later needs no restart.
//
// The profile lives at <store>/webcast/chrome-profile and is kept across
// casts, so logins to the app under test survive; delete the directory to
// reset it.
func webcastSettings(getenv func(string) string, store string, host *broadcast.Host, lookPath func(string) (string, error), exists func(string) bool) (webcast.CastConfig, error) {
	find := func() (string, error) { return webcast.FindChrome(getenv, lookPath, exists) }
	if getenv(webcast.ChromeEnv) != "" {
		if _, err := find(); err != nil {
			return webcast.CastConfig{}, err
		}
	}
	cfg := webcast.CastConfig{Host: host}
	cfg.Launch = func(ctx context.Context) (webcast.Browser, error) {
		if store == "" {
			return nil, errors.New("webcast: Couch has no store directory for the Chrome profile")
		}
		bin, err := find()
		if err != nil {
			return nil, err
		}
		return webcast.Launch(ctx, webcast.LaunchOptions{Binary: bin, ProfileDir: filepath.Join(store, "webcast", "chrome-profile")})
	}
	return cfg, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
