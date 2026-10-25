---
id: 000876
status: codecomplete
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '73eabea9478d6199f32b69958b91a8ed2c142d30' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T14:55:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: N/A
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
- 2026-10-24: closed — BR-1 all Core concepts paths/statuses reconciled with delivered diff, Revision appended. BR-2 exact reviewed95f06244a39371e5d3ec8f68183ec22e4b6a2dcd native host PASS4.46s/package4.709s exit0, Go1.27.3 darwin/arm64, command/env/output pinned in canonical plan.side/evidence, formal dispute recorded. Runtime source unchanged from queued lease-1020414290; inventory PASS before this close. Prior focused race,72 JS tests, native owner and existing operator, signed recovery exactly-once and counted4/8 checks PASS. TL407daf9b authorizes no-actual for exact initial refusal: measurement branch proof: measurement snapshot does not serialize: json: error calling MarshalJSON for type *gatestate.MeasurementInputs: measurement exceeds 65536 bytes; no window printed. Operator recheck remains outstanding before landing.; review verdict: SHIP
- 2026-10-24: flow upgraded quick → full — 121 added lines in code files (limit 100); an earlier round of this close already ran the full review

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

### 2026-10-24 — Retained verification and authorized measurement refusal

PR598 pushed source d130053e234d57b1788d1a5c98f0df027bcf261e. Recorded pre-close resolved to affected, then correctly selected the full-suite fallback because the individually audited prepared-note reader certificate changed. Retained terminal14305 / run lease-1020414290; worktree /Volumes/TestScratch/local-checks/sdlc-local-101265153; inflight log .git/sdlc/test-stats/inflight/56a07f9fd13dee58a4f13ef7bf2211ec/run.log. It is queued FIFO7 at the last returned result, not yet run. Do not duplicate/cancel it; next step is inspect that exact runner's result and fix any attributable failure.

Initial close terminal68867 exited1 before review with the exact refusal: `can't auto-compute (measurement branch proof: measurement snapshot does not serialize: json: error calling MarshalJSON for type *gatestate.MeasurementInputs: measurement exceeds 65536 bytes)`. No measurement window was printed. TL407daf9b grants standing --no-actual for this exact64KiB refusal until ariadne#593; use that precise waiver, not typed hours. Rerun inventory, then close with --no-actual and this refusal in --verified. The hub received the operator-stage request in receipt2102edc0-adb8-4c54-bd61-0c1cd6a0caae, submitted. #857 combined stage still untouched; landing remains gated on recorded verification, close and operator acceptance.

### 2026-10-24 — Close round1 REWORK; both findings answered

Full boundary review auto-upgraded from quick and returned REWORK: BR-1 Critical Core concepts file/status contradictions, BR-2 Important native host unverifiable (reviewer Chrome GPU unusable before page startup). No production defect was reported. BR-1 tables are reconciled across all rows: authority entity in board/annotation.go, actual operator_owner_live_test.go, CSS/policy and notedraft adapters/service unchanged. BR-2 exact reviewer-head95f06244a39371e5d3ec8f68183ec22e4b6a2dcd host run PASS4.46s/package4.709s, WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run ^TestOperatorOwnerLiveAnnotationOff$ -count=1 -timeout=75s -v, Go1.27.3 darwin/arm64, native disposable private profile. Full output and metadata retained under canonical plan.side/evidence/pinned-owner-native*. No runtime/test code changes since queued source d130053e. Next: record BR-2 counter-measurement, inventory and full re-close; retain the queued suite.

### 2026-10-24 — Retained suite PASS after restart

Retained terminal14305 / lease-1020414290 completed exit0: full suite PASS14m21s on source d130053e234d57b1788d1a5c98f0df027bcf261e, recorded sdlc-local/suite. Saved log .git/sdlc/local-checks/d130053e234d57b1788d1a5c98f0df027bcf261e/suite.log SHA2568b008ff5bbb4e31bc9dec1f14b6b6ddd72c58cf776b429ad851030422f5c11bb. Git diff against closed head c4393e6d0ff52244075eddb64bf1dbc227f2645e is empty for cmd/, tests/, README.md and atlas/: later changes are workshop review/evidence only. No check or merge runner remains active for876.

Full close round2 SHIP, BR-1 addressed and BR-2 withdrawn, zero open findings. The pending operator stage request was resent after Couch restart and receipt d6bd4857-9fd8-41bb-94a7-1b0101a165fa is submitted. Next step: wait for hub stage authorization/operator recheck; stage only as sequenced, retain the existing combined857/866/844 build until instructed; enter merge only after live acceptance. No new merge started. TL3ccec485 ends the64KiB no-actual waiver now that593 landed; any future close/re-close measures normally. Earlier authorized historical close evidence remains unchanged;128-source waiver lasts until603.

### 2026-10-24 — New combined stage and independent warm qualification PASS

TL995cbee2/a6fbf96b authorize rebuilding only after866 green, then personally verifying current-main-created warm sessions before operator handoff. Previous857/866/844 stage was abandoned after warm startup refusal and its lease cleared via --clear --abandoned857; no857/844 landing was attempted. Pair:5 reports866 corrected source031706970222f2f3d7dbdc34d68a8747321ed2bd green (full suite14m2s and closeSHIP), and pair:1 confirms844 source unchanged and checklist current.

Fresh main pin dc3031079bddd3c160cd9481fdef94f1ff29c83b plus exact tested8576615515753ca49f840209747317995d81913d24b,866031706970222f2f3d7dbdc34d68a8747321ed2bd,844f7f4ec084657b6f4bfcd669254f40f694a2b835a and876d130053e234d57b1788d1a5c98f0df027bcf261e form staging-only local ref000857-stage-857-866-844-876-20261024b at9bfd7ab51cef5f9d563b19dea087c9ef084c19c9. All pins are Git-verified ancestors. Conflicts preserve main warning routing, add both source-inventory sets, preserve same-provider optional closure/module metadata and adapt lint callbacks to the current API.876 owner reason retains857 pending-load priority and Event retains load intent while dropping old viewer admission capture. Focused Go/browser/catalog/artifact classification/provider(3 tests)/reader inventory checks pass. No staging branch is pushed.

sdlc stage --build exit0: pair:0 detached at9bfd7ab51cef5f9d563b19dea087c9ef084c19c9, clean; all5 binaries verified exact revision, modified=false. The one retained stage runner waited on879 close's repo lock and then completed normally.

Independent TestCombinedStageMainWarmUpgrade PASS146.58s/package146.745s, exit0, using those exact staged Couch/Pair/helper binaries (build-info revision + SHA256 checked, no substitute candidate rebuild). Pinned current main created Claude and Codex hub+worker sessions in two disposable testcouch fleets; staged startup adoption, public listing/resume and exact-tag Continue/native list-clients all pass. All4 original bindings, fixture PID/generation/nativeID/transcript and socket inode remain unchanged; readiness markers naturally absent. Both fleet roots removed; all owned generations absent/direct child and IO joined. The main-only scratch worktree was removed after use. Initial setup stopped before any fleet because fresh main lacked ignored generated runtime assets; generating them from pinned main fixed setup, and that failed setup log/metadata is preserved separately.

Evidence: plan.side/evidence/combined-restage/ contains raw warm-upgrade.log SHA2567f87f8c973bcefd6cbf4950d01cadb5645c43d34c06a6811c4f19442c163340c; warm-metadata.json SHA256b7c79232f43a58d7b8ec44385f856d87b27aa079e1bbb79fb8fed836cffc2d46,4 actor assertions/cleanup summary, all5 binary provenance, retained driver.go.txt/overlay/launcher, focused checks, and combined-operator-check.md. Old ~/couch-profile-871 exists and is preserved; fresh ~/couch-profile-871-restage-20261024 was verified absent after qualification and is the new restart line. Keep the staged build. Next: deliver revision/evidence/checklist to hub and wait for live verdict; no landing before acceptance.876's branch still contains no runtime changes after its passed suite and SHIP.

### 2026-10-24 — Operator acceptance; landing authorized

TL db1a5886 reports evening live check on combined stage9bfd7ab5: #876 PASS for owner Note and Pointer, viewer refusal and Note reachability; land it. #857 remains blocked on its separate live findings. Keep pair:0 and the combined stage unchanged until the hub sequences the next stage. Next: sdlc merge --yes for #876, report its actual outcome, then diagnose #857.
