package archivecmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

const syncIndexUsage = "Usage: pair archive sync-index [--from drive|local]\n\nRebuild the local label/location index. Drive synchronization downloads labels only; bundle bytes are always verified during restore.\n"

// DriveIndexSyncFunc lets CLI tests exercise configuration without a live remote.
var DriveIndexSyncFunc = func(ctx context.Context, cfg historyarchive.ContinuousConfig) (historyarchive.IndexSyncReport, error) {
	return (historyarchive.DriveIndexSync{Rclone: historyarchive.Rclone{Runner: historyarchive.ExecRcloneRunner{}}}).Run(ctx, cfg)
}

func runSyncIndex(ctx context.Context, args []string, stdout io.Writer, fail func(string, error) int) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := io.WriteString(stdout, syncIndexUsage); err != nil {
			return fail("write help", err)
		}
		return 0
	}
	from := "local"
	if len(args) != 0 {
		if len(args) != 2 || args[0] != "--from" || (args[1] != "drive" && args[1] != "local") {
			return fail("arguments", errors.New(syncIndexUsage))
		}
		from = args[1]
	}
	cfg, path, err := archiveConfiguration()
	if err != nil {
		return fail("read config", err)
	}
	if cfg == nil {
		return fail("read config", errors.New("archiving is not configured; configure "+path))
	}
	count := 0
	if from == "drive" {
		report, e := DriveIndexSyncFunc(ctx, *cfg)
		err = e
		count = report.LabelCount
	} else {
		index, e := historyarchive.SyncLocalLabelIndex(ctx, *cfg, time.Now().UTC())
		err = e
		count = len(index.Records)
	}
	if err != nil {
		return fail("synchronize label index", err)
	}
	if _, err := fmt.Fprintf(stdout, "Index synchronized from %s: %d label(s).\n", from, count); err != nil {
		return fail("write index result; snapshot may already be committed", err)
	}
	return 0
}
