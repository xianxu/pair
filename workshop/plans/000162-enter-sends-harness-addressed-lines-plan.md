---
collaboration-mode: discussion
---

# Native Return for agent slash commands

> **For agentic workers:** Consult AGENTS.md §3. This is one atomic boundary: composer Return and Couch's slash-producing callers must agree. No implementation before hub approval and change-code.

**Goal:** Plain Return submits a positively proved single-line slash command in the agent composer; ordinary text keeps its newline mapping, and Couch's slash automation uses that same native Return.

**Architecture:** Extend existing composer recognition with an optional complete text-region witness while preserving its current activity result. One pure slash grammar serves operator Return and the existing agent-drive reducer. Known automated slash sends wait for exact current echo, bind that proof to Enter, and use the same immutable frame for the final proof and Return decision. Keep the existing single input owner/writer, deadlines and cancellation.

**Tech stack:** Go terminal snapshots, existing profile/recognizer registry, agentdrive pure reducer, fixture replay, stateful drive/terminal fakes, disposable native conformance.

## Accepted scope and alternatives

Authenticated ops:0 scope confirmation1f501f4a supersedes the September extras: slash-only agent-composer behavior plus Couch slash sends. Draft-pane Return stays unchanged. Configurable ! sigils and PAIR_WRAP_COMMAND_ENTER are deferred. Existing PAIR_WRAP_REMAP_RETURN=0 and unconditional Alt+Return retain their contracts.

1. **Recommended: shared recognition evidence plus proved command Enter.** Reuse the geometry and current drive owner; preserve active/unknown distinctions and require exact echo for automatic sends.
2. **Starts-with-slash on a screen row, then swap submit to enter.** Reject: submits path/prose or a partial multiline capture, and an old complete empty frame can remap automated Enter to newline before its text renders.
3. **Parallel slash recognizer or timed waits in callers.** Reject: duplicates geometry or treats time as proof. An unrelated timer repaint is not command echo; no new delay/deadline tuning.

No general terminal parser, text editor, public command flag, ownership weakening, or broader model-cleanup authority is proposed. Public `submit` continues to mean Alt+Return; concrete slash producers and examples switch to `enter`.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| ComposerRecognition / ComposerRegion | cmd/internal/wrapcmd/composer_input.go; composer_recognizers.go | new / modified |
| SlashCommand | cmd/internal/agentdrive/command.go | new |
| CommandExpectation / command proof | cmd/internal/agentdrive/command.go; model.go | new |
| DriveState / DriveEffect | cmd/internal/agentdrive/model.go | modified |
| returnDecision | cmd/internal/wrapcmd/harness_tty.go | modified |

Recognition separates **Active** from an optional fully located input region. Existing activity recognizers retain their present boolean semantics as wrappers/consumers of Active. The registry owns the richer locator used by Return. Unknown text never changes an active composer into an inactive one: it retains today's newline mapping. Only a complete coherent frame and positively located entire command region can enable the new bypass. Claude shell/unknown prompt modes do not become slash-command proof. No per-profile declaration of speculative sigils is added.

Reuse codexComposerBounds and ruledBoxComposerBounds. Extract the existing qualifiedComposerText row projection and placeholder handling rather than making a second text reader. Expose existing Grok/Agy geometry from their current scans; Muse uses the shared ruled-box machinery. Preserve the six profiles' activity qualification. Peer-specific startup, default-mode, command-draft and attachment vetoes remain separate and unchanged; sharing geometry must not widen ordinary peer delivery. Ambiguous bounds, missing cells, partial frames and unknown mode produce no command text witness.

`SlashCommand(text)` requires one observed line with no CR/LF and `^/[A-Za-z][A-Za-z0-9_-]*([ \t].*)?$`. Leading user whitespace is significant; trim terminal right-padding only. `/Users/x/...`, `/ `, `@file`, ! commands, empty input, actual second/blank cursor lines, pasted multiline text and faint suggestions are negatives. Multiple rendered rows conservatively fail the single-line proof, including indistinguishable soft wraps; Alt+Return remains available. Do not strip continuation text or guess a missing prefix.

`CommandExpectation` is a bounded comparison target derived from this drive's authored steps, not screen authority. The initial positively empty admission, optional clear-line, contiguous text chunks and wait steps describe the supported raw slash-producing pattern; cursor/edit/control operations invalidate prediction. Do not build an editor emulator or claim coverage of arbitrary edited raw sequences. The structured model recipe has the exact authored `/model` target. All current concrete production producers are in these supported families. Unknown prediction keeps explicit existing key semantics; it never grants a slash proof.

Before emitting a known slash `enter`, the pure reducer observes until a complete, positively recognized, owned composer equals the exact expected command at an epoch newer than its text-write fence. Partial prefixes, another command, leading-space/multiline prose and newer empty timer frames emit no Return. Use one existing observation/wait lifecycle with an explicit pending-command phase/continuation, not a second driver. A valid observed command creates an Enter effect bound to that exact text/epoch. The last write gate re-proves current ownership and exact text; identical command repaint can remain valid, changed input refuses before bytes.

Keep `ReadModelScreen` recovery scoped to exact `/model`. Publishing general command evidence for Return must not authorize model cleanup to clear `/usage`, arbitrary prose or unknown input.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Profile registration and Return emission | wrapcmd/harness_tty.go; wrap.go | modified | terminal snapshots, telemetry and lifecycle |
| Current drive observation and sole writer | wrapcmd/agent_drive.go | modified | inputAdmission, peer mailbox, terminal model, child writer |
| Structured slash producer | agentdrive/model.go | modified | existing clear-line/text/submit recipe |
| Public documentation/help | agentdrive/keys.go; couchcmd/agent_drive_cli.go; couchcmd/skills/couch/SKILL.md; README.md; atlas/couch.md | modified | generated key meanings and slash examples |

The final drive command proof and Return decision consume **the same captured immutable frame**, with the existing input ownership gate. Avoid a second snapshot that can convert a proved command into newline. Extract the emitter's decision/application seam as needed; keep one decision implementation, one lifecycle/telemetry application and the existing writer. No proxy-side cached command flag authorizes output.

## Behavior and recovery table

Actions: press Return, press Alt+Return, inspect, cancel, explicitly retry after inspection.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Active composer, complete single-line slash proof | Native command Return | inspect native result | Native command may report unknown command |
| Active prose, leading space/path or multiline | Existing newline mapping | press Alt+Return when ready | No automatic submission |
| Active composer, text region incomplete/unknown | Existing newline mapping | inspect | Shortcut unavailable; input retained |
| Existing picker/overlay or inactive composer | Existing native Return behavior | inspect | Existing native selection semantics |
| Automatic known slash text not yet echoed | Existing drive in progress; no Return written | cancel | Authored text can remain in composer |
| Echo never becomes exact before existing deadline | Existing failed/unconfirmed receipt | inspect before explicit retry | Input progress may be uncertain; no blind replay |
| Human input/cancel or proof changes before write | Existing canceled/refused/unconfirmed outcome | inspect | Already accepted bytes remain; no cleanup over new input |
| Lost reply | Existing retained operation status | inspect | Unknown outcome; never automatic resubmit |

Covered: all six current profile recognizers through fixtures; real Claude/Codex and Qoder when credits permit; known structured and public slash-producing families. Not covered: a general raw-key editor prediction, speculative sigils, draft-pane changes, unknown agent UI grammar. Those do not receive new submission authority.

## Caller census and complete migration

- `agentdrive/model.go:commandNext`: clear-line, text:/model, **enter**, coupled to exact-echo proof. It is the sole production literal slash sequence found.
- Public raw `couch --keys`: enter shares operator Return; known slash text sequences receive the reducer echo fence. Preserve named `submit` as Alt+Return for arbitrary prose and compatibility.
- Change /usage and /model examples in CLI help, embedded Couch skill, README and atlas to enter. Update generated `enter` meaning to distinguish slash command / ordinary composer / picker. Change matching help tests and the raw slash conformance example.
- Preserve unsent /status and /model recovery drafts in tests. A typed command without a Return is not a producer to turn into a submission.
- Peer delivery already writes native keymap.altCR directly after a bracketed prose envelope. Orientation/recovery prompts are prose; retain their direct native submission. Keep arbitrary prose and folder-trust probes on their current paths.
- No dedicated /clear, /new, /usage or /status sender exists beyond generic public keys/examples in the current census. Re-run the source sweep at implementation and add any newly found producer in this same change.

## Bounds and architecture

ARCH-DRY/PURE: one grammar, one geometry owner, one Return decision, one drive proof/effect owner; colocated pure tests. ARCH-PURPOSE: caller sweep includes automated echo ordering, not just the visible keybinding. ARCH-MOCK: reuse harnessSessionFake/driveFixture and virtual-terminal conformance, with delayed paints and fake time; no function-call-only claim of native behavior. ARCH-SECURE: unknown text never grants submission; human ownership and effect veto stay mandatory, peer/model cleanup is not broadened.

ARCH-CONSTRAINTS/ORDER: no new deadline, retry worker or key replay. Existing 64 steps / 8 KiB aggregate text / wait caps, 30s admission and 36s CLI observation remain. Expectation storage is at most the existing 8 KiB authored text, only one active drive. For Return, one locator and at most one bounded text projection per decision, O(screen cells); single-line grammar O(line bytes). Reuse terminal dimension/cell limits. Count locator/projection calls over32/64 Returns and inspected cells over64x32/128x64 fixtures; reject repeated-per-row rescans. Derive a raw expectation once at its Enter boundary from the bounded command span; counted8/16 command groups (32/64 steps) reject rescanning an ever-growing prefix.

ARCH-FUNERAL: recognition is ephemeral; expectations/effects die with the existing drive and bounded receipt retention (eight wrapper receipts, five minutes, removed on shutdown). No new file family or goroutine. Existing test capture owners join their child/output and remove disposable roots. ARCH-EVOLVE: no persisted/request format or meaning change; in-process evidence additions are explicit and ordinary submit semantics stay stable.

## Implementation and verification (after approval)

- [ ] RED then GREEN `SlashCommand` in agentdrive/command.go: fuzz arbitrary UTF-8/control and token-shaped text against an independent one-line ASCII command oracle; a false result never creates native submission authority.
- [ ] RED then GREEN `recognizeComposerInput` / `projectComposerRegion` in wrapcmd/composer_input.go: mutate captured geometry/styles/cursor/completeness across the six profiles; compare Active with the legacy boolean oracle and require full coherent input coverage before producing a text witness. Count one locator/projection per Return and bounded inspected cells at the sizes above.
- [ ] RED then GREEN `decidePlainReturn` and the `emitPlainCR` decision/application seam: property-test classified input through legacy/KKP/chunked/paste translation; independent byte and turn-opening invariants require native command bytes only for positive slash proof, preserving existing mappings otherwise. Use the real translator and fixture replay.
- [ ] RED then GREEN `commandExpectation` in agentdrive/command.go: generate bounded authored key sequences and compare with an independent supported-span oracle; edits invalidate prediction, and counted8/16 command groups reject prefix rescans. The expectation is never observed-screen authority.
- [ ] RED then GREEN `ReduceDrive`: generate screen/write/tick/cancel sequences with fake time and independent invariants: no command Enter before exact fresh echo, at most one emitted Enter per authored boundary, no writes after terminal/cancel/deadline, and no cleanup authority outside exact owned /model. Reuse driveFixture and model tests.
- [ ] RED then GREEN `DriveEffect.MatchesScreen` / `effectMatchesScreen` and `dispatchAgentDrive`: mutate proof epochs, text, ownership and frame completeness in the existing stateful writer fake; assert changed proof writes zero bytes and command proof/Return decision consume the same immutable frame. Withhold echo in virtual-terminal conformance, then remove the echo gate as a mutation and require failure. Paint command echo before accepting Enter in existing picker fakes.
- [ ] Migrate every slash producer/help example above; keep generic submit, direct peer/prose writes and unsent draft tests. Add a narrow catalog rule under the existing Go lint enforcement if the concrete literal slash-plus-KeySubmit class is mechanically enumerable, citing162; avoid heuristic banning all submit calls.
- [ ] Run `go test ./cmd/internal/agentdrive ./cmd/internal/wrapcmd ./cmd/internal/couchcmd -run 'Test(DecidePlainReturn|HarnessTTY|.*Composer.*|.*Slash.*|.*Command.*|.*Drive.*|.*KeysHelp.*)' -count=1`, then complete affected Go packages and repolint/termcmd/artifactpath/inventory preflight. Refresh only audited reader certificates.
- [ ] Reuse existing isolated real-agent TTY/drive capture for Claude/Codex (Qoder with credits), plus shared disposable testcouch public-key conformance. Assert slash native Return, prose/path/multiline newline, unchanged Alt+Return/picker behavior and successful /model selection/cancel. Retain the real command/negative captures for normal replay. Keep existing capture, drive, frame/output and cleanup bounds. No new private Couch fixture. Ask the hub to select any required operator stage/live smoke before landing; no :0 effect is authorized by this plan.
- [ ] Update atlas/bring-up guide and record coverage gaps explicitly; run the required recorded affected check (full fallback when selected), mandatory close review and `sdlc merge --yes`. No manual integration churn or gate bypass.

Plan is proposed to the hub; full flow is expected because the activity/text distinction and automatic command proof cross the recognizer, reducer and writer. Derive estimate only after change-code's plan-quality pass. No production change or native run has begun for162.


## Revisions

### 2026-10-22 — PQ-1 function-specific verification strategy

Plan-quality accepted the design but required replacing the test-case inventory with named risky functions, adversarial input classes and mechanical guards (ARCH-PURE, ARCH-ORDER). The checklist above now names the shared grammar, locator/projector, expectation derivation, Return decision/emitter, reducer and final proof/writer. Generated sequence invariants and independent oracles supplement fixture replay; counted-work and withheld-echo mutation checks remain mandatory. Product scope, ownership, deadlines and accepted behavior are unchanged.


### 2026-10-22 — Exact command effects retain control lifecycle semantics

Public native qualification showed a proved raw slash Return generating synthetic conversational submission, canceling its own drive. Authenticated ops:0 delivery7e32642b approves marking only that exact-echo-bound Enter effect Control=true, as structured /model already does. Native turn observations, complete composer/proof, current ownership and all write vetoes remain mandatory. Ordinary public submit retains its existing conversational semantics. Add a regression through the lifecycle reducer and raw drive writer before the correction, then qualify each disposable personality once. No new state, deadline, retry or protocol. Claude native qualification uses only /status in the already-trusted checkout with safe mode/tools off, per delivery2282c209; Codex uses the existing settled production-drive owner. Preserve measured failures and distinguish command-native acceptance from synthetic negative fixtures.


### 2026-10-22 — Public test readiness uses current wrapper evidence

Authenticated ops:0 delivery651ebb9f approves the shared testcouch read-only ObserveDrive helper over the real public Actors binding and owned namespace. Cached resources have a60s cadence and cannot gate successive admission inside a120s fixture lifetime. Poll the existing current wrapper Idle proof, then issue each public keys command once; input admission remains authoritative. Keep cached resources and live Cursor evidence only as bounded diagnostics. Test-only process isolation must select the canonical owned root explicitly, with helper refusal on missing/foreign selection; no new RPC, credential fabrication, production state, deadline or cadence change. If this third readiness design still cannot qualify, stop and decide the native gap with the hub rather than add another patch or retry.


### 2026-10-22 — Authorized overnight acceptance hold

Authenticated ops:0 deliveriesc207f32b and6fca9a0a hold Qoder qualification until after07:00 Pacific2026-10-23 (credits available, cheap-window restriction). Close also waits for ariadne:3 to diagnose/restore the named #151 measurement evidence; no actual-hours bypass. Keep the existing162 recorded-suite and806 merge runners, start nothing more tonight. Landing still requires Qoder pass or a separately approved documented conditional gap, the recorded suite and mandatory close review. Product behavior and verification bounds are unchanged.
