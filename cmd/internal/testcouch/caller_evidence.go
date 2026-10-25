package testcouch

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// CaptureCallerEvidence observes the frozen genuine public Actors binding.
// Every native read is later evidence, never the frame judged by authorization.
// It neither retries a mutation nor uses another actor to read its receipt.
func (f *Fleet) CaptureCallerEvidence(ctx context.Context, caller couchmessage.Binding) string {
	return f.captureCallerEvidence(ctx, caller, nil, func(q context.Context) (string, error) {
		return f.runStartupQuery(q, "", "list-sessions", "--no-formatting")
	})
}

func (f *Fleet) captureCallerEvidence(ctx context.Context, caller couchmessage.Binding, dump startupDump, listing func(context.Context) (string, error)) string {
	if err := caller.Validate(); err != nil {
		return fmt.Sprintf("caller evidence unavailable: %v", err)
	}
	parent, bounded := ctx.Deadline()
	if !bounded {
		return "caller evidence unavailable: bounded context required"
	}
	deadline, err := startupDiagnosticDeadline(time.Now(), parent)
	if err != nil {
		return fmt.Sprintf("caller evidence unavailable: %v", err)
	}
	query, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	var out strings.Builder
	binding, _ := json.Marshal(caller)
	fmt.Fprintf(&out, "later caller observation; not judged authorization read; observed_at=%s diagnostic_deadline=%s frozen_binding=%s\n", time.Now().UTC().Format(time.RFC3339Nano), deadline.UTC().Format(time.RFC3339Nano), binding)
	// A single selected PID check is not a process census. Unknown generation
	// stays explicit and does not prove the caller absent.
	generation := procutil.Identity(strconv.Itoa(caller.PID))
	fmt.Fprintf(&out, "caller_pid=%d observed_generation=%q generation_available=%t matches_frozen=%t\n", caller.PID, generation, generation != "", generation != "" && generation == caller.Start)
	r, err := openEvidenceRoot(f.Roots.Root)
	if err != nil {
		fmt.Fprintf(&out, "fixture identity unavailable: %v\n", err)
	} else {
		// Read only allowlisted fields; argv and fixture transcript are not copied.
		dir, e := f.actorDirectory(0)
		if e != nil {
			fmt.Fprintf(&out, "fixture identity unavailable: %v\n", e)
		} else {
			raw, cut, e := r.read(filepath.Join(dir, "ready.json"), agentfixture.MaxReadyBytes, false)
			var ready struct {
				RunID      string `json:"run_id"`
				Agent      string `json:"agent"`
				PID        int    `json:"pid"`
				Generation string `json:"generation"`
				CWD        string `json:"cwd"`
			}
			if e == nil && !cut {
				e = json.Unmarshal(raw, &ready)
			}
			if e != nil || cut {
				fmt.Fprintf(&out, "fixture identity unavailable: truncated=%t read_error=%v\n", cut, e)
			} else {
				value, _ := json.Marshal(ready)
				fmt.Fprintf(&out, "later_fixture_identity=%s matches_owned=%t\n", value, ready.RunID == f.RunID && ready.CWD == f.Workspaces.Primary && ready.Agent == caller.Agent)
			}
		}
		r.root.Close()
	}
	out.WriteString(f.captureStartupEvidenceForCaller(query, dump, &caller))
	observed := time.Now().UTC()
	if listing == nil {
		fmt.Fprintln(&out, "later_native_listing unavailable: no observer")
	} else if query.Err() != nil {
		fmt.Fprintf(&out, "later_native_listing unavailable: shared diagnostic budget: %v\n", query.Err())
	} else {
		text, e := listing(query)
		// The adapter already caps retained stdout; this hash/count describes only
		// these retained bytes. Adapter truncation remains visible in query_error.
		cut := len(text) > startupSourceBytes
		if cut {
			text = text[:startupSourceBytes]
		}
		fmt.Fprintf(&out, "later_native_listing started_at=%s completed_at=%s bytes=%d sha256=%x hash_scope=retained_bytes truncated=%t query_error=%v output=%q\n", observed.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), len(text), sha256.Sum256([]byte(text)), cut, e, text)
	}
	return out.String()
}
