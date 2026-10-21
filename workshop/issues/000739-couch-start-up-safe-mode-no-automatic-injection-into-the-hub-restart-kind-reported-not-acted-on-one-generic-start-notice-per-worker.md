---
id: 000739
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 4.695
collaboration-mode: collaboration
card_mirror: '9ef7a944c1d4a0d67b0bfa18f80f9b80b832fb84' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T22:40:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker

## Problem

After a restart, Couch acts on its own: it injects system notices into every slot, the hub included ("Couch
restarted and the hub is available…"), and recovery has been attempted automatically at times. The operator (10-20,
drain redesign, ops side doc `2026-10-20-17-drain-redesign.md`): no automatic recovery; a restart may be for
something else, or the setup may be wedged; the operator tells the TL "start recovery" in one line.

## Spec

- **Hub safe mode:** on start, the hub's TL gets no automatic system message injection and nothing is auto-started
  inside the hub.
- **Restart kind, reported only:** Couch tells a normal Couch restart (an attachable Zellij session for the fleet
  exists) from a full shutdown (none), and reports it to the TL once, in one line ("full shutdown detected: N slots
  parked; recovery waits for the operator"), without acting on it.
- **Worker start notice:** when a worker slot starts (resume or fresh), Couch sends one short generic notice: "you
  restarted; reread your issue Log and continue where you left off; report to your hub". The TL may follow with its
  own message; Couch never sends more than the one generic notice per start.

### Confirmed interpretation for operator review

Safe startup attaches surviving hubs/workers only; absent hubs or slots stay
listed, with no cold resume or Fresh, including an empty installation. Publish
the restart-kind report as one retained Console status line, never a composer
message. Unknown or orphaned evidence is not a full-shutdown proof. All explicit
operator-authorized lifecycle actions keep their existing guards.

**Hub automation** controls only Couch-generated hub input and scheduled runs. CLI: `couch --automation off|on|status [hub]`, plus switcher control. It starts on for a positively surviving hub after normal restart; off after full shutdown/unknown startup, and off for missing hubs in a mixed fleet. Only explicit on/off changes it afterward. Human submissions never flip it: the TL can debug with the operator while off. The separate tl-lead rule will run automation on on the first operator message unless told keep off; Couch performs no message parsing.

CLI mutation is own-hub-only; optional hub must name the caller's canonical hub. Only the operator switcher may control another configured hub. While off, suppress automatic hub input and skip scheduled due runs (#737) like missed runs; on resumes at the next due time without replay/catch-up. Already-running scripts may finish (soft stop). Worker/peer reports, operator input, status and desktop output are never gated. The retained line says "hub automation off; say turn it on" and names missing hubs. Focus the launch hub if it survived, otherwise the switcher; extra-hub adoption never steals focus.

The generic worker cue is once per **wrapper launch incarnation**, including
Fresh, resume and relaunch; same-incarnation reconnect across a Couch restart
does not repeat it. Its memory lives in the wrapper, not a file or a supervisor
attempt set. Keep the **separate existing Couch-restart resend prompt** once per
worker address per Couch lifetime: that asks for still-relevant unanswered
reports, rather than claiming the worker's agent restarted. The operator reviews
this distinction and the per-Couch-lifetime alternative in the durable plan.

## Done when

- Tests pin: no injection into the hub on start; restart-kind classification from fixture Zellij state; exactly one
  generic notice per worker start.
- Live: after the next full-shutdown/unknown start, the TL sees one report line and no generated hub input until automation is explicitly enabled. The separate tl-lead rule owns the first-operator-message decision; Couch never parses that message.
- Cross-supervisor tests keep one worker alive through a private Couch restart
  and prove no duplicate generic cue; replacing/re-execing the worker creates one
  new opportunity. Unknown/uncertain delivery never authorizes duplicate input.
- Negotiated old/new wrapper compatibility preserves ordinary messaging without
  treating an unsupported cue report as "not yet delivered".

## Plan

- [x] Qualify safe startup classification and attachment-only selection for all hubs.
- [x] Keep startup lifecycle summaries out of hub composers and preserve a usable switcher.
- [x] Add wrapper-owned one-shot worker orientation through negotiated session reporting.
- [ ] Verify counts, compatibility and coordinated live behavior; close and land once.

Durable plan: [startup safe mode](../plans/000739-startup-safe-mode-plan.md).

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against
baseline-v3.1.md, Method A only; estimate-source flags calibration stale.
Implementation values are v2 primitive hours scaled once by0.4; familiarity1.
The approved thorough design discounts design by0.2 and uses15% design buffer.

Breakdown: issue/spec (v2 design0.5, impl0.3); startup observation extension
(0.3/0.5); one new automation policy owner (1.0/0.8, design halved by reuse of
existing reducer/identity/controller infrastructure before spec discount);
wrapper/session integration (1.0/1.5); three small producer-family extensions
for lifecycle, episode/reminder and resource/stall suppression (each0.3/0.5);
Console/CLI automation controls and focus qualification (TUI1.0/1.0);
scheduler predicate integration (0.3/0.5); docs/inventory (0.2/0.2); one close
review/active landing handling (0.2/0.5); bounded real Zellij/reconnect
qualification (impl0.6, the real-API discovery allowance); two operator-driven
UX/ownership iteration groups (each0.8/0.3). Existing broker, strict codecs,
version fallback and stateful fakes avoid a greenfield protocol/service or
external library. No machine-wait time or hub-owned restart work is counted.
Design1.30 ×1.15 + implementation3.20 =4.695h. #737 wait is not effort; its
injected predicate is an existing-stack integration, not a second scheduler.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.12
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.10 impl=0.32
item: api-integration design=0.20 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
item: ux-rename-iteration design=0.16 impl=0.12
item: ux-rename-iteration design=0.16 impl=0.12
design-buffer: 0.15
total: 4.695
```

## Log

### 2026-10-20

Read the fleet-drain project's design.md before design and claimed/start-planned
from fresh main. Source-only exploration confirmed implicit cold/Fresh hub
fallbacks in StartInteractive and hub injection via lifecycle startup summaries.
The existing resend coordinator already skips verified hubs; it is not the
producer of those hub summaries. A bounded read-only subagent traced message
producers and existing tests; no live process, production code or test changed.

Hub confirmed attachment-only startup and per-wrapper start-cue ownership, then
clarified that the separate Couch-restart resend prompt stays. A wrapper-held
constant-size delivery witness and negotiated reporting preserve this across
supervisor replacement. Existing launch ordinal distinguishes a native restart
through wrapper re-exec even if PID/birth survives. Operator review is pending;
no change-code or estimate has been run. The hub owns project edits and rollout.

The producer sweep found that filtering lifecycle startup alone still permits
inherited wait episodes, a hub's own session-file reminder, resource episodes
and stall reports to wake the TL. Hub chose the per-hub quiet gate until genuine
operator submission; all suppressed occurrences are retired, not queued for
release. The narrower baseline-only option stays in the operator-review table.
Archive warnings are passive status/attention/desktop paths, not broker agent
input. The unlanded #737 scheduler must share the gate for any future hub input;
a peer seam query was refused without a receipt, so no agreement is claimed.

### 2026-10-21 — implementation admission and checkpoint

Plan-quality first returned PQ-1 (replace prose scenario enumeration with named
risky-function/adversarial-strategy guards). The revised plan passed CLEAN;
change-code then passed on 6695cb081 after deriving the v3.1 estimate. All unit,
protocol and fake-backed tests are included in the estimate primitives: the
wrapper/session integration row includes negotiation and receiver tests;
supervision nudges belong to the episode/reminder family, and the common send
source guard belongs to the inventory/docs row. The two UX iteration allowances
cover completed operator shaping, not two future implementation rounds.

The hub relayed #737's agreed seam: messageService owns
hubAutomationOn(hubRoot string) bool; scheduler reserveDue, reserved-script
authorization and scheduleNotice.Admit consult it. Whichever lands second wires
the consumer on main, without stacking. The earlier refused query is superseded
by this explicit agreement.

Implementation in progress. Pure hub automation and CLI parser reds were
observed, then focused tests passed. The controller rejects workers, stale or
disconnected callers and foreign hubs; explicit concurrent switcher toggles are
serialized. Unknown startup remains off. Core attachment-only startup, negotiated
wrapper cue receipt and switcher Ctrl+G have separate targeted evidence from
bounded worker tasks. No full suite, close, stage or live acceptance is claimed.

### 2026-10-21 — verified component behavior

Targeted core startup/automation/lifecycle/resource tests passed (5.346 s).
Combined couchcmd automation/startup/producer/supervision/wait/stale tests passed
(17.257 s), and the CLI/controller/source-owner group passed with race checks.
The producer group passed under race, including delayed admission and status
output, off/on during unknown-identity resolution, and resource pending retries.
Suppression is an explicit terminal disposition, never a delivered claim.
Every generated SendSystem now enters through sendAutomaticSystem, pinned by an
AST source-reader guard. Human/worker sends use their unchanged normal path.

Protocol/session/registry/background/peer targeted tests passed in both packages
(5.770/5.889 s); new private-socket cue tests passed race count 3. They count
native input across supervisor replacement before and after confirmation,
receipt eviction, lost acknowledgment, batch followers and same-binding new
launch ordinals. Bounded decoder fuzz passed with two workers for 3 s; it is
bounded qualification, not a failure-rate or performance claim. Counted 20/40
worker tests prove one child proof initially per needed launch, zero extra
ready/process/git work on unchanged ticks, and one additional proof for one
changed ordinal. Unknown role, launch proof and receipt remain conservative.

Changed old expectations intentionally: implicit startup no longer creates or
cold-resumes a missing, parked or orphan-adjacent hub. Explicit lifecycle tests
remain. Lifecycle collector tests now use explicit operator/peer events during
a held startup pass; new tests prove automatic baseline observations never
flush. Background negotiation tests formerly hardcoded hello-v4 as newest and
now name hello-v5, while old-peer strict downgrade behavior remains tested.
The fast reader inventory is audited rather than waived; modified preflight
names follow renamed semantics and the sole-ingress source guard.

### 2026-10-21 — live acceptance dependency and close ordering

Code head 1580bf163bd5869428d5831e3d5d9cef2b18f3ec is published as PR #510.
Recorded affected selected the full suite because the clean runtime bundle graph
is unavailable (#738); attempt 4e8fab1bec9d0e9cb2e5fad70d697437 remains in the
normal machine queue. No duplicate suite, staging or live fleet action occurred.

Hub requires three private real-supervisor scenarios before landing: full
shutdown gives automation off, one status/no hub injection and empty-hub start
guidance, followed by explicit on and a fresh notice; normal restart preserves
hub automation on and reattaches workers without a new generic cue; an actually
started worker receives exactly one cue. pair:5 confirms #744's shared harness
is still unlanded. Hub c23219fb-e84d-4e8a-ae60-c54976d1f4f3 directs waiting for
#744, no duplicated fixture, finishing this suite and close review meanwhile,
and reporting if #744 slips beyond the next morning. Landing stays blocked on
that live evidence; the real fleet first use is the next restart after landing.

### 2026-10-21 — stale cue rejected before native input

Pre-close source audit identified a missing enforcement point in the promised
per-launch contract: the same Binding/socket survives a fresh exec, so an old
ordinal cue can reach the replacement endpoint before registry cancellation.
The initial model ignored that receipt for the new witness but did not reject
its native paste. A real private-endpoint regression proved the old cue was
pasted while ordinal 8 still reported needed; the earlier replacement test only
sent the new ordinal's cue and missed this adversary.

The receiver now calls shared ValidateWorkerStartNotice before singleton queue
mutation and before freezing every batch member. Canonical system self cues
need the exact current ordinal/ID; unknown ordinals fail closed. Ordinary peer
prose and other system messages are unaffected. Reds covered singleton, batch
head and batch follower with missing/new ordinal; the stale follower previously
pasted too. Green endpoint regression rejects old 7, releases its refused
reservation, and pastes current 8 exactly once. Broad peer/session/negotiation/
registry/background/worker tests passed normally and with race; focused cue
race tests passed count 3. This repairs the accepted contract, not a new
credential, queue or broker recovery policy.

The superseded recorded suite on 1580bf16 was cancelled through local-check
cancel after 501 s in queue, before taking machine capacity or running tests.
No test failure, flake retry or suite pass is claimed from that cancelled run.
The corrected pushed head needs a new recorded check. Live acceptance still
waits for #744, now itself delayed by #761's observed detach JSON refusal.

### 2026-10-21 — complete source-inventory preflight

A targeted artifactpath.TestProductionArtifactReferencesAreExactlyClassified
preflight found six new policy/coordination sources absent from the exhaustive
production inventory. Each delegates through existing stores/transports and
contains no Pair artifact filename constructor or vocabulary; audited them as
NonArtifactSources. The exact guard now passes (1.641 s). The second recorded
check on 670704af was still queued at 484 s and was cancelled through SDLC before
capacity/tests, so no machine suite was wasted on this known failure. Also
removed stale cold/Fresh startup and startup-summary prose from the shipped
skill, atlas/index and the older detailed atlas map. Added the stale-generation
admission and handoff-inventory lessons; no gate waiver or known-violation entry.


### 2026-10-21 01:34 PDT — first close review REWORK

The binary-owned first boundary review at 588e2ce6 returned BR-1 (concept
traceability), BR-2 (an old empty-start integration assertion), and BR-3
(ordinary core/tty fixture write envelopes). No close finalized. Correcting
BR-1's pure/IO ownership and locations; reproducing BR-2 before updating its
assertion and preserving a surviving-session attachment case; measuring BR-3
without increasing either cap. The recorded affected-to-suite attempt
88eb38ffab1923ce37fca6b243c5bc7e was cancelled through local-check cancel while
still queued, before any tests or capacity acquisition, because the reviewed
head needs fixes. Fallback registered its lease as suite (not affected); the
first affected-filtered cancel changed nothing, then suite-filtered cancel
confirmed cancellation. No other checkout's run was changed. The final head
needs a fresh recorded check. Live acceptance remains held for #744/#761.


### 2026-10-21 — BR-1/BR-2 disposition and BR-3 measured control

BR-1: durable plan Core concepts now names LifecycleNoticeState's pure
suppression transition in couchcore/lifecycle_notice.go separately from
couchcmd/lifecycle_notices.go's IO publisher, and locates ObserveStartupContext
in startup_policy.go; unchanged Console observation provenance is not claimed
modified. BR-2: reproduced the old empty-start test red, then split it into
no-launch usable-switcher and explicitly seeded warm-survivor cases. Related
couchcmd startup family passed in 2.171 s; affected inventory needs no new
reader entry/hash for that fixture-only test.

BR-3 tty control: ordinary go test ./cmd/internal/couchtty -count=1 -v passed
in 49.529 s, writing 39,800,832 bytes under the unchanged 67,108,864-byte cap.
The same complete suite excluding only the added SwitcherAutomation/RootMenu
tests passed in 53.201 s with exactly 39,800,832 bytes; those added tests alone
wrote zero. TMPDIR was /Volumes/TestScratch/tmp/pair-slot7. Thus the new tty
tests add zero measured writes. Reviewer's 87,949,312-byte result is retained
as observed, but not reproduced or explained by this control; no tty test or
cap was altered. Core's pure classification fixtures now avoid the filesystem;
its ordinary package run remains pending before claiming the envelope passes.

### 2026-10-21 — ordinary core package within unchanged envelope

After replacing five pure policy fixtures with direct ThreadSnapshot inputs and
omitting unrelated layout-update writes in the 20/40 store-backed observation
fixtures, go test ./cmd/internal/couchcore -count=1 -v passed in 346.072 s.
TestMain measured 101,834,752 bytes under the unchanged 134,217,728-byte cap.
Log: /tmp/pair739-core-budget-after.log. Focused startup tests also passed
(11.032 s; 2,895,872 bytes). The difference from the review's 154,050,560 bytes
is not a controlled per-fixture savings measurement because its TMPDIR was
not recorded; no such attribution is claimed. Both ordinary packages now pass
their strict existing envelopes. No production code, assertion bound, cap or
timeout was changed in this review round.


### 2026-10-21 01:51 PDT — second review and dead-symbol red

The second review disposed BR-1/2/3 as addressed and independently passed all
five ordinary changed-package suites under the unchanged caps. BR-4 found
SelectStartupHub has only test callers; reproduced the artifactpath
TestNoProductionSymbolIsReferencedOnlyByTests failure locally, then removed
the unused helper and corrected its test/plan/atlas claims. Production
StartInteractive already owns candidate selection and is unchanged. Recorded
attempt a03f6aec6edf17e9d977e064f5859480 on ea07b031 was cancelled through
local-check cancel while still queued, before capacity or tests. No test pass
is claimed for that attempt; final-head recorded verification is still due.


BR-4 verification after deletion: complete artifactpath package passes in
9.606 s, including the previously red dead-symbol guard and source inventory;
core startup tests pass in 2.692 s; affected reader inventory and git diff
--check pass. No allowlist entry, dummy caller or assertion bound was added.


### 2026-10-21 01:59 PDT — BR-5 blocks admission; re-plan requested

Third review returned REWORK for automation-admission-ordering after disposing
BR-1 through BR-4. Its controlled broker routing and resource resolution
regressions demonstrate stale authorization reaching mailbox admission. No
production change made in response yet. Proposed shared-mutex commit guard and
end-to-end occurrence permit are recorded in the durable plan for hub approval.
Attempt 93635701037ab150a21410e4c99d0b5c was cancelled through supported
local-check cancel while queued, before tests/capacity; not a failed suite or a
pass. Prior ordinary packages passed, but final corrected-head recorded suite
remains required. #738 is unblocked per hub2765f7be and may interleave after
this checkpoint; local-gates verification will be operator-owned.


### 2026-10-21 — BR-5 implementation and measured verification

Reproduced service routing off/off-on admitting a stale queued notice, and
the prior early check refusing a canonical admission after off. After the fix,
service routing, original-recipient/role, mutex extent and canonical receipt
controls pass under race x3 (1.848 s). Broker nine controlled routing, Observe,
Reserve, family and both-enqueue cases were red with no guard calls and green
after commit guards; guard lock/receipt and enqueue success controls pass.
Complete couchmessage package passes (25.150 s), focused guard race x3 passes
(1.249 s), and worker cue tests pass (0.429 s).

Producer regressions reproduce resource off/on during resolution and a pending
resource changing hub at an equal epoch. Original hub/epoch now survive pending
refusals and clear only on metric recovery. Lifecycle, wait (direct and
coordinator), stale, stall and supervision each cycle off/on at Reserve and
assert suppression, retirement and fresh-occurrence behavior. Resource's own
controlled delay is resolution; shared broker tests qualify its commit path.
Broad producer group passes (4.579 s) and race passes (7.566 s); core episode
normal/race pass (0.295/1.581 s). Full artifactpath passes (7.109 s).

The one generated-ingress source guard now requires the commit policy. A
compile-valid mutation passing nil reproduced its expected bypass failure;
restored production code afterward. Three changed readers retain their audited
dispositions (source guard fixed_scanner; resource fixture tests
local_or_literal_fixture). No new production file, artifact path or protocol
field. The total scheduler lock order is in atlas and broker/owner lock extent
is mechanically asserted. All targeted evidence precedes the final recorded
check; no final full-suite pass or private live acceptance is yet claimed.


## Revisions

### 2026-10-20 — operator-review startup boundary and two notice lifetimes

Per ops:0's dispatch and subsequent clarification, make startup attachment-only
for all hubs and workers, retain the report in Console without waking the TL,
and keep unknown evidence explicit. The agent-start cue is wrapper-owned and
once per launch incarnation; a Couch-only restart must not call a surviving
agent restarted. Preserve the older per-Couch-lifetime resend prompt as an
independent worker-only mechanism. The plan presents the simpler per-lifetime
alternative for the operator to accept or overrule before code. Existing
explicit recovery, ordinary later lifecycle notices and warm worker attachment
remain, with no new durable recovery state or automatic action.

### 2026-10-20 — gate every automatic hub-input producer, discard without replay

Source audit found automatic producers outside startup lifecycle summaries.
Per ops:0's explicit decision, all automatic system input to each hub stays
suppressed until genuine operator submission to that hub, and suppressed
notices never replay. The worker resend prompt remains distinct and deliverable
to workers while the hub is quiet. Add producer-by-producer tests, current-hub
submission negatives, protocol negotiation and cross-supervisor worker-cue
qualification to the operator-review plan. This is still design-only work.


### 2026-10-20 23:42 PDT — operator approval: explicit start gate and partial hubs

Operator approval relayed by ops:0 requires the name "start gate", an explicit `couch --start-gate open`/switcher act, read-only status, and no automatic release on operator submission. The TL may debug with the operator while closed. Keep scheduled-task exit-10 wakes behind it with no replay. Reattach surviving hubs with independent closed gates, name missing hubs in one status line, and focus the launch hub only if it survived; otherwise leave the switcher open. The active Spec and durable plan now use these rules. Earlier submission-release text above is superseded historical design. Proceed through change-code; no implementation yet.


### 2026-10-20 23:44–23:48 PDT — final operator policy and plan-quality disposition

Operator renamed the control to hub automation, with explicit off/on/status [hub] and switcher control. Initial on only for positively surviving hubs on normal restart; full shutdown, unknown state and missing mixed-fleet hubs remain off. Off is a soft stop for scheduled runs as well as notices, with missed runs skipped and no replay/catch-up on on; running jobs finish. Human/worker input and status/desktop output stay independent. CLI is own-hub-only, operator switcher may select another hub. tl-lead's first-operator-message default is a separate ariadne rule; no Couch parsing or implicit flip.

First plan-quality round returned PQ-1 Important for prose case inventories instead of named risky functions/adversarial strategies. Active plan compressed into function and producer strategy tables; all latest operator decisions are in the active Spec. No code admitted, no estimate derived. Hub relays #737's predicate seam once pair:1 is back; integration waits for that issue on main.

### 2026-10-21 — concrete #737 provider/consumer ordering

Hub message 752adf46-b8e6-4e99-b76a-dbcdf8e69331 confirms the agreed scheduler
getter and second-lander wiring. #739 may publish its provider first; it does
not claim unlanded scheduler coverage. #737 must wire reserveDue, script
authorization and scheduleNotice before its own landing if it is second. No
stacked development. The implementation's per-hub epoch is a memory-only fence
for delayed notices across off/on, within the approved occurrence-owned policy.

### 2026-10-21 — empty configured hub projection

Final integration proved a never-started configured hub has no ThreadRecord and
therefore no ordinary switcher actor row. Hub approval f7d34e39-4724-43c5-bba1-
32359ac86584 permits a display-only parked/missing hub entry with one explicit
recovery action, without a new selectable hub-row model. The existing Ctrl+Space
Start form accepts the exact hub path and previews before starting. The entry
names that action; no fake conversation address or primary-as-numbered identity
is introduced. Bare-cwd startup separately binds the same verified HubOwnership
to routing and Console, so Ctrl+G works there too; failed resolution stays
unknown with a usable switcher and an explicit diagnostic.

### 2026-10-21 — live wording aligned to final operator policy

The original Done-when phrase "until the operator speaks" is superseded by the
approved explicit automation policy. Full-shutdown/unknown starts stay off until
an explicit control; normal surviving hubs start on. The separate tl-lead rule
chooses the first-message action, and Couch never flips on a submission. Live
acceptance remains owed and must be coordinated with the hub; no stage or
restart is claimed here.

### 2026-10-21 — reviewed code before shared-harness live acceptance

Per hub c23219fb-e84d-4e8a-ae60-c54976d1f4f3, finish recorded checks and the close
review before #744 lands; then integrate its main code and implement the three
approved private-fleet scenarios. This is not permission to land without live
acceptance, nor to use a second fixture. The remaining live Plan checkbox is
intentionally pending at that review boundary; its precise close-plan exception
must say so, rather than falsely ticking acceptance. Any post-review integration
or test code follows the normal narrow re-close and verification gates.


### 2026-10-21 — close-review traceability and test alignment

BR-1 corrects the delivered concepts' source paths and PURE/IO classifications
in the durable plan. BR-2 aligns the existing startup integration test with the
already-approved attachment-only contract; BR-3 preserves the existing package
write caps and requires measured fixture attribution. These fixes do not add
startup behavior, recovery authority or another fixture. The private-fleet live
check remains outstanding on #744 after #761.


### 2026-10-21 — remove unused startup selector (BR-4)

Delete SelectStartupHub and its false production-consumer claim. Keep the
actual classifier and existing StartInteractive attachment logic, with direct
classifier candidate assertions and the seeded real entry-point regression.
No new selection mechanism or authority; no scope expansion.


### 2026-10-21 01:59 PDT — proposed BR-5 admission ordering, awaiting hub

The third close review disposed BR-4 and reproduced two remaining races:
automation off in the broker families callback still produced a queued system
receipt, and a resource alert captured its epoch but lost it across recipient
resolution, so off/on revived the old alert. A pre-send check is not an
admission boundary. Stop production edits pending approval of this revision.

Proposed smallest correction (ARCH-ORDER, ARCH-DRY):

- The resource making the invariant one-step is the existing hub-automation
  owner's mutex. Every generated hub occurrence carries its expected canonical
  hub, automation epoch and exact recipient binding through routing, observation
  and reservation. The same permit must reach the mailbox commit, never be
  recaptured as current after slow work. Worker-directed messages remain
  independent; no persistent state or new visible state.
- Extend only the internal generated-send seam with an optional admission
  guard. At each broker mailbox mutation (direct admission, already-busy
  enqueue, and busy-after-observation enqueue), under the existing broker
  mutex, the guard verifies the actual exact recipient, locks the automation
  owner, and checks enabled + expected epoch. Keep that lock through the
  in-memory mailbox mutation, then release it. No routing, endpoint IO, native
  write or callbacks with IO occur while the automation mutex is held.
  Lock order is broker then automation; control mutates automation only and
  never acquires the broker, avoiding an inverse dependency.
- Off and first mailbox admission become ordered: off first suppresses the
  occurrence, including an intervening off/on; admission first keeps its
  canonical receipt and existing delivery/uncertainty rules. Already-admitted
  work is not retracted or falsely reported undelivered. Receipt lookup and
  wrapper-retained committed receipts retain their current semantics.
- Sweep lifecycle, wait, stale reminder, resource, stall and supervision
  adapters to transport the occurrence's original permit. Resource dispatch
  must pass its stored epoch through resolve/send. Unknown or invalidated
  permits retire through each producer's existing suppressed disposition, not
  busy/retry. Scheduler #737 receives this same seam on second landing; no
  independent policy stack.
- Controlled red/green tests hold routing, Observe and Reserve, then complete
  off or off/on before releasing. Cover direct and both enqueue paths,
  unchanged-policy success, binding replacement, and the inverse ordering
  (admission first remains canonical), plus per-producer epoch transport.
  Preserve counts, worker/human exemptions, native timeouts, queue bounds and
  no-replay behavior. No automatic retry or extra executor.

Simpler alternative: another policy check immediately before SendSystem is
insufficient, because broker routing/endpoint IO follows it (the measured red).
Holding the automation mutex across all broker IO would serialize off behind
slow external work and deadlock callbacks that control automation; reject it.
The proposed guard holds it only over the actual in-memory commit.

Visible states remain hub automation on/off with explicit on/off recovery; no
new user state or data-loss mode beyond intentional retirement of suppressed
occurrences. Live acceptance remains owed through #744 after #761.


### 2026-10-21 02:01 PDT — BR-5 correction approved

Hub b88aa936 approves the original-permit/commit-guard revision above, with
total lock order hub-effect gate → scheduler authority → broker → automation
recorded in atlas and cheaply asserted. Implement now, then take #738 while
the private live check waits for #744. No new external protocol, persistence,
queue, retry, executor or visible state. The second #737/#739 lander verifies
their combined locks and getter/notice wiring.
