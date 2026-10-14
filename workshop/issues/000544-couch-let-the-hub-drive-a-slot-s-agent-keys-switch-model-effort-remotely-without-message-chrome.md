---
id: 000544
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-14
estimate_hours: 6.29
card_mirror: '62ab2b732b0e7595e1daa0f1b18b04fcc5606d48' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T01:55:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# couch: let the hub drive a slot's agent keys (switch model/effort remotely), without message chrome

## Problem

The operator wants the hub (TL) to switch a slot's model and effort remotely. Every agent does it through an interactive picker (`/model`, then arrow keys and Enter; Codex adds an effort step; Claude and others differ). `couch --send-to` can't do it: it delivers a prompt wrapped in peer-message chrome ("[Couch peer from …]"), and can't send arrow keys. `couch --switch-agent` changes the agent, not its model.

A manual drive of idle pair:6 (Codex) on 2026-10-13 worked, from the hub, with zellij and `couch --peek`:
- Target: the slot's zellij session (`ZELLIJ_SESSION_NAME` from the wrapper process's environment; wrapper PID in `pair-wrap-pid-<tag>`) and the agent pane (`pane_id` in `pane-<tag>-<agent>.json` → `terminal_<id>`).
- `zellij --session S action write-chars --pane-id terminal_N '/model'` typed into the agent's input box.
- **Plain Enter (CR) inserts a newline in the agent pane** (Pair's deliberate mapping, pair#162); **Alt+Enter (ESC CR) submits**. Inside the agent's picker, plain Enter selects.
- Arrows `ESC [ A` / `ESC [ B` moved the picker; a lone ESC (sent separately) backs out; Ctrl+U clears the input line.
- `couch --peek` between steps showed the picker state (the highlighted row is reverse video), so each step can be verified.

## Spec

Two layers, hub-only, idle slots only (refuse when the agent is busy or the agent’s composer has unsent text):
1. **`couch --keys repo:N KEYSPEC…`**: inject named keys into the slot's agent pane, with no chrome: `text:…`, `enter` (plain CR), `submit` (Alt+Enter), `up`, `down`, `esc`, `clear-line`, `wait:MS`. Couch resolves the session and pane itself (no env scraping), writes through the slot's terminal, and returns the agent pane's tail after the last key, like `--peek`. One request, one bounded sequence; no retries.
2. **`couch --set-model repo:N --model NAME [--effort LEVEL] --confirm`**: per-agent recipes built on (1) that open the picker, find the row by its visible label in the peeked screen (not by position), select it, handle the effort step where the agent has one, and verify the result from the agent's own status line (e.g. Codex's footer `GPT-6-Astra medium`). If the label isn't found, back out with ESC and report the visible choices; never guess.

Agents' pickers differ and change across versions; recipes live in one table per agent, tested against recorded screens. The slot-continuity target's spirit applies: one action, an honest result, no limbo (a failed drive backs out to the agent's prompt).

## Done when

- `couch --keys` drives an idle scratch slot's agent pane (text, submit, arrows, esc) and returns the tail; it refuses busy slots and non-hub callers.
- `couch --set-model` switches a Codex slot's model and effort and a Claude slot's model, verified from each agent's status line, with recorded-screen tests per agent; an unknown model backs out cleanly and lists the choices.
- The couch skill documents both, and the hub's TL skill notes the recipe.
- `--set-model` defaults to session-only scope, selects shared defaults only with explicit scope, and reports the observed model/effort/scope.
- Model-control confirmation lines do not generate false “waiting on a prompt” notices; actual questions and pickers still do.
- Exact caller/slot ownership and wrapper identity are revalidated; busy/unknown or occupied agent composers refuse, and failures never automatically replay uncertain keys.

## Plan

Durable implementation plan: [agent keys and model selection](../plans/000544-agent-keys-model-plan.md).

- [x] Implement typed bounded key driving and verified per-agent model recipes through the exact-bound wrapper.
- [x] Wire hub ownership/idle guards, receipts and model-control submission accounting; prevent false prompt waits.
- [ ] Verify recorded screens, stateful socket/PTY sequences and isolated live conformance; update docs, close and land.

## Log

### 2026-10-14 — implementation and conformance checkpoint

- Implementation HEAD ac8de5d4cc974bd461a4f6d40dfeefa7b610d70e built cleanly; bin/couch reported that exact revision with vcs.modified=false. Typed input uses the existing writer, exact binding/session/ownership guards, coherent current-viewport evidence and bounded retained outcomes. No credential plumbing or ops TL skill edit.
- Ad-hoc fresh-eyes review found stale cached picker actions, decision-time confirmation fences, Return-remap-disabled submission accounting and mixed terminal snapshots. Regression tests reproduced each class; fixes bind semantic picker proof to an atomic terminal generation, fence after actual emission, retain normal raw-input accounting, and separate visible viewport authority from scrollback evidence. Operator interruption is proved through the real input loop and a blocked context-aware writer.
- Real isolated Codex0.162.1 new-driver conformance passed all three cases (15.615s): session selection confirmed in five writes with disposable defaults unchanged; explicit default confirmed in five writes with only disposable defaults changed; unsupported effort recovered to a fresh empty composer in six writes with defaults unchanged. Native configuration bytes stayed unchanged in every case; model controls produced lifecycle generation0 and no submission events. Live session race check passed6.335s; screen fuzz passed37,570 executions.
- Recorded Claude2.1.299 evidence includes the combined picker, left/right effort, s=session, successful “Set model to Opus 5.5 for this session only” line and medium footer. Claude default scope refuses before input because default success evidence is absent. Actual new-driver Claude conformance remains pending with the hub/operator.
- Full affected packages passed: wrapcmd136.278s, couchcmd51.915s, couchmessage7.240s, agentdrive0.160s and artifactpath8.928s; prior unchanged couchcore214.905s, couchtty36.808s and ttyio5.773s also passed. Root focused wrapper/drive/socket races and docs/cost checks passed. Initial broad verification exposed master-pump fixtures missing the newly required transport; six direct fixture sites now initialize it and affected races pass. An initial full make -k run compiled during the viewport API change and failed only those build targets; final stable-head full make -k is queued at the native test gate. No full-suite success claim yet.
- TL receipt097a24f1-6418-4492-806e-079efcafced9 verified: operator-owned restarts and live Claude check are deferred to morning, after pair#528 M1 acceptance. Hold before close. A stage/build started before that instruction was received; it restarted no process, was cleared back to main, and main was rebuilt. #549 PR328 continues to await the operator’s protected-path label.

## Revisions

- 2026-10-14 02:15 — add (TL, from the first fleet-wide remote switch): (1) `--set-model` takes an explicit scope:
  Codex offers "session only" (`s`) vs "save as default" (Enter); default to session-only, since Codex's default is
  shared by every Codex slot. (2) After a remote switch, the agent's own confirmation line ("Model changed to … for
  this session only") must not trigger Couch's 10-minute "waiting on a prompt" notice: three false notices came from
  ariadne:5, parley.nvim:1 and others on 10-14. (3) The verified recipe used (Codex 0.x): Ctrl+U, `/model`,
  Alt+Enter (plain Enter is a newline in Pair's agent pane), ↑/↓ until the reverse-video row shows the label,
  Enter, ↓ to the effort label, `s`; confirm from the footer `GPT-6.1-Sol medium`. Skip slots whose screen shows
  "esc to interrupt" (mid-turn).

- 2026-10-14 — TL clarified the idle precondition: only the agent’s composer must be empty; the separate Neovim draft is unaffected and needs no guard. Existing wrapper Settled is the required evidence. Spec wording updated accordingly.
- 2026-10-14 — planning survey: exact-bound wrapper endpoint, styled terminal tail and single input writer already exist. Peer delivery adds chrome and can run during turns, so reuse its identity/transport and input ownership seams, not its delivery policy. No versioned model/effort/scope captures exist yet; gathering those in isolated scratch PTYs is the first execution step after approval. #549 PR #328 awaits the operator label and is not the base of this independent branch.

- 2026-10-14 — durable plan ready for TL approval. Shared wrapper transport/input lease and pure screen-driven reducer provide one driver; no implementation or estimate yet. Added scope and false-wait acceptance to Done when, matching the existing Revision. All outstanding design uncertainty about actual picker shapes is handled by the first isolated capture step and a stop/revision on mismatch.

- 2026-10-14 — plan approved by TL at 44e2e723. Full-flow change-code round 1 requested precise registered-wrapper lifetime, isolated Claude model auth (existing native-HOME fallback is unsuitable), and named-function test strategies. Added an active refinement for each; retained the approved feature/bounds. TL will update ops/skills-draft/xx-tl/SKILL.md after landing; this issue updates Pair docs only.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale: provisional ship wall-clock, not a deadline.

Decomposition: shaping 0.7 design / 0.2 implementation; four pure concerns (keys, recipe, screen evidence, reducer) each 0.8 / 0.6; wrapper protocol/input integration and Couch authority/receipt integration each 2.0 / 1.5; CLI, receipt extension and wait classifier each 0.3 / 0.5; docs 0.1 / 0.2; close review 0 / 0.5; Codex and Claude live discovery each 0 / 0.6. Library availability: existing PTY, terminal rendering, translator, transport and receipt seams are reused; no external library substitutes for versioned picker interpretation and domain transitions. No novel stack multiplier (1.0). Approved plan discounts implementation-phase design by 0.2; shaping is undiscounted. Implementation values below are already scaled by 0.40. Design subtotal 2.34, implementation 3.60, design buffer 15%: 2.34 × 1.15 + 3.60 = 6.291, rounded 6.29h. Tests/conformance are included in component allowances and two discovery primitives; no separate test surcharge.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.70 impl=0.08
item: greenfield-go-module design=0.16 impl=0.24
item: greenfield-go-module design=0.16 impl=0.24
item: greenfield-go-module design=0.16 impl=0.24
item: greenfield-go-module design=0.16 impl=0.24
item: api-integration design=0.40 impl=0.60
item: api-integration design=0.40 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 6.29
```

- 2026-10-14 — plan-quality round 2 CLEAN, all three findings addressed. Derived estimate after approval/gate; no implementation yet.

- Capture checkpoint: Codex0.162.1 model picker uses reverse-video selected numbered row; effort picker explicitly offers Enter=default and s=session. Session selection yields fresh model/effort footer plus “for this session only”; disposable default hash unchanged. Claude private credential copy lacks usable tokens and is expired; isolated composer says Not logged in. No native writable-HOME/config fallback used. Asked TL for an isolation-compatible credential-source Revision; raw-key/Codex core can continue independently. Estimate review INFO: auth helper lives in wrapper integration, final checks/CI/landing span integration and review allowances, which may be optimistic.

- 2026-10-14 — TL provided read-only real Claude picker/recovery captures and directed no credential plumbing/Keychain reuse. Observed version2.1.299; Claude combines model/effort and uses left/right for effort, s=session. Plan Revision replaces isolated Claude auth/conformance with recorded evidence and hub-mediated idle-slot live checks. Requested the missing successful session-apply confirmation from the hub; no Claude success claim yet.


### 2026-10-14 09:57 UTC — captured recipe and implementation-seam revision

**Reason:** Captures established the real phases and scope controls; implementation inspection identified existing bounded writer/transport cancellation seams. The durable plan's matching revision is the active delta; earlier text is retained for provenance.

**Delta:** Codex Advanced Reasoning adds a nested Max/Ultra picker; permit at most three separate recognized ESC steps for Advanced → reasoning → model → composer, while retaining the absolute budget and fresh recovery proof. Disposable captures verify both session-only unchanged defaults and explicit default persistence. Hub-supplied Claude 2.1.299 evidence now confirms session apply; Claude default scope remains unsupported before input. No new Claude auth helper or Keychain reuse.

Reuse `ttyio.Writer.WriteContext` and context-aware `couchmessage.Call`; jobs remain owned by console/wrapper lifetimes and the admission deadline, independently of short RPC exchanges (ARCH-DRY, ARCH-FUNERAL). Picker-derived writes carry the atomic Feed generation plus parsed screen proof. A changed generation permits input only when fresh complete picker evidence is identical in phase, selection, labels, model, effort and footer; changed recognized evidence refuses the cached action. Unknown/incomplete screens and initial/raw effects wait without keys within the absolute budget (ARCH-ORDER). Unknown or uncertain effects never replay, and human cancellation writes no cleanup over a new draft. README, atlas, the embedded Couch skill, verb cost labels and the ten-file artifact inventory now describe this bounded control surface; the ops TL skill remains hub-owned.

### 2026-10-14 — viewport authority correction from live conformance

Live-wrapper regressions showed that retained scrollback can contain a former picker after the current viewport changes, or duplicate headings after an identical repaint. Picker decisions and effect revalidation therefore read only the coherent current viewport; retained history is separate evidence for result tails and fresh confirmation occurrence counts. Unknown current views authorize no input. This preserves the original no-input-on-an-unknown-screen intent without refusing harmless identical repainting (ARCH-ORDER).
