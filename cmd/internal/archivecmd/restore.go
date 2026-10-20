package archivecmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

const restoreUsage = "Usage: pair archive restore --id ID [--id ID ...] --to DIR [--from auto|local|drive] [--path REL ...]\n\nRestore verified files into a new directory under an existing private scratch parent. Select full bundle IDs from find. Repeated --path values select literal relative file names. The default source is auto: use local ciphertext, or Drive when the local copy is absent.\n"

// RestoreFunc follows the configured restore implementation without requiring
// CLI tests to unlock a key or contact a remote.
var RestoreFunc = func(ctx context.Context, restorer historyarchive.Restorer, req historyarchive.RestoreRequest) (historyarchive.RestoreReport, error) {
	return restorer.Run(ctx, req)
}

func runRestore(ctx context.Context, args []string, stdout io.Writer, fail func(string, error) int) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := io.WriteString(stdout, restoreUsage); err != nil {
			return fail("write help", err)
		}
		return 0
	}
	req := historyarchive.RestoreRequest{From: "auto"}
	seen := map[string]bool{}
	for n := 0; n < len(args); n += 2 {
		if n+1 >= len(args) || args[n+1] == "" {
			return fail("arguments", errors.New(restoreUsage))
		}
		switch args[n] {
		case "--id":
			req.IDs = append(req.IDs, args[n+1])
		case "--path":
			req.Paths = append(req.Paths, args[n+1])
		case "--to", "--from":
			if seen[args[n]] {
				return fail("arguments", errors.New(restoreUsage))
			}
			seen[args[n]] = true
			if args[n] == "--to" {
				req.To = args[n+1]
			} else {
				req.From = args[n+1]
			}
		default:
			return fail("arguments", errors.New(restoreUsage))
		}
	}
	if len(req.IDs) == 0 || req.To == "" || (req.From != "auto" && req.From != "local" && req.From != "drive") {
		return fail("arguments", errors.New(restoreUsage))
	}
	if err := ctx.Err(); err != nil {
		return fail("restore canceled", err)
	}
	cfg, path, err := archiveConfiguration()
	if err != nil {
		return fail("read config", err)
	}
	if cfg == nil {
		return fail("read config", errors.New("archiving is not configured; configure "+path))
	}
	restorer := historyarchive.Restorer{Config: *cfg, Decoder: historyarchive.CommandDecoder{}}
	report, err := RestoreFunc(ctx, restorer, req)
	if err != nil {
		switch report.State.Phase {
		case historyarchive.RestorePublicationUncertain:
			err = fmt.Errorf("complete restore output may exist at %q; inspect before retry: %w", req.To, err)
		case historyarchive.RestorePublished:
			err = fmt.Errorf("restore output was published at %q; inspect before retry: %w", req.To, err)
		}
		return fail("restore", err)
	}
	if report.State.Phase != historyarchive.RestorePublished {
		return fail("restore", errors.New("complete output publication was not confirmed; inspect target before retry"))
	}
	if _, err = fmt.Fprintf(stdout, "Restored %d bundle(s), %d file(s) to %q.\n", report.Bundles, report.Files, report.Target); err != nil {
		return fail("write restore result; complete output is published; inspect target before retry", err)
	}
	return 0
}
