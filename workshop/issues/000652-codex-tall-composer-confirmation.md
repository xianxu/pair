---
id: 000652
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5461880772bcf05cad9711020d01dac989b2a2a9' # card fields mirrored from issue-cards; edit via sdlc
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

Unclaimed follow-up, per the TL's explicit #648 scope revision. Diagnose/plan before code; #648 initially enables only Claude hubs. Codex remains singleton until its own rollout can be qualified.

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
