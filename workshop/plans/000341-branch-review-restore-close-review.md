# Boundary Review — pair#341 (whole-issue close)

| field | value |
|-------|-------|
| issue | 341 — Alt+C restores review target from branch |
| repo | pair |
| issue file | workshop/issues/000341-branch-review-restore.md |
| boundary | whole-issue close |
| milestone | — |
| window | 41ab4f9add1908e854c3f41e491e4a64d53793db..af18ceaad04fb3862bf803f679b6435c78c060f3 |
| command | sdlc close --issue 341 |
| reviewer | codex |
| timestamp | 2026-09-28T21:33:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation covers branch restoration, authenticated activation, and bounded recovery well. A reproduced handoff-loss race blocks shipping. Routine typing also performs synchronous history scans, and the new operator commands are missing from README.

1. **Strengths**

   - The shared Go resolver pins HEAD, validates exact round subjects and safe paths, and bounds history reads.
   - Real-process tests verify A → B → A restoration, fresh-session isolation, and recovery across process exit.
   - Activation preserves retained buffers and restores the previous owner when setup fails.
   - Recovery snapshots have private permissions, atomic replacement, capacity limits, and explicit removal rules.

2. **Critical findings**

   - **Handoff consumption precedes final acceptance.** [handoff.lua:90](/Users/xianxu/workspace/pair/nvim/review/handoff.lua:90) deletes the payload before calling `on_agent_round`, whose [authorization check](/Users/xianxu/workspace/pair/nvim/review/init.lua:127) can still refuse. A checkout change between these checks loses the response without applying it. A controlled probe using the production functions produced `payload_exists=false text=before`. Require explicit apply/defer acceptance before consuming the payload; preserve it on refusal. **ARCH-ORDER, ARCH-PURPOSE.**

3. **Important findings**

   - **Typing synchronously scans Git history.** [review.lua:848](/Users/xianxu/workspace/pair/nvim/review.lua:848) runs recovery checks on every `TextChanged`/`TextChangedI`. Modified buffers reach `guard → resolve →` [identity.lua:23](/Users/xianxu/workspace/pair/nvim/review/identity.lua:23), blocking Neovim for up to 2.5 seconds per resolver call. Use asynchronous observation for proactive recovery, retaining authoritative checks at write/apply boundaries. Add a delayed-resolver responsiveness test. **ARCH-CONSTRAINTS.**
   - **README update missing.** [README.md:139](/Users/xianxu/workspace/pair/README.md:139) retains the old target-based Alt+C description and omits `:PairReviewRecover` and `:PairReviewDiscardRecovery`. Document branch restoration, blocked transitions, and recovery commands alongside the existing atlas update.

4. **Minor findings**

   - [review.lua:790](/Users/xianxu/workspace/pair/nvim/review.lua:790) registers rendering autocmds on every activation without removing earlier registrations. A real A → B → A probe increased A’s `TextChanged` callback count from one to two. Use an owned, clearable group and test stable registration counts. **ARCH-FUNERAL.**

5. **Test coverage notes**

   Passed: Go review package tests; all six new Lua review test modules; branch-restoration, fresh-restoration, and producer-context acceptance tests.

   `make test-lua` stopped in `scrollback_test.lua` with an editor-storage sandbox permission error. The full suite was therefore not verified. Existing handoff tests cover refusal before admission, but miss refusal after admission and before application.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared identity resolver replaces independent history interpretation.
   - **ARCH-PURE — pass:** declared pure entities are separated from integration code.
   - **ARCH-PURPOSE — flag:** handoff loss violates preservation of pending work.
   - **ARCH-MOCK — pass:** stateful host/producer fixtures and real Git/Neovim tests exercise integration boundaries.
   - **ARCH-CONSTRAINTS — flag:** synchronous history scans enter the typing path.
   - **ARCH-SECURE — pass:** identity, paths, context, and recovery storage receive explicit validation.
   - **ARCH-ORDER — flag:** deletion occurs before final acceptance.
   - **ARCH-FUNERAL — flag:** activation-owned autocmd registrations accumulate.

7. **Plan revision recommendations**

   Append a timestamped `## Revisions` entry specifying acceptance-before-consumption, asynchronous recovery observation, and autocmd ownership. Add their regression cases and README coverage to the active checklist.

```findings
findings:
  - id: new
    severity: Critical
    family: consume-after-acceptance
    title: |
      Handoff deletion precedes final application authorization
    detail: |
      nvim/review/handoff.lua:90 deletes before the callback; nvim/review/init.lua:127 can subsequently refuse after checkout movement. A controlled production-function probe confirmed payload loss without application. Consume only after explicit apply/defer acceptance and test this interleaving (ARCH-ORDER, ARCH-PURPOSE).
  - id: new
    severity: Important
    family: nonblocking-editor-observation
    title: |
      Routine typing synchronously resolves full Git history
    detail: |
      nvim/review.lua:848 routes TextChanged/TextChangedI through preserve_mismatch and guard to nvim/review/identity.lua:23, synchronously waiting up to 2.5 seconds per resolver call. Make proactive recovery observation asynchronous and test responsiveness with delayed resolution (ARCH-CONSTRAINTS).
  - id: new
    severity: Important
    family: user-surface-documentation
    title: |
      README omits branch restoration and recovery commands
    detail: |
      README.md:139 retains the previous target-based Alt+C description. This range introduces PairReviewRecover and PairReviewDiscardRecovery without any README update; document the changed behavior and recovery workflow.
  - id: new
    severity: Minor
    family: activation-resource-ownership
    title: |
      Repeated activation accumulates rendering autocmds
    detail: |
      nvim/review.lua:790 registers callbacks on every activation without corresponding cleanup. A real A-to-B-to-A probe increased A's TextChanged callback count from one to two. Use a clearable owned group and assert stable counts (ARCH-FUNERAL).
```

---

## Re-review — 2026-09-28T22:00:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 341 — Alt+C restores review target from branch |
| repo | pair |
| issue file | workshop/issues/000341-branch-review-restore.md |
| boundary | whole-issue close |
| milestone | — |
| window | 41ab4f9add1908e854c3f41e491e4a64d53793db..e03a792b4a8fbc8e5dc04684ada8f6154307cdb1 |
| command | sdlc close --issue 341 |
| reviewer | codex |
| timestamp | 2026-09-28T22:00:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All four prior findings are addressed, with meaningful regression evidence. One new blocking race remains: a delayed asynchronous observation can reload another branch’s bytes into a pane still bound to the original review. The repository remains unchanged.

1. **Strengths**

   - Handoff deletion now follows explicit acceptance; uncertain outcomes and replacement generations remain preserved (`nvim/review/handoff.lua:96`).
   - Git identity resolution enforces exact subjects, unique safe paths, bounded reads, and final branch/HEAD checks.
   - Activation rendering callbacks have explicit cleanup ownership (`nvim/review.lua:725`).
   - README and atlas cover restoration, blocked transitions, and recovery.

2. **Critical findings**

   **Late observation authorizes a fresh read from a different checkout** — `nvim/review/recovery_observer.lua:36` and `nvim/review.lua:843`.

   The observer checks its captured Git result against the pane’s unchanged binding, then invokes `checktime`, which reads the *current* checkout. I reproduced this through the production pane: capture resolution on `review/a`, pause delivery, checkout `review/b`, then deliver the result. The pane displayed B’s `a.md` bytes (`a`) while its binding remained `review/a`; previously it displayed `a reviewed`.

   **This is the 2nd finding in family `nonblocking-editor-observation`.** State and enforce the rule across asynchronous observation consumers: an observation cannot authorize a later, independently sourced read. Sweep refresh, preservation, and coalescing decisions. For reload, obtain bytes tied to a verified identity snapshot and apply those bytes; preserve asynchronous responsiveness. Add a controlled observation-before-checkout / completion-after-checkout regression. **ARCH-ORDER, ARCH-PURPOSE.**

3. **Important findings**

   None additional.

4. **Minor findings**

   None additional.

5. **Test coverage notes**

   - Passed: Go reviewcmd package tests; targeted Lua handoff, observer, controller, policy, recovery, definition-seam and client tests; observation, branch-restoration, and producer-context integration tests.
   - BR-1 mutation: restoring the old watcher failed the assertion that final application refusal preserves the handoff.
   - BR-2 mutation: restoring synchronous edit observation failed with approximately **1,155 ms** blocking and six scans.
   - BR-4 mutation: removing callback cleanup failed with callback counts increasing **1 → 3**.
   - The existing observation test passes but misses the newly reproduced ordering.
   - Handoff tests required temporary XDG storage under this sandbox; they passed with that isolation.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared identity interpretation and artifact-generation receipts.
   - **ARCH-PURE — pass:** identity classification and activation policy remain directly testable without IO.
   - **ARCH-PURPOSE — flag:** late reload violates branch-isolated review restoration.
   - **ARCH-MOCK — pass:** reviewed integration paths have controlled seams and real temporary-Git/Neovim coverage.
   - **ARCH-CONSTRAINTS — pass:** bounded resolution and coalesced asynchronous observations; responsiveness regression verified.
   - **ARCH-SECURE — pass:** reviewed paths validate canonical identity, session/activation context, and recovery storage.
   - **ARCH-ORDER — flag:** historical observation is treated as current authority for reload.
   - **ARCH-FUNERAL — pass:** activation callbacks and observer work have cleanup; recovery storage has bounded admission and removal.

7. **Plan revision recommendations**

   Append a `## Revisions` entry defining identity-bound reload snapshots and enumerating asynchronous callback decisions. Require the controlled late-completion regression before closing the boundary.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Handoff consumption follows explicit apply/defer acceptance. The passing production-function regression fails with the old watcher at the final-refusal preservation assertion.
  - id: BR-2
    disposition: addressed
    note: |
      Edit observation is asynchronous and coalesced. Restoring synchronous observation fails the responsiveness test at approximately 1155 ms with six scans. The distinct late-completion safety regression is reported below.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:99 documents branch restoration and blocking conditions; README.md:108 documents recovery/discard commands implemented in nvim/review.lua.
  - id: BR-4
    disposition: addressed
    note: |
      Activation owns a clearable rendering group. The passing observation test fails with callback counts increasing from one to three when cleanup is removed.
findings:
  - id: new
    severity: Critical
    family: nonblocking-editor-observation
    title: |
      Late asynchronous observation reloads another branch into the active review
    detail: |
      nvim/review/recovery_observer.lua:36 accepts a captured matching identity, then nvim/review.lua:843 invokes checktime against the current checkout. A controlled production-pane probe captured review/a, switched to review/b before delivery, and loaded B's bytes while the pane remained bound to A. This is the 2nd finding in this family: enforce identity-bound observation effects across refresh, preservation, and coalescing decisions rather than patching only this callback. Reload verified snapshot bytes and add a controlled late-completion regression without restoring synchronous editor observation. ARCH-ORDER, ARCH-PURPOSE.
```

---

## Re-review — 2026-09-28T22:17:55-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 341 — Alt+C restores review target from branch |
| repo | pair |
| issue file | workshop/issues/000341-branch-review-restore.md |
| boundary | whole-issue close |
| milestone | — |
| window | 41ab4f9add1908e854c3f41e491e4a64d53793db..b7c2cc78c43842bbdbef6cfbe1095344f9529a2f |
| command | sdlc close --issue 341 |
| reviewer | codex |
| timestamp | 2026-09-28T22:17:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The late-delivery fix has meaningful regression coverage, but BR-5 remains reproducible during an in-progress checkout. A separate activation bug corrupts CRLF documents on save. Both block shipping; the repository was left unchanged.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      Captured snapshots fix callback-time rereads, but identity.go:209,220-226 can still label B bytes as A during checkout. A deterministic real-Git probe paused checkout A→B in a smudge filter: index.lock existed, HEAD still named review/a, and a.md already contained B bytes. The production resolver returned status resolved, branch review/a, snapshot "B bytes\n". Enforce snapshot authority across concurrent checkout mutation, with fail-closed retry and a controlled in-progress-checkout regression. ARCH-ORDER, ARCH-PURPOSE; existing nonblocking-editor-observation family.
  - id: BR-1
    disposition: addressed
    note: |
      Consumption follows explicit acceptance; acceptance, uncertainty, replacement, and refusal regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      Typing/focus callbacks use coalesced asynchronous observation; delayed-resolver responsiveness tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      README documents branch restoration, blocked switching, and recover/discard commands, consistent with the implemented interfaces.
  - id: BR-4
    disposition: addressed
    note: |
      Activation clears owned rendering callbacks; repeated activation callback-count regression passes.
findings:
  - id: new
    severity: Critical
    family: document-byte-preservation
    title: |
      Activating a CRLF document corrupts its bytes on subsequent save
    detail: |
      nvim/review/restore_controller.lua:132-146 reads binary lines retaining carriage returns, then inserts them into a buffer whose bufload selected fileformat=dos. A controller probe activating a file containing "B\r\n" produced buffer line "B\r"; writing saved "B\r\r\n". Share byte-to-buffer decoding with the asynchronous refresh path, including fileformat and endofline handling. Add activation-and-save regressions for new and retained buffers, including branch-driven format changes. ARCH-DRY, ARCH-PURPOSE.
```

1. **Strengths**

   - The observation regression controls delivery after checkout movement. Replacing snapshot application with a disk reread in a scratch copy makes its assertion fail.
   - Handoff acceptance tests exercise refusal, partial-effect uncertainty, replacement, and cleanup failure.
   - README and atlas describe restoration, recovery, and activation ownership.

2. **Critical findings**

   - **BR-5 remains open:** [identity.go:209](/Users/xianxu/workspace/pair/cmd/internal/reviewcmd/identity.go:209). Unchanged HEAD/branch does not establish working-tree stability during checkout. Extend the existing family rule to snapshot acquisition, and enumerate capture, delivery, preservation, and coalescing interleavings.
   - **CRLF corruption:** [restore_controller.lua:132](/Users/xianxu/workspace/pair/nvim/review/restore_controller.lua:132). Consolidate document decoding with [review.lua:852](/Users/xianxu/workspace/pair/nvim/review.lua:852), then verify exact saved bytes.

3. **Important findings:** None additional.

4. **Minor findings:** None.

5. **Test coverage**

   Go race tests, the full Lua suite, and every review shell script passed. Lua and fresh-session tests required isolated environment settings after inherited settings caused storage failures. Neither green suite covers the two reproduced failures above.

6. **Architecture**

   - **ARCH-DRY — flag:** activation and refresh duplicate incompatible document decoding.
   - **ARCH-PURE — pass:** identity classification and activation policy have directly tested pure components.
   - **ARCH-PURPOSE — flag:** branch-correct restoration and byte preservation remain incomplete.
   - **ARCH-MOCK — pass:** injected seams and real temporary Git/Neovim tests provide useful coverage.
   - **ARCH-CONSTRAINTS — pass:** observation is asynchronous, coalesced, and bounded.
   - **ARCH-SECURE — flag:** a snapshot receives stronger identity authority than concurrent-checkout evidence supports.
   - **ARCH-ORDER — flag:** checkout’s intermediate working-tree state is omitted.
   - **ARCH-FUNERAL — pass:** snapshots have bounded admission and explicit consumption; observers and activation callbacks have cleanup owners.

7. **Plan revision recommendations**

   Append a `## Revisions` entry covering checkout-in-progress snapshot admission and shared document decoding. Add controlled checkout-overlap and exact-byte activation/save regressions to the verification contract.

---

## Re-review — 2026-09-28T22:39:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 341 — Alt+C restores review target from branch |
| repo | pair |
| issue file | workshop/issues/000341-branch-review-restore.md |
| boundary | whole-issue close |
| milestone | — |
| window | 41ab4f9add1908e854c3f41e491e4a64d53793db..90d6b09e3ace106759e34036a7620ab8c5b4d0c2 |
| command | sdlc close --issue 341 |
| reviewer | codex |
| timestamp | 2026-09-28T22:39:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-5 and BR-6 are addressed, with passing regressions and verified mutation failures. Branch restoration and captured-byte delivery work in the exercised scenarios. One new Important finding blocks shipping: the fresh-session test can overwrite caller-owned session data through inherited environment variables.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Acceptance precedes consumption; handoff acceptance, refusal, replacement, and uncertainty regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      Controlled slow-resolver tests pass for nonblocking edit/focus callbacks and coalesced observations.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:99 documents branch restoration, blocked transitions, and recovery commands, matching the implemented controller and recovery commands.
  - id: BR-4
    disposition: addressed
    note: |
      The production observation test verifies stable rendering callback counts across repeated A → B → A activation.
  - id: BR-5
    disposition: addressed
    note: |
      Captured snapshots reach activation and refresh without later filesystem reads. Independently disabling index-generation admission and staged-tree rejection in scratch overlays makes the paused-checkout and index-before-HEAD regressions fail.
  - id: BR-6
    disposition: addressed
    note: |
      Activation and refresh share document_bytes decoding. New/retained-buffer exact-save regressions pass across CRLF, BOM, LF, mixed endings, empty and no-EOL documents; removing CRLF decoding in a scratch copy makes the regression fail.
findings:
  - id: new
    severity: Important
    family: test-environment-isolation
    title: |
      Fresh-session regression overwrites inherited review context outside its fixture
    detail: |
      tests/review-fresh-restore-test.sh:109 inherits os.environ without rebinding PAIR_REVIEW_CONTEXT_PATH. Its finish_human_turn call at line 71 reaches nvim/review.lua:540 and overwrites that caller-owned path. Reproduced with a scratch sentinel outside the fixture: the test passed while replacing its contents with "A reviewed\n". Sanitize inherited Pair session variables, bind every writable artifact to fixture storage, and add a sentinel regression. Sweep the new branch/observation fixtures using the same environment construction. ARCH-SECURE.
```

1. **Strengths**

   - Checkout admission covers both an active Git writer and destination-index publication before HEAD changes.
   - Startup, activation, and asynchronous refresh consume identity-bound snapshots.
   - Shared byte decoding preserves document formats; exact-save tests exercise new and retained buffers.
   - README and atlas describe restoration, recovery, and the conservative staged-index restriction.

2. **Critical findings**

   None remaining.

3. **Important findings**

   **Test isolation:** [review-fresh-restore-test.sh:109](/Users/xianxu/workspace/pair/tests/review-fresh-restore-test.sh:109) inherits a writable path from the invoking session. The production write occurs at [review.lua:540](/Users/xianxu/workspace/pair/nvim/review.lua:540). Construct an isolated child environment and verify externally supplied sentinel files remain unchanged.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: Go review package tests, the full Lua suite with isolated environment, controller/document-byte tests, branch-restoration and observation integration tests, and fresh-session scenarios with controlled environment. Checkout-admission and CRLF-decoding mutations failed as expected.

   The ordinary inherited-environment runs exposed storage leakage; the sentinel reproduction confirmed an actual overwrite, not merely a sandbox limitation. Repository status is clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared identity resolver and document codec.
   - **ARCH-PURE — pass:** classification and activation policy remain separate from IO.
   - **ARCH-PURPOSE — pass:** restoration covers live panes, fresh sessions, retained buffers, and late observations.
   - **ARCH-MOCK — pass:** stateful doubles are supplemented by real Git and Neovim tests.
   - **ARCH-CONSTRAINTS — pass:** bounded collection and coalesced observation are exercised.
   - **ARCH-SECURE — flag:** test subprocesses inherit caller-owned writable state.
   - **ARCH-ORDER — pass:** controlled late completions and intermediate checkout states have regression coverage.
   - **ARCH-FUNERAL — pass:** recovery storage is bounded with explicit consumption/discard; owned callbacks and handles have cleanup.

7. **Plan revision recommendation**

   Append a `## Revisions` entry requiring isolated test environments across the new process fixtures and sentinel coverage proving inherited session artifact paths cannot be modified.
