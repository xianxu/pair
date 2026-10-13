# Boundary Review — pair#533 (whole-issue close)

| field | value |
|-------|-------|
| issue | 533 — draft: show progress while a DOM submission prepares, and hold the draft read-only until it's sent or fails |
| repo | pair |
| issue file | workshop/issues/000533-draft-show-progress-while-a-dom-submission-prepares-and-hold-the-draft-read-only-until-it-s-sent-or-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7efc4f904ece07c6fa2b554e9002e16dafde11df..c4f2b8f13c013c08f1f88d256dd91c6b7d443ef3 |
| command | sdlc close --issue 533 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T14:47:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Progress propagation, deadline handling and visible UI feedback are well covered, but the new read-only lock permits navigation to partially mutate draft state. I reproduced this overwriting an unrelated queued item after submission failure. That blocks shipping.

1. **Strengths**

   - Backend progress uses immutable snapshots, mutex protection and job-identity fencing.
   - The reducer preserves active operation ownership and separates late delivery receipts from draft clearing.
   - Embedded Neovim tests verify rendered progress before held external work; subprocess tests exercise actual timeout enforcement.
   - README and atlas document the changed behavior.

2. **Critical findings**

   **Read-only ownership does not guard draft navigation — ARCH-ORDER, ARCH-PURPOSE.**

   The lock in [nvim/init.lua:1666](/Users/xianxu/Library/Caches/sdlc/review-checkouts/887ec09011d723e1/nvim/init.lua:1666) rejects buffer writes, but `go_to` changes `nav.pos` before attempting that write (`nvim/init.lua:2527`). The resulting exception leaves the logical slot pointing at content that was never loaded.

   Reproduced against the pinned head with private files:

   - Draft contains `draft`; queue item contains `queued`.
   - Start DOM submission and hold preparation.
   - Press Alt+Right: navigation raises “Buffer is not 'modifiable'”.
   - Return preparation failure, then press Alt+Left.
   - `autosave_current_slot` writes `draft` over the queue item (`nvim/init.lua:2439`).

   Guard actions affecting the owned draft **before** changing navigation state or files. Sweep navigation, queueing, deletion, reset and send/clear actions for the same partial-mutation pattern. Add production integration tests asserting slot identity and persisted contents through failure and timeout.

3. **Important findings**

   None separately.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: Lua reducer/adapter/transport tests; sender tests under Neovim; DOM headless and embedded UI suite; Zellij timeout/redaction tests; Go submission, couchmessage, recorder and webcast packages.

   The checkout lacked generated runtime assets. After generating them in a disposable copy, couchcmd passed; the combined couchcmd/couchtty command reached the review’s 180-second limit before couchtty reported a result. No couchtty or full-suite pass is claimed. The disposable copy was removed.

   Existing lock tests reject direct buffer edits but omit navigation during an active operation—the reproduced failure class. No prior review findings required disposition.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing polling, reducer and sender seams are reused.
   - **ARCH-PURE — pass:** transition logic remains value-only; IO stays in adapters. Core-concept locations and classifications match the code.
   - **ARCH-PURPOSE — flag:** preserving authored state requires guarding existing draft actions, beyond rejecting text edits.
   - **ARCH-MOCK — pass:** controlled cast/CDP and pane fixtures share production seams.
   - **ARCH-CONSTRAINTS — pass:** remaining-time budgets reach DOM subprocesses; blocked-child tests verify enforcement.
   - **ARCH-SECURE — pass:** new progress formatting validates counts and escapes statusline text.
   - **ARCH-ORDER — flag:** submission ownership does not govern navigation’s state/file mutations.
   - **ARCH-FUNERAL — pass:** new progress is volatile; operation timers have retirement paths.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the policy for navigation and draft-mutating actions during active submission, enumerating affected handlers and requiring persisted-content regression tests. Keep final verification pending until its actual result is recorded.

```findings
findings:
  - id: new
    severity: Critical
    family: operation-ownership-must-guard-mutators
    title: |
      Read-only DOM preparation permits partial navigation that overwrites queued text
    detail: |
      nvim/init.lua:1666 applies nomodifiable without guarding go_to, which changes nav.pos at line 2527 before its buffer write fails. Reproduced: hold preparation with draft="draft" and queue="queued", press Alt+Right, return preparation failure, then press Alt+Left; autosave_current_slot at line 2439 overwrites the queue with "draft". ARCH-ORDER and ARCH-PURPOSE: guard owned-draft actions before state or file mutations, sweep navigation/queue/delete/reset/send-clear siblings, and add production regression tests preserving slot identity and stored contents through failure and timeout.
```

---

## Re-review — 2026-10-13T14:58:41-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 533 — draft: show progress while a DOM submission prepares, and hold the draft read-only until it's sent or fails |
| repo | pair |
| issue file | workshop/issues/000533-draft-show-progress-while-a-dom-submission-prepares-and-hold-the-draft-read-only-until-it-s-sent-or-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | 514ce3b4d36a51f403e9741004fd4697bdecffeb..88e05765662109a193b0e55440dd840fd96508b6 |
| command | sdlc close --issue 533 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T14:58:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Progress reporting, deadline propagation, and the original navigation correction are validated. BR-1 remains open because the ownership guard does not cover focus-event buffer mutations: losing focus during preparation can raise an exception and block the UI at a “Press ENTER” prompt.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The navigation correction passes production regressions; disabling its guard makes the Alt+Right regression fail. The required mutator sweep remains incomplete: nvim/init.lua:343 still writes from the scheduled FocusLost handler while DOM ownership sets nomodifiable. An embedded-UI reproduction enters insert mode at EOL during held preparation, fires FocusLost, and reaches “Buffer is not modifiable” followed by a blocking “Press ENTER” prompt. Cover focus padding/removal and other asynchronous mutators under the same ownership rule, with production event-sequence regressions (ARCH-ORDER, ARCH-PURPOSE).
```

1. **Strengths**

   - Progress comes from actual preparation/capture boundaries, with immutable snapshots and retired-job fencing.
   - The pure reducer retains active ownership and rejects stale completions; controlled tests exercise timeout and late delivery.
   - Production UI tests verify rendered progress before held external work.
   - README and atlas both document the new behavior; the Core concepts table matches the implemented entities.

2. **Critical findings**

   **BR-1 — incomplete ownership enforcement.** At `nvim/init.lua:343`, `pair_eol_ensure_pad` writes to the locked buffer through the scheduled callback at line 421. Reproduced using the embedded UI fixture after its active-lock assertion: enter insert mode at EOL, then execute `FocusLost`. The scheduled callback raises, displays “Press ENTER,” and blocks subsequent fixture RPCs.

   **This is the 2nd finding in family `operation-ownership-must-guard-mutators`.** Retain BR-1 rather than assigning another ID. Apply the rule to all synchronous and asynchronous draft mutators, including focus padding and removal (`nvim/init.lua:349`). Preserve deferred cleanup state when a write cannot proceed. Add focus-loss/gain sequences during preparation, failure, and timeout; verify no prompt, exception, or text corruption.

3. **Important findings**

   None newly raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: production DOM headless/embedded suite; Lua reducer, adapter, transport and sender tests; Zellij timeout/redaction checks; Go submission, couchmessage, recorder and webcast packages; focused CouchCMD/CouchTTY submission tests.

   Mutation verification confirmed the navigation regression fails without the guard. The independent focus-event reproduction fails on pinned head.

   Broader Go runs failed two pre-existing README delimiter checks and `TestCouchParentCaptureStableAcrossChildMouseModes`. Full-suite success is unverified. Generated assets and experiments were confined to a disposable checkout, now removed; tracked files remain unchanged.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: shared guard and existing polling reused. |
   | ARCH-PURE | Pass: reducer remains IO-free; adapters own effects. |
   | ARCH-PURPOSE | Flag: BR-1’s mutator class is incompletely covered. |
   | ARCH-MOCK | Pass: shared seams support controlled, stateful fixtures. |
   | ARCH-CONSTRAINTS | Flag: reproduced hit-enter prompt interrupts the bounded UI flow. |
   | ARCH-SECURE | Pass: progress validation, literal rendering and trace redaction retained. |
   | ARCH-ORDER | Flag: focus events bypass operation ownership. |
   | ARCH-FUNERAL | Pass: operation state is volatile; terminal paths retire its timer. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating asynchronous draft mutators alongside keybindings, explicitly including focus padding/removal. Require ownership-safe deferral and production focus-event regressions before considering BR-1 addressed.

---

## Re-review — 2026-10-13T15:10:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 533 — draft: show progress while a DOM submission prepares, and hold the draft read-only until it's sent or fails |
| repo | pair |
| issue file | workshop/issues/000533-draft-show-progress-while-a-dom-submission-prepares-and-hold-the-draft-read-only-until-it-s-sent-or-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | 514ce3b4d36a51f403e9741004fd4697bdecffeb..16f6b7fd9ee457f42a7fb1a56aeb158a97e4cd9f |
| command | sdlc close --issue 533 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T15:10:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned implementation satisfies the submission-progress and temporary read-only contracts. BR-1 is addressed, including asynchronous focus padding; production regressions pass and fail when the relevant guards are removed. No additional findings. Full-suite and live-Chrome results were not independently established in this review and remain subject to the plan’s pre-landing verification requirement.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Entry guards protect navigation and draft mutations; focus padding now respects ownership and retains deferred cleanup. Production draft/queue failure/timeout cases and embedded focus tests pass. Removing navigation guards makes dom_submission_integration_test.lua:72 fail; removing the focus guard at init.lua:338 reproduces nomodifiable, Press ENTER, and RPC timeout in dom-submission-ui-test.lua:129.
```

**1. Strengths**

- Progress follows actual preparation work through immutable, mutex-protected snapshots (`cmd/internal/couchtty/console_submission.go:280`).
- The adapter establishes ownership and an absolute deadline before effects, releases ownership before clearing, and fences late callbacks (`nvim/dom_submission.lua:62`).
- Production tests verify rendered status, persisted draft/queue contents, and recovery—not merely mocked call counts.
- README and atlas both document the changed interaction and timeout behavior.

**2. Critical findings:** None.

**3. Important findings:** None.

**4. Minor findings:** None.

**5. Test coverage notes**

Passed:

- DOM headless and embedded-UI suites.
- Lua reducer, adapter, and transport tests; sender tests under Neovim.
- Zellij timeout and trace-redaction tests.
- All six affected Go packages: `submission`, `couchmessage`, `couchcmd`, `couchtty`, `webcast/recorder`, and `webcast`.
- Pinned-range `git diff --check`.

Two Go packages initially lacked generated runtime assets; both passed after generation in the disposable copy. Both BR-1 mutation checks produced the expected failures. The disposable copy was removed, and the review checkout remains unchanged.

Full `make test`, race checks, and live-Chrome conformance were not rerun here.

**6. Architectural notes**

- **ARCH-DRY — Pass:** Reuses existing polling, submission reducer, capture pipeline, and a shared mutation guard.
- **ARCH-PURE — Pass:** Transition and formatting logic remain independent of IO; declared core entities match their implementations.
- **ARCH-PURPOSE — Pass:** Covers preparation, dispatch, terminal release, and the enumerated mutation paths, including asynchronous focus callbacks.
- **ARCH-MOCK — Pass:** Controlled manager/CDP/pane fixtures share production seams; existing live-conformance paths remain intact.
- **ARCH-CONSTRAINTS — Pass:** Absolute client budget propagates to external calls; repeated submission does not renew it.
- **ARCH-SECURE — Pass:** Progress fields are validated, status text is escaped, slot identity comes from the authenticated caller, and trace redaction remains tested.
- **ARCH-ORDER — Pass:** Production transitions use the reducer; tests exercise held, duplicate, expired, and late results without transferring ownership.
- **ARCH-FUNERAL — Pass:** Timers and operation contexts retire; progress adds no durable artifact family.

**7. Plan revision recommendations**

None. Existing BR-1 revisions accurately describe the correction. Record the outstanding stable-source full-suite result before landing, as the plan already requires.
