# Branch review restoration implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for the connected implementation; delegate bounded exploration and fresh-context review. Steps use checkboxes for tracking.

**Goal:** Alt+C restores the current review branch's document without reselecting it, while preserving edits and in-flight review work.

**Architecture:** Git history owns durable document identity. A read-only resolver supplies that identity to the draft and pane; a pane-owned transition decides whether a switch is safe. Cached targets are session-local projections, never branch authority.

**Tech Stack:** Go review CLI, Lua/Neovim, Git, Zellij, existing shell acceptance harness.

## Design and alternatives

Use the Pair checkout/current draft working directory to select the repository, not the stale target's directory. On a review branch, resolve its document before consulting pane visibility or the target cache. On a non-review branch, do not automatically restore historical reviews: prompt for explicit selection; a proposed target may still report preparation in progress. A live review bound to another branch cannot be shown as the active review.

Resolve only exact current-slug round subjects, `review(<slug>): human|agent r<N>` with the existing optional ` — summary` suffix, and their NUL-delimited changed paths. Require exactly one distinct tracked, existing regular file within the repository. Inspect all matching rounds; do not pick the first path or infer a filename from the slug. The pre-branch `review: track <basename>` commit supplies no branch identity. Zero candidates, multiple candidates, detached HEAD, missing/deleted files, unsafe paths, or failed Git reads produce distinct diagnostics and no cache/pane mutations. Ordinary unrelated commits do not establish review identity. Bound reads as described below; incomplete history is an error, not proof of uniqueness.

Preserve first opening before the first round: successful explicit `:PairReview <file>` preparation writes a current-session selection receipt containing canonical repository, branch, file and prepared HEAD. When history has zero candidates only, verify that receipt still matches the checkout and tracked regular file, then permit opening. A stale target without this receipt, another conversation's receipt, or a changed HEAD cannot authorize this exception. Once matching rounds exist, history wins. Test both already-tracked and newly tracked first opens. A fresh session with no history must select the file explicitly; no durable identity is invented.

For an already-matching pane, retain the existing show/hide behavior. For a different clean idle pane, switch inside the same Neovim process through a private RPC endpoint and reconstruct the new document. Keep the old buffer and undo history. A modified buffer, deferred round, outstanding request, definition request, unconsumed handoff, or applied-but-uncommitted round refuses retargeting with a concrete recovery message. Never kill a pane to replace it. An old pane without the new RPC capability refuses safely and asks the operator to finish/close it.

Before activating a retained clean buffer, compare it with the current file bytes even when the pathname is unchanged. Apply any disk-content difference as one undoable in-buffer replacement through the existing projection machinery, preserving the undo tree; reconstruct decorations only after the displayed bytes match the selected checkout. Never use `:edit!` to erase retained state. Modified retained buffers block activation. Test two branches reviewing the same path with different bytes, and A changed on disk while inactive.

The draft updates its target only after acknowledged activation; fresh sessions may explicitly restore committed branch context but do not adopt another conversation's transient requests. Retargeting sends no automatic review request and makes no Git writes. Counts and decorations derive from the selected branch/file's rounds, not any inherited review commit.

Alternatives rejected: a persistent branch-to-file cache duplicates Git authority and cannot repair existing branches; killing/reopening the pane loses protocol state and invokes exit-save on a potentially different checkout. A filename guess cannot distinguish equal basenames or ambiguous branches.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| ReviewIdentity | `cmd/internal/reviewcmd/identity.go` | new |
| classifyIdentity | `cmd/internal/reviewcmd/identity.go` | new |
| transition | `nvim/review/restore.lua` | new |
| validate_context | `nvim/review/restore.lua` | new |

`ReviewIdentity` carries repository root, branch ref, pinned HEAD, and repo-relative document path. HEAD is an observation token, not a permanent identity: legitimate round commits advance it. `classifyIdentity` consumes collected round/path facts and returns resolved, absent, ambiguous, or invalid; tests supply data without subprocesses. It replaces the first-file assumption in readiness (ARCH-DRY).

`transition(state, event)` owns activation states idle, active, switching, and blocked, including the current identity, activation token and pending-work reason. It returns state plus declared effects. `validate_context` compares a handoff's activation/document identity before consumption or application. These are the single decision owners; IO code does not independently infer permission (ARCH-PURE, ARCH-ORDER).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| resolveIdentity | `cmd/internal/reviewcmd/identity.go` | new | bounded Git reads through Runtime |
| RunReadinessCLI / gatherGitFacts | `cmd/internal/reviewcmd/runcli.go`, `run.go` | modified | read-only `--resolve <directory>` JSON mode; shared resolver |
| PairReviewToggle | `nvim/init.lua` | modified | resolver, pane RPC, target publication, Zellij visibility |
| restore controller | `nvim/review/restore_controller.lua` | new | Neovim RPC, buffers, live branch revalidation |
| start_review / finish_human_turn | `nvim/review.lua` | modified | controller activation and pending-work facts |
| handoff.watch / review.start | `nvim/review/handoff.lua`, `init.lua` | modified | admission before read/unlink/apply and scoped reconstruction |
| RunOpen | `cmd/internal/reviewcmd/run.go` | modified | refuses live replacement; singleton opening |
| recovery.save / recovery.restore / recovery.saved | `nvim/review/recovery.lua` | new | bounded private recovery snapshots |
| restore client | `nvim/review/restore_client.lua` | new | async resolver, RPC, launch acknowledgment and target publication |
| identity.resolve | `nvim/review/identity.lua` | new | shared Go resolver CLI |
| poke builders | `nvim/review/poke_bodies.lua` | modified | explicit document/activation identity in request contract |

Use the existing Runtime seam, extending its fake to persist branch/commit/file/pane state across calls. Real temporary Git repositories and two headless Neovim processes provide conformance for history and RPC. The existing Zellij fake persists visibility and pane identities; test show/hide against that same seam (ARCH-MOCK).

## Transition and round safety contract

| State/event | Decision and effects |
|---|---|
| No pane + resolved branch | Revalidate branch/HEAD; open; acknowledge identity; publish session target |
| Matching pane + Alt+C | Verify identity, then existing visibility toggle |
| Different pane + clean/idle | Enter switching; pause watcher; revalidate; activate new buffer; publish acknowledgment; resume watcher |
| Different pane + unsaved/pending work | Block; keep buffer, target and all artifacts; explain what must finish |
| Branch changed while pane active | Suspend consume/apply/save/send/ship; expose conflict; preserve in-memory state |
| Git/RPC failure or timeout | Unknown outcome: re-query same pane/token; never spawn or kill as a timeout fallback |
| Duplicate request | Return acknowledged activation if token matches; do not duplicate buffer setup/timers |
| Pane dies during switch | No success publication; preserve existing disk artifacts; next invocation resolves afresh |
| Late/wrong-context round or definition | Leave payload unconsumed; report mismatch; never apply to new document |

The controller serializes events on Neovim's event loop; revalidates repository branch and resolved document before activation and before each apply/save/send/ship boundary. The draft validates the activation acknowledgment and rechecks the selected checkout before updating the cache/showing the pane. A changed HEAD causes re-resolution, not an assumption that a new round is a different review.

Extend handoff transport to accept `{context: {repo, branch, file, activation}, records: [...]}`. New workbench request pokes explicitly supply that context and require it to be echoed. Keep commit-body record encoding unchanged. Legacy array-only payloads may continue only in an uninterrupted legacy activation; a restored or retargeted activation refuses them visibly and preserves them for reissue. This is necessary because record text alone cannot prove which document a late response belongs to. Validate again when deferred work is applied. Definition responses retain their request-ID guard and also belong to the active context.

Carry the same context in landed artifacts and human-finished, agent-applied and ship pokes. The agent protocol must revalidate canonical repository, checked-out branch and document immediately before every Git effect, and preserve artifacts and stop on mismatch. Update `tests/lib/fake-review-agent.sh` to exercise that rule through its production-like producer flow. Update the authoritative producer instructions in `../ariadne/construct/local/fix/SKILL.md` (the Pair xx-fix skill is a symlink there), through a small linked Ariadne issue and its own SDLC gates; do not fork the skill into Pair. Pair's target documents the wire schema, and request pokes include it explicitly so the required envelope is visible at the moment of production. Test a branch switch after apply acknowledgment but before agent commit, and before human-round/ship effects. This governs cooperative agents; arbitrary independent Git commands remain outside the pane's authority.

Track applied-but-uncommitted work until Git supplies positive evidence: the matching document/branch round contains the landed record body and expected file content. A lingering landed file by itself is neither proof of pending work nor proof of completion. Pending requests with an uncertain agent outcome remain blocked; a timeout never clears them. Expose the recovery instruction to return to the original branch and finish the round. On exit under a branch mismatch, persist unsaved text through `recovery.save` below and report its path; never save stale buffer text into the new checkout.

## Operating envelope and ownership

- ARCH-CONSTRAINTS: Alt+C is an explicit UI action, never a render-time Git scan. Start with a 2-second total resolver deadline, 5-second RPC operation deadline, 10,000 matching commits and 8 MiB history output caps (engineering assumptions). Exceeding a bound produces a diagnostic without mutation. Use asynchronous Neovim calls; disable duplicate activation while one is pending. Test cancellation and bounded failure rather than claiming latency from these assumptions.
- ARCH-SECURE: branch names, Git paths, state files and RPC replies cross process/version boundaries. Use argument vectors, NUL framing, exact subjects, canonical in-repo regular paths, JSON validation and activation tokens. Do not evaluate untrusted Lua expressions or trust PID liveness as pane identity. A private per-process socket plus expected conversation/token handshake authorizes RPC.
- ARCH-FUNERAL: no new durable review history store. RPC socket lives under a private temporary directory, removed on normal exit; startup removes only provably dead owned residue. Existing open-state record adds endpoint/version/identity metadata while preserving its first two lines for existing readers. Only its owning incarnation removes it. Activation context is carried in the bounded open-state metadata; the existing stripped-document context artifact keeps its text format; target remains one record per existing scoped namespace. Recovery snapshots use the explicit lifecycle below; swap/undo alone is not a durability mechanism.
- ARCH-PURPOSE: cover both draft activation and pane application; fixing only the toggle leaves late rounds able to corrupt the restored document. No automatic branch checkout, review generation, or round commit belongs to this feature.

## Recovery storage (PQ-1)

`nvim/review/recovery.lua` owns one atomic JSON snapshot per canonical repo/branch/file in a private `review-recovery` directory beside the scoped open-state file. It stores identity and exact buffer lines/end-of-line flag, never a pathname inferred from user text; filenames are SHA-256 identity digests. The pane writes before an orderly branch-mismatched exit and on edits observed while mismatched. `QuitPre` writes synchronously and reports failure while retaining the modified flag, so ordinary non-bang quit remains blocked by Neovim; VimLeave is a last best-effort fallback for external termination, whose failure is reported rather than treated as preservation. Explicit forced quit (`:qa!`) can ignore autocmd errors and retains Neovim's discard semantics if storage fails; uncatchable process death remains ordinary editor crash semantics.

A writer admits at most 32 identities and 8 MiB per snapshot; existing keys are replaced atomically, never append-only. Admission failure preserves the existing snapshot and blocks orderly exit. Recovery activation verifies the snapshot's full identity, offers `:PairReviewRecover`, and loads it as modified text into the correct active review. The snapshot is removed only after a successful save of that recovered text on the matching branch (or explicit `:PairReviewDiscardRecovery`), never merely after reading it. Refuse overwriting an unconsumed prior-process snapshot. The final consumer is that recovery action; unresolved snapshots occupy bounded capacity and the diagnostic tells the operator how to recover/discard them. Files and directory are private; reject symlinks/nonregular storage. No editor-controlled Git writes.

## Verification strategy (PQ-2)

| Risky function | Adversarial class and mechanical guard |
|---|---|
| `classifyIdentity` | Fuzz malformed subjects/path framing and conflicting history facts; exact grammar and unique safe path classification |
| `resolveIdentity` | Stateful Git failure/history movement and resource exhaustion; pinned observation plus deadline/size caps, real temporary-repo conformance |
| `transition` | Generated activation/interrupt/retry sequences; assert unchanged identity/artifacts on refusal and single owner on success |
| `validate_context` | Malformed, missing and cross-activation payloads; no consume/apply without exact contextual admission |
| `recovery.save` / `recovery.restore` | Failed writes and process restart; atomic old-snapshot preservation, bounded admission, exact recovered bytes, no checkout write |
| controller activation | Two headless processes with controlled RPC completion order and retained-buffer disk drift; acknowledgment/context and byte equality before show |
| `handoff.watch` / `apply_round` | Late/deferred cross-branch responses; production guard before unlink, apply and landed publication |
| producer Git boundary | Stateful fake paused after acknowledgment; changed checkout prevents human/agent commit and ship without deleting artifacts |
| `PairReviewToggle` / `RunOpen` | Stale/live/fresh-session state through real command entry points; branch authority, verified initial selection, no kill/spawn on refusal |
| `reconstruct_on_open` | Inherited unrelated rounds with matching text anchors; only selected branch/document records decorate |

## Chunk 1: implement and verify one atomic restoration change

### Task 1: branch identity and history scope

Files: `cmd/internal/reviewcmd/identity.go`, `identity_test.go`, `run.go`, `runcli.go`, `run_test.go`, `runtime.go`; `tests/review-readiness-cli-test.sh`, `tests/review-resume-test.sh`.

- [x] Add failing `classifyIdentity` and `resolveIdentity` tests using the verification strategies above.
- [x] Run `go test ./cmd/internal/reviewcmd -count=1`; confirm the new assertions fail on current behavior.
- [x] Implement pure identity classification plus bounded collection behind Runtime. Add read-only `pair review readiness --resolve <directory>` structured output; reuse it for readiness file matching.
- [x] Make reconstruction consume the resolved current-slug/file history; include inherited unrelated agent rounds and identical text anchors in the regression.
- [x] Run package tests and `bash tests/review-readiness-cli-test.sh`; require success, then commit with issue reference and Co-Authored-By trailer.

### Task 2: pane-owned safe activation and round admission

Files: `nvim/review/restore.lua`, `restore_test.lua`, `restore_controller.lua`, `nvim/review.lua`, `nvim/review/init.lua`, `handoff.lua`, `poke_bodies.lua`, their existing tests; `tests/review-window-test.sh`, `review-handoff-test.sh`, `review-loop-test.sh`, `tests/lib/fake-review-agent.sh`; linked peer instruction change in `../ariadne/construct/local/fix/SKILL.md`.

- [x] Add failing `transition`, `validate_context` and controller tests using the verification strategies above.
- [x] Confirm controlled-order integration regressions fail before implementation.
- [x] Implement the controller and private endpoint; refactor activation cleanup so old timers/autocommands do not survive a switch and old buffers/undo remain intact.
- [x] Gate save/apply/send/ship at the shared owner; implement `recovery.save`/`restore` and prove durability across process exit.
- [x] Add context envelopes and request instructions without changing committed record encoding; preserve rejected payloads. Prove old/new document identical anchors cannot bypass context checks.
- [x] Carry context through landed artifacts and all commit/ship requests; update the stateful fake producer and authoritative xx-fix instructions via a linked peer issue. Prove a branch switch before the agent's Git effect preserves artifacts and performs no commit/ship.
- [x] Run the window, handoff, loop and resume shell suites; require success, then commit.

### Task 3: authoritative Alt+C and end-to-end restoration

Files: `nvim/init.lua`, `cmd/internal/reviewcmd/run.go`, `run_test.go`, `tests/review-toggle-test.sh`, new `tests/review-branch-restore-test.sh`.

- [x] Reproduce the current failure using real temporary review branches and headless draft/review processes; assert displayed buffer path/content and decorations, not just resolver output.
- [x] Wire resolution before liveness/visibility; activate through the pane controller and publish cache only after acknowledgment. Remove RunOpen's unconditional live-pane kill and test its refusal has no kill/remove/spawn effects.
- [x] Exercise the issue Done-when through the production-boundary strategy above; require A → B → A without repeated selection when idle.
- [x] Verify the explicit-selection exception and retained-buffer reconciliation contracts through displayed bytes, decorations and undo preservation.
- [x] Mutation-check the resolver call, pending guard and pre-consumption context guard independently; each removal must fail the production-boundary regression. Restore from byte copies.
- [x] Run `go test ./cmd/internal/reviewcmd -count=1`, `go test -race ./cmd/internal/reviewcmd -count=1`, `make test-lua`, and all `tests/review-*-test.sh` plus `tests/pair-review-target-test.sh`. Build with `make build` and record any environmental limitation precisely.

### Task 4: operator contract and boundary

Files: `atlas/review-workbench.md`, `workshop/targets/review-protocol.md`, issue #341; `workshop/lessons.md` only for review findings.

- [x] Document branch authority, explicit fresh-session restore, blocked transitions, non-review behavior and envelope compatibility; append target Revisions and update its active seam table. Keep atlas index coverage.
- [x] Reconcile every plan symbol and acceptance row with delivered implementation and test evidence. Record verification in the issue log; checkpoint commits.
- [ ] Run `sdlc close --issue 341 --verified '<measured evidence>'`; its fresh-context review owns this single boundary. Fix findings, rerun affected tests, and record the verdict. Publication follows `sdlc pr` / `sdlc merge` under the session's authorization.

## Approval and estimate

This exceeds the quick-flow code limit. The operator approved the durable plan before implementation; `sdlc change-code` passed plan-quality and then estimate-quality. Operator approved execution; full-flow plan and estimate gates passed. Issue estimate is 3.24 calibrated hours; peer instruction work is tracked separately in ariadne#268.

## Revisions

2026-09-28 — Fresh-context review identified first-open, retained-content and agent-commit gaps. Added a verified explicit-selection exception for zero-round branches, undo-preserving content reconciliation, and end-to-end context propagation through producer instructions and Git effects. Corrected ARCH-ORDER attribution. These additions preserve the original issue contract.

2026-09-28 — PQ-1/PQ-2: replaced the unsupported swap/undo recovery assumption with bounded atomic snapshots and explicit recover/discard ownership, and compressed case inventories into named function-level adversarial strategies. Operator approved execution before this safety refinement. Linked producer work is ariadne#268.

2026-09-28 — Integration refinements: canonical path identity is checked before pane startup; failed activation restores the previous buffer/owner; empty first human rounds retain the already-authorized active selection. Explicit same-session peer-document selections remain usable while the draft checkout is non-review; any current review branch still takes precedence. Named delivered client, identity and recovery modules in the integration table. Context metadata lives in the existing open-state record rather than replacing stripped document text.

2026-09-28 — Real-process failed-storage probe showed Neovim force-quit ignores QuitPre callback errors. Narrowed the exit guarantee to ordinary non-bang quit with the modified flag retained; forced quit remains explicit discard semantics on failed storage. Regression injects unsafe storage, proves ordinary exit is blocked, then restores storage and proves successful forced-exit recovery in a new process.

2026-09-28 21:47 PDT — Boundary REWORK identified consumption ordering, blocking edit-event observation, missing operator docs, and activation callback ownership. Preserve the approved restoration contract while fixing the complete classes (ARCH-ORDER, ARCH-CONSTRAINTS, ARCH-FUNERAL):

- [x] Consume handoffs only after explicit apply/defer acceptance; preserve refused/error/replaced payloads and test admission-to-application branch movement.
- [x] Coalesce proactive recovery into bounded asynchronous observation; keep synchronous authority checks at effect/quit boundaries and test delayed resolution plus stale completions.
- [x] Own and replace activation rendering autocmds; assert stable callback counts through A → B → A.
- [x] Explain branch restoration, blocked transitions, and recover/discard commands in README.
- [ ] Rerun affected and full review tests, build, and repeat the SDLC boundary review.

2026-09-28 — Boundary class sweep also covers definition response generation receipts and changed-selection refusal, handoff callback reentry, partial-apply uncertainty, and accepted payload cleanup failure. Shared artifact receipts avoid duplicating replacement detection (ARCH-DRY). Asynchronous focus observations preserve same-branch clean-buffer reload while refusing other-branch disk bytes; empty definition polling no longer resolves history.

2026-09-28 22:07 PDT — Round 2 disposed BR-1 through BR-4, then reproduced late observation → later checkout read in asynchronous focus reload (BR-5, same nonblocking-editor-observation family). An identity observation must not authorize independently read future bytes. Reload will use a bounded snapshot captured inside the resolver's pinned identity checks; callbacks apply captured bytes only under the same activation and buffer revision. Preservation continues to read the owned buffer under its captured binding; newer edits must not disappear from the coalescing queue. Add controlled observation-on-A / delivery-after-checkout-B tests and snapshot bound/checkout-movement tests.

The operator's smoke test independently found hidden review could not reopen: Zellij reports false with exit 1, while the new client treated all nonzero exits as errors. Correct the predicate's result contract and its fakes; verify real draft/pane hide → show while an agent request is pending.

2026-09-28 — Identity-bound reload delivered through optional `--snapshot` on the shared resolver. Snapshot capture validates file identity/containment and text/output bounds inside the pinned Git window. Callback decisions validate activation and buffer revision; newer requests stay queued, and mismatch preservation uses only owned buffer bytes. Existing projection machinery now optionally records the prior decoration snapshot for refresh undo/redo. Full Lua, race, review and build verification passed, including controlled late completion and the operator's pending-round hidden-pane regression.

2026-09-28 — Round 3 reproduced Git's intermediate checkout state: working-tree bytes can already belong to B while HEAD remains A. Extend snapshot admission across checkout mutation itself, with fail-closed retry and a real Git smudge-filter pause regression; captured delivery alone is insufficient (BR-5, ARCH-ORDER). Capture, delayed delivery, owned-buffer preservation, and newer-event coalescing must each maintain their source identity. Also consolidate activation/refresh byte decoding: binary readfile retains CR while loaded buffers already use dos format, causing CRCRLF on save (BR-6, ARCH-DRY). Verify exact saved bytes through new and retained activation, branch-driven format changes, LF/CRLF/empty/no-EOL input. These fixes preserve the approved restoration and no-data-loss contract.

2026-09-28 — Checkout admission deliberately refuses any staged index tree (including attributes) until commit/unstage; unstaged edits remain supported. Index lock/generation observation takes no reader-owned Git lock. Shared document codec and recovery options now preserve exact bytes across activation, refresh, persisted recovery and landed-round proof. Initial opening and branch activation also consume the resolver's captured snapshot, completing the acquisition/delivery class sweep without independent checkout reads.
