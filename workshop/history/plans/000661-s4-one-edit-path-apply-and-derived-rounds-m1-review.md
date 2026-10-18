# Boundary Review — pair#661 (milestone M1)

| field | value |
|-------|-------|
| issue | 661 — S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply |
| repo | pair |
| issue file | workshop/issues/000661-s4-one-edit-path-apply-and-derived-rounds.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3851b7338ff107d9b676d1db4eda7c7f21a2aa62..3b12a97c4e21796062c23b4bdeb6b5e53cfa4d4f |
| command | sdlc milestone-close --issue 661 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-18T03:57:52-07:00 |
| verdict | REWORK |

## Review

Reviewing the sessionsync write path now; checking the legacy-file cleanup in Register next.

Now probing a whole-file write under `presentation: propose` that touches frontmatter.

```verdict
verdict: REWORK
confidence: high
```

M1 mostly delivers what the plan describes. The Go and Lua resolvers, the capped Myers line diff, the round derivation, the sidecar in both layouts, the order that prevents a stale round, `round --json` with `cleared`, and the CLI are all present. The Go suites (sessionsync, couchcmd, artifactpath) and the Lua fixture tests pass on the pinned head. Two correctness bugs in the propose converter block SHIP. I confirmed both with throwaway probe tests in this checkout and then removed them.
1. Under `presentation: propose`, an `apply` or `write --all` that touches the frontmatter writes 🤖 markers into it. The plan promises that frontmatter is never proposed.
2. Inserting or deleting a blank line under propose writes a bare `🤖` line. The pane grammar doesn't parse it as a marker, so it can't be accepted or rejected.

Both are cheap to fix. Neither is caught by the fixtures.

## 1. Strengths
- **One set of rules, two languages.** `testdata/edit-records` is the single source for both resolvers: `edits_test.go` goes through `ParseEdits`, and `resolve_fixtures_test.lua` also replays the propose fixtures through `markers.lua`. `apply.lua` now uses `reconstruct.resolve` instead of its own loop (ARCH-DRY).
- **No stale round.** `sync.go:661-667` removes the old sidecar before the document is replaced, and a failed final write becomes a warning. `TestRoundFailureNeverLeavesAStaleRound` covers both the failed write and the failed removal (ARCH-ORDER).
- **Faithful extraction.** `submitNames` (`owed.go:92`) was lifted out of `OwedForDocument` without changing its behaviour, and `SubmittedSince` reuses it for `cleared`, folded submits included.
- **Strong diff test.** The property test in `linediff_test.go:43` uses random 3-letter alphabets, which catches backtrack errors. The D-cap fallback is tested by forcing the cap down.
- **Bounded, validated sidecar.** It is written and read through `roundMaxBytes`, and `decodeRound` checks its shape. A malformed sidecar gives `round-invalid`, and Register's import skips it (ARCH-SECURE).

## 2. Critical findings
- **C1: propose markers land inside the frontmatter** (`sync.go:551-566`).
  - The guard is `!req.Frontmatter`, which only covers `write --frontmatter`. An `apply` or `write --all` that edits a frontmatter line goes through `ProposeMarkers` over the whole file.
  - Probe: `apply` of `presentation: propose` → `presentation: direct` on `proposeDoc` succeeded and wrote `presentation: 🤖~propose~{direct}`.
  - `Presentation()` then reads that as unknown, so the document silently switches to `highlight`. The next frontmatter edit (`title: x`) was written raw.
  - Fix: under propose, split off the frontmatter block (`FrontmatterBody`) and run `ProposeMarkers` on the body only, applying frontmatter changes directly. Or refuse frontmatter changes in non-frontmatter writes under propose. Add a rig test.
- **C2: a blank line inserted or deleted under propose becomes a bare `🤖`** (`propose.go:72-82`).
  - `markLine("", "\n")` and `markLine("\n", "")` both give `marker("", "")`, which is just `"🤖"`.
  - Probe: `ProposeMarkers("a\nb\n", "a\n\nb\n")` and `ProposeMarkers("a\n\nb\n", "a\nb\n")` both return `"a\n🤖\nb\n"`. `markers.parse_markers({'🤖'})` finds 0 markers.
  - The result can't be settled, insert and delete look the same, and adding any markdown paragraph (which brings a blank line) triggers it.
  - Fix: refuse an empty unpaired body as `propose-unencodable`, or fold the blank line into a neighbouring marker. Add propose fixtures for both directions.

## 3. Important findings
- **I1: Core concepts and the plan's numbers drift from the code, with no `## Revisions` entry.**
  - The table lists `submitsOfDocument → []Entry`. The code has `submitNames` and `SubmittedSince` (`owed.go:92,119`).
  - The plan says the Myers cap is D ≤ 4096. The code uses 1024 (`linediff.go:14`).
  - Both changes are only explained in the issue Log. Add a Revisions entry so the plan matches the code.

## 4. Minor findings
- **Occurrence below 1 isn't rejected at parse.** `ParseEdits` accepts `occurrence ≤ 0`, which later resolves as `not found`. The plan says it must be ≥ 1, as in `handoff.lua valid_payload`.
- **A no-op write leaves the round's `result_hash` behind.** A no-op write still restamps the file but keeps the previous round, so its `result_hash` no longer matches the file. If parley uses `result_hash` to detect staleness, a no-op write hides the highlight. Confirm with parley, or refresh `result_hash` on stamp-only writes.
- **Diff time isn't measured.** Myers worst-case time on a 4 MiB document at D = 1024 runs inside the flock, and there is no benchmark (ARCH-CONSTRAINTS).
- **Legacy sidecar left behind.** Register imports `.BASE.round` but leaves the legacy copy in place. This matches the existing shadow behaviour (ARCH-FUNERAL, low cost).

## 5. Test coverage notes
- **Covered:** stale and operator-lock refusals before resolving; an all-or-nothing batch with a byte-identical snapshot afterwards; the registry lifecycle (register, rename, forget); `cleared` timing, including another file's submit and a folded submit; the CLI envelope keys and exit codes 1 and 6.
- **Missing:** propose fixtures for blank-line insert and delete (C2), and propose combined with a frontmatter-touching `apply` or `write --all` (C1).
- **Not tested end to end:** "after `register` it is back" following a `forget`. Import is covered only from a fresh legacy file.

## 6. Architecture notes
| Marker | Result |
|---|---|
| ARCH-DRY | Pass |
| ARCH-PURE | Pass: resolver, diff, round and propose are pure, and the IO stays in `write` and `Round` |
| ARCH-PURPOSE | Flag (C1): "frontmatter is never proposed" is only enforced for one write form |
| ARCH-MOCK | Pass: no new external dependency; the rig uses the real filesystem and ledger |
| ARCH-CONSTRAINTS | Pass, with the minor note on diff time |
| ARCH-SECURE | Pass |
| ARCH-ORDER | Pass |
| ARCH-FUNERAL | Pass, with the minor note on the legacy sidecar |

For M2: `apply`'s register-on-first-apply side effect should keep `write`'s refuse-unregistered path, as the plan states. The pane should read `round` positions as byte columns.

## 7. Plan revision recommendations
- **Rename the helper:** "`submitsOfDocument` became `submitNames` (path matching shared with `OwedForDocument`) plus `SubmittedSince` (cleared)." Update the Core concepts row.
- **Record the cap:** "Myers cap D ≤ 1024, not 4096: the trace memory grows as D²."
- **State the frontmatter rule after the C1 fix:** "Under propose, frontmatter changes in any write form are applied directly (or refused); only the body is proposed."

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: propose-frontmatter-exclusion
    title: |
      Propose writes markers inside frontmatter on apply / write --all
    detail: |
      sync.go:551-566 guards only req.Frontmatter; an apply changing "presentation: propose" wrote "presentation: 🤖~propose~{direct}" (probe on proposeDoc), silently flipping Presentation() to highlight. Propose only the body (split via FrontmatterBody) or refuse frontmatter changes; add a rig test.
  - id: new
    severity: Critical
    kind: defect
    family: propose-marker-encodability
    title: |
      Blank-line insert/delete under propose writes a bare unparseable robot marker
    detail: |
      propose.go markLine with an empty unpaired body returns marker("","") = bare robot emoji; ProposeMarkers("a\nb\n","a\n\nb\n") and the reverse both give "a\n🤖\nb\n", and markers.parse_markers finds 0 markers, so it can never be settled. Refuse as propose-unencodable or encode differently; add propose fixtures.
  - id: new
    severity: Important
    kind: defect
    family: plan-core-concepts-drift
    title: |
      Plan Core concepts and Myers cap disagree with code without a Revisions entry
    detail: |
      Plan names submitsOfDocument returning []Entry and cap D<=4096; code has submitNames + SubmittedSince (owed.go:92,119) and maxDiffEdits=1024 (linediff.go:14). Only the issue Log records it; add a plan Revisions entry.
  - id: new
    severity: Minor
    kind: defect
    family: edit-record-parse-validation
    title: |
      ParseEdits accepts occurrence <= 0 (plan requires >= 1 at parse)
    detail: |
      It resolves as "not found" instead of a parse refusal matching handoff.lua valid_payload.
  - id: new
    severity: Minor
    kind: defect
    family: round-result-hash-staleness
    title: |
      A no-op (stamp-only) write keeps the round with a now-mismatched result_hash
    detail: |
      If parley treats result_hash != file hash as stale, any no-op write hides the highlight; confirm the contract or refresh result_hash.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b02be400-0d9f-418c-8cfa-7eaa8cf2ce8f","repository":"github.com/xianxu/pair","issue_id":"000661","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/e1a6bcd139762e5d","reviewed_head":"3b12a97c4e21796062c23b4bdeb6b5e53cfa4d4f","slot":"pair:2","requested_at":"2026-10-18T10:53:26.689806Z"},"phase":"finished","pid":88422,"started_at":"2026-10-18T10:53:26.98853Z","finished_at":"2026-10-18T10:57:52.008848Z","session_id":"101d5fcf-dfce-4492-94c7-be5f51f02fd8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.787337334s
head: "3b12a97c4e21796062c23b4bdeb6b5e53cfa4d4f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T04:05:17-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 661 — S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply |
| repo | pair |
| issue file | workshop/issues/000661-s4-one-edit-path-apply-and-derived-rounds.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3851b7338ff107d9b676d1db4eda7c7f21a2aa62..3aa68861be845a317cc9443204ac5cc85aca9396 |
| command | sdlc milestone-close --issue 661 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-18T04:05:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the propose fixtures and running the tests next.

Now reverting each fix in turn to confirm its regression test goes red.

Reviewing the Lua resolver changes, then writing up.

Running a quick probe on the frontmatter-fallback path, then writing the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The two Critical findings from the last round are fixed, and each fix has a test that fails without it. I reverted each fix in this disposable checkout to confirm that. Restoring `ProposeMarkers` in `sync.go:559` makes `TestProposeNeverMarksFrontmatter` fail with `presentation: 🤖~propose~{direct}`. Dropping the blank-line cases in `markLine` makes `TestProposeMarkersFixtures/blank-line-*` and `paragraph-insert` fail with a bare `🤖`. The Go suites (`sessionsync`, `couchcmd`, `artifactpath`) pass, and so does `nvim -l nvim/review/resolve_fixtures_test.lua` (16 + 9 fixtures). Two things remain before SHIP. First, the new `apply` and `round` verbs, and the propose behaviour, are missing from `README.md` and the agent-facing `skills/couch/SKILL.md`. Second, the BR-5 fix has no regression test: I removed the `n < 1` guard and every suite stayed green. A probe also found that `ProposeDocument` trusts any frontmatter split it gets; that is Minor and has a rule-level fix.

**1. Strengths**
- **One resolver, two languages:** `ResolveEdits` (`edits.go:104`) and Lua `reconstruct.resolve` (`reconstruct.lua:117`) both run the same `testdata/edit-records` fixtures, and `apply.lua` now delegates to the shared resolver instead of keeping its own copy (ARCH-DRY).
- **Safe round ordering:** the old round is removed before the document moves (`sync.go:660-665`), and a failed round write becomes a warning rather than a stale round. `RoundWriter` is a real seam for testing that failure.
- **Bounded input and output:** the round sidecar has a 16 MiB cap on read and on write (`round.go:173`). `decodeRound` checks the shape. `ParseEdits` parses into typed records before anything uses them (ARCH-SECURE, ARCH-CONSTRAINTS).
- **One submit-matching rule:** `submitNames` (`owed.go:95`) was pulled out of `OwedForDocument`, so the owed check and `Cleared` use the same logic.
- **Stronger Lua check:** the fixture test now requires every `🤖` in the written text to parse (`resolve_fixtures_test.lua:58`).

**2. Critical:** none.

**3. Important**
- **Docs are missing the new verbs.** `README.md:454-468` lists `read|write|diff|status`, frontmatter, register and lookup, but not `apply EDITS.json FILE --if-hash H` or `round FILE --json`. It also says nothing about `presentation: propose` or the round sidecar. `cmd/internal/couchcmd/skills/couch/SKILL.md` (what `couch --skill` prints, i.e. the agent's contract) does not mention `apply`, `edits-invalid`, `propose-too-large` or `propose-unencodable`. Only `run.go:1443` and the atlas were updated. Fix: add both verbs to the README block, and add a short `apply`/round/propose section to SKILL.md.

**4. Minor**
- **BR-5 is fixed but untested.** Add `[{"old":"a","occurrence":0,"new":"b"}]` and `occurrence: -1` to the `edits_test.go:72` refusal list.
- **`ProposeDocument` (`propose.go:65`) trusts whatever split `FrontmatterBody` returns.**
  - Probe: base has a body horizontal rule `---`; the after text deletes the frontmatter closer and changes `body` to `body2`.
  - Result: `---\n…presentation: propose\nbody2\n\n---\n🤖~body~\n\n🤖~---~…`, with no error and `IsSessionSync` true. Body text is written unproposed inside the frontmatter.
  - The `errA != nil` fallback also marks frontmatter lines whenever the after text has no closer; for now only the later owner check catches that.
- **Blank-line deletes under propose are silently skipped.** Deleting a blank line under propose (`blank-line-delete.json`) returns success and writes nothing. That fits the documented empty-line cost, but the CLI still prints `applied N record(s)`. Consider a warning.
- **BR-1** (prose test lists) is still open, as noted in the dispositions below.

**5. Test coverage notes**
- The propose fixtures pin hand-written `accepted`/`rejected` strings. Nothing checks them against `after`/`base`.
- Add a property check: accepting the markers gives `after`, and rejecting gives `base`, both modulo blank lines. That states the encoding's invariant once and would catch new unencodable cases without new hand-written fixtures.
- The Lua fixture test reads the fixture's `markers`, not Go output. Agreement between Go and Lua therefore depends on `TestProposeMarkersFixtures` also asserting `got == fx.Markers`, which it does.

**6. Architecture**
- **ARCH-DRY: pass.** The resolver is shared through fixtures, and `submitNames` is extracted.
- **ARCH-PURE: pass.** `ResolveEdits`, `LineDiff`, `DeriveRound` and `ProposeMarkers`/`ProposeDocument` are pure and tested without IO. The IO stays in `Sync.write` and `Round`.
- **ARCH-PURPOSE: pass on code.** The docs gap above is the one shortfall.
- **ARCH-MOCK: pass.** No new external binary. `RoundWriter` injects the sidecar-write failure.
- **ARCH-CONSTRAINTS: pass.** Limits are explicit: `maxDiffEdits=1024`, `proposeMaxLines=3`, `roundMaxBytes`.
- **ARCH-SECURE: pass.** The sidecar and `EDITS.json` are parsed into typed values, and a malformed round gives a visible `round-invalid`. Not covered: span positions are not bounded against the file's actual length; that is parley's job.
- **ARCH-ORDER: pass.** The sequence is: lock, hash, owed, resolve, content, then remove the old round, move the document, write the new round. A crash after the remove leaves no round, never a stale one. It is documented.
- **ARCH-FUNERAL: pass.** There is one sidecar per document. The next changing write replaces it, and it is removed with the registered state directory (`registry.go:268`). A legacy `.BASE.round` stays behind after `Register` copies it, the same as legacy shadow files today.

**7. Plan revisions**
- Add a line to the existing 2026-10-18 entry: under propose the frontmatter split must be unambiguous, otherwise the write refuses (the rule in the new finding).
- Note in Core concepts that `ProposeDocument` wraps `ProposeMarkers`. The Revisions entry covers it, but the table row still names only `ProposeMarkers`.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan keeps prose test lists by choice (plan-quality round 1 entry); Minor, non-blocking, carried for the close review.
  - id: BR-2
    disposition: addressed
    note: |
      sync.go:559 now calls ProposeDocument; TestProposeNeverMarksFrontmatter goes red when reverted to ProposeMarkers (verified).
  - id: BR-3
    disposition: addressed
    note: |
      markLine leaves unpaired blank lines unmarked; blank-line-insert/delete and paragraph-insert fixtures go red on revert (verified); Lua test requires every robot marker to parse.
  - id: BR-4
    disposition: addressed
    note: |
      Plan Revisions entry (plan lines 313-320) records the Myers cap 1024, submitNames + SubmittedSince, and ProposeDocument.
  - id: BR-5
    disposition: not-addressed
    note: |
      Guard added at edits.go:75 but no regression test; removing it leaves sessionsync and couchcmd suites green (verified). Add occurrence 0 and -1 to the edits_test.go:72 list.
  - id: BR-6
    disposition: withdrawn
    note: |
      Issue line 105 records the contract: parley re-anchors by span text when result_hash differs, and the fixed-width stamp keeps positions exact.
findings:
  - id: new
    severity: Important
    kind: defect
    family: docs-gate-new-surface
    title: |
      README and couch SKILL.md omit the new apply/round verbs and propose behaviour
    detail: |
      README.md:454-468 lists session-sync verbs but not `apply EDITS.json FILE --if-hash H` or `round FILE --json`, and nothing on presentation: propose or the round sidecar. cmd/internal/couchcmd/skills/couch/SKILL.md (couch --skill, the agent contract) has no apply, edits-invalid, propose-too-large or propose-unencodable. Only run.go usage and atlas/couch.md were updated.
  - id: new
    severity: Minor
    kind: defect
    family: propose-frontmatter-exclusion
    title: |
      ProposeDocument trusts any frontmatter split; a moved closer writes body text unproposed
    detail: |
      This is the 2nd finding in family propose-frontmatter-exclusion. Rule covering both: under propose, write the after-frontmatter as is only when both sides split unambiguously and the after block holds only frontmatter lines; otherwise refuse (propose-unencodable), never fall back to whole-document marking. Probe: base with a body HR `---`, after deleting the closer and changing body to body2, gives frontmatter containing `body2` plus markers, err nil, IsSessionSync true. Prevalence: two instances, the errA/errB fallback at propose.go:68 and the boundary-shift case.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"682c11aa-2cb0-48af-be8a-09700a05bb88","repository":"github.com/xianxu/pair","issue_id":"000661","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c418d06957eddb02","reviewed_head":"3aa68861be845a317cc9443204ac5cc85aca9396","slot":"pair:2","requested_at":"2026-10-18T10:59:57.883862Z"},"phase":"finished","pid":27865,"started_at":"2026-10-18T10:59:57.980486Z","finished_at":"2026-10-18T11:05:16.401535Z","session_id":"8a418d9a-a8f8-46e0-8269-ed7f875680cf","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.550348917s
head: "3aa68861be845a317cc9443204ac5cc85aca9396"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T04:11:49-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 661 — S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply |
| repo | pair |
| issue file | workshop/issues/000661-s4-one-edit-path-apply-and-derived-rounds.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3851b7338ff107d9b676d1db4eda7c7f21a2aa62..e40f8c76255534074034737670de11ba1fe8d208 |
| command | sdlc milestone-close --issue 661 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-18T04:11:49-07:00 |
| verdict | SHIP |

## Review

Verifying the prior-round fixes (BR-5/7/8 look addressed in the last commit); running the test suites now.

I'm writing up the verdict now. The fixes I reverted to check went red as expected, and the Go and Lua suites pass at the pinned head.

```verdict
verdict: SHIP
confidence: high
```

Ready to ship. All three open defect findings (BR-5, BR-7, BR-8) are fixed, with evidence. BR-8 had the most risk. Under `presentation: propose`, `ProposeDocument` now refuses any body write that changes the frontmatter, moves its closer, or can't be split on both sides (`cmd/internal/sessionsync/propose.go:68-71`). That one rule covers both cases in the `propose-frontmatter-exclusion` family. There is no longer a fallback that marks the whole document. I put the propose.go from `3aa68861b` back and ran the tests: `TestProposeNeverMarksFrontmatter` failed with `want refusal exit 1 propose-unencodable, got <nil>`. I then restored the file. At the pinned head these suites pass: `go test` for `sessionsync`, `couchcmd` and `artifactpath`, plus `nvim -l resolve_fixtures_test.lua` (16 shared fixtures + 9 propose fixtures) and `reconstruct_test.lua`. Nothing blocks.

1. **Strengths**
   - **One fix for the whole frontmatter family:** `propose.go:68` checks `errB != nil || errA != nil || fa != fb`. That covers both the flip of `presentation:` and the moved closer. `write --frontmatter` stays direct because `prepareWrite` skips propose when `req.Frontmatter` is set (`sync.go`).
   - **No stale round:** the old round sidecar is removed before the document moves, and a failure to save the new round only produces a `Result.Warning`. A test that injects a failing `RoundWriter` covers this.
   - **Shared submit matching:** `submitNames` (`owed.go`) is the one matching rule, used by both `OwedForDocument` and `SubmittedSince`. This is a clean ARCH-DRY extraction.
   - **One rule set for both resolvers:** the fixtures in `testdata/edit-records` drive both the Go and the Lua resolvers.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** none new.

5. **Test coverage**
   - **BR-5:** `TestParseEditsRefusesMalformed` now includes `occurrence: 0` and `occurrence: -1`. Without the `n < 1` check, `ParseEdits` would accept these, so the test fails without the fix.
   - **BR-8:** the second assertion in `TestProposeNeverMarksFrontmatter` (moved closer, no other `---`) would have hit the old whole-document fallback and returned no error, so it also guards the fix.

6. **Architecture**
   - **ARCH-DRY:** pass (`submitNames` shared; one resolver contract enforced by fixtures).
   - **ARCH-PURE:** pass. `ResolveEdits`, `LineDiff`, `DeriveRound`, `ProposeMarkers` and `ProposeDocument` are pure, and the IO stays in `Sync.write`.
   - **ARCH-PURPOSE:** pass for M1. Moving review-pane writes onto `apply` is M2's job, as the plan says.
   - **ARCH-MOCK:** pass. Ledger and filesystem work runs through the existing rig, and the injectable `RoundWriter` seam reaches the failure path.
   - **ARCH-CONSTRAINTS:** pass. `roundMaxBytes` caps both writing and reading the sidecar, and a round over the cap degrades to a warning.
   - **ARCH-SECURE:** pass. `decodeRound` turns the sidecar into a typed value and refuses a malformed one with `round-invalid`. `readRegular` is size-capped.
   - **ARCH-ORDER:** pass. The previous round is removed before the document is replaced, so the sidecar can never describe text it doesn't match.
   - **ARCH-FUNERAL:** pass. The round sidecar is replaced by the next changing write, moves with the file on `register`, and is exported or removed on `forget` (`exportLegacy`). It is at most one file per document.
   - **For M2:** the review pane's `round --json` consumer should handle `Round: null` and `round-invalid` visibly.

7. **Plan revisions:** none needed. The existing Revisions entry already records the stricter rule ("must leave the frontmatter byte-identical or it refuses").

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Overtaken by implementation: Tasks 3/5/6 are done and their enumerated cases exist as tests, so compressing the finished task prose no longer guides any work.
  - id: BR-5
    disposition: addressed
    note: |
      edits.go ParseEdits refuses n < 1; edits_test.go TestParseEditsRefusesMalformed adds occurrence 0 and -1, which the code accepted before the fix.
  - id: BR-7
    disposition: addressed
    note: |
      README.md:465-471 documents apply and round, the propose refusals and the side-doc fallback; SKILL.md adds both table rows and the apply/edits-invalid/exit 6/rounds/propose-too-large/propose-unencodable guidance, matching sync.go and propose.go.
  - id: BR-8
    disposition: addressed
    note: |
      propose.go:68 refuses with propose-unencodable unless both sides split and fa == fb, with no whole-document fallback (the stated family rule); putting back the 3aa68861b propose.go makes TestProposeNeverMarksFrontmatter fail.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"507b0c8d-ea80-4850-8dbd-8ccc80563ebf","repository":"github.com/xianxu/pair","issue_id":"000661","recipe":"milestone-review","round":4,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0fc9bf0a82547e0a","reviewed_head":"e40f8c76255534074034737670de11ba1fe8d208","slot":"pair:2","requested_at":"2026-10-18T11:06:51.587047Z"},"phase":"finished","pid":77864,"started_at":"2026-10-18T11:06:51.701437Z","finished_at":"2026-10-18T11:11:15.315707Z","session_id":"c99e80d3-cbd3-47bc-85fe-9186f07c360f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.53833975s
head: "e40f8c76255534074034737670de11ba1fe8d208"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
