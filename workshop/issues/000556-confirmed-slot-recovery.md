---
id: 000556
status: working
deps: [pair#528]
github_issue:
target: slot-recovery
created: 2026-10-14
updated: 2026-10-15
estimate_hours: 6.75
card_mirror: '5ecc1a115e45b5f378d73b176e25356b1b1f7292' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:10:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Couch confirmed disk recovery and always-reachable fresh startup

## Problem

Follow-up to #528's accepted M1. Startup and Continue are accepted; the revised
M2 disk-recovery execution contract still needs a complete review and operator
acceptance. Existing ranked candidate/CLI/UI and fresh-orientation code is a
baseline, not evidence that every recovery edge is complete.

## Spec

Preserve the slot-continuity target's three outcomes. Explicit Resume from disk
uses a proven native binding or a best-supported/selected ranked candidate with
evidence. Start fresh is always reachable, including incomplete, unsupported,
weak or absent scans and missing/expired tty; disclose what context may be lost.
Use #522's ordinary launch Prompt to point at available previous-screen evidence
and branch/issue, never embed screen bytes in argv. Native resume has no prompt.

Before stopping an attributed running owner, require short-lived, non-persistent
person-or-hub confirmation naming exactly what stops. Recheck identities at both
admission and effect. Unknown/foreign remnants are never killed; leave/report
them and use a fresh name when needed. Preserve intent/evidence before cleanup;
retire only attributed owners, then clean exact dead-owned remnants/stale layouts
without force, revalidate, launch and report registration/readiness honestly.
No silent fresh replacement after an uncertain successful resume. One bounded
observation retry, then an actionable outcome rather than pending limbo.

An unfinished StartClaim with no positive listing must exit Starting when its
owner is dead or its time bound expires, through this confirmed retirement path.
Automatic inference stays OFF pending a witnessed corpus and approved threshold;
retain default plus up to four alternatives with evidence via menu and CLI.

## Done when

- Proven, best and alternate inferred resumes select the right native identity;
  no native transcript has two live owners.
- Fresh executes despite no/weak/incomplete evidence or missing tty, with truthful
  orientation from available branch/issue/screen coordinates.
- Running-owner confirmation works for person and hub; changed identities,
  cancellation, interrupted effects and uncertain launches preserve evidence.
- Foreign remnants are left intact and do not strand the row; stale Starting has
  a bounded exit. No default edits, native-file deletion, git reset or draft loss.
- Tests use measured native shapes and stateful lifecycle doubles; a recorded
  full suite and fresh-context review pass. TL schedules scratch-only operator
  acceptance across proven/inferred/no-candidate/missing-tty/owned/foreign cases.

## Plan

- [x] Inspect current owners and revise #528's M2 design before implementation;
  keep a separate review and operator acceptance boundary.
- [x] M1 — Shared prepared recovery, confirmed Resume/Fresh, standalone hub CLI confirmation, schema-4 intent/read-only v3 migration; headless acceptance and morning scratch script; milestone-close.
- [ ] M2 — Shared native admission reservations and two-slot barrier tests; milestone-close.
- [ ] Record the full suite, close on SHIP and land; the hub owns the operator's restart and live acceptance.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; the calibration source is marked stale by
`sdlc estimate-source`, so this is provisional. Derived after plan-quality
round 2 CLEAN, not a target chosen before decomposition.

M1 has two bounded new Go concerns (prepared selection and ephemeral acceptance),
one lifecycle integration, a schema extension and the existing menu/CLI seam.
M2 adds a reservation reducer/store and the Pair/wrapper handoff integration.
The existing Go JSON, flock, procutil, retention, queue and launch owners avoid a
novel-stack library assumption. Apply the thorough-plan design discount (0.2)
to the module/API/TUI primitive design ranges, familiarity 1.0 for this code,
v3.1's 40% implementation scale exactly once, and the 15% design buffer.
The three review chunks are M1, M2 and whole-issue close.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.3 impl=0.12
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.6 impl=0.6
item: smaller-go-module design=0.06 impl=0.2
item: tui-screen design=0.2 impl=0.4
item: api-integration design=0.6 impl=0.6
item: greenfield-go-module design=0.3 impl=0.32
item: cross-cutting-refactor design=0.1 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0 impl=0.2
item: milestone-review design=0 impl=0.2
item: milestone-review design=0 impl=0.2
design-buffer: 0.15
total: 6.75
```

Design subtotal 2.60h × 1.15 + implementation subtotal 3.76h = **6.75h**.

## Log

### 2026-10-14

Split from #528 at operator/TL direction84fbcea7 so accepted M1 can land. Source:
#528 durable plan's revised M2 and Starting exception; no new code authorized.

### 2026-10-15 — claim and measured design audit
- 2026-10-15: closed M1 — BR-5 fixed at shared owner: validated candidate key resolves Fresh before profile/park/continuation/start guards. Twelve-case real-core explicit/candidate/operation matrix RED-to-GREEN PASS2.870s; genuine native choices still refuse without effects; broad recovery PASS17.992s, race PASS4.407s, core vet PASS, actual menu queue and printed CLI command tests PASS. Required corrected-head recorded suite PASS on 7f289c0089052d02c2b82c453dc7ccc5a5a27d02, exit0 in8m42s, retained suite.log sha2562c788d3dc827; make build PASS. R3 disposed BR-1/2/3/4. Normal re-close, no waivers. Morning script workshop/plans/000556-live-acceptance.md. No M2/live/stage effects.; review verdict: SHIP

Claimed by pair:2 on branch `000556-confirmed-slot-recovery`, based on main
`09c5b48dd7586d6ba908a45ea86899190ad3dd4a`. Refreshed the stale ariadne
substrate before starting planning. Two independent, read-only audits covered
core execution/ownership and menu/CLI admission; no live lifecycle effects ran.

The baseline already has ranked keys, immutable capture preservation, native
owner checks, opaque exact terminal retirement, ordinary launch Prompt and
start-claim persistence. Its deliberate M1 restrictions still reject running
sources, remove Fresh on some rows, require a capture for candidate execution,
and let Starting last while its supervisor lives. Registration uncertainty also
enters generic post-ack cleanup. These are the implementation gaps, rather than
a reason to introduce another recovery queue or launch mechanism (ARCH-DRY).

There is no slot/thread lifecycle CUE model in this Pair tree; the existing
authorities are `threadrecord/record.go`, `threadrecord/recovery.go`,
`couchcore/starttransaction.go` and the park reducer. The plan extends those
owners rather than inventing a second durable lifecycle (ARCH-ORDER).

Follow-up audit found that the native owner observer explicitly does not lock
against simultaneous cross-slot creates. The plan includes bounded shared native
admission reservations at the common Pair resume boundary; the hub must review
this addition before code. The existing two ownership observations alone cannot
prove the issue's no-two-live-writers contract.

### 2026-10-15 — M1 implementation checkpoint

Implemented the shared prepared executor and ephemeral exact-stop confirmation
owner, standalone authenticated hub confirmation, schema-4 tagged recovery with
read-only schema-3 migration, terminal attribution/rechecking, stale-start fencing,
and uncertain-launch retention. Native resume carries no Prompt; Fresh uses the
ordinary #522 launch Prompt pointing at branch/issue and available screen evidence.
The missing-record Fresh path now carries that orientation and strict readiness.

Targeted integrated checks: `go test ./cmd/internal/couchcore -run
'^TestProvenRecovery|^TestPrepared|^TestRecoveryConfirmation|^TestResumeCandidate|^TestRecoveryRetirement|^TestRecoveryFresh|^TestRecoveryStartFence'
-count=1 -timeout=90s` passed (7.124s). Surface and adapter contributors also ran
their focused regression sets; no full-suite or milestone-close claim yet.
Remaining M1 integration covers retained-screen reading, preflight/retention, the
headless matrix, documentation and acceptance script, then the recorded suite.
No live slot lifecycle effects or stage changes were performed.

### 2026-10-15 — M1 integrated headless checks

Integrated exact retention/preflight before stops, normal configuration fallback
for Fresh, safely addressed previous-screen reads after new-tag Fresh, and an
early young-Starting/native park/continuation guard. Real Git broken-host
fixtures exposed two actual failures: checkout identity preflight failed before
confirmation, and readiness attempted repair with unknown agent liveness. The
fix verifies the primary catalog read-only, then uses normal slot convergence
only after confirmed retirement/positive absence; post-repair checkout identity
must still equal the prepared common directory. Operator files survive.

Core recovery/reboot/prepared checks passed at the integration checkpoint
(15.296s); surface/CLI/menu/protocol/schema/launcher focused checks passed
(1.906s/0.304s/0.169s/0.140s/0.245s). Integrated expired-claim/young-claim/open-park
tests passed under race (1.737s). Tagged previous-screen/evidence reads passed
under race, including independent retention leases. Help/skill cost parity first
failed for the new standalone confirmation verb, then passed after recording its
heavy cost and updating the public protocol reference.

The morning script is `workshop/plans/000556-live-acceptance.md`. No live lifecycle
effects, automatic inference or stage changes ran. M1 now proceeds to the recorded
suite and its binary-owned boundary review; M2 exclusion is still outstanding.

### 2026-10-15 — recorded suite found integration regressions

The first recorded suite on `aab86bd03fb0672e9a36c4b16b63462cb1479288` failed
(exit 2, 13m54s). Retained log:
`/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/5b9ca5e5c1cacb22853c746a916aa39e.log`
(SHA-256 prefix `cce070e42401`). No waiver is requested. It exposed exhaustive
artifact inventory omissions, stale command/declaration expectations, lost warm
resume/advice routing, strict-Fresh fixture gaps, and a regression of the existing
unreadable-primary no-stop archive escape. Continuation reboot fixtures waited
for nonexistent fresh registration and reached the core ten-minute test limit.

Corrections classify the actual new artifact consumers (no exemptions), retain
normal facade entry points while removing the obsolete alternate Reboot
stop/launch implementation, and preserve exact unreadable bytes through the
existing archive journal with expected-byte CAS. Valid future versions still
refuse mutation. The primary no-path archive escape reports honestly that no
agent started and never touches its unproved processes. Normal readable Fresh
uses only the prepared executor, including the existing explicit agent choice
resolved from normal saved configuration. No free-form recovery argv is added.

A separate private-fixture measurement found recordless Fresh inheriting ordinary
Fresh's absence gate and refusing foreign/unknown remnants. Recovery-only
admission now allocates a distinct tag in the verified physical checkout, keeps
all exact claim/repository/readiness checks, and performs no workspace
convergence beneath those remnants. It leaves/report them and says setup was
not refreshed; ordinary Fresh remains strict. The initially retained dead legacy
API test strategy was replaced with tests of the shipped shared facade rather
than adding a dead-symbol exemption.

### 2026-10-15 — M1 component verification after corrections

Full `couchcore` passed (223.428s, `go test ./cmd/internal/couchcore -count=1
-timeout=8m`); full `couchcmd` passed (42.140s); full `couchtty` passed (31.527s).
The exhaustive artifact classification/dead-symbol guards pass (3.158s), without
allowlists or waived gates. Candidate dispatch retains its thin typed facade.
Warm-only/lost-pointer Continue never invokes workspace convergence or launches
an agent. Actual native cold resume uses prepared recovery; existing #522
Pending/Running continuation requests retain their separately owned executors
after positive-listing priority. The old implicit idle-approval argument is gone
from Fresh requests: the exact returned stop set is their only stop authority.

The recorded full-suite rerun is next on the corrected head; previous failures
remain recorded on their actual head. No operator acceptance, M1 SHIP, stage or
M2 implementation is claimed yet.

### 2026-10-15 — recorded rerun finds failed-attach completion regression

The suite at `2630da264cf785423b5e165e7f0b9dee1421a240` exited 2 after
9m7s (retained failed log `da88df7467432ddf832660f8a97202b9.log`, SHA-256
prefix `6ccb29b2acd1`). It passed full core
(284.887s), launcher (44.057s), tty (45.070s) and the other reported packages,
but `TestStartupWarmPassPaintsAndAcceptsInputWhileMixedAttachmentsBlock` failed
waiting for its rendered failed-attach reason. This was a real regression, not
a load waiver: both attach attempts finished, yet the row remained attaching.
The shared resume route returned an empty `StartResult` on pre-helper failure;
`Started()` advertised it and Console replaced the operation's exact target
with a zero address, so the completion could not finish the row's attempt.

The producer now returns no child for that definite no-helper failure. The
Console boundary also keeps its original target when a malformed/empty child
result has no address, while preserving genuine partial launch addresses and
handles on errors. The stateful console reproduction failed before the guard
and passes afterward; the original mixed-startup acceptance passes unchanged
(2.94s). New warm failure coverage passes with the warm-route selection (0.710s).
A fresh recorded suite will cover these corrections; M1 is not closed yet.

### 2026-10-15 — M1 recorded verification ready for boundary review

`sdlc local-check run --name suite` passed
at `b7fef55184f1c6e5f8f0a929419f0cd214d3d3e9`, exit 0 in 10m40s. The retained
log is `.git/sdlc/local-checks/b7fef55184f1c6e5f8f0a929419f0cd214d3d3e9/suite.log`
under the Git common directory; SHA-256 prefix `917e49963187`. Full core passed
388.885s, cmd 99.348s, tty 60.260s, launcher 42.659s and wrapper 53.973s. No
waiver, timeout increase or known failure is excluded. `make build` also passed
on that head; the checkout remains on the issue branch, no stage changed.

The three exact headless commands in `workshop/plans/000556-live-acceptance.md`
also passed: core 18.149s; cmd/tty/message 1.818/0.233/0.174s; launcher 0.206s.
The morning live script is ready for the hub/operator, with scratch allocation,
proven resume, explicit candidate keys, standalone stop-token acceptance, Fresh
orientation and previous-screen reads. No live acceptance is claimed. M1 does
not claim cross-slot native exclusion; that remains M2's separate review boundary.

### 2026-10-15 — M1 review round 1: REWORK, three families addressed

The normal boundary review at `a2b67ce4127cd90c06961204a4feed4c213dd1b2`
returned REWORK with BR-1/2 Critical and BR-3 Important. No milestone was closed.
The retained sidecar and close ledger record all findings; no dispute or waiver
is needed because source inspection and RED tests confirmed them.

- BR-1 (`recovery-preserves-surviving-workspaces`): retained and recordless Fresh
  share read-only readiness while source absence is unproved, rechecking after
  claim. The strengthened foreign fixture initially counted one forbidden normal
  readiness call. The fix leaves fixture sentinel/files/processes intact; absent
  sources retain normal readiness. Focused tests passed 6.619s; broader recovery
  tests passed 15.466s. Invalid/unavailable host evidence refuses without repair.
- BR-2 (`retirement-requires-owner-absence`): exact confirmed retirement now
  checks immutable generations before each leaf-first signal and independently
  waits for every named process to disappear; server absence alone never succeeds.
  Surviving/reparented/replaced/new owners, false signal success and final listing
  races are covered. Strict old retirement gains only the absence oracle, not
  broader kill authority. Full launcher race tests passed 80.131s. Connected
  settled owners use ParkExpected after preservation; the helper is named in the
  same stop offer. Failed park retains its transaction without fallback. New
  stateful park ordering/failed-publication/unknown-helper tests pass; core race
  selection passed 1.940s and broader recovery/Park selection 27.092s.
- BR-3 (`user-facing-docs-follow-surface`): README and the older Atlas accounts
  now describe standalone exact-stop confirmation, expiry, always-available Fresh,
  bounded Starting, retained uncertainty and survivor-safe workspace handling.
  README documentation guards passed 0.415/0.653s; deleted flow names were swept
  from the obsolete Atlas reboot account. Plan Revisions and lessons record the
  classes and their independent test oracles.

A new recorded suite and normal M1 re-close follow these corrections. M2 remains
unimplemented, and no operator acceptance or live effects are claimed.

### 2026-10-15 — unrelated shell flake before corrected Go verification

The recorded suite at `db86a287f8c74bbda952dfe00356e3b8f11d4601` exited 2
after 3m39s in the unchanged `tests/review-fresh-restore-test.sh`, before the Go
phase. Case `missing-history-does-not-adopt-old-receipt` observed a nonzero
Neovim return with empty stdout; the assertion omits its return code. Earlier
cases passed. The identical full script immediately passed all five isolated
cases. Root cause is unproved; no timeout was increased and this is not waived.
Retained failed log `eef944b68fc13088226c6a06a1a85b99.log`, SHA256 prefix
`92b0a48d20e5`. Filed independent unclaimed pair#568 with bounded evidence.

Artifact classification/dead-symbol checks pass (4.608s), the corrected build
passes, and the complete recovery/Park selection passes 27.092s plus targeted
race checks 1.940s. The next recorded suite must still pass on this code before
M1 re-close; a passed isolated flake rerun is not whole-suite evidence.

### 2026-10-15 — repeated #568 blocks the recorded suite

The next recorded run at `c68e49762163129bba1eb51f5f463cd926379caa`
again exited 2 before Go after 3m38s, now in `ambiguous-history-does-not-open`.
Log `c78c884add12c68765db6e7583413142.log`, SHA256 prefix `597c86022693`.
No recovery source changed between these runs. A diagnostic isolated copy adding
return code/result JSON still passes after stripping PAIR/COUCH/ZELLIJ session
environment. This rules out that simple environment hypothesis; the cause is not
yet established. Pair:3 now owns independent pair#568 from main. The hub and
owner have the bounded evidence; read-only measurements continue without a
stacked fix, timeout increase, gate waiver or M1 close claim.

### 2026-10-15 — checkpoint while independent suite repair and policy reply wait

All M1 review corrections are committed; no M1 milestone-close is finalized.
The passing complete suite is on the earlier pre-review-correction head. The two
corrected recorded attempts stopped before Go in independent pair#568, now owned
by pair:3. The bounded probe caught Neovim rc=1 after valid result JSON and an
`uv_loop_close() hang?` log; live fs_event/timer handles exist in successful runs
too, so the responsible handle/race is not yet proved. Probe evidence is under
`/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair568-diag-el_buyz1` and was
sent to the owner and hub. No #568 code/claim is taken in this branch.

Next: after #568 lands, merge main, run the recorded suite on that head, record
its actual result/build, and re-run normal M1 milestone-close to answer BR-1/2/3.
Do not skip any gate or reuse #528 waivers. Then implement M2 at its separate
boundary; the direct native-selector policy question in the latest plan Revision
is pending the hub's answer (receipt a2e880d4-619c-43c7-92a1-6d8e5413efcd,
submitted). No M2 code, stage modification, operator restart or live-slot effects
have occurred. The completed morning script remains in the plan directory.

### 2026-10-15 — independent suite blocker landed; merged main

Verified peer receipt `03f4a082-3baf-4255-ac04-3460ff3be56f`, then fetched and
merged origin/main containing pair#568 PR345, merge
`c0d469a68185402455c1f7ad2e2960d914506740`. The merge was clean; artifact
classification and lessons auto-merged. Its watcher ownership/teardown repair and
retained failure diagnostics remain intact alongside #556's recovery changes.
No history artifacts were read to infer acceptance. Deeper temporary-directory
socket refusal is separately tracked as #570, not conflated with this blocker.

A new recorded suite and build on the merged #556 head follow; only an actual
passing result will support M1's normal re-close. Resolve-and-pin policy is
approved and recorded for M2, but no M2 implementation or live/stage effects have
started.


### 2026-10-15 — native latest audit and temporarily unavailable hub channel

A read-only M2 audit distinguishes native latest selection from recovery tail
ranking. Codex 0.162.1 uses provider UpdatedAt and Cli+VsCode by default (Exec and
AppServer only with include-non-interactive); its worktree/all modifiers can
broaden cwd scope. Existing inventory validates canonical exact roots for all six
agents, but does not yet expose all native latest eligibility/order facts.
Recovery semantic activity, transcript mtime and creation time are not silently
interchangeable. The proposed narrow resolver must report unsupported/incomplete
rather than guess; approved resolve-and-pin remains the required behavior where
resolution is established. This audit starts no M2 code.

A bounded clarification report to the hub alias, then the dispatch slot ops:0
after read-only actors showed it, both refused `unavailable: Couch cannot find a
live caller or recipient registration` with no receipt. No uncertain send is
replayed, no terminal input or live recovery is attempted, and no operator restart
is requested. Preserve the question for the next verified reachable hub: may
unmeasurable provider selectors refuse with exact-ID/Fresh advice while measured
native latest resolution is supported? M1 verification continues independently.


### 2026-10-15 — corrected M1 complete verification after #568 integration

Recorded `sdlc local-check run --name suite` on
`ecb0daa2845ebe248f3fb598ffff2f38f31d12a5`: **PASS**, exit 0, 8m20s.
Retained log:
`/Users/xianxu/workspace/pair/.git/sdlc/local-checks/ecb0daa2845ebe248f3fb598ffff2f38f31d12a5/suite.log`,
SHA-256 prefix `73d52450dbb2`. Fresh-restore and diagnostic regressions now pass
with main's independent #568 watcher lifecycle repair. Go packages all pass,
including couchcore 258.739s, couchcmd 50.160s, couchtty 43.056s, launcher 32.579s,
wrapcmd 45.600s. `make build` also exited 0 on this code generation; no live/stage
operation was performed. The same M1 headless/morning scratch script remains at
`workshop/plans/000556-live-acceptance.md`.

This supplies whole-suite evidence for the post-review BR-1/2/3 corrections;
normal M1 milestone-close answers those findings next. Native admission remains
M2 and is not claimed by these results.


### 2026-10-15 — M1 round 2: prior findings addressed; copylocks correction

Normal M1 review round 2 at `a682dc8e4ccfa2ccb6717e9592e30bccc88db27f`
explicitly disposed BR-1/2/3 as addressed. Its FIX-THEN-SHIP verdict still left
Important BR-4 open, and the binary **did not finalize** the milestone; its
explicit next action is fix and re-run milestone-close, without a waiver.

BR-4's class is synchronized owners copied to substitute preview dependencies.
Reproduced `go vet ./cmd/internal/couchcore` exit 1 at production preflight and
its test fixture. Recovery now passes a read-only ThreadStore into the existing
ordinary/profile resolvers, preserving the actual Couch pointer and all its
mutexes/lifecycle ownership. Numbered profile resolution already accepts its
store directly. The test now constructs only its needed Couch dependencies.
Swept all Couch copy assignments; remaining ThreadStore preview copies contain
configuration/pointers but no synchronization primitives.

After correction: core vet **PASS** exit 0; targeted profile/preflight/prepared/
Fresh/start-resolution tests **PASS** 13.829s. Broader `go vet ./...` identifies
one independent pre-existing diagnostic at
`cmd/internal/pairlifecycletest/live_zellij_diagnostics_test.go:20` (unused String
result), not a synchronized-owner copy. No unrelated source edit is included.
The previously recorded complete suite passes the preceding code generation;
a new pushed-head recorded suite follows this correction before M1's re-close.


### 2026-10-15 — BR-4 complete verification and reporting

The corrected pushed head `52c5f6aceb31d3f9fc0980d03c34dcd1b050643d`
passed the required recorded suite: exit 0, **8m38s**, retained log
`/Users/xianxu/workspace/pair/.git/sdlc/local-checks/52c5f6aceb31d3f9fc0980d03c34dcd1b050643d/suite.log`,
SHA-256 prefix `e114fbb2b590`. `make build` passed, including rebuilding pair,
couch and the launch helper. Profile/preflight race tests passed in **2.990s**.
Core vet remains clean after its reproduced copylocks failure.

Hub messaging subsequently recovered. The round-2 outcome/fix report
`4ef4dee2-16c8-4492-be13-767b67bf2463` and M2 latest-provider clarification
`32602d27-64b2-42a4-972f-c2e3e7c10cff` are both verified **submitted** to ops:0.
The policy question now names measured Codex/Claude/Qoder native ordering versus
unproved Grok/Muse/Agy indirect selection. Exact canonical IDs remain supported
by all six inventory scanners. No M2 code or live/stage effects have occurred.

Normal M1 re-close follows the binary's explicit next action for open BR-4;
no bypass flags or manual ledger dispositions are used.


### 2026-10-15 — M1 round 3: Fresh candidate normalization correction

Round 3 at `dead1b665bddf4d053df5faba06b54a73420296a` disposed BR-4 (and
reconfirmed BR-1/2/3), but returned **REWORK**, Critical BR-5. M1 remains open.
A Fresh key entered as RecoveryInferred and encountered native profile/park/
continuation/young-start guards before it was recognized as Fresh.

Reproduced all four guard families: explicit Fresh controls passed while keyed
ResumeCandidate and the declared live-owner operation each refused. The 12-case
matrix also asserts genuine native candidates still refuse without effects,
then verifies one Fresh launch, no native session ID, a nonempty orientation
Prompt and no signals. Corrected the shared owner to validate/select a key and
normalize its mode before profile resolution and every native-only guard;
non-keyed modes keep their existing cheaper preflight order. Evidence is read
once per operation, not replayed. Stale/invalid keys retain their refusal.

RED-to-GREEN matrix **PASS** 2.870s, broad prepared/candidate/Fresh/selection/proven
suite **PASS** 17.992s; matrix race run **PASS** 4.407s; core vet **PASS**.
Independent surface tests at `31497237e` execute the actual printed command and
menu owner queue, preserving keyed Fresh versus keyless/explicit reboot through
the existing transport. Root rerun: tty **PASS** 0.293s, cmd **PASS** 0.296s.
No production surface change was needed. The next corrected-head recorded suite
and normal re-close follow; no gate is waived and no M2 code has started.


### 2026-10-15 — BR-5 full verification

Corrected code head `7f289c0089052d02c2b82c453dc7ccc5a5a27d02` passed the
required recorded full suite: exit 0 in **8m42s**, retained log
`/Users/xianxu/workspace/pair/.git/sdlc/local-checks/7f289c0089052d02c2b82c453dc7ccc5a5a27d02/suite.log`,
SHA-256 prefix `2c788d3dc827`. `make build` passed, rebuilding pair/couch/helper.
The subsequent native-selector measurement checkpoint changes only the plan.
Normal milestone-close must dispose BR-5 before M1 is considered closed.

The round-3/fix report attempted after the commit refused `unavailable` without a
receipt; no uncertain delivery is replayed. Read-only peek subsequently confirmed
the hub's terminal is functioning. Next report will name the actual new boundary
outcome. No stage, restart, native session experiment or M2 code has occurred.

### 2026-10-15 — M2 common launch admission checkpoint

M1 closed normally on SHIP at `69c213c289ac9f98aa7e16a9e9c1b54f76c90d26`;
M2 implementation now proceeds under the approved split. The first launcher
regression run failed functionally: all six native resume forms reached handoff
without a reserved grant, refusal still launched, inherited authority survived
Fresh, and both cancellation cases had zero reservations. After wiring the
common Runtime boundary, the same matrix passed in 0.185s. The extended launcher
regressions passed in 1.565s, including latest resolved once then pinned despite
a newer native session, unresolved inventory without writer publication, and
fork source IDs never recorded/reserved as the new writer.

Admission now precedes compatibility/native launch ledger publication. The
command digest covers the final executable/argv, including launch-only runtime
arguments. Authority remains private until terminal handoff, after title-helper
startup; definitely unexposed failures cancel the exact reservation, while an
uncertain terminal start retains it. The wrapper separately consumes authority
before PTY spawn, publishes strict child ownership before readiness and removes
it from native/watcher/Fresh environments (`c24060bd6`). The Zellij handoff still
uses its existing environment transport; no new persistent token transport or
layout rendering is introduced. Existing same-user process inspection is not a
security boundary; a consumed grant cannot start a second native writer.

The inventory provider-policy question remains submitted to the hub
(`32602d27-64b2-42a4-972f-c2e3e7c10cff`), with no answer observed. In particular,
Codex effective provider/SQLite-home configuration is not yet positively resolved
by the OS launcher adapter: do not claim latest support from unit fixtures alone.
The internal result is honestly Unsupported until that production seam is
established. No native files/defaults were changed; no live slot effects or stage
operations occurred. Full M2 verification and its independent boundary remain.

## Revisions

### 2026-10-15T05:33-07:00 — overnight M2 dispatch

The hub assigned #556 after #528 landed: plan to the hub, implement and land on
SHIP, with headless acceptance and a short live script for the operator's morning
restart. The issue now references the newer `slot-recovery` target; the earlier
`slot-continuity` commitments still constrain teardown, confirmation and Fresh.
The durable plan is awaiting the hub's approval; no implementation is started.
No messaging reconciliation (#561), wrapper endpoint reconstruction (#557),
automatic inference, live-slot experimentation or stage changes are included.

### 2026-10-15 — approved milestone split

Verified hub receipt `249a1bbd-5f2f-457a-b009-c2581eae3da9` approves the plan
with two separate milestone-close boundaries. M1 implements confirmed recovery
and schema migration, including standalone `couch --recovery-confirmation TOKEN`
for the hub and the morning scratch script. M2 separately enforces native
admission against concurrent writers. Tokens/reservations/schema are internal;
visible outcomes stay Continue / Resume from disk (confirmed) / Start fresh,
plus needs relaunch, with only the actual stop confirmation shown. Retain the
five-minute stale-Starting bound and one exact reobservation. Change-code next.

### 2026-10-15 — M2 selector policy decision

Verified hub receipt `14aba7d7-b325-43dc-a16c-6d8d23907f23`: resolve and pin
indirect latest selectors through existing native inventory for exact cwd/agent;
reserve and execute the same explicit canonical ID. Refuse only unresolved or
ambiguous selectors/name mappings; do not regress supported continue/last or edit
saved defaults. The durable plan Revision supersedes the earlier blanket-refusal
proposal. Native admission remains M2, after M1's independent verification block.
