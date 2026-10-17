package pairlog

import (
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type persistFunc func(string, []byte, time.Time, string) error

// RunCLI implements the streaming `pair session-log append` preparation route.
func RunCLI(args []string, stdin io.Reader, getenv func(string) string, now time.Time, stderr io.Writer) int {
	return runCLI(args, stdin, getenv, now, stderr, func(path string, body []byte, now time.Time, appendID string) error {
		return managedLogWrite(getenv, path, func(store SessionLogStore) error { return store.PrepareWithID(path, body, now, appendID) })
	})
}

// PeerAppendIDPrefix marks the entry pair-wrap records for a submitted Couch
// peer message (pair#618): the agent took that turn, so round correlation must
// see it, but the operator didn't write it, so the draft pane's history skips
// it (nvim/pairlog.lua names the same prefix).
const PeerAppendIDPrefix = "couch-peer-"

// PersistPeerTurn records one submitted peer message's envelope through the
// same retention-managed, locked writer as `pair session-log append` and
// `commit`; the delivery ID makes a repeat a no-op error, never a duplicate.
func PersistPeerTurn(getenv func(string) string, path string, envelope []byte, now time.Time, deliveryID string) error {
	appendID := PeerAppendIDPrefix + deliveryID
	return managedLogWrite(getenv, path, func(store SessionLogStore) error { return store.PersistWithID(path, envelope, now, appendID) })
}

// RunCommitCLI implements `pair session-log commit` after input dispatch.
func RunCommitCLI(args []string, getenv func(string) string, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "--append-id" || !sessioninventory.ValidPairLogAppendID(args[1]) {
		_, _ = fmt.Fprintln(stderr, "usage: pair session-log commit --append-id ID")
		return 2
	}
	path := getenv("PAIR_LOG_PATH")
	if path == "" {
		_, _ = fmt.Fprintln(stderr, "pair session-log commit: PAIR_LOG_PATH is unset")
		return 2
	}
	err := managedLogWrite(getenv, path, func(store SessionLogStore) error { return store.CommitID(path, args[1]) })
	if err == nil || commitoutcome.Of(err) == commitoutcome.Committed {
		return 0
	}
	_, _ = fmt.Fprintf(stderr, "pair session-log commit: %v\n", err)
	return 1
}

func runCLI(args []string, stdin io.Reader, getenv func(string) string, now time.Time, stderr io.Writer, persist persistFunc) int {
	if len(args) != 2 || args[0] != "--append-id" || !sessioninventory.ValidPairLogAppendID(args[1]) {
		_, _ = fmt.Fprintln(stderr, "usage: pair session-log append --append-id ID")
		return 2
	}
	path := getenv("PAIR_LOG_PATH")
	if path == "" {
		_, _ = fmt.Fprintln(stderr, "pair session-log append: PAIR_LOG_PATH is unset")
		return 2
	}
	body, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "pair session-log append: read stdin: %v\n", err)
		return 1
	}
	if err := persist(path, body, now, args[1]); err != nil {
		if commitoutcome.Of(err) == commitoutcome.Committed {
			_, _ = fmt.Fprintf(stderr, "pair session-log append: committed with cleanup warning: %v\n", err)
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "pair session-log append: %v\n", err)
		return 1
	}
	return 0
}
