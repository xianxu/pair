package archivecmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

// ContinuousPassFunc runs one configured pass; tests substitute it.
var ContinuousPassFunc = func(ctx context.Context, p historyarchive.ContinuousPass) (historyarchive.ContinuousReport, error) {
	return p.Run(ctx)
}

// continuousStore is $PAIR_DATA_DIR/archive (the archive store) and the
// private snapshot root beside it, outside the catalog's directory.
func continuousStore(home string, env map[string]string) (store, snapshots string, err error) {
	inv, err := historyarchive.Defaults()
	if err != nil {
		return "", "", err
	}
	data, err := inv.ResolveRoot("couch-sessions", "", home, env)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(data, "archive"), filepath.Join(data, "archive-snapshots"), nil
}

func runContinuous(ctx context.Context, args []string, stdout io.Writer, fail func(string, error) int) int {
	var kinds []string
	switch {
	case len(args) == 0:
	case len(args) == 2 && args[0] == "--kind" && args[1] != "":
		kinds = []string{args[1]}
	default:
		return fail("arguments", errors.New("use pair archive continuous [--kind KIND]"))
	}
	home, _ := os.UserHomeDir()
	env := map[string]string{}
	for _, key := range []string{"PAIR_DATA_DIR", "XDG_DATA_HOME", "CLAUDE_CONFIG_DIR", "CODEX_HOME", "QODER_HOME", "COUCH_STORE_DIR"} {
		env[key] = os.Getenv(key)
	}
	store, snapshots, err := continuousStore(home, env)
	if err != nil {
		return fail("resolve archive store", err)
	}
	configPath := filepath.Join(store, historyarchive.ContinuousConfigName)
	cfg, err := historyarchive.LoadContinuousConfig(configPath)
	if err != nil {
		return fail("read config", err)
	}
	if cfg == nil {
		fmt.Fprintf(stdout, "continuous archiving is not configured; write %s\n", configPath)
		return 0
	}
	host, _ := os.Hostname()
	pass := historyarchive.ContinuousPass{
		Archiver:     historyarchive.Archiver{Codec: historyarchive.CommandCodec{}, Sink: historyarchive.LocalSink{Root: cfg.Sink}},
		Config:       *cfg,
		Store:        store,
		SnapshotRoot: snapshots,
		Host:         host,
		Home:         home,
		Env:          env,
		Kinds:        kinds,
		Drive:        &historyarchive.DriveMirror{Rclone: historyarchive.Rclone{Runner: historyarchive.ExecRcloneRunner{}}},
		Snapshotter:  historyarchive.SQLiteSnapshotter{SQLite: "sqlite3", Scratch: filepath.Join(snapshots, ".scratch")},
	}
	report, err := ContinuousPassFunc(ctx, pass)
	if errors.Is(err, historyarchive.ErrPassRunning) {
		fmt.Fprintln(stdout, err.Error())
		return 0
	}
	kindsDone := make([]string, 0, len(report.Archived))
	for k := range report.Archived {
		kindsDone = append(kindsDone, k)
	}
	sort.Strings(kindsDone)
	for _, k := range kindsDone {
		fmt.Fprintf(stdout, "%s: %d file(s) archived\n", k, report.Archived[k])
	}
	for _, note := range report.Skipped {
		fmt.Fprintf(stdout, "skipped: %s\n", note)
	}
	if report.Drive != nil {
		remote := report.Drive
		fmt.Fprintf(stdout, "Drive: %d verified, %d already recorded, %d pending; %d downloaded checksum(s)\n", remote.Verified, remote.AlreadyRecorded, remote.Pending, remote.DownloadedChecksums)
		for _, note := range remote.Notes {
			fmt.Fprintf(stdout, "Drive: %s\n", note)
		}
	}
	if err != nil {
		return fail("continuous pass", err)
	}
	return 0
}
