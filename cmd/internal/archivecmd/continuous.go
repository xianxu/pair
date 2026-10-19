package archivecmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

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
	flags := flag.NewFlagSet("continuous", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	jsonMode := flags.Bool("json", false, "emit a typed report")
	kind := flags.String("kind", "", "source kind")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return fail("arguments", errors.New("use pair archive continuous [--kind KIND] [--json]"))
	}
	var kinds []string
	var emptyKind bool
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "kind" && *kind == "" {
			emptyKind = true
		}
	})
	if emptyKind {
		return fail("arguments", errors.New("kind must not be empty"))
	}
	if *kind != "" {
		kinds = []string{*kind}
	}
	maxAge, err := historyarchive.ParseDatabaseMaxAge(os.Getenv("PAIR_ARCHIVE_DATABASE_MAX_AGE"))
	if err != nil {
		return fail("PAIR_ARCHIVE_DATABASE_MAX_AGE", err)
	}
	maxDriveFiles, err := historyarchive.ParseDriveMaxFiles(os.Getenv("PAIR_ARCHIVE_DRIVE_MAX_FILES"))
	if err != nil {
		return fail("PAIR_ARCHIVE_DRIVE_MAX_FILES", err)
	}
	emit := func(outcome string, report historyarchive.ContinuousReport) error {
		raw, err := historyarchive.EncodeContinuousReport(historyarchive.ContinuousCommandReport{Version: historyarchive.ContinuousReportVersion, Outcome: outcome, MaxDatabaseAge: maxAge, MaxDriveFiles: maxDriveFiles, Report: report})
		if err != nil {
			return err
		}
		_, err = stdout.Write(raw)
		return err
	}
	home, env := archiveEnvironment()
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
		if *jsonMode {
			if err := emit(historyarchive.ContinuousNotConfigured, historyarchive.ContinuousReport{}); err != nil {
				return fail("report", err)
			}
		} else {
			fmt.Fprintf(stdout, "continuous archiving is not configured; write %s\n", configPath)
		}
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
		if *jsonMode {
			if err := emit(historyarchive.ContinuousAlreadyRunning, historyarchive.ContinuousReport{}); err != nil {
				return fail("report", err)
			}
		} else {
			fmt.Fprintln(stdout, err.Error())
		}
		return 0
	}
	if *jsonMode {
		outcome := historyarchive.ContinuousComplete
		if err != nil {
			outcome = historyarchive.ContinuousFailed
		}
		if reportErr := emit(outcome, report); reportErr != nil {
			return fail("report", reportErr)
		}
		if err != nil {
			return fail("continuous pass", err)
		}
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
	sort.Slice(report.Databases, func(i, j int) bool {
		a, b := report.Databases[i], report.Databases[j]
		return a.Kind < b.Kind || a.Kind == b.Kind && a.Path < b.Path
	})
	now := time.Now()
	for _, database := range report.Databases {
		fmt.Fprintln(stdout, database.Line(now))
		if warning := database.Warning(now, maxAge); warning != "" {
			fmt.Fprintln(stdout, warning)
		}
	}
	var inventory *historyarchive.DriveInventoryObservation
	if report.Drive != nil {
		remote := report.Drive
		inventory = remote.Inventory
		fmt.Fprintf(stdout, "Drive: %d verified, %d already recorded, %d pending (%d deferred); %d downloaded checksum(s)\n", remote.Verified, remote.AlreadyRecorded, remote.Pending, remote.Deferred, remote.DownloadedChecksums)
		for _, note := range remote.Notes {
			fmt.Fprintf(stdout, "Drive: %s\n", note)
		}
	}
	fmt.Fprintln(stdout, inventory.Line())
	if warning := inventory.Warning(maxDriveFiles); warning != "" {
		fmt.Fprintln(stdout, warning)
	}
	if err != nil {
		return fail("continuous pass", err)
	}
	return 0
}
