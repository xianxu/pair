# Boundary Review — pair#367 (milestone M1)

| field | value |
|-------|-------|
| issue | 367 — Recover local slots from durable issue ownership |
| repo | pair |
| issue file | workshop/issues/000367-recover-owned-slots.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | aad9bfa2f104af768d182b31c3357ee45a3eea6a..6326f0190f1d8b67fbfd6ef72d40e151a61b4d36 |
| command | sdlc milestone-close --issue 367 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-04T02:11:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** I checked the M1 range (aad9bfa2..6326f019) against the issue's Done-when and Task 1.1–1.8 of the plan. The plan's claims hold:
- There is a presence-aware v1 decoder with a pre-#288 golden file and typed rejections.
- `ActorActions` is extracted and behaves the same as before (I compared the old and new `menuRowActions` arms).
- `DeriveRecoverPlan` is pure and has a totality test over a derived domain.
- The shell groups primaries by `fleet_root` and degrades per source.
- The CLI operation, the restart acceptance test, README and atlas are all in place.

The recover/fleet/actor tests in couchcore, the couchcmd CLI tests and strictjson pass at 6326f019 (sandboxed). The live sdlc conformance test also passed. The other failures in those packages are PTY and `/tmp` permission errors from the sandbox. Two scratch tests (deleted afterwards; the tree is clean) confirmed two defects in how dependency-member claims are judged. Both make the report suggest something wrong, or show "nothing to recover", for a slot that holds an owned claim. They are cheap to fix and should land before M2 builds `resume` on these rows.

### 1. Strengths
- The rule table is a single first-match `switch` over closed evidence dimensions (`recoverplan.go:430-506`). The totality test covers 83k+ points, and its invariants are stated separately from the code (`recoverplan_test.go:441-483`): no unsafe reboot, unknown is never idle, `landed` only when clean.
- `TestConsistentExcludesOnlyImpossibleEvidence` pins every exclusion in the domain, so the domain can't shrink without a test failing.
- `ActorActions` is one shared table (`actor_actions.go`) that both the switcher (`menu_actions.go:119-123`) and rule A read. The behaviour matches the old arms.
- The decoder checks the version and then that each section is present before the full decode (`recoverplan_fleet.go:197-236`). An older sdlc can't read as zero slots.
- `FakeFleetSDLC` refuses any argv but the real one. The live conformance test pins the vocabulary to the real producer.

### 2. Critical findings
None.

### 3. Important findings

**A. A dependency's claim is judged against the host's branch** (`recoverplan.go:806-818, 859-873, 459-460`)
- Revision (c) attaches dependency-member claims as *inactive* claims. Rule 12 (`ClaimsOneInactive && BranchResting`) then reads them as a host claim stranded on the resting branch.
- Scratch-confirmed: host `pair:1` resting and clean, dependency `ariadne` holding `ariadne#000290` (the live golden's shape), agent parked. The row comes out `restore-workspace` with `Automatic: true` and steps `[resume, ask-agent-restore]`. The message tells the agent to "check out its issue branch" for ariadne#000290 in pair:1.
- A related case: a host on an open-issue branch with a lost claim, plus any dependency claim, turns into `conflict:claim-branch-mismatch` instead of `claim-likely-lost`.
- Fix sketch: judge whether each claim is active against its own member's branch. A dependency claim should never satisfy rule 12 or `claim-branch-mismatch`; keep the dependency claims in `Claims.Inactive` (or a separate dependency list) for display. Add fixtures for both cases.

**B. A dangling claim on a present slot is silently dropped** (`recoverplan.go:670-677, 760-856`)
- `conventionalSlotOfMember` maps a dangling claim on `worktree/pair-slot1/ariadne` (a missing dependency) to `pair:1` and appends it to `s.dangling`.
- `slotEvidenceOf` only reads `s.dangling` when `s.slot == nil`, so for a present slot the claim disappears.
- Scratch-confirmed: the row reads `idle` ("nothing to recover"), the claim list is empty, and `ignored` stays 0. An owned claim shows up as absent, which breaks Done-when's "never as absence".
- Fix sketch: always fold `s.dangling` refs into the row's claims (marked as dangling), and add a fixture.

### 4. Minor findings
- `recoverCandidates` (`recoverplan_source.go:163-168`) swallows `EnumerateSlotCandidates` errors. That fleet's `:1+` rows vanish with no recorded reason; append the error to the fleet observation's `Error`.
- `Missing: slot.Err != nil` (`recoverplan_source.go:166`) treats any stat error, permissions included, as `directory-missing`.
- `issueTerminalStatuses` (`recoverplan.go:601`) restates ariadne's `issue.cue` terminal set (ARCH-DRY). It matches today. The better long-term fix is for sdlc to emit a terminal flag on `FleetIssue`.
- A slot whose dependency member is `missing` still classifies on host facts alone, e.g. `idle` with disk verdict `missing`. Consider at least a note.
- The fake duplicates the issue-branch regex and the terminal set (`recoverplan_fake.go:206,288`). That is acceptable as an independent oracle, but say so in a comment.

### 5. Test coverage notes
- The coverage and totality tests are strong for `classifyRecover`.
- The gaps are in `slotEvidenceOf`, the join and evidence step: claims from dependency members and dangling claims on present slots are only fixtured in the with-active-host-claim case. Both Important findings live there, not in the rule table.
- The totality domain excludes `Quality ∈ {unknown, unsupported, absent}` with claims present. That combination is reachable when the host's claim read fails but a dependency's read lists a claim. Either drop the exclusion or make `slotEvidenceOf` unable to produce it.

### 6. Architectural notes
- **ARCH-DRY:** pass. One finding (terminal statuses, Minor).
- **ARCH-PURE:** pass. `DeriveRecoverPlan` does no IO; `launcher.ResolveRepoScope` is a pure hash; filesystem and git reads stay in `Couch.RecoverPlan` and `recoverCandidates`.
- **ARCH-PURPOSE:** flag (A, B). Owned claims on dependency members are the evidence the report exists to surface.
- **ARCH-MOCK:** pass. The stateful fake sits behind `ProvisionIO` and is backed by a golden file and a live conformance test.
- **ARCH-CONSTRAINTS:** pass. Fleets run sequentially (90 s timeout each, at most 8) on a batch CLI path. The worst case is long, but it is bounded and documented.
- **ARCH-SECURE:** pass. The decode boundary checks version, section presence and duplicate keys; unknown verdicts and reasons become `unknown`.
- **ARCH-ORDER:** pass for M1. The report keeps no state between events. M2's receipt machine is where this matters.
- **ARCH-FUNERAL:** pass. The report creates nothing durable; sdlc's ref update is documented.
- For M2: socket admission should re-derive from the same `ActorActions` at run time, as planned. Fix A first so `resume` never acts on a row that misread a dependency claim.

### 7. Plan revision recommendations
- Add a `## Revisions` entry amending Revision (c)'s "dependency claims attach as inactive claims": a dependency claim is judged against its own member's branch and never drives `restore-workspace` or `claim-branch-mismatch`. Dangling claims on a present slot's members count toward that slot's claims.

```findings
findings:
  - id: new
    severity: Important
    family: claim-judged-against-wrong-member
    title: |
      Dependency-member claims are judged against the host branch, yielding an automatic restore-workspace for a peer-repo claim
    detail: |
      Host pair:1 resting and clean plus dependency claim ariadne#000290 (the live golden shape) gives restore-workspace, Automatic true, steps resume + ask-agent-restore telling the agent to check out ariadne's issue branch (scratch-confirmed). A lost host claim plus any dependency claim similarly becomes conflict:claim-branch-mismatch instead of claim-likely-lost. Judge each claim's activity against its own member's branch.
  - id: new
    severity: Important
    family: unread-evidence-shown-as-absence
    title: |
      A dangling claim on a present slot's missing dependency is dropped and the row reads idle
    detail: |
      slotEvidenceOf reads s.dangling only when s.slot is nil, so a claim on worktree/pair-slot1/ariadne (dependency gone) vanishes from pair:1; scratch-confirmed: class idle, no claims, ignored 0. Always fold dangling refs into the slot's claims.
  - id: new
    severity: Minor
    family: unread-evidence-shown-as-absence
    title: |
      recoverCandidates swallows EnumerateSlotCandidates errors, so slot rows vanish without a recorded reason
    detail: |
      recoverplan_source.go:163-168; append the error to the fleet observation's Error.
  - id: new
    severity: Minor
    family: missing-conflates-unreadable
    title: |
      Candidate Missing is set from any stat error, so a permission error reads as directory-missing
  - id: new
    severity: Minor
    family: restated-vocabulary
    title: |
      issueTerminalStatuses restates ariadne issue.cue's terminal set (ARCH-DRY)
    detail: |
      It matches today; prefer sdlc emitting terminality on FleetIssue so Couch derives it.
  - id: new
    severity: Minor
    family: totality-domain-excludes-reachable-evidence
    title: |
      consistent() excludes unread claim quality with claims present, which slotEvidenceOf can produce via a dependency read
```

---

## Re-review — 2026-10-04T02:43:14-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 367 — Recover local slots from durable issue ownership |
| repo | pair |
| issue file | workshop/issues/000367-recover-owned-slots.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | aad9bfa2f104af768d182b31c3357ee45a3eea6a..84556aeda6382e28797199f7937491634fb16a91 |
| command | sdlc milestone-close --issue 367 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-04T02:43:14-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

Round 3 fixes the two Important findings from round 2. Each claim is now judged against the branch of the checkout that holds it (`judgeMemberClaim`, `foldMemberClaims`, new `DepClaims` dimension). Dangling claims now always count toward their slot. I confirmed both in a scratch copy at 84556aed by reverting each fix:
- Treating dependency claims as host claims fails 4 fixtures and `TestClaimsAttachToTheirSlotPath`.
- Dropping the dangling-claim fold fails the BR-5 fixture.

The candidate-error minors (BR-6/BR-7) now have a test that pins them. The recover-plan, fleet and actor-actions tests pass. Three failures were environmental:
- the sandbox blocks pty children;
- the archive copy has no `.git`, so the live conformance test can't run there;
- generated assets are missing from the archive.

New issue, confirmed in the scratch copy: a slot whose host is landed (clean, on a done issue's branch) and whose dependency holds an active claim is held as `unidentified-work`. Its reason, "work with no claim or issue branch (branch 000016-x)", is false on both counts. That is the only new Important code finding. The atlas also hasn't caught up with the new claim rule.

1. **Strengths**
   - `judgeMemberClaim` (`recoverplan.go` ~1058) judges a claim using the holding member's own `sdlcBranch`. This fixes the class of bug, not just the one site: host and dependency claims can no longer be confused.
   - `RestoreWorkspaceMessage(ref, address, checkout)` names the holding checkout. The table test now pins both the ref and the checkout.
   - The totality domain (516,784 points) gained the new dimension and three facts stated independently of the classifier: restore requires one claim on its own member's resting branch, a mismatch is made only of host claims, and a dependency claim is never `idle`/`landed`.
   - `recoverCandidates` now treats only `fs.ErrNotExist` as missing. Every other error is recorded on the fleet's error and the path stays present-but-unread. The `chmod 0` test is a real reproduction.

2. **Critical:** none.

3. **Important**
   - `recoverplan.go` `dependencyClaimDecision`: it treats the host as at rest only when the host is on `BranchResting`. The `landed` rule also counts a clean done-issue branch as at rest. So a clean `pair:1` on landed `000016-x` with an `ariadne#000290` claim on ariadne's own `000290-y` branch comes out `unidentified-work`, with no step and a false reason. Without the dependency claim the same slot is `landed`.
     - Fix: extract one `hostAtRest(e)` predicate (no dirt, unlanded commits or operation, on resting or done-issue branch) and use it in rules 7/9 and `dependencyClaimDecision`. Add a fixture for this case. (ARCH-DRY)
   - `atlas/couch.md` (#367 section): the atlas got a line for `landed` (5e99f024) but not for the per-member claim rule. Missing: the `DepClaims` dimension, `conflict:dependency-claim`, the `claim-member-unread` note, the `claims.dependency` JSON field, and the fact that restore names the holding checkout. Add one paragraph.

4. **Minor**
   - `recoverReason` for `RecoverRestoreWorkspace` reads "…ask the slot's agent to restore it; no restore request: …" when the slot is dirty. The text contradicts itself, and the claim ref was dropped from it.
   - A host claim and a dependency claim, both resting on their own resting branches: the restore asks only about the host's claim. The dependency claim gets no note (`depNotes` isn't applied in `restoreWorkspaceDecision`).

5. **Test coverage**
   - The BR-4/BR-5 fixtures fail when their fixes are reverted (confirmed above).
   - Missing: a landed host with a dependency claim, and a host claim plus a dependency claim both resting.
   - `TestFleetInventoryLiveConformance` ran sandboxed and unsandboxed at f7373ad5 (issue Log line 214). That covers BR-2 for M1. Record it again at M2.

6. **Architecture**
   - **ARCH-DRY:** flagged (the at-rest predicate is restated); BR-8's terminal set is still restated, but deferred with a comment.
   - **ARCH-PURE:** pass. `slotEvidenceOf` and `classifyRecover` are pure, and IO stays in `recoverplan_source.go`.
   - **ARCH-PURPOSE:** pass. BR-4 was fixed as the class (every claim judged against its own member), not only the site it named.
   - **ARCH-MOCK:** pass. `SetMemberBranch` extends the stateful fake through the same seam.
   - **ARCH-CONSTRAINTS:** pass. No new fan-out.
   - **ARCH-SECURE:** pass. Errors on untrusted layout reads are recorded on the fleet instead of being turned into a fabricated "missing".
   - **ARCH-ORDER:** pass. M1 is a stateless report; the BR-3 recheck belongs to M2.
   - **ARCH-FUNERAL:** pass. The report creates nothing.
   - Noted for later, not raised: if sdlc ever reports the **host** member as `missing`, `slotEvidenceOf` keeps `Dir=present`, and the row reads `evidence-unavailable` or `partial-evidence` rather than `directory-missing`. The point (`GitSourceUnknown`, `DepActive`) that this produces is excluded by `consistent()`. Neither the golden capture nor the fake has that shape, so I haven't counted it as a finding. Pin it if sdlc can emit it.

7. **Plan revisions:** extend the 2026-10 M1-review revision to say the host counts as at rest on a resting branch or a clean done-issue branch, wherever a dependency claim is decided.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Tasks 2.4/2.5 are M2 plan text and are unchanged; still Minor, revisit when M2 starts.
  - id: BR-2
    disposition: addressed
    note: |
      Issue Log line 214 records the live conformance run, sandboxed and unsandboxed, at M1; repeat at the M2 close.
  - id: BR-3
    disposition: not-addressed
    note: |
      Task 2.3 is M2 scope; nothing in this window to recheck.
  - id: BR-4
    disposition: addressed
    note: |
      judgeMemberClaim plus DepClaims; reverting to host-judged dependency claims fails 4 fixtures and TestClaimsAttachToTheirSlotPath (reviewer-run).
  - id: BR-5
    disposition: addressed
    note: |
      Dangling claims always fold into their slot; reverting the dependency fold fails the present-slot-missing-dependency fixture (reviewer-run).
  - id: BR-6
    disposition: addressed
    note: |
      Enumeration errors are appended to the fleet's Error; TestRecoverPlanCandidateErrorsAreRecordedNotMissing asserts "foreign slot path".
  - id: BR-7
    disposition: addressed
    note: |
      Only fs.ErrNotExist is Missing; the permission-denied primary stays present, pinned by the same test.
  - id: BR-8
    disposition: not-addressed
    note: |
      Deferred with a comment at issueTerminalStatuses pending sdlc emitting terminality; Minor.
  - id: BR-9
    disposition: addressed
    note: |
      consistent() now excludes host claims only for absent and unsupported quality.
findings:
  - id: new
    severity: Important
    family: host-at-rest-predicate-restated
    title: |
      A landed host beside an active dependency claim is held as unidentified-work with a false reason
    detail: |
      dependencyClaimDecision treats the host as at rest only on BranchResting, while rule 7 (landed) also treats a clean done-issue branch as at rest. Scratch-confirmed: pair:1 clean on done 000016-x plus ariadne#000290 on its own branch gives unidentified-work, "work with no claim or issue branch". Extract one hostAtRest predicate for rules 7/9 and dependencyClaimDecision, and add the fixture (ARCH-DRY).
  - id: new
    severity: Important
    family: docs-lag-new-vocabulary
    title: |
      atlas/couch.md does not describe per-member claim judgment (DepClaims, conflict:dependency-claim, claim-member-unread, claims.dependency)
    detail: |
      The landed rule got its atlas line in the same window, but d5052e19's new conflict fact, note, JSON field and the holding checkout in the restore message did not.
  - id: new
    severity: Minor
    family: row-text-contradicts-decision
    title: |
      The restore-workspace reason says "ask the slot's agent to restore it; no restore request" when the slot is dirty
    detail: |
      recoverReason's new base text no longer names the claim and contradicts the resting-branch-dirty suffix.
  - id: new
    severity: Minor
    family: evidence-dropped-from-next
    title: |
      A host claim and a dependency claim both resting: the restore names only the host's, and the dependency claim gets no note
    detail: |
      restoreWorkspaceDecision does not apply depNotes, so the second resting claim appears only in claims.dependency.
```

---

## Re-review — 2026-10-04T03:00:48-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 367 — Recover local slots from durable issue ownership |
| repo | pair |
| issue file | workshop/issues/000367-recover-owned-slots.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | aad9bfa2f104af768d182b31c3357ee45a3eea6a..5ec108f224cd476634450865de514c0b1f92b369 |
| command | sdlc milestone-close --issue 367 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-04T03:00:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** All four round-2 fixes hold. For each one I put the bug back in a scratch copy of the code, and the tests failed every time:
- **BR-10:** a single `hostAtRest` predicate is now shared by the `idle` rule, the `landed` rule and `dependencyClaimDecision`.
- **BR-11:** the atlas section on dependency claims matches the code, including the `RestoreWorkspaceMessage(ref, address, checkout)` signature.
- **BR-12:** `recoverReason` builds the restore text from the row's own steps and notes.
- **BR-13:** a host restore that has a dependency claim beside it now carries the note `inactive-claims`.

One new Important bug, which I reproduced in a scratch copy. `hostAtRest` and the `conflict:issue-terminal` check read dirty, unlanded and operation values that are merged across the host and its dependency checkouts (`foldTri`, `recoverplan.go:921`). So a dependency's own work counts against the host:
- **Host on its resting branch, ariadne claimed on `000290-y` with unlanded commits.** This is the normal state of an issue being worked on. The row comes out `unidentified-work` with the reason "the host carries work no claim or issue branch names".
- **Same setup, but the ariadne checkout has uncommitted files instead.** The row comes out `agrees [resume]`, so dirty files and unlanded commits are handled differently.
- **Landed host with any dependency work.** The row comes out `conflict:issue-terminal`.

The landed-host case in the BR-10 fix therefore only works when the dependency has no commits yet. Both wrong outcomes are holds, never wrong actions, so this doesn't block the gate. It should still be fixed before M1 is called done.

1. **Strengths**
   - `hostAtRest` (`recoverplan.go:571`) is now the single definition of "host at rest", and its doc comment explains why each caller checks dirt itself.
   - `assertReasonMatchesDecision` (`recoverplan_test.go:743`) checks every fixture row's reason against that row's decision, not just one row. Making the ask text unconditional fails it.
   - `restoreWorkspaceDecision(e, extra)` passes along notes for claims it doesn't act on. Dropping them fails the "host and dependency claims both resting" fixture.
   - The README and atlas changes are short, accurate and name the real symbols.

2. **Critical:** none.

3. **Important**
   - **Dependency git facts count as the host's** (`recoverplan.go:921`, read at `:550`, `:572`, `:592`). This is the 2nd finding in family `claim-judged-against-wrong-member`. The rule that covers both: any fact used to judge a member is that member's own. Claims (BR-4) were fixed; dirty, unlanded and operation still aren't.
     - Fix: split the evidence into the host's own facts and the slot-wide union. `hostAtRest`, the `issue-terminal` conflict and the idle/landed rules should read the host's facts. The union should feed only the `evidence` work list.
     - Each dependency's facts belong with its own claim. Unlanded commits on the claimed issue branch are expected; on its resting branch they are not.
     - Test gap: the totality domain has one combined `Unlanded`/`Dirty`/`Operation` dimension, so it can't catch this. Add host-versus-dependency dimensions, and add fixtures for a dependency with unlanded commits beside a resting host and beside a landed host. The fake needs a member-level `SetAhead`/`SetDirty`/`SetOperation`; today these only set the host.

4. **Minor**
   - The `landed` rule's `e.Branch == BranchTerminalIssue && hostAtRest(e)` repeats the branch check that `hostAtRest` already does (harmless).
   - BR-1, BR-3 and BR-8 are still open; their dispositions are below.

5. **Test coverage.** `go test ./cmd/internal/couchcore -run 'Recover|ActorActions'` passes and `go vet` is clean. The mutation results for round 2:
   - Making `dependencyClaimDecision` resting-only fails both the fixture test and the totality test.
   - Passing `nil` in place of `depNotes` fails the fixture.
   - Making the ask text unconditional (`asks || true`) fails `assertReasonMatchesDecision`.

6. **Architecture**
   - **ARCH-DRY:** pass. `hostAtRest` is shared. BR-8 is deferred, and a comment at the constant points to the deferral.
   - **ARCH-PURE:** pass. The decision logic stays in pure functions.
   - **ARCH-PURPOSE:** pass. The round fixed the whole class of each finding, not just the named instance (reason sweep, notes sweep).
   - **ARCH-MOCK:** pass with a gap. The fake can't model git facts on a dependency checkout, which is why the bug above went unseen.
   - **ARCH-CONSTRAINTS:** pass.
   - **ARCH-SECURE:** pass.
   - **ARCH-ORDER:** flag, through the merged facts above. The pure state can't tell which checkout a fact came from. BR-3 is still open.
   - **ARCH-FUNERAL:** pass. The report writes only stdout.

7. **Plan revisions.** Add to the M1-review revision: "Host decisions (`hostAtRest`, `issue-terminal`) read the host checkout's own dirty/unlanded/operation; the slot-wide union feeds only the evidence list; each dependency's git facts are judged against its own claim."

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Tasks 2.4/2.5 are M2 scope and the plan text is unchanged; Minor, revisit at M2.
  - id: BR-3
    disposition: not-addressed
    note: |
      Task 2.3 (remote resume origin) is M2 and not implemented in this window; recheck at M2 close.
  - id: BR-8
    disposition: not-addressed
    note: |
      Explicitly deferred with a pointer comment at recoverplan.go:687; acceptable as Minor pending sdlc emitting terminality.
  - id: BR-10
    disposition: addressed
    note: |
      hostAtRest shared by idle/landed/dependencyClaimDecision; landed-host fixture plus a totality invariant; making it resting-only fails both tests.
  - id: BR-11
    disposition: addressed
    note: |
      atlas/couch.md now covers judgeMemberClaim, foldMemberClaims, DepClaims, conflict:dependency-claim, claim-member-unread, claims.dependency and the 3-arg RestoreWorkspaceMessage; all match the code.
  - id: BR-12
    disposition: addressed
    note: |
      recoverReason derives the ask/refusal suffix from steps and notes; assertReasonMatchesDecision fails when the ask text is made unconditional.
  - id: BR-13
    disposition: addressed
    note: |
      restoreWorkspaceDecision(e, depNotes(e)) on the host restore; the both-resting fixture fails when the notes are dropped.
findings:
  - id: new
    severity: Important
    family: claim-judged-against-wrong-member
    title: |
      hostAtRest and conflict:issue-terminal read git facts merged across checkouts, so dependency work counts as the host's
    detail: |
      2nd in family. Rule: any fact used to judge a member is that member's own (claims were fixed in BR-4; dirty/unlanded/operation were not). Reproduced in scratch: resting host + ariadne#000290 claimed on 000290-y with ahead=2 gives unidentified-work "the host carries work no claim or issue branch names"; the same with dirty=2 gives agrees; a landed host with either gives conflict:issue-terminal. Fix: keep the host's own facts for hostAtRest/issue-terminal/idle/landed, use the union only for the evidence list, add host-vs-dependency dimensions to the totality domain, and add member-level SetAhead/SetDirty/SetOperation to the fake plus fixtures.
```

---

## Re-review — 2026-10-04T03:22:04-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 367 — Recover local slots from durable issue ownership |
| repo | pair |
| issue file | workshop/issues/000367-recover-owned-slots.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | aad9bfa2f104af768d182b31c3357ee45a3eea6a..2beedac8e48932273618950c3c4ebfe14cdf452b |
| command | sdlc milestone-close --issue 367 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-04T03:22:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The round-3 fix holds. BR-14's rule is now enforced where the facts are read. In `slotEvidenceOf` (`cmd/internal/couchcore/recoverplan.go`, around lines 960–985), only member 0 fills `Dirty`, `Unlanded`, `Operation`, `Branch` and the claims fields. Each dependency goes through `judgeDependency` and is reduced by `foldDependencies`. The facts of all checkouts are combined in one place only: the row's display-only `Evidence` list. `hostAtRest` and `hostCarriesWork` read only the host's fields. The fake now has `SetMemberDirty`, `SetMemberAhead` and `SetMemberOperation`, and four fixtures use them to reproduce the round-3 cases: a resting or landed host beside a claimed dependency that has unlanded commits or dirt. All four now come out as `agrees`. The recover and fleet tests pass locally (`go test ./cmd/internal/couchcore/ -run 'Recover|Actor|Fleet'`, about 52s). Three Minor findings stay open. Two of them sit on M2 tasks that this boundary doesn't deliver. One new Minor finding: rule A's reboot guard reads only some of the dependency facts.

**Strengths**
- The evidence is modeled per member: `dependencyJudgment` and `judgeDependency` read only that member's branch, dirt, unlanded commits, operation and claims. `DepOperation` is clearly marked as the one slot-level fact.
- `TestHostJudgmentsReadOnlyHostFacts` states the BR-14 rule as a property over the whole consistent domain, not as a single fixture.
- `TestDependencyFactsStayOnTheDependency` pins the producer side: the host's `Git.Dirty` stays `no` while `Evidence` contains `dirty`.
- The totality test keeps strong invariants: no unsafe reboot, no idle with unknown work, and a restore always needs a claim on its own member's resting branch.

**Critical:** none.

**Important:** none.

**Minor**
- The reboot guard in `withRuleA` (`recoverplan.go`, around line 745) checks dependencies only through `DepOperation` and `DepClaims == DepUnknown`.
  - A dependency with an active claim whose dirt or unlanded state is unread still folds to `DepActive`, so reboot is allowed. The same unread state on the host gives `reboot-unsafe-git`.
  - Dependency dirt never adds `inspect-uncommitted-first`. Before BR-14 it did, because the facts were merged.
  - Since reboot replaces the whole slot's agent, the guard should read slot-level dirt and unknown-git across all members, the same way it already reads operation.
- In `restoreWorkspaceDecision`, `notes := append(extra, …)` followed by `notes = append(extra, …)` replaces the first note rather than adding to it. It works today only because `depNotes` returns fresh slices. Building `notes` explicitly would remove that dependency.

**Test coverage notes**
- The host-facts property test only varies `DepClaims` and `DepOperation` on a `SlotEvidence` value whose fields are already separate. On its own it can't catch facts being merged again in the producer.
- That producer-side protection comes from the five member-level fixtures; the commit message says they fail if the facts are merged back together. Leave both layers in place.

**Architecture**
- ARCH-DRY: pass. `hostAtRest` is the single reading of "host at rest", used by the idle, landed, issue-terminal and dependency-claim decisions. `issueTerminalStatuses` still restates the vocabulary's terminal set (BR-8, still open).
- ARCH-PURE: pass. `DeriveRecoverPlan` and `classifyRecover` do no IO.
- ARCH-PURPOSE: pass for M1.
- ARCH-MOCK: pass. The stateful `FakeFleet` now models per-member facts.
- ARCH-CONSTRAINTS: pass. The totality domain is enumerated and timed.
- ARCH-SECURE: pass. Unknown verdicts and reasons degrade to unknown, never to absent.
- ARCH-ORDER: pass for M1 (pure derivation, no state carried between events). BR-3 belongs to M2.
- ARCH-FUNERAL: pass. The report is computed on demand and nothing durable is written.

**Plan revisions:** none needed for M1.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Tasks 2.4/2.5 are M2 scope; still Minor, carry to the M2 boundary.
  - id: BR-3
    disposition: not-addressed
    note: |
      Task 2.3 remote-resume origin is M2 scope; recheck at M2 close.
  - id: BR-8
    disposition: not-addressed
    note: |
      issueTerminalStatuses still restates issue.cue (comment marks it deferred); Minor.
  - id: BR-14
    disposition: addressed
    note: |
      slotEvidenceOf fills host dimensions from member 0 only; judgeDependency/foldDependencies per member; union only in Evidence; SetMember* fixtures at recoverplan_test.go:292-333 plus TestDependencyFactsStayOnTheDependency pin the producer.
findings:
  - id: new
    severity: Minor
    family: slot-level-guard-reads-subset-of-member-facts
    title: |
      Rule A's reboot guard treats only dependency operation and unknown verdict as slot-level, so dependency dirt or unread git allows reboot silently
    detail: |
      A DepActive member with an unread dirty or ahead count allows reboot where the same host state holds reboot-unsafe-git, and dependency dirt no longer adds inspect-uncommitted-first. Fold slot-level dirt and git-unknown across members for withRuleA as DepOperation already is.
  - id: new
    severity: Minor
    family: slice-alias-append
    title: |
      restoreWorkspaceDecision appends to extra twice, so the second append overwrites the first through a shared backing array
    detail: |
      Correct today only because depNotes returns fresh len-1 slices; build notes explicitly instead.
```
