---
id: 000144
status: done
deps: []
github_issue:
created: 2026-08-19
updated: 2026-08-19
estimate_hours: 2.22
started: 2026-08-19T07:09:43-07:00
actual_hours: 0.05
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

Configuration readers that derive transcript or review identity must validate
Codex IDs through the same transcript contract before returning them. This
includes context usage, slug generation, and review-target scoping; a polluted
config behaves as missing identity and may fall through only to a validated
live root. Neovim is the deliberate thin consumer exception: it does not parse
transcripts, and reads only the environment/config state whose Pair launch
boundary has already quarantined invalid automatic identity.

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
  classifier; context usage rejects a polluted config; Neovim no longer parses
  live rollout filenames independently.
- A saved config containing a subagent ID is quarantined before config-picker
  or `Alt+n` automatic resume. If no valid live root is available, Pair starts a
  fresh session with the saved non-resume args instead of resuming the
  subagent.
- Malformed, incomplete, mismatched, or explicitly nested `session_meta`
  records, plus unknown source shapes, do not authorize a session ID; scanners
  still find a later valid root candidate.
- Focused and repository-wide automated tests pass.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                design=0.25 impl=0.10
item: smaller-go-module        design=0.05 impl=0.18
item: cross-cutting-refactor   design=0.15 impl=0.20
item: smaller-go-module        design=0.05 impl=0.18
item: smaller-go-module        design=0.05 impl=0.18
item: lua-neovim               design=0.25 impl=0.24
item: atlas-docs               design=0.02 impl=0.04
item: milestone-review         design=0.00 impl=0.16
design-buffer: 0.15
total: 2.22
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.*

## Plan

- [x] Add the pure Codex root-session metadata classifier and exhaustive unit
  tests for root, subagent, malformed, incomplete, and mismatched events.
- [x] Route every live Codex identity consumer through the shared classifier,
  with regressions for ambiguous root/subagent candidates.
- [x] Validate and quarantine persisted Codex IDs at automatic resume
  boundaries, and remove Neovim's independent live filename parser.
- [x] Verify focused packages and the full repository; update the session
  identity atlas if its current map omits root-vs-subagent semantics.

## Log

### 2026-08-19
- 2026-08-19: closed — After REWORK fixes: full environment-cleared make test passed, including go test ./... and review integration suite; focused launcher/codexsid/slug/sessionwatch tests passed; review-toggle proves no ps/lsof; git diff --check clean; all stale atlas identity descriptions corrected; plan Core concepts reconciled.; review verdict: FIX-THEN-SHIP

- Root cause evidence: the saved config contained session
  `01a017b6-af00-7c91-a656-0611a3750669`; that rollout's first event declares
  parent thread `01a016d8-0a53-72e2-a62a-456c0c72f1a2` and a depth-1 subagent
  source. The live root rollout instead had a null parent and CLI source.
- ARCH-DRY/ARCH-PURPOSE: filename-to-ID logic currently exists in transcript,
  sessionwatch, and codexsid, while launcher and slug consume the filename-only
  result. The fix must centralize semantic authorization and cover every live
  identity consumer rather than patching only `Alt+n`.
- TDD implementation: added a bounded first-event root classifier; routed
  launcher, watcher, context, slug, and review consumers through it; added
  ambiguous subagent/root ordering regressions for process and birth scans.
- Automatic resume now revalidates saved Codex identity in both the config
  picker and the in-process `Alt+n` loop. Invalid IDs quarantine the canonical
  config while saved non-resume flags survive the fresh launch; ledger fallback
  is covered by the same decision.
- Removed Neovim's duplicate process/rollout parser. The review-target headless
  test proves an available live rollout cannot authorize or stamp identity
  without environment/config state.
- Verification: the eight focused identity packages passed; the headless
  review-toggle regression passed; environment-cleared `go test ./... -count=1`
  passed; and the full environment-cleared `make test` passed after updating
  two shell integration fixtures to provide the now-required root metadata.
- Shadow sweep found no package-local or Neovim filename-only authorizer. The
  remaining `CodexSessionIDFromPath` is the transcript classifier's low-level
  path parser; sessionwatch's UUID regex is candidate-shape extraction followed
  by metadata authorization; Neovim's remaining Codex filesystem lookup is an
  age hint for a separately established ID, not identity discovery.
- Updated `atlas/session-identity.md` with root metadata, quarantine, and thin
  Neovim consumer semantics (ARCH-DRY, ARCH-PURPOSE).
- Boundary review returned REWORK: the durable plan misstated the resume
  decision's name/location and called the reused `procutil` seam modified;
  older atlas sections still described filename authority; and requested
  integration assertions were missing. Corrected the plan and all stale maps,
  then added real on-disk subagent rejection, subagent-only launcher/codexsid/
  slug cases, malformed-before-root watcher continuation, and an explicit
  Neovim no-`ps`/`lsof` assertion.
- Re-review returned FIX-THEN-SHIP with behavior, architecture, coverage, and
  atlas all accepted. Removed the generated raw review transcript whose embedded
  patch whitespace alone violated branch-wide `git diff --check`.

## Revisions

### 2026-08-19 07:18 PDT — Fresh-context spec review

- Added Neovim's review-target `ps`/`lsof` scanner to the consumer sweep and
  specified removing that duplicate in favor of validated Go-authored state.
- Added quarantine semantics for already-polluted config/ledger IDs so a failed
  live lookup cannot fall back to the same subagent.
- Defined accepted root metadata (`cli`/`exec`, null parent, matching ID),
  fail-closed unknown shapes, scan continuation, and the pure classifier/thin
  first-event IO seam.

### 2026-08-19 07:29 PDT — Fresh-context plan review

- Expanded persisted-ID validation to context, slug, and review-target config
  readers; polluted configuration is missing identity, never authority.
- Required all Go process discovery implementations to reuse `procutil` for
  `ps`/`lsof` parsing while preserving their injected/stateful test seams.

### 2026-08-19 07:38 PDT — Post-plan estimate

- After the plan cleared fresh-context review, derived 2.22 ship-wall-clock
  hours from eight v3.1 Method A primitives with the thorough-plan 15% design
  buffer and already-scaled implementation values.
