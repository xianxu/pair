package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/fidelity"
)

// fidelityFlag checks a broadcast-web recording against the real tab's
// frames (#478): `couch --broadcast-web-fidelity [ID|DIR]`. A recorded
// cast is checked by itself when it ends (fidelityHook).
const fidelityFlag = "--broadcast-web-fidelity"

// Exit codes: within tolerance, error, usage, some frame over tolerance.
const (
	fidelityWithin = 0
	fidelityError  = 1
	fidelityUsage  = 2
	fidelityOver   = 3
)

// fidelityCheck is webcast.CheckFidelity, a seam for tests.
type fidelityCheck func(ctx context.Context, chromeBin, dir string) (fidelity.Report, error)

func parseFidelityArgs(args []string) (target string, err error) {
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "-"):
			return "", fmt.Errorf("unknown flag %s", a)
		case target != "":
			return "", errors.New("one recording at a time")
		default:
			target = a
		}
	}
	return target, nil
}

// fidelityDir is the recording to check: the one named (an id in the
// store or a directory, as the replay resolves them), else the store's
// newest.
func fidelityDir(target, store string) (string, error) {
	if target == "" {
		if store == "" {
			return "", errors.New("Couch has no store directory; name a recording directory")
		}
		recs, err := webcast.ListRecordings(recordingsDir(store))
		if err != nil || len(recs) == 0 {
			return "", fmt.Errorf("no recordings in %s", recordingsDir(store))
		}
		return filepath.Join(recordingsDir(store), recs[0].ID), nil
	}
	root, id, err := resolveReplay(target, recordingsDir(store))
	if err != nil {
		return "", err
	}
	return filepath.Join(root, id), nil
}

// reportLine is what the operator is told: the score and where the report is.
func reportLine(dir string, rep fidelity.Report) string {
	return fmt.Sprintf("fidelity %s; report: %s", rep.Summary(), filepath.Join(dir, fidelity.ReportDir, "report.md"))
}

func runFidelity(ctx context.Context, args []string, store string, findChrome func() (string, error), check fidelityCheck, stdout, stderr io.Writer) int {
	fail := func(code int, err error) int {
		fmt.Fprintf(stderr, "couch %s: %v\n", fidelityFlag, err)
		return code
	}
	target, err := parseFidelityArgs(args)
	if err != nil {
		return fail(fidelityUsage, err)
	}
	dir, err := fidelityDir(target, store)
	if err != nil {
		return fail(fidelityError, err)
	}
	bin, err := findChrome()
	if err != nil {
		return fail(fidelityError, err)
	}
	fmt.Fprintf(stderr, "Replaying %s at its frames in a headless Chrome…\n", dir)
	rep, err := check(ctx, bin, dir)
	if err != nil {
		return fail(fidelityError, err)
	}
	fmt.Fprintln(stdout, reportLine(dir, rep))
	switch {
	case len(rep.Shots) == 0:
		return fidelityError
	case rep.Over > 0:
		return fidelityOver
	}
	return fidelityWithin
}

// fidelitySlot lets one end-of-cast check run at a time in a Couch, so
// back-to-back casts don't stack headless Chromes.
var fidelitySlot = make(chan struct{}, 1)

// fidelityHook is CastConfig.Check: the check of a cast's recording when
// the cast ends, its line for the operator.
func fidelityHook(findChrome func() (string, error), check fidelityCheck) func(context.Context, string) (string, error) {
	return func(ctx context.Context, dir string) (string, error) {
		select {
		case fidelitySlot <- struct{}{}:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		defer func() { <-fidelitySlot }()
		bin, err := findChrome()
		if err != nil {
			return "", err
		}
		rep, err := check(ctx, bin, dir)
		if err != nil {
			return "", err
		}
		return reportLine(dir, rep), nil
	}
}
