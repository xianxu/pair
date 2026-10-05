---
id: 000367
status: working
deps: [pair#366, ariadne#277, ariadne#278, ariadne#279, ariadne#280, ariadne#288, ariadne#289, pair#363]
github_issue:
created: 2026-10-01
updated: 2026-10-02
estimate_hours: 2.82
card_mirror: 'f0aebb45368c9e7f3e5f293788b7d0266c2d0974' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-02T11:50:33-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Recover local slots from durable issue ownership

## Problem

After restart, humans must remember which local slots owned which issues. Recovery needs to reconcile durable responsibility with local runtime and repository evidence without asserting a globally consistent live world.

## Spec

Project: pair/workshop/projects/cross-slot-work-scheduling.md. Captured for operator review; no implementation is authorized by this issue creation.

Provide a recovery skill consuming canonical read-only SDLC observations and Couch’s local inventory, including parked/stopped slots. Derive a reviewable recovery report: matching assignment and worktree, already running, stopped, deliberately parked, missing worktree, conflicting work, legacy/unattributed claim or foreign machine ownership.

At the operator’s request resume assigned local work in its existing slot (e.g. pair:4 owns #444), preserving branch, dirty files and usable conversation history. Inspect parked state without blindly waking intentionally parked work. Reclaim only under explicit operator direction, after out-of-band synchronization where needed; a crash does not release a claim. Keep observations in the owning binaries; the skill explains/applies them instead of implementing another state scanner. No remote-machine execution/proxy support.

## Done when

(Rewritten 2026-10-03; see the Revision of that date. The original bullets are
superseded.)

- A read-only Couch report (`couch --recover-plan-from-sdlc`, JSON output)
  reconstructs, after a restart, one row per Couch slot path (`:0` and `:1+`;
  off-slot paths are ignored, a vanished slot path keeps its row), joining
  `sdlc fleet inventory` (this machine's claims, dangling claims, slot verdicts)
  with Couch's slot and thread state. Each row carries git state, disk state,
  agent state, the union of work evidence, and a suggested next step with its
  reason. Missing, stale, partial or unknown evidence is shown as such, never
  as absence, and degrades only the rows it touches.
- Suggestions follow the 2026-10-04 operator decisions: evidence that agrees
  suggests resume (or reboot, or asking the slot's agent to restore its
  workspace); work evidence with no visible claim suggests resume and flags the
  claim for repair; conflicting evidence, ambiguous claims, an unreconciled start
  or unknown agent state suggest no step and list the facts. Dirty files do not
  block resume; a Git operation in progress or a detached HEAD never suggests
  reboot; dirty files on a resting branch with no claim, or an idle conversation
  alone, suggest no step.
- `resume <slot>` and `reboot <slot>` (pair#363's operations) are callable through
  the running Couch's socket from a live Couch slot only; any other caller is
  refused; results and refusals are typed.
- Couch's skill (`couch --skill`) documents the recovery procedure an agent
  follows: read the report, review with the operator, resume/reboot per row,
  delegate disk fixes to the slot's own agent via `--send-to`, verify by
  re-reading; it never acts on a slot flagged for recovery.
- Tested with stateful fixtures covering every report row class and the caller
  rule, plus a restart acceptance case through the real report path.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module     design=0.2 impl=0.32
item: smaller-go-module        design=0.1 impl=0.2
item: smaller-go-module        design=0.1 impl=0.2
item: milestone-review         design=0.0 impl=0.2
item: greenfield-go-module     design=0.2 impl=0.32
item: smaller-go-module        design=0.1 impl=0.2
item: skill-or-dispatcher      design=0.1 impl=0.12
item: atlas-docs               design=0.05 impl=0.08
item: milestone-review         design=0.0 impl=0.2
design-buffer: 0.15
total: 2.82
```

Design hours are discounted because the durable plan is reviewed and approved;
`impl=` values are 40% of the v2 primitive ranges (v3.1).

- `greenfield-go-module` — M1 pure DeriveRecoverPlan: union-of-evidence table + derived-domain totality
- `smaller-go-module` — M1 fleet v1 decoder (presence-aware) + stateful FakeFleetSDLC
- `smaller-go-module` — M1 report operation/CLI, fleet_root grouping, ProbeSlotGit fallback, ActorActions move
- `milestone-review` — M1
- `greenfield-go-module` — M2 socket request kinds, receipt store, queue admission
- `smaller-go-module` — M2 --resume/--reboot CLI, caller rule, polling with uncertain outcomes
- `skill-or-dispatcher` — M2 couch --skill recovery section + command sweep
- `atlas-docs` — M1+M2 README/atlas
- `milestone-review` — M2

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* (Calibration doc flagged stale; numbers provisional.)

## Plan

Durable plan: `workshop/plans/000367-recover-owned-slots-plan.md` (reviewed in three
rounds and approved 2026-10-04). Each milestone is a review boundary.

- [x] M1 — Read-only `couch --recover-plan-from-sdlc`: presence-aware decoder for `sdlc fleet inventory --json` v1 (fleets grouped by `fleet_root`), stateful sdlc fake behind ProvisionIO, `ActorActions` single-sourced in couchcore, pure total `DeriveRecoverPlan` over the union of work evidence (derived-domain totality test), ProbeSlotGit fallback for unsupported fleets, restart acceptance through the real dispatch, docs.
- [x] M2 — `resume`/`reboot <slot>` through the running Couch's socket: live-slot callers only (server-side), queued like a switcher keypress (background, operator focus preserved), receipts + polling with "uncertain" on lost outcomes, `couch --resume`/`--reboot --confirm`, skill recovery section, end-to-end report → resume → report acceptance.

## Log


- 2026-10-05: closed — M1 (read-only couch --recover-plan-from-sdlc) and M2 (resume/reboot over the running Couch socket) milestone-closed SHIP; M3 dropped by operator (setup repair moves to pair#387 slot reconciler). Operator smoke test on pair:0 drove fixes, each test-first with mutation: caller rule by Couch liveness (not messaging registration), caller/version-skew messages, store read lock waits briefly for foreground reads and never while holding another lock (adoption/maintenance/family read zero-wait), no resume without an established conversation + launch nonce handed to Pair (fixed the 15s registration stall), per-step launch timings in errors. Final verification: go test ./... (full PAIR_*/COUCH_*/ZELLIJ* scrub) only the 2 pre-existing failures (artifactpath 33 pre-existing entries, none #367; gcruntime locator); couchsingleton contention regression found and fixed (da19d16e); make -k: test-changelog (passes with scratchpad TMPDIR) + test-pair-embedded-runtime env leak from the live Couch slot (passes with full scrub); -race couchcore/couchtty/couchcmd/couchmessage green.; review verdict: SHIP
### 2026-10-01

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.

### 2026-10-02 — surface survey after claim (sdlc 3516e1bf, ariadne#277–280 landed)

What exists:
- `sdlc issue show N --json` (schema v1) is the per-issue observation: claimant
  {operator, machine (hashed), machine_name, workspace "pair:N", worktree,
  repository}, `relation` (this-workspace / other-workspace / unattributed /
  unknown, judged from the cwd checkout), `claimant_worktree` (holds-branch /
  elsewhere / missing / other-machine / unknown), `workspaces.holding[]` with
  dirty/ahead counts and `is_claimant`, and card status/revision. Foreign
  machine = other-workspace + other-machine; legacy = unattributed (5 such
  working issues today: #121, #207, #253, #278, #292).
- `sdlc fleet inventory --json`: every worktree's git facts plus the issue its
  branch prefix names; no claimant, no slot address.
- `sdlc workspace [addr] --json`: resolves `:0` / `pair:N` to kind, path, branch.
- `sdlc help recovery` (#280): claim/reclaim convergent-retry, issue show
  read-only; verify effects via `issue show --json`, never via message receipts.
- `sdlc reclaim`: inspect-then-`--expect REV` CAS, operator-directed only.
- Couch: per-slot `.couch/thread.json` + global records; states live, detached,
  parked, busy, unusable(reason), archived. `OSSlotCatalog.Discover` already
  enumerates slots by the `worktree/<repo>-slotN/<repo>` rule joined with
  `git worktree list`, each verified through `sdlc workspace --json`.

Gaps against the Revisions' assumptions:
1. Couch does not start a preallocated slot set on restart; it only reattaches
   detached/unknown agents. Parked slots stay parked.
2. No operator/Couch-originated delivery: `couch` peer messaging requires a
   live registered slot as sender; there is no way to inject "continue #N"
   from outside a slot.
3. No machine-readable Couch inventory outside a slot (`--list` is text;
   `--actors --json` is live-only, in-slot only).
4. No bulk "claims naming this machine" query; `issue list` has no `--json`.
   Per-issue `issue show --json` costs a tracker read each.
5. Deliberate park vs crash-park is only implied by `verified_park`.

Scope decisions requested from the operator before the durable plan.

### 2026-10-04 — operator decisions on the plan's resolved ambiguities
- 2026-10-04: closed M2 — M2: M1 round-4 advisories fixed by rule (reboot safety folded slot-level across members, domain property; slice alias); resume/reboot <slot> over the running Couch socket with server-side live-slot caller rule, console-queue execution (background, focus preserved), receipts with one mutex owner and polling (fresh per-call contexts, uncertain on dial/unavailable/timeout), couch --resume/--reboot --confirm, skill recovery section with command sweep, step commands that parse; also fixed slot resume missing repo-scope (broken on main since #306, found by the e2e test). Tests written first, each mutation red. go test ./...: only the 2 pre-existing failures (artifactpath 33 pre-existing, none #367); make -k: only test-changelog (passes with scratchpad TMPDIR); -race couchcore/couchtty/couchcmd/couchmessage green; live sdlc conformance ran unsandboxed; actual = sdlc actual cumulative 6.93 minus M1 6.11; --no-project: project tracks pair#367 at issue granularity (final close will update it); review verdict: SHIP
- 2026-10-04: closed M1 — M1 read-only couch --recover-plan-from-sdlc; review rounds 1-3 fixed. BR-14 fixed by rule: per-member evidence model, every judgment reads its own member's facts, union only for the evidence list; domain property test (1.55M points) proves dependency-only changes never alter host judgments; re-merge mutation fails 5 fixtures incl. the 3 reproduced shapes (code preceded tests here; mutations are the red evidence); lesson added. couchcore/couchtty/couchcmd green, -race green, artifactpath 33 pre-existing; earlier full recipe: go test ./... only 2 pre-existing failures, make -k only test-changelog + flaky embedded-runtime cleanup; terminal-status restatement deferred (logged); actual = sdlc actual (first milestone); --no-project: project tracks pair#367 at issue granularity; review verdict: SHIP

- Confirmed: report scope = fleets of Couch-enrolled repositories, same result
  from any slot; JSON only (the agent explains it); repeats refused harmlessly
  once a slot is live (a reboot after a failed fresh start may start a second
  fresh record); admission checked at run time with the switcher's rules;
  remote resume/reboot run in the background without moving the operator's
  screen.
- **Union of evidence.** Work in a slot is evidenced by any of: this machine's
  claim on the path, a checked-out issue branch, unlanded commits, dirty files,
  a Couch thread with a conversation. Work evidence with no visible claim is
  treated as a probably-lost claim (a bug, or an unreadable card): still resume,
  flag the claim for repair. Conflicting evidence goes to the TL agent to
  inspect; no automatic step.
- **Claim read quality is per source, not a blanket hold.** Stale claims still
  inform suggestions (marked "as of"); partial/unknown reads contribute what
  they have; one unreadable card never holds a whole repository.
- **Dirty files don't block resume.** Neither primitive touches disk; dirty files
  on the active branch suggest resume; a Git operation in progress or detached
  HEAD suggests resume, never reboot; dirty on the resting branch with no claim
  gets no step.
- **Rows are per slot path** (`:0`/`:1+` only); off-slot paths are ignored; a
  vanished slot path still gets its row.
- **Several claims in one slot:** the checked-out claim is active; others are
  listed as inactive; a resting branch with several claims is ambiguous (TL
  decides).
- **Busy** is a persisted start claim kept for crash safety (a dead Couch may have
  left a running agent); a row that stays busy after startup reconciliation gets
  no step, and is never rebooted, to avoid a duplicate agent.

### 2026-10-04 — plan approved

Operator approved the durable plan after three review rounds (findings
resolved: presence-aware decode for pre-#288 sdlc, fleet_root grouping,
IsPrimaryRow as the single `:0` rule, warm-only in ActorOperationArgs,
per-call poll contexts, receipt mutex owner, evidence-table totality over a
derived domain, unknown git never read as absence, overlapping mutations).

### 2026-10-04 — M1 implementation notes

All Chunk 1 tasks landed as one commit per TDD cycle (c2fe58da … f7373ad5), each
with red observed before green and the plan's mutations observed red, then
restored from byte copies:
- 1.1 decoder: dropping the section-presence check fails the unsupported cases.
- 1.2 `ActorActions`: `[resume]` for unusable primaries fails both couchcore (2)
  and couchtty (6 switcher tests). The domain test now iterates
  `checkpoint.AllPhases()`.
- 1.3 fake: removing sdlc's "only claims unread" precedence fails the copied
  `TestJudgeCheckout` case.
- 1.4 shell: per-primary instead of per-fleet sdlc runs, probing healthy fleets,
  and returning the store error each fail.
- 1.5 table: all five mutations (6↔13, 11↔13, rule A's operation guard, 3↔rule A,
  `IsPrimaryRow` accepting subdirectories) fail. Totality: 83,504 consistent
  evidence points in about 0.3 s.
- 1.6/1.7: reverting `DirectStoreExecutor`'s case fails the CLI and acceptance
  tests; dropping rule 10 fails the acceptance test.

Deviations (plan Revision 2026-10-04 (c)): 1.5 before 1.4; no `PrimaryScopes`
(pure hash, computed in the derivation); `SlotEvidence.Offer` dimension; new
hold `resume-only`; dirt-unknown restore is resume-only; a slot-target row the
fleet lacks is `evidence-unavailable` per the total table; dependency members
fold into the slot. The full suite caught two repository audits the targeted
runs missed (phase list restated, test-only production symbols), fixed in
f7373ad5.

Live sdlc conformance (`TestFleetInventoryLiveConformance`) ran and passed
both sandboxed (5.7 s; the fetch still decoded) and in the unsandboxed full run,
against the PATH sdlc.

Observed while testing, for the coordinator: an enrolled primary whose
directory is gone makes Couch's whole `Snapshot` fail (`EnumerateSlotCandidates`),
so the report marks every agent unknown (`--list` fails the same way, so this
predates #367). `conflict:issue-terminal` holds any slot still sitting on a
done issue's branch, even when it is clean and unclaimed, which may be common
right after a close.

Verification (Chunk 3, unsandboxed, at f7373ad5): `go test ./... -count=1` fails
only `TestProductionArtifactReferencesAreExactlyClassified` (its 33 entries are
main's 25 plus 8 generated runtimebundle `nvim/review/*.lua` assets; none is a
#367 file) and `TestCouchReferencesLocalArchiveLocatorRoundTrip`, both known.
`-race` on couchcore, couchtty, couchcmd and strictjson passes. gofmt and vet
are clean. `TMPDIR=<scratch>/tmp make test-changelog` passes. `make -k test`:
the first run at 3e9cdcbe failed only `test-changelog`. The re-run at f7373ad5
also failed `test-pair-embedded-runtime`; its smoke passes, then the script's
`trap rm -rf "$tmp"` reports "Directory not empty" for `custom-data`. That
reproduces on standalone re-runs and touches no #367 code (pair runtime only),
so it looks environmental: something keeps writing into the temp tree.
The coordinator re-ran `test-pair-embedded-runtime` standalone and it passed
(rc=0). It is a flaky cleanup race (the test's own `trap rm -rf` meets
"custom-data: Directory not empty" while something still writes), unrelated
to #367.

Coordinator decision, 5e99f024: a clean, unclaimed slot on a
done/wontfix/punt issue's branch is the new class `landed` (note
`issue-done-branch`, no step), not `conflict:issue-terminal`, which now needs
dirt, unlanded commits, an operation or a claim on that branch. Unread
dirt/commits fall to `evidence-unavailable`. Red first: the new fixture got
`conflict`, the coverage test reported no `landed` fixture, and totality
flagged the clean tuple. Mutation (drop the clean condition) fails coverage
and totality.

M1 boundary review (FIX-THEN-SHIP, `000367-recover-owned-slots-m1-review.md`):
- BR-4, claim judged against the wrong member (d5052e19). Each claim is now
  judged against the branch of the member holding it. Host claims keep the
  `Claims` dimension. Dependency claims fold into a new `DepClaims` dimension
  (none/active/resting/conflict/unknown) that the table answers:
  - a restore names the checkout holding the claim
    (`RestoreWorkspaceMessage(ref, address, checkout)`);
  - a claim on another branch is `conflict:dependency-claim`;
  - a host-decided row lists a resting dependency claim as `inactive-claims`.

  Red first on the reviewer's shapes: the resting-dependency claim was
  restored with the host-worded message, and a lost host claim beside a
  dependency claim was `claim-branch-mismatch` (now `claim-likely-lost`).
  Mutation: judging dependency claims as host claims fails 4 fixtures plus
  `TestClaimsAttachToTheirSlotPath`.
- BR-5, unread evidence shown as absence (d5052e19). Dangling claims always
  fold into their slot. On a missing dependency, the claim is a dependency
  claim with an unread member: `partial-evidence [resume]`, note
  `claim-member-unread`. Red first: the row was `idle`. Mutation: reading
  dangling claims only for absent slots fails the fixture.
- Totality: the domain gains `DepClaims`, now 516,784 points in about 1.5 s.
  It admits unread claim quality with host claims present (reachable through
  a weaker dependency read), and pins three facts: a restore has one claim on
  its own member's resting branch, a mismatch is made of host claims, and a
  dependency claim is never idle or landed. Dropping `!work` from rule 9
  fails it.
- Minors (8e8f2aec). Candidate enumeration and stat failures are appended to
  the fleet's error, and only `fs.ErrNotExist` is missing; a permission error
  is present-but-unread. Red first, and mutating back to "any error is
  missing" fails `TestRecoverPlanCandidateErrorsAreRecordedNotMissing`.
- Deferred: `issueTerminalStatuses` restates ariadne's terminal set. It needs
  sdlc to emit terminality on `FleetIssue`, a candidate ariadne follow-up; a
  comment at the constant points here.

M1 review round 2 (4337e9a4 code, 7ce9b0a2 docs):
- BR-10, host-at-rest predicate restated. `hostAtRest` (resting or landed
  branch, nothing unlanded, no operation) is now the one reading. Rules
  `idle` and `landed` and `dependencyClaimDecision` read it; a landed host
  adds note `issue-done-branch`. Red first: the landed host beside an active
  ariadne claim was `unidentified-work` with a false reason; it is now
  `agrees [resume]`. The totality domain already crosses a terminal-issue
  host with every `DepClaims` value; a new invariant pins that a host at rest
  beside a dependency claim is never unidentified work. Mutation (resting
  only) fails the fixture and totality.
- BR-11, docs lag new vocabulary. The atlas now covers per-member judgment:
  `DepClaims`, `conflict:dependency-claim`, `claim-member-unread`,
  `claims.dependency`, the holding checkout in the restore message, and
  `hostAtRest`. The README notes `claims.dependency`.
- Minor, row text contradicts decision. `recoverReason` takes the row's
  facts; the restore text names the claim and checkout and appends exactly
  one of ask / dirty-refusal / unread-refusal from the steps and notes.
  `assertReasonMatchesDecision` sweeps every fixture row: no "no claim" text
  on a claimed row, ask text if and only if an ask step, never ask plus
  refusal, every unacted dependency claim named in the notes. Red first on
  the dirty restore row; mutation (always append the ask text) fails it.
- Minor, evidence dropped from next. `restoreWorkspaceDecision` carries the
  notes for claims it does not ask about: a host restore beside a resting
  dependency claim notes `inactive-claims`. Red first; mutation (drop the
  notes) fails the fixture.

M1 review round 3 (a44adc9b): BR-14, a repeat of the
claim-judged-against-wrong-member family. `hostAtRest` and
`conflict:issue-terminal` read git facts merged across the slot's checkouts.
I fixed the rule, not the site: the evidence model is per member.
- Host dimensions (branch, dirt, unlanded, operation, claims, claim quality)
  are the host's own. Each dependency is judged by `judgeDependency` on its
  own facts and folded into `DepClaims` (adds `resting-dirty` and `work`)
  plus `DepOperation`. `DepOperation` is a slot-level fact that only rule A's
  reboot guard reads.
- The cross-member union is built only for the row's evidence list.
- Vocabulary: note `claim-member-unread` is renamed `dependency-unread`,
  since it now also covers unclaimed unread members; new note
  `dependency-work`. A dependency restore's dirt is the holding checkout's own.
- Fake: member-level `SetMemberDirty`, `SetMemberAhead`,
  `SetMemberOperation`. Fixtures for the four reproduced shapes, plus
  unclaimed dependency dirt, a claimed host beside dependency work, and a
  dirty dependency restore.
- Property: `TestHostJudgmentsReadOnlyHostFacts` runs over the whole derived
  domain (1,554,640 points, about 5 s; the totality test takes another 5 s).
  Changing only dependency facts never moves `hostAtRest`, the
  issue-terminal fact, or a host-only idle/landed answer. Host claim quality
  is again host-only, so unread host quality with host claims is again
  excluded from the domain (a reversal of the round-1 BR-9 note).
- Red and mutations: re-merging the facts in the producer (the pre-fix
  behaviour) fails five fixtures; letting `hostAtRest` read a dependency fact
  fails the property and the totality test.
- The rule is added to lessons.md.

### 2026-10-04 — M2 implementation notes

One commit per task (921d272b … a316320a), tests written first and observed red
(compile red for new symbols, behavioural red for changed ones), plus mutations
restored from byte copies:
- M1 round-4 advisories (921d272b). `DepOperation` → `DepTree`; reboot safety is
  slot-level (`slotOperation`/`slotGitUnknown`/`slotDirty`), host judgments stay
  host-only. Red: a claimed dependency with an unread base got `agrees [reboot]`,
  a dirty one lost `inspect-uncommitted-first`; the alias test failed on a slice
  with spare capacity. Totality gains the property "any member dirty/unknown/in an
  operation ⇒ no reboot without the hold/note". Mutations on each of the three
  helpers fail the fixtures and/or totality.
- 2.1 receipt machine: the table test walks every (state, event) pair; letting a
  different op reuse an ID fails it.
- 2.2 `ActorOperationArgs`: dropping warm-only fails couchcore and couchtty.
- 2.3 console queue: calling `finished` before adoption fails the attach-failure
  test; dropping the reattach clear fails three cases.
- 2.4 socket: skipping `authority.current` fails the caller rule; letting a
  duplicate enqueue fails the duplicate case.
- 2.5 CLI: an invocation-wide context fails the fresh-context test; raw errors
  mid-poll fail the uncertain cases. Fault cases and argv malformations are
  table-driven, one strategy line each.
- 2.6 skill: a bogus class name or an unparsable command fails the sweep.
- 2.7 acceptance: removing the `handle` intercept answers invalid-request;
  dropping `finished` runs the poll into the 3-minute uncertain line; dropping
  the not-offered check lets the resend through.

Found by the end-to-end loop (ec554b43): `resume` declares `repo-scope`
Required, and the switcher's slot effect (since #306) sent only `path`, so
DispatchOperation refuses Tab → resume on any slot row, on main too.
`ActorOperationArgs` adds the slot checkout's own scope; the args tests now
dispatch every shape through the declared table. Lesson added. Worth a
coordinator look: this is live breakage on main independent of #367.

Deviations: plan Revision 2026-10-04 (d) (EnqueueRemoteOperation takes the op;
`remoteOperation` completion state; `newMessageService` takes `*slotOperations`;
a no-identity request is invalid-request; `--confirm` derived from
`OperationConfirms`; socket acceptance uses a detached `:0`).

Live sdlc conformance (`TestFleetInventoryLiveConformance`) ran unsandboxed and
passed (5.95 s, PATH sdlc `~/.local/bin/sdlc`).

Verification (unsandboxed, at the docs commit): `go test ./... -count=1` fails only
`TestProductionArtifactReferencesAreExactlyClassified` (33 entries, none a #367
file) and `TestCouchReferencesLocalArchiveLocatorRoundTrip`, both known. `-race`
on couchcore, couchtty, couchcmd, couchmessage passes. `make -k test` fails only
`test-changelog`; `TMPDIR=<scratch>/tmp make test-changelog` passes. gofmt and vet
clean. Remaining for M2: operator live smoke test, then `sdlc close`.

### 2026-10-04 — smoke-test fixes

The operator's live smoke test found three things; fixed on the branch:
- **Caller rule (operator decision, design change).** `pair:0` under codex was live
  in `couch --list`, but its wrapper's one-shot peer setup had failed, so it had no
  messaging binding and `--resume` answered "unavailable: recipient is not live".
  The operator decided slot-operation callers are authenticated by Couch's own
  liveness, not messaging registration: a live Couch pane for the request's thread,
  and its recorded launch matching the request's session and launch nonce.
  Red first: an unregistered live caller was refused, and the refusal text was the
  broker's. Mutations: re-requiring `broker.Caller` fails the unregistered case;
  dropping the recorded-launch check lets a wrong nonce/session through.
- **Refusal wording.** "caller is not a live Couch slot (…)", tested on every
  caller fault (wrong nonce, wrong session, thread not live, record moved on,
  unknown tag), none of which enqueues.
- **Version skew.** An older Couch's "unknown message operation" becomes a
  restart hint; a table row in the CLI outcome test (red first, mutation fails it).
- **Side-quest: a store read waits briefly for a busy store** (operator-approved).
  `couch --reboot tools:0 --confirm` failed once with "read launch preference:
  resource temporarily unavailable": with slots enabled the preference read takes
  store.lock nonblocking and returned the raw EWOULDBLOCK whenever a write held it
  for a moment. `retentionReadLock` now polls (5 ms) for up to 1 s, never past the
  caller's context, and refuses with `ErrThreadStoreBusy` ("thread store busy;
  retry"). Retention maintenance passes a zero wait, since it yields a busy store to
  its schedule by contract. Red first: a 50 ms hold failed the read with the raw
  errno; past the bound the refusal is typed. Mutations: a zero wait for reads fails
  both new tests; ignoring the caller's context fails
  `TestRepositoryNamesWaitsOutABusyStore`.
- **Remote cold resume timed out (tools:0, investigated, cause not yet evidenced).**
  `couch --resume tools:0` (parked, cold) failed with "await Pair registration:
  context deadline exceeded (waited 15s; session … IS live)". Evidence: the failing
  launch was the 23:26:05 cold resume of `1-tools-15` (helper 65447); Pair finished
  starting (pair wrap 65509, agent-ready launch 4 and the pane sidecar at 23:26:07,
  zellij server 📁1-62 alive), and the earlier remote reboot's fresh start of that
  thread had registered. Ruled out: unattached output (dropped and acked,
  `couchtty/console.go:1187`), size, sink, context, timeout, args and launch shape,
  all identical across origins. The 15 s went inside `awaitResumeRegistration`
  (pane-birth wait or the zellij ownership poll), which the error could not name.
  - The timeout now names the phase that consumed the budget, both phases' elapsed
    time, the number of ownership polls and the last check result (red first for
    both phases; dropping the phase text fails both tests).
  - `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins` drives a parked `:0`
    cold resume through the real console path from the switcher (Enter) and from
    `--resume` over the socket; both register and end live, and both fail when the
    session never comes up. The fakes cannot tell the origins apart: everything
    before adoption is the same code. The next live failure will name the phase.
  - A failure with no diagnostic code no longer prints "failed ()".
  - The operator then saw a switcher cold resume of parked tools:0 come up live in
    about 2 s on this build. A code comparison still finds no remote-only behaviour
    before adoption: the reattach pass and continuation controller defer to an
    operator `InFlight` but share the single queue runner, so neither can run
    alongside a remote job; the launch helper is `pair-launch-helper`, not the
    Couch binary rebuilt at 23:26:03. The console-path test (both origins green)
    therefore cannot encode a difference yet.
  - Every step is now timed in memory and carried into a failure: the launch's
    `[steps: claim, checkout, prepare, spawn, record+baseline, ack, registration]`
    and a remote job's `[remote job: queued, prepare, operation]`, which reach the
    receipt's detail. Red first (and by mutation for the launch steps).
  - **Root cause (evidenced, operator decision: fix both).** The stalled thread
    `1-tools-15` had just been created by a reboot and never took a turn: its
    ledger records a Pair-chosen id with no stored conversation, and the 23:26
    "resume" launched Claude with a brand-new chosen id. #346 M2 admitted that as
    a fresh restart (`bindingResumeDiagnostic`, `resume.go`), whose registration
    (`awaitFreshRegistration`) needs the ready file to carry Couch's start nonce
    (`switchcontext.go` readReadyFile), but Pair takes a Couch nonce only from
    `Orientation.Attempt` (`createflow.go`), so it minted its own and every such
    resume waited the full 15 s, from any origin. The switcher "2 s" was a warm
    reattach, so the origin comparison was void. Fixed: (b) a never-turned thread
    is not resumable (resume and relaunch refuse at once with
    `resume-binding-unbound` naming reboot; the resume-fresh branch is removed;
    a parked binding-lost slot is not offered resume in the switcher, socket or
    report); (a) every fresh launch must hand Pair its nonce through the
    orientation's attempt (`freshNonceReachesPair`, checked before a child
    starts). Red first: resume of a never-turned thread waited out the budget,
    relaunch parked it, a binding-lost slot was offered resume, and the guard
    admitted nonce-less fresh launches. Mutations: re-admitting the provisional
    binding fails both refusal tests; restoring the slot offer fails the
    ActorActions spec, the report fixture and the socket admission test. The two
    relaunch tests that encoded #346 M2's fresh restart are removed.
  - The `tools:1` label on other repositories' never-started slot rows is on main
    (`renderThreadRows` keys labels by an address those rows lack, since #181/#306);
    left for a separate issue.

Environment note: this session now runs inside a live Pair/Couch slot, and the
standard five-variable scrub leaks the rest (`COUCH_ISOLATED_ROOT`,
`COUCH_PAIR_DATA_DIR`, other `PAIR_*`). Under it,
`TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` (couchcmd) and
`TestSpawnComposesProductionPairRegistrationBoundary` (couchcore) fail; with every
`PAIR_*`/`COUCH_*`/`ZELLIJ*` variable unset, both pass, as do the full couchcmd,
couchcore and couchmessage suites and `-race` on couchcmd and couchcore.

## Revisions

### 2026-10-01 — recovery shape settled in operator discussion

Reason: operator review of the captured spec. Delta (still requirements, not a plan):

- **Slots are live at recovery time.** Couch will start a preallocated slot set (resuming parked slots) on restart, so recovery does not have to avoid waking stopped agents. Started slots come up idle; being started does not mean being assigned work.
- **Two steps.** (1) A central, read-only allocation report joins sdlc claims, Couch inventory and per-slot worktree state (branch, uncommitted files, whether the issue is still open) into one row per slot. (2) An operator-approved assignment step: Couch delivers "continue #N" to matched slots only. Conflicts, parked intent, foreign-machine and legacy-owner rows wait for the operator. Delivery goes through Couch, not peer-to-peer slot messaging.
- **Slot-side self-check.** Before working, an assigned slot checks branch → issue → "the claim names me". On a mismatch it reports and does not work. This is a local deterministic guard, not a messaging protocol.
- **Scan scope.** Slots are the repo's `:0` checkout plus `worktree/<repo>-slotN` worktrees; other worktrees (e.g. scratchpad detached-HEAD ones) are not slots. Sources: Couch inventory (switcher view minus archived), claims assigned to this machine, and `git worktree list` filtered by that rule. No filesystem crawl.
- **Lost-slot evidence.** Claim with no slot (no Couch row, or worktree gone) and worktree with no Couch row are both inspected and reported as evidence, naming which source is missing. Neither triggers an automatic reclaim or adoption.

### 2026-10-02 — scope settled after the surface survey

Reason: the survey (Log, same date) found three assumptions of the previous
revision unmet. Operator decisions, delta:

- **Slot-set start is pair#384.** Couch starting the enrolled slots (idle) on
  restart is its own issue; #367 depends on it and does not start slots.
- **Delivery is ordinary Couch peer messaging from a live coordinator slot.**
  A project-management thread (typically `:0`) is always live. After the report,
  the operator decides: "schedule #N", or ask that thread to schedule
  everything by dependency and free slots (the latter is pair#362's ground).
  No new Couch sender identity. Effects are verified through `sdlc issue show
  --json`, per #280, not through message receipts.
- **Bulk claim observation is ariadne#288.** The report consumes one read-only
  bulk claim query instead of a per-issue `issue show` loop; #367 depends on it.
- **The report is a deterministic `couch` subcommand** that owns the join
  (Couch slot inventory × claims × slot worktree state); the skill explains its
  rows and drives the operator-approved assignment, and reimplements no scan.

### 2026-10-02 — slot view absorbed from pair#384; pair#384 closed

Reason: operator design review of pair#384 (closed wontfix). Delta:

- **No automatic slot start.** Couch startup stays reattach-only; parked slots
  may be ready, stale or corrupted, so nothing resumes them unasked. The
  earlier revision's "slots are live at recovery time" no longer holds.
- **This issue owns the Couch slot view:** one row per slot joining ariadne#289's
  per-slot readiness verdict (from `sdlc fleet inventory`) with Couch's thread
  state (live / detached / parked / unusable) and ariadne#288's claims. Couch
  reimplements no git scan.
- **Recovery actions are pair#363's.** Bulk resume of slots holding work and
  bulk reboot of N ready slots fold into #363; this issue adds the claim join,
  the report, and the operator-approved "continue #N" step on top.
- deps: pair#384 dropped; ariadne#289 and pair#363 added.

### 2026-10-02 — owns bulk recovery and Couch-owned workspace shaping

Reason: operator design review; pair#363 is settled as actor-only. Delta:

- **Two layers.** pair#363's resume and reboot get a live agent into a slot
  without reshaping the worktree. This issue is automation: it also wants the
  worktree in a known shape, and composes per slot from ariadne#289's verdict:
  holds-work → #363 resume, then "continue #N"; ready and wanted for new work →
  shape the workspace, then #363 reboot for a fresh agent; needs-recovery /
  missing / unknown → report why and leave it.
- **Bulk commands live here:** resume the slots holding work; prepare N ready
  slots for new work (names TBD).
- **Couch owns workspace shaping**, because Couch owns the named slot
  worktrees: re-check the verdict at action time, switch every checkout of the
  slot to its resting branch (from `sdlc workspace`), run `weave refresh`
  (which refuses dirty trees and active Git operations). Never stash, reset or
  discard. sdlc supplies only the verdict and resting-branch name; weave the
  refresh.
- **Known constraint:** a second `couch` invocation cannot run operations while
  the console runs (owner routing deferred to #147); bulk commands need a route
  to the running Couch or must be triggered from inside it.
- A deleted slot directory is pair#387's repair, not this issue's.
- deps unchanged (pair#363 stays: resume and reboot are its primitives).

### 2026-10-03 — recovery as LLM-driven steps 1-3 over Couch primitives

Reason: operator design session after pair#363 landed. Delta:

- **Steps.** (1) Git: what this machine claims and has not finished. This is
  already `sdlc fleet inventory --json` (`rows[].claims`, `dangling_claims`,
  `slots[]`); no new sdlc work. (2) Git vs disk per slot, and (3) whether a live
  agent is there: a new read-only Couch report that also derives a recovery
  plan, the steps that make each slot ready for an agent to resume. (4) Driving
  agents to continue, the TL-in-`:0` workflow and notifications, is not this
  issue (pair#362 and later).
- **LLM-driven first.** An agent (typically the TL in `:0`) follows the report
  and a skill section, calling Couch primitives. A deterministic one-shot
  verb comes later, once the plan's row classes prove stable.
- **Primitives on the existing socket.** `resume <slot>` / `reboot <slot>` run
  in the running Couch through its operation queue. Callers are live
  registered Couch slots only (the `--send-to` rule); outside callers come later.
- **Disk fixes belong to the slot's own agent.** The TL resumes the slot, then
  asks its agent (existing `--send-to`) to restore its workspace per the plan.
  No agent acts in another slot's repository.
- **Removed from scope:** preparing N ready slots for new work (archive,
  resting branch, `weave refresh`, fresh agent) moves to step 4 / pair#362, and
  so does Couch-owned workspace shaping.
- Naming (operator, 2026-10-03): the report is `couch --recover-plan-from-sdlc`.
- Couch runs `sdlc fleet inventory --json` itself, through its existing
  sdlc/weave shell-out seam (`ProvisionIO`, timeouts, fakes), so the plan is never
  built from stale or partial piped input. "from-sdlc" in the flag names that
  coupling explicitly: the report consumes sdlc's versioned fleet contract
  (schema_version 1) and must refuse any other version.

### 2026-10-04 — Done-when follows the operator decisions

Reason: the first two Done-when bullets still described the superseded rules
(no step for dirty files or foreign claims; a row per claimed issue). Delta:
rows are per Couch slot path, and suggestions follow the union-of-evidence
decisions in the Log entry of 2026-10-04 (with idle conversations and a dirty
resting branch with one claim settled the same day: no step, and resume only).

### 2026-10-05 — M3: rebuild a broken slot (from the smoke test)

Reason: smoke test found `tools:1` almost empty (setup interrupted Sep 27: no
`ariadne` checkout, no generated files; `tools/construct/deps` declares
`../ariadne` without a clone source). Reboot is actor-only and failed at workspace
setup with an unhelpful "open again to retry". Operator decisions:
- New verb `couch --rebuild repo:N --confirm` in #367 (M3): save what's savable,
  remove the slot's checkouts and worktree registration, re-provision that slot
  number via add slot, start a fresh agent. Destructive, and every message naming it
  says so. Reboot stays actor-only.
- Rebuild covers reboot's refusal rows "`:1+` directory missing" and "`:1+` setup
  broken" only; it keeps reboot's refusals for unknown/live/busy/unusable-unknown
  and session-won't-stop, and never touches `:0`.
- "Setup incomplete" must be observable before acting (today no read-only check
  shows it): report class + `couch --show`, the same check rebuild re-runs.
- Saved work joins a registered archive family on storagegc's 60-day retention
  (archives have no size cap today, only age; `recovery/` caps 16 files/64 MB);
  rebuild refuses above 64 MB and lists the largest paths unless the operator
  explicitly discards.
- pair#387 (deleted-slot repair) is superseded by rebuild; close it when #367 lands.

### 2026-10-05 — M3 split: actionable setup errors here, rebuild to pair#387

Reason: plan review of the rebuild chunk (workshop/plans/000367-recover-owned-slots-plan.md,
Revision g; reviewer findings below). Provisioning already repairs a present but
unconfirmed slot in place: `ensureHost` re-runs `weave compile`
(`couchcore/provision.go:177-198`), which re-clones missing dependencies. `tools:1`
fails only because `tools/construct/deps` declares `../ariadne` without a clone
source, so a rebuild would delete the slot and then fail at the same step. Operator
decision: split.
- #367 M3 = actionable setup errors: `setup-incomplete` observable (report class,
  `couch --show`); reboot/resume/report lead with setup's actionable error and the
  in-place fix (fix the cause, reboot re-runs setup); never "open again to retry"
  when a retry cannot succeed.
- `couch --rebuild` moves to pair#387, re-scoped as "rebuild a slot whose
  leftovers block provisioning" (host missing with registration/`main-slotN`/
  unowned environment left; identity or admin-dir mismatch; conflicting setup
  evidence). Carry these review findings into #387's design:
  1. no archive-only slot retirement exists (reboot's is `replaceSlotCurrent` with a
     successor) — specify one;
  2. a resumed rebuild journal must fingerprint what each phase expects and refuse
     on mismatch; add slot/reboot/resume refuse a slot with an unfinished rebuild
     journal; provide an abandon path;
  3. save ignored files too (`.env`, local settings), dependency-clone stashes
     (`refs/stash`) and local-only refs (`--all --not --remotes`), scoped for the
     host to HEAD/`main-slotN`/checked-out branches; ignored paths count toward
     the cap or need `--discard-unsaved`;
  4. the saved-work family needs a real collector (extend `archive_gc` / the Couch
     store GC, not a storagegc family that does not exist), and a carried `.couch`
     `archive/` must not drop out of `couch --archived` if provisioning fails;
  5. derive the vacancy predicate inside `ensureHost` (it also refuses conflicting
     `branch.main-slotN.*` config and a `main-slotN` checked out elsewhere);
  6. hold `.weave-setup.lock` from save through removal (probe without O_CREAT);
  7. `SetupConfirmed` name collides with `provision_host.go:21`;
  8. switcher text must not name a CLI-only verb as if the row could run it.
  Component model and real-git vacancy oracle from Revision g remain good groundwork.

### 2026-10-05 — M3 dropped; #367 closes with M1–M2; setup repair goes to pair#387

Reason: operator decision. pair#387 starts right after #367 as "the slot reconciler":
a declarative model of a slot's resources across git, weave, Couch, the agent and
sdlc, with their dependency graph and idempotent per-resource observe/converge
steps run in topological order (reconciliation, not a scripted saga). Actionable
setup errors and "setup incomplete" detection fall out of the reconciler there
("resource X did not converge: cause; fix"), so M3 here would be a stopgap
replaced a week later. Delta: the M3 milestone row and its Done-when bullet are
removed; their substance, the M3 review findings (Log, "M3 split") and the
component model in the plan's Revision g/h/i move to pair#387. Until #387 lands,
the "open again to retry" setup message remains; `tools:1` is fixed by adding the
ariadne clone source to `tools/construct/deps` and rebooting.
