---
id: 000652
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours: 1.753
collaboration-mode: collaboration
card_mirror: 'fd420de4d6af87de947a42f823cc38814dde48ee' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T15:33:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Codex tall composer misses submission confirmation beyond 20 rows

## Problem

The Codex composer recognizer scans at most 20 rows above the cursor (`codexComposerMaxRows`, `cmd/internal/wrapcmd/composer_recognizers.go`). A tall pasted composer is therefore Unknown even when its visible bold prompt and continuation rows are intact. During an already active turn, the wrapper writes the paste and submit, then sees a genuine empty composer but never sets `sawOccupied`; `turnAtSubmit` was already true, so neither confirmation branch succeeds. It reports Indeterminate after the existing 30-second delivery window despite the input submission. A legacy 26-line singleton reproduces this on main; #648 batching amplified the same preexisting issue.

## Spec

Fix positive Codex composer/submission evidence for tall native pastes without treating Unknown as Occupied or a successful write as proof of submission. Keep partial writes, stale screens, menus, status rows, operator input and loss of identity conservative. Do not increase the post-paste timeout or add a retry.

The recognizer is also used by Return remapping: a broad scan-bound change can grant newline/submit authority on historical prompts or unrelated screen rows. Review that shared safety surface explicitly; retain menu/footer/cursor/left-edge guards and add adversarial parser tests. A viewport-bounded qualified occupied reading is a candidate, not an approved design.

The owner can reproduce the preexisting class with this test in `cmd/internal/wrapcmd/` on main (uses existing fake/reducer seams, no live slots or prompts). It intentionally expects the required Submitted outcome and currently fails after recording an actual submit and an empty composer:

```go
package wrapcmd

import (
    "bytes"
    "fmt"
    "strings"
    "testing"
    "time"
    "github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestCodexTallComposerConfirmationRegression(t *testing.T) {
    now := time.Now()
    f := newHarnessSessionFake(t, "codex", true)
    defer f.close()
    f.resize(100, 60)
    m := peerTestMessage(now)
    m.Body = strings.TrimSuffix(strings.Repeat("report line\n", 25), "\n")
    d := newPeerDelivery(m.To, func() time.Time { return now })
    f.proxy.peer = d
    paint := func(text string) string {
        rows := strings.Split(text, "\n")
        var out strings.Builder
        out.WriteString("\x1b[2J\x1b[?2004h\x1b[5;1H\x1b[1m›\x1b[22m ")
        for i, line := range rows {
            fmt.Fprintf(&out, "\x1b[%d;3H%s", 5+i, line)
        }
        fmt.Fprintf(&out, "\x1b[?25h\x1b[%d;%dH", 5+len(rows)-1, 3+len(rows[len(rows)-1]))
        return out.String()
    }
    peerRender(f, d, paint(""))
    if err := d.reserve(m.ID, d.sequence); err != nil { t.Fatal(err) }
    if err := d.enqueue(m); err != nil { t.Fatal(err) }
    f.proxy.turnActive.Store(true)
    var input bytes.Buffer
    f.proxy.dispatchPeer(&input)
    paste := input.String()
    if paste == "" { t.Fatal("no paste") }
    peerRender(f, d, paint(peerEnvelope(m)))
    t.Logf("post-paste composer=%v", peerComposerState("codex", f.proxy.terminal.Snapshot()))
    now = now.Add(PeerSubmitDelay + time.Second)
    f.proxy.dispatchPeer(&input)
    if input.String() != paste+"\r" { t.Fatal("missing exactly one submit write") }
    peerRender(f, d, paint(""))
    f.proxy.dispatchPeer(&input)
    now = now.Add(couchmessage.DeliveryTimeout)
    f.proxy.dispatchPeer(&input)
    if got := d.receipt(); got.Status != couchmessage.Submitted {
        t.Fatalf("genuine clear after submit not confirmed: %+v", got)
    }
}
```

## Done when

- A deterministic tall-singleton regression and equivalent tall-batch fixture confirm the submit on positive evidence during an already active turn; short and idle controls still pass.
- Adversarial tests preserve refusal/uncertainty for historical prompts, menus/status rows, stale or partial frames, interrupted input and incomplete writes; Return-remapping behavior is checked alongside peer confirmation.
- A new owned Codex PTY with a representative tall paste provides native conformance evidence; no existing slot or clipboard is manipulated. Any live submission is explicit and harmless.
- No blanket timeout increase or retry, and no inference that Unknown proves a paste or that a write alone proves a submit.

## Plan

Durable design: [Codex tall composer](../plans/000652-codex-tall-composer-plan.md). One atomic close boundary. Await hub approval before implementation; Codex batch rollout remains disabled.

- [x] Pin complete tall geometry and adversarial Return/peer evidence in tests.
- [x] Share bounded positive recognition and verify actual writer confirmation without changing delivery ordering.
- [ ] Complete native qualification, documentation and recorded verification, then close and land through normal gates.

## Log

2026-10-17, discovered while validating #648. Claude Code 2.1.302 and Codex CLI0.163.0 retained measured large pastes exactly through external-editor export, so this is confirmation evidence, not a measured truncation limit. A fresh native Codex five-envelope paste rendered fully but was Unknown to the same production parser; narrowing the native fixture alone would hide this risk.

Actual writer/stateful fake diagnostic, four cases (all had one submit write):

| Case | Composer after paste | sawOccupied | After genuine empty | Final |
|---|---|---|---|---|
| 2 members, 7 lines, already busy | Occupied | true | Submitted | Submitted |
| 5 members, 32 lines, already busy | Unknown | false | Delivering | Indeterminate at 30 s |
| 5 members, 32 lines, initially idle | Unknown | false | Submitted via new turn | Submitted |
| Legacy singleton, 26 lines, already busy | Unknown | false | Delivering | Indeterminate at 30 s |

Detail: `uncertain: delivery deadline elapsed after the submit: submitted; composer empty but never showed the paste; a turn was already running (30s)`.

Local full diagnostic: `/tmp/pair648-peer-batch-viewport-diagnostic_test.go`; result `/tmp/pair648-viewport-diagnostic.log`; native failure `/tmp/pair648-wrapper-native-codex.log`. These scratch files are supplementary; the complete main-compatible singleton reproduction and result table above are durable. No production fix or new timeout/retry was made. The full batch diagnostic uses #648's new fixture helpers after that issue lands; the singleton defect and reproduction predate it.

TL authorization: verified submitted delivery `00eac401-7b14-4ee8-8255-ef5cc30e078e`, explicitly asks for a separate unclaimed pair issue and the Return-remapping safety note.


- Fresh submitted dispatch `886c2aa5-23ca-4be1-a058-8e6c43d316e9` confirms #648 landed and assigns #652, requiring plan before code and Return safety. Ran fresh `tl-worker --help`, claimed and published slot description. First `start-plan` refreshed ariadne and stopped before branching because the binary was stale; reran as directed, creating this fresh main-based branch and starting the issue. Inspected both consumers of `codexComposerActive`, existing status/left-edge guards, footer signature and sequence-based confirmation. No implementation code changed.

- Plan report `7abfda9d-f1f3-4dd9-9b2c-958bd982d252` is submitted. Verified fresh submitted hub approval `8acc22f6-c522-4333-bd5e-250e5ddb66c4` accepts the design/live arrangement and adds changed/unrecognized footer wording as a required unknown case. Before the owned PTY run, ask hub for the native lane shared with pair:1/pair:5; no focus stealing. The real operator Return/submit smoke rides the hub's next restart. Proceeding to full-flow implementation admission because this changes shared input authority; no production code changed yet. Refreshed `sdlc agent-id` reports `codex/-/-`, copied for new manual commits per its contract.

- Full-flow plan-quality round 1 found no architectural blocker but raised Important PQ-1: function-level test strategy must replace case inventories. Revised the durable plan with named generated-input/controlled-event strategies and independent guards for every risky function; kept tall acceptance regressions and native qualification. Clarified prompt-geometry projection preserves the distinction between existing short Return recognition and stricter peer text reading. No implementation code changed.

- `change-code --flow full` admitted implementation after plan-quality acceptance and the calibrated 1.753h estimate. Estimate-quality INFO calls the test/native allowances optimistic; active diagnosis/qualification/landing work remains measured, while only idle queue waiting is excluded. No estimate bypass.
- TDD: native-shaped snapshot tests fail beyond 20 rows and tall Return emits CR on main (`/tmp/pair652-tall-red.log`). Corrected an initial empty-screen fixture to include Codex's actual faint suggestion; the corrected red (`/tmp/pair652-tall-red-corrected.log`) then has short/idle controls passing and tall busy singleton/batch-shaped input failing after exactly one submit and a fresh genuine clear. No production fix preceded the red.
- Implemented one pure shared `codexComposerBounds`: preserve short recognition, require complete styled prompt/continuation/footer geometry for tall input, and expose its footer boundary to text reading. Moved the existing footer signature to the shared recognizer; no new allowlist, status, retry, timeout or rollout. Tall blank-first-line text is extracted exactly without including native footer; short extraction remains unchanged. Codex batching remains disabled.
- Generated heights 19/20/21/26/32/60, viewport sizes, blank/Unicode content and broken-authority inputs pass. Actual writer confirmation passes busy tall singleton and five-report batch-shaped text; controlled no-occupied, changed-footer, open-frame, stale-clear, failed/partial-write, human-input, picker and child-exit sequences never falsely submit or replay. Focused checks PASS 0.819s (`/tmp/pair652-focused.log`), selected race checks PASS 3.103s (`/tmp/pair652-race.log`). Negative controls: disabling the extension breaks tall positives/Return/confirmation; removing the footer predicate breaks changed-wording and trailing-footer negatives (`/tmp/pair652-{extension,footer}-mutant.log`); production source restored.
- Read-only native-harness exploration found fixed 120x38 geometry in both capture and classifier. Native qualification will parameterize that existing seam to 100x60, use a singleton body containing five envelopes, require exact visible tall text, then test LF plus a sentinel without allowing the pending automatic delivery to submit. A separate existing explicit-submit scenario carries the one approved harmless native model request. No native client was run; shared lane not yet requested or used.


- Native fixture TDD: the added geometry/probe tests failed before their capture/classifier seam existed (`/tmp/pair652-native-fixture-red.log`), then passed in 0.349s (`/tmp/pair652-native-fixture-green.log`). Both PTY and terminal classifier now accept the same explicit viewport, preserving the default. The tall probe requires exact text and at least 26 visible qualified rows, emits production Return once, then requires exact text plus newline/sentinel and exactly one additional row; collapsed/unknown/unchanged screens and repeated input cannot prove success.
- Fresh lane grant `8b4c204c-9ece-4106-83d1-d9f704ed9270` verified Submitted before native runs. Two sequential owned Codex 0.163.0 PTY checks passed: paste-tall at 100x60 retained the exact 556-byte singleton envelope, ordinary Return LF grew 33 to 34 composer rows, capture SHA256 `0de02f4461a1bc33f3dca0702992e8859e2d67152aced755a23f8cfc76ca5ae5`; separate explicit-submit confirmed Submitted for the approved harmless no-tools prompt, SHA256 `4a8793143487e6255c23acb7b2c4d13b404ebe11c2330e2259d17894ce4f17f8`. No response-token search is used as proof. Captures remain private in the owned scratch directory. Lane release report `4f8e6bdb-b0b1-4874-af2a-16ca77166c64` sent; no focus, clipboard or existing slot input changed.
- Exact native commands: `PAIR_LIVE_PEER_HARNESS=codex PAIR_LIVE_PEER_SCENARIO=paste-tall go test ./cmd/internal/wrapcmd -run '^TestPeerLiveConformance$' -count=1 -timeout=90s -v -peer-live-use-local-auth -peer-live-capture-dir=<private-absolute-dir>` (PASS 1.520s), then `PAIR_LIVE_PEER_HARNESS=codex go test ./cmd/internal/wrapcmd -run '^TestPeerLiveConformance$' -count=1 -timeout=90s -v -peer-live-use-local-auth -peer-live-submit -peer-live-capture-dir=<same-dir>` (PASS 1.567s).
- All wrapper tests PASS 71.867s (`/tmp/pair652-wrapcmd.log`); broader core/native-fixture race checks PASS 3.990s (`/tmp/pair652-all-race.log`). README and atlas now map the shared tall authority and unknown behavior, retaining Claude-only batching and deferred operator restart smoke. Preparing pushed code head for the required recorded suite.

- Lane release `4f8e6bdb-b0b1-4874-af2a-16ca77166c64` confirmed Submitted. Recorded pre-close suite attempt `8e42cbc6659b6298be33f6e03d89d05d` queued on pushed code head dbd1abb87. Fresh Submitted hub approval `35128107-2a46-492b-a515-ee6359537e3d` authorizes an early draft PR for macOS CI overlap; suite, close review and merge gates still precede landing.

- Early draft PR https://github.com/xianxu/pair/pull/416 opened with the verified exception; macOS conformance is running alongside the queued recorded suite. Normal close and merge gates remain required. Initial PR command refused a concurrently written Log edit before any publication; committed that documentation and reran normally.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration is flagged stale by `sdlc estimate-source`; treat this as provisional focused ship wall-clock, excluding test/landing queue waits.

Primitive decomposition after plan-quality acceptance: issue/spec uses base design 0.6×0.5 and implementation 0.2×0.4, including the current diagnosis/plan dialogue. Two smaller well-specced Go extensions (shared geometry/footer and the two consumers with stateful regressions) each use design 0.2×0.2 and implementation 0.5×0.4. The existing native-PTY integration extension uses design 1.0×0.2 and implementation 0.5×0.4, with one Codex discovery allowance 0.3×0.4. Atlas/README maintenance uses design 0.1×0.2 and implementation 0.1×0.4; the one close/review administration uses design 0.1×0.2 and implementation 0.5×0.4. Component implementation allowances include their own tests/debugging; the review primitive does not duplicate those. Library check: the existing terminal snapshot, footer predicate, input writer/fake and native PTY capture already provide the stack; no greenfield library needed. Familiarity 1.0 from #648's exact diagnosis and native capture. Thorough accepted plan: 15% design buffer. Design subtotal 0.62; implementation subtotal 1.04; total 0.62×1.15+1.04=1.753h.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.3 impl=0.08
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: api-integration design=0.2 impl=0.2
item: real-api-discovery design=0 impl=0.12
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 1.753
```

## Revisions

- 2026-10-17 15:36 PDT — Proposed design: preserve short-composer behavior, admit taller geometry only with a complete visible prompt/continuation block and positive trailing native footer. The initial no-footer diagnostic above remains historical evidence of the cutoff, but cannot itself authorize broader Return remapping; the accepted regression will include the native footer and remain red on main. Codex batching stays disabled; test batch-shaped text through ordinary singleton delivery, with no rollout-guard bypass. Awaiting hub approval.
