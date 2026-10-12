---
id: 000466
status: codecomplete
deps: [000457]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '483b4de832fd2061175714a920852d2d69eef02c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T00:53:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.92
---

# broadcast-web: remote-driving contract — viewer input, driver control, attribution in the format

## Problem

Remote driving is in couch-broadcast-web's MVP for internal viewers (the operator, 10-12): a product manager uses an app hosted by another operator, for product design and live debugging. Outside users (research participants, customers) stay later, with the privacy work. pair#457's format and seams are one-way (operator's Chrome → viewers): no reverse channel, no input injection, no attribution of who acted. This issue extends the contract before the components that need it are built (ops `teams/devinfra/projects/couch-broadcast-web/project.md`, Later requirements: remote driving, now MVP).

## Spec

Design and contract only, as an additive v1 revision of `atlas/webcast-format.md` and #457's Go seams (no component implementation):
- **Viewer → Couch input messages:** element-addressed (node id) plus page coordinates for pointer actions (click, double click, pointer down/up/move while dragging, wheel/scroll), and keyboard input (text, keys, composition) to the focused element; scoped to the page: never browser UI (address bar, downloads, file pickers, devtools, new windows, permission prompts).
- **Driver control:** one driver at a time; a viewer requests control, the operator grants (and can revoke at any time); the driver gives it back. Who may request: anyone with the viewer link can watch; driving needs the operator's grant per viewer.
- **Attribution in the recording:** context records (`click`, `input`, `focus`, `hover`, …) gain a source (operator or viewer id), and control changes are records, so an agent can tell a participant's actions from the operator's.
- **Seams:** an input-injection interface on the browser-control side (`webcast`, CDP `Input.dispatch*` into the cast tab) with a fake; the reverse channel on the live transport (viewer shell → Couch; the recorded page stays script-free).
- Security notes: injected input is confined to the cast tab's page; the operator sees who is driving in Couch and in the page.

## Done when

- The format page and seams carry the additions, with validator/reducer support and fakes; fixtures show a driven segment.
- pair#458/#460 and the remote-driving build issue can implement against it without asking.
- The TL reviewed it (the operator if a decision is theirs).

## Plan

Durable plan: `workshop/plans/000466-broadcast-web-remote-driving-contract-viewer-input-driver-control-attribution-in-the-format-plan.md` (single pass).

- [x] `recording`: `Source`/`by`, `ctl` + `NextControl`, reducer attribution check, catch-up keeps the latest `ctl`
- [x] `driven` fixture
- [x] `drive`: `Batch`/`Act` wire + `ParseBatch`, `Confine` policy
- [x] `drive.Injector` + `FakeInjector` + `RunInjectorContract`
- [x] `drive.Controller` + `FakeController` + `RunControllerContract`
- [x] atlas (format page, remote-driving section, components rows); TL review

## Log

### 2026-10-12
- 2026-10-12: closed — post-close delta is the close advisory only: policy.go doc comment and atlas restate the confinement rule as defended (effects outside the document; Escape reviewed harmless); no code change, go build + drive tests green (fixed-then-ship); review verdict: SHIP
- 2026-10-12: closed — go test ./... green unsandboxed before review; make -k test only failure artifactpath inventory, fixed c4e3747c; rounds 1-3 fixed (BR-1..3, BR-9, BR-11): injector follows the stream with typed ErrNotDriver; Confine is a closed chord table, activation keys take no modifier, test compares with a hand-written chord set plus cited Chrome/Blink bindings (mutation-checked); no seg/ctl after stop end; webcast/... broadcast artifactpath green; grace-timer injection advisory left (plan Revisions); TL approved (Done when); review verdict: SHIP
- 2026-10-12: flow upgraded quick → full — 1678 added lines in code files (limit 100); an earlier round of this close already ran the full review
- Early asks sent to the TL and agreed (by on interaction kinds incl. #458's hover/ptr; ctl carries post-state; routes; refusals). pair:1 correction: the single writer is the recorder's queue, the id map is `Recorder.NodeOf`/`IDOf`; `input {secret:true}` for passwords.
- TL approved the plan; asked that Revoke and leave-on-grace be unconditional: RunControllerContract checks Revoke returns while a driver's batch is mid-dispatch, and a gone driver leaves on grace (both mutation-checked).
- Discovery: a late joiner's catch-up starts at the latest `ctl`, with no prior state to check its transition against. Rule: the first `ctl` a reader sees is its baseline (consistent in itself); later ones must equal `NextControl(prev)`; `Validate` (whole recording) stays strict from the first. Tested both ways.
- `drive.Answer`, `SeatHeader`, `DefaultGrace` added so the atlas's reverse-channel answers are code, not prose (#463's lesson).
- Side change: `broadcast.decodeStrict` → `DecodeStrict`, shared by both reverse channels (ARCH-DRY).
- Close review round 1 (FIX-THEN-SHIP): fixed BR-1 (injector follows the stream, typed ErrNotDriver, revoke step + composed fakes), BR-2/BR-3 (Confine: Meta+Y, Shift+Escape, Code), and minors (empty driver id, stop end final, ime sel, label chars, driver-note coverage). Grace-timer injection left advisory (see plan Revisions).
- Close review round 2: BR-9 Shift+Delete (Cut). Fixed the rule, not the instance: Confine keys are a closed chord table (key x exact modifiers) with a test walking every key x 16 modifier sets plus a list of known Chrome/Blink bindings. Also: no seg or ctl after stop end (one guard in State.apply).
- Close review round 3: BR-11 activation keys (Enter/Space) with modifiers are keyboard twins of refused modified clicks; the chord test was tautological. Fixed the rule: activation keys take no modifier; the test compares Confine with an independent hand-written chord set plus cited Chrome/Blink bindings (mutation-checked).
