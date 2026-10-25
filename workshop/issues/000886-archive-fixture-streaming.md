---
id: 000886
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '7b19273f710ed4b677f70e84d0a1ab0b08a535b3' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

Filed unclaimed per ops:0 while pair:3 remains on #737. Measured attribution
and retained failure details are in #737's Log. Primary implementation sites:
cmd/internal/couchcmd/archive_schedule_protocol_test.go's
archiveProtocolFixture, TestArchiveProtocolProcess and
TestRunArchivePassRefusesInvalidProtocol; existing budget constant in
cmd/internal/testdisk/budget.go. Independent work can start from main; no
stacking on #737, and no source edits were made to the archive fixtures here.
