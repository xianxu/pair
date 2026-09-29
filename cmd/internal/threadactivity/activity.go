// Package threadactivity is the one definition of when a Pair thread was last
// active (pair#247): the operator's input or its agent's work. Couch's idle
// fading and the title poller's heat ramp both read it, so the two cannot
// disagree about how idle a thread is.
package threadactivity

import (
	"context"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/panebirth"
)

// Thread names one thread's activity sources. ScopeDir is the per-repository
// data directory (artifactpath.ResolveScopeDir), never the global data root.
type Thread struct {
	ScopeDir string
	Scope    string
	Tag      string
	Agent    string
}

// InScope names a thread's sources under Pair's global data root: each thread's
// artifacts live in its own repository's scope directory.
func InScope(dataDir, scope, tag, agent string) (Thread, error) {
	scopeDir, err := artifactpath.ResolveScopeDir(dataDir, scope)
	if err != nil {
		return Thread{}, err
	}
	return Thread{ScopeDir: scopeDir, Scope: scope, Tag: tag, Agent: agent}, nil
}

// Runtime is the IO Latest needs: file mtimes and the thread's bound native
// session.
type Runtime interface {
	ModTime(path string) (time.Time, bool)
	SessionActivity(ctx context.Context, t Thread) (time.Time, bool)
}

// Latest is the newest of three signals, or zero when none resolves:
//
//   - the bound agent transcript's mtime, written on the operator's prompts and
//     on the agent's own work;
//   - the Pair log's mtime, appended only when the operator sends, which covers
//     an agent whose session is not bound yet;
//   - the current launch's pane-birth evidence, rewritten on every real launch
//     (create or resume) and never on attach -- launching is operator activity,
//     and it gives a session nobody has written to yet an age.
//
// The draft is deliberately absent. Its autosave writes on every focus loss
// whether or not anything changed, so its mtime says when the operator last
// LEFT the draft; a thread switch would refresh it. None of the three signals
// moves on redraws, cursor blink or polling.
func Latest(ctx context.Context, rt Runtime, t Thread) time.Time {
	var latest time.Time
	consider := func(m time.Time, ok bool) {
		if ok && m.After(latest) {
			latest = m
		}
	}
	if paths, err := artifactpath.ResolveScoped(t.ScopeDir, t.Tag); err == nil {
		consider(rt.ModTime(paths.Log()))
	}
	if pane, err := panebirth.Evidence(t.ScopeDir, t.Tag, t.Agent); err == nil {
		consider(rt.ModTime(pane))
	}
	if ctx.Err() == nil {
		consider(rt.SessionActivity(ctx, t))
	}
	return latest
}
