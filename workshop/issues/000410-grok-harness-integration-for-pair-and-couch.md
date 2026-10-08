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

- [x] M1 — usable in pair and Couch-resumable: registry + ledger membership, resume spellings, fresh-launch table, shared inline mode before `--`, resume compose + mint, TTY profile + live captures, overlay markers, orientation glyph, grok scanner + event normalizer + every dispatch site (re-cut: old M1+M2, see plan Revisions)
- [x] M2 — slug, prompt glyph (scrollback/distill), permission allowlist, docs sweep, follow-up issue, operator live smoke (closes with the final `sdlc close`)

## Log

### 2026-10-07

- Brainstorm: scope is full parity in three milestones; peer delivery goes to a follow-up. Bring-up loop is `pair-dev grok` from a Claude slot; Couch at M3. Provisional Couch attach deferred. Grok login confirmed; session layout measured live (stub session without updates.jsonl observed).
- Spec review (fresh eyes): 10 findings, all folded in. Measured `grok -s <uuid>` in the TUI: it creates the session dir, so pair mints the ID. Added an M1 known-gap entry, the fresh-launch tables, inline flag before `--`, M2 session-side wiring, the orientation glyph, and concrete smoke observations.
- Spec re-review: 4 more findings folded in: `resumeform.Forms` moves to M1 (the fresh validator reads it), `resumeToken`/`composeResumeArgs` go in M2, both `-c` strip sites are named, and the orientation interim is noted.

### 2026-10-08 — M1 implementation
- 2026-10-08: closed M1 — Unit+fixture suites green for launcher, resumeform, sessioninventory, sessionwatch, sessionledger, wrapcmd (clean env, unsandboxed). Live: grok TTY recognizer fires on real 1.0.46 bytes; pickers captured and declined; ESC CR inserts newline in live grok; session-inventory --agent grok lists 12 real resumable roots with zero grok diagnostics. Full suite: only failures are pre-existing on main (artifactpath, couchcmd, couchsingleton, gcruntime) plus Couch-slot env noise; see Log. Round-1 review findings BR-1..BR-5 fixed at class level.; review verdict: FIX-THEN-SHIP

- **Re-cut:** M1+M2 merged (plan Revisions). A registered agent that the session ledger rejects cannot launch: `freshAgentInvocation` encodes a ledger record per launch. And `TestFreshAgentInvocationHandsTheWatcherToTheReplacementWrap` needs a watcher for every registry agent. So registry, ledger and the full session side land in one boundary.
- **TTY (captured live, grok 1.0.46):**
  - The composer is a rounded box at column 2 with `❯` at column 4, recognized as a `ruledBoxComposerSpec` registration (`ruleCol`/`sideGlyph`); mutation-checked.
  - Grok queries the Kitty keyboard flags (`CSI ? u`) and pushes none when unanswered, so plain Return becomes ESC CR.
  - **ESC CR measured live as Grok's newline:** a prompt typed as `first line` ESC CR `second line` CR arrived as one `user_message_chunk` with `"first line\nsecond line"`.
  - Pickers captured through `TestHarnessTTYLiveDrivenConformance`: `overlay.raw` (permission, via `chmod`, because pair's `.claude/settings.json` allows `rm` and Grok honors Claude allow rules) and `selection.raw` (question). The gate declines on both. Markers are chrome only.
  - `ttyFixtureVersionDir` now prefers the dotted token, because Grok's version string ends in a build hash.
- **Session side:**
  - The scanner reads `<enc-cwd>/<uuid>/updates.jsonl`. The fixture is a sanitized real TUI turn plus a tool turn, a stub session and sibling files.
  - **Live inventory:** `pair session-inventory --agent grok --scope all` found 12 real roots, all resumable, with zero grok diagnostics (the two `pair_record_malformed` are pair-data, and appear for qoder too). Grok events normalize with zero `turn_unusable`, against qoder's 19.
- **Pre-existing failures, not #410:**
  - `artifactpath` `TestProductionArtifactReferencesAreExactlyClassified` fails identically on a clean `main` worktree (Couch files absent from the inventory).
  - `couchcmd` `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins` fails with grok removed from the registry too.
  - PTY-spawning tests fail inside the sandbox ("operation not permitted"). They pass unsandboxed with a clean env (`env -i`), per the memory note on session env leaks.
- **Aspect 6 finding:** Grok falls back to Claude Code's `.claude/settings.json` permission rules when it has no TOML rules. Pair's repo allowlist already applies, as seen live in the capture work. M2 Task 11 verifies it rather than adding config.
- **M1 full-suite evidence (2026-10-08):**
  - `make -k test` (five-var scrub): every target passes except `test-pair-embedded-runtime` (expected Couch-slot env noise, per memory) and `test-changelog` under the default `TMPDIR`. `test-changelog` with a scratchpad `TMPDIR` passes (exit 0).
  - `go test ./...` (clean `env -i`): failures only in `artifactpath`, `couchcmd`, `couchsingleton` (selection-size fixture) and `gcruntime` (`TestCouchReferencesLocalArchiveLocatorRoundTrip`). All four fail identically on a clean `main` worktree.
  - `couchcore` hit Go's default 10-minute package timeout. The test it was in (`TestSetAsideHoldsMatchThePlan`) passes alone on both branch and main, and a 30-minute whole-package run is in progress.

### 2026-10-08 — M2 implementation

- **Slug:** `grok -p` ignores stdin (measured: it answered a stdin-counting prompt with a preamble only), and grok has no persistence switch (README, `grok -p --help`). `runGrok` therefore uses `--prompt-file` (instructions + input) in a fresh `pair-model-grok-*` temp dir, with `-m grok-4.7-build-fast --max-turns 1 --no-subagents`, then removes that dir's `~/.grok/sessions/<PathEscape(EvalSymlinks(dir))>` entry. Grok encodes the resolved cwd (`/private/var/…`, not `/var/…`), and the entry holds exactly the session dir plus `prompt_history.jsonl`. Live `TestRunGrokLiveConformance` passes (BANANA via the prompt file, ~3.5 s) and leaves 0 `pair-model-grok` entries.
- **Prompt glyph:** Grok echoes a submitted prompt as `❯ text` at column 5, one right of the composer glyph (column 4, inside `│`). Captured as `echo.raw` by a driven scenario. It is kept outside the tty replay oracle: the 65 KB stream can't be trimmed, and replaying it at every byte split cost 49 s of the wrapcmd suite (37 s total without it, 85 s with). `grokEchoPromptCol` is pinned against it. Scrollback, distill and orientation all derive from the authorities through table-driven parity tests. Distill learned Grok's footer rows (box edges, the `│ ❯ │` row, the `Shift+Tab:mode` hints, the `[stop]` spinner), taken verbatim from the capture.
- **Permissions (aspect 6), measured:** Grok merges `.claude/settings.json` (pair's repo file only allows `mkdir/cp/mv/rm/ln`), so the standard dev set did not reach it. Before: a headless `make --version` was `permission_cancelled`. Added a committed `.grok/config.toml` `[permission]` set (`git`, `make`, `sdlc`, `lsof`, `zellij`; qoder's equivalent is the tracked `.qoder/settings.local.json`). After: `grok inspect` shows both sources (21 rules) and `make --version` runs (`GNU Make 3.81`).
- **Fixture header note:** Grok's composer header shows the abbreviated cwd and branch (`~/w/w/pair-slot6/pair`, `000410-…`) in `composer.raw` and `echo.raw`. These are not absolute home paths (the neutrality oracle passes), and they name only this repository.
- **Follow-up filed:** #411, a Couch peer-delivery receiver profile for grok (Done-when item).
- **Docs:** README (M1 round), atlas (`architecture`, `session-identity`, `index`, `couch`, the bring-up guide), `doctor/README.md` + `SKILL.md`, and CHANGELOG all list Grok.

### 2026-10-08 — operator smoke round 1

- **Verified by the operator:** Return/Alt+Return (1), the question picker (3), Alt+n and resume (5). Live session `grok` launch 9 bound to the minted id `ec6f0528…` by correlation through the grok scanner; the agent ran as `grok --minimal --resume ec6f0528… --no-alt-screen`.
- **Issue 1, status line showed `grok [!]`:** `ParseTokenUsage` had no grok case. Measured on the live session: the last update's `params._meta.totalTokens` (56912) equals grok's `signals.json` `contextTokensUsed`. `turn_completed` usage is per-turn billing summed over model calls (262k), so it is not occupancy. Fixed; `pair context grok grok` now prints `57k` live.
- **Issue 2, `--no-alt-screen` is not enough:** grok's inline mode stays off the alternate screen but repaints its whole UI in place, so nothing reaches scrollback. `--minimal` (scrollback-native) works, and alone it never enters the alternate screen (0 × `?1049h`, measured), so the inline flag is now `--minimal`. Minimal mode is a different UI: no box, `❯` at column 0 for the composer and the echo alike, a faint `minimal · /help` row above and a `·`-separated status row below.
  - The recognizer is rewritten (status row as discriminator); the box spec fields `ruleCol`/`sideGlyph` were removed (dead).
  - `grokEchoPromptCol` was merged back into `grokPromptCol = 0`; scrollback, distill and orientation derive from it.
  - Pickers repaint word-by-word at absolute columns, so markers carry both spellings. The question picker has no footer in minimal mode, so its free-text row is the marker.
  - All four captures were retaken in minimal mode from a fresh directory (`harnessTTYCaptureDir`), because the minimal welcome card prints the absolute cwd. `echo.raw` is now 5.9 KB.
  - Distill footer rows were replaced with minimal-mode ones (hint, meter-keyed status, braille spinner).
- **`:PairDoctor` run (grok tag):** the perf half shows a healthy host (pipe_hop 0.006 ms, fork_exec 2.1 ms, zellij 17.9 ms). The adapt log `adapt-grok.jsonl` is absent on disk (only its diagnostics sidecar remains), so there is no drift tally for that session; the wrapper's `wrap-events` trace has only I/O labels. The next smoke should rerun `doctor/doctor.sh` while the session is live.

### 2026-10-08 — operator smoke round 2

- **Context meter verified live:** the title reads `grok (31k)`.
- **`[!]` after the title is zellij's bell mark.** Grok rang 3 bare BELs in that session; Claude rings none, since every one of its 48 BELs terminates an OSC. Grok's `[ui.notifications] method = auto` falls back to BEL inside zellij. Measured: `osc9` set in a `config.toml` makes Grok emit `OSC 9 "Turn complete in 2.2s. · Grok"`, but the project file, `GROK_CONFIG` and `GROK_CONFIG_PATH` all ignore `ui.*` (`grok inspect`: "set but ignored"), and no `GROK_*` env var exists for it. Pair therefore translates the bell itself: `bellAttentionHarnesses` (grok) makes the notification rewriter turn a ground-state BEL into the canonical attention notification and drop it from the stream. #14's false-positive class can't recur: `outputBoundary` tracks string state across chunk splits. Replaying the real grok2 session: 3 notifications, 3 bytes dropped, the OSC-terminating BEL kept.

### 2026-10-08 — operator: draft Ctrl+C must send Ctrl+C to grok

- The draft pane's Ctrl+C wrote ESC (27) to every agent. Grok cancels a turn on Ctrl+C (its own `Ctrl+c :cancel` hint) and treats ESC as input. `nvim/interrupt.lua` now holds the per-agent interrupt byte (ESC by default, Ctrl+C for grok), read against the tag's current agent file at keypress so a switch-agent is followed. Tested by `nvim/interrupt_test.lua` (in `make test-lua`). The wrapper passes `0x03` through untouched. This was missing from the bring-up checklist: added as "Set the interrupt byte", plus "Check the attention signal" from the bell work.

### 2026-10-08 — Couch smoke and close decision

- **Couch, operator-verified:** Grok is offered by Couch and runs hosted. pair:5's thread was switched Claude → Grok and is live (`pair --couch-session-v1 resume 1-pair-20` → `grok --minimal --session-id …`), the context count shows in the title, and the `[!]` bell mark is translated.
- **Not met here, filed as #414:** switch-agent to Grok left the console attach refused (`attach record/handle process identity mismatch`, then `start handle is unavailable`) and the orientation prompt undelivered. A Codex → Claude switch did not reproduce it. Root cause is in Couch's switch/attach path (not localized); the operator chose to file it and close #410.
- **Follow-ups:** #411 (Grok Couch peer delivery), #414 (switch-agent attach/orientation).
- **Close window and actual (ariadne#269/#270/#304):** `origin/main` was merged into this branch before the slot move. The whole-issue review is still correct: the reviewer diffs `merge-base(main, HEAD)` = main's tip against HEAD, which is exactly #410's net change (86 files, M1 + M2, nothing of main's), even though close prints a short commit range. A milestone close for M2 would have over-covered (its base predates the merge), so M2 closes with the issue close. The actual is measured from pair:6, where this session's transcripts live (pair:0 saw only 0.26 h): 2.72 h. That is known low by roughly an hour: ariadne#270 attribution credited the 23:00–01:00 M2 segment to #411, which was filed inside it.

