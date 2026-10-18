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
	Warning string                `json:",omitempty"`
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
		if err := json.NewEncoder(stdout).Encode(applyResponse{Code: "ok", Hash: res.Hash, Round: res.Round, Body: res.Body, Warning: res.Warning}); err != nil {
			return fail(err)
		}
		return 0
	}
	fmt.Fprintf(stdout, "applied %d record(s)\nhash: %s\nround: %s\n", len(edits), res.Hash, roundPath)
	if res.Warning != "" {
		fmt.Fprintln(stderr, "pair review apply: warning:", res.Warning)
	}
	return 0
}

func tagOrDefault(tag string) string {
	if tag == "" {
		return "default"
	}
	return tag
}

// roundResponse is `pair review round FILE --json`: the last applied round
// of FILE and its commit body, and whether the file still is that round's
// result. It is how an agent recovers a lost apply reply: commit Body when
// Current, else re-read and rebuild.
type roundResponse struct {
	Code    string
	Error   string
	Round   *sessionsync.Round
	Body    string
	Current bool
}

// RunRoundCLI is `pair review round FILE --json`. Round is null when no
// round of FILE is recorded (none yet, or the last one was another file's).
func RunRoundCLI(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	var files []string
	asJSON := false
	for _, a := range args {
		if a == "--json" {
			asJSON = true
		} else {
			files = append(files, a)
		}
	}
	if !asJSON || len(files) != 1 {
		fmt.Fprintln(stderr, "usage: pair review round FILE --json")
		return 2
	}
	reply := func(res roundResponse, exit int) int {
		_ = json.NewEncoder(stdout).Encode(res)
		return exit
	}
	fail := func(err error) int {
		res := roundResponse{Code: "error", Error: err.Error()}
		var r *sessionsync.Refusal
		if errors.As(err, &r) {
			res.Code = r.Code
		}
		return reply(res, 1)
	}
	roundPath, err := resolveRoundPath(getenv)
	if err != nil {
		return fail(err)
	}
	file, err := filepath.Abs(files[0])
	if err != nil {
		return fail(err)
	}
	artifact, err := sessionsync.ReadRoundArtifact(roundPath)
	if err != nil {
		return fail(err)
	}
	if artifact == nil || !sameFile(artifact.File, file) {
		return reply(roundResponse{Code: "ok"}, 0)
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return fail(err)
	}
	return reply(roundResponse{Code: "ok", Round: &artifact.Round, Body: artifact.Body,
		Current: sessionsync.Hash(string(raw)) == artifact.Round.ResultHash}, 0)
}

func sameFile(a, b string) bool {
	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return ra == rb
}

// resolveRoundPath is $PAIR_REVIEW_ROUND_PATH, else the session's scoped one.
func resolveRoundPath(getenv func(string) string) (string, error) {
	if p := getenv("PAIR_REVIEW_ROUND_PATH"); p != "" {
		return p, nil
	}
	dataDir := getenv("PAIR_DATA_DIR")
	if dataDir == "" {
		return "", errors.New("PAIR_DATA_DIR not set")
	}
	paths, err := artifactpath.ResolveScoped(dataDir, tagOrDefault(getenv("PAIR_TAG")))
	if err != nil {
		return "", fmt.Errorf("resolve artifact namespace: %w", err)
	}
	return paths.ReviewRound(), nil
}
