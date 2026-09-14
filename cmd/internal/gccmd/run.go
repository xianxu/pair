// Package gccmd exposes preview, explicit migration, and collection of Pair data.
package gccmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/xianxu/pair/cmd/internal/gcruntime"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

type pathsFlag []string

func (p *pathsFlag) String() string         { return fmt.Sprint([]string(*p)) }
func (p *pathsFlag) Set(value string) error { *p = append(*p, value); return nil }

func Run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fail := func(err error) int { fmt.Fprintf(stderr, "pair gc: %v\n", err); return 1 }
	flags := flag.NewFlagSet("pair gc", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Pair storage root (defaults to selected Pair storage)")
	apply := flags.Bool("apply", false, "initialize clocks and collect eligible data after migration")
	jsonOutput := flags.Bool("json", false, "write detailed metadata-only JSON")
	complete := flags.Bool("complete-migration", false, "acknowledge the complete Couch store list supplied by --store")
	var register, expected pathsFlag
	flags.Var(&register, "register-store", "register an existing Couch namespace; repeat for multiple stores")
	flags.Var(&expected, "store", "Couch namespace included in explicit migration acknowledgment; repeat for every store")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || (*apply && *complete) || (len(expected) > 0 && !*complete) {
		return fail(errors.New("unexpected arguments or incompatible operations"))
	}
	if getenv == nil {
		return fail(errors.New("environment is required"))
	}
	selected := *root
	if selected == "" {
		selected = getenv("PAIR_DATA_DIR")
	}
	if selected == "" {
		selected = launcher.ResolveDataDir(getenv("HOME"), getenv("XDG_DATA_HOME"))
	}
	if !filepath.IsAbs(selected) {
		return fail(errors.New("storage root must be absolute"))
	}
	resolved, err := gcruntime.RootFromSelected(selected)
	if err != nil {
		return fail(err)
	}
	service, err := gcruntime.New(resolved)
	if err != nil {
		return fail(err)
	}
	coordinator := service.Collector.Coordinator
	ctx := context.Background()
	for _, path := range register {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return fail(err)
		}
		if err := coordinator.RegisterStore(ctx, absolute); err != nil {
			return fail(err)
		}
	}
	if *complete {
		for _, path := range expected {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return fail(err)
			}
			if err := coordinator.RegisterStore(ctx, absolute); err != nil {
				return fail(err)
			}
		}
		paths := make([]string, 0, len(expected))
		for _, path := range expected {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return fail(err)
			}
			paths = append(paths, absolute)
		}
		if err := coordinator.CompleteMigration(ctx, paths); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, "Couch store inventory acknowledged; automatic collection is enabled. Session data keeps its 60-day grace.")
		return 0
	}
	if *apply {
		// These configured namespaces are known without guessing legacy custom ones.
		// Any additional store must still be registered and explicitly acknowledged.
		defaults := []string{filepath.Join(resolved, "couch")}
		if configured := getenv("COUCH_STORE_DIR"); configured != "" {
			absolute, e := filepath.Abs(configured)
			if e != nil {
				return fail(e)
			}
			defaults = append(defaults, absolute)
		}
		for _, path := range defaults {
			if st, e := os.Stat(path); e == nil && st.IsDir() {
				if e := coordinator.RegisterStore(ctx, path); e != nil {
					return fail(e)
				}
			}
		}
	}
	var report gcruntime.Report
	if *apply {
		report, err = service.Apply(ctx, 100)
	} else {
		report, err = service.Preview(ctx)
	}
	if err != nil {
		return fail(err)
	}
	if *jsonOutput {
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			return fail(err)
		}
		return 0
	}
	render(stdout, report)
	registry, e := coordinator.ReadRegistry()
	if e == nil {
		fmt.Fprintln(stdout, "Registered Couch stores:")
		for _, path := range registry.Stores {
			fmt.Fprintf(stdout, "  %s\n", path)
		}
		if len(registry.Stores) == 0 {
			fmt.Fprintln(stdout, "  (none)")
		}
	}
	if !report.Storage.MigrationComplete {
		fmt.Fprintln(stdout, "Collection is disabled until all Couch stores are registered and acknowledged with --complete-migration --store PATH (repeat --store for every registered namespace). Use --apply to initialize 60-day session clocks.")
	}
	return 0
}
func render(out io.Writer, r gcruntime.Report) {
	type total struct {
		count int
		bytes int64
	}
	totals := map[string]total{}
	add := func(bucket, status string, n int, b int64) {
		key := bucket + " / " + status
		v := totals[key]
		v.count += n
		v.bytes += b
		totals[key] = v
	}
	for _, item := range r.Storage.Items {
		add(string(item.Bucket), string(item.Decision.State), len(item.Members), item.Bytes)
	}
	for _, item := range r.Diagnostics {
		status := "retained"
		if item.Eligible {
			status = "eligible"
		} else if item.Reason != "within seven-day retention" {
			status = "blocked"
		}
		add("debug", status, 1, item.Bytes)
	}
	keys := make([]string, 0, len(totals))
	for key := range totals {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Fprintln(out, "Retention: session 60 days; parked captures and debugging logs 7 days.")
	for _, key := range keys {
		v := totals[key]
		fmt.Fprintf(out, "%-25s %6d files %9.3f GiB\n", key, v.count, float64(v.bytes)/(1<<30))
	}
	if r.Storage.BlockReason != "" {
		fmt.Fprintln(out, r.Storage.BlockReason)
	}
	if len(r.Storage.Unknown) > 0 {
		fmt.Fprintf(out, "Unrecognized paths: %d (retained; --json lists them).\n", len(r.Storage.Unknown))
	}
	if r.Storage.Collected > 0 || r.DiagnosticCollectedBytes > 0 {
		fmt.Fprintf(out, "Collected %.3f GiB.\n", float64(r.Storage.CollectedBytes+r.DiagnosticCollectedBytes)/(1<<30))
	}
}
