# Boundary Review — pair#779 (whole-issue close)

| field | value |
|-------|-------|
| issue | 779 — Couch tab bar: wrap onto a second line when it overflows, instead of hiding slots behind +N |
| repo | pair |
| issue file | workshop/issues/000779-couch-tab-bar-wrap-onto-a-second-line-when-it-overflows-instead-of-hiding-slots-behind-n.md |
| boundary | whole-issue close |
| milestone | — |
| window | fd596ab66cf1be1839109e6f98e5fb07a331f7a5..3e86e8f202a00e7329c21859f6234ef5f9e578eb |
| command | sdlc close --issue 779 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T17:24:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation and tests support the two-row behavior, shared geometry, input fencing and broadcast protections. No runtime defect was found. One plan/code contradiction blocks admission under the explicit Core concepts review rule.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: core-concepts-code-traceability
    title: |
      Core concepts table names planner entities absent from the implementation
    detail: |
      workshop/plans/000779-tab-bar-wrap-plan.md:33 declares statusPlan / planStatusRows in status_layout.go, but neither exists. Production uses statusPacking, renderStatusBar and packStatusChips; the verification table at lines 119–120 repeats planStatusRows. Update both tables and add a Revisions entry identifying the delivered pure planner entities (ARCH-PURE).
```

1. **Strengths**

   - Packing reuses the existing drawing pass, keeping clipping and clickable spans aligned.
   - `Frame.ChromeRows` supplies shared geometry for composition, privacy and annotation guards.
   - Controlled-writer tests exercise partial writes, suppressed mouse input and automatic confirmation recovery.
   - Both #757 evidence moves are verified as 100% identical renames.

2. **Critical findings**

   - Plan traceability mismatch above. This requires a documentation correction; no runtime change is indicated.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**

   - Couch and terminal packages passed.
   - Broadcast passed with `TMPDIR=/tmp`; the initial run failed an unchanged named-tunnel socket expectation.
   - Browser tests: 100 passed.
   - Packing fuzz: 57,909 executions passed.
   - Affected-reader inventory and diff whitespace checks passed.
   - Operator live acceptance remains explicitly pending in the plan.

6. **Architectural checks**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass — shared drawing and geometry helpers. |
   | ARCH-PURE | Flag — Core concepts names do not match delivered entities. Pure packing itself is directly tested. |
   | ARCH-PURPOSE | Pass — panes, both-row input, anchors and broadcast consumers are covered. |
   | ARCH-MOCK | Pass — stateful host/endpoint fixtures, loopback server tests and native PTYs. |
   | ARCH-CONSTRAINTS | Pass — bounded rows and counted planner complexity tests. |
   | ARCH-SECURE | Pass — trusted chrome metadata, spoofing negatives and private-body filtering. |
   | ARCH-ORDER | Pass — serialized admission and controlled paint-recovery tests. |
   | ARCH-FUNERAL | Pass — no new durable runtime artifacts or workers. |
   | ARCH-EVOLVE | Pass — additive live metadata; no persisted format changes. |

   Atlas updates cover the new architecture. No new command, flag, keybinding or configuration surface requires a README update.

7. **Plan revision recommendation**

   Add a `## Revisions` entry recording that the delivered pure planner uses `statusPacking`, `renderStatusBar` and `packStatusChips`, and correct both tables accordingly. No wording-presence test is needed.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"50bfc7ca-a6ad-4d32-866a-f38e6e595689","repository":"github.com/xianxu/pair","issue_id":"000779","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/6f5b46eff059e967","reviewed_head":"3e86e8f202a00e7329c21859f6234ef5f9e578eb","slot":"pair:4","requested_at":"2026-10-22T00:21:35.828566Z"},"phase":"finished","pid":87676,"started_at":"2026-10-22T00:21:35.933836Z","finished_at":"2026-10-22T00:24:52.47192Z","session_id":"01a160fc-1769-7be0-ab9b-583a5db19e6b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.831768417s
head: "3e86e8f202a00e7329c21859f6234ef5f9e578eb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T17:27:48-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 779 — Couch tab bar: wrap onto a second line when it overflows, instead of hiding slots behind +N |
| repo | pair |
| issue file | workshop/issues/000779-couch-tab-bar-wrap-onto-a-second-line-when-it-overflows-instead-of-hiding-slots-behind-n.md |
| boundary | whole-issue close |
| milestone | — |
| window | fd596ab66cf1be1839109e6f98e5fb07a331f7a5..b1c68720d3e721e9d7684535b91af1f870354c17 |
| command | sdlc close --issue 779 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T17:27:48-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the documented wrapping, priority and shared-geometry contracts. BR-1 is addressed by the corrected tables and revision entry. No new blocking findings were identified. Operator staging and recorded landing checks remain explicit outstanding workflow steps.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan lines 33 and 119–120 now name statusPacking, renderStatusBar and packStatusChips, which exist in status_layout.go at lines 40, 51 and 242. The revision at plan line 149 records the correction. The before/after diff changes documentation only; no behavioral regression test is required.
```

1. **Strengths**

   - Pure packing tests cover group wrapping, oversized groups, active/attention priority, Unicode widths and overflow counts.
   - `ChildGeometry` centralizes rectangular chrome validation; Console, compositor and broadcast consumers use declared chrome geometry.
   - Controlled partial-write tests verify that pending geometry suppresses clicks while preserving keyboard input and supports automatic recovery.
   - Atlas updates document the new layout and trusted metadata. The unrelated #757 evidence moves are verified identical renames.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Full couchtty and terminal packages passed.
   - Full broadcast package passed with `TMPDIR=/tmp`. The initial run failed an unchanged named-tunnel test under the longer temporary path.
   - Focused race tests passed, including pending confirmation and native PTY resizing.
   - Packing fuzzing passed 74,712 executions.
   - `git diff --check` passed; the checkout remains clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared row drawing and geometry validation.
   - **ARCH-PURE — pass:** Packing and geometry logic remain directly testable without IO.
   - **ARCH-PURPOSE — pass:** Wrapping propagates through panes, input, anchors, privacy and annotation guards.
   - **ARCH-MOCK — pass:** Stateful fixtures exercise production seams; native PTYs verify resize behavior.
   - **ARCH-CONSTRAINTS — pass:** Counted complexity tests enforce planner bounds.
   - **ARCH-SECURE — pass:** Child pixels cannot authorize broadcast; client geometry cannot override trusted metadata.
   - **ARCH-ORDER — pass:** Serialized publication and revision confirmation fence uncertain geometry.
   - **ARCH-FUNERAL — pass:** No new durable runtime artifacts or workers.
   - **ARCH-EVOLVE — pass:** Live metadata is additive; no persisted-format rewrite introduced.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f4119f89-bb14-415d-91d2-0dca8109b830","repository":"github.com/xianxu/pair","issue_id":"000779","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0518b965aea01a70","reviewed_head":"b1c68720d3e721e9d7684535b91af1f870354c17","slot":"pair:4","requested_at":"2026-10-22T00:25:47.640677Z"},"phase":"finished","pid":31406,"started_at":"2026-10-22T00:25:47.7407Z","finished_at":"2026-10-22T00:27:48.277632Z","session_id":"01a160ff-eef4-78d1-b510-43bfe81345db","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.825836875s
head: "b1c68720d3e721e9d7684535b91af1f870354c17"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
