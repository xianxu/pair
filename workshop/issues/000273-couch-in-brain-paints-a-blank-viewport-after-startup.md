---
id: 000273
status: open
deps: [pair#265]
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# couch in brain paints a blank viewport after startup

## Problem

Split out of `pair#265`, whose `## Spec` bullet 4 and `## Done when` #2 were
narrowed on 2026-09-16 to the routing crash. This issue owns the half that is
not yet attributed.

Operator report: `couch` in `/Users/xianxu/workspace/brain` starts, the tab bar
appears, and the main viewport stays fully blank. couch then sits there. A
keypress at that point printed a raw escape sequence at the top-left instead of
being handled. Pressing `ctrl+space` exited couch with
`couch: terminal: terminal: no admitted endpoint` — **that exit is `pair#265`
and is not this issue**; this issue is the blank screen before it, and the
escape-sequence leak.

What the `pair#265` inspection established, which narrows this considerably:

- The console **is** running when the blank viewport shows. `beginConsole`
  dispatches the initial attach *before* `Console.Run`, and an attach failure
  returns an error that `runConsole` prints and exits on (`run.go`), so a
  visible tab bar with no exit means the attach committed and `Run` took the
  `c.switchTo(initial, …)` branch. The blank area is therefore a **painted
  frame of a child that has produced no output**, not an unpainted screen.
- It is **not** couch refusing to spawn. `brain` holds no resumable thread (3
  stale + 1 wedged park, `pair#271`), so startup goes through `spawnResolved`
  and mints a fresh thread. The three stale records dated 09:02, 09:29 and 10:00
  on 2026-09-16 are one per attempt, each with `last_active_at` at the zero time
  — spawned, never became active.
- **The fresh-spawn path is not broken in general.** `pair` also holds no
  resumable thread (5 stale + 1 binding-lost) and couch works there. So whatever
  this is, it is not "couch cannot start a new thread".

Open questions, in the order worth asking:

- Does the spawned child ever write? `brain`'s remembered agent is `muse`
  (`last_agent: muse`, argv `["--trust-workspace"]`) while `pair`'s is `codex` —
  but `pair` has run `muse` too (three live `muse` agents in its scope), so the
  agent alone does not explain it.
- Is the child alive and silent, or dead? A dead child fires `onExit`, and a
  last-actor exit with an actor focused ends the console — which is *not* what
  the operator sees, so "alive and silent" is the likelier branch.
- What is the escape sequence at the top-left, and who wrote it — the parent
  (a reply couch failed to consume) or the child (output painted at the wrong
  origin)?
- How long is "hangs"? `pair#218` measured couch thread startup at 8.85s; a cold
  `muse` in a fresh zellij session could plausibly exceed that without being
  broken.

## Spec

- Attribute the blank viewport before designing anything. One instrumented start
  settles it:

  ```
  COUCH_INPUT_TRACE=/tmp/couch-input.jsonl COUCH_TRACE=/tmp/couch-trace.jsonl couch
  ```

  `COUCH_TRACE` timestamps startup through first frame (`pair#206`);
  `COUCH_INPUT_TRACE` records every byte the parent delivered, which is what
  identifies the leaked escape sequence and whether couch consumed it.
- Distinguish the three candidate layers before proposing a fix: the agent
  (`muse` not painting), `pair`/zellij (session or layout not coming up), or
  couch (a frame composed but not published, or published to the wrong surface).
  Say which, with evidence, in the `## Log`.
- Whatever the cause, couch should not present an indefinitely blank viewport
  with no indication. A pane that has produced nothing for longer than a stated
  budget needs an operator-visible state — this is the `pair#265` Done-when #2
  clause that moved here, and it stands on its own regardless of the root cause
  (ARCH-CONSTRAINTS: name the budget and the bounded behaviour when exceeded).
- The escape-sequence leak is its own defect and needs its own answer, not a
  side effect of fixing the paint.

## Done when

- The blank viewport is attributed to a named layer with trace evidence in the
  `## Log`.
- `couch` in `brain` either paints its child or shows a coherent placeholder;
  a pane silent past its budget is never indistinguishable from a working one.
- A keypress against such a pane does not leak raw bytes to the screen.
- Regression coverage exists for whichever layer the attribution lands in.

## Plan

- [ ] Capture `COUCH_TRACE` + `COUCH_INPUT_TRACE` for one `couch` start in `brain`.
- [ ] From the trace, determine whether the child wrote anything, and when the first frame was published.
- [ ] Identify the leaked escape sequence and its writer.
- [ ] Attribute to agent / pair+zellij / couch, and record the evidence.
- [ ] Fix the attributed layer, or file against it if it is not couch.
- [ ] Give a silent pane a bounded, visible state.

## Log

### 2026-09-16

- Split out of `pair#265` when its Spec was narrowed to the routing crash.
  Background above is what that issue's state-machine inspection established;
  nothing here is attributed yet. `deps: [pair#265]` because the crash makes this
  hard to observe — couch exits as soon as the operator opens the switcher to
  investigate.

## Revisions

### 2026-09-16 — NOT brain-specific: every fresh spawn blanks

Reason: operator reproduced the blank viewport in `parley.nvim` and `tools`
within one sitting. The title and Problem were written when `brain` was the only
known instance. This is the whole issue's premise changing.

The discriminator is **resume vs. spawn**, not which repo:

| thread | how it started | `last_active_at` | outcome |
|---|---|---|---|
| `tools·b3458e3244d72aeb` | resumed from a verified park | 2026-09-15T20:58:35 | worked — operator got in |
| `tools·b94603eb0ae2ab85` | fresh spawn | zero time | blank |
| `parley·64d1422d927996bf` | fresh spawn (after archiving the parked one) | zero time | blank |
| `brain·*` (3 records, 2026-09-16) | fresh spawn | zero time | blank |

**Every fresh spawn carries `last_active_at` at the zero time** — the thread was
created and never became active. Every resume worked. That is a much sharper
statement than "brain is odd", and it means the earlier reasoning that "the
fresh-spawn path is not broken in general, because `pair` works" was wrong: the
operator reaches `pair` through a direct `pair` session, not through couch.

Retitle to match: this is *couch's fresh-spawn path paints a blank viewport*.
`brain` is one instance.

Also observed in the same sitting, and not yet separate issues:

- **Resuming a parked `parley.nvim` thread failed.** A `verified_park` record —
  the healthy, resumable shape — refused to resume. Capture the error text; if
  resume is failing too, the "resume works / spawn blanks" split above is
  narrower than it looks and this issue's discriminator needs re-checking.
- **`archive` returned errors** on some `brain` rows while succeeding on others.
- **`ctrl+return` killed couch**, a second `pair#265` trigger path
  (`HitNewestPage` → `onNewestPageHotkey` → panel), alongside `ctrl+space`.

## Log

### 2026-09-16 (second entry)

- Operator evidence above. Store inspected directly: `tools` went from one
  healthy `verified_park` record to two stale ones; `parley.nvim` from one
  healthy parked record to one stale one. Remaining repos still holding a
  resumable thread: `xianxu.dev`, `ariadne`, `arc-agi-3`.
