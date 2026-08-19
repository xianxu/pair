---
id: 000144
status: working
deps: []
github_issue:
created: 2026-08-19
updated: 2026-08-19
estimate_hours:
started: 2026-08-19T07:09:43-07:00
---

# Reject Codex subagent sessions during Pair identity discovery

## Problem

Pair identifies a live Codex conversation from rollout filenames exposed by
the agent process tree. Codex subagents write rollout files in the same
directory and may be open in the same process tree, so filename-only discovery
can persist a subagent ID. `Alt+n` then gives that ID precedence over the saved
config and resumes the subagent rather than the operator's root conversation.

The same ambiguity exists in the asynchronous session watcher's birth-time
fallback. Issue #143 extends that watcher for the lifetime of an agent, making
it more likely to observe later subagent rollouts when the root session has not
already been captured.

## Spec

Define Codex root-session identity once in `cmd/internal/transcript`: a rollout
is eligible only when its filename contains a valid session UUID and its first
JSONL event is a coherent `session_meta` record for that same UUID. An accepted
record has `type: "session_meta"`, `payload.id` equal to the filename UUID,
`payload.parent_thread_id` absent or null, and the currently observed root
`payload.source` string `"cli"` or `"exec"`. Reject the observed subagent shape
(a non-null parent plus an object-valued source containing `subagent`) and fail
closed on unreadable, malformed, incomplete, mismatched, or unknown-source
metadata. A rejected candidate does not end a scan; later candidates remain
eligible.

Apply the classifier anywhere Pair turns an open or newly-created Codex rollout
into the active conversation identity: launcher `Alt+n` capture, asynchronous
session watching (both `lsof` and birth-time discovery), the shared
`codexsid` resolver used by review targeting, and live slug transcript
resolution. Remove Neovim's independent `ps`/`lsof` filename parser for review
target scoping; it must derive from `PAIR_SESSION_ID` or the Go-authored saved
config rather than restating Codex identity rules in Lua. Path-shape extraction
remains a lower-level helper only; it must not by itself authorize a resumable
session ID. This is the shadow-sweep for ARCH-PURPOSE and keeps one
classification contract under ARCH-DRY.

Treat persisted Codex IDs as untrusted at automatic resume boundaries. Before
the config picker or `Alt+n` fallback composes a saved ID, resolve its rollout
and apply the same root classifier. Warn, clear the invalid ID from automatic
selection, and remove the polluted config so Neovim cannot consume it; preserve
the saved non-resume args for a fresh launch. Ledger fallback remains subject
to the same validation. An explicitly typed Codex `resume <id>` remains user
authority and is outside automatic discovery.

Keep filesystem and process discovery at the existing integration seams. The
metadata decision is a pure function over a path and first JSONL event; a thin
transcript adapter reads only the first JSONL event, and candidate scanners
continue until that classifier accepts a root (ARCH-PURE). Tests pass root and
subagent files through the shared selector using temporary rollout trees, then
exercise each consumer with its existing stateful runtime/process seam
(ARCH-MOCK).

Alternatives rejected:

- PID ancestry cannot distinguish root and subagent sessions because Codex may
  host both inside the same process tree.
- Choosing the oldest or newest rollout merely changes which concurrent
  session wins and cannot establish semantic identity.

## Done when

- Given an open root rollout and an open subagent rollout, `Alt+n` captures the
  root session regardless of rollout order.
- Session watcher `lsof` and birth-time discovery ignore Codex subagent
  rollouts and persist the root ID when it becomes available.
- Review-target and slug live-session resolution use the same root-only
  classifier; Neovim no longer parses live rollout filenames independently.
- A saved config containing a subagent ID is quarantined before config-picker
  or `Alt+n` automatic resume. If no valid live root is available, Pair starts a
  fresh session with the saved non-resume args instead of resuming the
  subagent.
- Malformed, incomplete, mismatched, or explicitly nested `session_meta`
  records, plus unknown source shapes, do not authorize a session ID; scanners
  still find a later valid root candidate.
- Focused and repository-wide automated tests pass.

## Plan

- [ ] Add the pure Codex root-session metadata classifier and exhaustive unit
  tests for root, subagent, malformed, incomplete, and mismatched events.
- [ ] Route every live Codex identity consumer through the shared classifier,
  with regressions for ambiguous root/subagent candidates.
- [ ] Validate and quarantine persisted Codex IDs at automatic resume
  boundaries, and remove Neovim's independent live filename parser.
- [ ] Verify focused packages and the full repository; update the session
  identity atlas if its current map omits root-vs-subagent semantics.

## Log

### 2026-08-19

- Root cause evidence: the saved config contained session
  `01a017b6-af00-7c91-a656-0611a3750669`; that rollout's first event declares
  parent thread `01a016d8-0a53-72e2-a62a-456c0c72f1a2` and a depth-1 subagent
  source. The live root rollout instead had a null parent and CLI source.
- ARCH-DRY/ARCH-PURPOSE: filename-to-ID logic currently exists in transcript,
  sessionwatch, and codexsid, while launcher and slug consume the filename-only
  result. The fix must centralize semantic authorization and cover every live
  identity consumer rather than patching only `Alt+n`.

## Revisions

### 2026-08-19 07:18 PDT — Fresh-context spec review

- Added Neovim's review-target `ps`/`lsof` scanner to the consumer sweep and
  specified removing that duplicate in favor of validated Go-authored state.
- Added quarantine semantics for already-polluted config/ledger IDs so a failed
  live lookup cannot fall back to the same subagent.
- Defined accepted root metadata (`cli`/`exec`, null parent, matching ID),
  fail-closed unknown shapes, scan continuation, and the pure classifier/thin
  first-event IO seam.
