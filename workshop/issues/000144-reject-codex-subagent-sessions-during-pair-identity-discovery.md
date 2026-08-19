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
JSONL event is a coherent `session_meta` record for that same UUID. Reject
records whose metadata identifies a parent thread or subagent source, and fail
closed on unreadable, malformed, incomplete, or mismatched metadata. Root
records from the CLI remain eligible.

Apply the classifier anywhere Pair turns an open or newly-created Codex rollout
into the active conversation identity: launcher `Alt+n` capture, asynchronous
session watching (both `lsof` and birth-time discovery), the shared
`codexsid` resolver used by review targeting, and live slug transcript
resolution. Path-shape extraction remains a lower-level helper only; it must
not by itself authorize a resumable session ID. This is the shadow-sweep for
ARCH-PURPOSE and keeps one classification contract under ARCH-DRY.

Keep filesystem and process discovery at the existing integration seams. The
metadata decision is a pure function over a path and first JSONL event
(ARCH-PURE); tests use temporary rollout files or the existing stateful runtime
fakes rather than adding command mocks (ARCH-MOCK).

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
  classifier.
- Malformed, incomplete, mismatched, or explicitly nested `session_meta`
  records do not authorize a session ID.
- Focused and repository-wide automated tests pass.

## Plan

- [ ] Add the pure Codex root-session metadata classifier and exhaustive unit
  tests for root, subagent, malformed, incomplete, and mismatched events.
- [ ] Route every live Codex identity consumer through the shared classifier,
  with regressions for ambiguous root/subagent candidates.
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
