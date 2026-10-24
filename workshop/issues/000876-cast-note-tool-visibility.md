---
id: 000876
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'eda20e8b100043183f822d1618b6f6c1c6af866e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T14:55:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ccce414c", done: "1e787a6b"}
---

# Cast Note tool is missing or undiscoverable in the live tool column

## Problem

During the combined live check of #856, the operator said: “works. there's no annotation tool though, is that by design?” The accepted mock included a Note tool in the column. Relayed by TL delivery35862e4f. The live submission itself worked; Cancel is separately tracked as #875.

Source inspection finds an explicit `▤ Note` button: `web/annotate.js` creates it, `web/cast-tools.js` places it in the note group below Pen/colors, and clicking it enters annotation box/tap mode. Pointer is a separate pointing tool, not the annotation entry point. The tool was not deliberately removed. Native tests already exercise this actual button, but its presence alone does not explain the operator's missing visible control.

## Spec

Diagnose why the dedicated Note action was not visible or discoverable in the operator's actual cast tool column. Check the built/staged asset revision, viewport and column scroll position, group/button hidden state and annotation capability/readiness. The column has vertical overflow and Note follows the palette; below-fold placement is a candidate, not an established cause.

Restore a clear, reachable Note action matching the accepted tool set. Preserve the distinct Pointer, Pen and Note modes and existing capability/page-load guards. If annotation is unavailable, retain an understandable disabled reason rather than inventing an alternate Pointer workflow or weakening admission.

Approved owner/viewer refinement (TL9a388ca5, TLeed8fb98): both Note and Pointer work by default for the trusted cast operator, while every other viewer still follows Annotation OFF unless granted. This supersedes the earlier blanket OFF wording for the owner only. Exact owner proof is the recorder's current registered couchcast isolated CDP execution context in the current top session/frame and stamped loader/document segment, plus live/not-paused/not-ended/no-driver admission and originating generation at final write. A claimed viewer/author/owner name, cookie, header, localhost origin or UI flag is not that proof. Non-owner OFF must show a visible reason and how to ask the cast owner to enable annotation, not merely a dim button.

## Done when

- Root cause is recorded from the actual column state or a faithful disposable native reproduction; no unsupported claim that Note was absent from implementation.
- A live ready cast visibly offers a reachable labelled Note action, including at the reported viewport and with Pen controls expanded; existing Pointer and Pen behavior remains intact.
- Selecting Note permits the existing anchored-box/tap note flow; viewer capability-off, paused and pending-load states preserve their guards and explanation; trusted owner availability follows the approved settled OFF/ON rule.
- A regression pins the observed visibility/reachability failure, including overflow/scroll geometry if that is the cause; operator recheck confirms the tool is easy to find.

- Trusted current operator Note and Pointer operate while the viewer capability stays OFF, including generation0; normal authenticated HTTP viewers and forged operator claims remain refused.
- Generation/context/document retirement and unresolved cleanup still fence new owner writes; prepared capture and authenticated restoration keep owner/surface bindings and exact unknown-outcome recovery.

## Plan

Approved durable design: [owner annotation plan](../plans/000876-note-tool-owner-plan.md). Hub approved; change-code passed with inferred quick flow. Keep separate from #857 and #875.

- [x] Add the private authority binding and migrate every prepared/direct admission consumer, preserving viewer permits and cleanup fences.
- [x] Wire only authenticated current operator ingress and mode delivery; make viewer OFF/read-only recovery visible.
- [x] Pin owner/viewer forgery, cleanup/generation/context races and one native1728×997 Note/Pointer run.
Workflow postconditions: run queued verification and mandatory close; stage only with hub authorization and obtain operator acceptance before landing.

## Log

Filed from the operator's accepted-layout live feedback. No private capture is copied. Production button creation/adoption and native test references were inspected read-only; root cause of the operator's missing visible action remains unconfirmed.


### 2026-10-24 — Cause-first native diagnosis and approved scope

Claimed and started from main56986a072ef70b7ee0d00c317395088b2bb44915, no #857 commits copied; existing combined stage remains untouched. Start-plan catch-up deferred the seeded workflow refresh correctly, with no seeded-file mutation. Queried the supplied local recording and inspected the older screenshot read-only; private capture not copied. A single disposable native probe at1728×997 proves Note is present/hit-testable at1664,284.75,48×36, within a216.75px-high column with no overflow, but disabled/opacity0.4 by default Annotation OFF. Existing enable changes availability to enabled/opacity1 without layout changes. PASS2.342s, log SHA256fc79f964d1284100bd034ee2a0119701775dd7d00b7d1aa2f27bcab050053401; probe source and log retained in canonical plan.side evidence, temporary Go test removed. This reproduces a default-availability defect; the exact old private UI state is not in the recording.

TL9a388ca5 approves owner-default semantics and visible viewer recovery; TLeed8fb98 includes both Note and Pointer. Read-only authority audit confirms exact CDP context/frame/document proof and all prepared-note permit consumers must be handled, not just UI or AnnotationAdmission.Check. The plan uses one comparable private authority binding, preserves viewer OFF and existing cleanup/generation fencing, and creates no new wire/persisted state. Plan sent for hub approval; no production code changed. #875 still waits for #857 landing.


### 2026-10-24 — Hub plan approval

TL4b0ceec1 approves the durable plan in full and directs change-code. No production implementation before that gate.

### 2026-10-24 — Implementation and focused qualification

TL0edcd963 directs keeping inferred quick flow; no --flow pin or estimate is added. Close upgrades its review automatically if the diff leaves the shell. Implemented one comparable private authority binding through Board checks, direct viewer surfaces and prepared-note prepare/reserve/restore. Owner authority accepts settled OFF/ON at the original generation and never becomes ordinary viewer authority; console sketch policy is preserved. Recorder mints only after actual CDP identity validation on its actor; requests retain their originating generation, and final ordered writer fences remain. Chrome exact-context permission delivery and mode selection use that owner admission. Viewer OFF/read-only recovery is visible in Notes. No wire, registry, lifecycle state or timer added.

RED observed owner Board Note/Pointer and recorder owner preparation at generation0 returning ErrDisabled; Chrome worker emitted hook(false,"0"); viewer guidance lacked recovery. Focused Board/recorder/Chrome tests now pass and browser tests72/72 pass. Native TestOperatorOwnerLiveAnnotationOff PASS4.36s (package4.611s): mounted owner Note native hit/box/capture/save and Pointer, authenticated viewer and forged owner HTTP403, unchanged global OFF,1728x997 and short viewport with expanded Pen, settled ON/OFF. An initial native attempt failed only because the fixture omitted the existing required sketch label on tool handoff; completing the actual label flow passes with no production change. Exact historical private UI state remains unproven; combined #857/#866/#844 stage is untouched. Further race/lint/inventory/recorded verification and mandatory close remain outstanding; #876 still needs hub-sequenced live recheck before landing.

## Revisions

### 2026-10-24 — Separate operator authority from viewer permission

Reason: native diagnosis shows a real Note/Pointer default-availability mismatch, not overflow at the reported size. TL9a388ca5/TLeed8fb98 replace the original all-surfaces OFF guard with authenticated owner Note+Pointer availability while viewers remain grant-gated. Add exact owner proof, cleanup/preparation fences and visible viewer recovery to Spec/Done when; durable full-flow plan precedes implementation.

### 2026-10-24 — Keep inferred quick flow

TL0edcd963 confirms --flow is operator-owned and directs retaining quick. The approved implementation/design remains unchanged; the close boundary upgrades review automatically when the diff exceeds the shell. No full-flow pin or estimate is claimed.

### 2026-10-24 — Bidirectional role binding and capture recovery

Real signed capacity-fallback testing exposed that a valid viewer admission could restore an operator-surface token when handed the exact authenticated owner tuple. Added the reverse-role regression, observed RED, and made the shared surface checker require the matching viewer/owner kind in both directions; trusted OperatorSession dispatch also requires owner kind. Existing operator fixtures now explicitly request owner authority rather than borrowing viewer permission. Native owner qualification already passed on the owner path; service signed unavailable restore/save and unknown-ACK reconciliation now pass, exactly one write and global viewer OFF unchanged. Race qualification is being rerun on this stricter implementation before committing.

### 2026-10-24 — Ready for recorded check

Final focused race checks PASS in Board1.326s, recorder1.739s, notedraft3.521s and Chrome2.074s. Exact-context counted worker test PASS at4/8 contexts:3C sends and receipt checks across OFF0/ON1/OFF2; repeated unchanged refreshes send none. Existing native TestOperatorLiveAnnotation also PASS4.93s/package5.170s after updating only the superseded owner-OFF assertions. Catalog PASS6.305s; inventory PASS after individually auditing prepared_note_test.go (local builder/reference-writer fixture, no repository artifact reads) and recording only that stale certificate. New test files are staged before inventory. Further final lint/inventory precede the queued pre-close record; operator acceptance is still pending and pair:0 remains untouched.

Final staged-file catalog PASS7.227s and affected-reader inventory PASS, with all new tests tracked. Submission-focused checks PASS1.878s. Implementation checkpoint is ready to push, then queue pre-close on that exact head and run the mandatory close review. No merge or stage is authorized before operator acceptance.
