// Package archivecmd exposes explicit, operator-selected history backups.
package archivecmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"golang.org/x/sys/unix"
)

const usage = "Usage: pair archive inventory\n       pair archive create --request FILE\n       pair archive continuous [--kind KIND] [--json]\n       pair archive find [--kind KIND] [--since UTC] [--until UTC]\n       pair archive sync-index [--from drive|local]\n       pair archive restore --id ID [--id ID ...] --to DIR [--from auto|local|drive] [--path REL ...]\n\nCreate verified encrypted backups, find local labels, synchronize the label/location index, or restore into a fresh scratch directory (configured in $PAIR_DATA_DIR/archive/continuous.json). Sources are preserved. Drive index synchronization downloads labels only.\n"

type CreateFunc func(context.Context, historyarchive.ArchiveRequest) ([]historyarchive.CatalogRecord, error)

func Run(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return RunContext(ctx, args, stdout, stderr, nil)
}

// RunContext permits a persistent archive implementation at the create seam.
// Help and inventory do not read source directories or create storage.
func RunContext(ctx context.Context, args []string, stdout, stderr io.Writer, create CreateFunc) int {
	fail := func(phase string, err error) int {
		// One diagnostic line, even when a child reports several diagnostic lines.
		message := rowtext.Sanitize(strings.Join(strings.Fields(err.Error()), " "))
		fmt.Fprintf(stderr, "pair archive: %s: %s\n", phase, message)
		return 1
	}
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help")) {
		if _, err := io.WriteString(stdout, usage); err != nil {
			return fail("write help", err)
		}
		return 0
	}
	if len(args) == 1 && args[0] == "inventory" {
		inventory, err := historyarchive.Defaults()
		if err != nil {
			return fail("load inventory", err)
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(inventory); err != nil {
			return fail("write inventory", err)
		}
		return 0
	}
	if len(args) >= 1 && args[0] == "continuous" {
		return runContinuous(ctx, args[1:], stdout, fail)
	}
	if len(args) >= 1 && args[0] == "find" {
		return runFind(ctx, args[1:], stdout, fail)
	}
	if len(args) >= 1 && args[0] == "restore" {
		return runRestore(ctx, args[1:], stdout, fail)
	}
	if len(args) >= 1 && args[0] == "sync-index" {
		return runSyncIndex(ctx, args[1:], stdout, fail)
	}
	if len(args) != 3 || args[0] != "create" || args[1] != "--request" || args[2] == "" {
		return fail("arguments", errors.New("use pair archive inventory or pair archive create --request FILE"))
	}
	if err := ctx.Err(); err != nil {
		return fail("create canceled", err)
	}
	raw, err := readRequest(args[2])
	if err != nil {
		return fail("read request; provide a regular JSON file up to 4 MiB", err)
	}
	home, env := archiveEnvironment()
	request, err := historyarchive.DecodeRequestWithRoots(raw, home, env)
	if err != nil {
		return fail("validate request; correct JSON fields and explicit destinations", err)
	}
	if create == nil {
		create = func(ctx context.Context, r historyarchive.ArchiveRequest) ([]historyarchive.CatalogRecord, error) {
			a := historyarchive.Archiver{Codec: historyarchive.CommandCodec{}, Sink: historyarchive.LocalSink{Root: r.Sink}}
			return a.Create(ctx, r)
		}
	}
	records, err := create(ctx, request)
	if err != nil {
		return fail("create; inspect the failure before retrying the same request", err)
	}
	// Public output contains only bundle labels and object keys, never manifests
	// or selected source paths. The private catalog retains the detailed manifest.
	type result struct {
		Label     historyarchive.BundleLabel `json:"label"`
		BundleKey string                     `json:"bundle_key"`
		LabelKey  string                     `json:"label_key"`
	}
	enc := json.NewEncoder(stdout)
	for _, r := range records {
		if err := enc.Encode(result{r.Label, r.BundleKey, r.LabelKey}); err != nil {
			return fail("write result; backup may already be committed; retry the same request", err)
		}
	}
	return 0
}

func readRequest(path string) ([]byte, error) {
	// O_NONBLOCK prevents FIFO opens from hanging before the regular-file check;
	// O_NOFOLLOW refuses a final-component symlink and fstat closes replacement races.
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), "archive request")
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsRegular() {
		return nil, errors.New("request must be a regular file")
	}
	if stat.Size() > historyarchive.MaxMetadataBytes {
		return nil, errors.New("request exceeds 4 MiB")
	}
	b, err := io.ReadAll(io.LimitReader(f, historyarchive.MaxMetadataBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > historyarchive.MaxMetadataBytes {
		return nil, errors.New("request exceeds 4 MiB")
	}
	return b, nil
}
