---
id: 000739
status: codecomplete
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-23
estimate_hours: 4.695
collaboration-mode: collaboration
card_mirror: '87eae4743fec4794a42f0ae776e7067330cbbed2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T22:40:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 6.02
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
- [x] Verify counts, compatibility and coordinated live behavior.

Finish recorded verification, the narrow re-close and landing through their SDLC gates; these future gate actions are not pre-ticked implementation claims.

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





- 2026-10-23: closed — Current-maine6fc81e integration resolved one reader-certificate conflict after individual audit; README scans remain fixed_scanner, new noticeCopyAssertion uses private temp fake pbcopy, no host clipboard or extra repository inputs. Main #789 raw hub-path diagnostic requirement exposed both later-hub tests RED; attachment-only failure now wraps canonical hub path with %w, both GREEN1.509s and broader startup cohort green. Main40-cell notice cap exposed unknown-hub fixture RED30.370s; fixture now checks capped visible projection and exact retained Ctrl+Y full diagnostic via existing fake tool, GREEN0.679s and race2.367s. No startup authority/state/cap change. Full suite recorded pass with only registered #825 exact rerun on00758b61. New --pre-close chose affected then duplicated same-head full; hub e7b60be7 explicitly directed cancellation of duplicate lease-2108133384 and use of existing complete pass; supported cancellation confirmed, no result erased. Required inventory passed immediately before this close; project updates belong to hub. (fixed-then-ship; full suite: Recorded suite PASS on00758b613ad81cc917ba1a1a2b61eb3b66b7852c with registered #825: original613efed12adfe88e07be6e28e21736c1 exit2 in13m34s retained hash a96147fb6b3f; SDLC-selected exact rerun3fd8ebe977c7b1b1303afa1239f58d92 passed2.75s and final local-check exited0, status pass. #679 and couchcmd passed first run. Only Log commit after tested head.); review verdict: SHIP
- 2026-10-23: closed — Round8 SHIP retained; newer-maineb5171b integration had only atlas/index.md conflict. Resolution preserves both #739 startup automation/cues and main #827 full-quit entries. No code-file conflict or branch behavior change; code auto-merged. Integrated targeted startup/automation/worker/admission/quit/artifact tests PASS couchcmd7.373s,couchtty1.325s,artifactpath0.176s; reader inventory unchanged and passes again immediately before re-close. Recorded full suite PASS12m38s on9aa240f1, all scripts/packages completed. Previous operator full-shutdown and explicit-enable acceptance retained, no new stage or unrun native matrix claim. Project state belongs to hub. (fixed-then-ship; full suite: Affected selection chose full suite; recorded PASS on9aa240f1d7480f572a4955002c7d3785c6650db5, attempt8cf3d0e3e83d8e0d37074a8603101baf/lease-101118425 exit0 in12m38s, suite.log SHA prefix98ef73fd50b1. Exact tested head retained unchanged.); review verdict: SHIP
- 2026-10-23: closed — After full round7 SHIP, landing required mainc0dbb32 integration. Three conflicts resolved additively: artifactpath inventory keeps both startup and sketch sources; each of two reader inputs audited individually before recording combined digest, preserving fixed_scanner and local_or_literal_fixture. No bulk refresh or source guard deletion. Targeted integrated startup/automation/cue/admission/protocol/artifact cohort PASS, README/attachment/warm-pass cohort PASS0.933s. Full integrated suite PASS12m37s on1c94b285; retained runner survived restart, no duplicate run. make test-affected-inventory just passed again before this re-close. Prior operator full-shutdown/explicit-enable acceptance and full round7 scope disposition remain intact; #737 second-lander seam unclaimed and #833 shared shutdown cause limitation retained. Project updates belong to hub. (fixed-then-ship; full suite: Full recorded suite PASS on integrated1c94b2857cba472ddb2cf171b1b98e0fb893deb9, attempt88d3d5919c0ea4c1859393b07b247572, exit0 in12m37s, saved suite.log hash6c088b0e9063; subsequent commit only retains the result in Log.); review verdict: SHIP
- 2026-10-23: closed — Full close requested by BR-6: explicit acceptance-substitution table now records each replaced native matrix case, actual operator shutdown evidence, retained portable socket/count tests, and #737 second-lander obligation. Operator hub4339b063 authorized close/landing after actual full shutdown on eccddab6: automation OFF, no automatic cold hub starts, manual hub attach, TL explicit ON; no unrun native matrix claimed. Startup status recovery wording and exact historical stdout regression pass. Corrected startup-ordering fixture holds the write then handles normal fake-host termination; RED0.398s, GREEN20x0.601s, race1.754s. Shared shutdown-cause issue #833 filed unclaimed, no presenter fix included. Recorded full suite PASS on9d3ab0e70aa05783ad47ef73544967d0142b5250 attempta792a901883fb83113a37db329b949e9 exit0 in10m47s loghashb3f86c1f4b99; subsequent commits only log, plan ticks and generated review evidence. Earlier #679 timeout and fixture failure retained in Log. Required reader inventory just passed again without refresh. Project updates belong to hub.; review verdict: SHIP
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
- 2026-10-21: closed — Recorded final-code full suite attempt080a8de027740459d8a9639a4d4f603f on e94267fd6c9026a06a26bafd6f7e2911a576af44 PASSED exit0 in11m2s, log SHA prefix492cf5028e0c; later commits only evidence/dispute. BR-1/2/4/5 addressed in prior review; BR-5 broker direct and both enqueue commit races plus producer original-permit controls passed normal/race and mutation source guard. Installed command explicit-start PTY test passed3 repeats15.141s; unused deadline removed, lint passed. BR-3 disputed with matched main/branch cold and warm cache measurements plus retained trace attribution: main already exceeds caps, full paired attribution reverses branch delta, trace accounts129–131MB; no cap change, no claim of zero production cost. Independent #765 owns cache-enabled envelope. Atlas contains hub-effect/scheduler/broker/automation order, all producer and worker cue coverage. Only unchecked Plan item is live disposable fleet acceptance after #744/#761 and landing; hub explicitly permits suite and close now but landing remains HELD until that live check. No live/stage claim and no pair:0 access. #737 second-lander wiring remains explicit. Hub owns project.; review verdict: SHIP

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



### 2026-10-21 — first completed recorded suite, two branch failures fixed

Attempt 1b71544c093cf5636404605c1bac6c19 on 1b948b3c completed the full
suite in 10m8s, exit2, log hash prefix 0cfc2528dcda. Retained failure log:
/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/1b71544c093cf5636404605c1bac6c19.log.
Failures were cmd/couch.TestBareCouchInstalledCommand (old automatic-launch
expectation) and internal/repolint.TestTrackedGoFilesPassCatalog (an unused
20ms registration timeout in StartupDoesNotResumeProvenHubConversation). All
remaining reported packages passed, including core, tty, messaging and wrapper;
the suite continued to completion. Existing affected-config source qualification
and real-docflow smoke were skipped for unavailable sibling provider source in
the disposable checkout. This is not a passing suite or a flake classification.

Installed test red reproduced its unchanged15s timeout. Fix first waits for
the actual empty switcher/full-shutdown status and proves no Pair launch, then
drives the existing Ctrl+Space Start form with dot + Enter. Exact one-launch
argv, rendered marker and real fixture wrapper readiness remain asserted.
Three complete repetitions pass (15.141 s total); no deadline increase or new
fleet harness. The no-resume test's unrelated registration-timeout override is
removed: no registration should happen there. Focused core case passes0.325s;
tracked lint catalog passes3.804s. Audited installed-test reader remains
local_or_literal_fixture; fast inventory and whitespace checks pass.

Round4 independently disposed BR-5, including mutation failures when each fix
was removed. It reopened BR-3 for cache-enabled Go's access-log writes, despite
-count=1 success; close is still REWORK, not finalized. Filed unclaimed #765
(cache-test-write-envelope) per hub91039d1a. A frozen-main/branch comparison is
running in private roots with separate empty caches, identical cold/warm modes,
and no shared-cache clearing; its actual numbers are required before dispute.
No test-runner policy or cap change is in #739.


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


### 2026-10-21 02:56 PDT — BR-3 controlled counter-measurement

Hub91039d1a requested the same cache-enabled invocation on frozen main and
branch, cold and warm, before deciding causation. Filed unclaimed pair#765
for the independent cache-enabled invocation envelope; no cap or runner-policy
change belongs to #739. The controlled evidence follows.

#739 controlled cache-enabled package measurements

All four invocations exited 1. All test assertions passed; only the existing physical-write budgets failed. No cached package result was reused, including warm runs. No source or budget was changed.

Pinned main: 53a9d42e503efa47ecef42beec294a30ba87768f (base)
Pinned branch: 1b948b3c040f6979eaafa77b1ad2da127f524da4 (work)
Go: go1.27.1 darwin/arm64, cgo enabled.
Command: go test ./cmd/internal/couchcore ./cmd/internal/couchtty
GOFLAGS=-p=2 -trimpath -v (verbose, cache-compatible; no -count or -run)
Each variant has a separate initially empty private GOCACHE, reused for its warm run. Identical TMPDIR/GOTMPDIR point at this task's tmp directory. Runtimebundle generation happened beforehand with the normal shared cache. No shared cache was cleared. Runs were sequential, with at most two package processes.

Go source cmd/go/internal/test/test.go:1861 admits -test.v to cacheArgs. Thus these measurements preserve cache-enabled execution but do not use default quiet output.

| Mode | Package | Bytes written | Cap | Seconds | Failure |
|---|---|---:|---:|---:|---|
| base-cold | couchcore | 143224832 | 134217728 | 283.825 | budget only |
| base-cold | couchtty | 105287680 | 67108864 | 54.419 | budget only |
| base-warm | couchcore | 143204352 | 134217728 | 219.561 | budget only |
| base-warm | couchtty | 104529920 | 67108864 | 42.836 | budget only |
| work-cold | couchcore | 158978048 | 134217728 | 158.853 | budget only |
| work-cold | couchtty | 105304064 | 67108864 | 47.208 | budget only |
| work-warm | couchcore | 162770944 | 134217728 | 146.409 | budget only |
| work-warm | couchtty | 105287680 | 67108864 | 42.784 | budget only |

Main already exceeds both caps in both cache modes. Branch adds 15,753,216 cold / 19,566,592 warm core bytes versus matched main modes; tty adds 16,384 cold / 757,760 warm. Therefore this establishes a pre-existing defect, but does not establish zero incremental cost from #739.

Concrete candidate mechanism: cache-enabled Go supplies -test.testlogfile. During main cold execution at 09:30:08 UTC, the core testlog was already 27 MiB. These observations do not separately attribute all write differences to Go's testlog.

Complete stdout/stderr logs and SHA256 hashes are recorded alongside this report in metadata.json. Worktrees, generated bundle, caches and logs retained for the owner; no cleanup performed.

Log SHA256 (complete stdout/stderr):

- base-cold: 9702cbaa68e9b10be86c980205552123b1b2dbe79ff04c0fa5b85cbc47e248a7
- base-warm: d8b801d8d281ea631323584229417490932230645f229ce1f27c2d266f7707f2
- work-cold: 3effd7b37d19e762952334c0f4833b9393cd7f6b6abba7bc82293740339556a1
- work-warm: 9d4007e1935e95d45e83d197462d8b58fd18747c09f5ce67a1005bd5e34b61ee

Hub5c4f9ccc accepts the pre-existing cap failure but requires attribution of
the branch increment before the dispute. Attribution controls are running;
do not yet claim the extra writes are tests rather than production.

Final-code recorded suite attempt080a8de027740459d8a9639a4d4f603f on
e94267fd6c9026a06a26bafd6f7e2911a576af44 passed exit0 in11m2s; log
SHA prefix492cf5028e0c, saved under .git/sdlc/local-checks/<head>/suite.log.
Core190.474s, tty40.275s; installed explicit-start command and lint passed.
Live disposable-fleet acceptance remains owed after #744/#761; no stage or
merge is authorized before it.


### 2026-10-21 03:04 PDT — BR-3 attribution controls

Do not attribute the initial +16–20MB aggregate to production writes. The
controlled follow-up does not reproduce that direction. Exact frozen variants
and private caches are those above; source untouched, <=2 package processes.
A conservative union of 37 changed/new/removed test names (exact function blocks
and regex retained in changed-tests.json/.regex) and its complement gave:

| Cache-enabled cohort | Main bytes | Branch bytes |
|---|---:|---:|
| Changed test names | 5779456 | 5304320 |
| Unchanged test names | 127647744 | 127021056 |
| Full core, retained trace | 137310208 | 134885376 |

All assertions passed. Cohorts passed the budget; full runs failed only the
unchanged cap. The full control uses go test -work ./cmd/internal/couchcore,
same cache-compatible GOFLAGS and private caches. -work retains builder output;
it does not disable cache logging. Main301.209s, branch309.294s. The full
physical-write delta reverses to branch2424832 bytes LOWER. Its retained Go
test-cache access log is main130916787 versus branch129125877 bytes, itself
1790910 fewer bytes. Counts: main stat1146582/open170047/getenv20257; branch
stat1126055/open170358/getenv20008. Repeated canonical-path ancestor stats
dominate the cache log. Go's test-cache trace is the measured bulk write
source; the initial positive aggregate delta is not stable under the paired
full control, and neither disjoint test cohort grows. This does not assert
zero production cost or precisely assign every physical byte. It rejects
treating the initial aggregate delta as proof of added production writes.
Logs, traces and hashes remain at the same private measurement root. #765
owns the cache-enabled invocation envelope; #739 keeps all existing caps.


### 2026-10-21 03:08 PDT — close SHIP; acceptance held

Round5 SHIP, all findings disposed; BR-3 withdrawn after the reviewer
independently verified retained trace/log hashes and counts. Recorded suite
pass remains final-code evidence. No merge: disposable-fleet acceptance still
waits for #744 after #761, and second-lander #737 integration stays explicit.
Attribution report: /Volumes/TestScratch/tmp/pair-slot7/pair739-cache.QFbUSM/attribution-report.md.
Final retained evidence SHA256:

- base-changed.log: e6b05114e40e1dfc71325aefcf1db3de368c9ad75a943e13750cad0b764698dd
- work-changed.log: f7521ebe473f098ba2e46af5c2243cb9426e6494cefcb81b89eb825eddfcf0eb
- base-unchanged.log: c0aab6708f9018b99d6481df6a93c9457ac217790fdcf5af1a6656cb9826db8c
- work-unchanged.log: 9d36a1fcfeae4942a21bcb709ea96ab76c2d51065fe2a225c4fc8a03e0184be4
- base-trace.log: 9ffb59fe789c19244fdfe27a0408a715279377988e393aa5195ec8e67cd4ad8d
- work-trace.log: 0967f05d917a1285edfd5a20923a5191f294b10a358a503b6c582219d078686e
- base-testlog.txt: 8dfcc5a6b9e9b9d99ece981a5973a75882c3783c871b052121a35e82285ae485
- work-testlog.txt: 8cb5b803f17edbe05e28edb5e7a1556da49b7d3454b09dd152f505705a8309ec

Calibration caveat: this close adopted measured0.39h after excluding activity
already adopted by interleaved #738 (which adopted9.91h). This is the binary
provenance result, not a hand estimate; the cross-item attribution looks
suspect and is reported to the hub for calibration, with no card override.


### 2026-10-22 — morning live-check preparation

Hub5f401991 schedules #739 for pair:0 first in the morning, before #737; no staging until explicit go. After #738 landed (PR513), merged origin/main7702a3dddbfdb5c7d8a021bb8949df98f0a6b282. Resolved additive atlas/artifact-source/help conflicts preserving both main and #739. Main retired the aggregate reader inventory; migrated #739's six affected reader certificates to the landed per-package format through its supported record tool, preserving main's aggregate deletion. Fast inventory passes with no waiver.

Targeted startup/automation/original-permit/worker-cue/compatibility cohort passed across couchcmd3.511s, couchcore2.715s, couchtty0.567s, couchmessage4.327s and wrapcmd0.642s. Commit-guard cohort passed couchmessage0.122s; exhaustive production artifact inventory passed2.230s. Actual native acceptance remains owed; no stage/live claim or landing here.

Three-step operator check: (1) hub coordinates a full-shutdown or unknown-state start on the selected build; a normal restart with a positively surviving hub intentionally starts automation ON and does not qualify as the OFF scenario. (2) Observe one retained startup status line and hub automation OFF; from the hub, couch --automation status confirms OFF, and no generated hub composer input arrives (worker reports/operator input remain deliverable). (3) Send the TL the first genuine operator message; unless it says keep automation off, the TL explicitly runs couch --automation on, then status confirms ON and a fresh automatic occurrence can arrive with no replay. Couch never parses that message or flips implicitly. Keep the normal-restart and per-wrapper-cue disposable qualification owed separately unless the hub changes that acceptance scope.

Hub59dbba43 changes the subsequent order to #773 before #736, after this morning preparation. #773 peer captures are pending attachment on its own branch; do not mix that independent issue's changes here.

Final preparation checks: automatic-system source owner and native-input worker witness cohort PASS (couchcmd0.316s, wrapcmd0.343s); fast reader inventory PASS after migration. Conflict-resolution changes and new Log have clean diff checks against incoming main. The whole historical merge diff reports existing whitespace in main's retained VT/patch evidence; those raw captures are deliberately preserved, not normalized. No staging, full-suite pass on this new integration or live acceptance claimed.


### 2026-10-22 — refreshed operator shutdown checklist (hub b6bfeab3)

This supersedes the earlier morning checklist only for rollout sequencing and operator machine maintenance; automation semantics are unchanged. #753 currently owns pair:0. No #739 stage until the hub explicitly releases that stage and says go.

1. Hub coordinates the approved full shutdown, not a supervisor-only restart with surviving sessions. While Couch is fully down, the operator deletes the old AgentScratch volume and runs brew upgrade go (approved1.27.1→1.27.3), then starts Couch on the selected #739 build. These machine actions belong to the operator; this worker does not execute them.
2. After restart the hub coordinates one rebuild in every slot with the upgraded Go toolchain. Observe one retained startup status line and hub automation OFF; couch --automation status from that hub confirms OFF. No generated hub composer input should arrive; genuine worker reports and operator input stay deliverable. A positively surviving hub on a normal restart intentionally starts ON and does not qualify for this OFF scenario.
3. Send the TL the first genuine operator message. Unless it says keep automation off, the TL explicitly runs couch --automation on; status confirms ON and a fresh automatic occurrence arrives without replay. Couch never parses the message or flips implicitly. Retain normal-restart/per-wrapper-cue disposable qualification separately unless the hub changes that scope.

Integrated current main after #773 landed. Conflict resolution retains #758 editor service initialization/cleanup and #792 proven/legacy orphan fields/validation alongside #739 automation ownership/control. Main orphan payload guards run before automation's existing strict request-shape check; both admission rules remain. Atlas/help/artifact-source conflicts preserve both sides. Two reader certificates were refreshed only after auditing the merged existing source readers: run_test remains fixed_scanner and couch_test remains local_or_literal_fixture. Fast affected inventory and exhaustive artifact classification pass; selected automation/startup/worker-cue/compatibility cohort is running. No stage or new full-suite/native pass is claimed.

Selected merged-head cohort completed PASS: couchcmd2.000s, couchcore19.480s, couchmessage0.329s, couchtty0.784s, wrapcmd0.458s; artifactpath3.132s and fast reader inventory pass. Whole staged merge diff reports pre-existing trailing whitespace in two incoming main review sidecars (#758/#792); those main artifacts are not edited. Branch-versus-main diff check passes. Full integrated suite/native smoke remain outstanding; readiness is for the authorized operator staging check only.


### 2026-10-22 21:08 PDT — stage authorized for operator full shutdown

Authenticated hub ae3eba4b authorizes stage on pair:0 now, superseding the prior stage hold and tonight’s general no-new-landing instruction only for this requested smoke preparation. Switched the clean slot to #739 and integrated origin/main5f64ce54412afec1982b296b641cd515034482a0. Three conflicts resolved additively: #739 startup/automation atlas text plus main’s #796 tab styles; #739 non-artifact classifications plus #743 VM paths; root switcher retains main’s scoped title and #739 Ctrl+G automation hint. No policy redesign or machine action.

Merged targeted cohort PASS: couchcmd4.515s, couchcore2.706s, couchtty0.612s, couchmessage0.196s, wrapcmd0.717s; commit/queue admission and group/switcher cohort PASS couchmessage0.115s/couchtty0.511s. Artifact classification cohort PASS2.387s; fast reader inventory PASS. No full-suite or native pass is claimed for this fresh integration. Commit/push, then sdlc stage --build; stage output must verify exact binaries before operator restart.

Final operator checklist: (1) fully shut down Couch and its slots, not a supervisor-only restart with surviving hubs; while fully down the operator deletes AgentScratch and runs brew upgrade go1.27.1→1.27.3. (2) start Couch on the verified #739 staged build. Observe one startup status line and automation OFF; cold missing hubs remain parked, with Ctrl+Space explicit Start form if needed to bring the TL up. From each tested hub couch --automation status must show OFF; no automatic hub composer input, while operator input and worker reports remain deliverable. (3) send the TL the first genuine message; unless it says keep automation off, the TL explicitly runs couch --automation on and confirms ON, with a fresh generated occurrence and no replay. Couch does not parse that message or flip itself. The hub coordinates one rebuild per slot after the Go upgrade. Normal surviving-hub restart is intentionally ON and is a separate scenario. Await the operator’s result; no close/merge authorized by staging alone.

Stage/build completed exit0 on pair:0 at eccddab6adfbf6e0ef0e3c6b1c6aa3f0dd222d9b. Verified pair:0 HEAD again after build and its clean tree. Binaries couch, couch-conformance-bridge, pair, pair-launch-helper and pair-slug each report that exact vcs.revision and vcs.modified=false. Sent the full five-step operator checklist as Couch receipt0c8236f3. Stage stays held for operator full-shutdown acceptance; worker performs no AgentScratch deletion, Go upgrade or live restart. Next after authorized restart: reread this Log, rebuild this slot once with upgraded Go when hub instructs, await actual smoke result, and reconcile the outstanding normal-restart/worker-cue disposable qualification and final recorded/re-close requirements before any landing. No merge runner or background suite is active here. This post-stage Log commit changes only evidence, not the staged binary pin.


## Revisions

### 2026-10-22 — operator shutdown acceptance and historical status wording

Authenticated hub4339b063 reports actual full shutdown around21:15 on staged eccddab6: startup classified full-shutdown, automation OFF, workers parked and no cold hub start; operator manually attached ops and ops-echo, then the TL enabled automation after the first message and status confirmed ON. That is the approved live full-shutdown/explicit-enable acceptance. Worker launch delivered its generic cue in this session. Existing real-private-socket cross-supervisor/negotiation regressions remain the evidence for surviving-wrapper non-duplication; no unrun disposable-fleet matrix or scheduled-run implementation is claimed. #737 remains second-lander wiring per the existing contract. Hub directs stage clear, close and landing after two acceptance follow-ups.

Cold missing hubs are intentionally not resumed by startup; make the existing retained message name explicit switcher recovery (Resume from disk on a retained hub, Ctrl+Space Start if absent), with no change to lifecycle authority. Classify the apparent missing-after-attach defect: owner.missing is an immutable startup observation, not a current liveness view. Preserve that snapshot/protocol field, label CLI rows missing at startup, and document the distinction instead of adding live polling or attach-triggered policy transitions. The complete live three-line stdout from authenticated hubfe9ff602 is retained as a fixture in automation_cli_test.go. New regressions failed before these wording fixes (couchcmd0.569s). Existing tests and docs now defend historical semantics and actionable recovery.

The final issue checkbox mixed verified behavior with future close/land operations. Record the verified behavior as complete and keep those gates as separate mandatory workflow instructions, not an advance claim of landing. The coordinated actual operator shutdown/enable result supersedes the held rollout entry; portable count/session tests retain their own scope. No gate or native coverage is fabricated.

### 2026-10-22 — post-upgrade recovery and stage clear

Go reports1.27.3. Rebuilt this slot once with make build; bin/couch confirms Go1.27.3. Authenticated hub6725a312 lifts the freeze and permits suites/landings. queue status still printed helper inspect exit2; reported once as requested. Hubc901ab67 classified it as known display-only version skew (ariadne:3 fixing); suite admission is separate and remains usable. No workaround, duplicate run or repeated incident report.

Stage clear completed exit0: pair:0 restored clean main at54a02d7aa3971f50987bcef321de6534d8d566a9; all five built binaries report that exact revision, modified=false. The current running supervisor remains the already-accepted staged build until its next restart. Local startup/automation/cue/commit/group cohort PASS after fixes: couchcmd9.252s, core25.215s, tty1.246s, message0.233s, wrapcmd1.425s; reader inventory PASS. No suite or merge runner survives from before shutdown; start one fresh recorded check on the pushed corrected code, then narrow re-close with that full-suite evidence, and land on SHIP. No more pair:0 staging is needed for these bounded wording corrections.

Final exact-live-fixture and startup-summary cohort PASS0.373s; focused CLI/summary/owner race PASS1.534s. make test-affected-inventory passes on the final edited readers without hash changes or waivers. Build and stage clear both finished; no pending pre-shutdown runner was resumed. Commit/push this coherent correction, then record a full suite because the narrow re-close also covers earlier code-conflict resolutions; do not substitute an unrun affected subset as full-suite evidence.


### 2026-10-22 23:24 PDT — full suite completed; #679 timeout classified

Exact pushed code cc7892568e0e91d4cc74516729c5d2d82e8924f0, attempt7c7510bb47fa2dbaa737a27a12e3a9fc/lease-1229791935, completed exit2 after26m13s. Sole failure: TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse in changelogcmd timed out20m; main test waits retention_test.go:53, distiller reads FIFO via os.ReadFile/changelogcmd.go:100. This is the exact pre-existing #679 stack, with zero changelogcmd diff against main. All other scripts/packages passed; no known failure ended the suite early. Saved log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7c7510bb47fa2dbaa737a27a12e3a9fc.log, full SHA2565a13e9973c2201b82f217132c031fa46ea75417479152fb90010ead425a4c130.

One unchanged same-head focused repeat passed package0.268s/test0.10s under a60s diagnostic bound. That is not a full-suite pass and does not authorize manual status posting. SDLC classified original package timeout as unattributable-failure (no failing test), so named-test resume cannot explain it. Existing unclaimed #679 occurrence published through sdlc issue publish; no duplicate issue, registration override or production/timeout fix. ariadne:3 notified that the natural timeout released this suite token.

Authenticated hub26cdf202 approves ONE unchanged full recorded rerun on exact cc789256. Next command: ariadne:0 sdlc local-check run --name suite --rev cc7892568e0e91d4cc74516729c5d2d82e8924f0. This checkpoint changes only this Log; the approved runtime revision stays exact. Record the second receipt/result when available. On pass, narrow re-close with full evidence, then land; same package timeout again means STOP/report, no third blind run. No active old suite/merge runner; no source or test change since the failed attempt.


### 2026-10-23 — recorded rerun and startup fixture ordering correction

Hub4909f07c authorized the sole rerun on pushed eafb27680f56c799d06d18e7639b15b6e5d154c4: Git confirmed it differs from cc789256 only by nine appended Log lines, with identical runtime and tests. Attempt598a61ed6ef5c7be79c26f6cf168a978/lease-3761720268 completed exit2 in11m43s; changelogcmd/#679 passed1.456s. The sole failure was branch-owned TestRunConsoleObservesStartupBeforeArmingProducers (0.02s), console startup=1 with presenter unavailable. No further full rerun or close was attempted before classification. Full log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/598a61ed6ef5c7be79c26f6cf168a978.log; SHA2564d2073b0e476e23dc459f3e348fdbd72e8d08ce6839f2f3fcd6cf06bb0763bf8. All other scripts/packages passed. Hub sweep answered with submitted receipt551cc31a.

Fresh-eyes read-only trace agrees: observable host bytes precede completion of Panel and its chrome bookkeeping. Calling asynchronous Stop at that instant cancels an in-flight publication; synthetic presenter-unavailable errors can lose the causal cancellation on subsequent requests. Focused unchanged repeat20 passed2.691s, which did not explain away the suite failure. Hubf9476c31 approved correcting this startup-ordering fixture and filing the shared production cancellation-classification defect separately. Filed unclaimed #833 through sdlc issue new, carrying the suite receipt, source trace and limits of the deterministic reproduction. No shared presenter production fix or blanket suppression is included here.

Held-write control first proved RED0.398s: old Stop cancelled the startup paint before release. That deterministic control proves the fixture's unsafe teardown ordering, not the exact synthetic-error interleaving. The initially proposed Escape exit was qualified and rejected: with no active live actor, today's switcher Escape displays no live thread can receive focus and does not quit. The exploratory invocation reached its unchanged2m test bound; no deadline bump. Notified hub via submitted receiptf91def06. Corrected the fixture to queue existing FakeHost.Terminate(SIGTERM), handled by Run only after the held initial publication, rather than cancelling from the test goroutine. No production hook, setting, policy or process signal is introduced; this is the fake host's normal termination channel. Cleanup now joins Run even on an assertion failure.

Held-write corrected test passed20 repetitions0.601s; targeted startup/summary/automation race cohort passed1.754s. make test-affected-inventory passed without hash refresh or waiver; git diff --check passed. Hubf9476c31 authorizes one full recorded suite on this corrected new pushed head, then narrow re-close and landing on SHIP. Commit all evidence before starting that one check; do not move the pushed tip during its run. No previous failing attempt is a passing-suite claim.


### 2026-10-23 — corrected full recorded suite passed

The one authorized corrected-head suite on9d3ab0e70aa05783ad47ef73544967d0142b5250 finished PASS exit0 in10m47s, attempta792a901883fb83113a37db329b949e9/lease-1721583275. Saved full log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/9d3ab0e70aa05783ad47ef73544967d0142b5250/suite.log, reported SHA256 prefixb3f86c1f4b99. All scripts and Go packages completed; prior #679 package timeout and branch startup-fixture failure remain recorded above as failures, not erased or relabeled passes. No production code changed in the fixture correction. The tip stayed fixed throughout the recorded run. Next: required fast reader inventory again immediately before narrow re-close, then sdlc merge --yes on SHIP; landing owns current-main integration and its authoritative checks. No new manual main catch-up or pair:0 stage.


### 2026-10-23 — BR-6 acceptance substitution submitted to full close

The narrow round6 review correctly requires a full close for this acceptance change. This revision does not treat an unrun native matrix as completed. Operator/hub4339b063 explicitly directed close and landing after the coordinated real full-shutdown check and its two wording follow-ups; that rollout is the substitute for the earlier disposable-fleet acceptance contract.

| Earlier acceptance obligation | Replacement or retained evidence | Remaining obligation |
| --- | --- | --- |
| Disposable full shutdown: no surviving hubs, automation OFF, startup status/recovery, explicit enable | Actual operator full shutdown around21:15 on staged eccddab6: OFF/full-shutdown, hubs absent until manual attach, ops and ops-echo recovered, TL explicitly enabled on first message; exact status stdout replay plus recovery-message regressions | No claim that the disposable version ran; full close must judge the substitution |
| Disposable normal restart: surviving hub starts ON, retained worker gets no new start cue | Portable startup-observation cases for normal/mixed/unknown and private real-socket cross-supervisor cue-witness/count/negotiation regressions | Native normal-restart matrix removed from this landing acceptance, not marked passed |
| Disposable fresh/relaunch worker: exactly one generic start cue | Wrapper delivery/count/guard regressions and actual worker-start cue delivered to this session after the coordinated shutdown | No native per-agent matrix or measured whole-fleet count claimed |
| Scheduler pause/skip/notice suppression through automation | #739 getter/permit boundary and commit guard tests; documented lock order | #737's second lander owns actual scheduler wiring and combination qualification; no unlanded scheduler acceptance claimed |

No outstanding planned implementation is hidden by these substitutions. Startup authority, message suppression, cue ownership and compatibility requirements are unchanged. Shared shutdown diagnostic gap #833 remains a separately filed known limitation; #739's ordering fixture now exits via its existing fake-host termination event after a held publication. Recorded full suite PASS10m47s on9d3ab0e7 remains valid for the unchanged code. Run full sdlc close with this explicit contract instead of narrow close; no review bypass or fabricated native evidence.


### 2026-10-23 — full close SHIP; landing-main inventory conflicts

Full round7 close returned SHIP, zero open blocking findings, explicitly accepting the revised rollout contract; measured actual5.79h, no manual override. Immediately entered sdlc merge --yes, became ready on PR510 atb5ca04e7e658d0e4b6476bdef41d6e3906e58a6b. Landing integrated origin/mainc0dbb32decc052500172ded01ff36e3fa8e49cc9 and stopped on three inventory conflicts. It left the source branch clean and directed manual integration, full suite, narrow re-close, retry. No server merge, active runner or queued landing is claimed after that exit.

Reperformed that exact integration. artifactpath/manifest.go keeps both #739's existing non-artifact policy sources and main's #753 sketch-source entries, with no deletion or reclassification. Each reader certificate was audited individually by a separate read-only worker: couchcmd/run_test.go still reads README in two source guard tests, all62 tests remain in SOURCE_GUARDS, so fixed_scanner stays; other inputs use isolated fixture files/fakes. couchcore/couch_test.go reads only test-owned stores, namespace fixtures, marker/log/claim files and test-executable-backed fake native commands, so local_or_literal_fixture stays. Neither parent digest describes the combined file. Used exactly two per-file record commands after resolving their certificate text; no bulk refresh, source-guard deletion or waiver. Reader inventory and diff check PASS.

Merged targeted cohort PASS: artifactpath0.168s, couchcmd8.297s, couchcore25.788s, couchtty1.216s, couchmessage0.211s, wrapcmd1.811s. Named README/first-attachment/warm-pass/interactive-hub integration cohort PASS0.933s. Auto-merged main endpoint, sketch, switch parameters, configuration and wait-diagnostic changes stay intact beside #739; no policy redesign. Next commit/push the coherent integration and run one recorded full suite on that new exact head. Only after its pass: required fast inventory again, narrow re-close with --full-suite evidence, and re-enter merge. Prior branch pass remains historical; integrated code is not yet full-suite verified.


### 2026-10-23 — integrated suite survived restart and passed

Retained run88d3d5919c0ea4c1859393b07b247572/lease-2992196955 finished exit0 on integrated head1c94b2857cba472ddb2cf171b1b98e0fb893deb9 in12m37s, log SHA256 prefix6c088b0e9063. Saved /Users/xianxu/workspace/pair/.git/sdlc/local-checks/1c94b2857cba472ddb2cf171b1b98e0fb893deb9/suite.log. Every script/package completed. Authenticated Couch restart noticeee02a694 prompted checking the retained result; no duplicate or replacement runner was launched. No runtime or tests changed after this result. Next fast inventory, narrow re-close with this full integrated evidence, then merge on SHIP. Hub owns project state and live restart follow-up; no new stage.


### 2026-10-23 — round8 SHIP; newer main atlas-only conflict

Narrow round8 SHIP, no new/open findings, accepted the three additive inventory resolutions with full integrated suite evidence. Immediately re-entered sdlc merge --yes; gate vouched PR510 atb421eb4a7. Newer origineb5171b07c91b8784139a1ab8cd42f6cc35e8c66 then conflicted only in atlas/index.md. Tool aborted its disposable merge and directed source integration/re-close. Resolved that same merge by retaining main's #827 full-quit description and #739's startup automation/cue link. All code auto-merged; no code-conflict resolution or new branch behavior. Reader inventory passes unchanged, no hash refresh. Targeted startup/automation/worker/admission/quit/artifact checks PASS couchcmd7.373s, couchtty1.325s, artifactpath0.176s; diff check passes.

Next commit/push this integration, record affected verification on the exact new head, then required fast inventory and narrow re-close. The last full integrated PASS12m37s remains historical evidence on1c94b285; do not claim it tested this newer main. Narrow close's code-conflict full-suite precondition does not apply to this atlas-only resolution; let the gate check it rather than bypassing it. The landing gate still owns a full suite on the final integrated code. No active previous merge runner remains after its exit1.


### 2026-10-23 — train320 hold; merged diagnostic compatibility repair

Retained merge exited1: train320 considered #739/PR510 and #802/PR563, held #739 for a main/train conflict, then abandoned with no surviving cars or split suites. No landing or live runner was claimed. Hub sweepd2f8a43a directed recovery; fetched and integrated origine6fc81eac82c5106f942c09dca4fac7139b17a0b. Only conflict is couchcmd/run_test.go's reader certificate. Reviewed actual merged source and new noticeCopyAssertion helper: two README reads still require fixed_scanner; new clipboard IO uses t.TempDir, private fake pbcopy and observed output, no host clipboard or additional repository reads. Existing SOURCE_GUARDS coverage stays intact. Recorded only that one reader's new hash; inventory PASS.

Targeted qualification found a real auto-merge incompatibility: both TestHubStartupLaterFailureLeavesFirstUsable and its AfterWarmWorker counterpart now require canonical failed-hub path plus diagnostics (main #789), but #739's no-action surviving-terminal attachment returned a bare attach error. RED couchcmd3.310s named both assertions. Corrected only that attachment-error return to wrap the existing canonical hub path with %w; no new retry, startup selection, recovery authority or state. Reported source cause and bounded repair to hub in submitted receipt21e52028. Same pair GREEN1.509s. Broader integrated startup/automation/worker/admission/README/borrowed-attach/artifact cohort PASS couchcmd4.702s,couchcore4.327s,couchtty0.825s,artifactpath0.085s; reader inventory/diff check PASS. No assertion was weakened, no current failure registered as a flake.

Commit/push this integrated repair and record a fresh exact-head check. Prior full suites are historical, not a claim on this code. After pass, fast inventory before narrow re-close, then sdlc merge --yes; no old merge runner remains. #842 planning still waits until #739 actually lands, with #843 after it.


### 2026-10-23 — full run failed on #679 and bounded-notice fixture

Attempt781b71ad61ee23da17e5241e9e635768/lease-2661605640 onb4083e3de7777a50b5d1132a51b7a0c24f51caab completed exit2 in25m55s. Two failures: recurring #679 TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse timed out20m (retention_test.go:53 waits, changelogcmd.go:100 FIFO os.ReadFile); and TestBareCwdHubResolverFailureStaysUnknownAndUsable timed out30s waiting for its full unknown-notice prefix. All remaining scripts/packages completed. Full log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/781b71ad61ee23da17e5241e9e635768.log, SHA256543305f185a343bf6b2eeb6fe67d6aa5c05e4102037ac03a193a9efc35b899f6. Zero changelogcmd diff against main. SDLC reported unattributable-failure for the package timeout. Stopped full retries and reported both causes, rather than interpreting that summary as the only failed package.

The branch-owned notice failure is deterministic: main #789's ChromeStatusCells40 bounds the paint, while this test searched for the49-cell unbounded prefix. Focused unchanged RED30.370s confirmed it. Fixed the fixture only: observe ChromeStatusText(want), send Ctrl+Y through a private input pipe, and use main's noticeCopyAssertion with its private fake pbcopy to verify the complete retained unknown-startup diagnostic. The expected observation explicitly describes unknown identity, missing hub, parked-unknown and the original resolver error. No production cap, suppression, identity admission or clipboard change. GREEN0.679s, startup/unknown/summary/late-hub race cohort PASS2.367s; required reader inventory and scoped diff check pass, no new reader certificate.

Authenticated hubf158ddc2 approves ONE new full suite after this fixture correction, and registered-flake resume for #679 if it alone recurs; do not blind full-rerun. Sent exact #679 evidence for its dispatched owner; if SDLC's unattributable timeout classification refuses a named resume, report exact refusal. Commit/push before recording the new check and preserve its tip. No close/merge is currently active. Native raw terminal captures newly inherited from main retain their original whitespace bytes; the earlier whole-merge cached diff check warned on those .raw files, while the authored startup fix/certificate/Log scoped check passes. Do not normalize captured terminal evidence to silence that diagnostic.


### 2026-10-23 — recorded pass with #825 exact rerun; duplicate pre-close cancelled

On unchanged pushed00758b613ad81cc917ba1a1a2b61eb3b66b7852c, run613efed12adfe88e07be6e28e21736c1/lease-2114806963 completed all scripts/packages. #679 passed1.535s; couchcmd passed64.618s. Sole failure was registered #825 TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation, cycle42 leaked results. SDLC automatically selected exactly that package/top-level test, rerun attempt3fd8ebe977c7b1b1303afa1239f58d92/lease-3160770864, PASS2.75s after normal capacity wait. Original run13m34s, original log hash prefixa96147fb6b3f retained at /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/613efed12adfe88e07be6e28e21736c1.log. SDLC exited0 and recorded sdlc-local/suite PASS with known-flake #825 rerun; original exit2 was not relabeled as a clean first pass. No manual status or extra full retry.

Fleet notice1ecca8ea introduced local-check run --pre-close. Tried it on that unchanged exact head: it correctly chose affected for Pair, but affected selected full for cmd/couch/main_test.go and queued duplicate879cb7f3123ce97bdec1590ff694d52a/lease-2108133384. Reported the same-head duplicate promptly. Authenticated hube7b60be7 directed supported cancellation and use of the existing full-pass/#825 evidence, which covers the affected selection; hub is filing the reuse defect. Cancelled exactly lease-2108133384 through sdlc local-check cancel; exit0 confirmation. Its source runner joined with context-cancelled queue admission, no test ran and no new result replaces the existing pass. No other check was cancelled.

Next required reader inventory and narrow re-close on the two already-tested integration/fixture corrections, with this full recorded evidence and explicit #825 disposition; then immediate sdlc merge --yes on SHIP. No further main catch-up before close, no new source or tests since the passing head. Future checks use --pre-close, with the existing accepted pass used here exactly per hub instruction. #842 remains next after this item actually lands, then #843; #850 was reassigned to pair:4.
