package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// fastPeek answers `couch --peek` from the running Couch alone (pair#429):
// every reference is a slot (repo:N), and the broker maps each to its
// connected wrapper and reads its tail, one round trip per slot, all at once.
// Nothing builds a Couch, which is what made a peek take seconds. It answers
// only when every slot's live tail does; otherwise -- no Couch, an older one,
// a reference that is not a slot, a slot that cannot be read, or
// --transcripts -- it reports false and the typed peek runs, which names
// each failure and falls back to the recording.
func fastPeek(inv cliInvocation, storeDir string, stdout io.Writer, call messageCall) bool {
	lines, asJSON := couchcore.DefaultPeekLines, false
	for _, arg := range inv.args {
		switch {
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "--lines="):
			n, err := strconv.Atoi(strings.TrimPrefix(arg, "--lines="))
			if err != nil || couchmessage.ValidTailLines(n) != nil {
				return false
			}
			lines = n
		default:
			return false
		}
	}
	refs, err := couchcore.ExpandPeekReferences(inv.ref)
	if err != nil {
		return false
	}
	for _, ref := range refs {
		if parsed, recognized, err := couchcore.ParseWorkspaceReference(ref); err != nil || !recognized || parsed.Repo == "" {
			return false
		}
	}
	results := make([]couchcore.PeekResult, len(refs))
	failed := make([]bool, len(refs))
	var wg sync.WaitGroup
	for i, ref := range refs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tail, thread, err := askTail(context.Background(), call, storeDir, couchmessage.Request{Op: "tail", Target: ref, Lines: lines})
			if err != nil {
				failed[i] = true
				return
			}
			results[i] = couchcore.PeekResult{Ref: ref, Tag: thread.Tag, Agent: thread.Agent, Lines: tail.Lines,
				Source: "live", Cursor: tail.Cursor, Truncated: tail.Truncated}
		}()
	}
	wg.Wait()
	for _, f := range failed {
		if f {
			return false
		}
	}
	for i := range results {
		if results[i].Lines == nil {
			results[i].Lines = []string{}
		}
	}
	var result any = couchcore.PeekSnapshot{Slots: results}
	if len(results) == 1 {
		result = results[0]
	}
	// Rendered whole before writing, so a false return never leaves half an
	// answer ahead of the typed peek's.
	var out bytes.Buffer
	if asJSON {
		if json.NewEncoder(&out).Encode(result) != nil {
			return false
		}
	} else {
		op, _ := Resolve("peek")
		if render(&out, op, result) != 0 {
			return false
		}
	}
	_, _ = stdout.Write(out.Bytes())
	return true
}
