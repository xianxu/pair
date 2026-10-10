package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// slotTailReader is Couch's SlotTail: the broker of the Couch whose store is
// storeDir.
func slotTailReader(storeDir string) couchcore.SlotTailReader {
	return func(ctx context.Context, address couchcore.ThreadAddress, maxLines int) (couchcore.TerminalTail, error) {
		return readSlotTail(ctx, couchmessage.Call, storeDir, address, maxLines)
	}
}

// readSlotTail asks the running Couch's broker for a thread's live tail
// (pair#425), which it reads from the thread's wrapper. Peek works from any
// shell, so the request carries no slot identity, like --broadcast-list.
func readSlotTail(ctx context.Context, call messageCall, storeDir string, address couchcore.ThreadAddress, maxLines int) (couchcore.TerminalTail, error) {
	tail, _, err := askTail(ctx, call, storeDir, couchmessage.Request{Op: "tail", TailScope: address.RepoScope, TailTag: string(address.Tag), Lines: maxLines})
	return tail, err
}

// askTail sends one tail request -- naming its thread by scope and tag, or by
// slot (pair#429) -- and returns the tail and the thread it was read from.
func askTail(ctx context.Context, call messageCall, storeDir string, request couchmessage.Request) (couchcore.TerminalTail, couchmessage.TailThread, error) {
	socket, err := couchmessage.SocketPath(storeDir, "broker")
	if err != nil {
		return couchcore.TerminalTail{}, couchmessage.TailThread{}, err
	}
	// More lines than a live tail holds is refused, not clamped: the peek
	// then answers from the recording, which honours the count, and names why.
	if err := couchmessage.ValidTailLines(request.Lines); err != nil {
		return couchcore.TerminalTail{}, couchmessage.TailThread{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err := call(ctx, socket, request, &result); err != nil {
		return couchcore.TerminalTail{}, couchmessage.TailThread{}, fmt.Errorf("no running couch answered (%v)", err)
	}
	switch {
	// A couch older than this request refuses its fields, or the request with
	// its generic identity check or as an unknown operation.
	case result.Code == "invalid-request" && (strings.Contains(result.Error, "unknown field") || strings.Contains(result.Error, "unknown message operation") ||
		strings.Contains(result.Error, "requires the calling conversation identity")):
		return couchcore.TerminalTail{}, couchmessage.TailThread{}, errors.New("the running Couch predates live tails; restart Couch to use them")
	case result.Code != "ok":
		return couchcore.TerminalTail{}, couchmessage.TailThread{}, fmt.Errorf("%s: %s", result.Code, result.Error)
	case result.Tail == nil:
		return couchcore.TerminalTail{}, couchmessage.TailThread{}, errors.New("the running Couch returned no tail")
	}
	tail := couchcore.TerminalTail{Lines: result.Tail.Lines, Truncated: result.Tail.Truncated}
	if result.Tail.Cursor != nil {
		tail.Cursor = result.Tail.Cursor.String()
	}
	var thread couchmessage.TailThread
	if result.TailThread != nil {
		thread = *result.TailThread
	}
	return tail, thread, nil
}
