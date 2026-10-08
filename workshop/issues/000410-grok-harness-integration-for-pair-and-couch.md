---
id: 000410
status: working
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours: 5.25
card_mirror: '44a700581cdaa640e603852c687258b15958b9b0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-07T22:22:25-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Grok harness integration for pair and couch

## Problem

Pair and Couch host `claude`, `codex`, `agy`, `muse` and `qoder`, but not Grok.
To bring Grok in, it has to join the agent registry and pass every integration
surface that the existing harnesses already pass, so that it runs under pair as
smoothly as they do and Couch can launch, park and resume it.

## Spec

Bring up a `grok` harness by following
[How to Bring Up a New Harness CLI](../../atlas/how-to-bring-up-a-new-harness-cli.md)
(§0 registry, §2 ten-item checklist), to full parity with the existing harnesses.
Couch peer delivery is out of scope (follow-up issue).

### The harness (measured 2026-10-07)

- **CLI:** `~/.local/bin/grok` is xAI's "Grok Build TUI", 1.0.46 stable, a
  full-screen TUI. It has `--no-alt-screen` and `--minimal` modes.
- **Keys:** Enter sends; Shift+Enter or Alt+Enter inserts a newline. This is the
  opposite of pair's convention, so Grok needs the Return remap.
- **Session ID:** the TUI honors `-s/--session-id <uuid>` for a new
  conversation: `grok -s <uuid>` with no prompt created
  `~/.grok/sessions/<cwd>/<uuid>/` immediately. The README's "headless only" note
  is wrong for 1.0.46. Pair therefore mints the ID, as for claude and qoder
  (`MintsSessionID`, `cmd/internal/launcher/agentargs.go`). Resume is
  `-r/--resume <id-or-title>`; `-c/--continue` resumes the cwd's latest session.
- **Sessions:** stored at `~/.grok/sessions/<url-encoded-cwd>/<uuidv7>/`.
  `summary.json` holds `info.id`, `info.cwd`, `created_at` and `updated_at`;
  `updates.jsonl` is an ACP `session/update` stream with `params.sessionId` on
  every line. A stub session can have `summary.json` but no `updates.jsonl` (seen
  after `grok login`). The per-cwd `prompt_history.jsonl` lists
  `{session_id, prompt}`.
- **Permissions:** set in `~/.grok/config.toml` or with `--allow` /
  `--always-approve`. Headless mode is `grok -p <prompt>`.

### Bring-up loop

Grok is developed in pair first, not in Couch. Plain `pair-dev grok` already runs
an unregistered agent (only the fresh-launch, continuation, restart-marker
(`markers.go`, so Alt+n) and Couch-profile paths check `IsSupportedAgent`), and
each aspect that misbehaves there is a checklist item. The work is done from an existing Claude slot. Couch enters at M3: once
`grok` is registered and has a scanner, Couch's start and switch-agent menus pick
it up (guide §0), and switching a slot to Grok is the acceptance test.
Registering in M1 does list Grok in Couch's menus early: a Couch launch works
from M1, but park and resume only after M2, and switch-agent's orientation
auto-submit only after M3. That interim is acceptable, and the M1 known-gap entry
(below) records both.

### Design: follow the per-harness pattern

Grok entries go next to the existing harnesses' in the existing tables. There is
no new mechanism, with one exception: Codex's forced inline mode
(`codexAltScreenArgs`, `cmd/internal/launcher/agentargs.go`) becomes a shared
per-harness field ("inline flag + opt-out env") that both Codex and Grok use, so
the alt-screen handling is not copied (ARCH-DRY).

- **M1, usable in pair:**
  - Registry: launcher `supportedAgents`, sessioninventory `Agent` enum and
    `supportedAgents`. Add a `sessionInventoryKnownGaps["grok"]` entry
    (`cmd/internal/launcher/agent_parity_test.go`) so the parity test passes
    while the session side is still missing. M2 deletes it.
  - Grok's resume spellings (`--resume <id>`, `--resume=<id>`, `-r <id>`, glued
    `-r<id>`) in `resumeform.Forms`. That one table is what the fresh validator,
    the launcher and sessionwatch all read (#300 BR-15), so it lands here, not in
    M2.
  - The fresh-launch entry (`freshAgentSpecs`, `freshValueOption`,
    `freshVariadicOption` in `fresh_args.go`): forbidden flags
    `--continue --session-id --fork-session`, forbidden short flags `cs`
    (`-r`/`--resume` come from `resumeform`), and Grok's value flags (`--agent`,
    `-m`, `--cwd`, `--permission-mode`, …) declared so their values are not
    mistaken for the positional `[PROMPT]`.
  - Inline mode becomes a per-harness `{flag, optOutEnv}` field. Codex keeps
    `--no-alt-screen` with `PAIR_CODEX_ALT_SCREEN`; Grok gets `--no-alt-screen`
    with `PAIR_GROK_ALT_SCREEN`. The `CodexAltScreenOptOut` option becomes
    agent-keyed. The flag must go **before** any `--`, because Grok reads
    trailing words as `[PROMPT]`. Today's Codex placement after `--`
    (`peer_launch_test.go`) changes, and a test pins the new order for both.
  - TTY profile in `harnessTTYProfiles`: a composer recognizer
    (`composerGatePositive`; without one the profile fails closed and Enter is
    never remapped), the Return remap (pair Enter → Grok Alt+Enter newline, pair
    Alt+Enter → Grok Enter send), and an overlay detector for Grok's permission
    and selection pickers. Captured fixtures under
    `cmd/internal/wrapcmd/testdata/tty/grok/1.0.46/` are required for the
    composer, the remap and each overlay.
- **M2, Couch-resumable:** the session side, landing together as the guide's
  item 4 requires:
  - A versioned scanner over `summary.json` + `updates.jsonl` with a conformance
    fixture. A stub session without `updates.jsonl` reads as "no conversation",
    not an error.
  - `sessionwatch.SupportsAgent`, the ledger's `isSupportedAgent`,
    `NormalizeNativeEvent`/`ProviderContractFor`, the completed-round watcher and
    the provisional launch baseline.
  - A `grok` case in `observationNativeID` (`sessioninventory/target.go`), plus
    tests proving `OSRuntime.AgentSessionExists` and `EstablishedSessionID` read
    the scanner inventory.
  - `MintsSessionID` includes grok. `resumeToken` and `composeResumeArgs`
    (`agentargs.go`) gain a grok case; without it a cold resume passes no
    `--resume <id>` and silently starts a fresh conversation. `-c/--continue` is
    stripped from persisted config at both sites, the launcher's
    `persistedConfigArgs` and sessionwatch's, each reading `resumeform`.
  - `pair-slug` through `grok -p`.
  - Delete the known-gap entry, so the parity test passes on real wiring.
- **M3, polish + live check:**
  - The permission allowlist in `~/.grok/config.toml`.
  - Grok's user-prompt glyph in `nvim/scrollback.lua` (Alt+b) with its
    `nvim/scrollback_test.lua` row, and in `orientationPromptOK`
    (`wrapcmd/orientation.go`). Without the latter, switch-agent's orientation
    auto-submit never fires on Grok.
  - Add the orientation glyph to the guide's §2 checklist (the guide is missing
    it), and list Grok among the supported harnesses.
  - Operator live smoke test under Couch.

### Alternatives considered

- **ACP-generic scanner** (because `updates.jsonl` is ACP): no other harness in
  pair speaks ACP, so this is speculative generality. Revisit when a second ACP
  harness arrives.
- **Drive `grok agent stdio` headless:** abandons pair's model (a real agent TUI
  in a pane) and every existing surface. Rejected.
- **Provisional "attach any agent" mode in Couch:** it could only launch. Park,
  cold-resume and the live/parked listing all go through the scanner, so it would
  be a second, weaker supervision mode to maintain, for a capability that
  `pair-dev` already gives. Deferred; revisit if the next harness bring-up hits
  the same friction.

### Durable state (ARCH-FUNERAL)

Pair creates nothing durable of its own for Grok. Grok owns and retains its
session directories. Pair's existing adaptation log and inventory records gain
`grok` rows under their current retention, with no new family. Captured fixtures
are versioned test data, replaced when the harness version moves.

## Done when

- `grok` is in the launcher `supportedAgents`, the sessioninventory `Agent` enum
  and its `supportedAgents`, and `TestAgentInventoryParityWithSessionTables`
  passes with `sessionInventoryKnownGaps` empty again.
- A fresh launch of grok rejects `-c`, `-r`, `--resume`, `-s` and
  `--fork-session` (the resume spellings through `resumeform.Forms`), and a test
  parses a value flag (`--agent X`) as a flag, not as the prompt.
- Codex and Grok share one per-harness inline-mode field. The flag lands before
  `--` for both (test-pinned), and `PAIR_CODEX_ALT_SCREEN` and
  `PAIR_GROK_ALT_SCREEN` each opt out.
- Captured 1.0.46 fixtures under `cmd/internal/wrapcmd/testdata/tty/grok/` pin
  the composer recognizer, the Return remap and each overlay detector.
- The Grok scanner's conformance fixture covers a full session and a stub
  session. `AgentSessionExists` and `EstablishedSessionID` tests read the
  inventory. A launch mints `--session-id`. The resume spellings round-trip,
  a test shows `composeResumeArgs("grok", …)` placing `--resume <id>` before any
  `--`, and persisted config drops `-c` in both the launcher and sessionwatch. `pair-slug` produces a slug for a Grok session.
- `nvim/scrollback_test.lua` and an orientation test cover Grok's prompt glyph.
- Operator live smoke test in `pair-dev grok`: a multi-line prompt (Enter
  inserts a newline, Alt+Enter sends), a permission picker confirms with plain
  Enter, mouse scroll moves through scrollback, Alt+b jumps to the prompt, and
  `doctor/doctor.sh` shows grok `return-remap`, `session-id` and `slug-parse`
  firing. Then under Couch: the start and switch-agent menus list Grok, switching
  a slot to Grok auto-submits orientation, and the thread parks and cold-resumes.
- A follow-up issue for a Grok Couch peer-delivery receiver profile is filed.
- The atlas guide lists Grok and includes the orientation-glyph checklist item.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` (calibration flagged stale by `sdlc estimate-source`; numbers provisional). Method A only.*

**Derivation:**
- **Design hours:** v2 ranges with the ×0.2 spec-quality discount on every code primitive, because the durable plan resolves files, seams and failure modes. Two items are undiscounted: `issue-spec` (spec, plan, two spec reviews and two plan-quality rounds already spent since the claim) and `ux-rename-iteration` (operator smoke iteration cannot be pre-resolved).
- **Library check (Step 2.5):** no external library applies. The internal short-circuit is smaller than qoder's: Grok is not claude-family, so the scanner is a greenfield module on the `scan_muse.go` shape, and every other surface mirrors an existing per-agent seam.
- **Implementation hours:** 40% of v2 ranges (v3.1), in the upper part for the live-capture items (`tui-screen`, both `real-api-discovery` budgets) and for the cross-cutting inline-mode/`resumeform` refactor that touches codex too.
- **Familiarity 1.0:** pair#300 is a fresh, detailed precedent. The novel surfaces (Grok TTY bytes, the ACP transcript, `grok -p` persistence) carry their own discovery budgets.
- **Design buffer +15%** (thorough plan doc, v2.1).
- **Boundaries:** M1 and M2 milestone-closes plus the final close, one `milestone-review` each.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=1.00 impl=0.08
item: smaller-go-module       design=0.06 impl=0.12
item: smaller-go-module       design=0.06 impl=0.16
item: cross-cutting-refactor  design=0.10 impl=0.20
item: tui-screen              design=0.30 impl=0.28
item: real-api-discovery      design=0.00 impl=0.24
item: smaller-go-module       design=0.06 impl=0.16
item: greenfield-go-module    design=0.30 impl=0.28
item: smaller-go-module       design=0.06 impl=0.16
item: smaller-go-module       design=0.06 impl=0.16
item: real-api-discovery      design=0.00 impl=0.12
item: smaller-go-module       design=0.05 impl=0.10
item: atlas-docs              design=0.04 impl=0.06
item: ux-rename-iteration     design=0.30 impl=0.08
item: milestone-review        design=0.00 impl=0.10
item: milestone-review        design=0.00 impl=0.10
item: milestone-review        design=0.00 impl=0.10
design-buffer: 0.15
total: 5.25
```

**Item order**, top to bottom:
1. Spec, durable plan and review rounds (already spent).
2. M1: registry join; `resumeform` + fresh-launch table; shared inline mode / before-`--` placement / `resumeform` selector groups / resume compose + mint; TTY profile + live captures; TTY discovery; overlay markers + recognizer spec.
3. M2: grok scanner (greenfield); event normalizer + every dispatch site.
4. M3: slug via `grok -p` with confined cleanup; slug/permission discovery; prompt glyph across three consumers; docs/atlas sweep; operator smoke iteration.
5. Three boundary reviews (M1, M2, final close).

## Plan

Durable plan: [000410-grok-harness-integration-for-pair-and-couch-plan.md](../plans/000410-grok-harness-integration-for-pair-and-couch-plan.md)

- [ ] M1 — usable in pair: registry (+ known gap), resume spellings, fresh-launch table, shared inline mode before `--`, resume compose + mint, TTY profile + live captures, overlay markers
- [ ] M2 — Couch-resumable: grok scanner + event normalizer + every dispatch site; known gap deleted
- [ ] M3 — slug, prompt glyph (orientation/scrollback/distill), permission allowlist, docs sweep, follow-up issue, operator live smoke

## Log

### 2026-10-07

- Brainstorm: scope is full parity in three milestones; peer delivery goes to a follow-up. Bring-up loop is `pair-dev grok` from a Claude slot; Couch at M3. Provisional Couch attach deferred. Grok login confirmed; session layout measured live (stub session without updates.jsonl observed).
- Spec review (fresh eyes): 10 findings, all folded in. Measured `grok -s <uuid>` in the TUI: it creates the session dir, so pair mints the ID. Added an M1 known-gap entry, the fresh-launch tables, inline flag before `--`, M2 session-side wiring, the orientation glyph, and concrete smoke observations.
- Spec re-review: 4 more findings folded in: `resumeform.Forms` moves to M1 (the fresh validator reads it), `resumeToken`/`composeResumeArgs` go in M2, both `-c` strip sites are named, and the orientation interim is noted.
