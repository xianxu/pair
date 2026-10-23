---
id: 000162
status: working
deps: []
github_issue:
created: 2026-09-01
updated: 2026-10-22
estimate_hours: 1.97
card_mirror: '9687db047473023bdc933d8d10c3368591b717aa' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T18:09:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Enter sends when the composer line is addressed to the harness

## Problem

In the **agent pane** — the Claude/Codex tty itself, not the nvim draft —
sending `/login` costs Alt+Enter. So does `/clear`, `/model opus`, `/context`,
`/resume`. Natively, every one of these agents submits on plain Enter; Pair is
what took that away.

That was deliberate. `cmd/internal/wrapcmd/wrap.go:127-143` documents it: the
draft uses Enter = newline / Alt+Enter = send, the agent's native TUI uses
Enter = send, "that mismatch is jarring when the user moves between panes", so
pair-wrap rewrites stdin to give the agent the inverted mapping too
(`sendKeymap`, `PAIR_WRAP_REMAP_RETURN`). For prose the inversion is right and
should stay. For a slash command it is pure cost: it charges a modifier for the
shortest, most frequent input, and it is *subtracting* behavior the agent
shipped with.

The path is already traced end to end:

- `decidePlainReturn` (`cmd/internal/wrapcmd/harness_tty.go:90`) is the single
  place a plain Enter is resolved.
- Its first branch is the overlay bypass, which does **not** cover this. Claude's
  detector keys on OSC 777 with body `notify;Claude Code;Claude needs your
  permission` (`wrap.go:650-688`) — a permission prompt. The slash-command
  dropdown is not that, so `overlayActive` stays false.
- So the `composerGatePositive` branch runs, `claudeComposerActive` correctly
  reports the cursor is inside the composer's ruled box, and the decision is
  `remap()` → `plainCR` = `\<CR>` → **a newline in the composer.**

Nothing is malfunctioning. The gate answers the only question it is currently
able to ask — *is a composer active?* — and for a slash command the right
question is a different one.

## Spec

### Current contract — approved 2026-10-22

The October scope Revision and approved durable plan at ../plans/000162-enter-sends-harness-addressed-lines-plan.md are authoritative. Plain Return uses native submission only for a positively proved single-line slash command in the agent composer. Prose, paths, leading user whitespace and multiline text keep the existing newline mapping. Preserve activity recognition when complete text is unavailable. Couch's concrete slash producers use plain Enter, with exact fresh authored echo proved within the existing drive deadline and the final proof and Return decision sharing one immutable frame. Public submit remains Alt+Return. Draft-pane behavior, configurable ! sigils and PAIR_WRAP_COMMAND_ENTER are deferred.

### September design retained for history — superseded where different

### The rule

**Enter sends when the line is addressed to the harness. Enter inserts a
newline when the text is addressed to the model.**

`/` is only its first instance. Prose is addressed to the model and is
inherently *growable* — a second line is always a meaningful thing to want, so
Enter must stay a newline there. A sigil-led line is addressed to the
*harness*, whose grammar is line-terminated by construction: there is no
two-line `/login`. The binding follows the addressee, not a hardcoded
character. That framing also settles what does **not** qualify — `@file.go` is
a reference *inside* prose, addressed to the model, so it keeps today's
mapping.

Stated against the existing code, the rule is one more branch in
`decidePlainReturn`'s `composerGatePositive` arm: composer active **and** its
content harness-addressed → emit bare `\r` (submit), `adapt.Bypass`, its own
`reason` string so `pair-doctor` can see it fire.

### The predicate

`harnessAddressed(line, harness)` is true when all three hold:

1. The composer holds exactly one line.
2. Its first character is a **line-committed sigil** for this harness.
3. The line matches that sigil's token grammar — for `/`:
   `^/[A-Za-z][A-Za-z0-9_-]*(\s.*)?$`.

Rule 3 is why the predicate is not just "starts with `/`", and it is the part
that has to be right. `/Users/xianxu/workspace/brain — look at this repo` is an
ordinary opening line of a prompt in this workflow and it begins with a slash;
an inner `/` in the first token disqualifies it, as does a space straight after
the sigil.

Sigils are per-harness data, not a constant: `!` (bash) and `#` (memory) are
Claude-specific, Codex has only `/`. They belong in `harnessTTYProfiles`
alongside `keymap`, `overlay` and `recognize` — the table that already exists
for exactly this kind of per-harness fact — so bringing up a new harness
declares its sigils where it declares everything else
(`atlas/how-to-bring-up-a-new-harness-cli.md`).

### The structural change

`composerRecognizer` is `func(terminalSnapshot) bool`. To ask the second
question the gate needs the composer's *text*, not just its existence.

The geometry is already computed and thrown away.
`ruledBoxComposerActive` (`composer_recognizers.go:124`) walks up to find
`promptY`, calls `ruledBoxBottomRule` for the closing rule, and carries
`spec.minCursorX` as the first text column — the exact region of the composer —
then returns `true`. `codexComposerActive` and `agyComposerActive` do the same.
So the change is to return that located region instead of discarding it, and
let the caller read cells from it. No new terminal parsing, no second source of
composer geometry (ARCH-DRY).

### Why this is not the failure mode the recognizer was hardened against

`claudeComposerActive`'s own comment is explicit that false negatives are the
expensive direction: a decline means the next Return submits a half-written
draft, and `claudeComposerMaxRows` is unbounded precisely so a tall draft can
never be mis-submitted. This issue makes Return submit *on purpose*, so it has
to say why that is a different thing.

It is strictly narrower, and gated the opposite way. The dangerous case is a
composer that is **not** recognized — an unknown state where arbitrary
multi-line prose gets committed. This rule fires only when the composer **is**
positively recognized, holds exactly one line, and that line matches a command
grammar. Every direction of failure degrades to today's behavior, and the worst
misfire sends one command-shaped line, which the agent answers with "unknown
command". The lost-draft risk is not reachable from here: a multi-line draft
fails predicate rule 1 before anything else is consulted.

### Invariants

- **Alt+Enter always submits, in every mode.** The rule is strictly *additive*:
  Enter gains a meaning, nothing loses one. Muscle memory never breaks.
- **Predicate false → today's bytes, exactly.** Overlay bypass keeps first
  claim; `composerGateLegacy`, `composerGateUnknown`, an absent profile and an
  empty `plainCR` all keep their current fail-closed paths untouched.
- **A targeted escape hatch.** `PAIR_WRAP_REMAP_RETURN=0` already turns the
  whole inversion off; add `PAIR_WRAP_COMMAND_ENTER=0` for this rule alone, so
  switching it off does not cost the inversion.
- **The new branch is instrumented** like every other branch in
  `decidePlainReturn` — its own `adapt` outcome and reason, visible to
  `pair-doctor` (`doctor/SKILL.md`, bring-up guide §3).

### The draft pane

Secondary, and it follows for free. `nvim/init.lua:3565` routes insert-mode
`<CR>` through `cr_keys`, whose contract is that `<CR>` always breaks the line
when no completion is selected; the same predicate becomes a prior branch
there, with the completion popup keeping first claim. Worth doing in the same
pass — the two panes are supposed to agree, and that agreement is the entire
justification for the inversion existing.

### Later, same predicate

**CORRECTED 2026-09-17 — this paragraph is wrong.** The picker case is not
deferred work awaiting this predicate; it already works through the overlay
branch of `decidePlainReturn`, which emits bare CR on `overlayActive`. Whatever
remains is overlay-DETECTION coverage, not a shared rule. Kept for the record;
see `## Plan` and the 2026-09-17 `## Log`.

Named so the rule reads as a rule, not scoped in: a numbered menu answer
(`1`, `2`) or `y`/`n` while a picker is up is the same shape — a closed
utterance addressed to the harness. That one keys on agent *state* rather than
composer text, which is what `overlayDetector` already computes. Same
predicate, different input.

## Done when

The following current criteria supersede the retained September checklist below:

- A positively recognized single-line slash command, including arguments, takes native Return in each supported agent composer.
- Ordinary prose, path-like first tokens, leading user whitespace, multiline input, suggestions and incomplete text proof retain the existing mapping; activity recognition is not weakened.
- Overlay priority, Alt+Return, remap-off and draft-pane Enter retain their current contracts. No configurable sigils or new command-enter switch are introduced.
- Every concrete Couch slash producer and documented slash-send example uses enter; public submit retains its Alt+Return meaning.
- Known automatic slash sends emit no Enter before exact fresh authored echo. The final write rechecks ownership and command text against the same immutable frame used by the Return decision, without new deadlines or blind retries.
- Fixture, pure reducer, stateful delayed-echo and counted-growth tests from the approved durable plan pass; ordinary peer delivery and exact /model cleanup authority stay unchanged.
- Disposable native conformance covers Claude and Codex, and Qoder when credits permit, with captures retained for replay; required recorded checks and close review pass before landing.

### September checklist retained for history — superseded where different

- `/login` typed in the **agent pane** submits on plain Enter, no Alt. Same for
  `/clear`, `/context`.
- `/model opus` — sigil plus arguments — submits on Enter.
- `/Users/xianxu/workspace/brain is the repo` gets a newline, not a submit. So
  does `/ ` followed by prose, and so does any composer already holding a
  second line.
- Alt+Enter still submits in every case above, including the ones where Enter
  now also submits.
- A permission-prompt overlay still takes the bare-CR bypass first, unchanged.
- An agent absent from `harnessTTYProfiles`, and one with no sigils declared,
  behave exactly as they do today.
- `PAIR_WRAP_COMMAND_ENTER=0` restores today's behavior with the inversion
  still on.
- `pair-doctor` shows a distinct signal when the branch fires.
- The existing `decidePlainReturn` / composer-recognizer tests pass unchanged,
  and the recognizers' returned region is asserted against the committed
  terminal fixtures in `cmd/internal/wrapcmd/testdata`.
- Verified against the real binaries, not just unit tests — this is a
  screen-reading change on a live tty, which is what `probes/` is for.

## Plan

Current implementation follows ../plans/000162-enter-sends-harness-addressed-lines-plan.md, approved by ops:0 delivery50ff4ba1. One atomic review boundary; no Mx tags.

- [ ] Clear change-code's full-flow plan-quality and estimate gates.
- [ ] Add red grammar/region/Return regressions, then shared recognition evidence and the native-command Return branch.
- [ ] Add red delayed-echo/reducer/writer regressions, then exact command expectation and final immutable-frame proof.
- [ ] Migrate the complete slash-producer/help family while preserving public submit, peer/prose sends and model cleanup limits.
- [ ] Run counted-cost tests, focused packages and audited affected-reader inventory, then disposable native conformance and retain captures.
- [ ] Update atlas and complete required recorded checks, mandatory close review and SDLC landing.

### September plan retained for history — superseded where different

TENTATIVE — written 2026-09-17, decisions settled by the operator the same day
(see `## Log`). One boundary, no `Mx`: shipping the agent pane without the draft
pane leaves the two panes disagreeing, and that agreement is the stated
justification for the inversion existing at all.

**Decisions, settled:**

- **No leading whitespace — but measure from the TEXT REGION, not the row.** The
  `\s*` in the operator's original sketch was composer CHROME (the box border,
  the prompt indicator, the padding), not user-typed space: *"the white space I
  mentioned is not something cursor can be, just part of prompt."* The harness
  already models this — `minCursorX` is *"the first column the harness leaves for
  composer text"* (`composer_recognizers.go:136`; 2 for both Claude and muse) and
  `ruledBoxComposerActive` already refuses when `Cursor.X < spec.minCursorX`
  (`:145`). So the predicate reads from `minCursorX`, and "first character" means
  first character of the text region. No `\s*` in the pattern; user-typed leading
  space disqualifies.
- **Sigils are configurable per harness; `/` and `!` are the common two.** Ship
  both as the default set, configurable rather than hardcoded. The existing token
  grammar covers both (`!ls -la` → sigil, `[A-Za-z]` head, `\s.*` tail), so one
  shared pattern is enough for now. The case to check during implementation is a
  sigil followed by a non-alpha — `!./script.sh`, `!../x` — which today's grammar
  REJECTS (falls back to newline). Confirm that is wanted rather than discovering
  it in use.
- **The draft pane ships in the same pass.** Per the Spec. If the recognizer
  refactor grows, cutting it is a scope decision to state, not a discovery.

**Steps:**

- [ ] Settle the three decisions above; record them in `## Log`.
- [ ] Declare sigils per harness where the harness declares everything else
      (`atlas/how-to-bring-up-a-new-harness-cli.md` names the place). An absent
      declaration must behave exactly as today — that is already a Done-when.
- [ ] Return the composer's located region from the recognizers instead of
      discarding it. Four of them, not three: `claudeComposerActive`,
      `codexComposerActive`, `agyComposerActive`, `museComposerActive`
      (`composer_recognizers.go:241, 38, 270, 199`), with `ruledBoxComposerActive`
      (`:143`) as the shared walker. Geometry is already computed; no new terminal
      parsing (ARCH-DRY). Assert the returned region against the committed
      fixtures in `cmd/internal/wrapcmd/testdata`.
- [ ] Implement `harnessAddressed(line, harness)` as a pure predicate — the three
      rules from the Spec. Unit-test it directly, including the
      `/Users/xianxu/workspace/brain is the repo` negative and the `/ ` negative.
- [ ] Add the branch to `decidePlainReturn`'s `composerGatePositive` arm
      (`harness_tty.go:90`). **Emit `profile.keymap.altCR`, not a literal `\r`**
      — all four profiles already define `altCR: []byte{'\r'}`
      (`harness_tty.go:24-77`), so the branch stays harness-agnostic and muse is
      covered for free. Muse matters here: its `plainCR` is the Kitty-protocol
      `\x1b[13;2u` with a documented push precondition, and this branch bypasses
      `plainCR` entirely, so the precondition does not apply on this path. Confirm
      that rather than assume it.
- [ ] Instrument the branch like every other arm — own `adapt` outcome + `reason`,
      visible to `pair-doctor`. Add `PAIR_WRAP_COMMAND_ENTER=0` as the rule-only
      hatch, leaving `PAIR_WRAP_REMAP_RETURN` untouched.
- [ ] Draft pane: same predicate as a prior branch in `cr_keys`
      (`nvim/init.lua:3565`), with the completion popup keeping first claim.
- [ ] Verify against the real binaries via `probes/`, not unit tests alone — this
      is a screen-reading change on a live tty. Cover all four harnesses, or state
      which were not exercised and why.

**Explicitly out of scope: the picker / `y`/`n` / numbered-answer case — and it
needs no predicate at all.** The Spec's "Later, same predicate" paragraph
mischaracterises it (corrected there). It is ALREADY handled, by a different
mechanism: `decidePlainReturn`'s first branch emits bare `\r` with `adapt.Bypass`
when `overlayActive` (`harness_tty.go:112-120`), and the composer-inactive path
never remaps. Operator: *"that's handled generically, basically without a cursor
we default to `<CR>` as submission."* Any residual work there is COVERAGE of
`overlayDetector` — which overlays it recognises — not a new rule, and belongs to
its own issue.


## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md, Method A only. Calibration is marked stale by estimate-source, so its rates are provisional. Decomposition covers the work with existing-stack primitives: grammar/expectation helper (smaller Go module), shared geometry and immutable-frame wiring (cross-cutting refactor), reducer/Return behavior with stateful tests (TUI screen), two real-agent conformance checks (two real-API discovery units), documentation and one boundary review. Existing terminal snapshots, recognizers, projector, reducer, writer and disposable harness supply the libraries/seams; no novel stack or greenfield-library assumption is priced.

For the thorough accepted spec, design uses the v2/v2.1 raw picks times0.2: smaller Go0.2→0.04; cross-cutting0.75→0.15; TUI1.5→0.30; docs0.10→0.02; review0.10→0.02. Discovery has no design component. Implementation picks are the v2/v2.1 hours times0.4: smaller Go0.30→0.12; cross-cutting0.50→0.20; TUI1.00→0.40; each discovery0.50→0.20; docs0.20→0.08; review0.40→0.16. Familiarity stays1.0; apply the thorough-plan15% design buffer. Thus design0.53×1.15 + implementation1.36 =1.9695h, rounded1.97h. These are derived focused ship-hours, not queue wall time.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.12
item: cross-cutting-refactor design=0.15 impl=0.20
item: tui-screen design=0.30 impl=0.40
item: real-api-discovery design=0.0 impl=0.20
item: real-api-discovery design=0.0 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.16
design-buffer: 0.15
total: 1.97
```

## Log

### 2026-09-01

- Filed from a brain session, then **corrected the same session**: first draft
  scoped this to the nvim draft pane and explicitly ruled the agent pane out of
  scope. Wrong surface — the operator meant the Claude/Codex tty directly, and
  the agent pane is the whole point. Body rewritten against
  `decidePlainReturn`; the draft pane demoted to a follow-on paragraph. The
  rule itself survived the correction unchanged, which is some evidence it is
  the right rule.
- The out-of-scope argument in the first draft was wrong on the facts, not just
  on scope: I claimed byte-level stdin rewriting has no view of the composer.
  It has a full one — `terminal_model.go` keeps a snapshot and
  `composer_recognizers.go` already locates the composer's ruled box in it.
- Confirmed mechanically rather than assumed: `detectClaudeOverlayOpen` keys on
  OSC 777 body `notify;Claude Code;Claude needs your permission`, a permission
  prompt — so the slash-command dropdown does not set `overlayActive`, and the
  composer branch is what runs on `/login`. Worth a probe to confirm the
  dropdown emits no OSC of its own.
- `/Users/...` as a legitimate slash-leading prose opener is the case that
  forced the token grammar in predicate rule 3; "first char is `/`" would
  mis-submit it, and in this operator's workflow that line is common.
- The recognizers computing and discarding the composer region is the reason
  this is a small change rather than a new subsystem.

### 2026-09-17

Tentative plan added from a brain session, pre-brainstorm. Two things checked
while writing it, so the plan does not rest on the Spec's prose alone:

- **Four recognizers, not three.** The Spec names Claude, Codex and agy;
  `museComposerActive` (`composer_recognizers.go:199`) exists too and muse has a
  full profile (`harness_tty.go:69-77`). The refactor covers four.
- **The branch should emit `altCR`, not a literal `\r`.** All four profiles
  define `altCR: []byte{'\r'}` while their `plainCR` values differ wildly —
  Claude `\\r`, Codex/agy `\n`, muse the Kitty-protocol `\x1b[13;2u`. Writing the
  branch against the keymap rather than a hardcoded byte makes it harness-agnostic
  and picks up muse with no extra case. Muse is also the one whose `plainCR`
  carries a runtime precondition (progressive-enhancement push); this branch does
  not touch that path.

### 2026-09-17 — decisions settled, and one Spec paragraph corrected

Operator answered all three open questions.

**Leading whitespace** — the answer reframed the question rather than picking a
side. The `\s*` in the original sketch was never user input; it was composer
chrome. *"there shouldn't be leading white space. but really depending on where
you measure. the white space I mentioned is not something cursor can be, just
part of prompt."* Confirmed in code: `minCursorX` already names that boundary
(`composer_recognizers.go:136`) and the recognizer already refuses a cursor left
of it (`:145`). So the predicate is first-character-of-text-region, with the
region's origin being `minCursorX` — which is also why the recognizer returning
its located region (the structural change this issue turns on) is what makes the
predicate expressible at all. The two halves fit better than they looked.

**Sigils** — configurable per harness, `/` and `!` as the common pair, shipped as
defaults. One shared token grammar covers both; the open sub-case is a sigil
followed by non-alpha (`!./script.sh`), which today's pattern rejects.

**Picker case** — out of scope, and the Spec's "Later, same predicate" framing was
wrong. Confirmed in code before correcting it: `decidePlainReturn`'s FIRST branch
already returns bare `\r` with `adapt.Bypass` and `submits: false` when
`overlayActive` (`harness_tty.go:112-120`), commented *"pair-local picker confirm;
never reaches the agent"*. So it is handled by overlay detection, not by a
composer-text predicate, and it shares no machinery with this rule. The Spec
paragraph is annotated rather than deleted.


## Revisions

### 2026-10-22 — Operator narrows scope to slash commands and Couch sends

Authenticated ops:0 dispatch48a143ee-38a9-4508-9666-a3308d605cb1 and scope confirmation1f501f4a-3261-4694-9457-70c4efb12b5a supersede the September extras. Current deliverable is slash-only behavior in the agent composer, plus switching Couch's programmatic slash sends to plain Return. Draft-pane Enter stays unchanged; configurable ! sigils and PAIR_WRAP_COMMAND_ENTER are deferred. Preserve the command-versus-path grammar, typed leading-space and multiline negatives. A menu/overlay takes its existing plain Return path. Alt+Return remains the unconditional submit action for prose and compatibility.

Current-source differences from the old analysis: codexComposerBounds and ruledBoxComposerBounds already exist, as does qualifiedComposerText for Claude/Codex/Qoder. That reader intentionally excludes unknown/shell modes and must not widen ordinary peer input. Current profiles include Muse/Agy/Grok too. Design must preserve activity recognition even when a complete single-line text witness is unavailable; unknown text must keep the active composer's newline mapping rather than accidentally granting bare Return (ARCH-DRY, ARCH-PURPOSE).

## Planning checkpoint

2026-10-22: claimed162 and entered its own main-based branch000162-enter-sends-harness-addressed-lines at09c8964de39cdb5e6bd9db20601c232446af6949, with no806 source commits. Catch-up hit the known ariadne#557 seeded workflow diff; per the fleet instruction saved /Volumes/TestScratch/tmp/pair-slot2/pair162-seeded-refresh.taoYmx/merge-check.yml.patch and restored only that workflow before rerunning start-plan successfully. No code edited or tests/native launched for162.

Read-only slash-send census: KeyEnter emits CR through the existing translator; KeySubmit is explicitly Alt+Return (ESC CR). Structured commandNext in agentdrive/model.go is the sole production literal slash sequence found (clear-line, text:/model, submit). Public CLI/embedded skill/README examples also use /usage or /model plus submit and must change to enter; generated enter help must document the new command case. Generic submit retains Alt+Return semantics for arbitrary prose. Peer delivery already writes native keymap.altCR directly after a bracketed prose envelope and must not be rerouted; orientation/recovery prompts are prose, not slash automation. No dedicated /clear or /new production sender found. Existing unsent /model and /status test drafts remain unsent.

Important design discovery: commandNext emits its three steps on successive EventWritten without requiring exact command echo. Merely changing KeySubmit to KeyEnter could therefore run the snapshot-dependent Return classifier against an old complete empty frame and insert newline. Codex's existing150ms quiet bound is not proof of echo; Claude has no such delay. Plan must cover delayed/coherent echo, unrelated timer repaint, changed text, cancellation/deadline and the final pre-write proof for structured and public slash sends without sleep tuning, cached-input authority or wider deadlines. Existing ScreenCommand, command index, writeEpoch and MatchesScreen are the likely reusable seams. A read-only helper is checking that bounded causal path; no implementation choice approved yet.

Next: finish the durable plan, enumerate all six profile readers, the shared predicate and return decision, every slash producer and its write proof, tests/counted bounds and live acceptance; send the plan to the hub before code. #806's original suite has now passed10m35s on23c008e9; checkpointing162 before switching back to806 for its landing. No162 change-code or production work is authorized yet.


### 2026-10-22 — Proposed durable design

Plan: ../plans/000162-enter-sends-harness-addressed-lines-plan.md. One richer recognition result preserves existing activity while optionally proving complete command text; one shared pure slash grammar feeds Return and command expectations. The drive reducer waits for exact fresh echo, binds it to Enter and rechecks it at the existing writer; expectation text is a comparison target, never authority. Structured /model and public examples switch to enter; generic submit/prose/direct peer writes keep their contracts. Preserve exact /model-only cleanup. Full flow expected; plan approval and change-code remain pending.

A temporary plan-only worktree under /Volumes/TestScratch/tmp/pair-slot2/pair162-plan.aUsFyM holds this existing162 branch while806's retained merge runner owns the numbered checkout. The worktree follows the repository's test-scratch preference, shares no working tree/index with806, and will be removed after returning162 to its numbered slot; no implementation or native process runs there. #806's suite is PASS10m35s and its merge is ready behind801.

### 2026-10-22 — Approved plan; shutdown checkpoint

Authenticated ops:0 delivery50ff4ba1-fd38-4c7b-89a6-a3bb0093a088 approves the durable plan and listed tests, including exact authored echo and one immutable frame for the final proof and Return decision. Approval is recorded; change-code has not run and no implementation or native check has begun. The coupled recognizer/reducer/writer change is beyond a short checkpoint-sized edit, so leave it intact for after the planned22:00–23:00 shutdown rather than start a partial implementation. The fleet notice6e79eb07 prohibits new merges/landings and requires pushed checkpoints by21:45.

Exact next step after restart: verify #806's retained merge outcome through sdlc landing/queue status and PR549 before creating any runner; return this162 branch from its plan-only worktree to pair:2 once806 releases the numbered checkout, remove the clean temporary worktree, reconcile the active Done-when/Plan with the scope Revision while preserving the September history, then run sdlc change-code --issue162 for plan-quality and the measured estimate gate. Implement the approved red/green sequence only after that gate. No162 merge is authorized tonight. At this checkpoint806's original merge session77210 (pid80534) remains first ready behind801, with no duplicate runner started.


### 2026-10-22 — Post-shutdown resumption and contract reconciliation

Authenticated ops:0 delivery8065c647 lifts the shutdown freeze and directs rebuilding on Go1.27.3, re-entering806 merge, then162 change-code. Pair:2 make build completed and pair/couch report go1.27.3. The former806 runner was absent (PID80534 gone, terminal77210 unknown, no landing ticket, PR549 still open); one replacement runner36605 was started and vouched ready, awaiting the shared repository lock. Queue inspection still reports helper inspect exit2, reported to the hub without changing shared tools.

Made the active Spec/Done-when/Plan reflect the already approved October contract. All September text/checklists remain verbatim under explicitly superseded historical subheadings; no new design delta. The temporary162 worktree remains documentation-only until806 releases the numbered checkout. Next: return162 to pair:2, remove the clean temporary worktree, run change-code full flow and follow its named gates before implementation.


### 2026-10-22 — Plan gate and estimate derivation

Plan-quality round1 raised PQ-1 (function-specific-test-strategy); round2 CLEAN disposes it addressed. The accepted design names risky functions with adversarial classes and mechanical guards, including generated sequence invariants, independent submission oracles, counted work and withheld-echo mutation. No product design changed. The temporary plan worktree was removed cleanly and162 returned to pair:2 while806's required integrated suite runs from its own detached checkout (lease-3049011771 on c683e0a4406c8c9792285a0ab325b0ba8fa4ee3e); no806 merge runner is active.

