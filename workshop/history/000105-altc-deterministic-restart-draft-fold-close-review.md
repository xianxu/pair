# Boundary Review — pair#105 (whole-issue close)

| field | value |
|-------|-------|
| issue | 105 — alt+shift+c: deterministic writer-triggered restart + fold draft WIP into continuation NEXT ACTION |
| repo | pair |
| issue file | workshop/issues/000105-altc-deterministic-restart-draft-fold.md |
| boundary | whole-issue close |
| milestone | — |
| window | b9dc30a9e64da75e8a6c0e0606c25292a5bcb529..HEAD |
| command | sdlc close --issue 105 |
| reviewer | claude |
| timestamp | 2026-07-06T22:52:04-07:00 |
| verdict | SHIP |

## Review

I've completed my verification. All checks pass: build is green, all package tests pass (`continuationcmd`, `launcher`, `adapt`), `go vet` is clean, and my independent parity probe confirms the Go `StripStickyComments` is byte-faithful to the Lua `strip_comments` on the drift-prone edges (trailing newline, leading comment, comment-only, indented, `===`-not-at-line-start).

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This is a clean, well-architected boundary. The three new `draft.go` entities are genuinely pure and unit-tested without IO; the draft-read and restart-exec are correctly isolated as injected seams (`run()` takes `env runEnv` + `restart func(string) error`), so the compaction logic is testable with fakes and no live pane. The Lua→Go strip port is faithful and pinned with a comment; the fold logic reuses `isATXHeading`; both data-dir resolutions (Go `adapt.DataDir()` and Lua `pair_data_dir()`) agree, so the writer reads exactly the file nvim's `:w` persists. The `--no-restart` escape suppresses both fold and restart as spec'd, and the atlas is updated for the new writer-owned flow. Every Plan checkbox maps to delivered code. Nothing blocks SHIP. The one residual is verification, not code: the composed real-exec link (`os.Executable()` → `pair continue` → `runCompaction`) is only exercisable via the documented live `alt+shift+c` smoke — each half is tested, the composition is not (an inherent IO-boundary limit, correctly pushed to the thin seam).

**1. Strengths**
- **ARCH-PURE done right** (`draft.go:15,40,85`): the fold/strip/gate are pure string functions; `continuationcmd.go:48-66` reads env and builds the exec seam at the outer boundary and threads both into `run()`. Tests drive `run()` with a fake `restart` and a temp repo — no mocks needed for the "pure" entities.
- **Faithful port, honestly scoped** (`draft.go:5-14`): the pinning comment names `nvim/init.lua:995` and the Revisions justify skipping a cross-language harness with the repo's own "wrong vs. merely less-ergonomic" lesson test. My parity probe confirms the semantics match.
- **Fold ordering is deliberate and correct** (`continuationcmd.go:104-115`): folding *before* the `HasNextAction` guard lets parked WIP rescue a thin section, and inserting *after* existing content (`draft.go:40-73`) keeps `NextActionPreview` stable — both documented and test-pinned.
- **Durable-first restart** (`continuationcmd.go:169-177`): restart fires only after write+commit, and a restart error is non-fatal with the continuation kept — the doc survives even if the relaunch seam fails.
- **Test coverage matches the Done-when** point-for-point: fold-lands (`TestRun_FoldsDraftWhenInCompaction`), restart-called-with-slug (`TestRun_TriggersRestartInCompaction`), standalone-no-op (`TestRun_NoFoldOrRestartStandalone`), `--no-restart` suppresses both (`TestRun_NoRestartFlagSuppressesInCompaction`).

**2. Critical findings** — none.

**3. Important findings** — none blocking.

**4. Minor findings**
- **Proxy breadth is a live watch-item** (`draft.go:81-86`): `InCompactionContext` fires the restart on *any* in-pane `pair continuation`, not just the Alt+Shift+C prompt. The comment/Revisions assert "the compaction prompt is the sole in-pane invoker," but an agent asked to "park this session" mid-session (via the xx-datatype writer, in-pane) would now trigger an unrequested restart. Documented + accepted with a `--no-restart` escape, and it matches the spec — so not a defect — but the agent has no way to know to pass the flag. Worth an eye during dogfood.
- **`NextActionPreview` label leak in the empty-heading rescue** (`draft.go:68`): when NEXT ACTION was empty and the fold rescues it, the first content line becomes `_Parked draft at compaction:_`, so the `pair continue` list preview shows the label rather than the WIP. Cosmetic, edge-only.
- **README has no Alt+Shift+C entry** (`README.md:56-82`): pre-existing (from #55), *not* introduced by #105 — the changed surface here is internal (writer-owned restart) and the new `--no-restart` flag is on an internal writer, not a user command. No README update owed by this boundary; noting the pre-existing gap for a future doc pass.
- **nvim `:w` persists the current buffer** (`nvim/init.lua:PairConfirmCompact`): the pre-send `silent! write` writes whatever buffer is focused; if the operator sat on a history/queue slot at compaction, the folded WIP could reflect that slot rather than the `*` draft. Best-effort feature; verify in the live smoke.

**5. Test coverage notes**
- The composed real-exec link (writer → `os.Executable() continue <slug>` → `runCompaction` → relaunch) is **not** covered by any automated test — only the two halves are. This is an inherent live-zellij limitation, correctly confined to the injected seam, but it means the headline Done-when ("alt+shift+c reliably restarts") rests on the manual smoke. **Recommend the operator actually run the documented live smoke before recording the close** (the Log frames it as pending) — that is the remaining real verification, not a code change.
- No test asserts restart-fires-after-write ordering, but the code structure (restart at end of `run` after the write) plus `readWrittenContinuation` in the fold test make this adequately implied. Not worth adding.

**6. Architectural notes for upcoming work**
- **ARCH-DRY — PASS (with a soft note):** `FoldDraftIntoNextAction`'s section scan and `firstNextActionLine` are genuinely different scans (section-end bounds vs first-content-line) and both correctly reuse `isATXHeading`; no duplication warranting extraction today. If a third consumer needs "NEXT ACTION section bounds," a small `nextActionSectionBounds` helper would consolidate them — not warranted now.
- **ARCH-PURE — PASS:** textbook pure-core/thin-shell split; the only IO in the fold/restart path (`os.ReadFile`, `exec.Command`) lives in `Run`/the seam, injected into `run()`.
- **ARCH-PURPOSE — PASS (shadow-sweep clean):** the issue's purpose was *deterministic restart with no agent step to forget*. The consumers are: (a) the writer, which now owns the restart — enforced in code, not prose; (b) `COMPACT_PROMPT`, collapsed to one step with the second step *removed* (`nvim/init.lua:3329-3337`), not merely reworded; (c) the atlas, updated. No hand-maintained restatement of the old two-step remains in active docs (the surviving `pair continue <slug>` references in README/createflow/compaction are the legitimate standalone command + pre-existing seed path, not the removed step). The purpose is fulfilled, not deferred.

**7. Plan revision recommendations** — none. The plan's Core-concepts table matches the code exactly (all three entities at `draft.go`, new, pure, IO-injected as described), the Revisions already reconcile the drift-test substitution and the integration-test framing, and every checklist item is delivered. No `## Revisions` entry needed.

---

## Re-review — 2026-07-06T23:35:02-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 105 — alt+shift+c: deterministic writer-triggered restart + fold draft WIP into continuation NEXT ACTION |
| repo | pair |
| issue file | workshop/issues/000105-altc-deterministic-restart-draft-fold.md |
| boundary | whole-issue close |
| milestone | — |
| window | b9dc30a9e64da75e8a6c0e0606c25292a5bcb529..HEAD |
| command | sdlc close --issue 105 |
| reviewer | claude |
| timestamp | 2026-07-06T23:35:02-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have a complete picture. Let me record the pending-smoke verification framing by checking the exact Revisions wording once more, then write the verdict.

The verification is complete. Here is my review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The code is SHIP-quality: the three `draft.go` entities are genuinely pure and unit-tested without IO; the draft-read and restart-exec are correctly isolated as injected seams (`run()` takes `env runEnv` + `restart func(string) error`); the Lua→Go strip port is byte-faithful (I traced the line-split/edge-trim semantics against `nvim/init.lua:995`); the `PAIR_FAKE_IN_ZELLIJ=1` fake correctly flows into `compactionDecision(opts.ForceInSession, rt.InZellijPane() || opts.FakeInZellij, …)` at `createflow.go:45` with the real tag-match still guarding the session; build + full `go test ./...` are green. The one thing blocking a clean SHIP is **documentation drift, not code**: the diff added a new "writer-owned compaction" paragraph to `atlas/architecture.md` but left **two older atlas passages (L326 and L442) still describing the removed agent-runs-`pair continue` two-step flow** as current — the atlas now self-contradicts on the exact flow this issue changed. That is precisely the Docs-gate + ARCH-PURPOSE shadow-sweep class, and the prior close-review's claim "no hand-maintained restatement of the old two-step remains in active docs" is incorrect. Fix those two lines, then ship.

**1. Strengths**
- **ARCH-PURE done right** (`draft.go`, `continuationcmd.go:46-64`): env read + `os.Executable()` exec seam built at the outer `Run` boundary and threaded into `run()`; the pure fold/strip/gate functions need no mocks. Textbook pure-core/thin-shell.
- **Faithful, honestly-scoped port** (`draft.go:5-14`): I verified `StripStickyComments` matches Lua `strip_comments` on the drift-prone edges (trailing newline → `["a",""]` then trailed off; leading/comment-only/indented-`===`). The pinning comment names the source; the Revisions justify skipping a cross-language harness with the repo's own "wrong vs. less-ergonomic" lesson test — a defensible call.
- **Correct fake-env wiring** (verified end-to-end): `newContinueRestartCmd` inherits `os.Environ()` (so `PAIR_TAG`/`ZELLIJ_SESSION_NAME` ride through) + adds `PAIR_FAKE_IN_ZELLIJ=1`; the child's real tag-match still runs, so it can't compact the wrong session. This is the actual root-cause fix and it's wired soundly.
- **Durable-first, non-fatal restart** (`continuationcmd.go:184-194`): restart fires only after write+commit; a restart error is logged, not fatal — the continuation survives a relaunch-seam failure.
- **Fold ordering is deliberate and correct** (`continuationcmd.go:121-132`, `draft.go:40-89`): folding before the `HasNextAction` guard lets WIP rescue a thin section; inserting after existing content keeps `NextActionPreview` stable; the empty-section branch omits the label so the preview surfaces the WIP, not `_Parked draft…_`. All test-pinned.
- **Test coverage matches the Done-when point-for-point**: fold-lands, restart-called-with-slug, standalone-no-op, `--no-restart` suppresses both, fake-env pinned.

**2. Critical findings** — none.

**3. Important findings**
- **Atlas self-contradicts on the changed flow — two stale passages** (`atlas/architecture.md:326` and `:442`). The diff added the new writer-owned paragraph (L215-238) but did not update:
  - **L326** (keybinding table): "…so it asks the agent to write a continuation + run `pair continue <slug>` (context-aware → compaction)."
  - **L442** (In-session compaction section): "…an agent-agnostic prompt that defers to the `continuation` datatype procedure — flush-first, then write the continuation **and run `pair continue <slug>`** — rather than enumerating…"
  Both describe the removed step-2 as current behavior, directly contradicting the new paragraph and the shipped code (writer owns the restart; `COMPACT_PROMPT` now says "do NOT run `pair continue` yourself"). Fix sketch: rewrite L326 to "…asks the agent to write a continuation via `pair continuation`; the writer then triggers the restart itself (#105)" and edit L442's parenthetical to drop "and run `pair continue <slug>`" (the writer restarts). This is the Docs-gate finding (atlas is the "always-current" map) and the ARCH-PURPOSE shadow-sweep miss.

**4. Minor findings**
- `PairConfirmCompact`'s `pcall(vim.cmd, 'silent! write')` (`nvim/init.lua:3350`) saves *whichever buffer is focused*; `pair_ensure_visible_then` (`nvim/init.lua:3086`) only un-minimizes, it does **not** refocus the draft — so the comment "the Alt+Shift+C binding focuses it first" is inaccurate. If the operator sits on a history/queue slot buffer, `:w` writes that file and the fold reads the autosaved draft instead of the freshest WIP. Harmless (pcall-guarded, autosave fallback), but the comment overstates the guarantee — soften it and verify in the live smoke.
- `InCompactionContext` proxy breadth (`draft.go:101`): any in-pane `pair continuation` restarts, not just the Alt+Shift+C prompt. Documented + `--no-restart` escape, but an agent invoking the writer in-pane for a non-compaction reason has no signal to pass the flag. Dogfood watch-item.
- No `run()`-level test asserts the *fold-rescues-empty-NEXT-ACTION-then-write-succeeds* path (fold before `HasNextAction`); the unit test + wiring order make it adequately implied — not worth adding.

**5. Test coverage notes**
- Strong on every automatable surface. The **composed real-exec link** (`os.Executable()` → `pair continue <slug>` → `runCompaction` → relaunch) is inherently un-headless-testable; correctly confined to the injected seam. Note for the operator: the Revisions state "**Re-smoke required to confirm the restart now fires end-to-end**" after the `PAIR_FAKE_IN_ZELLIJ` fix — the headline Done-when ("alt+shift+c reliably restarts, even under the sandbox") rests on that specific re-smoke, which the Log frames as a still-pending manual step. Run it before trusting the close.

**6. Architectural notes for upcoming work**
- **ARCH-DRY — PASS:** reuses `adapt.DataDir()`, `isATXHeading`, the `pair continue`→`runCompaction` path, and mirrors `compactionDecision`'s tag-match (`InCompactionContext`). `StripStickyComments` is a deliberate, pinned cross-language mirror — the one acceptable duplication, justified in-comment.
- **ARCH-PURE — PASS:** pure string core; the only IO (`os.ReadFile`, `exec.Command`, `os.Getenv`) lives in `Run`/the seam, injected into `run()`.
- **ARCH-PURPOSE — FLAG (= the Important finding):** the *code* fulfills the purpose (restart enforced in code, `COMPACT_PROMPT` step-2 removed, not reworded). But the single-source shadow-sweep is incomplete: the atlas has three consumers of "how compaction restarts," and two of them (L326, L442) still restate the removed model as current. Per ARCH-PURPOSE a hand-maintained restatement that no longer derives is an unfinished consumer — reconcile them as part of this boundary.

**7. Plan revision recommendations** — none. The Core-concepts table matches the code exactly (all three entities at `draft.go`, new, pure, IO-injected), and the Revisions already reconcile the drift-test substitution, the integration-test framing, and the `PAIR_FAKE_IN_ZELLIJ` root cause. No `## Revisions` entry needed; the gap is in the atlas, not the plan.

---

## Re-review — 2026-07-07T01:02:21-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 105 — alt+shift+c: deterministic writer-triggered restart + fold draft WIP into continuation NEXT ACTION |
| repo | pair |
| issue file | workshop/issues/000105-altc-deterministic-restart-draft-fold.md |
| boundary | whole-issue close |
| milestone | — |
| window | b9dc30a9e64da75e8a6c0e0606c25292a5bcb529..HEAD |
| command | sdlc close --issue 105 |
| reviewer | claude |
| timestamp | 2026-07-07T01:02:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Shadow-sweep complete. The atlas passages the prior re-review flagged (L326, L442) are now fixed, but the sweep surfaced a **fourth consumer the re-review's enumeration missed**: `zellij/config.kdl:273` still describes the removed two-step flow. Everything else — code, tests, wiring, ARCH principles — is clean. Full suite green, build + vet clean.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The code is SHIP-quality and the headline bug is fixed at the root: the `pair continuation` writer now owns the restart (`continuationcmd.go:186-194`) instead of relying on a skippable agent step, and `PAIR_FAKE_IN_ZELLIJ=1` correctly threads into `compactionDecision(opts.ForceInSession, rt.InZellijPane() || opts.FakeInZellij, …)` at `createflow.go:45` with the real `ZELLIJ_SESSION_NAME` tag-match still guarding the session — the actual sandbox root-cause. The three `draft.go` entities are genuinely pure and unit-tested without IO; the draft-read and exec-restart are cleanly injected seams; the Lua→Go strip port is byte-faithful (I traced `strip_comments` at `nvim/init.lua:995` against `StripStickyComments` on the drift-prone edges — trailing-newline element count, leading/comment-only/indented-`===`, and the leading-vs-trailing trim order is order-independent). `go build ./...`, `go vet`, and the full `go test ./...` are all green. What blocks a clean SHIP is **one residual documentation drift, not code**: the prior FIX-THEN-SHIP re-review's shadow-sweep enumerated the atlas consumers and reconciled L326/L442, but **missed a fourth consumer** — `zellij/config.kdl:273` still says the binding "prompts the agent to write a continuation **and run `pair continue <slug>`**", the exact removed two-step. It's the most natural place a reader looks to learn what Alt+Shift+C does, and it now contradicts the shipped flow. One-line comment fix, then ship.

**1. Strengths**
- **ARCH-PURE done right** (`draft.go`, `continuationcmd.go:46-64`): env read + `os.Executable()` exec seam built at the outer `Run` boundary and threaded into `run()` as `env runEnv` + `restart func(string) error`; the pure fold/strip/gate functions need no mocks. Textbook pure-core/thin-shell.
- **Root-cause fix, wired soundly** (`continuationcmd.go:96-101` → `runcli.go:90` → `createflow.go:45`): `newContinueRestartCmd` inherits `os.Environ()` (so `PAIR_DEV`/`PAIR_TAG`/`ZELLIJ_SESSION_NAME` ride through — pair/pair-dev preserved with no explicit branch) plus `PAIR_FAKE_IN_ZELLIJ=1`; the child's real tag-match still runs, so it can't compact the wrong session.
- **Durable-first, non-fatal restart** (`continuationcmd.go:184-194`): restart fires only after write+commit+push; a restart error is logged, not fatal — the continuation survives a relaunch-seam failure.
- **Fold ordering is deliberate and correct** (`continuationcmd.go:121-132`, `draft.go:40-89`): folding *before* the `HasNextAction` guard lets parked WIP rescue a thin/empty section; inserting *after* existing content keeps `NextActionPreview` stable; the empty-section branch *omits* the label (`draft.go:79-84`) so the preview surfaces the WIP, not `_Parked draft…_`. All test-pinned, incl. the empty-section `NextActionPreview` assertion (`draft_test.go:71-73`).
- **Test coverage matches the Done-when point-for-point**: fold-lands (`TestRun_FoldsDraftWhenInCompaction`), restart-called-with-slug (`TestRun_TriggersRestartInCompaction`), standalone-no-op (`TestRun_NoFoldOrRestartStandalone`), `--no-restart` suppresses both (`TestRun_NoRestartFlagSuppressesInCompaction`), fake-env pinned (`TestNewContinueRestartCmd_FakesInZellij`).

**2. Critical findings** — none.

**3. Important findings**
- **Fourth stale consumer missed by the prior shadow-sweep** (`zellij/config.kdl:273`). The comment on the `Alt C`/`Ctrl Alt c` binding still reads: "Routes to nvim, which asks Y/N then prompts the agent to **write a continuation and run `pair continue <slug>`**." That is the removed two-step — the writer now owns the restart and `COMPACT_PROMPT` explicitly says "do NOT run `pair continue` yourself" (`nvim/init.lua:3334`). Same Docs-gate / ARCH-PURPOSE class the re-review flagged for atlas L326/L442, in a consumer its enumeration didn't reach — so the sweep is still incomplete. **Fix sketch:** rewrite `config.kdl:272-273` to "Routes to nvim, which asks Y/N then prompts the agent to write a continuation via `pair continuation`; the writer then triggers the restart itself (#105)." Cheap, non-blocking.

**4. Minor findings**
- `InCompactionContext` proxy breadth (`draft.go:101`): any in-pane `pair continuation` restarts, not just the Alt+Shift+C prompt. Documented + `--no-restart` escape, but an agent invoking the writer in-pane for a non-compaction reason has no signal to pass the flag — dogfood watch-item.
- `TestNewContinueRestartCmd_FakesInZellij` (`run_test.go:120-135`) pins `PAIR_FAKE_IN_ZELLIJ=1` but does not assert the env *inheritance* (`os.Environ()` ride-through) that carries `PAIR_DEV` — the pair/pair-dev-preservation claim rests on reading the one-liner, not a test. Very minor; a `len(c.Env) > 1` or ride-through assertion would close it.

**5. Test coverage notes**
- Strong on every automatable surface. The **composed real-exec link** (`os.Executable()` → `pair continue <slug>` → `runCompaction` → relaunch) is inherently un-headless-testable and correctly confined to the injected seam. Per the issue's 2026-07-07 Revision, the live re-smoke confirmed detection now fires under the sandbox but `zellij kill-session` is still blocked by the sandbox's unix-socket restriction; the flow was verified end-to-end **unsandboxed**, and the remaining gap is scoped as a sandbox-policy decision, not a #105 code bug. That scope call is defensible and documented — the two #105 deliverables are verified. No code change owed here.
- No `run()`-level test asserts the fold-rescues-empty-NEXT-ACTION-then-write-succeeds path, but the `draft.go` unit test plus the fold-before-`HasNextAction` wiring order make it adequately implied. Not worth adding.

**6. Architectural notes for upcoming work**
- **ARCH-DRY — PASS:** reuses `adapt.DataDir()`, `isATXHeading`, the `pair continue`→`runCompaction` path, and mirrors `compactionDecision`'s tag-match in `InCompactionContext`. `StripStickyComments` is a deliberate, in-comment-pinned cross-language mirror — the one acceptable duplication (no Lua unit harness; `^\s*===` is trivial and stable). `FoldDraftIntoNextAction`'s section scan and `firstNextActionLine` are genuinely different scans (insert-bounds vs first-content-line) and both reuse `isATXHeading`; no extraction warranted today.
- **ARCH-PURE — PASS:** pure string core; the only IO in the fold/restart path (`os.ReadFile`, `exec.Command`, `os.Getenv`) lives in `Run`/the seam, injected into `run()`.
- **ARCH-PURPOSE — FLAG (= the Important finding):** the *code* fulfills the purpose (restart enforced in code; `COMPACT_PROMPT` step-2 *removed*, not reworded). But the single-source shadow-sweep is still one consumer short: `zellij/config.kdl:273` remains a hand-maintained restatement of the removed two-step model that no longer derives from the shipped behavior — reconcile it as part of this boundary. Atlas ×2 are now clean; this is the last one.

**7. Plan revision recommendations** — none. The plan's Core-concepts table matches the code exactly (all three entities at `draft.go`, new, pure, IO-injected), and the Revisions already reconcile the drift-test substitution, the integration-test framing, and the `PAIR_FAKE_IN_ZELLIJ` root cause. The `zellij/config.kdl` gap is an *omission* (the plan's Task 6 updated the nvim comment but never listed the kdl comment), not a false claim the plan makes — so no `## Revisions` entry is owed; fix the comment and note it in the issue `## Log` alongside the atlas reconciliation.
