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
