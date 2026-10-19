package archivecmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

const findUsage = "Usage: pair archive find [--kind KIND] [--since UTC] [--until UTC]\n\nFind local label metadata without contacting Drive. UTC accepts YYYY-MM-DD or RFC3339; the until bound is exclusive. Run sync-index --from drive to discover remote labels.\n"

func archiveEnvironment() (string, map[string]string) {
	home, _ := os.UserHomeDir()
	env := map[string]string{}
	for _, key := range []string{"PAIR_DATA_DIR", "XDG_DATA_HOME", "CLAUDE_CONFIG_DIR", "CODEX_HOME", "QODER_HOME", "COUCH_STORE_DIR"} {
		env[key] = os.Getenv(key)
	}
	return home, env
}

func archiveConfiguration() (*historyarchive.ContinuousConfig, string, error) {
	home, env := archiveEnvironment()
	store, _, err := continuousStore(home, env)
	if err != nil {
		return nil, "", err
	}
	path := filepath.Join(store, historyarchive.ContinuousConfigName)
	cfg, err := historyarchive.LoadContinuousConfig(path)
	return cfg, path, err
}

func runFind(ctx context.Context, args []string, stdout io.Writer, fail func(string, error) int) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := io.WriteString(stdout, findUsage); err != nil {
			return fail("write help", err)
		}
		return 0
	}
	var q historyarchive.FindQuery
	seen := map[string]bool{}
	for n := 0; n < len(args); n += 2 {
		if n+1 >= len(args) || seen[args[n]] || args[n+1] == "" {
			return fail("arguments", errors.New(findUsage))
		}
		seen[args[n]] = true
		var err error
		switch args[n] {
		case "--kind":
			q.Kind = args[n+1]
		case "--since":
			q.Since, err = historyarchive.ParseFindTime(args[n+1])
		case "--until":
			q.Until, err = historyarchive.ParseFindTime(args[n+1])
		default:
			return fail("arguments", errors.New(findUsage))
		}
		if err != nil {
			return fail("arguments", err)
		}
	}
	// Validate even with no configured archive; malformed input is never a no-op.
	if _, err := historyarchive.SelectLabels(historyarchive.LabelIndex{Version: historyarchive.Version, Records: map[string]historyarchive.LabelIndexRecord{}}, q); err != nil {
		return fail("arguments", err)
	}
	cfg, path, err := archiveConfiguration()
	if err != nil {
		return fail("read config", err)
	}
	if cfg == nil {
		return fail("read config", errors.New("archiving is not configured; configure "+path))
	}
	view, err := historyarchive.LoadLabelView(ctx, *cfg, time.Now().UTC())
	if err != nil {
		return fail("read local labels", err)
	}
	records, err := historyarchive.SelectLabels(view, q)
	if err != nil {
		return fail("select labels", err)
	}
	enc := json.NewEncoder(stdout)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			return fail("write labels", err)
		}
	}
	return 0
}
