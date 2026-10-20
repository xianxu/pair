# Boundary Review — pair#726 (milestone M2)

| field | value |
|-------|-------|
| issue | 726 — Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis |
| repo | pair |
| issue file | workshop/issues/000726-hub-lifecycle-ops-time-out-client-side-resume-pair-6-relaunch-parley-nvim-4-context-deadline-exceeded-with-no-diagnosis.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 5028c01cb1ed223646dd009f6a8ea6ff1b1b7dd6..2e9772fb0bdc2e99cb1d9379591101ad10f1d847 |
| command | sdlc milestone-close --issue 726 --milestone M2 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-20T09:19:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M2 skip: checking that Proven/Fresh callers never read `Ranked`/`Incomplete` from the skipped observation.

Now checking the M1 round-3 fix (267f9f6a3), which says it is "covered by the next boundary review", then running the tests.

The 4 failures come from this review sandbox: it refuses `git init` in temp dirs, and they hit M1 Discover tests, not M2 code. Next I'll run the M2 tests and a mutation check.

Mutation confirmed red, and the checkout is restored. Now checking whether the Proven path still walks the native store some other way (`resumeEvidence`, `checkRecoveryConversationInUse`).

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: FIX-THEN-SHIP.** M2 does what the plan says. A direct Proven or Fresh selection now sets `SkipCandidates`. The OS reader then still builds `Source`, the screen, the capture, `Agent`, `Generation`, the branch and the issue, and still runs the capture check when there is no `Source`. It skips only the call to `NewNativeTranscriptRuntime` and `CollectRecoveryEvidence`. I traced every place the Proven or Fresh path reads the observation (`recovery_prepared.go:297-311`, `Preserve`, `launchRecoveryClaim`). None of them reads `Ranked`, `Incomplete` or the native `Reasons`. The rest of the Proven path does not walk the native store either: `resumeEvidence` → `bindTranscript` is one ledger read plus one stat. So the issue's purpose is met, the costly scan no longer runs under the hub effect gate. There is a side benefit: the retry loop in `recoveryObservation` (`recovery_prepared.go:39-88`) used to run that 30 s scan twice, because the scan always came back incomplete. It no longer runs at all on this path.

I checked the key test by breaking the code. Disabling the reader's skip branch makes `TestRecoverySkipCandidatesKeepsEveryNonCandidateFact` fail ("skip read a native transcript"). I restored the file and the checkout is clean.

One thing should be fixed before shipping. The plan's Tests section and the issue's Done-when ask for a fixture that is over the entry cap and has an oversized transcript, with transcript reads counted. That fixture was not built and the plan was not revised to drop it.

The M1 round-3 fix (`267f9f6a3`: failures aged from when they finish, `Warm` capping its own wait) is correct. `started` orders entries, `finished` ages them, and both regression tests target the right mutation.

**1. Strengths**
- `recovery_evidence_os.go:199-207`: the skip sits after the whole block that builds `Source`, including the handling of an expired screen. This matches the boundary pair:3 set. It also keeps the capture check when there is no `Source`, so a missing capture cannot turn into a new Preserve attempt.
- `recovery_skip_candidates_test.go:61-66`: the test has a real control. The full read must rank a matching transcript before the skipped read's empty `Ranked` counts as evidence. It also checks every consumed fact for equality and that the skip adds no reasons, across four screen states.
- `recovery_prepared.go:199-203`: the flag is set only on the non-keyed branch. An inferred choice that later turns into Fresh still scans, as the plan requires. `TestPreparedRecoveryScansCandidatesOnlyForAnInferredChoice` covers all three routes.
- `hub_target_memo.go`: the round-3 fix separates ordering (`started`) from expiry (`finished`), and `live()` puts the age rule in one place used by both `Warm` and the sweep.
- `atlas/couch.md` documents the new request field and the `--resume-candidates` wording in the same commit.

**2. Critical findings**
None.

**3. Important findings**
- **Done-when fixture not delivered and plan not revised** (`recovery_skip_candidates_test.go`). The plan asks for "zero transcript reads, counted, on a fixture over the entry cap with an oversized transcript". The Done-when asks for "a fixture with an oversized native inventory … completes or refuses with a reason within the CLI budget (test)". The test uses one 456-byte transcript and infers "no reads" from an empty `Ranked`.
  - The inference holds for this code, and the mutation test confirms it.
  - However, the acceptance test in the issue does not exist, and nothing in Revisions says it was dropped.
  - Fix options: either add a cheap version (one transcript above `RecoveryTotalReadLimit` and a counting `ReadRequest` wrapper, then assert Proven `RecoverSelection` reads nothing), or add a Revisions entry saying that the empty-`Ranked` control replaces the counted fixture.
  - **This is the 3rd finding in family `plan-code-drift`.** Earlier rounds fixed instances. The rule that covers all of them: *before a milestone closes, every Plan Tests bullet and Done-when test it claims is either delivered as written or rewritten in Revisions in the same commit.* Apply that check at each close, not only to this bullet.

**4. Minor findings**
- `recovery_evidence_os.go:199-206` and `224-230` (ARCH-DRY) repeat the capture check that runs when there is no `Source`. If the scan were wrapped in `if !q.SkipCandidates { … }` and execution fell through to one shared check, only one site would exist.
  - **This is the 2nd finding in family `duplicated-guard-structure`.** The rule: a guard exists at one site, and variants pass in what differs rather than copying the guard.
- `resume_candidates.go:123-126`: the new line prints whenever `Incomplete && Proven != nil`. But `Incomplete` is also set when the capture check fails ("recovery screen changed"), and that failure does affect the confirmed resume, because Preserve runs the same check. In that case "affects only the inferred choices" is wrong. Condition the line on the scan being the cause, or soften the wording.

**5. Test coverage notes**
- The targeted M2 tests in couchcore and couchcmd pass under `-race`, and so do the hub memo and admission tests.
- I broke the reader's skip branch, the test failed, and I restored the file.
- Four M1 tests in this sandbox (`TestDiscover*` and `TestSlotEnrollmentRebuilds…`) failed with "git init refused … outside the reviewer's directories (sdlc #339)". That is the sandbox's git confinement, not M2 code. M2 does not touch those paths, so I raise no finding.
- The `--resume-candidates` wording test covers only the Proven-present branch. Nothing checks the plain wording when there is no Proven choice.

**6. Architecture notes**
- **ARCH-DRY:** flagged (Minor, above).
- **ARCH-PURE:** pass. The decision to skip is made in core (`executePreparedRecovery`), and the IO reader just follows the flag.
- **ARCH-PURPOSE:** pass. The scan that motivated the issue is gone from the Proven and Fresh paths, and I found no other native walk on them.
- **ARCH-MOCK:** pass. The tests use the existing recovery execution fixture and a real filesystem in a temp directory.
- **ARCH-CONSTRAINTS:** pass. This takes up to 2×30 s of scan out from under the one-wide hub effect gate. No new unbounded work.
- **ARCH-SECURE:** pass. No new untrusted input. The capture identity check is kept.
- **ARCH-ORDER:** pass. `SkipCandidates` is a per-call request parameter, not state kept between events. The proven-binding check at launch (`launchRecoveryClaim`) is unchanged.
- **ARCH-FUNERAL:** pass. M2 creates nothing durable. The M1 memo has its sweep and its 256-entry cap.
- **ARCH-EVOLVE:** N/A. `RecoveryObservationRequest` is in-process only: it has no JSON tags and is never sent over the wire or persisted.
- The plan's `--recover-plan-from-sdlc` step (third checkbox) is still open and does not belong to M2. It must land before the issue closes.

**7. Plan revision recommendations**
- Either add the counted, oversized-inventory fixture, or add a Revisions entry: "M2 proves no native read through a full-read control (one matching transcript ranked) against a skipped read with empty `Ranked`, plus a mutation test. This replaces the counted over-cap fixture in Tests and Done-when."
- After this round, tick M2 in Design.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: plan-code-drift
    title: |
      Done-when and Plan Tests fixture (over-cap inventory, oversized transcript, counted reads) not delivered nor revised
    detail: |
      3rd finding in this family. Rule: before a milestone closes, every Plan Tests bullet and Done-when test it claims is delivered as written or rewritten in Revisions in the same commit. recovery_skip_candidates_test.go uses one 456-byte transcript and infers no reads from empty Ranked; the inference is sound (reverting the skip branch goes red) but the stated acceptance fixture is absent. Add an oversized-transcript plus counting-reader Proven RecoverSelection test, or record the substitution in Revisions.
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-guard-structure
    title: |
      Capture-identity fence when Source is nil is duplicated in ReadRequest (lines 199-206 and 224-230)
    detail: |
      2nd finding in this family. Rule: one guard, one site; variants pass in what differs rather than copying it. Wrap the native scan in if !q.SkipCandidates and fall through to the single shared fence (ARCH-DRY).
  - id: new
    severity: Minor
    kind: defect
    family: refusal-must-carry-reason
    title: |
      resume-candidates says an incomplete scan affects only inferred choices even when Incomplete came from the capture fence
    detail: |
      resume_candidates.go:123-126 keys on Incomplete and Proven != nil, but a failed validateRecoveryCaptureSnapshot ("recovery screen changed") also sets Incomplete and does affect the confirmed resume via Preserve. Condition the wording on the scan being the cause, or soften it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"77f97969-8591-4d37-92ae-7b149fcf94a7","repository":"github.com/xianxu/pair","issue_id":"000726","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/afb06a2400ca1fd2","reviewed_head":"2e9772fb0bdc2e99cb1d9379591101ad10f1d847","slot":"pair:2","requested_at":"2026-10-20T16:14:54.824962Z"},"phase":"finished","pid":36060,"started_at":"2026-10-20T16:14:54.981843Z","finished_at":"2026-10-20T16:19:42.155721Z","session_id":"1c8649d8-b80d-49c3-b031-050baad5475f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.40653125s
head: "2e9772fb0bdc2e99cb1d9379591101ad10f1d847"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-20T09:26:12-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 726 — Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis |
| repo | pair |
| issue file | workshop/issues/000726-hub-lifecycle-ops-time-out-client-side-resume-pair-6-relaunch-parley-nvim-4-context-deadline-exceeded-with-no-diagnosis.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 5028c01cb1ed223646dd009f6a8ea6ff1b1b7dd6..b5995e632b94b90d31ac92b9be709134a17d63aa |
| command | sdlc milestone-close --issue 726 --milestone M2 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-20T09:26:12-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M2 diff against the plan; reading the issue's Plan/Done-when next.

Tests pass. Next I'll revert the skip branch to confirm the oversized-store test goes red.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 does what the plan says. A direct Proven or Fresh selection sets `SkipCandidates`, which skips only the native transcript scan. The reader still builds `Source`, the screen, the capture, `Agent`, `Generation`, the branch and the issue, and still runs the one capture-identity fence. The inferred (keyed) path still scans. All three prior findings are addressed. I re-ran the M2 tests: all green. I also reverted the `!q.SkipCandidates` guard, and the oversized-store test went red with pair:6's production text: "native transcript incomplete: session inventory read exceeds limit". One Minor finding is left: atlas and the plan's M2 bullet still use the wording BR-15 had corrected in the code. That should be fixed at close, but it does not block.

1. **Strengths**
   - `recovery_evidence_os.go:199-209`: pulling the scan out into `collectCandidates` leaves exactly one Source-nil `validateRecoveryCaptureSnapshot` fence. A skipped read and a full read both reach it. When the native home is unavailable, the read still returns early, as before.
   - `recovery_skip_candidates_test.go:24-79`: every case has a control. If the full read ranks nothing, the test fails instead of passing for no reason. It also checks that the skip adds no reason the full read lacks. It covers the valid, expired, missing-screen and retained-source cases from the plan.
   - `recovery_skip_candidates_test.go:127-181`: this is a real acceptance fixture. It builds more than `RecoveryMetadataLimit` entries and one record larger than `RecoveryRecordLimit`. It counts bytes with the reader's own `BytesRead` and compares against a screen-only read, which the Revisions section records.
   - `recovery_prepared.go:199-204`: only direct selections set the flag, and only right before their own `observe()`. The keyed path calls `observe()` earlier and never sees the flag. The routing test covers both sides.
   - `hub_target_memo.go:32-46`: tracking `started` (which result wins) separately from `finished` (when an entry expires), behind one `live()` predicate, fixes the "slow failure expires at once" bug. `TestHubTargetMemoAgesFailuresFromTheirEnd` pins it.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `atlas/couch.md:83-84` still says "`--resume-candidates` says an incomplete scan affects only the inferred choices". The plan's M2 bullet says the same. The code now prints "comes from the established binding, not from this scan" (`resume_candidates.go:126`), because BR-15 showed the old claim is false when `Incomplete` comes from the capture fence. **This is the 4th finding in family `plan-code-drift`.** The rule: a change that alters user-visible wording or behavior must, in the same commit, sweep every atlas passage and Plan bullet that restates it (grep the old phrase). Atlas and plan must not keep a claim the code dropped.

5. **Test coverage notes**
   - Mutation checked by me: reverting the skip guard makes `TestRecoverySkipCandidatesOnAnOversizedNativeStore` go red.
   - The wording test (`resume_candidates_test.go:196`) covers only the Proven + Incomplete case. That is enough for a single string branch.
   - `TestHubTargetMemoWarmCapsItsOwnWait` uses `time.After(20*slotAdmissionBudget)` only as a hang guard. A pass depends on `errSlotCatalogWarming`, not on timing, so I did not count it as a wall-clock test oracle.

6. **Architecture**
   - ARCH-DRY: pass. One fence, one expiry predicate (`live`).
   - ARCH-PURE: pass. The flag is a pure request field. The IO reader honors it, and the fake reader in the routing test records it.
   - ARCH-PURPOSE: pass. It removes the full-store walk from the confirmed resume, which is pair:6's next bottleneck, and the Spec's `--resume-candidates` bullet is answered. Confirming pair:6 live is still a Done-when item for after the fix.
   - ARCH-MOCK: pass. The fixture uses a real filesystem tree, and the routing test goes through the injected `RecoveryEvidence` seam.
   - ARCH-CONSTRAINTS: pass. The 30 s gate-held scan is gone from direct selections, and `Warm` now bounds its own wait.
   - ARCH-SECURE: pass. Nothing new is trusted; skipped and full reads go through the same capture fence.
   - ARCH-ORDER: pass. The flag is set only on the non-keyed branch, just before its single `observe()`, and the keyed `observe()` runs earlier without it.
   - ARCH-FUNERAL: pass. Memo entries still expire (swept) and are capped at 256; M2 creates nothing durable.
   - ARCH-EVOLVE: pass. Nothing persisted changes. `SkipCandidates` is an in-process request field, not a wire or disk format.

7. **Plan revisions**
   - Reword the M2 `--resume-candidates` bullet to the shipped wording: the confirmed Resume from disk comes from the established binding, not from the scan. Make the same correction in `atlas/couch.md:83-84`.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      TestRecoverySkipCandidatesOnAnOversizedNativeStore builds a store over RecoveryMetadataLimit with an over-RecoveryRecordLimit record and counts bytes via BytesRead against a screen-only read; Revisions records the counting substitution; reverting the SkipCandidates guard turns the test red (verified).
  - id: BR-14
    disposition: addressed
    note: |
      The scan moved into collectCandidates behind !q.SkipCandidates; one Source-nil validateRecoveryCaptureSnapshot site remains at recovery_evidence_os.go:202-208.
  - id: BR-15
    disposition: addressed
    note: |
      resume_candidates.go:126 now says only that the confirmed choice comes from the established binding, which holds whatever caused Incomplete; pinned by TestResumeCandidatesCLIIncompleteScanSparesTheConfirmedChoice. Atlas and plan restatements are raised separately.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: plan-code-drift
    title: |
      Atlas and the plan M2 bullet still claim an incomplete scan affects only the inferred choices
    detail: |
      atlas/couch.md:83-84 and the issue's M2 bullet keep the wording BR-15 removed from resume_candidates.go:126. This is the 4th finding in this family. Rule: a change to user-visible wording or behavior sweeps every atlas passage and Plan bullet that restates it in the same commit (grep the old phrase).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"32fb8751-afe8-4b86-8bb5-38c1210839f3","repository":"github.com/xianxu/pair","issue_id":"000726","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/0e5c1334d2d15431","reviewed_head":"b5995e632b94b90d31ac92b9be709134a17d63aa","slot":"pair:2","requested_at":"2026-10-20T16:24:10.198475Z"},"phase":"finished","pid":79661,"started_at":"2026-10-20T16:24:10.309067Z","finished_at":"2026-10-20T16:26:12.179266Z","session_id":"7edb68f7-08bf-4dbd-bfc1-ef31cea8a1cf","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.66559075s
head: "b5995e632b94b90d31ac92b9be709134a17d63aa"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
