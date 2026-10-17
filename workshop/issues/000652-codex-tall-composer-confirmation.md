---
id: 000652
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9525b2b8b63c13a87c1a4181147d9ab112b979b4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T15:33:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
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

- [ ] Pin complete tall geometry and adversarial Return/peer evidence in tests.
- [ ] Share bounded positive recognition and verify actual writer confirmation without changing delivery ordering.
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

## Revisions

- 2026-10-17 15:36 PDT — Proposed design: preserve short-composer behavior, admit taller geometry only with a complete visible prompt/continuation block and positive trailing native footer. The initial no-footer diagnostic above remains historical evidence of the cutoff, but cannot itself authorize broader Return remapping; the accepted regression will include the native footer and remain red on main. Codex batching stays disabled; test batch-shaped text through ordinary singleton delivery, with no rollout-guard bypass. Awaiting hub approval.
