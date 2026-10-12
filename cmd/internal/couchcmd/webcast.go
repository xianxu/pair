package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// webcastSettings configures the cast Chrome (#454) on the Host the screen
// broadcast shares. COUCH_WEBCAST_CHROME, when set, must name an existing
// file: checked now, so a typo refuses the start rather than surfacing at the
// first cast. Otherwise Chrome is looked up when a cast starts, so installing
// it later needs no restart.
//
// The profile lives at <store>/webcast/chrome-profile and is kept across
// casts, so logins to the app under test survive; delete the directory to
// reset it. Each cast is recorded under <store>/webcast/recordings/<id>/
// from its first go-live (#458), kept until the operator deletes it.
// recordingsDir is where casts record (#458) and replay reads (#471).
func recordingsDir(store string) string { return filepath.Join(store, "webcast", "recordings") }

func webcastSettings(getenv func(string) string, store string, host *broadcast.Host, lookPath func(string) (string, error), exists func(string) bool) (webcast.CastConfig, error) {
	find := func() (string, error) { return webcast.FindChrome(getenv, lookPath, exists) }
	if getenv(webcast.ChromeEnv) != "" {
		if _, err := find(); err != nil {
			return webcast.CastConfig{}, err
		}
	}
	cfg := webcast.CastConfig{Host: host}
	if store != "" {
		cfg.RecordingsDir = recordingsDir(store)
		// The viewers' fan-out (#460's hub), one per cast.
		// The recording's asset store (#459), persisting through its DirWriter.
		cfg.NewAssets = func(w recording.AssetWriter) assets.Served { return assets.NewRecorded(w, assets.DefaultPolicy) }
		cfg.NewLive = func() (recording.Sink, recording.Live) {
			h := recording.NewHub(liveBuffer)
			return h, h
		}
		// Each recorded cast is checked against its frames when it ends
		// (#478); the report lands in the recording's fidelity/.
		cfg.Check = fidelityHook(find, webcast.CheckFidelity)
	}
	cfg.Launch = func(ctx context.Context, rec *webcast.RecordOptions) (webcast.Browser, error) {
		if store == "" {
			return nil, errors.New("webcast: Couch has no store directory for the Chrome profile")
		}
		bin, err := find()
		if err != nil {
			return nil, err
		}
		return webcast.Launch(ctx, webcast.LaunchOptions{Binary: bin, ProfileDir: filepath.Join(store, "webcast", "chrome-profile"), Record: rec})
	}
	return cfg, nil
}

// liveBuffer is how far a viewer may fall behind the recording before the
// fan-out drops it (it reconnects with a fresh catch-up).
const liveBuffer = 4096

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
