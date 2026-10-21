---
type: continuation
slug: cast-sketch-recording-start
agent: codex
created: 2026-10-20T21:42:28
branch: 000751-cast-sketch-recording
worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
issues: [000750, 000751]
---

# Continuation: cast-sketch-recording-start

## NEXT ACTION

Run `sdlc state`, read #751 and `workshop/plans/000751-cast-sketch-recording-plan.md`, then run `sdlc change-code --issue 751 --flow full --worktree=no`. The TL explicitly authorized this slice after plan-quality with no new operator round unless product design changes. Fix gate findings and derive estimate only after plan-quality clears. Then execute the plan with TDD and bounded subagents; finish and land #751, report its outcome to ops:0, then continue the assigned lane. Do not stop at the plan.

## State of play

#750 design is DONE and LANDED as PR #501 https://github.com/xianxu/pair/pull/501. Integrated full suite passed in 12m14s, final CI green, sdlc merge exit0. Its issue/plan are archived. Hub outcome receipt 02336554-7fa3-4fcc-ab17-368a1a29f79f was submitted. TL owns ops project and was told to tick pair#750 / PR501.

#751 is CLAIMED/WORKING in pair:9 on branch `000751-cast-sketch-recording`, created from fresh main after #750 landed. No implementation yet; change-code not run. Plan committed in c0e0622a4, integration append dbd1b721d (obtain full SHA from Git if reporting). `start-plan` refreshed ariadne and tools through weave. No long-running tools or suites remain. Last working tree tracked state clean. This checkpoint writer adds/pushes the continuation; don't confuse that prose commit with code.

## Thread arc & user model

Operator wants sketches for echo's design workflow and explicitly prioritized cast-sketch. Initial TL local-draft proposal was overridden: strokes stream immediately. Operator approved the detailed #750 design, changing drawing privilege to follow driving: console operator or currently granted drive-link driver, one at a time; support console Chrome and pressure pen including iPad Pencil. All other product choices were approved. The TL asks for autonomous delivery, no further design questions: bring only genuine changes to ops:0. #751 numeric reserve correction is already approved and must not be asked again.

The dedicated lane is #751 → #752 → #753 → #754 without unrelated work; #756 joins after #752 (we acknowledged doing it there before export/acceptance). #756 records mock version/screen scoping for sketches AND notes; review its spec at that point. #755 mock serve is another slot's work. TL owns project and landing order; follow sdlc merge admission.

## Artifact map

Read `workshop/issues/000751-cast-sketch-recording.md`, then its new durable implementation plan. The plan includes pure/integration tables, state/recovery mapping, concrete TDD tasks, all architectural lenses and approved numeric Revision. Parent #750 is the archived design in workshop/history/plans (resolve with sdlc rather than read history broadly); it is explicitly the source for this issue.

Code entry points are recording/{record.go,codec.go,reduce.go,control.go,catchup.go,contract.go,files.go}, recorder/mirror.go, board/writer.go, web/{model.js,replay.js,annotation-anchor.js}, server.go, plus existing strictjson helpers. Shared corpora live tests/webcast and replay_corpus_test.go. `/tmp/cast-sketch-751-api.md` and `/tmp/cast-sketch-751-plan-draft.md` are optional local scratch; their material decisions were appended to the durable plan. No code depends on these scratch files.

Fleet/project center is ops at `/Users/xianxu/workspace/ops`, hub ops:0; use `couch --send-to hub`, not direct project edits. Read fresh tl-worker --help per item; already read for #751 before start-plan. Slot description published as pair#751 cast-sketch recording.

## Decisions & dead ends

Required label on leaving tool, held drawer source scene, current-sketch Undo/Redo, colour fixed per sketch, one anchor per stroke, no ownership takeover on expired link, operator-only recording/CLI export handoff all approved. UI/transport/source-snapshot preparation belongs #752; local CLI historical image export #753; device/echo acceptance #754. #751 must complete wire/model/pressure/catch-up/ordering/budget, no UI claims.

Terminal reserve: Go JSON encoding a valid1024-byte `<` label is6146 bytes BEFORE envelope, so original4KiB is wrong. TL explicit approval: derive worst-case terminal record sequence (label, envelope, stroke-end or interrupt), round up >=8KiB, keep total32MiB, adversarial-label test. Conservative label+end+interrupt with maximal bounded IDs may need16KiB; compute actual Encode lengths. No new operator approval needed. IDs bounded128UTF8 bytes, digesthex64, owner existinghex64/operator shape are implementation recommendations in plan.

Existing affected-check graph failure pair#738/ariadne#517 required full-suite fallback for #750. Generating runtime bundle locally did not fix separate detached check worktree. Do not bypass or fix unrelated issue; use full local-check suite if still present. #750's merge took a long time queueing behind #748, then required exact integrated-head suite. Do not relaunch merge or kill another slot's suite; own checks canceled only with sdlc local-check cancel.

## Lessons learned

Preserve source exactness with strict timestamp fences before AND after sketch-begin, including samefile/equal-clock cases; both producers use one helper. Catch-up keeps causal historical ctl events for sketch interruptions; bare historical segments mean anchor replay cannot require removed old DOM snapshots. Duplicate reducer no-op alone is insufficient: producer must suppress duplicate sink writes. Strictjson.EncodeObject preserves raw nested values; passing output through json.Marshal can compact them. Existing KnownFields doesn't flatten Header; use focused owned key sets.

Hot paths append only new point chunks, Undo/Redo constant stack changes; snapshot flattening is off hot path. Validate all transition failures before mutating State; ctl handoff interruption must be atomic with validated control. Sink failure after Apply is unknown persistence outcome, not guaranteed rollback. Browser asset registry needs new module in both embed and assetmap; operator module assembler accepts named function/const exports, not export class/default.

Commit trailer for this session was Agent: codex/gpt-6.1-sol/medium plus Co-Authored-By: GPT-6.1 Sol <noreply@openai.com>; rerun sdlc agent-id after restart. Gates own mandatory review; do not dispatch redundant review at close. Full run via sdlc local-check, and report outcomes once receipt submitted.
