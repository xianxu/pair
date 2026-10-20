---
id: 000714
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b5bdac4c26241aac08a84e98807cf44829a2cb4b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T21:04:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Cold-start recovery gaps: a failed resume wedges the slot in starting; post-reboot candidate scans incomplete; a codex slot lost its candidate

## Problem

First real cold start (10-19 ~15:55 hard crash, full machine reboot, 11 working slots down). Recovering the fleet by
`couch --resume-candidates` + `couch --resume` / `--reboot` per slot worked for 9 of 11 on the first try, but surfaced
four gaps:

1. **A failed resume wedges the slot in "starting".** `couch --resume ariadne:2` failed with "Pair exited before
   registering (exit 1)" and recorded no diagnostics (`failures.json` has no entry for nonce start-f65945f19dabb48e).
   Every retry afterwards refused for several minutes with "resume-starting: source is still starting; wait for its
   bounded launch … diagnostics could not be saved: invalid launch failure identity". Only `--reboot` (fresh, losing
   the conversation) got out.
2. **Every candidate scan was incomplete after the reboot:** each slot printed "last activity unknown", "current
   recovery witness names another file generation", "Conversation evidence unavailable", "Candidate scan incomplete".
   Resume still worked, so the warnings were noise and gave the TL nothing to judge by.
3. **pair:3 had no resume candidate at all** (codex, issue pair#712, its suite running at the crash): only "Start
   fresh" was offered, though its transcript should be on disk like the other codex slots'.
4. **No fleet view:** the TL ran 11 `--resume-candidates` calls and joined them with sdlc claims by hand (the "resume
   the fleet" verb is ariadne#456's scope; this issue only needs the per-slot outputs to be trustworthy).

## Spec

Proposed design, awaiting TL approval: [durable plan](../plans/000714-cold-start-recovery-plan.md).

First distinguish confirmed failure from uncertainty at the existing recovery launch boundary. Under the per-thread operation hold, an owned helper that has completed plus an absent exact proposed terminal permits a nonce/revision-fenced terminal failure transition. Retain the selection/context, park and thread while removing only the failed creating incarnation. Present/unknown terminal or unknown helper evidence preserves the occupied claim. Optional diagnostics must never decide that disposition.

Compose diagnostic identities atomically: a partial outer address must not replace the captured complete launch address. Keep strict failure codecs and one terminal publisher; unresolved refusals use path/reference evidence. Test the actual repo-scope-only resume dispatch and immediate second resume.

Screen availability and native conversation evidence are independent. Keep existing screen-generation checks and scan budgets. Preserve readable descriptive activity with accurate source labels and precise warnings. Add explicit, unconfirmed native candidates when no usable screen exists, using a versioned native-only snapshot through the existing inferred-selection route, exact artifact fingerprints, complete bounded scan, revision/digest-bound explicit choice and independent exec revalidation. Never establish a binding or select automatically from cwd/time. Proposed native-only selection lifetime: five minutes; TL/operator approval required for this state/authority extension.

## Done when

- A failed resume/launch records its diagnostics and leaves the slot retryable (a test: a launch that exits before
  registering, then a second `--resume` that proceeds instead of refusing as "starting").
- After a reboot, a resumable slot's candidate shows its last activity and no spurious generation/incomplete warning,
  or the warning names what is actually missing (a test from a post-reboot fixture).
- pair:3's missing candidate is explained (root cause in the Log) and fixed, or shown to be a real loss.

## Plan

- [ ] M1 — Confirmed failed launch disposition, immediate retry and durable diagnostics.
- [ ] M2 — Separate screen/native evidence and present explicit unbound candidates.
- [ ] M3 — Validate native-only authority through exec, qualify restart behavior and land.

## Log

### 2026-10-19

- 10-19 17:30 TL root-cause read (read-only, from records + source):
  - **pair:3 (gap 3): never bound, not lost.** Binding rows live in `~/.local/share/pair/repos/<scope>/ledger-<tag>.jsonl`
    (`kind:"binding"`, `confirmation_reason:"correlation"`). pair:6's ledger-1-pair-59 has them; pair:3's
    ledger-1-pair-58 has only the 10-16 15:02 launch row. sessionwatch (`cmd/internal/sessionwatch/run.go` ~232,
    send gate #420) correlates turns from `log-<tag>.md` after the launch offset; `readPairLog` treats a missing log
    as no turns, so it never binds. pair:3 had no log-1-pair-58.md. The codex rollout exists
    (`~/.codex/sessions/2026/10/16/rollout-2026-10-16T15-02-33-01a146bc-….jsonl`, cwd pair-slot3).
    `ResumeCandidates` (couchcore/recovery_preview.go:229) offers resume only via `ResolveEstablished`. Open: why
    no Pair-log turn in 3 days. Fix direction: bind by rollout cwd + launch time as a weaker candidate rather than
    offering only fresh.
  - **ariadne:2 (gap 1): failure path leaves the start claim.** `launch_existing.go:235-239` (unconfirmed) calls
    `fail(h, cause, nil)` with no rollback; the archived record kept `state:"creating"`, nonce
    start-f65945f19dabb48e, owner_pid = the live supervisor. `resume_route.go:171` refuses while `DecideStaleStart`
    (actionableinventory.go:700) is false; with a live owner only the 5-min age releases it.
  - **Diagnostics lost:** `couch --resume` sets repo-scope but no tag; `lifecycleFailureTargetForCall`
    (lifecyclefailure_record.go:54) builds an empty-Tag address that overwrites the launch's full address;
    `normalizeLaunchFailure` (launchfailure.go:95) rejects it ("invalid launch failure identity"). Also hit the first
    exit-1, whose message still claimed diagnostics went to failures.json (last written 15:49). Exit-1 cause unknown.
  - **Possible 4th:** supervision.jsonl has had no write since 15:56 though the post-reboot supervisor is its writer.

### 2026-10-19 — slot 3 planning

- Claimed and started #714 on its own main-based branch after the #715 landing. Read current launch/failure/recovery owners; no implementation changes. #699 workflow files are clean and remain outside this item.
- Confirmed current source paths from the TL read: recovery's unconfirmed tail retains every post-ack claim; outer diagnostic enrichment treats a repo-scope-only address as complete. Existing helper-exited uncertainty tests deliberately retain claims when observation is unavailable, so an unconditional rollback would violate the current safety contract (ARCH-ORDER).
- Read-only metadata for old pair:3 proves a device-ID mismatch: witness device 16777232, current stat device 16777230; raw inode 731744298 and events inode 731744297 match their witness values. This does not prove durable volume identity and is not a reason to ignore generation checks. Old raw capture size is 381260612 bytes, also beyond the 256 MiB recovery-read budget. Native discovery must not depend on accepting that screen (ARCH-SECURE, ARCH-CONSTRAINTS).
- A bounded read-only subagent traced existing candidate discovery: lack of established binding is not by itself why all inferred choices disappear; usable screen/witness and a complete scan currently gate them. The old missing Pair log explains absent causal binding, but its underlying capture/logging cause remains unproved. A native-only explicit candidate variant is proposed rather than changing watcher binding rules (ARCH-DRY, ARCH-PURPOSE).
- Durable plan written with state/action/loss table, terminal event enumeration, counted growth tests, v1/v2 compatibility boundary and three review milestones. Waiting for hub approval before change-code or implementation.

## Revisions

- 2026-10-19: Initial design separates definite failed launches from uncertain outcomes and proposes independent native-only candidate evidence. Reason: current on-disk device mismatch plus an oversized historical screen prevents safe reuse of the capture-backed inference contract; old acceptance criteria remain in scope.
