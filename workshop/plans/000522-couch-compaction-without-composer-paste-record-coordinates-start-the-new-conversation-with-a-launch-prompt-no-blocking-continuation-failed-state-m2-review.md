# Boundary Review — pair#522 (milestone M2)

| field | value |
|-------|-------|
| issue | 522 — couch: compaction without composer paste: record coordinates, start the new conversation with a launch prompt, no blocking 'continuation failed' state |
| repo | pair |
| issue file | workshop/issues/000522-couch-compaction-without-composer-paste-record-coordinates-start-the-new-conversation-with-a-launch-prompt-no-blocking-continuation-failed-state.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 420860a7317b3539de376976274bc75efbe328bc..855c4b5045451c3662dda6a11f4edc893ab4911c |
| command | sdlc milestone-close --issue 522 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-13T20:34:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Read the plan and issue; now inspecting the M2 code diff.

Sweeping for stale references; next I'll verify the BR-1 regression test by reverting the fix.

Now reverting the fix to confirm the test goes red.

The regression test fails once the fix is reverted, so the fix is pinned. Now running the affected packages' tests.

These failures come from the checkout missing generated runtime-bundle assets. Checking how they're generated so I can rerun.

All green. Checking the claimed new lease test and the readiness tombstone test.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 removes the composer-paste machinery cleanly. The wrapper's orientation delivery, `PAIR_ORIENTATION_REQUEST`, the `orientation-status` operation, `OSOrientationStatusReader.Read`, `ReadyRecord.Orientation` and the delivery model are all gone, and nothing outside tests still uses them. `go build ./...` passes. The tests for the eight affected packages are green: wrapcmd, couchcore, couchcmd, launcher, artifactpath, readiness, orientation and checkpoint. To run them I had to generate the runtime bundle, because the checkout doesn't include it. The deletion had one hidden side effect, and the commit handles it: removing the orientation code also removed the only place the per-chunk render counter (`automaticRenderEpoch`) was advanced. It now advances in the same deferred spot as before, and a new lease test pins it. I reverted the M1 BR-1 fix in the disposable checkout and `TestAFailedPreviousSessionWriteLeavesTheLine` went red, so that fix is now pinned too. One thing keeps this from SHIP: the "sweep the references" part of M2 is incomplete. Stale mentions of the retired machinery remain in comments, test names and orphaned fixtures, and this family (`stale-retired-references`) has already had one finding on this issue. Everything left is cheap and none of it blocks.

**1. Strengths**
- **Render counter preserved.** `cmd/internal/wrapcmd/wrap.go:3250` advances `automaticRenderEpoch` in the same deferred position the old `observeOrientationTerminal` used, so it still lands after the chunk is processed. The lesson is recorded in `workshop/lessons.md`.
- **Old ready records still decode.** `cmd/internal/readiness/record.go:13-17` keeps a decode-only field, so a slot started by the old build can still register. `TestALegacyOrientationStateStillDecodes` covers both directions: an old record decodes, and nothing new writes the field.
- **BR-1 is pinned by a real failure, not a mock.** The `BeforeUpdate` store hook (`threadstore.go:114-117,433-437`) fails one named write inside a live compaction. I confirmed the test fails without the M1 fix.
- **The rename is true to what remains.** `OSOrientationStatusReader` became `OSReadyReader` and `orientationReplies` became `terminalReplies`, which now describe the surviving responsibilities (registration and generation; reply classification for peer delivery).
- **Docs updated.** The atlas edits to `couch.md`, `architecture.md`, `how-to-bring-up-a-new-harness-cli.md` (step 9 now points at `PromptArgs`) and `session-identity.md` are accurate. The README's compaction section was already updated in M1.

**2. Critical findings**

None.

**3. Important findings**
- **ARCH-PURPOSE / sweep — the 2nd finding in family `stale-retired-references`.** M2 promises "sweep the references", but these remain:
  - `nvim/draft_send.lua:12` cites the deleted `wrapcmd/orientation.go`.
  - `cmd/internal/couchcmd/wait_notice.go:55` mentions "a stuck orientation".
  - `cmd/internal/launcher/args.go:10` mentions "optional orientation delivery".
  - In `cmd/internal/couchcore/switchcontext_test.go`, the tests at `:84` (`TestOrientationStatusRequiresExactReadyIdentity`) and `:256` (`TestOrientationSessionObservationHonorsCallerDeadline`) are named after removed APIs; they now test `OSReadyReader`.
  - `cmd/internal/wrapcmd/testdata/orientation/agy/1.1.25/*` has no consumer left.
  - The codex fixtures under `testdata/orientation/` are used only by peer tests now.

  Don't fix only these sites. **The rule:** a retirement's sweep starts from a written list of every identifier, file path, env var, operation name and fixture directory being retired, and greps that list across every file type (Go, Lua, Markdown, test function names, testdata directories), not just compiled callers. Write the list into the plan's Revisions, run the grep, and fix every hit in one pass: rename the tests, delete or move the orphaned fixtures, and correct the three comments.

**4. Minor findings**
- **ARCH-FUNERAL:** `ReadyRecord.DeprecatedOrientation` doesn't say when it can be removed (for example, "after every slot has restarted on a #522+ build"). `Encode`/`Validate` also don't reject a non-empty value, so "never written" is a convention rather than something enforced.
- **ARCH-ORDER:** `wrapperSettled` dropped the explicit `orienting` gate. It now relies on the agent opening its turn from the launch prompt before the 3 s settle window passes. That's plausible in practice, but no test shows that a fresh session with a launch prompt doesn't report Settled before its turn opens.
- `switchcontext_test.go`: the obsolete-record cases now accept `err == nil && !registered`, which is weaker than the old "must error". It's still correct for `Registered`'s contract.

**5. Test coverage notes**
- BR-1 regression: I confirmed it goes red without the fix and green at head.
- The new lease test in `automatic_input_test.go` covers: lease refused before any repaint, refused while the composer still holds text, released on an empty repaint. That is the exact failure described in the lesson.
- Coverage that pinned only the retired code was deleted with it. Nothing kept behaviour has lost coverage.

**6. Architectural notes**
- **ARCH-DRY:** pass. `codexStartupStatus` and `terminalRowText` were moved into `composer_rows.go`, not copied.
- **ARCH-PURE:** pass. `wrapperSettled` stays pure.
- **ARCH-PURPOSE:** flagged; see the Important finding.
- **ARCH-MOCK:** pass. The new hook sits behind the real store's update path.
- **ARCH-CONSTRAINTS:** pass. The per-chunk atomic increment is unchanged in cost.
- **ARCH-SECURE:** pass. The decode-only field accepts any JSON but nothing reads it, and the reader's 16 KiB cap still applies.
- **ARCH-ORDER:** Minor note above.
- **ARCH-FUNERAL:** Minor note above.
- Still to do before the issue closes: the live Claude/Codex check (the plan's third item).

**7. Plan revision recommendations**
- Add a Revisions entry for M2 listing the retired identifiers that drive the sweep grep: `orientation.go`, `orientation_replies`, `PAIR_ORIENTATION_REQUEST`, `orientation-status`, `OrientationStatus`, `DeliveryState`, `testdata/orientation`.
- Record the decode-only field's end condition.

```findings
findings:
  - id: new
    severity: Important
    family: stale-retired-references
    title: |
      M2 sweep incomplete: comments, test names and orphaned fixtures still reference the retired orientation paste
    detail: |
      This is the 2nd finding in family stale-retired-references; fix the rule, not the instances. Instances: nvim/draft_send.lua:12 (cites deleted wrapcmd/orientation.go), couchcmd/wait_notice.go:55 ("a stuck orientation"), launcher/args.go:10 ("optional orientation delivery"), couchcore/switchcontext_test.go:84 and :256 test names after removed APIs, and orphaned wrapcmd/testdata/orientation/agy/1.1.25/* (no consumers) plus codex fixtures still under testdata/orientation used only by peer tests. Rule: a retirement sweep starts from a written list of every retired identifier, path, env var, op name and fixture dir, and greps it across all file types (Go, Lua, Markdown, test names, testdata). Record the list in the plan Revisions and clear every hit in one pass.
  - id: new
    severity: Minor
    family: tombstone-end-unnamed
    title: |
      ReadyRecord.DeprecatedOrientation names no removal condition and never-written is not enforced
    detail: |
      ARCH-FUNERAL: say when the decode-only field can be deleted (every slot restarted on a #522+ build); have Encode or Validate reject a non-empty value so "never written" holds structurally.
  - id: new
    severity: Minor
    family: plan-claimed-test-missing
    title: |
      No test pins that a launch-prompted fresh session does not report Settled before its turn opens
    detail: |
      ARCH-ORDER: wrapperSettled dropped the explicit orienting gate (peer_settle.go:20) and now depends on the agent opening its turn from the launch prompt within the settle window. The atlas states this, but no test pins it. This is the 2nd finding in family plan-claimed-test-missing; the class is a removed explicit gate whose replacement invariant is only stated in prose.
```

---

## Re-review — 2026-10-13T20:47:14-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 522 — couch: compaction without composer paste: record coordinates, start the new conversation with a launch prompt, no blocking 'continuation failed' state |
| repo | pair |
| issue file | workshop/issues/000522-couch-compaction-without-composer-paste-record-coordinates-start-the-new-conversation-with-a-launch-prompt-no-blocking-continuation-failed-state.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 420860a7317b3539de376976274bc75efbe328bc..5a9eddbf10942a2e07da7f94b333e81e98cfaf16 |
| command | sdlc milestone-close --issue 522 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-13T20:47:14-07:00 |
| verdict | SHIP |

## Review

I've checked the sweep and the tombstone; now I'm checking one live test that re-encodes a decoded record, then running the test suite.

Generating the runtime bundle assets so the remaining packages can build, then re-running their tests.

All touched packages pass. Now reading the core M2 code diff: settle gate, wrapper, and composer rows.

Atlas looks updated. Verifying the identifiers the new atlas text cites actually exist.

Checking whether the moved Codex captures fit the `testdata/tty` conformance layout they now live in.

```verdict
verdict: SHIP
confidence: high
```

M2 does what the plan says. The composer-paste machinery is gone: the wrapper's orientation delivery, `PAIR_ORIENTATION_REQUEST`, the `orientation-status` op and its reader, the settle gate's `orienting` input, the composer-probe profile fields and the orphaned agy captures. The pieces that survived were renamed by their real job (`terminalReplies`, `OSReadyReader`, `terminalRowText`/`codexStartupStatus`). I grepped every identifier on the written retirement list across the tree. The only hits left are deliberate history, retired-name assertions, or false positives (`PeerDeliveryState`). I checked that every name the new atlas text cites exists in code: `launcher.PromptArgs`, `TestPromptArgsPerAgent`, `freshNonceReachesPair`, `TestTheContinuationGuardsReachMatchesEveryRowAction` and `TestReadyReader…`. All touched packages pass: readiness, orientation, wrapcmd, couchcore, couchcmd, launcher, artifactpath, couchmessage and runtimebundle. To build, I first ran the runtimebundle generator into the gitignored `assets/` directory of this disposable checkout. All three prior findings are disposed below. The two new findings are both Minor: dead code the sweep orphaned, and one fixture directory filed in the wrong family.

**Strengths**
- The tombstone is now enforced by the code, not just stated. `readiness.Encode` refuses a non-empty `DeprecatedOrientation` (`cmd/internal/readiness/record.go:27`). `TestALegacyOrientationStateStillDecodes` decodes a legacy record and asserts that `Encode(got)` fails, so removing the guard turns the test red.
- `automaticRenderEpoch.Add(1)` moved into `handleChunk` (`wrap.go:3250`). It used to live inside the deleted orientation observer, and `TestAutomaticInput…` in `automatic_input_test.go` pins that the lease is released only after a fresh empty repaint. This is the regression `lessons.md:1643` describes.
- `wrapperSettled` is still pure, with an exhaustive table test (`peer_settle_test.go:16`). Deleting the `settleDue`, `orientationWake` and deadline arms from `translateStdinFrom` takes a lot of interleaving out of the stdin loop.
- The M1 fix BR-1 is pinned by `TestAFailedPreviousSessionWriteLeavesTheLine`, through a narrowly scoped `BeforeUpdate` hook (`threadstore.go:114`).

**Critical findings:** none.

**Important findings:** none.

**Minor findings**
- **Dead code left by the sweep (`stale-retired-references`, 3rd finding in this family).**
  - `terminalReplies.operatorData` (`cmd/internal/wrapcmd/terminal_replies.go:96`) has no callers. Its only caller was the deleted `orientationDelivery.admitOperatorInput`.
  - `orientation.BuildPrompt`, `validateBody`, `MaxBodyBytes`, `ReaderIntent` and `TargetAgent` (`cmd/internal/orientation/model.go:16-60`) are used only by `switchcontext_test.go:78`. They were already dead before this range, but they belong to the briefing machinery being retired.
  - **The rule:** a retirement list built from what you deleted can't catch what the deletion orphaned. After a retirement, also run a mechanical unused-symbol pass (`staticcheck -checks U1000` or `golang.org/x/tools/cmd/deadcode -test=false`) over the affected packages. Any symbol whose only remaining references are in tests gets deleted or justified.
- **Codex 0.154.0 captures are filed under the wrong fixture family.** They moved to `testdata/tty/codex/0.154.0`, but that family's atlas contract requires a `metadata.json` with SHA-256 pins, and `TestHarnessTTYFixtureConformance` globs `*/*/metadata.json`. So this metadata-less directory is silently skipped, yet it looks like Codex's newest conformance capture. Under the retention rule, that would be read as superseding `0.152.0`. The existing home for peer-test captures that carry only a README is `testdata/peer/<agent>/<version>/`. Move them there and extend `.gitattributes` to `testdata/peer/**/*.raw binary`.

**Test coverage notes**
- BR-10's guard is reachable, and removing it fails the test.
- Removing `withoutOrientation` from the child env has no test. That's harmless, because Couch and Pair ship as one build and nothing sets `PAIR_ORIENTATION_REQUEST` anymore.
- The live check is still the only evidence that agents actually open a turn from the launch prompt. That's acceptable, and the plan records it.

**Architecture**

| Marker | Result | Reason |
|---|---|---|
| ARCH-DRY | pass | `terminalRowText` is now one helper shared by `peer_composer` and the startup card. |
| ARCH-PURE | pass | — |
| ARCH-PURPOSE | Minor flag | The orphaned symbols above. |
| ARCH-MOCK | pass | `BeforeUpdate` is a test seam inside the real store, not a mock. |
| ARCH-CONSTRAINTS | pass | It removes a blocking gate and its timers. |
| ARCH-SECURE | pass | The legacy field is decoded as opaque `RawMessage` behind the existing 16 KiB read cap and strict decode, and never acted on. |
| ARCH-ORDER | pass | It removes a state dimension (`orienting`) and three select arms. No new state is carried between events. |
| ARCH-FUNERAL | pass | The tombstone names its end. Nothing new is durable. |

**Plan revisions:** add a line to the round-3 Revisions entry saying the retirement list is completed by a dead-symbol pass, and recording where the Codex 0.154.0 captures finally live.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      Written list recorded in plan Revisions; grep of every listed name is clean outside deliberate history; draft_send.lua:12, wait_notice.go:55, args.go:10 and test names fixed; agy fixtures and .gitattributes line removed.
  - id: BR-10
    disposition: addressed
    note: |
      record.go:13-18 names the removal condition; Encode refuses non-empty value (record.go:27) and orientation_test.go asserts Encode(decoded legacy) errors, which fails if the guard is removed.
  - id: BR-11
    disposition: withdrawn
    note: |
      Rebuttal accepted: Settled reduces to turn/picker/composer inputs, which TestWrapperSettledRule pins exhaustively; agent start-up timing is external and covered by the live check, and a first-turn gate would wedge a slot whose prompt is swallowed.
findings:
  - id: new
    severity: Minor
    family: stale-retired-references
    title: |
      Symbols orphaned by the retirement survive: terminalReplies.operatorData, orientation.BuildPrompt/validateBody/MaxBodyBytes/ReaderIntent/TargetAgent
    detail: |
      3rd finding in this family. terminal_replies.go:96 has zero callers; orientation/model.go BuildPrompt is test-only. Rule: a retirement list enumerates what was deleted, not what the deletion orphaned, so close every retirement with a mechanical unused-symbol pass (staticcheck U1000 or deadcode -test=false) over the touched packages and delete or justify every symbol left with test-only references.
  - id: new
    severity: Minor
    family: fixture-home-convention
    title: |
      Codex 0.154.0 peer captures moved into testdata/tty, whose contract requires metadata.json, so the conformance replay skips them
    detail: |
      testdata/tty/codex/0.154.0 has no metadata.json, so TestHarnessTTYFixtureConformance skips it, yet it looks like Codex's newest conformance version. Peer-only README captures already live in testdata/peer/<agent>/<version>; move them there and extend the .gitattributes binary rule to that tree.
```
