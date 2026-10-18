---
id: 000672
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
severity: critical
collaboration-mode: collaboration
card_mirror: '2752ba5c9fda4c40be38a5af1c49feb7087cfcc0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T11:27:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "a07c8597", done: "105c1e00"}
---

# Accept native Zellij client replies without blocking launches

## Problem

Operator launches/relaunches fail on the new build containing #647. Observed by the hub: exact session name 📁1-292 is rejected because ProbeSessionName's successful list-clients reply fails strict decoding with invalid Zellij clients header. Installed Zellij is0.45.1. This blocks new slots and takes priority over #665 M2.

## Spec

Capture the real list-clients bytes including header and the exact probe context. Fix the confirmed strict-decoder compatibility defect while preserving fail-closed handling of malformed/wrong-kind/duplicate replies. No fallback to loose parsing. Use the captured native output as a permanent fixture. No operator session mutations, local Zellij build or unrelated #665 mechanism changes.

## Done when

- Exact native reply/header causing refusal is retained and the cause reported to the hub.
- Captured format is accepted by the strict contract; malformed/wrong-kind/duplicate identities still refuse, with deterministic regression red before green.
- Relevant consumers normal/race and repository checks pass; required recorded suite and quick close review pass before landing.
- Land through the line and report the PR/merge so the hub can rebuild pair:0.

## Plan

- Capture and classify the native successful reply and parser mismatch.
- Add captured fixture plus adversarial controls, reproduce red, implement the smallest strict compatibility repair.
- Qualify, review, recorded suite, land; return to checkpointed #665 M2 afterward.

## Log

2026-10-18: urgent dispatch hubda698b0e. #665 M2 clean checkpoint da22f58a48e89393717f617a30d51f867997a884; new work begins from fresh main, not stacked. Disposable nonexistent-session control on installed0.45.1 emits exit1, empty stdout, stderr There is no active session!; this is already an accepted diagnostic and does not reproduce the reported successful malformed-header path. Keep diagnosis open until real bytes are captured.

### 2026-10-18 — Captured cause, red/green and native qualification

Exact read-only probe for requested absent 📁1-292 returned exit0, a colored complete session listing on stdout, and only the requested-name not-found heading on stderr. The clients header itself is unchanged; private owned-session control emits the standard three-column header. Native source zellij/src/commands.rs:438–442 calls list_sessions(false,false,true); zellij-utils/src/sessions.rs:367–401 exits with its own0 before the intended outer exit1. This is the confirmed production launch failure, not evidence that list-clients changed format. Raw capture is preserved verbatim under cmd/internal/launcher/testdata/zellij-0.45.1-name-absence/ with the header positive control.

Regression RED reproduced invalid Zellij clients header on the captured absence and exposed a contradictory clients-table/not-found pair being accepted. GREEN uses the exact complete stderr heading only on successful collected replies, then strictly validates the full colored session listing through existing sessionListingStates after removing only native SGR metadata. The requested identity must be absent; malformed/wrong-kind/truncated/unknown-escape/duplicate/contradictory/failed/cancelled replies remain unknown. ParseClients is unchanged. Final focused normal PASS0.194s/race PASS1.232s; /tmp/pair672-final-focused{,-race}.log. Installed0.45.1 parse-only validator and production name acceptor conformance PASS6.808s, /tmp/pair672-native.log; read-only probes only, no operator mutations. Full affected normal/race and repository catalog checks are running; no full-suite result claimed. Quick close review may overlap the required recorded suite; final PASS still gates landing.

### 2026-10-18 — BR-1 host evidence and hash correction

Round1 REWORK has one Important unverifiable finding, no code defect: review TMPDIR exceeds Unix socket limits and its clone lacks main authority. Re-ran exact code5e299f51362df4c72591502cca46971ec7df5a80 on the registered checkout with TMPDIR=/tmp, no source/deadline changes: full launcher/reply/session normal PASS61.335s/0.132s/0.042s, full race PASS132.609s/1.194s/1.066s, authoritative repolint PASS7.513s. Commands and full log hashes are in workshop/plans/000672-host-qualification-evidence.md.

Correction: the dispute command accidentally included an incorrect evidence-file hash. The actual SHA256, independently printed by shasum, is 8a2add92c7481d50a218dfa4d10e303c6b6f0484bde1bdf9470fa239831d88aa. This supersedes the erroneous 4306e429... value in that dispute's prose; the owned ledger is not hand-edited. The evidence file and its per-log hashes have not changed. Round2 must answer BR-1 using this verified file and the measured results. Required suite remains queued on the unchanged code head; no passing suite claimed.

## Revisions

### 2026-10-18 — R1: exact native absence classification, clients parser unchanged

Hubf8f584e5 approves the captured cause and narrow repair. The name probe's exit0 stdout is an ANSI session listing, not a clients table; stderr is exactly the requested-name not-found heading. Native commands.rs calls list_sessions in the multi-session not-found branch; sessions.rs:list_sessions exits0 itself, so control never reaches the later exit1. Keep ParseClients unchanged. Add a pure sessionNameAbsentListing(session,stdout,stderr): exact complete stderr heading, nonempty stdout, remove only four native SGR sequences (green name/magenta age/red EXITED/reset), reuse sessionListingStates strict full-row/UTF8/duration/suffix/duplicate checks, and require requested identity absent. Call only after successful collected process with live caller context, before clients parsing. Existing pure-exit1 absent diagnostics and exit2 validator behavior remain. No retries/new deadlines/effects or universal absence claim; this proves name length acceptance only (ARCH-PURE, ARCH-DRY).

Captured raw fixture /tmp/pair672-operator-probe.stdout and.stderr (read-only query of absent dispatched name); private controlled-session header remains CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND. Store exact capture as testdata with provenance. RED: actual name-probe successful-reply seam must accept that fixture; parser independently still refuses it as wrong-kind. Matrix: absent/foreign target heading, requested identity present, empty/truncated listing, client-table/JSON/unrelated stdout, malformed metadata/unknown escapes/duplicates/invalidUTF8, extra stderr, nonzero/joined/cancelled collection outcomes. Existing generator diagnostic matrix and fake-process contracts remain. Targeted launcher/reply normal+race, catalog, then recorded branch suite and close review before landing. Healthy path work is one O(bytes+rows) normalization/strict scan on this exceptional probe reply, no new query.
