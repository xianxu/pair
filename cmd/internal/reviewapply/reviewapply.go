// Package reviewapply is `pair review apply` (pair#661): the review
// workbench's one edit path, the shared sessionsync engine with no Couch
// owner, ledger or registry. It is its own package because reviewcmd sits
// below the launcher, which sessionsync's Couch half reaches.
package reviewapply

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

// applyResponse is `pair review apply --json` (pair#661): Code is ok or the
// refusal's code; Invalid lists bad records on edits-invalid; Hash is the new
// whole-file hash (the current one on stale); Body is the agent's commit body.
type applyResponse struct {
	Code    string
	Error   string
	Hash    string                `json:",omitempty"`
	Invalid []sessionsync.BadEdit `json:",omitempty"`
	Round   *sessionsync.Round    `json:",omitempty"`
	Body    string                `json:",omitempty"`
}

// RunCLI is `pair review apply EDITS.json FILE --if-hash H [--json]`:
// the review workbench's one edit path, the shared engine with no Couch.
// Exits: 0 written; 1 edits-invalid, propose-* or refused; 3 operator-lock;
// 6 stale.
func RunCLI(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	var positional []string
	ifHash, asJSON := "", false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--if-hash":
			if i+1 >= len(args) || args[i+1] == "" {
				fmt.Fprintln(stderr, "usage: pair review apply EDITS.json FILE --if-hash H [--json]")
				return 2
			}
			ifHash, i = args[i+1], i+1
		case "--json":
			asJSON = true
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 2 || ifHash == "" {
		fmt.Fprintln(stderr, "usage: pair review apply EDITS.json FILE --if-hash H [--json]")
		return 2
	}
	fail := func(err error) int {
		res, exit := applyResponse{Code: "error", Error: err.Error()}, 1
		var r *sessionsync.Refusal
		if errors.As(err, &r) {
			res.Code, res.Hash, res.Invalid, exit = r.Code, r.Hash, r.Invalid, r.Exit
		}
		if asJSON {
			_ = json.NewEncoder(stdout).Encode(res)
			return exit
		}
		fmt.Fprintln(stderr, "pair review apply: refused:", err)
		for _, b := range res.Invalid {
			fmt.Fprintf(stderr, "  record %d: %s\n", b.Index, b.Reason)
		}
		if res.Hash != "" {
			fmt.Fprintln(stdout, "hash:", res.Hash)
		}
		return exit
	}
	dataDir := getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		return fail(errors.New("PAIR_DATA_DIR not set"))
	}
	paths, err := artifactpath.ResolveScoped(dataDir, tagOrDefault(getenv("PAIR_TAG")))
	if err != nil {
		return fail(fmt.Errorf("resolve artifact namespace: %w", err))
	}
	roundPath := getenv("PAIR_REVIEW_ROUND_PATH")
	if roundPath == "" {
		roundPath = paths.ReviewRound()
	}
	file, err := filepath.Abs(positional[1])
	if err != nil {
		return fail(err)
	}
	raw, err := os.ReadFile(positional[0])
	if err != nil {
		return fail(fmt.Errorf("reading the edits: %w", err))
	}
	edits, err := sessionsync.ParseEdits(raw)
	if err != nil {
		return fail(&sessionsync.Refusal{Exit: sessionsync.ExitRefused, Code: "edits-invalid", Reason: err.Error()})
	}
	res, err := sessionsync.ApplyFile(sessionsync.FileApplyRequest{File: file, IfHash: ifHash,
		FlockPath: paths.ReviewApplyLock(), RoundPath: roundPath, Edits: edits, Now: time.Now()})
	if err != nil {
		return fail(err)
	}
	if asJSON {
		if err := json.NewEncoder(stdout).Encode(applyResponse{Code: "ok", Hash: res.Hash, Round: res.Round, Body: res.Body}); err != nil {
			return fail(err)
		}
		return 0
	}
	fmt.Fprintf(stdout, "applied %d record(s)\nhash: %s\nround: %s\n", len(edits), res.Hash, roundPath)
	return 0
}

func tagOrDefault(tag string) string {
	if tag == "" {
		return "default"
	}
	return tag
}
