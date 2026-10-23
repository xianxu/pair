# Boundary Review — pair#802 (whole-issue close)

| field | value |
|-------|-------|
| issue | 802 — Couch: shorten common statuses |
| repo | pair |
| issue file | workshop/issues/000802-couch-common-status-wording.md |
| boundary | whole-issue close |
| milestone | — |
| window | a1b7725b28a1325d7ca5ea04805f844a8f933d73..2f4c125d723d258b4118c82babb7002e6d956024 |
| command | sdlc close --issue 802 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-23T11:26:32-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

The focused tests pass. Next I'm checking whether the inventory JSON format matches what the tool writes.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary:** All seven wording changes match the Spec table and its Revisions, and nothing outside the named set was touched. The focused tests pin each new string exactly, and I ran them green on the pinned head. The change does not touch the 40-cell cap or the full-text copy path. `TestGuessRotationAlertAndFlash` still checks both the full copied text and the ellipsized row. One thing should be fixed before shipping. The agent-facing Couch skill (`cmd/internal/couchcmd/skills/couch/SKILL.md`, embedded into the binary via `skill.go:8`) still describes the old wait-notice format: `waiting M min` and `idle off its resting branch`. The atlas was updated but this second description of the same format was not.

1. **Strengths**
   - The change stays inside the scope: seven strings in three files, and the rejected "Viewers need its code" suffix was left alone.
   - `lifecycle_notices_test.go:683-686` now compares the drain diagnostic exactly, so the count is still checked, not just a substring.
   - A new test, `TestLifecycleProjectionCapacityDiagnostic`, covers the capacity diagnostic, which had no test before.
   - Cast notices are now checked against the whole notice text (`console_castcodes_test.go:70`), not with `Contains`. The ellipsized `shown` value was updated to match the 40-cell cap.
   - The hash in the affected-reader record matches `wait_notice_test.go` (`0508b7dc…`).

2. **Critical:** none.

3. **Important**
   - `cmd/internal/couchcmd/skills/couch/SKILL.md:299-302` still says `slot repo:N waiting M min <reason>` and `idle off its resting branch`. This text is embedded and shipped to the TL agent as its guide to wait notices. It now disagrees with `wait_notice.go:36,201` and with the updated `atlas/couch.md:1491`. **Fix:** change it to `waiting Mm` and `idle off resting branch`. (ARCH-PURPOSE shadow-sweep: the atlas was updated, the skill was missed.)

4. **Minor**
   - `tests/affected-readers/.../wait_notice_test.go.json` is written on one line, while the other records use indented JSON. The content is valid, but regenerating it with the inventory tool would avoid formatting churn.
   - `wait_notice_test.go:546` has a comment saying "idle off its resting branch". It describes the situation rather than quoting the string, so it is harmless.
   - With N=1 the drain message reads "1 slots'". The TL approved this and it is logged.

5. **Test coverage:** Every changed producer has an exact-match test. The capacity test fills `p.roots` directly through the mutex, which is acceptable inside the package. No catalog-wide assertion was added, as the Spec required.

6. **Architecture**
   - **ARCH-DRY:** pass. The format strings repeat in tests on purpose, as exact pins.
   - **ARCH-PURE:** pass. `waitNoticeBody` stays a pure function.
   - **ARCH-PURPOSE:** flag (the SKILL.md item above).
   - **ARCH-MOCK, ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-ORDER:** N/A. This is a string-only change: no new IO, state, input parsing or secrets.
   - **ARCH-FUNERAL:** N/A. Nothing durable is created; the inventory record replaces the old one.
   - **ARCH-EVOLVE:** N/A for formats. The wait-notice body is free text sent to the TL, and the only consumer that describes its shape is the stale SKILL.md above.

7. **Plan revisions:** none needed. The log already records the fourth reader that was found (`picker_notice_test.go`).

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: doc-restatement-not-swept
    title: |
      Embedded couch SKILL.md still describes the old wait-notice wording (waiting M min, idle off its resting branch)
    detail: |
      cmd/internal/couchcmd/skills/couch/SKILL.md:299-302 is embedded via skill.go:8 and tells the TL agent the notice format. It now contradicts wait_notice.go:36,201 and the updated atlas/couch.md:1491. Change it to `waiting Mm <reason>` and `idle off resting branch`.
  - id: new
    severity: Minor
    kind: defect
    family: generated-artifact-hand-written
    title: |
      Affected-reader record for wait_notice_test.go written as single-line JSON unlike sibling records
    detail: |
      The content and sha256 are correct. Regenerate it with the inventory tool so its formatting matches the indented sibling records.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6d21e7ab-8cd7-4365-a01d-0d85b703e32b","repository":"github.com/xianxu/pair","issue_id":"000802","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/042a920757906057","reviewed_head":"2f4c125d723d258b4118c82babb7002e6d956024","slot":"pair:8","requested_at":"2026-10-23T18:25:14.700222Z"},"phase":"finished","pid":15742,"started_at":"2026-10-23T18:25:14.880369Z","finished_at":"2026-10-23T18:26:30.890222Z","session_id":"1c660dfd-107f-4f8b-b163-a035c85e7570","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.381276417s
head: "2f4c125d723d258b4118c82babb7002e6d956024"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T11:29:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 802 — Couch: shorten common statuses |
| repo | pair |
| issue file | workshop/issues/000802-couch-common-status-wording.md |
| boundary | whole-issue close |
| milestone | — |
| window | a1b7725b28a1325d7ca5ea04805f844a8f933d73..3f4c34fc99b4a74101c9ad363eed0c78d3e15a43 |
| command | sdlc close --issue 802 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-23T11:29:06-07:00 |
| verdict | SHIP |

## Review

I'm checking how the reader-record tool formats its output, so I can settle prior finding BR-2.

```verdict
verdict: SHIP
confidence: high
```

All seven strings in the issue's Selected set appear verbatim in the three production files, and nothing outside that set was changed. Each new string keeps its dynamic parts (slot, minutes, summary, mute path, unverified count, cast kind and link) and its actions (peek, mute, hover, rotate). Both prior findings are resolved. BR-1 is addressed: `cmd/internal/couchcmd/skills/couch/SKILL.md:299-302` now reads `waiting Mm <reason>` and `idle off resting branch`. A repo-wide search outside `workshop/` and `.git` finds none of the seven old wordings left in any live copy. The one remaining hit is a test comment that describes the condition rather than quoting the notice (`wait_notice_test.go:546`). BR-2 is withdrawn: the record tool (`tests/affected-go-readers-test.py:128,193`, `certificate_json`) writes one-line JSON itself, and the existing records already mix the two layouts (115 one-line, 285 multi-line). `go vet` on both packages, the focused couchcmd tests (WaitNotice, Picker, Lifecycle) and the focused couchtty tests (Codes, GuessRotation, RapidGuess) all passed on the pinned head. `make test-affected-inventory --check` exited clean. Nothing blocks SHIP.

1. **Strengths**
   - Tests compare whole messages rather than fragments. `lifecycle_notices_test.go:684` checks the full drain string, and `console_castcodes_test.go:70` checks the full notice through `f.notice()` instead of a `Contains` on the screen. That catches drift in the wording around the dynamic parts, not just in the parts themselves.
   - There is a new test for the capacity diagnostic, which had none before (`lifecycle_notices_test.go:695-722`). It fills `p.roots` to the 128 cap and drives the real `observe` path.
   - The display cap from #789 is still exercised. `TestGuessRotationAlertAndFlash` checks the cut-off `shown` form (`LIVE view link: too many wrong codes; c…`) next to the full text used for copying.
   - Scope held. The repeated "Viewers need its code" suffix that the issue rejected is untouched.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `Lifecycle: 1 slots' metadata unverified` reads awkwardly when the count is 1. The TL approved this wording (Revisions, 08:20), so it is not raised as a finding.
   - The atlas line at `atlas/couch.md:1492` adds history ("#789 caps…, #802 shortened…"). That is fine, but it is the kind of note that goes stale.

5. **Test coverage notes:** every changed producer has a test pinning its exact text. `picker_notice_test.go` was a fourth reader the plan missed; the implementer found and updated it, and its test passes. Cap and full-copy behaviour is still covered by the existing tests from #789.

6. **Architectural notes**
   - **ARCH-DRY:** pass. The wait-notice text is built only in `waitNoticeBody`, and the reasons come from the `waitReason` consts. The cast messages are still written out in both the code and the tests, which is acceptable for pinning exact wording.
   - **ARCH-PURE:** pass. `waitNoticeBody` is a pure function.
   - **ARCH-PURPOSE:** pass. The prior-round class (written restatements of the notice format) was searched across the repo, and SKILL.md plus the atlas were the only live copies.
   - **ARCH-MOCK:** pass. The diff adds no external calls.
   - **ARCH-CONSTRAINTS:** pass. Strings got shorter, and the 40-cell cap stays in place.
   - **ARCH-SECURE:** pass. No new inputs or secrets are involved.
   - **ARCH-ORDER:** pass. No state changes.
   - **ARCH-FUNERAL:** pass. Nothing durable is created; one existing reader record was rewritten in place.
   - **ARCH-EVOLVE:** pass. The record's schema is unchanged; only its hash and layout changed. The notice body is a human-readable message rather than a stored format. I found no code that parses `waiting N min`.

7. **Plan revision recommendations:** none. The plan and Log match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      SKILL.md:299-302 now says `waiting Mm <reason>` / `idle off resting branch`, matching wait_notice.go:36,201; repo-wide sweep finds no other live copy.
  - id: BR-2
    disposition: withdrawn
    note: |
      The record tool (tests/affected-go-readers-test.py:128,193 certificate_json) writes one-line JSON; the existing records mix layouts (115 one-line, 285 multi-line), and --check passes.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c8674099-5aec-430f-b826-5ed728fd1a8e","repository":"github.com/xianxu/pair","issue_id":"000802","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/fb0d0d24acf71b0a","reviewed_head":"3f4c34fc99b4a74101c9ad363eed0c78d3e15a43","slot":"pair:8","requested_at":"2026-10-23T18:27:51.698483Z"},"phase":"finished","pid":31807,"started_at":"2026-10-23T18:27:51.83623Z","finished_at":"2026-10-23T18:29:05.701368Z","session_id":"1861299b-d9ad-435b-bd69-2b0ec7eda1c7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.174022875s
head: "3f4c34fc99b4a74101c9ad363eed0c78d3e15a43"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
