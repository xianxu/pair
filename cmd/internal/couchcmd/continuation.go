package couchcmd

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
)

// The writer's working directory can be a sibling worktree. Only the exact
// inherited Pair/Couch address and launch generation identify its live source.
func bindContinuationEnvironment(args map[string]string, getenv func(string) string) error {
	scope, tag := getenv("COUCH_THREAD_SCOPE"), getenv("COUCH_THREAD_TAG")
	session, agent := getenv("PAIR_SESSION_NAME"), getenv("PAIR_AGENT")
	ordinal := getenv("PAIR_LAUNCH_ORDINAL")
	if scope == "" || tag == "" || scope != getenv("PAIR_SCOPE_KEY") || tag != getenv("PAIR_TAG") {
		return fmt.Errorf("continuation requires matching Pair and Couch thread addresses")
	}
	if session == "" || session != getenv("ZELLIJ_SESSION_NAME") || agent == "" {
		return fmt.Errorf("continuation requires the exact live Pair pane session and agent")
	}
	n, err := strconv.ParseUint(ordinal, 10, 64)
	if err != nil || n == 0 {
		return fmt.Errorf("continuation requires a positive Pair launch ordinal")
	}
	digest := getenv(checkpoint.DigestEnv)
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 || digest != strings.ToLower(digest) {
		return fmt.Errorf("continuation requires the committed checkpoint digest")
	}
	args["repo-scope"], args["tag"] = scope, tag
	args["agent"], args["session"], args["launch-ordinal"] = agent, session, ordinal
	args["expected-digest"] = digest
	return nil
}
