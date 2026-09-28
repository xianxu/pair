---
id: '000280'
status: done
started: 2026-09-17T21:46:16-07:00
created: 2026-09-17
updated: 2026-09-17
estimate_hours: 2.22
actual_hours: 0.96
---

# A retained continuation failure masks a live thread's state in the switcher

## Problem

Operator report, 2026-09-17. The `pair` thread is **healthy and in active use**,
and the switcher renders it:

```
  brain        /Users/xianxu/workspace/brain        live
  tools        /Users/xianxu/workspace/tools        live
  parley.nvim  /Users/xianxu/workspace/parley.nvim  live
  ariadne      /Users/xianxu/workspace/ariadne      live
▸ pair         /Users/xianxu/workspace/pair         continuation failed — retry available
  astro        /Users/xianxu/workspace/astro        session gone
```

Couch itself classifies it live, at the same moment:

```
$ couch --show pair
pair                   /Users/xianxu/workspace/pair
  address: e108517d46ab4575/couch-c945633f5c806f21
  recorded: live     pid 80199
  live
```

So the state is right everywhere except the surface the operator actually reads.

### Cause — an unconditional precedence, one line up from the state

`couchtty/menu_render.go:414`, `rootStateText`:

```go
func rootStateText(thread couchcore.ActionableThreadSummary, now time.Time) string {
	if request := thread.Continuation; request != nil {
		switch request.Phase {
		case checkpoint.Pending:  return "continuation queued"
		case checkpoint.Running:  return "continuing…"
		case checkpoint.Failed:   return "continuation failed — retry available"
		}
	}
	switch thread.State { … }
}
```

The continuation request is consulted **before** `thread.State` and returns
outright, so any retained request shadows the row's real state for as long as it
is retained. The thread does not have to be unusable, parked or even idle — this
one is live and being typed into.

The retention itself is correct and deliberate. `pair#249` made a failed or
unconfirmed continuation survivable on purpose: *"Acceptance is not completion.
Registration proves a fresh target exists; `complete` requires the matching
orientation `submitted` receipt. A failed, canceled, or unconfirmed delivery
remains recoverable, and text may already be present in the target."* What was
never bounded is how long that fact outranks everything else the row could say.
This thread's request most likely dates from the `#256` M2 close continuation
(`731f99b7`), whose restart worked — the conversation continued — while the
`submitted` receipt never landed, so the request sits in `Failed` indefinitely.

**This is the same external/internal mismatch family as `pair#272` and
`pair#275`, from the other end:** not a stale liveness witness, but a stale
*request* outliving the situation it described.
