# Agentic Review Workbench — M4 (agent protocol) Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the execution approach (superpowers-subagent-driven-development or superpowers-executing-plans). Steps use checkbox (`- [ ]`) syntax.

**Goal:** Turn the workbench's loop over to a *real* agent — the pair-side nvim stops writing git (the agent owns branch + rounds + ship), driven by the writing-assistant SKILL (ariadne **#000121**), with modes / voice / fact-check and a faithful end-to-end round-trip.

**Architecture:** M1 scaffolded the nvim shelling `docflow` (human + agent rounds) so the contract was provable headlessly with a fake agent. M4 inverts that per `workshop/targets/review-protocol.md` invariant #1: the nvim *reads* git (reconstruct decorations + indicator counts) but **writes none** — it applies records undo-ably, saves, and pokes a **commit signal**; the agent commits. The agent half is ariadne #000121 (a cross-repo `deps`); the pair side is provable with a protocol-faithful fake agent (`fake-agent-v2`) before the real SKILL lands.

**Tech Stack:** nvim Lua (`nvim/review/*`), the existing `pair_poke` seam, `docflow` (now agent-only), shell test fakes; ariadne's writing-assistant SKILL (#000121, separate repo).

**Sub-milestones** — **re-sliced structure-first** (2026-06-21; see ## Revisions): get the *whole loop standing thin* before any tuning.

- **M4a** — real loop + agent-owns-git unwind (this plan, in detail). *Implemented/headless-verified; evidence folded into the current M4 skeleton boundary rather than retroactively milestone-closed.*
- **M4a'** — review-start & resume flow (`:PairReview` proposes → agent preps via the pure readiness probe → Alt+r opens; reconstruct-on-open). **Spec: `workshop/targets/review-protocol.md` → "Review-start & resume flow — M4a'" + seam #6.** *Pair side implemented/headless-verified; folded into current boundary.*
- **M4b — skeleton (structure):** the 🤖[] fulfill/punt + **accept/reject** (parley §5) conversation + a **default editing posture** + **ship** (`docflow ship`) — completes the thin full cycle. The smallest set that makes the loop *usable* end-to-end. *In progress: accept/reject + marker nav landed; fulfill/punt/default posture/ship remain.*
- **M4c — thicken (tuning; sub-slices when reached):** modes menu + `🪄 Mode`/spinner/lean-history (`-92 < -3 > +0`) bar; voice (`voice: <slug>` → `~/.personal/<slug>-writing-style.md`); fact-check pass (`doc-review` → records); pending-🤖{} quickfix; diagnostic-display polish; cross-session undo-of-style completeness; `xx-fix`→`writing-assistant` rename; the faithful e2e demo (final Done-when). *One pure spinner helper exists as unwired pre-work.*

---

## Core concepts (M4a)

M4a is mostly **removing IO** (the nvim's docflow writes) and relocating the round-commit responsibility to the agent — so the conceptual core (`record` / `reconstruct` / `apply`) is **unchanged and stays pure**; the work is at the integration seams.

### Decision (gates Tasks 1 & 3): the post-apply *landed-artifact* (plan-quality blocking finding)

"The nvim stops committing; the agent owns the round commit" relocates not just the
`docflow.round('agent', …)` *call* but the *authorship of the commit body*. Today the body
is `record.embed_in_body(summary, enriched)` where `enriched` is `apply.apply`'s output —
records carrying the computed `new_occurrence`, with the **dropped** (unanchorable) set
filtered out. The resume path depends on this: `reconstruct` locates by `new_occurrence`.
A count-only poke (`agent_applied(n, file)`) cannot carry that, and `fake-agent-v2` has no
intelligence to re-derive it. So the *apply authority* (the nvim — which alone knows what
actually landed) writes a structured artifact; the committing party reads it and commits
**verbatim**:

- **`$PAIR_DATA_DIR/review-landed-<tag>.json`** (new seam #2b, nvim→agent), written by
  `on_agent_round` right after `apply` + `save`: `{ "summary": "N edit(s)", "body":
  "<record.embed_in_body output>", "applied": N, "dropped": M }`. The **body** is built by
  the *one* encoder (`record.embed_in_body`) the nvim already owns — so there remains a
  single serialization across handoff / commit-body / resume (ARCH-DRY). The agent /
  `fake-agent-v2` reads it and runs `docflow round --side agent -m <summary> --body <body>`.
- **Invariant #1 still holds:** the nvim writes a *data file*, not git — the agent does the
  git action (the `docflow round`). **Invariant #3 holds by construction:** the body
  reflects only what landed (the nvim computed it post-apply); `dropped` is surfaced (poke +
  the existing WARN), never silently swallowed.
- This must be pinned with a **dropped-record** test, not just the happy path where
  `new_occurrence` is trivially 1.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `review.poke_bodies` (commit-signal message builders) | `nvim/review/poke_bodies.lua` | new |
| `record` / `reconstruct` / `apply` (the consumer core) | `nvim/review/*.lua` | unchanged (PURE) |

(Its own standalone file — like `record.lua`/`reconstruct.lua` — so `nvim -l nvim/review/poke_bodies_test.lua` runs without `dofile`-ing the IO orchestrator. plan-quality INFO-1.)

- **`review.poke_bodies`** — pure functions building the agent-facing poke bodies: `agent_applied(applied, dropped, file)` → `"applied N edit(s) (M dropped) to <abs> — commit the agent round"`, `human_committed(file)` → `"committed my edits to <abs> — please review"`. PURE (string-only), colocated unit test. **DRY rationale:** the round→poke wording is the protocol's commit signal (review-protocol.md seam #3); one source so the nvim and any test assert the same strings. **Future extensions:** carries the active mode (M4b) once the mode seam exists. `apply` already returns `(enriched, dropped)`, so `record.embed_in_body` (the body in the landed-artifact) is unchanged — the same encoder, just authored into a file the agent commits instead of committed inline.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `review-landed-<tag>.json` | `$PAIR_DATA_DIR/` | new (seam #2b) | nvim→agent: the prepared agent-round commit payload |
| `review.on_agent_round` | `nvim/review/init.lua` | modified | (was) docflow agent-round → now apply+save+landed-artifact+poke |
| `review.human_round` | `nvim/review/init.lua` | modified | (was) docflow human-round → now save+poke |
| `pair_poke.send` | `nvim/pair_poke.lua` | reused (injectable) | zellij write-chars (the commit-signal poke) |
| `fake-agent-v2` | `tests/lib/fake-review-agent.sh` | modified | a protocol-faithful agent: writes records, **reads the landed-artifact, commits rounds** |

- **`review-landed-<tag>.json`** — the nvim→agent payload (the Decision above): `{summary, body, applied, dropped}`, `body` = `record.embed_in_body(summary, enriched)`. Written by `on_agent_round`; read by the agent/fake to commit the agent round verbatim. **Future:** carries the active mode (M4b).
- **`review.on_agent_round(buf, records)`** — applies undo-ably (unchanged), saves, **writes `review-landed-<tag>.json`**, then **pokes `agent_applied(#enriched, #dropped, file)`** instead of `docflow.round('agent', …)`. The commit is the agent's, *after* the poke (invariant #3). **`file` comes from `sessions[buf].file`** (the watcher only has `buf` — thread the session's file through). **Injected:** `pair_poke.send` is injectable (a module-level seam the headless test stubs) since `on_agent_round` is driven directly by `review-loop-test.sh` and must not shell `zellij`. Reconstruct/counts read the agent's resulting commit.
- **`review.human_round`** — saves the human's edits and signals via `human_committed` instead of `docflow.round('human', …)`. The agent commits the saved diff (`docflow round --side human`; no body/artifact — a human round carries no records). The poke is issued by `nvim/review.lua`'s `finish_human_turn` (the UI layer), which replaces the temporary `/xx-fix` body with `human_committed`. **Layering note:** the *agent* poke fires from the handoff watcher registered in `init.lua` (the orchestrator) while the *human* poke fires from `finish_human_turn` in `review.lua` (the UI) — a deliberate asymmetry (each poke fires where its trigger lives), not an accident to "consolidate."
- **`fake-agent-v2`** — extends M1's `fake-review-agent.sh`: after writing the handoff records it waits for `review-landed-<tag>.json`, then runs `docflow round --side agent -m <summary> --body <body>` (verbatim from the artifact) in the doc's repo; on the human-round signal it runs `docflow round --side human`. The **process-level fake** that proves the pair side end-to-end without the real SKILL (the agent owns git, as #000121 will) — and reading the artifact is exactly what keeps the fake faithful without intelligence.
- **`docflow.lua`** (nvim) — its `round`/`ship` writers become **unused by the nvim** (agent-only). Kept as a thin module only if a read (`status`) is still needed; otherwise its calls are removed from `init.lua`. (Decide in Task 1; don't delete blindly — `reconstruct`/counts read git directly, not via docflow.lua.)

**ARCH-PURE:** the change deletes IO from the seam (the nvim's git writes leave entirely); the pure core is untouched. **ARCH-DRY:** reuse `pair_poke` + the `record`/`reconstruct` core; the commit-signal wording lives once in `poke_bodies`.

---

## Milestone 4a — real loop + agent-owns-git unwind

> **Dependency:** the *live* loop needs ariadne **#000121** M4a (the SKILL recognizing review-mode + owning git). Everything below is provable headlessly with `fake-agent-v2`; the live smoke (Task 5) waits on #000121.

### Task 1: `on_agent_round` stops writing git — apply + save + landed-artifact + commit-signal poke

**Files:** Modify `nvim/review/init.lua`; extend `tests/review-loop-test.sh` (+ a pure `poke_bodies` assertion); reuse `nvim/pair_poke.lua` (injectable)

- [x] **Step 1a: failing test (pure)** — `poke_bodies.agent_applied(2, 1, '/a/doc.md')` returns the exact string (incl. the "(1 dropped)" segment); `agent_applied(2, 0, …)` omits the dropped segment.
- [x] **Step 1b: failing test (integration, the dropped-record case — invariant #3)** — drive `on_agent_round` with TWO records where one **cannot anchor** (so `apply` drops it), `pair_poke.send` stubbed to record, `docflow` stubbed to record. Assert: (a) **no `round --side agent`** invoked by the nvim; (b) `review-landed-<tag>.json` written with `applied=1, dropped=1` and `body` = `record.embed_in_body` of the **one** enriched record (the dropped one absent); (c) the poke body carries `agent_applied(1, 1, file)`.
- [x] **Step 2: run → fail.**
- [x] **Step 3: implement** — add pure `nvim/review/poke_bodies.lua` (+ colocated `poke_bodies_test.lua`, run under `make test-lua`); make `pair_poke.send` injectable (module-level, default the real send; the test swaps a recorder). In `on_agent_round`: thread `file` from `sessions[buf].file`; after `apply` + `save`, write `review-landed-<tag>.json` (`{summary, body=record.embed_in_body(summary, enriched), applied=#enriched, dropped=#dropped}`); `pair_poke.send(poke_bodies.agent_applied(#enriched, #dropped, file))`; **remove** the `check(docflow.round('agent', …))` call. (Apply/projection/decorate unchanged.)
- [x] **Step 4: run → pass.**
- [x] **Step 5: commit** — `#66 M4a: on_agent_round — apply+save, write landed-artifact, commit-signal poke (nvim writes no agent round)`.

### Task 2: `human_round` stops writing git — save + commit-request poke

**Files:** Modify `nvim/review/init.lua`, `nvim/review.lua`; extend the test

- [x] **Step 1: failing test** — a driven `human_round` asserts the buffer is saved and **no `round --side human`** is invoked; and `finish_human_turn` pokes the `human_committed` body (not the temporary `/xx-fix …`).
- [x] **Step 2: run → fail.**
- [x] **Step 3: implement** — `human_round`: keep `save(buf)`, drop `check(docflow.round('human', …))`. In `nvim/review.lua` `finish_human_turn`, swap `poke.send(REVIEW_TRIGGER .. ' ' .. abs)` for `poke.send(poke_bodies.human_committed(file))`. (Retire `REVIEW_TRIGGER` once #000121 makes the agent review-mode-aware; until then, a one-line note.)
- [x] **Step 4: run → pass.**
- [x] **Step 5: commit** — `#66 M4a: human_round — save+commit-request poke (nvim writes no human round)`.

### Task 3: `fake-agent-v2` — the agent owns the round commits

**Files:** Modify `tests/lib/fake-review-agent.sh`; Modify `tests/review-loop-test.sh`

- [x] **Step 1: failing test** — update `review-loop-test.sh` to expect the **agent** (fake-agent-v2), not the nvim, to have created the `review(<slug>): agent r1` / `human r1` commits, with the agent-round **body taken verbatim from `review-landed-<tag>.json`** (so the committed records == what landed); the nvim only applies + the decorations/counts reconstruct from those agent commits.
- [x] **Step 2: run → fail** (the old test asserted nvim-made commits).
- [x] **Step 3: implement** — fake-agent-v2: after emitting the handoff records, **wait for `review-landed-<tag>.json`**, then `docflow round --side agent -m <summary> --body <body>` (both from the artifact via `jq`) in the doc's repo; on the human-round signal, `docflow round --side human`. (Same `DOCFLOW_BIN` fake/real as M1; reading the artifact is what keeps the no-intelligence fake faithful.)
- [x] **Step 4: run → pass.**
- [x] **Step 5: commit** — `#66 M4a: fake-agent-v2 — reads landed-artifact, agent owns round commits (relocates M1's nvim-commit e2e)`.

### Task 4: counts + decorations reconstruct from the agent's commits

**Files:** Modify `tests/review-loop-test.sh` (or `review-indicator-test.sh`)

- [x] **Step 1: failing test** — on the `review/<slug>` branch the fake-agent-v2 built, assert `_pair_review_bar(file)` shows the live `🤖N/M` (no longer `0/0`) and decorations reconstruct from the agent commit body.
- [x] **Step 2: run → fail / Step 3: confirm** the read paths already work (they read git directly) — likely green once Task 3 lands; if not, fix the read.
- [x] **Step 4: run → pass.**
- [x] **Step 5: commit** — `#66 M4a: indicator counts + decorations reconstruct from agent commits (go live)`.

### Task 5: live smoke (real agent, gated on ariadne #000121 M4a)

Manual, in a real pair session once #000121 M4a lands:

- [ ] Poke a review → the agent (real SKILL) recognizes review-mode, proposes records to the handoff, the pane applies + styles, and **the agent** commits the agent round; the bar's `🤖N/M` ticks.
- [ ] Edit + Alt+Return → the agent commits the human round; undo stays continuous; no nvim-side git.
- [ ] Record the smoke in `## Log`.

This smoke remains the real-agent proof for ariadne #000121. It is not a separate M4a
review boundary after the 2026-06-21 reconciliation; include it in the current M4
skeleton boundary evidence when available.

### Task 6: milestone close

- [ ] `make test-lua` + `make test-review` green; manual smoke recorded when the real-agent dependency is available.
- [x] Update `atlas/review-workbench.md` / `review-protocol.md` for the nvim-writes-no-git invariant and the commit-signal seam.
- [ ] Close the **current M4 skeleton boundary** once M4b's remaining structure is in place. Do not retroactively `milestone-close M4a`; the branch already crossed that line before reconciliation.

---

## Open details to resolve in-milestone (M4a)

- **`docflow.lua` in the nvim** — after Tasks 1–2 the nvim calls no `docflow` writer. Keep the module only if a `status` read earns it; otherwise drop its `init.lua` usage (don't delete the file blindly — M4b–d / the fake may still want it). Decide in Task 1.
- **Commit-signal timing + body authorship** — *resolved* by the landed-artifact Decision: the agent commits *after* the nvim's "applied" poke (invariant #3), reading the body verbatim from `review-landed-<tag>.json` (so the committed records == what landed, drops surfaced). With fake-agent-v2 the ordering is one script; with the real SKILL it's the poke→read-artifact→commit handshake. The dropped-record test (Task 1b) pins it.
- **`REVIEW_TRIGGER` retirement** — the `/xx-fix` poke is a stopgap; it retires when #000121 makes the agent recognize review-mode from the `human_committed`/`agent_applied` signals. Keep the constant until then.

## Revisions

### 2026-06-20 — M4a Tasks 1–4 as-built (commits `2a4d95d`, `fabfaf8`; 103 checks green)

Implemented per the plan, with these deltas (the as-planned text above is the record; this is the delta):

- **Landed-artifact location: the handoff's data dir, not `$PAIR_DATA_DIR`.** It lives at
  `$XDG_DATA_HOME/pair/review-landed-<tag>.json` via `nvim/review/handoff.lua`
  (`write_landed`/`landed_path`), NOT `$PAIR_DATA_DIR` + `seam.lua` as the Decision section
  said. Rationale: it's the **reverse channel of the handoff** (agent↔nvim record channel),
  so it co-locates with seam #2 — whereas `seam.lua`/`$PAIR_DATA_DIR` own the draft↔pane
  files (`.open`/`.mode`). Target seam #2b reconciled to this.
- **`review.start` no longer calls `docflow.start`** (beyond literal Tasks 1–2): the agent
  owns the `review/<slug>` branch too (seam #4, invariant #1). So the nvim calls `docflow`
  **nowhere** — the `docflow` dofile + `check()` were removed from `init.lua`; `pair_poke.send`
  is injectable as `M.poke` so headless tests record without shelling zellij.
- **`docflow.lua` kept** (the Task-1 open-detail decision): the nvim no longer uses it, but
  it + `review-docflow-test` now guard the **docflow commit-shape contract** the agent must
  produce (same subject/author/body the nvim reconstructs from). Flagged as a clean-removal
  candidate for the M4a boundary review.
- **Invariant #1 is BUILT (pair side)** — flipped in the target ahead of Task 6 since it's
  headless-verified; the live smoke (Task 5, gated on #000121) confirms it with the real agent.
- **For the live smoke / #000121:** the real SKILL must (a) create `review/<slug>` on
  review-start, (b) read `review-landed-<tag>.json` from `$XDG_DATA_HOME/pair` and commit the
  agent round verbatim, (c) commit the human round on the `human_committed` poke.
  `tests/lib/fake-review-agent.sh` (fake-agent-v2) is the exact protocol reference.

### 2026-06-21 — re-slice structure-first (operator)

The original M4 slices were feature-columns (M4b modes, M4c voice/fact-check, M4d
ship/e2e). Re-sliced to **get the whole loop standing thin before any tuning**:
- **M4a'** carved out (review-start & resume) — surfaced by the live smoke (no
  review-START signal; resume unhandled). Spec lives in the protocol target.
- **M4b = skeleton**: pulled **ship** *up* from the old M4d and the 🤖[]
  conversation + accept/reject *up* into one milestone, with just a **default**
  posture — the minimal set that makes the cycle usable end-to-end.
- **M4c = thicken**: everything else (modes menu/UI, voice, fact-check, quickfix,
  display polish, rename, the faithful e2e demo) becomes tuning, sub-sliced when
  reached. So no single feature is polished before the skeleton stands.

### 2026-06-21 — continuation reconciliation: fold crossed M4a/M4a' evidence into current boundary

Continuation `pair-pair` resumed after M4a, M4a', early M4b, and unwired M4c commits had
already landed without separate `sdlc milestone-close` boundaries. Reconciliation decision:
do not manufacture retrospective M4a/M4a' closes over an interleaved commit window. Treat
M4a + M4a' as implemented evidence inside the current M4 skeleton boundary, keep M4b open
for fulfill/punt/default posture/ship, and record the spinner helper as M4c pre-work.
