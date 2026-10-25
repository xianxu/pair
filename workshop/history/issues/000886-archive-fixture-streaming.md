---
id: 000886
status: done
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'f5e75bc57b5b80a1367bfbfa4dd6b31174f98a35' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T09:20:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:11
    worktree: /Users/xianxu/workspace/worktree/pair-slot11/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6129d9e9", done: "7e1e8089"}
actual_hours: 0.58
---

# Stream oversized archive protocol fixtures and restore the couchcmd write allowance

## Problem

pair#737's recorded suite crossed couchcmd's96MiB own-process physical write
budget. After removing redundant scheduler setup, all functional tests pass,
but measured totals remain101781504 and101408768 bytes. A native sampler
(RUSAGE_INFO_V2 on the owned exact test process, top-level stdout intervals,
no read/identity errors) attributes16842752 bytes to
TestRunArchivePassRefusesInvalidProtocol. Its two oversize fixtures each write
an entire MaxContinuousReportBytes+1 response to disk before the helper emits
it, then production retains the bounded diagnostic log.

TL ops:0 delivery5ff9ae2d authorizes a separately filed, unclaimed fixture
repair; archive fixtures must not change on #737. After auditing/trimming its
own fixture writes, #737 may use a measured104MiB allowance. This follow-up
must restore96MiB when its streamed fixture makes that viable.

## Spec

Generate oversized fixture stdout incrementally in the existing owned archive
protocol helper rather than persisting the complete oversized response first.
Preserve the production parser, capture limit, timeout and cleanup behavior.
Keep both non-JSON oversize and a valid JSON prefix followed by whitespace at
the exact existing MaxContinuousReportBytes+1 boundary. Keep every small
malformed/trailing/empty, diagnostic, cancellation and inherited-descriptor
case and its assertions. No hardlinked mutable payload, skipped boundary,
smaller payload, write-budget bypass or production cap increase.

## Done when

- Oversize fixtures emit the original exact logical byte counts and contents
  through the real child/stdout path without writing a full oversized source
  file. Tests pin the bounded setup and still reject a valid-prefix overflow.
- All archive protocol cases retain original diagnostic and partial-report
  authority/cleanup assertions and pass.
- Measure the saved own-process physical writes and run the complete couchcmd
  package with its96MiB cap. Restore cmd/internal/testdisk.CouchCmd to96MiB if
  #737's104MiB allowance is present; if it has not landed, retain96MiB and
  coordinate qualification with #737. Keep actual excess-write refusal tests.
- Recorded pre-close and boundary review pass. Update the performance atlas
  allowance text if #737 changed it. This is a fixture efficiency correction,
  not a relaxation of archive acceptance or a new runtime protocol.

## Plan

- [x] Stream the two oversize suffixes from the existing helper in bounded blocks;
  keep small prefix files and assert their setup size. Pin exact max+1 output
  length/content and bounded write sizes with an in-memory fixture test.
- [x] Verify protocol/refusal/cancellation/descriptor tests and compare native
  before/after write counters; retain excess-budget refusal coverage.
- [x] Audit changed reader inputs, repair only its inventory, update the
  testing atlas, and queue the recorded pre-close check.

Workflow postconditions: the recorded check must pass complete couchcmd at the
unchanged 96 MiB cap before landing. Close owns the mandatory boundary review;
landing follows the TL's SHIP instruction.

## Log

Filed unclaimed per ops:0 while pair:3 remains on #737. Measured attribution
and retained failure details are in #737's Log. Primary implementation sites:
cmd/internal/couchcmd/archive_schedule_protocol_test.go's
archiveProtocolFixture, TestArchiveProtocolProcess and
TestRunArchivePassRefusesInvalidProtocol; existing budget constant in
cmd/internal/testdisk/budget.go. Independent work can start from main; no
stacking on #737, and no source edits were made to the archive fixtures here.

### 2026-10-25 — resumed in pair:11
- 2026-10-25: closed — Recorded pre-close suite PASS exit 0 in 14m4s on unchanged implementation dd3570f1aa18dcffc1eed4beb2a387088362103b; complete couchcmd PASS 77.066s with original 96 MiB write guard; all suite packages PASS. Exact max+1 child output, refusal/cleanup, padding failures and excess-write guard PASS. Same-selection native writes save 8392704 bytes. Bounded evidence: workshop/evidence/000886/pre-close.txt. BR-1 required verification supplied; later commits prose only. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Recovered the specified previous recording (startup only) and empty local Log;
claim and start-plan brought fresh main details, including the filed scope.
A generated merge-check.yml edit blocked start-plan. TL authorized restoring
main's copy; refresh now correctly deferred that local-gates seed and planning
passed. Generated missing runtime embed assets via make runtimebundle-generate.

Quick-flow plan sent to ops:0, proceeding unless objected under tl-worker rules.
ARCH-DRY: reuse the owned subprocess and report prefix fixture, adding only its
bounded stdout suffix writer. ARCH-PURPOSE: both exact max+1 streams retain all
existing refusal, log and cleanup checks. No production state, protocol, caps,
secrets or lifecycle changes; fixture files retain t.TempDir cleanup and child
ownership. Streaming is O(payload bytes), with a fixed 32 KiB block.

Coordination: main still has 96 MiB. TL and pair:3 agree #886 lands independently;
#737 then integrates it, removes its temporary 104 MiB allowance and qualifies
the combined scheduler package. Do not edit or stack on #737. Targeted runs only,
count <=10, one at a time, per dispatch; recorded pre-close remains required.

### 2026-10-25 — fixture implementation and targeted evidence

The existing helper reads only its small prefix and emits padding in checked
32 KiB writes. Both exact max+1 payloads now have child-output length/content
checks; counted writer tests at N, 2N and the exact suffix size pin block growth,
and writer failures/short writes propagate. The original refusal, diagnostics,
partial-report authority, cancellation and inherited-descriptor assertions remain.
No production code or cap changes. Reset fixture-only padding in setup to keep
small cases independent of inherited environment.

Same command before/after:
go test ./cmd/internal/couchcmd -run
'^TestRunArchivePass(Protocol|RefusesInvalidProtocol|CancellationAndInheritedDescriptors)$'
-count=1 -v. Both PASS. Native own-process writes: 16891904 before, 8499200
after; 8392704 bytes saved. New padding/failure/exact-child-output tests PASS
separately, writing 12288 bytes total. The initial added padding test failed to
compile against the missing generator before implementation, then passed.

Reader audit: this file reads only t.TempDir fixture report/invocation/start/log
files, this test executable, owned TCP handshake, and explicit helper environment.
New padding adds no repository or external state reads. Retain disposition
local_or_literal_fixture and repair only this file's inventory. Atlas records
fixture ownership, exact payload contract, retained log writes and measured saving.
TL confirms recorded pre-close may run complete couchcmd through machine queue;
targeted-only applies to ad-hoc runs. Checkpointing before that queued run.

### 2026-10-25 — recorded check queued

PR #613, implementation head dd3570f1aa18dcffc1eed4beb2a387088362103b.
The targeted testdisk TestWriteBudgetVerdict (including over/two-runs-over)
passed. make test-affected-inventory passed after the single audited repair.
sdlc local-check run --pre-close selected affected, then fell back to suite due
to the inventory JSON change. Run lease-3072633283 queues on the machine FIFO;
its complete couchcmd result is pending. No check cancellation or restart.
Close review may overlap the retained queued check under tl-worker rules;
a real final-code check pass remains required before landing.

## Revisions

- 2026-10-25 09:44 PDT — Separate verification deliverables from workflow
  postconditions so close is not asked to tick its own future review. Plan now
  records completed targeted verification and queue admission; full recorded
  pass and review remain explicit requirements before landing. Scope unchanged.

### 2026-10-25 — first close review, qualification pending

Close round 1 returned REWORK with BR-1 (Important, unverifiable,
required-verification-evidence): complete couchcmd at 96 MiB and the recorded
pre-close pass remain outstanding. No code defect or contract revision was
requested. Independent focused protocol/fixture tests passed; reviewer measured
8511488 bytes for its combined selection and confirmed actual-excess refusal.
Its ad-hoc complete-package attempt timed out at four minutes during
TestForcedSwitchSupervisorRejectsChangedAuthority without a final write verdict;
that is incomplete evidence, not an established branch defect.

Accepted BR-1: await retained recorded suite lease-3072633283, preserve both
complete-package budget output and recorded result, then rerun ordinary close
with that evidence to dispose the entire finding family. Do not change code or
weaken Done when to satisfy a pending check. Hub informed. The allowed overlap
saved no close round here because qualification is an explicit acceptance term.

### 2026-10-25 — recorded full suite PASS; BR-1 evidence supplied

Retained pre-close lease-3072633283 completed: suite exit 0 in 14m4s, recorded
sdlc-local/suite on implementation dd3570f1aa18dcffc1eed4beb2a387088362103b.
Complete couchcmd passed in 77.066s with unchanged 96 MiB TestMain enforcement;
complete testdisk passed in 18.396s. All Go, shell, editor and inventory checks
passed, with no flake rerun. [Bounded verification record](../evidence/000886/pre-close.txt)
retains the exact head, command, full-log digest and selected package results.
The full raw log remains in the SDLC-owned local-check store.

BR-1's entire required-verification-evidence class is now covered: both complete
couchcmd within the original cap and successful recorded pre-close. The earlier
review's four-minute ad-hoc timeout is superseded by this managed complete run.
No code changed after the tested implementation; subsequent changes are issue,
review and verification prose only. Request the ordinary close retry; keep the
96 MiB budget and all acceptance requirements intact.
