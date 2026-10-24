---
id: 000838
status: working
deps: [ariadne#575]
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours: 3.73
collaboration-mode: collaboration
card_mirror: '4bb76ecd4955ad124c2821af2d8ec212badb2123' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T14:02:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Couch: single collaborative mode and the @@chat@@ carve-out; legacy mode keys mapped or ignored

## Problem

## Spec

Implements ariadne's single-collaborative-mode spec (filed alongside, from the operator's 10-23 pivot) in Couch,
simplifying pair#823's per-section mode machinery:
- Rounds follow the one collaborative mode; presentation is always highlight for agent edits.
- `@@chat@@` above a heading marks a chat section (numbered inputs always get side-doc replies); Couch recognises it,
  protects the tag as operator-owned, and enforces the chat rule there.
- Legacy keys: `collaboration-mode: chat` (document-wide or per-section map) maps to the chat carve-out; the
  presentation and processing keys and instructions are ignored with one deprecation notice per round; nothing new
  writes them. pair#823's migration command converts old maps to `@@chat@@` tags (operator-run).
- Remove the per-section mode compiler paths that only served the retired keys, keeping the legacy reads above.

## Done when

- A round on a document with a `@@chat@@` section follows the chat rule there and the collaborative rule elsewhere
  (test against the spec's fixtures).
- Documents with legacy keys resolve as mapped, with one notice and no refusal (test; plus a read-only audit of the
  real ops and brain session docs like pair#823's).
- Brain's chat document keeps working unchanged.

## Plan

Draft: [single collaborative mode and chat plan](../plans/000838-collaborative-chat-plan.md).
One atomic full-flow implementation, one issue-close review (no Mx tags).

- [x] Approve design and verify ariadne#575 landing; qualify final scanner/notices against ariadne#576.
- [x] Pin shared fixtures; simplify the editorial compiler/guards around sidedoc.ChatSections.
- [x] Wire new highlight rounds, legacy notices and lossless operator map-to-tag migration.
- [x] Prove public seams, old rounds, real-doc read-only audit and disposable native acceptance; update atlas/help.
- [ ] Inventory, recorded affected check, close review, operator acceptance as agreed with TL, then land.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Calibration source reports stale; figures
are provisional ship wall-clock hours, not machine queue elapsed time.

The design is settled by the approved plan and shared scanner. Library check:
Goldmark, sidedoc and the stateful session-sync/testcouch rigs already exist;
there is no greenfield parser or transport. The three smaller Go units are
compiler/byte adapter, edit-guard projection, and lossless migration. The
cross-cutting refactor is the distinct wiring/retirement across round, read,
CLI and source inventory consumers. Native API discovery budgets Claude and
Codex separately, bounded synthetic runs through the existing harness.

For design choose v2 raw1.0h for issue/spec and skill, raw0.3h per smaller Go
unit, raw1.0h for cross-cutting refactor, raw0.2h for atlas and raw0.1h for the
one close review; apply x0.2 thorough-spec discount. Implementation uses the
upper v2 bounds (0.2h issue, 0.5h per Go/refactor/skill/review, 0.2h atlas,
0.6h per native API discovery), scaled once by v3.1 x0.4. Familiarity1.5:
Couch is familiar but this bounded Goldmark/editorial policy and native fixture
integration is new in this slot. Thorough plan gives +15% design buffer.
No separate cross-repo implementation overhead: upstream fixes are owned by
ariadne:2, and no vendor propagation or peer source edits are planned here.

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: issue-spec design=0.20 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.20 impl=0.20
item: skill-or-dispatcher design=0.20 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.02 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
item: real-api-discovery design=0.00 impl=0.24
design-buffer: 0.15
total: 3.73
```

Design0.84 x1.15 + implementation1.84 x1.5 =3.726h, rounded3.73h.
The operator acceptance is one short stage check already approved in principle;
Qoder conformance is conditional on credits, with no speculative extra build.

## Log

### 2026-10-23

### 2026-10-23 — claimed and draft plan

#811 completed before claim: its merge runner exited0, tracker/archive and
main reachability verified, final proof Log published. Claimed #838 in pair:3,
then start-plan created its fresh-main branch and refreshed the ordinary
Ariadne dependency. No code, live store or :0 changes.

Read the operator's ops side doc and ariadne#575's issue and committed candidate
6ae80e4e69aacb72d8745e479791ada7c4454c17. Its shared sidedoc.ChatSections
already owns tag/legacy grammar and notices (ARCH-DRY). Current Pair source
still has independent three-axis resolution, runtime briefs and mode filling;
the durable plan removes those paths while preserving transaction/marker/chat
ownership and old saved-round decoding.

Cost inspection found shared #575 prefix counting and heading-end scans can
cost O(B*H+H²). Authenticated ariadne:2 receipt
c8bda0b0-3e7f-43f4-9301-6a5521d89dc6 confirms it, with no current bound/test;
the owner takes the linear scanner/count tests and two notice corrections in
ariadne#576 and is informing the TL. Plan proposes #576 before final fixture
pin/performance qualification/close, with no Pair scanner fork (ARCH-CONSTRAINTS).

Draft also names a migration compatibility trap: new tags accept numbered
items, old chat accepts bullets and whole-document/preamble asks. Proposed
operator transform refuses conversions that alter the eligible-item projection,
leaving legacy reads fully functional and source untouched. This is explicit
in the approval request, not a silently chosen product refusal.

Next: commit/push this design and send it to ops:0; wait for plan approval and
#575 landing before change-code, and #576 for final qualification. No estimate
yet and no implementation started.

### 2026-10-23 14:08 — approved; waiting for upstream landing

Design commit f6ac0a46af6ffb9a75a28c1c6d520195c8eb426e is pushed. Plan report
d8885e44-dd35-4de8-af81-86f68720e9e6 is submitted. Authenticated ops:0
a7ad7cf6-257b-419f-b538-e40f2b0a457b approves the design: migration refusal
names the affected section, no renumbering; code after ariadne#575 lands;
final fixture pin/performance/close after ariadne#576; stage check is the
project live acceptance. Approval is preserved in the plan's Revisions.

Fresh tracker observation still reports ariadne#575 codecomplete/not-landed
on candidate6ae80e4e69aacb72d8745e479791ada7c4454c17. Exact next step:
verify its landing on main, refresh the ordinary Ariadne dependency, then run
sdlc change-code --issue 838 --flow full and satisfy plan-quality before
deriving the estimate or writing code. No code/change-code or tests yet;
only design changed and git diff --check passed. No own background check,
merge runner or stage remains.

### 2026-10-23 — #575 landed; first plan gate blocked

Authenticated ariadne:2 receipt64863819-3103-415b-a904-e29579a5cebf
reports #575/PR418 landed; GitHub reports MERGED at
df49c55b9c768a37dfbf4455d13f3ff63543539b. A normal weave refresh correctly
refused the issue branch before updates; `weave refresh --substrate --planning`
then advanced the ordinary private substrate to
c39d1899c04e1176d5bdce47f9f06848dd17ce2c, containing that merge (Git
ancestor check EXIT0), preserving Pair HEAD. The stale seeded CI workflow was
deferred without tracked source changes.

change-code --issue838 --flow full ran its mandatory first plan review and
exited1 before estimates/code: PQ-1 shared-authority-preserves-provenance and
PQ-2 function-scoped-test-strategy, both Important. PQ-2 is revised into named
function/adversarial-input/mechanical-guard strategies. PQ-1 was verified:
Source=tag loses independent legacy grammar; whole-document legacy also
suppresses valid inner tags needed for source ownership. Upstream #576 owns
the correction, with Legacy and Tags APIs and their independent invariants
now explicit in the plan. The item-parser extraction's final signature is
still forthcoming from its owner. No fallback parser or candidate dependency.

TL ccfe62c7-130f-4095-944f-e13cfa85b3d8 confirmed the upstream fix and normal
pinned module update. `go get` updated go.mod/go.sum to the verified landed
#575 pseudo-version v0.0.0-20261023212235-df49c55b9c76 (no replace).
Existing focused catalog/contract tests passed0.155s with that dependency;
this is module compatibility evidence, not new feature acceptance.

Next: commit the revised design/ledger and module preparation, rerun the plan
gate against the explicit #576 API, then derive estimates only on acceptance.
#576's landed provenance API is necessary before the core adapters can compile;
final fixture pin/performance/close continue to wait for it. No implementation
or own long-running check is active.

### 2026-10-23 14:36 — gate accepted; RED baseline; waiting for #576 API

Plan round2 CLEAN disposed PQ-1/PQ-2, no new findings. Derived estimate3.73h
from the named v3.1 source after acceptance; card setter and deterministic
reconciliation passed. Estimate review INFO: explicitly account for fixture,
old-round/public-seam verification and audit/stage preparation within the
existing items; implementation allowances may be optimistic. Retain those
risks for actual comparison, not a fabricated precision claim. Issue/spec
includes completed authoring, cross-cutting work includes public-seam/old-codec
tests and inventory preparation, atlas/skill includes source fixture guidance
and real-doc read-only audit; provider discovery covers native qualification
and its shared harness/stage preparation. Queue and operator idle waits are
excluded from focused work.

Session86711 exited0 and checkpointed accepted full-flow design at
4ed7f036fac126928237ad92e9f4e89a17483710. I misread its return status and
started a redundant confirmation86711->89599: plan passed through cached and
only estimate review repeated INFO; session89599 also exited0. No bypass or
code mutation. Do not repeat it merely to reconfirm success.

Added collaborative_test.go against the real existing API. Focused run EXIT1
with expected RED behavior: tag item resolves non-chat, malformed/unknown or
duplicate retired values refuse, and deleting a real tag (including under
whole-document legacy chat) incorrectly succeeds. This is deliberate TDD
evidence, not a recorded passing suite. Existing catalog/source-pin tests
on the #575 dependency passed0.155s.

Upstream now exposes Legacy, Tags and shared ChatItems on #576's unlanded
branch. These APIs are required for the core adapter; no copied candidate or
local replace. New item API and a suspected mixed-fence defect are recorded
in the plan Revisions; its owner was asked to qualify that class.

Exact next step: on #576 landing notice verify its Git/main reachability,
update the pinned Go module and substrate, reconcile final API/fixtures and
rerun change-code for the clarified plan, then implement the RED regressions
and remaining planned tasks. No feature implementation or long-running check
is active. Expected-red tests are checkpointed with this dependency wait.

### 2026-10-23 14:39 — upstream fence qualification corrected

Authenticated ariadne:2 receipt1fbe8bfc-2363-4734-a721-86910b5b597c confirms
the suspected shared ChatItems fence defect with measured pre-fix failures.
Fix f8b819032777535397f4eb4be6af4c020def126a on #576's branch uses the
Goldmark-backed sidedoc.CodeLines mask. Verified the committed shared call and
test definition by Git inspection; no candidate installed and no local test
pass claimed. Owner reports mixed/long-fence, indented-code and item-boundary
coverage plus linear counted CodeLines tests.

Plan Revisions records the CommonMark item-boundary detail and shared CodeLines
API for final provenance. #838's existing RED baseline remains the current
local feature evidence. Exact next step is still #576 landing verification,
normal module/substrate update, final API/fixture reconciliation and required
change-code refresh before core implementation. No active runner or stage.

### 2026-10-23 17:34 — landed shared API and first core GREEN

Verified ariadne#576 landed; normal weave substrate refresh contains its merge,
and go.mod pins v0.0.0-20261024002258-354e5fac5c96 without replacement.
Final API change-code passed (plan round 4 CLEAN, estimate INFO). The three
previously behavioral-RED collaborative regressions now PASS through the first
shared scanner adapter: tags route chat, retired values are nonfatal, and tags
remain owned under whole-document legacy. This is partial implementation,
not whole-package acceptance. Full editorial tests expose both retired-axis
expectations and live chat-item integrity cases that must be resolved before
close. Runtime catalog/API deletion, transaction wiring, migration and broader
qualification remain outstanding. No recorded suite or stage is active.

Exact next step: preserve the item/structural guards while replacing retired
axis tests with shared-grammar oracles; delete Catalog arguments and wire the
existing transaction owner. Bounded migration and new-round tasks are delegated
in disjoint files. Fixture pin and docs edits are checkpointed here for recovery.

### 2026-10-23 17:41 — single-mode implementation checkpoint

Removed the runtime Catalog, processing briefs/axis resolver/fill paths and new
proposal generation. Compile/ValidateEdits now accept only immutable source
facts. New rounds/spans explicitly highlight; old direct/propose decoding,
rendering and nested unknown-member preservation remain in their existing codec.
Tag bytes and heading attachment are owned independently of routing, including
whole-document legacy; section freshness includes ancestor tags. CLI human reads
print the notice sequence once; JSON exposes the snapshot diagnostics once.

Lossless map-to-tag migration uses shared headings/sections and exact translated
ask projections. No settings are filled, asks renumbered or independent resolver
introduced. Bullet/preamble asks, setext attachment and changed continuation
refuse with a section name. Frontmatter removal precedes ordered insertions,
so the transform needs no sort. Existing transaction stale/lock/uncertain tests
remain effective. ARCH-DRY, ARCH-ORDER and ARCH-EVOLVE shaped these choices.

Verification: full sessionsync PASS (9.613s); focused shared CLI/migration/round
checks PASS, authored native-input preflight PASS, sessionsync editorial+new-round
race PASS (7.195s). All 13 shared authored primary documents replay through public
CLI with exact notices, owned sources and final bytes; two move cases remain
historical fixture evidence. The existing coarse line diff can coalesce adjacent
chat changes; the chat-tag fixture uses four localized apply transactions. A
coarse-fullfile refusal is characterized, not bypassed. No new submission diff
is introduced (#839 remains separate). Generated shared interval fuzz PASS
5,570 executions; edit-batch oracle fuzz PASS 7,218; migration fuzz PASS 6,314.
These are bounded runs, not a recorded affected/suite result.

Read-only real ops and brain session audits both compile and validate a no-op;
second reads prove unchanged bytes. Metadata-only digests/counts are retained in
workshop/evidence/000838/read-only-legacy-audit.json; no live contents copied.
Brain's existing legacy chat yields seven eligible items and remains unchanged.

The only current pure-package failure is upstream: malformed and empty answer
URLs incorrectly set shared ChatItems.Answered. Authenticated ariadne:2 receipt
f1e61901-6801-4963-92e0-1fd48f09ed01 confirms ariadne#581 owns the correction.
Both regression cases remain RED; no local fork or weakened expectation. Hub
incident receipt d28e7692-a776-434c-bda1-0a3ffda1637d is submitted. Final close
waits for the landed #581 pin, followed by full pure-package/race qualification.

Audited reader inputs before exact certificate commands: historical contract
reads only its frozen 39 members and #823 provenance (existing fixed scanner);
new contract reads its 77 exported members plus #838 provenance, under non-Go
full fallback. document/edits/markdown tests read only frozen editorial fixtures,
keeping their non-Go fallback. marker_test has no repository reads after retired
fixture selection was removed, so its certificate is removed. Native acceptance
reads the selected candidate skill, shared fixtures and testcouch runtime/assets;
new CLI replay reads exported fixture data. Both retain non-Go full fallback.
No local-gate configuration or seeded workflow was changed.

Exact next step: run committed-head disposable native editorial qualification
(Claude/Codex fixture personalities, mechanical CLI conformance, not model
judgment); retain cleanup evidence. Then pin #581 when landed, complete pure
checks/inventory, push and record affected per the fleet notice; request the
hub-authorized operator stage using operator-check.md, then whole close/landing.
No recorded suite, landing runner or stage lease is active at this checkpoint.

### 2026-10-23 17:43 — native harness setup failure classified

Committed/pushed candidate e1e529ee010157fc0a3e1bf29714235798d34f3a native
editorial run EXIT1 after 26.284s, before fixture mutation. The added read helper
unnecessarily invoked session-sync diff, but testcouch's intentionally isolated
PATH has no diff executable. This is a branch test-harness setup error, not a
policy failure or registered flake. Exact candidate skill/prefix passed. Shared
owner teardown confirmed all generations absent, direct child and IO joined;
Codex did not start. Retained native-e1e529ee-failed.log SHA256
c13c0cff549e3bdaa724c7d45f549fe7a51b1a2457ab8753292d1b872823901d.

Removed that extra diff call: Read already provides/acknowledges the snapshot
and no tested editorial action needs a diff subprocess. The harness still uses
real ActorCLI reads/writes and the same isolated PATH; no provisioner broadening,
timeout change or blind same-head retry. Re-audited native reader inputs (same
candidate skill/fixtures/runtime bundle), exact certificate refreshed. Next is
a new committed-head native run. Pure-core scoped race PASS 4.776s; the two
confirmed #581 cases remain the only full-package blockers.

### 2026-10-23 17:44 — native PASS; upstream gate remains

Corrected committed/pushed head afb420dc6863376050548e1e7f40697ca09345d2:
TestNativeEditorial EXIT0, 29.166s total; Claude personality12.31s and Codex
personality10.02s. Each runs the real selected Couch/Zellij path in an isolated
store with synthetic fixture workers. Five authored cases per personality cover
numbered question/command chat, prose marker, legacy bullet map, ordinary command,
question side-doc and all retired-key notices. Exact parent/side-doc bytes,
new highlight rounds, unchanged owned-refusal artifacts, migration preview /
confirm / stale / lossy refusal and candidate skill/prefix passed. Both owners
explicitly confirm all generations absent and direct child/IO joined.
Native log native-afb420dc-passed.log SHA256
580bd3d79b9031008dc194c3c10e76e53e117f275b6b47a10bc0faeea97519c5.

This is mechanical native CLI qualification, not a real model judgment about
questions versus commands. That semantic/editor experience is in the prepared
operator checklist and remains required before close. Qoder is installed, but
the existing shared fixture owner supports only Claude/Codex; no Qoder model
credits or semantic run is claimed and no speculative adapter was built.

Fresh whole editorial package on this same head fails ONLY the two confirmed
shared #581 malformed/empty-destination subtests (0.311s); retained exact output
in core-afb420dc-upstream-581.log. No recorded affected check was queued against
this known-red dependency. Last inventory PASS; no active stage or landing turn.
Next: await #581 landing notice, update the normal pinned module, require the
unmodified regression/full pure package and race PASS, then recorded affected
on pushed head, hub-authorized operator stage and one issue-close review.

### 2026-10-23 19:27 — pre-close selection contract updated

Authenticated hub receipt3ac9bb0c-8bd5-4c73-bf96-a90900380800 supersedes the
explicit affected command above: after the #581 pin and passing targeted checks,
run `sdlc local-check run --pre-close` on the pushed head for close and boarding.
Current installed help exposes that flag and explains affected selection for a
train-enabled repository whose main declares it. Inspect its printed choice and
reason and report a discrepancy to the hub. No check was queued against the
known-red dependency. Fresh authoritative #581 observation: still not landed,
claimed by ariadne:2. All other next steps and preserved evidence remain valid.

### 2026-10-23 20:28 — #581 pinned; pure checks GREEN

Authenticated hub dispatch dfeecc73-2520-4a73-815d-b45c926cd920 resumes final
qualification. Independently verified ariadne PR423 MERGED at
5a17f789619d3dcaafec62ae51b350833836a3e7, and ordinary weave substrate refresh
contains that merge. Normal module pin is now
v0.0.0-20261024032245-5a17f789619d; no replace or candidate fork. Seeded workflow
refresh remains deferred as its own operator-verified change, not included here.

Repinned export v4 at that landed revision:84 members plus manifest,15 active
editorial cases and2 historical moves. New authored fixtures cover invalid answer
links and unread closed frontmatter; all15 replay through public CLI. Historical
#823 bytes stay unchanged. Native harness now includes both new cases too.

A targeted integration test caught Pair's remaining duplicate direct-link regex:
Unicode whitespace destinations still marked an ask answered, including an old
escaped pending marker. Both public edit-guard cases were RED before the fix.
Removed the duplicate direct-link scan (shared ChatItems already owns it), and
reused sidedoc.ValidAnswerTarget for appends and decoded historical marker links.
Both new cases and the old malformed/empty cases now GREEN. No alternate target
rule, scanner, retry or timeout was introduced (ARCH-DRY).

Fresh whole editorial PASS0.396s, whole sessionsync PASS6.880s; full race on both
PASS3.823s/19.952s. Focused CLI Editorial/Round/SessionSync/Collaborative PASS1.199s;
authored native input preflight PASS. Go reports1.27.3. Reader inventory PASS after
individual re-audits of the four changed readers: same candidate skill/runtime
and manifest roots, now with the two new exported fixtures; same frozen fuzz
inputs. Exact certificate commands retained each non-Go full fallback.

Next: commit/push this qualified head; retain `sdlc local-check run --pre-close`
to completion, inspect its printed selection, and run final exact-head native
fixtures. Then send the verified revision and checklist to the hub for its
possibly coordinated #838/#419 stage. No :0 mutation before that coordination.
Whole issue close/review and landing follow actual operator acceptance.

### 2026-10-23 21:18 — native PASS; recorded fixture dependency failure repaired

Exact-head native run on 8e92803f80a2fef84918d40adc7b35baaac83d70 passed
34.999s (Claude and Codex, seven authored cases each). Both disposable fleet
owners proved every generation absent and joined child/IO cleanup. This proves
the real CLI path using synthetic personalities; semantic real-agent acceptance
remains the operator's stage check. Raw evidence: native-8e92803f-passed.log,
SHA256 1237dc29fcc08cb288051e83383819179c181d922369a9b43cd26cc03137f2f0.

Pre-close attempt f933ba7bdbd58992b61fba928e9c6c8e waited normally, selected
affected then the expected cmd/couch full fallback (hub tracks policy in #853),
and exited failed before the Go suite. Its source fixture copied affected and
jsonstrict only, but the pinned planner now imports processgroup, which imports
pkg/procgroup and x/sys. Targeted repeat reproduced the exact compile failure;
this is a deterministic fixture assembly defect, never flake or infrastructure
acceptance. Retained pre-close-8e92803f-qualification-failed.log SHA256
107ee33a30b1eb4c2ada26ce2e9446861419cd825acd9b73338ceb75861b23ed.

Repair b3da4973e copies all four unmodified provider packages plus that same
snapshot's go.mod/go.sum into a disposable root module. No planner fork, fake
process implementation, gate configuration change, retry or timeout change
(ARCH-DRY). Sibling qualification PASS18.146s, installed #581 module PASS20.091s,
exact #581 Git-ref PASS22.022s; each checked778 distinct group/file inputs.
Affected inventory and diff whitespace checks PASS. Hub and pair:2 were informed;
#851 holds the :0 stage lease. No stage mutation was made.

Next: push the repaired checkpoint and run one new recorded --pre-close check.
After PASS, coordinate with pair:2 for the authorized staging-only combined
main+#851+#838 build, not pair:6's superseded stage. Operator checklist is
workshop/evidence/000838/operator-check.md. Actual live acceptance, whole-issue
close/review and landing remain outstanding. Never restart while the new own
recorded check is queued/running; retain its runner and FIFO.

### 2026-10-24 00:03 — pre-close pin-integration failure; adoption #861 first

Retained pre-close attempt5966ebc327601a4b5febe2c1bdf4ed3f on8448a507ba579038d83eadf15ab70810bc8a030f finished FAILED11m41s. Qualification fixture repair passed. The complete named failure set was repolint build (Rule.Match now requires CheckContext) and artifactpath.TestProductionArtifactReferencesAreExactlyClassified (retired editorial/catalog.go still named in NonArtifactSources). Targeted tests reproduced both; removed only that stale inventory entry, artifactpath whole package then PASS34.182s. Suite log retained as pre-close-8448a507-pin-adoption-failed.log, SHA25669149fe9335d1b664e172bc0a7bcd1482b04a33bb54c68f828ab00d279f0d9c0.

A minimal AST-only catalog adapter compiled, revealing51 newly active lint findings:40 test-shrunk-deadline,5 test-socket-fixture-root,5 json-recursive-scanner,1 tmp-literal. Raw capture retained as catalog-pin-findings.log. Adapter patch saved at /tmp/pair838-contextual-lint-adapter.patch, removed from this checkout for the independent main-based adoption. No rules were disabled, no waivers, no new suite or stage.

Filed unclaimed #861 with exact evidence and findings. Authenticated hub dispatch a942ac93-44d3-441a-90c6-a627e3cd83ed assigns adoption to this slot, on its own branch from main, and approves adoption-first sequencing. #838 cannot use main's old81c832d5: production requires ChatSections and RetiredKeys (#575), Legacy/Tags/Headings/ChatItems provenance and linear CommonMark behavior (#576), and ValidAnswerTarget plus unread-header fixes (#581). Keep the normal required pin; never fork this shared authority.

Next for #838: after #861 lands, integrate main, verify inventory and rerun --pre-close. On PASS coordinate main+#851+#838 staging-only combined build with pair:2 (owns :0 lease); parley.nvim#419 is separately staged. Actual operator acceptance, whole-close/review and landing remain pending. No own runner remains live now. Work resumes on #861 next.

## Revisions

### 2026-10-23 — initial implementation design

Expanded the empty Plan after the TL's dispatch, preserving the operator's
Spec and Done when. Recorded the approved ariadne#575 dependency, shared
scanner ownership and one atomic review boundary. Proposed (not yet approved)
lossless-only migration refusals and ariadne#576 final qualification, with the
reasons and alternatives in the durable plan.

### 2026-10-23 14:08 — approved migration detail

TL approval makes the draft's lossless-only migration refusal part of this
implementation contract; the refusal must name the affected section. Upstream
sequencing and project operator acceptance are approved as recorded above.

### 2026-10-24 — adoption landed; integrated targeted qualification

#861 landed PR580 as965ba1ee6097dba463e75a3409547bf589904430, tracker done/archive verified. Merged landed main75594fccb65aee96bc538d8690ff7396c669f19c into this original issue branch. Resolved modify/delete catalog.go/catalog_test.go by preserving this issue's approved deletion; no runtime consumers remain. Adopted main's go.sum, removing only obsolete intermediate pin sums; go.mod stays normal581. Combined artifact manifest preserves this issue's deleted catalog removal and861's new root scanner/cue helper entries. No protected gate config changed.

Fresh targeted checks PASS: full editorial race5.736s, full sessionsync race61.458s, public CLI Editorial/Round/SessionSync/Collaborative11.126s, full artifactpath23.433s and repolint21.042s. Reader inventory and diff whitespace checks PASS without certificate refresh. No runtime behavior resolution beyond the approved catalog deletion, no new pin, timeout or suppression.

Next: commit/push integration, rerun exact-head native editorial through shared disposable Claude/Codex personalities, then one recorded --pre-close. On PASS send exact qualified head to pair:2 for the authorized combined main+#851+#838 stage. Pair:2 reports851 repaired-head suite PASS and preserves its lease. Operator semantic/render acceptance and this issue's mandatory whole close/landing still remain pending.

### 2026-10-24 — integrated native blocked by shared startup harness

Exact integrated229e2925ab14c8ad7d537a76c89ed046563e3a47 native editorial rerun EXIT1 after324.602s. Claude personality failed Fleet.Start before any editorial case, with steward startup context deadline exceeded; terminal shows threads, hub Ctrl+Space:start and startup:full-shutdown; missing hubs. No actor was started (startup_sources=0, no wrapper-startup.log; empty Zellij server log). Default Options.ExplicitHub=false uses bare Couch, so this extends #863's documented startup-waits-for-auto-agent symptom beyond its explicit-hub reproduction. Fleet.Start still calls WaitAgent(ctx,0) without driving the public start gesture. No #838 edit/read/migration case ran. Codex was not attempted after the first personality failed.

Retained native-229e2925-startup-failed.log, SHA256 27c423eb7f20746631a44c145cb6cd3208b85f8d924d68c930ce53873103da3e. Native owner61220568ef76be9721ce154d6f9af34f cleaned root /Volumes/TestScratch/pc-1766773190: all owned generations absent; direct child/IO joined. Failure capture includes empty server tail and absent wrapper diagnostics. Prior exact8e928 native14 authored cases passed before this main startup change; that evidence remains historical, not current-head native acceptance. No timeout, retry or startup workaround added.

Hub notified of the #863 family occurrence; shared harness repair/acceptance sequencing is its call. Proceeding with the already-required recorded --pre-close on this integrated source while preserving the native qualification gap. Stage still requires recorded PASS and hub coordination with851; no :0 mutation.

### 2026-10-24 — authorized native operator bootstrap

Authenticated hub2da96548 authorizes native qualification to bootstrap via the operator's explicit start gesture instead of waiting for863. Peer pair:6 supplied the measured root/form/path/agent-preview/Enter sequence (f304cdb7); no unlanded runtime code imported. Added only an opt-in PAIR_NATIVE_EDITORIAL_BOOTSTRAP=1 path to the native test. It runs Fleet.Start concurrently, waits for the cold switcher root before Ctrl+Space, supplies the absolute primary path, selects the requested agent through the real form, waits for accepted repo-default preview, and presses Enter once. Start is joined on success and on failure/cancel. Existing editorial steps and300s whole-operation budget remain unchanged; no new timing threshold, shared harness mutation or product behavior. Once863 repairs default startup, ordinary runs omit this temporary qualification flag.

The earlier queued check a802e54730ddd8193f255cffcb6f7b8a / lease-231693481 was still waiting at FIFO6, never executing. Cancelled exactly that owned lease through sdlc; terminal joined EXIT1 naming suite lock interrupted:context canceled, with no test result. Requeue only the final bootstrap head. This avoids an obsolete recorded test head.

Re-audited the one changed reader: original candidate embedded skill/runtime and contract838 fixture trees, generated native mailbox/files, and existing public agent inventory; no new repository read population. Re-recorded only editorial_acceptance_test.go with its existing non_go_full_fallback. Fresh inventory PASS, public authored fixture tests PASS0.213s, full catalog PASS9.377s. Next: commit/push this test-only adaptation, native run with explicit bootstrap on both personalities, then final-head recorded --pre-close.

### 2026-10-24 — integrated native operator-bootstrap PASS

Exact committed fa4d09531 native qualification with PAIR_NATIVE_EDITORIAL=1 PAIR_NATIVE_EDITORIAL_BOOTSTRAP=1 completed EXIT0 in62.010s. Claude and Codex each passed all7 authored cases (14 total), exact parent/side-doc outputs, highlight rounds, shared notices, ownership refusals, and migration preview/confirm/stale/lossy checks. Both isolated owners proved all generations absent and direct child/IO joined. This is mechanical synthetic-agent conformance; operator semantic/render acceptance remains pending. No live fleet was touched.

Raw native-fa4d0953-bootstrap-passed.log SHA256 cbf894e8473c96015c315a9d6b4298c2eb5bb81a2bb4b64221109661692ad9ba. The explicit start gesture resolved the previously captured startup-only failure without changing editorial assertions, product code or budgets. Default startup remains owned by863. Next: push this evidence-only checkpoint, queue one final --pre-close, retain runner/FIFO and tell hub its exact queue identity for offered priority. On PASS coordinate exact reviewed head/checklist with851's owner for the combined stage; never overwrite its lease independently.
