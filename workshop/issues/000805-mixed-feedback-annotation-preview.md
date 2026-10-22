---
id: 000805
status: open
deps: [753]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '69b33c95e15e5dfc7b123a55f240977e09d8f012' # card fields mirrored from issue-cards; edit via sdlc
---

# Show annotation notes and saved crops in the mixed feedback preview

## Problem

The operator uses bounding-box annotations and freehand sketches together as LLM feedback. #753 previews sketches but does not show existing annotation notes/crops in the acknowledged pending set. The pointer is a separate ephemeral presentation tool and must never join feedback.

## Spec

- Extend the pending feedback panel to show bounding-box annotations beside sketch cards, with combined counts, saved note text and saved crop thumbnails. One operator submission carries both typed kinds under the existing pinned-target prepare/begin/ack/resolve transaction.
- Reuse each annotation's saved draw-time PNG, strictly within its original box intersected with the captured viewport; do not recapture current page pixels or use the generic padded structural crop as the saved asset. Preserve original URL/elements/author and revision. Legacy/offscreen/unavailable captures must say unavailable rather than fabricate pixels.
- Candidate invalidation and acknowledgment must include note identity/revision/capture identity and note state changes. Freeze exactly the displayed notes atomically; stale edits/resolution/reopening invalidate the preview. Keep conditional resolution after confirmed delivery and existing unknown-outcome recovery.
- Add bounded read-only access to private saved capture assets for preview. Previewing never claims/releases/deletes capture ownership and never exposes storage paths or capture capabilities to the page.
- Keep pointer state and pointer pixels out of both typed feedback lists and images. Match page/view eligibility to the approved #753 shared-image/feedback batch policy; same-screen versus retained off-view items must be made explicit during design.
- This is a full-flow follow-up after #753; provisional size6–9 focused hours before its own settled plan/calibrated estimate. Root transaction and original notes/crop format remain shared; avoid a second submission route or delivery registry.

## Done when

- Mixed and annotation-only pending sets visibly show the exact notes, independent saved crops and counts that the chord will submit; sketch thumbnails remain alongside them and pointer never appears.
- Two overlapping annotations produce distinct box-contained images; scrolling/DPR/partial clipping and legacy/unavailable capture cases remain explicit and correct.
- Note add/edit/resolve/reopen and late/out-of-order acknowledgments cannot freeze or resolve an undisplayed/replaced revision; failed/unknown delivery retains assets and recovery identity.
- Preview image reads are bounded, private and do not change source ownership; tests use the production capture/store seams and stateful fixtures, plus native Chrome/operator acceptance.

## Plan

- [ ] Claim/start-plan after #753 lands; settle page/view eligibility and extend the mixed preview state/freeze contract.
- [ ] Implement and verify using the normal full-flow gates, including artifact/reader inventory, recorded checks and actual operator mixed-feedback smoke.

## Log

- 2026-10-22: filed from authenticated operator feedback via hub8e034f38-b5a2-44a0-843b-6058623cf685 and framing4d04fb04-3599-4779-a843-8c505276daa8. Read-only audit: castDraftCapture.Capture already uses Box=Region=p.Region and visibleCrop clips only to viewport; major work is mixed preview candidates, exact displayed note freeze and read-only asset transport. No implementation or formal estimate authorized yet.
