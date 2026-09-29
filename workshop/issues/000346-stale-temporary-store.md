---
id: 000346
status: working
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '413c83b3c173c293b1dffb3b77e0f61cd4f7531b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-29T08:41:28-07:00
---

# Stale temporary store blocks Couch startup

## Problem

After a macOS upgrade/reboot, both interactive `couch` and `couch --list` fail with `registered store ".../scratchpad/store.tYjN" unavailable: lstat /private/tmp/claude-501: no such file or directory`.

The persistent Pair retention registry contained the normal `~/.local/share/pair/couch` store plus a temporary Claude scratchpad store from a Pair slot1 development session. The temporary directory no longer exists; the normal store remains present. The path suggests a development/smoke-test store leaked into the production inventory; the exact originating invocation is not yet verified. `/tmp` resolves to `/private/tmp` on this machine, and registration canonicalizes symlinks.

`storagegc.StoreRegistry.validate` checks every registered path. `RegisterStore` validates the existing inventory before registering even the normal store; `NewCoordinatedThreadStore` invokes registration during construction. One vanished auxiliary store therefore prevents startup and read-only listing. `UnregisterStore` also reads the fully validated registry and requires an existing canonical directory, leaving no normal recovery route for a missing store.

## Spec

Prevent development/test stores from contaminating the operator's durable retention inventory. Trace the source of the scratchpad registration and ensure smoke tests isolate both Couch namespaces and Pair retention roots.

Define supported recovery for vanished registered stores and allow unaffected stores to remain usable where safe. Preserve retention safety: an unavailable store is not proof that it had no references; do not silently enable collection or discard real thread data. Distinguish missing temporary stores from unreadable or temporarily unmounted durable stores and make the recovery action explicit.

Relevant code: `cmd/internal/storagegc/stores.go`, `cmd/internal/couchcore/retention.go`, and `cmd/internal/couchcmd/run.go`. ARCH-SECURE: tests must not write operator state. ARCH-FUNERAL: registration must define the lifecycle and removal of ephemeral stores.

Additional recovery design (2026-09-29): allow intact namespaces to operate despite unrelated unavailable registrations while keeping GC fail-closed; provide explicit permanent-abandonment recovery that resets migration acknowledgment. Revalidate exact proof-named conversations after filesystem metadata changes, using `(agent, native UUID)` as durable identity, with cold-resume boundary tests and bounded catalog reuse. Accept parent-free Codex `vscode` roots and recover existing unbound version-2 launches only from unique completed causal rounds through a supported preview/apply command. Detailed design: `workshop/plans/000346-stale-temporary-store-plan.md` (awaiting approval).

## Done when

- A regression reproduces registration of an auxiliary temporary store, deletion of its directory, and subsequent launch/list behavior for the intact normal store.
- A supported, actionable recovery path handles stale missing registrations without fabricating empty-store proof or weakening garbage-collection safeguards.
- Development/smoke-test invocations isolate all persistent roots; regression coverage proves the operator registry remains untouched.
- Missing, unreadable, and temporarily unavailable durable stores have tested behavior that preserves references and communicates recovery requirements.

- Device/inode/generation changes no longer invalidate a saved surviving conversation by themselves; contradictory internal conversation identities remain refused, and repeat queries do not repeatedly replay the body.
- Codex `vscode` roots bind through the production watcher; existing unbound launches have a safe, tested preview/apply recovery path using recorded causal evidence.
- Every audited row has a verified outcome: validated cold authority, already-live authority, empty slot, or explicit insufficient evidence. No unrelated conversation is selected and brain:0 is not restarted.

## Plan

Detailed plan: `workshop/plans/000346-stale-temporary-store-plan.md` (draft; operator approval required before implementation).

- [ ] M1 — Keep intact stores usable, retain GC safeguards, provide explicit missing-store recovery and isolated smoke coverage.
- [ ] M2 — Revalidate exact saved conversations across device renumbering and verify cold-resume behavior.
- [ ] M3 — Support Codex vscode roots, recover existing unbound launches through causal evidence, verify and document incident recovery.

## Log

### 2026-09-29

Reproduced locally: `couch --list` exited 1 with the same missing-store error as the screenshot. With the retention coordinator's exclusive flock held, backed up `stores.json`, removed only the exact missing scratchpad registration, atomically replaced the registry, and retained the original migration flag and normal store. Backup: `~/.local/share/pair/.retention/stores.json.backup-20260929-072547`. No thread payloads were removed. Afterwards `couch --list` exited 0 and enumerated existing threads. Many report lost session bindings following the reboot; those are separate from the registry startup blocker. This ticket tracks the permanent fix; no implementation changes made.

### 2026-09-29 — Full thread audit after OS upgrade and restart

Operator confirmed this was a macOS upgrade followed by a restart, not merely a reboot. Audited all 23 rows in the screenshot against current `couch --list`, the global threadstore manifest/records, numbered-slot `couch --show` addresses, and the corresponding scoped ledgers. This is an investigation, not a repair: no thread, ledger, proof, transcript, or retention-registry data was modified.

The earlier registry recovery above only made Couch inventory/startup usable. It did not recover cold-resume authority for every thread. There are **15 saved-proof identity mismatches, four current launches without bindings, one live thread also lacking a binding, and three never-started slots**.

#### A. Fifteen saved proofs rejected after device renumbering

Every proof-named transcript below exists, has the same inode as recorded, is at least as large as the recorded proof size, and has a first JSON record naming the saved native session ID. Every recorded device is `16777230`; every current device is `16777232`. These checks establish a common rejection mechanism, not complete transcript validation or authorization to rewrite the proofs. The evidence does not distinguish whether upgrade or reboot caused the renumbering.

| Thread | Agent | UI status | Native session ID | Unchanged inode |
| --- | --- | --- | --- | --- |
| 42shots:0 | qoder | binding lost | `3a5781c6-d861-4926-a067-65e443723b98` | `185512106` |
| ariadne:0 | claude | binding lost | `bee2be25-757b-4027-8573-2bdcaf857203` | `246185815` |
| ariadne:1 | claude | binding lost | `1f48235f-b720-4bcd-b5fe-e162d58f0bec` | `238895656` |
| ariadne:2 | claude | binding lost | `cf3391ab-0ac5-4464-bbb2-be53048558a6` | `240413721` |
| ariadne:3 | claude | binding lost | `b966b671-7eec-484a-93fc-930a54bdbc62` | `208706390` |
| astro:0 | claude | session gone | `f8209a7c-1cce-4325-ac6e-95b88a503560` | `160040825` |
| tools:0 (define) | claude | binding lost | `20d98988-f072-4cba-a88f-51eceeaef985` | `195338098` |
| ducks:0 | claude | binding lost | `cd295436-7ec1-426f-a851-2cdc3ca83b5f` | `233841032` |
| kbench:0 | claude | session gone | `ae4c7a3b-20f5-4dc3-9588-4fbee1370563` | `162494622` |
| nous:0 | claude | binding lost | `286b0136-54a0-40f7-8bd2-60ad0a9f08ca` | `233294735` |
| pair:1 | claude | binding lost | `4acd368e-45c2-452d-a198-49585642c8b0` | `236192803` |
| pair:3 | claude | binding lost | `04f01644-a62a-4c14-b6a8-5fccec140aa8` | `237533351` |
| parley.nvim:1 | claude | binding lost | `614d7266-98f2-4cb4-901b-cb2b908f26f6` | `230764949` |
| parli:0 | claude | binding lost | `61daadf1-9e39-44f3-a55b-fe09544fc7b2` | `166208996` |
| xianxu.dev:0 | claude | binding lost | `75bd6632-bb7b-4095-ab91-9760a61a7dd6` | `153707675` |

Source chain: `sessioninventory/filemeta_darwin.go:platformFileMetadata` builds `StableFileID` from `stat.Dev` and `stat.Ino`; incremental target validation rejects a changed stable ID; `query.go:proofAllowsFullRevalidation` also refuses the mismatch. `QuerySessionContext` leaves failed proofs provisional. `couchcore/actionableinventory.go:ClassifyThread` reports `binding-lost` when a verified park receipt exists, otherwise `session-gone`. Thus **astro and kbench do not demonstrate missing transcript data**: they hit the same proof mismatch but have no verified park receipt. Their UI label must not be interpreted as evidence that their conversation is irrecoverable.

#### B. Missing current-launch bindings

| Thread | Agent | Evidence | Interpretation |
| --- | --- | --- | --- |
| pair:0 | codex | Two typed launches; latest physical ledger line 3; zero binding rows; legacy session ID empty | Cold resolver returns provisional before checking file identity. Device renumbering does not explain the absent binding. |
| parley.nvim:0 | codex | One typed launch at line 2; zero binding rows; legacy session ID empty | Same missing-binding shape as pair:0. |
| pair:2 | codex | Five typed launches across agents; current Codex launch at line 13 has no binding; four historical bindings belong to Claude generations; latest legacy ID empty | Old bindings cannot authorize the current launch. Do not restore an older conversation just because its ID is available. |
| pair:4 | claude | One typed launch at line 2; zero binding rows; legacy session ID `d75674b7-7278-40af-b4d3-62203aaf190f` | No transcript filename matching that legacy ID was found under `~/.claude/projects`; recovery remains unproven. |

All four display `binding lost` and have park receipts. `sessioninventory/query.go:QuerySessionContext` returns provisional for a current launch without a binding before consulting transcript proofs. The two current-launch bindings on ariadne:1 name the same root and are not a conflict. Why the producer failed to commit each missing binding remains a separate investigation; the reboot cannot be assigned as the cause from these ledgers alone. Pair:4 also has an empty Pair send log and only 84 retained wrapper events, so no sent/completed round is evidenced there. Binding publication calls `ObserveAndPersist` after completed-round correlation (`sessionwatch/lifecycle.go`); the retained Codex wrapper events supply no watcher/binding diagnostic that establishes why publication did not occur.

#### C. Live and never-started rows

- **brain:0:** live, recorded PID 41885 at audit time; current Codex thread `2e51fcf9799b1d8f/couch-6b111ea230c149dc`. One typed launch at line 2, no binding rows, empty legacy session ID. Live status proves current execution, not cold resumability. This is an additional instance to cover when investigating Codex binding publication; do not restart it as a diagnostic experiment.
- **brain:1, parli:1, tools:1:** `never started`; `couch --show <slot>` reports no readable current thread. These are empty-slot cases, not evidence of transcript loss. The CLI listing currently mislabels all three as `tools:1`; their distinct working paths and the screenshot identify the intended slots. This rendering discrepancy is separate from proof validation.

#### Recovery/design implications

- Preserve the existing stale-store recovery requirements. Add upgrade/restart recovery coverage for persisted native-session authority; a mount device number cannot serve as an indefinitely stable identity by itself (ARCH-PURPOSE, ARCH-FUNERAL).
- A fix must revalidate the exact saved conversation safely when filesystem identity changes, including rejection of replaced files or conflicting conversation identities. Blindly replacing `dev:` values or ignoring identity checks is not an established recovery method.
- Cover the two diagnostic branches (`binding lost` with a receipt and `session gone` without one) with the same surviving-transcript/device-renumber regression. Test the production cold-resume authorization boundary, not just metadata formatting.
- Diagnose binding publication independently for the four unbound parked launches and the live brain launch. Preserve current-launch ownership; historical or legacy IDs are not sufficient proof.
- No code fix or operational recovery has been attempted in this audit. Native transcript contents were checked only for identity evidence, not replayed or modified.

## Revisions

### 2026-09-29T07:59:14-07:00 — Expand incident evidence to all visible threads

At the operator’s request, appended the OS-upgrade context and a complete 23-row failure-mode audit. The original startup-blocker history and acceptance contract remain intact; this revision records additional recovery requirements and unresolved binding-publication cases without claiming implementation or recovery.

### 2026-09-29 — Begin the fix

Operator requested publishing the audit on main and starting implementation work. Added the initial work checklist before publication; a detailed design will be prepared on the claimed issue branch before code changes. `issue move-detail` confirmed creation was already completed, so the audit is an ordinary update to the existing main details.

### 2026-09-29 — Claimed in pair:0; concrete Codex diagnosis

Audit/checklist published on main as `b622052f`; #346 claimed and `sdlc start-plan` entered branch `000346-stale-temporary-store` in `/Users/xianxu/workspace/pair` as requested. The initial move was already complete; publication used `sdlc push` after associating the audit commit with the issue branch to satisfy the handoff ownership guard. No bypass flags used.

Read-only matching found all four audited Codex transcripts use `source: vscode`: brain `01a0eda0-b38c-7992-988a-74ad95e6925e`, pair:0 `01a0eb3e-8a92-71a2-b35d-65d20ae3e5f4`, pair:2 `01a0e914-ec6a-7e62-bf63-e73bbf5ae9cd`, parley.nvim:0 `01a0eb40-e0b1-7db0-be0d-403f75e873e5`. Each has matching normalized Pair sends and lies outside its launch baseline. This is candidate evidence, not a replacement for the production completed-round/proof checks. `codexRole` rejects vscode before correlation; [official Codex SessionSource](https://github.com/openai/codex/blob/main/codex-rs/protocol/src/protocol.rs) recognizes VSCode as a root-capable source. The earlier audit's unknown-producer cause is now narrowed to this scanner incompatibility.

### 2026-09-29 — Design revision

Expanded Spec/Done when and replaced the initial checklist with three actual review boundaries covering the original registry defect, proof device renumbering, and Codex binding/recovery. Added the durable implementation plan. All implementation remains behind plan approval and `sdlc change-code`; the active brain session stays running.

### 2026-09-29 — Identity model corrected by operator

Native conversation UUIDs make volume information unnecessary for conversation scoping/deduplication. Revised the draft plan: `(agent, native UUID)` is durable authority; file identity is a cache-continuity hint. A metadata change triggers full validation of the exact saved artifact set, not immediate loss of binding or a special device-only exception. The filename alone is insufficient; internal UUID, root role, schema, undisputed state, size bounds and read stability are checked. This replaces the earlier proposed same-inode requirement.

### 2026-09-29 — Plan review correction

Fresh-context review found one Important issue: repair must not reuse the watcher config refresh with empty argv. Changed the plan to preserve existing config bytes (and missing config), repair only ledger/catalog authority, and verify actual cold resume uses the recorded launch profile. Added old-catalog vscode regression coverage. Re-review pending; no code changes.
